## Part 1

### Package Layout

I've used the recommended layout from the doc:

```text
url-shortener/
├── cmd/
│   └── server/
│       └── main.go       # Entry point: parses flags and starts the HTTP server
├── internal/             # Private application packages
│   ├── handler/          # HTTP handlers (/api/shorten, /{short_url})
│   ├── shortener/        # Short URL generation logic
│   └── storage/          # Bi-directional in-memory store (short <-----> long URL)
├── go.mod
├── README.md
└── DECISIONS.md

```

### Determining URL Equality (Normalization)

URL normalization is handled in the `shortener` package. To ensure we do not create multiple short links for the exact same destination, the normalization process applies the following rules:

* Trims leading and trailing whitespace.
* Lowercases the protocol scheme and hostname. (It does not lowercase the path or query parameters, as these can be case-sensitive).
* Removes default protocol ports (port 80 for HTTP and 443 for HTTPS).(same url, we don't want seprate code for)
* Removes the rightmost trailing slash, if present.(the same url, we don't want to codes for)


### Storing and Retrieving URLs

When we receive a new URL, we normalize it first and then check if it is already stored using the `GetByURL` method in the `storage` package (which is backed by an in-memory map). If it is already stored, we return the existing code. If it is not, we generate a new code and store the mapping using the `Save` method.

### Code Generation for New URLs

Codes are strictly generated only if the normalized long URL does not already exist in the map (checks with `urlToCode`). When a new code is needed, the `shortener.GenerateCode()` function generates a 6-character string. It uses `crypto/rand` to generate cryptographically secure random bytes and maps each byte to a 62-character base alphabet (`a-z`, `A-Z`, `0-9`).

### Collision Handling

When a new code is generated, the handler checks the storage using `h.Store.GetByCode(newCode)` to see if that code is already in use. If a collision is detected, the system loops and generates a fresh random code. It repeats this process until it finds an unused code, at which point it breaks the loop and proceeds to save it.

### Mutex Locking Strategy

We use `sync.RWMutex` because, in a URL shortener, read operations (redirecting to a link) are significantly more frequent than write operations (creating a new link). The `RWMutex` allows multiple simultaneous read locks but completely blocks both reads and writes during a write lock, ensuring thread safety while optimizing for high-read throughput.


## Part 2
### `Store` interface location and methods.

I've defined Store interface in api package so my handlers can define what exactly do they need for a storage and how ever implements them can be used as storage, before that we had an Store attribute in Handler struct which made my api coupled to storage and would made trouble if I wanted to change it. the Store interface define 3 `methodGetByCode(code string) (storage.LinkData, error)`find URL from it's shortend code `GetByURL(url string) (string, error)`check if the url have already been shortend `Save(url, code string) (string, error)` store a new mapping 
(all three functions could return erros ,they fullfil their duity like Save can return Collision sentinel error etc)


### How errors become status codes and response bodies.
translation of erros (sentinels erros) to HTTP response codes happens in api package using errors.Is we check if the specificed error is present. (the only error that has a JSON body is ErrNotFound which writes Uknowncode, ErrInvalidURL only return the code and ErrCodeCollison never get sends we stay in the for loop until it's resolved)

ErrInvalidURL -> 400 Bad Request :` if errors.Is(err, shortener.ErrInvalidURL) {http.Error(w, err.Error(), http.StatusBadRequest)}`
ErrNotFound -> 404 Not Found :  `if errors.Is(err, storage.ErrNotFound) {w.WriteHeader(http.StatusNotFound)w.Write([]byte("Unknown code"))}`

## Part 3
### Locking choice
As I've explained in Part 1 my locking choice were to use RWMutex because of reading happens more frequently than writing to the map(shortrening a new URL) so I didn't need to chaing anythin in this part

### Timeout values and expected slow-client behavior
URL shorteners handle tiny payloads, requiring strict limits to maintain high throughput and protect server resources.
read timeout is 5s for receiving the user's request so we drop too slow connections and attacks fast
write timeout is 3s for doing the map lookup or save and sending the redirect back to them (maybe when I change to DataBase I have to increase it) I keep write low because memory is instant so if it takes longer your app is frozen
idle timeout is 60s for keeping the connection open in case the user sends another request right away 


### Eviction cap 
I didn't implement this optional part because I didn't find it rational. To do it right, we'd need to balance data age, usage frequency, and the last time it was accessed. We could just delete the least-used item, but finding it in the map takes O(n), which is really inefficient just to insert a new link. We could delete a batch of the least-used items in O(N log K) time, but that still adds overhead. Another option is a hash map with a doubly linked list to move recently used items to the head. However, since we use an RWMutex, this makes our locking strategy pointless—every time we read a URL, we'd need a write lock to update the list, making it very slow. All that said, the correct way is to write the data to the hard disk, not delete client data.

## Part 4
### Storage choice
I have used GORM + Postgres for my database because Postgres automatically handle concurrency which is it's advantage over file also looking up in a data base (especially when useing indexes) is much faster than looking in a file

### Schema/models and migrations && How `created_at` is stored.
I use a surrogate PK as we didn't have any unique integer (integer PK are better than strings), the urls are stored as text type of Postgres (I didn't specify one but GORM we look at the type and choose best one for it which in this case is text , we know URLs can get very long that why we use Postgre text type) , for short url I used varchar(15) I personally belive that char(6) is enough but in real productions maybe our shortened code size needs to get more space  , for create at it send the data base a timestamp time becuase when I wrote `autoCreateTime` it would converted it to UTC so now I send it timestamp type so it won't change it . I set it's time to `Asia\Tehran` before inserting it to the database (it is automatically called when `s.db.Create(&link)` is runned) 

### Crash Safety and Atomicity 
because that we used Database (postgre) instead of file our database is ACID so (A for atomicity and D fir durability) when we  call `s.db.Create(&link)` insertion of all attributes happens in a single transaction means they all get inserted or none.as for crash saifty postgres writes ahead meaning when our Save method is finished (successfully) the data is written to the hard disk the and for concurrency the data base automatically put locks on rows not on database (like what we nearly had to do with the map)

### Idempotency + Persistence 
my save method check if the url it's trying to insert is already in the data base or not (with `GetByURL()`) and if it's written it will return the written value if it exist and if doesn't it will generate a new code for it and save it , because they are beaing saved on a disk and not in a memory and our program connect's to the data base every time that it's runned the written data won't be lost