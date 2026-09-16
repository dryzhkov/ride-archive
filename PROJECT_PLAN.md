# Ride Archive — six-step MVP design plan

Updated September 15, 2026. This plan supersedes the earlier experiment implementation sequence. The PoC is accepted as a useful foundation; it is not the final MVP architecture.

## Current state and objective

The local PoC stores five entries (three recordings and two supplied rally routes), executes validated structured queries against SQLite, presents matching map/list results, and exports/restores original files with metadata. Natural-language translation is not connected. General GPX ingestion, the final domain model, and production architecture remain undesigned.

The MVP makes ride data owned, structured, queryable, and portable. Its design must separate persistence, natural-language translation, deterministic query execution, and presentation. Querying and exporting must remain usable independently of an LLM or a particular UI.

## 1. Define the domain and data model

Define source files, tracks/segments/points, rides, trips, bikes, annotations, derived metrics, and provenance. Decide which are first-class entities and their cardinalities before choosing the physical schema. A ride as a real outing and a trip as a group of rides are proposals to review, not settled requirements.

Address multiple recordings per outing, multiple tracks per file, supplied routes versus observed paths, duplicates, missing timestamps, reported participation, and overlapping recordings. Original evidence, user assertions, enrichment, and computed values must remain distinguishable. Supplied route length must not silently become observed riding distance.

**Deliverable:** domain glossary, relationship diagram, logical schema, invariants, and examples mapping the five real files to the model. Include ownership/export requirements and schema evolution considerations.

**Completion criterion:** the model represents the real examples without invented dates, forced one-file/one-ride assumptions, or ambiguous metric ownership.

## 2. Define supported questions and query semantics

Build 20–30 realistic questions with expected interpretations and results. Include scalar filters, any proposed spatial/aggregate needs, missing data, ambiguous names, relative dates, units, unsupported requests, and empty results. Decide the MVP scope explicitly.

Define the result unit (rides, tracks, routes, trips, or other entities), query operators, null semantics, sorting, pagination, limits, and reproducibility. Distinguish schema-valid queries from semantically correct translations. These examples guide the query language and later NL evaluation.

**Deliverable:** versioned query-language draft and a labeled acceptance/evaluation corpus, including clarification and unsupported outcomes.

**Completion criterion:** supported questions have unambiguous executable meanings and expected results; unavailable evidence is not presented as a measured zero or a known fact.

## 3. Design system architecture and API contracts

Complete the architecture before building general ingestion. Separate responsibilities logically; this does not require separate deployed services.

| Layer | Responsibility | Contract to define |
|---|---|---|
| Persistence / database | Preserve original sources, normalized entities, relationships, provenance, and derived values | Storage/repository interfaces, read/write models, transactions, migrations, consistency, and original-file storage |
| NL translation | Convert a question plus supported semantics and relevant catalog context into a proposed query | Versioned translation request; `ready`, `needs_clarification`, or `unsupported` response; query and interpretation; explicit error states |
| Query execution / data fetching | Validate and resolve structured queries, compile trusted database operations, and return deterministic results | Versioned query request and result envelope; entity IDs, geometry projections, evidence, unknowns, pagination, limits, and typed errors |
| Presentation | Collect input and display interpretations/results through a replaceable client | Public application APIs for translation, query execution, import status, catalogs, and export; no dependence on SQL tables or storage-specific display blobs |

The translator targets the public query contract rather than arbitrary database SQL. The executor revalidates model output and direct structured input. Presentation consumes stable response models; map, table, CLI, or another client can use the same capabilities. Domain records and presentation projections are separate concerns.

Specify contract versions, compatibility rules, validation ownership, units/timezones, entity resolution, request identity, timeouts, cancellation, and observable failures. Define trusted ownership context if hosted access is chosen. Clarify where authentication and authorization apply without letting model output grant access.

Choose deployment shape, runtime/language, database/spatial capabilities, file storage, and model integration based on steps 1–2. Document tradeoffs and boundaries in architecture decisions. SQLite/Python are current PoC choices, not automatic MVP commitments. An agent framework is not assumed.

**Deliverable:** component/data-flow diagrams, architecture decisions, machine-readable request/response schemas appropriate to the chosen transport, examples, and contract-test strategy.

**Completion criterion:** translation, execution, persistence, and presentation can be implemented/tested against explicit interfaces; ingestion and export have defined integration points. Remaining architecture decisions are resolved or explicitly deferred with scope consequences.

## 4. Design and implement ingestion and portability

Against the agreed architecture, implement the source-to-domain pipeline: preserve original bytes, identify duplicates, parse multiple tracks/segments, normalize observations, derive versioned metrics, and attach user context. Specify retries, idempotency, partial failures, and correction/reprocessing behavior.

Define full archive and selected-result exports, including original sources, metadata/provenance, and saved-query portability. Restore must preserve identities and supported semantics.

**Deliverable:** repeatable import pipeline, import status/errors, export/restore workflow, and varied GPX fixtures.

**Completion criterion:** arbitrary supported GPX inputs—not just the five seed fixtures—follow the documented rules; duplicate imports and round trips are tested without silent loss or fabricated evidence.

## 5. Implement query execution and NL translation

Implement or adapt the deterministic executor to the agreed query/data contracts first. Validate it against the expected-result corpus. Then connect a model through the translation interface, supplying only the necessary semantic definitions and catalog context.

Keep structured queries independently executable. Validate all proposed queries, check entity references, expose interpretations, and return clarification or unsupported outcomes when needed. Evaluate semantic correctness and dropped constraints, not merely JSON validity; measure latency, cost, and failure categories.

**Deliverable:** tested query API, provider adapter, translation API, and repeatable evaluation report.

**Completion criterion:** deterministic cases return expected results and evidence; NL meets an agreed evaluation threshold, with ambiguity and unsupported behavior verified. Do not select a success threshold after seeing the results.

## 6. Build the replaceable presentation layer and validate the MVP

Build a minimal client using the contracts from step 3: import/status, query input, inspectable/editable structured query, clarification, map/list results, details/provenance, saved queries, and export. Presentation design begins in step 3; this step implements and validates the client.

Keep storage and translation logic outside the UI. Verify that another client or a direct API call can execute the same query and retrieve equivalent results. Test the complete import → query → inspect → export/restore flow with a larger personal corpus and real questions.

**Deliverable:** usable personal MVP and an end-to-end acceptance report, with remaining limitations recorded.

**Completion criterion:** useful independent use on newly imported data, consistent results across clients, portable data, and clear treatment of unknowns and failures.

## Sequencing

Steps 1 and 2 inform each other. Step 3 consolidates them into architecture and strong API contracts before general ingestion implementation. Steps 4–6 build against those contracts; discoveries feed back through explicit revisions rather than ad hoc UI/database coupling. This plan does not authorize or imply a specific hosting provider, model connection, or deployment topology.
