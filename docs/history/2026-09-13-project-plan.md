# Ride Logger — phased project plan

Draft for review • 13 September 2026 • Planning only

## Practical architecture

**Recommendation:** one TypeScript application, managed Postgres/auth/storage, and a small background worker from the same repository when ingestion requires it. Avoid introducing Python, microservices, a vector database or an agent framework solely for learning.

| Component | Choice and reason |
|---|---|
| UI/application | React + TypeScript using Next.js; responsive web app, server endpoints and shared validation in one codebase. |
| Data/auth/files | Managed Supabase Postgres, Auth and private object storage. It combines these services and supports PostGIS if spatial search later needs it. [Database docs](https://supabase.com/docs/guides/database/overview). |
| Map | MapLibre GL JS for route rendering; licensed hosted tiles and geocoding, provisionally MapTiler. Validate retention/export rights and total costs before choosing a plan. [MapLibre docs](https://maplibre.org/maplibre-gl-js/docs/), [MapTiler search](https://www.maptiler.com/search/). |
| Hosting | One managed Node deployment; worker as a second process only when needed. Choose vendor after checking file-processing limits; no hosting purchase required for planning. |
| Ingestion | Original upload → persisted import status → bounded parse/validation → normalized segments + metrics → optional region lookup → editable draft → save. Failed enrichment does not block saving. |
| Search | Postgres text search and indexed bike/date/region fields. Add spatial indexes when there is an actual spatial query. |
| Jobs | Durable Postgres job/import records, retry count and leases; idempotent processing. No Redis/Kafka initially. |
| AI | Optional later server-side feature translating language into allowlisted filter objects. Standard search remains available. |

Store originals and full ordered point data in private object storage; store metadata, quality flags, metrics and simplified display geometry in Postgres. Do not create one database row per point without a demonstrated query need. Preserve full precision separately from map simplification. Geocoding must not replace user corrections during reprocessing.

Security baseline: ownership checks on every query/download, row-level policies, short-lived media URLs, upload size/point limits, XML external entities disabled, image decoding limits, and no precise location in routine telemetry. Backups must cover object files as well as database rows.

## Initial domain model

| Entity | Responsibility and relationships |
|---|---|
| User | Owns all records; timezone and unit preferences. |
| Motorcycle | Owner, name/model, optional default ride style; archive rather than delete a referenced bike. |
| Import | Source object, checksum, original filename, processing state, failure information, parser version. A file is evidence, not automatically a ride. |
| Track | Imported geometry with ordered segments and timestamps; quality flags and provenance. One import may contain multiple tracks. |
| Ride | A rider-confirmed outing; motorcycle, editable date/timezone, region/title/note, computed metrics and source links. May contain several sequential tracks. |
| RideTrack | Links a track to a ride with ordering and inclusion in metrics. Prevent accidental double counting of parallel recordings. |
| Trip | Optional container of ordered rides; title/note; derived date range and overview. A ride belongs to at most one trip in MVP. |
| Photo | Private original and display variants; belongs to a ride; optional caption/order. A trip gallery derives from its rides. |

**Ride and trip are separate entities.** A ride is an outing, not a file or necessarily a calendar day. A trip groups outings; a trip can use different bikes because the motorcycle belongs to each ride. No separate “Day” entity yet. Deleting a trip ungroups its rides. Do not double-count trip totals in archive statistics.

Multi-track MVP behavior: preview named tracks and select those representing one outing, or save them as separate rides. Reject unsupported combinations clearly. Duplicate files resolve to the existing import for that owner; different exports of the same ride may need manual review. Timestamp-free data requires confirmation/date handling, not fabricated times. Complex splitting and parallel-recorder merging can wait.

## Experiments before a full build

| Priority | Experiment | Decision it informs / proposed exit criterion |
|---|---|---|
| 1 | Personal retention trial from the brief | Compare archive vs journal; next four rides and two meaningful returns. Pause/pivot if neither earns voluntary use. |
| 2 | GPX corpus: about 20 exports across Garmin, onX, Gaia; include technical loops, gaps, multiple tracks, missing timestamps/elevation and duplicates | Account for every valid segment; never synthesize connecting distance across breaks; unknown metrics remain unknown. Record source-specific discrepancies. Requires your actual exports, not yet available. |
| 3 | End-to-end import walkthrough | Measure export separately; aim for under one minute active review/save work. Determine whether single-file import is tolerable. |
| 4 | Naming/geocoding sample | Review broad region suggestions on the corpus; target 90% acceptable without edit, otherwise broaden labels or omit suggestions. Test rural ambiguity and timezone edges. |
| 5 | Phone photo and long-track spike | Verify orientation/formats, memory limits, private thumbnails and responsive maps on representative phone hardware; set upload caps from measured results. |
| 6 | Bounded AI search, after archive utility is demonstrated | About 30 labeled queries, including ambiguity and no-results; target 90% correct filters and zero cross-user disclosure; track latency and per-query cost. Do not ship if it underperforms filters. |

Experiments 2–6 requiring code start only after brief/MVP/architecture review. No source GPX files have been inspected yet. Thresholds are proposed gates, not completed results.

## Milestones and acceptance criteria

Rough planning estimate: 80–140 focused engineering hours after approval, plus real-world observation time. At 6–8 hours/week this is roughly 10–24 weeks. Re-estimate after ingestion/photo spikes; estimate excludes integrations, public sharing and billing.

| Milestone | Scope | Acceptance criteria |
|---|---|---|
| M0 — Decide whether to build | Review documents; compare archive/journal examples and current workflow | Choose primary job; agree MVP/photo scope; document import tolerance. Explicit approval before app implementation. |
| M1 — Prove the data path | Corpus, parser, metric policy, draft import, photo/map spikes | Every fixture handled or rejected with a useful reason; original retained; no gap bridging; duplicate retry creates no duplicate ride; uncertainty visible. |
| M2 — Private vertical slice | Auth, bikes, import review, save and ride detail | Owner can save/reopen a ride; another user cannot read records or objects; failed imports can retry; missing metadata does not become false data. |
| M3 — Earn repeat use | Timeline, filters/search, simple trips, optional photos | Five real retrieval tasks succeed; trip grouping reversible; archive totals count each ride once; photo-free entries look complete on phone and desktop. |
| M4 — Make the archive dependable | Export/delete, backup/restore, limits and operational checks | Export contains originals/photos/metadata; sample restore succeeds; deletion behavior documented and verified; no private coordinates in logs. |
| M5 — Personal pilot and small beta | Next four ride opportunities, then five similar riders | Evaluate voluntary imports/returns and support burden; decide continue, pivot or pause before more features or payments. |

## Prioritized backlog

Priority order within each group; P0 is before implementation, P1 forms the MVP, P2 follows evidence, P3 is deferred.

| ID | Priority | Work item | Dependency / done condition |
|---|---|---|---|
| 01 | P0 | Review primary job, MVP and architecture | User review recorded; unresolved assumptions stay labeled. |
| 02 | P0 | Compare archive and trip-memory examples | Use actual ride history; log task outcomes and import effort. |
| 03 | P1 | Build representative fixtures and ingest policy | After approval; covers missing fields, segments, duplicates and large files. |
| 04 | P1 | Ownership, private storage and motorcycle records | Cross-user access tests pass for records and files. |
| 05 | P1 | GPX import, durable status and retry | Original preserved; preview reliable; malformed/oversized input bounded. |
| 06 | P1 | Date/place review and simple title | Overrides persist; enrichment outage does not block save. |
| 07 | P1 | Ride page and map | Correct segments; clear estimates; acceptable phone performance. |
| 08 | P1 | Timeline and indexed retrieval | Bike/date/place/text queries solve personal retrieval tasks. |
| 09 | P1 | Trip grouping and derived totals | Reversible grouping; no duplicate distance/count totals. |
| 10 | P1 | Optional gallery/cover | Phone format policy tested; metadata private; map-only layout complete. |
| 11 | P1 | Portable export, deletion and restore | Includes stored originals/assets; restore verified before beta. |
| 12 | P1 | Pilot observations and operating-cost baseline | Evaluate retention on ride opportunities, not daily active use. |
| 13 | P2 | Batch import and bulk editing | Promote first if exporting/importing blocks repeat use. |
| 14 | P2 | Natural-language filter experiment | Labeled eval, grounded result links, permission enforcement and fallback. |
| 15 | P2 | Photo matching and voice notes | Only if users add media/notes; tolerate camera clock offsets. |
| 16 | P2 | Payment test and capped subscription | Retained users first; clear storage/export terms. |
| 17 | P2 | Share-page privacy design | Preview, redaction, sanitized photos, revocation and protected originals before publication. |
| 18 | P3 | Spatial overlap and personal atlas | Explicit definition of “same route”; test noisy parallel tracks and direction. |
| 19 | P3 | Recaps, books and integrations | Validate demand, provider access and economics separately. |

## Next collaboration step

Review the recommended direction and scope first. Then inspect a few representative recordings and compare the two product presentations. Your uncertainty about retention is the key open question; it should shape the pilot rather than be treated as agreement with the archive hypothesis.
