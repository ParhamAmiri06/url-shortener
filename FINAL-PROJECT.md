# Final project — URL shortener

Build a small HTTP service that **shortens long URLs** and **redirects** short codes to the original link (like bit.ly or t.co). You implement it in phases; **Parts 1–4 are required**, **Parts 5–6 are optional bonus**.

**Stack:** Go 1.22+, `cmd/server` + `internal/`.

### Dependencies

| Kind | Policy |
|------|--------|
| **Web server / router** (Gin, Echo, Chi, Fiber, `net/http`, etc.) | Allowed — **do not** list in `## Dependencies` |
| **GORM + SQL driver** (Part 4 persistence) | Allowed — **do not** list in `## Dependencies` |
| **Every other module** in `go.mod` | **Required:** add to **`DECISIONS.md` → `## Dependencies`**: import path, what it does, and **why** you did not use stdlib or the allowed stack above |

`README.md` must explain how to run the app (including DB if you use GORM: DSN, migrations, docker-compose, etc.).

---

## What you are building

| Flow | Behavior |
|------|----------|
| **Shorten** | Client sends a long URL; service returns a short **code** and **short_url**. |
| **Redirect** | Browser or `curl -L` hits the short link → **302 Found** with `Location` set to the long URL. |
| **Lookup (Part 2)** | JSON metadata for a code without redirecting. |

Skills from the bootcamp: HTTP handlers, packages, errors (`%w`, `errors.Is`), tests, `-race`, **coverage**, benchmarks, persistence.

---

## Submission

Submit a **single repository** (or zip) containing:

| Item                                                                     | Required |
| ------------------------------------------------------------------------ | -------- |
| Complete Go module (`go.mod`, `go test ./...` passes)                    | Yes      |
| `README.md` — how to run, flags, example `curl`                          | Yes      |
| **`DECISIONS.md`** — every design choice listed under each part below    | Yes      |
| `go vet ./...` clean                                                     | Yes      |
| `go test -race ./...` clean                                              | Yes      |
| **Test coverage ≥ 70%** (statement coverage, all packages under `./...`) | Yes      |

### `DECISIONS.md` (mandatory)

For **each part you claim points for** (1–4 required; 5–6 if attempting bonus), add **`## Part N`**. Document **every** bullet under that part’s **“Decisions you must document”**. If you picked an obvious default, say so and why. Any other choice (code gen, locking, schema) belongs here too.

Include **`## Dependencies`** only for modules that are **not** an allowed web router/framework or GORM (+ its SQL driver). One bullet per module with a clear reason.

Example:

```markdown
# Design decisions

## Dependencies
- github.com/google/uuid — …

## Part 1
- Code generation: …

## Part 2
- Idempotency: …
```

Graders may deduct from a part if work is done but the matching decision is missing from `DECISIONS.md`.

Copy the checklists below into your repo (e.g. `CHECKLIST.md`) and mark `[x]` what you believe is complete.

---

## Scoring overview

| Part | Title                            | Points  | Required |
| ---- | -------------------------------- | ------- | -------- |
| 1    | MVP — shorten & redirect         | 25      | Yes      |
| 2    | API, errors, `Store` interface   | 25      | Yes      |
| 3    | Performance & measurement        | 25      | Yes      |
| 4    | Persistence                      | 25      | Yes      |
| 5    | Scale to millions (architecture) | +10 max | No       |
| 6    | Production habits                | +10 max | No       |
|      | **Base total**                   | **100** |          |
|      | **With bonus cap**               | **120** |          |

---

## Suggested layout

```text
url-shortener/
  go.mod
  cmd/server/main.go
  internal/...
  README.md
  DECISIONS.md
```

Use `cmd/` for entrypoints and `internal/` for packages you design (names and boundaries are up to you). Keep `main` thin.

---

## Part 1 — MVP (25 points)

Single process, in-memory `code → longURL` map, protected by `sync.Mutex` or `sync.RWMutex`.

### HTTP API

| Method | Path           | Status              | Notes                                                                                                                                                                                               |
| ------ | -------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `POST` | `/api/shorten` | **201 Created**     | Request JSON: `{"url":"https://example.com/path"}`. Response JSON: `{"code":"a1B2c3","short_url":"http://localhost:8080/a1B2c3"}`. **Same URL again → same `code` and `short_url`** (still **201**) |
| `GET`  | `/{code}`      | **302 Found**       | Header `Location: <long url>`                                                                                                                                                                       |
| `GET`  | `/{code}`      | **404 Not Found**   | Unknown code                                                                                                                                                                                        |
| `POST` | `/api/shorten` | **400 Bad Request** | Missing `url`, empty string, or scheme not `http` / `https`                                                                                                                                         |

### Domain rules

- **Same long URL → same code:** every `POST /api/shorten` with the same URL (after your normalization rules) must return the **same** `code` and `short_url`, not a new one.
- Generate **code**: 6–8 characters, URL-safe `[a-zA-Z0-9]` (or base62 from a counter) for **new** URLs only.
- **Collision:** retry, return error, etc. — document in `DECISIONS.md` (distinct URLs must not share a code).
- **Validate URL:** non-empty; only `http` or `https`. Do **not** HTTP-fetch the long URL (no SSRF).
- Flag **`-base`** (default `http://localhost:8080`) when building `short_url`.
- Flag **`-addr`** (e.g. `:8080`) for listen address.

### Example

```bash
go run ./cmd/server -addr :8080 -base http://localhost:8080

curl -s -X POST localhost:8080/api/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://go.dev/doc/"}'

curl -sI localhost:8080/<code>
# expect: HTTP/1.1 302 Found
#         Location: https://go.dev/doc/

# same URL again — same code in the JSON response
curl -s -X POST localhost:8080/api/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://go.dev/doc/"}'
```

### Checklist

| Done | Pts | Requirement                                                                                           |
| :--: | :-: | ----------------------------------------------------------------------------------------------------- |
| [x]  |  4  | `POST /api/shorten` returns **201** with `code` and `short_url`                                       |
| [x]  |  3  | **Idempotency:** second `POST` with the **same URL** returns the **same** `code` (test required)      |
| [x]  |  4  | `GET /{code}` returns **302** with correct `Location`                                                 |
| [x]  |  3  | Unknown code → **404**; bad/missing URL → **400**                                                     |
| [x]  |  3  | URL validation and no server-side fetch of long URL                                                   |
| [x]  |  2  | Codes 6–8 chars for new URLs; collision strategy for **new** codes only                               |
| [x]  |  2  | `-base` flag used for `short_url`                                                                     |
| [x]  |  2  | `httptest`: shorten + redirect; table tests for bad URL and unknown code                              |
| [x]  |  2  | Concurrent test (include concurrent duplicate shorten for same URL); **`go test -race ./...`** passes |

### Decisions you must document (Part 1)

- How you decide two URLs are “the same” (normalization: trailing slash, case, etc.).
- How same URL → same code is stored (e.g. long URL → code index).
- How codes are generated for **new** URLs only.
- Collision handling.
- Mutex type and which methods lock.
- Package layout.

---

## Part 2 — API & errors (25 points)

### HTTP API (add)

| Method | Path                   | Status            | Body                                                        |
| ------ | ---------------------- | ----------------- | ----------------------------------------------------------- |
| `GET`  | `/api/v1/links/{code}` | **200 OK**        | `{"url":"…","created_at":"2026-01-15T12:00:00Z"}` (RFC3339) |
| `GET`  | `/api/v1/links/{code}` | **404 Not Found** | Unknown code                                                |

### Requirements

- Sentinel errors: `ErrNotFound`, `ErrInvalidURL`.
- Wrap with `%w`; handlers use `errors.Is` → **404** / **400**.
- **`Store` interface** at the consumer (handlers or `main`); in-memory impl + fake/mock in tests.
- JSON request/response bodies; routes may use **`net/http` or an allowed web library**.
- Part 1 **idempotency** must still hold (same URL → same code) through the `Store` layer and after refactor.

### Checklist

| Done | Pts | Requirement                                                                             |
| :--: | :-: | --------------------------------------------------------------------------------------- |
| [x]  |  5  | Metadata route **200** / **404** with correct JSON                                      |
| [x]  |  4  | `ErrNotFound`, `ErrInvalidURL` from store/domain                                        |
| [x]  |  4  | `%w` + `errors.Is` in HTTP mapping                                                      |
| [x]  |  5  | `Store` interface + fake used in tests                                                  |
| [x]  |  4  | Tests for metadata route and error mapping                                              |
| [x]  |  3  | Test or note in README: idempotency still works via `Store` / HTTP after Part 2 changes |

### Decisions you must document (Part 2)

- `Store` interface location and methods.
- How errors become status codes and response bodies.

---

## Part 3 — Performance & measurement (25 points)

One binary; measure before imagining clusters.

### Requirements

- Server **timeouts** (`http.Server` fields or your framework’s equivalent).
- `RWMutex` if redirects ≫ creates — or stay with `Mutex` and explain why.
- Benchmarks: `go test -bench=. -benchmem ./...` on shorten and redirect paths.
- README: **one benchmark line** (ns/op, allocs/op) and **one profiling insight** (`-cpuprofile`, `go tool pprof -top`, or similar).
- Optional: cap max links in memory + eviction — if used, document; if not, say why.

### Checklist

| Done | Pts | Requirement                                              |
| :--: | :-: | -------------------------------------------------------- |
| [x]  |  5  | Server timeouts configured                               |
| [x]  |  5  | `Mutex` vs `RWMutex` matches behavior and `DECISIONS.md` |
| [x]  |  5  | Benchmarks for shorten and redirect                      |
| [x]  |  5  | README: benchmark line + profiling insight               |
| [x]  |  5  | Tests and `-race` still green                            |

### Decisions you must document (Part 3)

- Locking choice.
- Timeout values and expected slow-client behavior.
- Eviction cap (if any).

---

## Part 4 — Persistence (25 points)

Same HTTP API; swap in a **durable** `Store` behind the Part 2 interface.

### Requirements

- Second **`Store` implementation** with **durable** storage. Preferred options:
  - **GORM** + SQL database (SQLite, PostgreSQL, MySQL, …), or
  - **File-backed** store (`encoding/json`, `os`, …) without an ORM.
- **Load on startup** so links survive restart.
- **Persist before 201** (transaction commit, fsync, atomic rename — document).
- Tests: temp DB/file dir; **restart simulation** (new store instance reads data written by a previous one).
- Flag or env to choose memory vs persistent store.
- `go test -race ./...` still passes (use in-memory SQLite or testcontainers only if you document extra deps under **`## Dependencies`**).

### Checklist

| Done | Pts | Requirement |
|:----:|:---:|-------------|
| [x] | 6 | Persistent `Store` (GORM+DB or file-backed) |
| [x] | 5 | Startup load |
| [x] | 5 | Create persisted before response |
| [x] | 4 | Restart test (temp DB or temp files) |
| [x] | 3 | Config selects memory vs persistent store |
| [x] | 2 | `-race` clean with persistent store |

### Decisions you must document (Part 4)

- Storage choice: GORM + which DB, or file format.
- Schema/models and migrations (if GORM).
- Crash safety and atomicity.
- How `created_at` is stored.
- Idempotency (Part 1 rule) + persistence: same URL after restart → same code.

---

## Part 5 — Millions of requests (optional, +10 max)

No cluster required. Credit for **architecture in `DECISIONS.md`** and/or optional code.

### Topics to address (if claiming points)

- **Stateless app:** many Go replicas behind a load balancer.
- **Shared store:** Redis, PostgreSQL, etc. — not per-pod memory.
- **Read path:** CDN caches **302** for hot codes; cache TTL and stale redirect tradeoffs.
- **Write path:** rate limits, async queue, pre-generated code pools — pick at least one.
- **Sharding:** partition codes when one database is not enough.

### Checklist (any combination, cap 10 pts)

| Done | Pts | Requirement |
|:----:|:---:|-------------|
| [x] | 4 | `DECISIONS.md`: LB → N apps → shared store |
| [x] | 3 | CDN / edge caching for redirects |
| [ ] | 3 | Write-path scaling (rate limit, queue, or code pool) |
| [x] | 3 | Sharding / partitioning strategy |
| [ ] | 4 | **Bonus code:** cache layer, load-test script + README numbers, etc. |

### Decisions you must document (Part 5)

- Everything you claim above, with tradeoffs (cost, consistency, stale redirects).

---

## Part 6 — Production habits (optional, +10 max)

### Topics

- Graceful shutdown: `signal.Notify`, `Server.Shutdown` with timeout.
- Rate limit on `POST /api/shorten` (any implementation; extra libs → **`## Dependencies`**).
- Blocklist/allowlist for dangerous long URLs (open redirect / phishing).
- Metrics/logging: RPS, errors — and **never** log full URLs with secrets in query strings.

### Checklist (any combination, cap 10 pts)

| Done | Pts | Requirement |
|:----:|:---:|-------------|
| [ ] | 3 | Graceful shutdown working + documented |
| [ ] | 3 | Rate limit on create |
| [ ] | 2 | Domain policy in `DECISIONS.md` |
| [ ] | 2 | What you log vs never log |
| [ ] | 4 | **Bonus:** pprof/metrics behind flag, or structured logging |

### Decisions you must document (Part 6)

- Shutdown drain; in-flight requests.
- Rate limit parameters; blocklist rules.
- Observability approach.

---

## Out of scope (not required)

User auth, custom domains, full click-analytics pipeline, production K8s manifests. Redis or other clients are fine if used and **documented under `## Dependencies`**. You may **describe** scale-out in Part 5–6 without implementing it.

---

## Grader quick verify

```bash
cd your-repo
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -n1
# total: (statements) must be ≥ 70.0%
go run ./cmd/server -addr :8080 -base http://localhost:8080
# use curl examples in Part 1
```

Submissions below **70%** total statement coverage fail the global requirement even if part checklists are complete. Paste the `go tool cover -func=…` total line into `README.md` (optional but helpful).

---

## Academic integrity

Your own implementation. Discuss ideas with classmates; do not submit another student’s repo or copy a public solution verbatim. Cite any adapted snippet in `DECISIONS.md`.

**AI tools:** Do **not** use AI as your coder (no generated handlers, tests, or whole modules pasted into the project). AI is allowed only as **search** (docs, APIs, error messages) and **consultant** (concepts, tradeoffs, debugging hints you implement yourself). If you used AI in an allowed way for a specific question, say so briefly in `DECISIONS.md`.
