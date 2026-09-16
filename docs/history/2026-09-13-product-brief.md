# Ride Logger — product brief

Draft for review • 13 September 2026 • No implementation approved

## Thesis and decision status

**Turn the recordings you already make into a motorcycle history you can find, revisit, and keep.** A ride should become a useful, attractive entry with only a GPX file and a bike selection. Notes and photos enrich it; they never unlock its basic value.

| Status | Position |
|---|---|
| Confirmed by your request | GPX-first; minimal entry; ADV and enduro; simple architecture; solo side project; AI learning is desirable; no closure aggregation or route-planning core; review before coding. |
| Recommended | Private motorcycle archive; target riders who already record; simple ride/trip model; optional photos in MVP; web first. |
| Assumed, unvalidated | Consolidating recordings is valuable enough to justify repeated exports; a polished archive beats existing tools for a specific recurring job. |
| Unresolved | You answered “Not sure” about what would bring you back. Retention, willingness to pay, photo importance, and tolerable import effort need behavioral evidence. |

“Permanent” should mean original-file preservation, portable exports, tested recovery, and a clear exit path. It cannot mean a promise that a solo-operated service will exist forever.

## Strongest jobs and emotional value

1. **Find a ride:** “When did I ride Middle Fork, on which bike, and where is that track?” Retrieve the record and original file without searching several apps.
2. **Remember a trip:** See several days as one journey, with a map and a few optional pictures. Preserve memories without writing a travel blog.
3. **See a riding life accumulate:** Browse seasons and motorcycles, including short local sessions that otherwise blur together. Create a sense of continuity rather than performance pressure.

The strongest initial user is an ADV/dual-sport rider who already records rides, makes occasional multi-day trips, and has history scattered across devices or apps. Your enduro riding is a valuable second use case; owning two bikes is useful context, not an entry requirement. “All motorcyclists” includes too many people without recordings or an archival habit.

## Competitive landscape

Official product/support pages reviewed for this draft; this is desk research, not hands-on verification. The opportunity column is our inference, not a claim that a competitor lacks every related feature. Pricing and tiers vary; avoid making the concept depend on a competitor's exact price.

| Product | Existing value and overlap | Opportunity to test |
|---|---|---|
| Garmin Connect | Health/fitness activity history, analysis and sharing; activity photos already exist. [Overview](https://connect.garmin.com/?cid=3471500), [photo support](https://support.garmin.com/en-CA/?faq=DVP5X0X0tV0kgEYzqK9zB7). | Organization around motorcycles and multi-day memories across recording sources. A map plus statistics alone is insufficient. |
| Strava | Training history, matched activities, personal heatmaps, routes and cumulative statistics. [Subscription features](https://support.strava.com/en-us/articles/15402044-what-features-are-included-in-a-strava-subscription). | Motorcycle context and memory retrieval rather than fitness progress; no claim that historical maps are novel. |
| REVER | Motorcycle tracking/navigation, GPX import, offline functionality and paid PRO. [FAQ](https://www.rever.co/faqs). | Closest motorcycle benchmark: test whether archive organization and trip presentation justify another product. |
| Relive | Imported recordings, chronological activities and photo-enhanced videos. Imports require GPS and time data. [Import support](https://support.relive.com/kb/guide/en/how-can-i-import-an-activity-to-relive-qQQJOMElbR/Steps/3692503), [Plus](https://www.relive.com/plus?hl=en). | Searchable long-term history and handling incomplete historic files, beyond a replay. |
| Polarsteps | Trip tracking, stories/photos, privacy controls, reels and printed travel books. [Product](https://www.polarsteps.com/). | Strong emotional benchmark. Test whether an archive built from existing motorcycle recordings is meaningfully easier. GPX import support was not verified here. |
| MyRoute-app | Uploaded/recorded tracklogs, folders and a personal tracklog list. [Tracklog manual](https://myrouteapp.freshdesk.com/en/support/solutions/articles/12000002719-manual-tracklogs-in-your-personal-tracklog-list-). | Another direct archive substitute; motorcycle branding and folders alone will not differentiate this product. |

**Potentially underserved combination:** low-effort import from existing recorders + motorcycle context + ordinary rides and multi-day trips + reliable retrieval + ownership of the archive. None of these is a defensible moat individually. Research supports overlap, not proven unmet demand.

## Three directions

| Direction | Main reason to return | Main weakness | Recommendation |
|---|---|---|---|
| Personal motorcycle archive | Save the next ride; find an old one | Export friction; willingness to pay uncertain | Start here |
| ADV trip memory journal | Relive and share trips, eventually buy a book | Seasonal use; photos become central; strong Polarsteps overlap | Pivot here if memories win validation |
| Enduro riding atlas | Recall local loops and compare coverage across visits/bikes | Accurate overlap analysis adds complexity; could become a generic mapping tool | Later experiment |

Keep this concept separate from Prerun Moto during validation. Reuse proven generic components only after inspecting them; do not inherit closure pipelines or route-planning architecture. Reconsider branding when the primary job is clear.

## Retention hypothesis and validation

**Ride → export/import → immediately recognizable entry → retrieve or revisit it later → preserve the next ride.** The reward must exist on save and grow with the archive. Annual recaps are an occasional payoff, not the entire retention loop.

Before building, compare two manually assembled presentations of roughly 12 of your own rides: a searchable logbook and a trip-memory layout. Include local enduro, an ADV trip, and a repeated area. Use existing tools or sketches; obtain approval before implementing a prototype.

- Try five real retrieval questions and compare the effort with your current Garmin/files workflow.
- Measure the full export-to-save effort, not just upload time.
- Observe the next four actual ride opportunities, allowing for seasonality. Record whether you voluntarily archive them and whether you revisit earlier entries for a reason beyond testing.
- Provisional continuation signal: archive at least three of those four rides and return twice to find or revisit something. This is a personal decision aid, not market validation.
- If retrieval wins, prioritize archive/search. If trip memories win, shift toward photo-led trip pages. If neither wins, reduce this to a personal utility or pause rather than adding AI to manufacture engagement.
- After personal validation, recruit five similar riders for the same task comparison; do not assume your enthusiasm transfers to them.

## MVP and subsequent releases

| MVP: a private, usable archive | Next release: driven by observed friction | Longer term |
|---|---|---|
| Sign-in; motorcycle list; preferred bike preselected but visible | Larger batch import with bulk corrections | Recorder integrations and additional file types |
| One GPX per import; explicit handling of multiple tracks and missing data | Easier splitting/combining recordings | Native capture only if justified |
| Editable date, broad region, generated template title, optional note | Photo time/location suggestions; voice-note transcription | Rich journal layouts and printed books |
| Interactive ride map; distance estimate and timestamp span when valid | Elevation, moving-time and stop estimates after validation | Route overlap and repeat detection |
| Month/year timeline; text search; bike/date/place filters | Structured natural-language retrieval | Personal atlas and annual recap |
| Manually group rides into a trip; overview map and ordered ride list | Public sharing with privacy preview | Trip reels and selective exports |
| Optional small photo gallery and cover image | More image formats if MVP conversion proves costly | Collaboration, only with demand |
| Original download; full archive export; deletion; backup/restore | Payments once retention is demonstrated | Best-effort external context, never the core |

**Photos:** recommend a small gallery in MVP because visual/emotional value is part of the thesis. No automatic placement, editing suite or AI captions. Entries without photos must still look finished through maps, typography and clear titles. Validate phone-photo compatibility early; either support HEIC conversion or clearly disclose supported formats before selection.

**Smallest six-month product hypothesis:** import, bike/date/place, attractive timeline/map, retrieval, lightweight trip grouping, optional photos and reliable export. No AI required for that promise. No difficulty questionnaire, terrain inference, navigation, social feed or maintenance system.

## Key user flows

1. **First ride:** add bike → upload → see map and detected fields → correct only what is missing/wrong → optionally add photos/note → save → open timeline entry. Target under one minute of active work after obtaining the file; benchmark separately from export friction.
2. **Later ride:** preferred bike preselected → upload → review → save. Exact duplicate offers the existing entry. No silent bike inference from speed or location.
3. **Trip:** select saved rides → name trip → review ordering → save overview. Grouping never changes original tracks; ungrouping never deletes rides.
4. **Retrieval:** search a known name or filter bike/date/region → inspect matching entries → open ride/trip → download original if needed. “Middle Fork” is only searchable if an indexed name/alias exists; broad reverse geocoding will not reliably identify a trail.
5. **Incomplete GPX:** preview available geometry → show missing timestamp/elevation information → confirm this is an actual ride and enter date if known, otherwise retain as an undated draft. Never convert a planned route into a claimed completed ride silently.
6. **Ownership:** export originals, photos and metadata together; delete a record or account with a clear backup-retention explanation.

## ADV versus enduro

Use one ride model with optional ride style, not separate products. A bike may supply an editable default style; never require classification to save.

| ADV/trip presentation | Enduro presentation |
|---|---|
| Trip overview, days/rides, regions, gallery, combined map | Compact single-session card, local area, date and bike |
| Preserve disconnected daily tracks | Preserve loops and revisits |
| Show recorded distance, not assumed complete itinerary | Avoid treating slow technical movement as stopped |

No terrain, difficulty, engine hours or ride quality can be safely inferred merely from a GPX trace. Recorded time is not engine runtime or a service interval.

## Data trust and AI boundary

GPX permits tracks, routes and waypoints; point time and elevation are optional. Design for missing information rather than zero values. [GPX schema](https://www.topografix.com/gpx/1/1/).

| Information | Treatment |
|---|---|
| Geometry | Preserve original points and segment breaks; flag invalid coordinates and implausible jumps. Do not draw measured connections across gaps. |
| Date | Use valid point timestamps with an explicit display timezone; allow correction. File creation/export date is not a ride date. |
| Distance | Deterministic estimate within valid segments; disclose incomplete tracks. GPS noise and sampling affect accuracy. |
| Duration | Label first-to-last valid timestamp span as elapsed recording span; gaps/pauses mean it is not necessarily riding time. |
| Elevation gain, moving time, stops | Derived estimates needing representative data and documented thresholds; defer from MVP. |
| Region and landmarks | External lookup suggestions, editable and source-tagged. A nearby landmark is not evidence of a visit. |
| Title | Deterministic place/date template first; user text always wins. |

The first worthwhile AI learning experiment is **natural language → validated search filters → deterministic query → links to matching rides**. Test “Oregon trips on the 890” before attempting open-ended chat. Cross-bike route overlap requires a spatial algorithm, not a language model.

Use a small labeled evaluation set, schema validation, ambiguous/no-result cases, latency/cost tracking, and a normal search fallback. Never execute model-generated SQL. Restrict every query to the authenticated user. Treat notes as data, not instructions. AI summaries come later, derived only from verified statistics and user notes; no invented weather, difficulty or emotions. Do not send raw precise tracks or photos to a model by default.

## Risks, privacy and business

| Risk | Initial response |
|---|---|
| Import becomes a chore | Measure export friction first; prioritize imports over enrichment if users stop returning. |
| Attractive but redundant | Compare real retrieval and memory tasks against existing tools before polishing extensively. |
| Inaccurate metrics/names | Preserve provenance, allow overrides, label estimates, tolerate missing data. |
| Sensitive locations | Private by default; owner checks on records and media; no public pages in MVP. Keep coordinates out of logs/analytics. Map/geocoding providers still receive location-related requests; minimize them. |
| Sharing leaks home/trails/camps | Before sharing ships: preview the exact public artifact, redact every visit to privacy zones, remove photo EXIF and prevent original-file access. Endpoint trimming alone is insufficient. Links must be revocable. |
| “Permanent” exceeds solo capacity | Back up database and objects independently, test restore, provide portable exports and a shutdown/export policy. |
| Storage and support erase revenue | Set photo/storage limits, track cost per active archive, avoid unlimited lifetime hosting. |

**Monetization hypothesis:** a small free trial/archive allowance followed by roughly $30–50/year for ongoing archiving and capped photo storage. This is a proposed price experiment, not a revenue forecast. Keep export available when a subscription ends; decide read-only retention terms before selling. Printed books or paid digital recaps could suit infrequent trip users later. Validate payment intent with actual retained users before building billing.

## Review points

1. Does the private archive direction deserve a small validation trial despite the unresolved return habit?
2. Does an optional photo gallery earn its MVP cost compared with map-only entries?
3. Is manual GPX export acceptable in practice on your recording devices?

Architecture and implementation sequence are proposed in [PROJECT_PLAN.md](PROJECT_PLAN.md). These are recommendations awaiting review, not settled product decisions.
