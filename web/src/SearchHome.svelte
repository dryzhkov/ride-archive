<script lang="ts">
  import { onMount } from 'svelte';
  import { allCatalog, allRecords, api } from './api';
  import type { CatalogItem, Entry } from './api';
  import type { PathGeometry } from './api';

  type Result = Entry & { trip_name: string; bike_name: string; paths: PathGeometry[]; distance_m: number | null };
  type Tile = { key: string; x: number; y: number; size: number; url: string };
  type QueryClause = {field:string;op:string;value?:string|string[]};
  type QueryPreview = {version:1;entity:'archive_entries';mode:'browse'|'text'|'structured';where:{all:QueryClause[]};order_by:{field:string;direction:'asc';nulls:'last'}[];limit:null};
  let entries = $state<Result[]>([]), trips = $state<CatalogItem[]>([]), bikes = $state<CatalogItem[]>([]);
  let loading = $state(true), error = $state(''), locationPending = $state(true);
  let userLocation = $state<{lat:number;lon:number}|null>(null);
  let mode = $state<'text'|'structured'>('text'), view = $state<'map'|'list'>('map');
  let textDraft = $state(''), textQuery = $state('');
  let tripFilter = $state(''), bikeFilter = $state('');
  let appliedTripFilter = $state(''), appliedBikeFilter = $state('');
  let activeQuery = $state<'nearby'|'text'|'structured'>('nearby');
  let executedQuery = $state<QueryPreview>({version:1,entity:'archive_entries',mode:'browse',where:{all:[]},order_by:[{field:'nearest_linked_geometry',direction:'asc',nulls:'last'}],limit:null});
  let center = $state({lat:37.8,lon:-121.9}), zoom = $state(7);
  let mapSize = $state({width:900,height:500});
  let drag = $state<{x:number;y:number;worldX:number;worldY:number;pointerId:number;target:{entry:Result;path:PathGeometry}|null}|null>(null);
  let dragMoved = $state(false);
  let selectedPath = $state<{entry:Result;path:PathGeometry}|null>(null);
  let zoomTarget = 7, zoomFrame = 0;
  const tileTemplate = import.meta.env.VITE_MAP_TILE_URL || 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
  let hasMapCenter = $derived(Boolean(userLocation)||entries.some(e=>e.paths.some(p=>p.point_count>0)));

  const nearMiles = (p: PathGeometry, lat:number, lon:number) => {
    let best = Number.POSITIVE_INFINITY;
    for (const segment of p.segments) for (const pt of segment) {
      const r=Math.PI/180, dLat=(pt.latitude-lat)*r, dLon=(pt.longitude-lon)*r;
      const a=Math.sin(dLat/2)**2+Math.cos(lat*r)*Math.cos(pt.latitude*r)*Math.sin(dLon/2)**2;
      best=Math.min(best,6371000*2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a)));
    }
    return Number.isFinite(best)?best:null;
  };
  const distanceOf = (paths:PathGeometry[]) => {
    if (!userLocation) return null;
    const values=paths.map(p=>nearMiles(p,userLocation!.lat,userLocation!.lon)).filter((v):v is number=>v!==null);
    return values.length?Math.min(...values):null;
  };
  async function load() {
    loading=true; error='';
    try {
      const [tripList,bikeList,all] = await Promise.all([allCatalog('trips'),allCatalog('bikes'),allRecords<Entry>('entries')]);
      trips=tripList; bikes=bikeList;
      entries=await Promise.all(all.map(async e=>{
        const data=await api<{items:PathGeometry[]}>(`/entries/${e.id}/map`);
        return {...e,trip_name:tripList.find(x=>x.id===e.trip_id)?.name??'No trip',bike_name:bikeList.find(x=>x.id===e.bike_id)?.name??'Unknown bike',paths:data.items,distance_m:distanceOf(data.items)};
      }));
    } catch(e) { error=e instanceof Error?e.message:'Could not load archive results.'; }
    finally { loading=false; if(!locationPending&&!userLocation)fitArchive(); }
  }
  function requestLocation() {
    if (!navigator.geolocation) { locationPending=false; return; }
    navigator.geolocation.getCurrentPosition(position=>{
      userLocation={lat:position.coords.latitude,lon:position.coords.longitude};
      center=userLocation;
      entries=entries.map(e=>({...e,distance_m:distanceOf(e.paths)}));
      locationPending=false;
    },()=>{locationPending=false;fitArchive();}, {enableHighAccuracy:false,timeout:7000,maximumAge:300000});
  }
  onMount(()=>{
    const map=document.querySelector('.map');
    const observer=map?new ResizeObserver(([entry])=>{mapSize={width:entry.contentRect.width,height:entry.contentRect.height};}):null;
    if(map&&observer)observer.observe(map);
    window.addEventListener('pointermove',pointerMove);
    window.addEventListener('pointerup',pointerUp);
    window.addEventListener('pointercancel',pointerUp);
    void load(); requestLocation();
    return ()=>{observer?.disconnect();window.removeEventListener('pointermove',pointerMove);window.removeEventListener('pointerup',pointerUp);window.removeEventListener('pointercancel',pointerUp);};
  });

  let results = $derived.by(()=>{
    let list=entries;
    if(activeQuery==='text' && textQuery.trim()) {
      const terms=textQuery.toLocaleLowerCase().split(/\s+/).filter(Boolean);
      list=list.filter(e=>terms.every(term=>[e.title,e.trip_name,e.bike_name,...e.paths.map(p=>p.name),...e.paths.map(p=>p.original_filename)].some(v=>v.toLocaleLowerCase().includes(term))));
    }
    if(activeQuery==='structured') {
      if(appliedTripFilter) list=list.filter(e=>e.trip_id===appliedTripFilter);
      if(appliedBikeFilter) list=list.filter(e=>e.bike_id===appliedBikeFilter);
    }
    return [...list].sort((a,b)=> (a.distance_m??Infinity)-(b.distance_m??Infinity));
  });

  const world = (lat:number,lon:number,z:number) => {
    const scale=256*2**z, safeLat=Math.max(-85.0511,Math.min(85.0511,lat)), sin=Math.sin(safeLat*Math.PI/180);
    return {x:(lon+180)/360*scale,y:(0.5-Math.log((1+sin)/(1-sin))/(4*Math.PI))*scale};
  };
  let tiles = $derived.by(()=>{
    if(locationPending||!hasMapCenter) return {items:[] as Tile[],cx:mapSize.width/2,cy:mapSize.height/2,left:0,top:0};
    const baseZoom=Math.floor(zoom),size=256*2**(zoom-baseZoom),c=world(center.lat,center.lon,zoom), width=mapSize.width,height=mapSize.height, left=c.x-width/2,top=c.y-height/2;
    const out:Tile[]=[];
    for(let ty=Math.floor(top/size);ty<=Math.floor((top+height)/size);ty++) for(let tx=Math.floor(left/size);tx<=Math.floor((left+width)/size);tx++) {
      const n=2**baseZoom, x=(tx%n+n)%n; if(ty<0||ty>=n) continue;
      out.push({key:`${baseZoom}/${x}/${ty}`,x:tx*size-left,y:ty*size-top,size,url:tileTemplate.replace('{z}',String(baseZoom)).replace('{x}',String(x)).replace('{y}',String(ty))});
    }
    return {items:out,cx:c.x-left,cy:c.y-top,left,top};
  });
  function project(lat:number,lon:number){const p=world(lat,lon,zoom);return {x:p.x-tiles.left,y:p.y-tiles.top};}
  function lineFor(segment:PathGeometry['segments'][number]) {const stride=Math.max(1,Math.ceil(segment.length/450));const sampled=segment.filter((_,i)=>i%stride===0||i===segment.length-1);return sampled.map((p,i)=>{const xy=project(p.latitude,p.longitude);return `${i?'L':'M'}${xy.x.toFixed(1)},${xy.y.toFixed(1)}`;}).join(' ');}
  function focusPath(path:PathGeometry){
    const points=path.segments.flat();if(!points.length)return;
    let minX=1,maxX=0,minY=1,maxY=0;
    for(const point of points){
      const projected=world(point.latitude,point.longitude,0);
      const x=projected.x/256,y=projected.y/256;
      minX=Math.min(minX,x);maxX=Math.max(maxX,x);minY=Math.min(minY,y);maxY=Math.max(maxY,y);
    }
    const spanX=Math.max(maxX-minX,1e-8),spanY=Math.max(maxY-minY,1e-8);
    const availableWidth=Math.max(120,mapSize.width-140),availableHeight=Math.max(120,mapSize.height-120);
    const fitZoom=Math.min(Math.log2(availableWidth/(256*spanX)),Math.log2(availableHeight/(256*spanY)));
    const x=(minX+maxX)/2,y=(minY+maxY)/2,n=Math.PI-2*Math.PI*y;
    center={lon:x*360-180,lat:180/Math.PI*Math.atan(Math.sinh(n))};
    setZoom(Math.max(2,Math.min(15.75,fitZoom)));
  }
  function setZoom(target:number){
    zoomTarget=Math.max(2,Math.min(16,target));
    if(zoomFrame)cancelAnimationFrame(zoomFrame);
    const animate=()=>{
      const difference=zoomTarget-zoom;
      if(Math.abs(difference)<0.002){zoom=zoomTarget;zoomFrame=0;return;}
      zoom+=difference*0.18;
      zoomFrame=requestAnimationFrame(animate);
    };
    zoomFrame=requestAnimationFrame(animate);
  }
  function fitArchive(){
    const pts=entries.flatMap(e=>e.paths.flatMap(p=>p.segments.flat()));if(!pts.length)return;
    let minLat=90,maxLat=-90,minLon=180,maxLon=-180;
    for(const p of pts){minLat=Math.min(minLat,p.latitude);maxLat=Math.max(maxLat,p.latitude);minLon=Math.min(minLon,p.longitude);maxLon=Math.max(maxLon,p.longitude);}
    center={lat:(minLat+maxLat)/2,lon:(minLon+maxLon)/2};setZoom(9);
  }
  function fitResults(){
    const points=results.flatMap(e=>e.paths.flatMap(p=>p.segments.flat()));
    if(!points.length){if(userLocation)center=userLocation;return;}
    let minLat=90,maxLat=-90,minLon=180,maxLon=-180;
    for(const p of points){minLat=Math.min(minLat,p.latitude);maxLat=Math.max(maxLat,p.latitude);minLon=Math.min(minLon,p.longitude);maxLon=Math.max(maxLon,p.longitude);}
    center={lat:(minLat+maxLat)/2,lon:(minLon+maxLon)/2};
    setZoom(points.length>500?8:9);
  }
  function makeQuery(mode:'browse'|'text'|'structured', text='', trip='', bike=''):QueryPreview {
    const all:QueryClause[]=[];
    if(mode==='text'&&text.trim())all.push({field:'searchable_names',op:'contains_all_terms',value:text.toLocaleLowerCase().split(/\s+/).filter(Boolean)});
    if(mode==='structured'&&trip)all.push({field:'trip_id',op:'eq',value:trip});
    if(mode==='structured'&&bike)all.push({field:'bike_id',op:'eq',value:bike});
    return {version:1,entity:'archive_entries',mode,where:{all},order_by:[{field:'nearest_linked_geometry',direction:'asc',nulls:'last'}],limit:null};
  }
  function runText(){textQuery=textDraft.trim();activeQuery=textQuery?'text':'nearby';executedQuery=makeQuery(textQuery?'text':'browse',textQuery);selectedPath=null;fitResults();}
  function runStructured(){appliedTripFilter=tripFilter;appliedBikeFilter=bikeFilter;activeQuery='structured';executedQuery=makeQuery('structured','',tripFilter,bikeFilter);selectedPath=null;fitResults();}
  function resetNearby(){textDraft='';textQuery='';tripFilter='';bikeFilter='';appliedTripFilter='';appliedBikeFilter='';activeQuery='nearby';executedQuery=makeQuery('browse');selectedPath=null;if(userLocation)center=userLocation;}
  function pointerDown(event:PointerEvent){
    if(event.button!==0||(event.target as HTMLElement).closest('.map-zoom,.map-attribution,.map-popup'))return;
    const pos=world(center.lat,center.lon,zoom),pathID=(event.target as Element).closest('[data-path-id]')?.getAttribute('data-path-id')??'';
    const entry=results.find(item=>item.paths.some(path=>path.id===pathID)),path=entry?.paths.find(item=>item.id===pathID);
    drag={x:event.clientX,y:event.clientY,worldX:pos.x,worldY:pos.y,pointerId:event.pointerId,target:entry&&path?{entry,path}:null};dragMoved=false;
    event.preventDefault();
  }
  function pointerMove(event:PointerEvent){
    if(!drag||event.pointerId!==drag.pointerId)return;
    const dx=event.clientX-drag.x,dy=event.clientY-drag.y;
    if(Math.hypot(dx,dy)>4)dragMoved=true;
    if(!dragMoved)return;
    const scale=256*2**zoom,x=drag.worldX-dx,y=drag.worldY-dy,n=Math.PI-2*Math.PI*y/scale;
    center={lon:x/scale*360-180,lat:180/Math.PI*Math.atan(Math.sinh(n))};selectedPath=null;
  }
  function pointerUp(event:PointerEvent){if(!drag||event.pointerId!==drag.pointerId)return;if(!dragMoved&&drag.target){selectedPath=drag.target;focusPath(drag.target.path);}drag=null;}
  function onWheel(event:WheelEvent){
    event.preventDefault();
    const delta=event.deltaMode===WheelEvent.DOM_DELTA_LINE?event.deltaY*16:event.deltaMode===WheelEvent.DOM_DELTA_PAGE?event.deltaY*mapSize.height:event.deltaY;
    setZoom(zoomTarget-delta*0.0012);
  }
  function selectRoute(entry:Result,path:PathGeometry,event:Event){event.stopPropagation();if(dragMoved)return;selectedPath={entry,path};focusPath(path);}
  function selectResult(entry:Result){const path=entry.paths[0];if(path){selectedPath={entry,path};focusPath(path);}view='map';}
</script>

<div class="intro"><p class="eyebrow">YOUR RIDES, TOGETHER</p><h1>Find a ride.</h1><p>Search your archive or explore what’s nearby.</p></div>
{#if error}<div class="message error" role="alert">{error}<button class="quiet" onclick={() => load()}>Retry</button></div>{/if}
<section class="panel search-panel" aria-label="Search rides">
  <div class="mode-tabs" role="tablist" aria-label="Search mode"><button class:chosen={mode==='text'} role="tab" aria-selected={mode==='text'} onclick={()=>mode='text'}>Text search</button><button class:chosen={mode==='structured'} role="tab" aria-selected={mode==='structured'} onclick={()=>mode='structured'}>Build a query</button></div>
  {#if mode==='text'}
    <form class="text-search" onsubmit={(event)=>{event.preventDefault();runText();}}><label class="sr-only" for="ride-search">Search rides, trips, bikes or GPX names</label><input id="ride-search" bind:value={textDraft} placeholder="Search rides, trips, bikes…" /><button>Search</button></form>
  {:else}
    <form class="query-builder" onsubmit={(event)=>{event.preventDefault();runStructured();}}>
      <label>Trip<select bind:value={tripFilter}><option value="">Any trip</option>{#each trips as trip}<option value={trip.id}>{trip.name}</option>{/each}</select></label>
      <label>Bike<select bind:value={bikeFilter}><option value="">Any bike</option>{#each bikes as bike}<option value={bike.id}>{bike.name}</option>{/each}</select></label>
      <button>Run query</button>
    </form>
  {/if}
  <p class="search-hint">{#if activeQuery==='nearby'}{locationPending?'Finding your location…':userLocation?'All rides are shown, closest first. The map starts at your location; drag or zoom to explore. Ride Archive does not receive your coordinates; OpenStreetMap receives requests for the map area you view.':'All rides are shown; location is unavailable, so distance sorting is omitted.'}{:else if activeQuery==='text'}Matches ride, trip, bike and GPX names.{:else}Trip and bike filters are combined with AND. All matching rides are included, nearest geometry first when location is available.{/if}</p>
  <details class="query-inspector"><summary>Query currently applied</summary><p>{#if executedQuery.mode==='text'}Every search term must appear in at least one ride, trip, bike, or GPX name.{:else if executedQuery.mode==='structured'}Only the selected trip and bike filters apply; all matching results are included.{:else}All archive entries are included, with nearest linked geometry first when location is available.{/if}</p><pre>{JSON.stringify(executedQuery,null,2)}</pre></details>
</section>
<section class="results panel">
  <div class="results-heading"><div><h2>Results</h2><p>{loading?'Loading archive…':`${results.length} ${results.length===1?'ride':'rides'}`}</p></div><div class="results-controls"><button class:chosen={view==='map'} class="view-button" onclick={()=>view='map'} aria-pressed={view==='map'}>Map</button><button class:chosen={view==='list'} class="view-button" onclick={()=>view='list'} aria-pressed={view==='list'}>List</button><button class="quiet" onclick={resetNearby}>Reset</button></div></div>
  {#if view==='map'}
    <div class="map" class:dragging={drag!==null} role="application" aria-label="Map showing matching ride tracks. Drag to pan and use the plus and minus buttons or scroll to zoom." onpointerdown={pointerDown} onwheel={onWheel}>
      <div class="map-tiles">{#each tiles.items as tile(tile.key)}<img src={tile.url} alt="" style={`left:${tile.x}px;top:${tile.y}px;width:${tile.size}px;height:${tile.size}px`} loading="lazy" />{/each}</div>
      <svg class="map-lines" viewBox={`0 0 ${mapSize.width} ${mapSize.height}`} preserveAspectRatio="none" role="group" aria-label="Selectable GPX routes">
        {#each results as entry,ei (entry.id)}
          {#each entry.paths as path (path.id)}
            {#each path.segments as segment,si}
              <path data-path-id={path.id} d={lineFor(segment)} class="route-hit" role="button" tabindex="0" aria-label={`${entry.title}, ${path.name||path.original_filename}`} onclick={(event)=>selectRoute(entry,path,event)} onkeydown={(event)=>{if(event.key==='Enter'||event.key===' '){event.preventDefault();selectRoute(entry,path,event);}}} />
              <path d={lineFor(segment)} class="route-line" class:route-alt={ei%2===1} class:route-selected={selectedPath?.path.id===path.id} aria-hidden="true" />
            {/each}
          {/each}
        {/each}
        {#if userLocation}<circle class="you-dot" cx={tiles.cx} cy={tiles.cy} r="6" />{/if}
      </svg>
      <div class="map-zoom"><button aria-label="Zoom in" onclick={()=>setZoom(zoomTarget+0.25)}>+</button><button aria-label="Zoom out" onclick={()=>setZoom(zoomTarget-0.25)}>−</button></div>
      <small class="map-attribution">© <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">OpenStreetMap contributors</a></small>
      {#if selectedPath}<div class="map-popup"><button class="popup-close" aria-label="Close route details" onclick={()=>selectedPath=null}>×</button><strong>{selectedPath.entry.title}</strong><span>{selectedPath.path.name||selectedPath.path.original_filename}</span><small>{selectedPath.entry.trip_name} · {selectedPath.entry.bike_name} · {selectedPath.path.point_count.toLocaleString()} points</small></div>{/if}
      {#if !hasMapCenter}<div class="map-empty">Map a ride by adding a GPX recording or allowing location access.</div>{:else if !results.some(e=>e.paths.length)}<div class="map-empty">No recorded geometry in these results.</div>{/if}
    </div>
  {:else}
    <div class="result-list">
      {#each results as entry (entry.id)}
        <article class="result-card"><div><h3>{entry.title}</h3><p>{entry.trip_name} <span>·</span> {entry.bike_name}</p><small>{entry.paths.length?`${entry.paths.length} GPX path${entry.paths.length===1?'':'s'}`:'No mapped GPX'}{entry.distance_m!==null?` · ${(entry.distance_m/1609.344).toFixed(1)} mi from you`:''}</small></div><button class="quiet" onclick={()=>selectResult(entry)}>Show on map</button></article>
      {:else}<div class="empty"><h3>No rides match this search.</h3><p>Try a wider distance or clear one of the filters.</p></div>{/each}
    </div>
  {/if}
</section>
