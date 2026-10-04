# Ride Archive

Go + SQLite backend, Svelte + TypeScript client. Initial core-model/API foundation; Map views, derived metrics, query execution, and NL translation are not yet connected to this service. The existing Python experiment is preserved under `ride_archive/` and `prototype/`.

## Run locally

Requires Go 1.27+ and Node 22.12+ (Node 26 used for verification).

```sh
npm ci --prefix web
npm run build --prefix web
go run ./cmd/api
```

Open http://127.0.0.1:8080. The server creates and migrates `local_data/core.sqlite3`, separate from the prototype database. No old records are automatically imported.

For frontend development, run the Go server above and `npm run dev --prefix web` in another terminal. Vite proxies `/api` to port 8080. An installed Go toolchain is recommended; this session's temporary verified toolchain is `/private/tmp/ride-archive-tools/go/bin/go` and may be removed by the OS.

## Available workflows

- Create trips and bikes; create, list, and edit metadata entries.
- Group entries into trips, assign a bike, or clear those optional associations.
- Inspect append-only entry history through the API. Edits use a revision number to reject stale changes.
- Preserve original files (up to 10 MiB), deduplicate identical bytes within an owner, and download the exact bytes. Each submission records its original filename and import time.

Use **Attach GPX** on an entry, or **Attach to entry** on an uploaded file. Choose the entry, file, and track/route, then click **Attach track**. Uploads also open this form. Entry **Edit** controls trip and bike membership. Linked tracks appear below each entry; **Unlink** removes the association while preserving the original.

Parsing supports multiple tracks/routes and preserves segment boundaries, coordinate/elevation samples, and raw time strings. Parsing is atomic and repeatable; malformed/unsupported files stay preserved and show an error. Multiple files can contribute paths to an entry, and paths can be reused across entries. New entries remain `unclassified`; no riding dates, distances, or participation are inferred. Whole archive export/restore is still available only in the old PoC, not this service.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `ARCHIVE_ADDR` | `127.0.0.1:8080` | HTTP bind address |
| `ARCHIVE_DB` | `local_data/core.sqlite3` | New SQLite database path |
| `ARCHIVE_WEB_DIR` | `web/dist` | Compiled static assets only |
| `ARCHIVE_OWNER_ID` | `local-owner` | Trusted owner selected at startup |
| `ARCHIVE_OWNER_NAME` | `My archive` | Display name used when creating the owner |
| `ARCHIVE_API_TOKEN` | unset | Personal bearer token; at least 32 characters if set |

Without a token, the server only permits loopback binding and loopback Host headers. Non-loopback startup requires a token. With a token, all `/api/` routes require `Authorization: Bearer …`; enter it into the client when prompted. It stays in browser memory. Health checks and static assets remain public. This is a single-owner access boundary, not a multiuser login system; don't share this token with other users. Changing the configured owner does not transfer existing data.

## API

See [API contract](docs/API.md) and [OpenAPI](docs/openapi.json).

```sh
curl -H 'Content-Type: application/json' \
  -d '{"name":"Touratech Rally"}' http://127.0.0.1:8080/api/v1/trips

curl -H 'Content-Type: application/json' \
  -d '{"title":"First day"}' http://127.0.0.1:8080/api/v1/entries
```

## Verification

```sh
go test ./...
go vet ./...
npm run check --prefix web
npm run build --prefix web
```

Go integration tests use temporary on-disk SQLite databases and HTTP test handlers. They cover owner isolation, actual database constraints, byte preservation, duplicates, migrations, persistence, atomic history, stale revisions, input validation, and pagination.

## Storage and deployment

Migrations are embedded SQL with stored checksums, applied transactionally under an exclusive writer reservation at startup. The repository uses one SQLite connection initially, foreign keys, WAL, a busy timeout, and full synchronous writes. Original files are BLOBs in SQLite for transactional storage and deduplication in this first slice; there is no object-store dependency. Review this choice when corpus size and ingestion are measured.

`Dockerfile` builds the client and a Go binary. `fly.toml.example` is a **single-Machine** deployment template, not a provisioned app. The runtime drops privileges after preparing the mounted volume. Startup migrations run on the app Machine with `/data` mounted.

Before deployment: choose the app/region, set a random personal API token through Fly secrets, provision an `archive_data` volume, and configure/test independent database backups. Use `fly deploy --ha=false` for the first deploy and verify there is exactly one Machine. Do not horizontally scale this SQLite setup: each volume is independent. No off-machine backup job or automated recovery is implemented yet; the template alone is not a durable hosted archive. Neither Docker nor Fly deployment is assumed to have been verified merely by passing local tests.

## Design

- [Living design document](DESIGN_DOC.md)
- [Core domain draft](docs/CORE_DATA_MODEL.md)
- [Stack decision and remaining tradeoffs](docs/STACK_PROPOSAL.md)
- [MVP plan](PROJECT_PLAN.md)
