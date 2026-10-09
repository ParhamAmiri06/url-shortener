## How to Run

### Prerequisites
You can run the application entirely in memory, or with PostgreSQL. 

### Environment Variables
You can configure the application using a `.env` file or exported variables:
- `USE_DB` (default `true`): Set to `false` to use the in-memory map instead of Postgres.
- `DB_DSN`: The complete Postgres connection string (optional).
- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`: Used to build the DSN if `DB_DSN` is not provided.
- `SERVER_PORT` (default `8080`): The port the server listens on.
- `BASE_URL` (default `http://localhost:8080`): The base domain used to construct short URLs.

### Running the Server
```bash
go run ./cmd/server 
```
if you want to run on a different port you should mention it in .env file

## API Usage Examples

**1. Shorten a URL**
```bash
curl -s -X POST localhost:8080/api/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://go.dev/doc/"}'
# Response: {"code":"a1B2c3","short_url":"http://localhost:8080/a1B2c3"}
```

**2. Redirect to original URL**
```bash
curl -s localhost:8080/a1B2c3
# Expect: HTTP/1.1 302 Found
# Location: https://go.dev/doc/
# Cache-Control: public, max-age=3600
```

**3. Get Link Metadata**
```bash
curl -s localhost:8080/api/v1/links/a1B2c3
# Response: {"url":"https://go.dev/doc/","created_at":"2026-10-09T12:00:00Z"}
```

## Performance & Testing
**Benchmarks:**
- `BenchmarkShorten-24`: 846679 ops, 1267 ns/op, 1856 B/op, 23 allocs/op
- `BenchmarkRedirect-24`: 1922692 ops, 637.1 ns/op, 1265 B/op, 13 allocs/op

*Insight*: Profile captured 3050ms of real time and 3910ms of total CPU sample time
Benchmarking took 70 percent of CPU work while Go runtime (mostly garbage collection)took 30 percent
Test time was split into 1.6s for the redirect benchmark and 1.1s for the shorten benchmark
Almost all redirect benchmark time went to the core http redirect function which took 1510ms
The shorten benchmark bottlenecks were JSON decoding at 310ms and URL normalization at 200ms

