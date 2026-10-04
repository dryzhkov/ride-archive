# Core API v1

Updated September 25, 2026: GPX parsing and entry attachment added. [OpenAPI 3.1 contract](openapi.json).

All private routes start with `/api/v1`. The server selects the owner from trusted configuration. Requests cannot set owner_id. Lists use `limit` (1–100, default 50) and `offset` (0–100000, default 0); results are ordered by created_at then id, except history which uses revision. Responses are `{ "items": [], "next_offset": null }`; a numeric next_offset means another page exists. Offset pagination does not provide a snapshot across concurrent edits. The initial API does not expose results past offset 100000.

JSON writes require `application/json`, accept one object up to 32 KiB, and reject unknown fields. IDs and timestamps are server-generated. Names/titles must be nonblank and at most 200 Unicode characters. Error bodies are `{ "error": { "code": "…", "message": "…" } }` with 400 validation, 401 authentication, 403 origin/host, 404 missing record, 409 stale revision, 413 size, 415 content type, or 500 internal failure.

| Method | Path | Behavior |
|---|---|---|
| GET | `/healthz` | Process liveness, public |
| GET | `/readyz` | Database readiness, public; startup already verified migrations |
| GET | `/api/v1/me` | Current owner profile |
| GET / POST | `/api/v1/trips` | List / create `{ "name": "…" }` |
| GET / POST | `/api/v1/bikes` | List / create `{ "name": "…" }` |
| GET / POST | `/api/v1/entries` | List / create. JSON create remains available; the UI uses multipart form fields `title`, optional `trip_id` / `bike_id`, and required `file` to atomically create an entry, preserve/parse the GPX, and attach its non-empty paths. |
| GET | `/api/v1/entries/{id}` | Owner-scoped entry detail |
| PATCH | `/api/v1/entries/{id}` | Revision-checked title/trip/bike update |
| GET | `/api/v1/entries/{id}/history` | Paginated accepted user changes with before/after snapshots |
| GET | `/api/v1/entries/{id}/map` | Entry's linked path summaries and original coordinate segments for map display |
| POST | `/api/v1/entries/{id}/sources` | Multipart `file` plus current `revision`; atomically preserves/parses the GPX and attaches its non-empty paths to an existing entry |
| GET / POST | `/api/v1/sources` | List / preserve original bytes |
| GET | `/api/v1/sources/{id}/original` | Download exact original bytes |

## GPX attachment

- `POST /api/v1/sources/{id}/parse` reads the preserved file and returns `{ "items": [PathSummary] }`. Repeated calls reuse the same parser-version/path IDs. Failures return 422 without losing the original or leaving a partial parse.
- `GET /api/v1/sources/{id}/paths` lists that source's parsed paths (empty before parsing).
- `GET /api/v1/entries/{id}/paths` lists attached path summaries including source filename, track name, point count, and segment count.
- `POST /api/v1/entries/{id}/paths` takes `{ "path_id": "…", "revision": 1 }` and returns the updated entry.
- `DELETE /api/v1/entries/{id}/paths/{pathID}?revision=2` removes only the relationship and returns the updated entry. Original bytes and parsed geometry remain available for reattachment.

Each actual link change increments the entry revision, appends entry history, and records the path ID/action/user in an append-only link audit table in the same transaction. Repeating an already-current link/unlink with the current revision is a no-op. Stale revisions return 409. Foreign-owner entries/paths return 404, and composite database keys enforce ownership as well.

GPX 1.0/1.1 or unnamespaced GPX tracks/routes are supported. Track segments and point order are preserved in stored geometry JSON; GPX routes get one logical segment. Source locators identify tracks/routes. Coordinates and elevation must be finite and in valid coordinate ranges. Raw time strings are preserved, not interpreted or repaired. Limits: 10 MiB source, 1000 paths, 250000 points. Invalid geometry rejects the entire parse; waypoint-only files have no attachable track/route and return 422. Original files retain all unsupported fields/extensions. Waypoint normalization, date interpretation, metrics, maps, and evidence classification remain pending.

New entries are always `unclassified`. `recorded` and `route_reference` exist in the domain vocabulary but are not accepted by this API or database migration until evidence validation is implemented. No client may write distances or observed dates.

PATCH requires the last observed `revision` and at least one editable field. Omitted associations stay unchanged; null clears them. Title cannot be null. Each successful create/edit writes an accepted user-origin history record in the same transaction. Missing or foreign-owner trip/bike references fail validation without disclosing which case occurred. A stale revision returns 409 and does not append history. The client should reload and reapply its intended edit.

Source POST uses raw bytes with `Content-Type: application/gpx+xml` and `X-Filename` containing a filename without directories. It accepts 1 byte to 10 MiB and preserves bytes without GPX validation. The response is `{ "source": {…}, "duplicate": false, "parse_status": "not_started" }`. A new source returns 201; byte-identical content within the same owner returns 200 and records another import occurrence, including its filename. Source upload itself does not convert files to entries or tracks. The UI opens the attachment form and calls the parse endpoint for the selected file. Source download uses a safe generic filename; the first import filename is returned as original_filename in source JSON. Later import filenames remain in the database for future provenance/export APIs.

The original BLOB is not present in list JSON. SHA-256 and byte length are calculated by the server. Immutable originals are stored transactionally in SQLite in this slice. No original-file update endpoint exists. Deletion and full archive export/restore await explicit contracts.

Local loopback mode has no login. Setting `ARCHIVE_API_TOKEN` requires a bearer token on every `/api/` request. Non-loopback startup requires a token at least 32 characters long. Use TLS through Fly's proxy in hosted mode. This initial personal-token mechanism maps to one configured owner; it is not multiuser authentication. Browser cross-origin writes are rejected, and no CORS access is enabled.
