# DataStory

**DataStory** is a hackathon-sized monorepo: a Go API that pulls **lineage** and **data quality** signals from [OpenMetadata](https://open-metadata.org/), then asks **Claude** to draft a human-readable **data incident report**. A small React UI renders the report, a simple text lineage trace, and a mock dataset for demos.

Remote: [github.com/bansalbhunesh/Datastory](https://github.com/bansalbhunesh/Datastory)

## Quick start

1. Copy env template:

```bash
cp .env.example .env
```

2. Fill `.env`:

- `OM_BASE_URL` (default `http://localhost:8585`)
- Either `OM_TOKEN` **or** `OM_EMAIL` + `OM_PASSWORD`
- `CLAUDE_API_KEY`

3. Install deps:

```bash
make install
```

4. Start OpenMetadata (Docker):

```bash
make mock
```

This uses the official quickstart compose pinned under `docker/openmetadata/docker-compose.yml` (from the OpenMetadata 1.9.0 tree). See also: [Local Docker Deployment](https://docs.open-metadata.org/quick-start/local-docker-deployment).

5. Run app:

```bash
make dev
```

- Frontend: `http://localhost:5173` (Vite proxies `/api` → `http://127.0.0.1:8080`)
- Backend: `http://localhost:8080`

## API

- `GET /api/ready` — UI readiness (`openmetadata.reachable`, `openmetadata.auth`, `claude.configured`)
- `GET /api/search/tables?q=dim` — table search hits for autocomplete
- `POST /api/generate-report` with JSON `{ "query": "dim_address" }` or `{ "tableFQN": "service.db.schema.table" }`
  - Response includes `source`: `claude` or `deterministic` (draft works without `CLAUDE_API_KEY`)
  - `warnings[]` explains fallbacks (for example Claude errors)
- `GET /api/debug/lineage?q=dim_address` for a fast OpenMetadata sanity check (parsed summary + raw JSON)

## OpenMetadata endpoints used

- Search: `GET /api/v1/search/query` ([Search](https://docs.open-metadata.org/latest))
- Lineage: `GET /api/v1/lineage/table/name/{fqn}` ([Lineage](https://docs.open-metadata.org/v1.12.x/api-reference/lineage/get))
- Quality: `GET /api/v1/dataQuality/testCases?entityLink=...` ([List test cases](https://docs.open-metadata.org/v1.12.x/api-reference/data-quality/test-cases))

## Repo layout

- `backend/` — Gin server, thin `handlers/report.go`, services in `services/`
- `frontend/` — Vite + React + Tailwind; `src/mock/sampleReport.ts` is the demo safety net
- `docker/openmetadata/` — OpenMetadata quickstart compose
- `Makefile` — `make dev`, `make mock`, `make mock-down`

## License

Apache-2.0 (recommended to match OpenMetadata ecosystem norms — change if you prefer MIT).
