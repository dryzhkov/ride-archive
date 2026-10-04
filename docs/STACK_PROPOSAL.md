# Initial stack and backend foundation

September 25, 2026 · Go + SQLite + Svelte/TypeScript confirmed by the user; operational and ingestion details remain under design

## Selected stack

Go, SQLite, and a small Svelte + TypeScript frontend built as static assets. Serve the frontend and JSON API from one process on one Fly Machine with a persistent volume. No separate frontend server, Redis, message broker, or agent framework is needed initially.

Rust with Axum, Tokio, Serde, and SQLx is an equally viable product choice if Rust experience or stronger domain types are a priority. My preference for Go here is implementation simplicity for an application dominated by HTTP, SQL, file parsing, and external model calls. Neither language's performance is likely to determine the first MVP's usefulness. Python remains useful for the existing analysis and prototype, but does not provide a compelling reason to choose it for the new service.

The Go implementation uses net/http, database/sql, modernc.org/sqlite, embedded versioned SQL migrations, and explicit request/response structs. Dependencies are pinned in go.mod/go.sum and web/package-lock.json. SQL stays in repositories, not handlers or the NL translator. Original sources are immutable BLOBs in SQLite for transactional storage in this first slice. See [API reference](API.md) for implemented routes, including source preservation and entry history.

## Database and geographic queries

SQLite is sufficient for a personal archive with modest writes and one application instance. Enable foreign keys, use WAL mode, bound transactions, configure a busy timeout, and serialize migrations. The existing PoC database is not the new schema; create a separate database and later implement an explicit importer.

Initial map-area queries can use bounding boxes to select candidates followed by an exact geometry predicate. Bounding-box overlap alone must not claim the rider passed through an area. Postgres with PostGIS is worth selecting early if proximity, intersections, or geographic aggregation become central to the accepted question corpus. Postgres is also a better fit when multiple writable app instances or substantial concurrent ingestion are required.

Moving from SQLite to Postgres later is possible but requires actual migrations, SQL changes, and regression tests. Repository boundaries contain the work; they do not make the databases interchangeable. Confirm spatial requirements before general ingestion, as the project plan requires.

## Fly hosting and cost

Checked September 25, 2026 against [Fly pricing](https://fly.io/pricing/) and [volume documentation](https://fly.io/docs/volumes/overview/).

- The pricing page lists a shared CPU Machine with 256 MB at $1.94/month and a 512 MB shared-2x preset at $3.89/month. Actual compute rates depend on region and configuration; memory sizing requires measurement.
- Volumes cost $0.15 per provisioned GB/month. A 3 GB volume adds $0.45/month. These examples put small SQLite app compute/storage around $2.39–$4.34/month before backup storage, network, and model usage.
- Managed Postgres Basic is $38/month plus $0.28 per provisioned GB/month, in addition to app compute. It includes high availability, automated backups, and connection pooling.
- Self-managed Postgres can cost less, but we own database operations and recovery. It does not serve the stated simplicity goal as well as SQLite or managed Postgres.

For SQLite, mount the database and originals under a persistent directory such as /data. The Machine root filesystem is ephemeral. Run a single writable instance; adding Machines creates independent volumes, not shared database storage. Run migrations where the volume is mounted, not in a Fly release command.

This topology accepts downtime during deployment or host failure. Fly volume snapshots are secondary protection, not the sole backup. Before hosting real archive data, implement consistent SQLite backups and original-file backups to storage independent of the Machine, and verify a restore. Decide backup frequency and acceptable data loss explicitly. Autosuspend is optional and must account for import jobs and backup scheduling.

## First implementation slice

Start with core catalog and entry APIs, database migrations, and tests. Keep general GPX parsing, geometry composition, NL translation, and the final structured query contract separate until their semantics are settled.

| Model | Initial responsibility |
|---|---|
| User | Stable archive owner; owner identity comes from trusted server context |
| Trip | ID, owner, name; entries carry an optional trip_id |
| Bike | ID, owner, display name; an optional entry association, still a proposed cardinality |
| ArchiveEntry | ID, owner, title, kind, optional trip/bike links; no ambiguous distance field |
| Source | Stable original-file identity, owner, checksum, byte length, stored bytes/reference |
| SourceParse / Path / EntryPath | Subsequent evidence slice preserving multiple paths per file and multiple recordings per entry |

Do not replace EntryPath with a permanent one-file/one-entry foreign key. Source preservation and evidence attachment are distinct operations. Metadata-only entries begin unclassified; declaring an observed recording or reference route must eventually validate its attached evidence in the same transaction.

Implemented initial metadata routes (additional source/history routes are documented in the API reference):

| Route | Contract |
|---|---|
| GET /healthz | Process health; no private archive data |
| GET /readyz | Database reachable and required migrations applied |
| GET /api/v1/me | Current owner profile |
| GET, POST /api/v1/trips | List/create trips |
| GET, POST /api/v1/bikes | List/create bikes |
| GET, POST /api/v1/entries | List/create metadata-only unclassified entries |
| GET /api/v1/entries/{id} | Entry detail |
| PATCH /api/v1/entries/{id} | Update title and optional trip/bike associations with revision checking |

Use server-generated IDs and timestamps, bounded pagination, JSON errors with stable codes, and strict input validation. Clients cannot select ownership through request bodies. Enforce same-owner relationships in database constraints as well as application validation. Validate factual changes and write user provenance atomically. Expose neither arbitrary SQL nor filesystem paths.

Tests should cover fresh/repeated migrations, foreign keys, cross-owner access and relationships, missing/null associations, stale edits, invalid payloads, pagination, persistence across reopening, and API/database integration. Source ingestion later adds byte-preservation, duplicate, multi-track, and rollback tests.

Initial development can bind to loopback with an explicitly configured local owner. Public hosting requires an authentication boundary; choose that before deploying private GPS data. No Fly resources are provisioned by this proposal.

## Pending choices

1. Which geographic queries to support initially, and when they justify moving to PostGIS.
2. Long-term hosted authentication, independent backups, and acceptable downtime/data loss. An initial personal-token boundary is implemented; backup automation is not.
3. Evidence parsing/attachment and remaining core-model policies before the corresponding endpoints ship.
