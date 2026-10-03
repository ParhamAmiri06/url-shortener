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