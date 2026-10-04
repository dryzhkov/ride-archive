<script lang="ts">
  import { onMount } from 'svelte';
  import { api, allCatalog, allRecords, request, setToken, ApiError } from './api';
  import type { CatalogItem, Entry, Page, PathSummary } from './api';
  import SearchHome from './SearchHome.svelte';

  let entries = $state<Entry[]>([]), trips = $state<CatalogItem[]>([]), bikes = $state<CatalogItem[]>([]);
  let title = $state(''), tripId = $state(''), bikeId = $state('');
  let tripName = $state(''), bikeName = $state(''), error = $state(''), notice = $state('');
  let busy = $state(false), needsToken = $state(false), tokenInput = $state('');
  let offset = $state(0), nextOffset = $state<number | null>(null);
  let editing = $state<Entry | null>(null), editTitle = $state(''), editTrip = $state(''), editBike = $state('');
  let selectedFile = $state<File | null>(null);
  let addingGPXEntryID = $state('');
  let additionalFiles = $state<Record<string, File | null>>({});
  let linkedPaths = $state<Record<string, PathSummary[]>>({});
  const isAddTrip = window.location.pathname === '/add-trip';
  async function detachPath(entry: Entry, path: PathSummary) {
    await api(`/entries/${entry.id}/paths/${path.id}?revision=${entry.revision}`, 'DELETE');
    await loadEntries(); notice = 'Track unlinked. The original file is still preserved.';
  }


  async function act(work: () => Promise<void>) {
    busy = true; error = ''; notice = '';
    try { await work(); }
    catch (e) {
      error = e instanceof Error ? e.message : 'Something went wrong.';
      if (e instanceof ApiError && e.status === 401) needsToken = true;
    } finally { busy = false; }
  }
  async function loadEntries(at = offset) {
    const result = await api<Page<Entry>>(`/entries?limit=20&offset=${at}`);
    const links = await Promise.all(result.items.map(async entry => {
      const paths = await api<{items: PathSummary[]}>(`/entries/${entry.id}/paths`);
      return [entry.id, paths.items] as const;
    }));
    linkedPaths = Object.fromEntries(links);
    entries = result.items; offset = at; nextOffset = result.next_offset;
  }
  async function refresh() {
    const catalog = await Promise.all([allCatalog('trips'), allCatalog('bikes')]);
    [trips, bikes] = catalog;
    await loadEntries();
    needsToken = false;
  }
  function nameOf(items: CatalogItem[], id: string | null, fallback: string) {
    return items.find(item => item.id === id)?.name ?? fallback;
  }
  async function createEntry() {
    if (!selectedFile) throw new Error('Choose a GPX file for this entry.');
    if (selectedFile.size > 10 * 1024 * 1024) throw new Error('Choose a file no larger than 10 MiB.');
    const body = new FormData(); body.set('title', title); body.set('trip_id', tripId); body.set('bike_id', bikeId); body.set('file', selectedFile);
    await request('/entries', { method: 'POST', body });
    title = ''; tripId = ''; bikeId = ''; selectedFile = null;
    const input = document.getElementById('entry-gpx') as HTMLInputElement | null; if (input) input.value = '';
    await loadEntries(); notice = 'Ride and GPX saved together.';
  }
  async function addGPX(entry: Entry) {
    const file = additionalFiles[entry.id]; if (!file) return;
    if (file.size > 10 * 1024 * 1024) throw new Error('Choose a file no larger than 10 MiB.');
    const body = new FormData(); body.set('title', ''); body.set('revision', String(entry.revision)); body.set('file', file);
    await request(`/entries/${entry.id}/sources`, { method: 'POST', body });
    additionalFiles[entry.id] = null; addingGPXEntryID = ''; await loadEntries(); notice = `GPX added to “${entry.title}”.`;
  }
  async function createCatalog(kind: 'trips' | 'bikes') {
    const item = await api<CatalogItem>(`/${kind}`, 'POST', { name: kind === 'trips' ? tripName : bikeName });
    if (kind === 'trips') { trips = [...trips, item]; tripName = ''; tripId = item.id; }
    else { bikes = [...bikes, item]; bikeName = ''; bikeId = item.id; }
  }
  function startEdit(entry: Entry) {
    editing = entry; editTitle = entry.title; editTrip = entry.trip_id ?? ''; editBike = entry.bike_id ?? '';
  }
  async function saveEdit() {
    if (!editing) return;
    await api(`/entries/${editing.id}`, 'PATCH', {
      revision: editing.revision, title: editTitle, trip_id: editTrip || null, bike_id: editBike || null,
    });
    editing = null; await loadEntries(); notice = 'Changes saved.';
  }
  onMount(() => { void act(refresh); });
</script>

<svelte:head><meta name="description" content="Your rides, sources, and trips in one personal archive." /></svelte:head>

<header><a class="brand" href="/">◈ <span>Ride Archive</span></a><nav class="top-nav"><a href="/" aria-current={!isAddTrip?'page':undefined}>Search</a><a href="/add-trip" aria-current={isAddTrip?'page':undefined}>Add a ride</a></nav><span class="edition">Personal archive · Foundation</span></header>
<main>
  {#if isAddTrip}
  <div class="intro"><p class="eyebrow">YOUR RIDES, TOGETHER</p><h1>Add a ride.</h1><p>Save a GPX recording with its ride details.</p></div>
  {#if error}<div class="message error" role="alert">{error} <button class="quiet" onclick={() => act(refresh)} disabled={busy}>Reload</button></div>{/if}
  {#if notice}<div class="message" role="status">{notice}</div>{/if}
  {#if needsToken}
    <section class="panel"><h2>Open your archive</h2><p>This archive requires your personal API token.</p>
      <form onsubmit={(event) => { event.preventDefault(); setToken(tokenInput); tokenInput = ''; void act(refresh); }}>
        <label>API token<input type="password" bind:value={tokenInput} required autocomplete="off" /></label>
        <button disabled={busy}>Unlock</button>
      </form>
    </section>
  {:else}
    <div class="layout">
      <div class="primary">
        <section class="panel">
          <div class="section-heading"><h2>Add a ride</h2><button class="quiet" onclick={() => act(refresh)} disabled={busy}>Refresh</button></div>
          <form class="entry-form" onsubmit={(event) => { event.preventDefault(); void act(createEntry); }}>
            <label class="gpx-primary">GPX recording <span class="muted">Your original file is kept as uploaded · up to 10 MiB</span><input id="entry-gpx" type="file" accept=".gpx,application/gpx+xml" required onchange={(event) => selectedFile = event.currentTarget.files?.[0] ?? null} /></label>
            <label>Ride name<input bind:value={title} placeholder="e.g. Hells Canyon, day one" required maxlength="200" /></label>
            <div class="pair">
              <label>Trip<select bind:value={tripId}><option value="">No trip</option>{#each trips as trip}<option value={trip.id}>{trip.name}</option>{/each}</select></label>
              <label>Bike<select bind:value={bikeId}><option value="">Unknown bike</option>{#each bikes as bike}<option value={bike.id}>{bike.name}</option>{/each}</select></label>
            </div>
            <div class="form-footer"><small>One ride, with its recording and original file.</small><button disabled={busy || !selectedFile}>Add ride</button></div>
          </form>
          <div class="section-heading archive-heading"><h2>Your rides</h2><span class="muted">{entries.length} shown</span></div>
          {#if entries.length === 0}<div class="empty"><span>↗</span><h3>Your archive starts here.</h3><p>Add a ride with its GPX above. Trips and bikes are optional.</p></div>{/if}
          <div class="entry-list">
            {#each entries as entry (entry.id)}
              <article>
                {#if editing?.id === entry.id}
                  <form onsubmit={(event) => { event.preventDefault(); void act(saveEdit); }}>
                    <label>Title<input bind:value={editTitle} required maxlength="200" /></label>
                    <div class="pair"><label>Trip<select bind:value={editTrip}><option value="">No trip</option>{#each trips as t}<option value={t.id}>{t.name}</option>{/each}</select></label>
                    <label>Bike<select bind:value={editBike}><option value="">Unknown bike</option>{#each bikes as b}<option value={b.id}>{b.name}</option>{/each}</select></label></div>
                    <div class="actions"><button type="button" class="quiet" onclick={() => editing = null}>Cancel</button><button disabled={busy}>Save changes</button></div>
                  </form>
                {:else}
                  <div class="entry-details"><h3>{entry.title}</h3><p>{nameOf(trips, entry.trip_id, 'No trip')} <span>·</span> {nameOf(bikes, entry.bike_id, 'Unknown bike')}</p>
                    {#if linkedPaths[entry.id]?.length}
                      <span class="badge">{linkedPaths[entry.id].length} GPX track(s) linked</span>
                      <ul class="linked-paths">{#each linkedPaths[entry.id] as path}<li><div><strong>{path.name || path.source_locator}</strong><small>{path.original_filename} · {path.point_count.toLocaleString()} points</small></div><button class="quiet" disabled={busy} onclick={() => act(() => detachPath(entry, path))}>Unlink</button></li>{/each}</ul>
                    {:else}<span class="badge">No GPX attached</span>{/if}
                  </div>
                  <div class="entry-actions"><button class="quiet" onclick={() => startEdit(entry)} disabled={busy}>Edit</button><button class="secondary" onclick={() => {addingGPXEntryID = addingGPXEntryID === entry.id ? '' : entry.id;}} disabled={busy}>Add GPX</button></div>
                  {#if addingGPXEntryID === entry.id}<form class="add-gpx" onsubmit={(event) => { event.preventDefault(); void act(() => addGPX(entry)); }}><label>Add another recording<input type="file" accept=".gpx,application/gpx+xml" required onchange={(event) => additionalFiles[entry.id] = event.currentTarget.files?.[0] ?? null} /></label><div class="actions"><button type="button" class="quiet" onclick={() => addingGPXEntryID = ''}>Cancel</button><button disabled={busy || !additionalFiles[entry.id]}>Add GPX</button></div></form>{/if}
                {/if}
              </article>
            {/each}
          </div>
          <nav aria-label="Entry pages"><button class="quiet" disabled={busy || offset === 0} onclick={() => act(() => loadEntries(Math.max(0, offset - 20)))}>Previous</button><span>Page {Math.floor(offset / 20) + 1}</span><button class="quiet" disabled={busy || nextOffset === null} onclick={() => act(() => loadEntries(nextOffset!))}>Next</button></nav>
        </section>
      </div>
      <aside>
        <section class="panel"><p class="eyebrow">ORGANIZE</p><h2>Trips</h2><p class="muted">A weekend, a rally, or a longer journey.</p>
          <div class="tags">{#each trips as t}<span>{t.name}</span>{/each}</div>
          <form onsubmit={(event) => { event.preventDefault(); void act(() => createCatalog('trips')); }}><label>Trip name<input bind:value={tripName} placeholder="e.g. Touratech Rally 2025" required maxlength="200" /></label><button class="secondary" disabled={busy}>Add trip</button></form>
        </section>
        <section class="panel"><h2>Bikes</h2><div class="tags">{#each bikes as b}<span>{b.name}</span>{/each}</div>
          <form onsubmit={(event) => { event.preventDefault(); void act(() => createCatalog('bikes')); }}><label>Bike name<input bind:value={bikeName} placeholder="e.g. KTM 890 Adventure R" required maxlength="200" /></label><button class="secondary" disabled={busy}>Add bike</button></form>
        </section>
        <p class="aside-note">Dates, distances, and recorded paths will come from evidence. Missing information stays unknown.</p>
      </aside>
    </div>
  {/if}
  {:else if needsToken}
    <section class="panel token-panel"><h2>Open your archive</h2><p>This archive requires your personal API token.</p><form onsubmit={(event) => { event.preventDefault(); setToken(tokenInput); tokenInput = ''; void act(refresh); }}><label>API token<input type="password" bind:value={tokenInput} required autocomplete="off" /></label><button disabled={busy}>Unlock</button></form></section>
  {:else}
    <SearchHome />
  {/if}
</main>
<footer>Ride Archive <span>Record wherever you want. Keep it yours.</span></footer>
