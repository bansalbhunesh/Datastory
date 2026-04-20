# DataStory Backend (v2)

Production-grade rewrite of the hackathon prototype. Clean architecture, thread-safe OM client, retries, structured logging, deterministic-first incident reports, LLM-as-rewriter with schema validation.

## Run

```bash
cp .env.example .env  # fill in OM creds + optional CLAUDE_API_KEY
make run              # starts on :8080
make test             # unit tests
```

## Layout

```
cmd/server/           entrypoint + wiring
internal/
  config/             env parsing + validation
  logging/            slog + request-id helpers
  errs/               typed errors → HTTP status
  domain/             pure data types
  clients/
    openmetadata/     REST client (retry, singleflight login, thread-safe token)
    llm/              Anthropic client behind a small interface
  services/
    lineage.go        edge-directional BFS (+ safe fallback)
    report_draft.go   deterministic severity + markdown
    report.go         orchestration (errgroup + cache + LLM rewrite)
    cache.go          TTL cache
  api/
    dto.go            request/response shapes
    handlers.go       thin HTTP glue
    router.go         routes + CORS
    middleware/       request id, access log, recovery
```

## Endpoints (backward-compatible)

| Method | Path                    | Purpose                              |
|--------|-------------------------|--------------------------------------|
| GET    | /healthz                | liveness                             |
| GET    | /api/health             | JSON health                          |
| GET    | /api/ready              | OM reachability + auth + LLM status  |
| GET    | /api/search/tables?q=   | table autocomplete                   |
| POST   | /api/generate-report    | incident report (JSON)               |
| GET    | /api/debug/lineage      | parsed lineage for a table           |

## Design notes

- **Deterministic-first**: facts come from OpenMetadata → deterministic draft. LLM only *rewrites* the markdown. If the LLM output fails schema validation (missing H2 sections or severity mismatch), we fall back to the deterministic draft with a warning.
- **Severity is deterministic** — never trusted to the LLM.
- **Retries** on 429/5xx/network with exponential backoff + jitter; 401 triggers one re-login.
- **Concurrency**: lineage + quality fetches run in parallel via `errgroup`.
- **TTL cache** on FQN-resolution + lineage keeps demo latency low.
- **Structured logs** with `X-Request-ID` propagated from header → ctx → every log line.
