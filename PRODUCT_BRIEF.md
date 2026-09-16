# Ride Archive — product brief

Updated September 14, 2026 · Direction agreed; implementation details proposed

## Thesis

**Record wherever you want. Bring your data together. Ask your own questions. Take it anywhere.**

Ride Archive turns motorcycle recording exports into a structured, portable personal database. Its main interface is a simple query box: **natural language → inspectable structured query → database → map and matching records**.

Data control and queryability are the value proposition. The product does not depend on sentimental attachment, route completion, fitness analysis or planning.

## Decisions and evidence

| Status | Position |
|---|---|
| Confirmed | Name: Ride Archive. Ownership, consistent structure and querying across sources lead the product. |
| Confirmed | NL translates into structured queries; the map is a result view. |
| Confirmed | GPX-first, minimal manual input, ADV/enduro, pragmatic solo project with AI engineering learning value. |
| Observed | Five exports: three timestamped recordings and two rally-supplied routes you rode. The two-recording map prototype works; you would add rides, but organization alone felt insufficient. |
| Recommended | Local query experiment; versioned JSON query language; deterministic execution; portable export from the start. |
| Unresolved | Most useful recurring queries, import friction, local versus hosted delivery, model provider and willingness to pay. |

The [original brief](docs/history/2026-09-13-product-brief.md) is retained as history. The existing [prototype](prototype/README.md) is a completed map/list baseline; it has no database or NL interface yet.

## Primary user and jobs

Start with riders like you who record across tools and want to explore/reuse their data independently of source applications. Motorcycle riding supplies the domain; a generic personal-data platform is outside scope.

- Ask consistent questions across sources, motorcycles, dates and geographic areas.
- Inspect why a record matched and where each fact originated.
- Reuse queries and export matching geometry/metadata.
- Preserve originals, corrections and enrichment without depending on our UI.

Examples: “September 2026 recordings on the 890 over 50 miles”; “recordings through this selected area”; “recorded on Garmin, exported by Gaia.” Route overlap is a later spatial operation, not a language-model judgment.

## Query-led experience

1. Import exports, preview detected records/quality issues, optionally supply bike/context, save. Unknown data stays unknown.
2. Ask in the text box. Resolve aliases, units and dates; display a readable interpretation and expandable, editable structured JSON.
3. Run the validated query. Initially use an explicit Run action so translation can be inspected.
4. Show the same matching records on the map and in a list. Select a result to inspect matched fields, provenance and source data.
5. Save the query or explicitly export results. Direct structured input works without AI.

Panning the map does not silently change the query. “Use this map area” explicitly adds a condition. Counts and aggregate answers belong in numbers/tables when appropriate. A browse-all/recent fallback remains available.

Ambiguous requests trigger a focused clarification. Unsupported conditions are explained rather than silently dropped. Zero matches, missing data and unsupported queries are distinct outcomes.

## Ownership and trust

- Preserve original GPX bytes and checksums.
- Publish a versioned export schema including normalized records, geometry, annotations, enrichment provenance, metric methods, quality flags and saved queries.
- Provide original files, JSON metadata and interoperable geometry. Document representation limits: GeoJSON alone does not retain all GPX time/extension semantics.
- Test independent re-import; a successful download alone does not prove portability.
- Keep exports readable without our UI, AI provider or subscription.
- Use one documented query contract for the UI and future programmatic access.
- Choose enrichment sources compatible with intended storage/export rights.

Separate **source observations**, **user assertions**, **deterministic derivations** and **external/model suggestions**. Keep source/method/version and overrides.

Our examples make this concrete: Garmin recorder identity and the TE 300 association came from you; GaiaGPS exporter identity came from the file. The onX recording has timing/location anomalies despite a broadly plausible total distance. Derived metrics need explicit definitions and quality flags. See [findings](analysis/FINDINGS.md).

A recording is not necessarily an entire outing. Hells Canyon is part of day 1; Kittitas intentionally ends away from its start. Tyee and Alder Creek are supplied rally routes you rode, with no timestamps: preserve participation and trip context without claiming their geometry is your exact observed path. Route lengths are not measured riding mileage. Neither bike, trip membership, difficulty nor actual moving time should be invented from GPX.

## Scope

| Next experiment | Proposed personal MVP | Deferred |
|---|---|---|
| Normalize two known recordings into a local database | Repeatable GPX import, duplicates, missing-data handling | Automatic integrations and additional source formats |
| Scalar query contract and deterministic executor | NL translation, inspection, clarification and direct structured input | General chat/agent behavior |
| Exact result expectations, errors and unknown cases | Query-driven map/list, basic geographic predicate, saved queries | Trail matching and advanced route overlap |
| Portable export/re-import | Editable annotations, backup/restore, complete export | Photos, journals, recaps and public sharing |
| NL evaluation after deterministic results pass | Local persistence; access control before any remote use | Billing, hosted sync and broader public API |

Broad place names and labels should improve queries but need provenance. Geography lookup is not automatically an AI task. Manual names can bridge the experiment; do not auto-classify terrain/difficulty from speed.

## Validation and positioning

Proposed loop: **import → ask or export → useful inspectable result → extend coverage with more data**.

Compare against a GPX folder plus map viewer or a small database/script. Two samples demonstrate a data path, not retention or broad query accuracy. Continue if you can independently ask useful questions, inspect answers and reuse exports more easily than today; then expand to 10–20 recordings and new questions.

Earlier competitor research is retained in the original brief. This direction needs a focused comparison of export/query/API capabilities. Do not claim all source apps lack APIs: availability, scope and terms need individual verification.

Monetization is unresolved; hosted convenience could eventually be paid while portability stays fundamental. Keep separate from Prerun Moto. See [project plan](PROJECT_PLAN.md) and [query experiment](QUERY_EXPERIMENT.md).
