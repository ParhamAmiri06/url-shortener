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

## Part 5
### Load Balancer
To implement this, the first change is that users won't send requests directly to our server IPs. They will only know our Load Balancer (it works as a reverse proxy, so the users don't know about our servers' IPs, which could even be private IPs). We can mirror the route that the user is requesting on the Load Balancer and forward the exact same route to the server. The server won't see the real user IP directly, but if we want to know the user's IP for something like rate limiting, we can have the Load Balancer inject it into an HTTP header. about how the Load Balancer chooses which server a request should get sent to, we can use the Least Connections algorithm, the Load Balancer monitors how many active connections each server has and routes new traffic to the one with the least.
The reason why we choose to use this algorithm instead of something like Round Robin is that we can use it with an Auto-Scaler . Its job is to monitor metrics like CPU usage, memory, etc. When our Auto-Scaler detects a high usage of CPU, it turns on a new server. Once the server is ready, the Auto-Scaler enrolls it in the Load Balancer. This works for low traffic as well, when the Auto-Scaler sees that our servers are consuming very few resources, it decides to turn one off and removes it from the Load Balancer (it first tells the Load Balancer not to send new connections to this server, and waits until the server finishes processing its active connections before shutting it down).
For the database, we don't need any changes because I have used PostgreSQL. Multiple servers can connect, read, and write data to it simultaneously (they can and should do it using the same connection credentials).

### Sharding Database
For database sharding, the approach that I would use is to shard the databases based on the short URL. For example, if we have 4 databases, we can pass a short URL through a hash function (something like modulo 4) and decide which database to check; as I have used base-62 strings, we can hash them to integers first, so that is okay. Alternatively, we can just range-partition them; for example, we can say codes that start with 0 to 9 and a to e are stored in database 1, etc.

With this approach, we can make the redirect process much faster because we know exactly which database to check. However, for shortening a long URL (when a user makes a POST request), it will take a longer time because we have to check if that long URL is shortened already across all shards. (We could simply stop checking and remove the constraint that each long URL should be exactly mapped to one short URL; if we do that, I would have to remove the unique constraint on the original URL in the database).

If we maintain the check and it's already shortened, we return the stored code. If not, we generate a new code, check if that new code exists (handling `ErrCollision` by generating a new one until there are no collisions), and store it in the corresponding database. This approach makes redirecting faster but generating a code for a long URL slower. I think this is acceptable because redirection happens much more frequently than generating a new shortened URL, so it's okay if creation takes a bit more time.

We could do something to make the generation faster, but I don't find it rational for our scale: we could have 2 separate database clusters, one sharded based on the long URL and one sharded based on the short URL. That way, we could check if the long URL exists in the database faster, but then our storage would take 2 times the space, and we would have to insert each new URL into 2 databases.

### CDN and Edge Caching
To optimize the read path (resolving short URLs to long URLs) and reduce load on our servers, we can place a Content Delivery Network (CDN) in front of our Load Balancer.
Anycast vs Geo-DNS Routing
When using a CDN, we must choose how users are routed to the nearest edge server:
Anycast Routing: Multiple CDN servers globally share the exact same IP address. The internet's natural BGP routing take the user to the physically closest server based on user's network ip.
Geo-DNS: The DNS server looks at the user's IP address, determines their geographic location, and returns the unique IP address of a specific server in that region.
as URL shortener's primary goal is speed. Anycast uses the internet's natural physical infrastructure to route users to the closest server instantly.


The 302 Redirect and TTL Tradeoff
When our Go server issues a redirect, we use a `302 Found` status instead of a `301 Moved Permanently`. A 301 is cached indefinitely by browsers, meaning we permanently lose control of that link. 

Using a 302 allows us to tell the cache duration . The CDN will cache the redirect and serve it instantly to users.
The TradeoffIf we introduce a feature allowing users to edit the destination of their short link, the CDN will suffer from "stale redirects." If the TTL is 1 hour and a user edits their link, global users will still be sent to the old destination for up to an hour until the CDN cache expires. We must balance protecting the database (longer TTL) with the speed of link updates (shorter TTL). 
Analytics Tradeoff Additionally, CDN caching prevents our Go server from tracking click analytics, because the requests never reach our backend. To solve this, we would require edge-computing to count clicks.