/* Disposable read-only prototype. Data comes only from five supplied exports. */
const $ = id => document.getElementById(id);
let rides = [], filtered = [], selected = null, map = null, layers = null, mode = 'map';
const paths = new Map();
const dateFormat = new Intl.DateTimeFormat('en-US', {month:'long',day:'numeric',year:'numeric',timeZone:'UTC'});
const dateLabel = date => dateFormat.format(new Date(date+'T12:00:00Z'));
const duration = hours => { const minutes=Math.round(hours*60); return `${Math.floor(minutes/60)}h ${minutes%60}m`; };
const timeLabel = value => new Intl.DateTimeFormat('en-US',{hour:'numeric',minute:'2-digit',timeZone:'America/Los_Angeles'}).format(new Date(value));
const escapeHTML = value => String(value).replace(/[&<>"']/g, c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

const entryDate = r => r.date ? dateLabel(r.date) : `${r.reportedYear} rally · exact date unknown`;
const isGap = (a,b) => a[4] && b[4] && new Date(b[4])-new Date(a[4])>60000;

function card(ride) {
  const button=document.createElement('button');
  button.className='ride-card'; button.type='button'; button.style.setProperty('--ride-color',ride.color);
  button.setAttribute('aria-pressed',String(selected===ride.id));
  button.innerHTML=`<span class="ride-date">${entryDate(ride)}</span><h3>${escapeHTML(ride.title)}</h3><span class="bike-line"><i class="swatch"></i>${escapeHTML(ride.bike)}</span><span class="ride-stats"><span>≈ ${ride.miles} mi</span><span>${ride.hours===null?'Supplied route':duration(ride.hours)+' elapsed'}</span></span><span class="ride-style">${ride.style} ${ride.id==='hells'?'· PARTIAL DAY':''}</span>`;
  button.addEventListener('click',()=>selectRide(ride.id));
  return button;
}
function renderLists() {
  $('ride-list').replaceChildren(); $('timeline').replaceChildren();
  $('count').textContent=`${filtered.length} entries · ${filtered.filter(r=>r.kind==='recorded').length} recordings · ${filtered.filter(r=>r.kind==='route_reference').length} supplied routes`;
  if (!filtered.length) {
    for(const id of ['ride-list','timeline']) { const p=document.createElement('p'); p.className='empty';p.textContent='No matching entries. Edit the query or choose another example.';$(id).append(p); }
  }
  let month='';
  for(const ride of filtered) {
    $('ride-list').append(card(ride));
    const group=ride.date ? ride.date.slice(0,7) : 'undated';
    if (month!==group) { month=group;const h=document.createElement('h2');h.textContent=ride.date ? new Intl.DateTimeFormat('en-US',{month:'long',year:'numeric',timeZone:'UTC'}).format(new Date(ride.date+'T12:00:00Z')) : 'Exact dates unknown · 2025 rally';$('timeline').append(h); }
    $('timeline').append(card(ride));
  }
}
function drawMap() {
  if (!map) return;
  layers.clearLayers();paths.clear();
  for(const ride of filtered) {
    const group=L.featureGroup().addTo(layers); paths.set(ride.id,group);
    for(const segment of ride.segments) {
      let chunk=[segment[0]];
      const add=points=> {if(points.length<2)return; L.polyline(points.map(p=>[p[0],p[1]]),{color:ride.color,weight:selected===ride.id?5:3.5,opacity:selected&&selected!==ride.id?.45:.95}).addTo(group).bindTooltip(ride.title).on('click',()=>selectRide(ride.id));};
      for(let i=1;i<segment.length;i++) {
        if(isGap(segment[i-1],segment[i])) {
          add(chunk);L.polyline([[segment[i-1][0],segment[i-1][1]],[segment[i][0],segment[i][1]]],{color:ride.color,weight:3,dashArray:'5 6'}).addTo(group).bindTooltip('Sampling gap · path between observations unknown');chunk=[segment[i]];
        } else chunk.push(segment[i]);
      }
      add(chunk);
    }
    const all=ride.segments.flat();
    if(selected===ride.id) {
      for(const [point,label] of [[all[0],'Start'],[all.at(-1),'End']]) L.marker([point[0],point[1]],{icon:L.divIcon({className:'endpoint'+(label==='End'?' end':''),html:label[0],iconSize:[21,21],iconAnchor:[10,10]}),keyboard:true,title:label}).addTo(group).bindTooltip(label);
    } else {
      L.circleMarker([all[0][0],all[0][1]],{radius:7,color:'#fff',weight:2,fillColor:ride.color,fillOpacity:1}).addTo(group).bindTooltip(ride.title).on('click',()=>selectRide(ride.id));
    }
  }
}
function fitFiltered() { if(map&&filtered.length) map.fitBounds(layers.getBounds(),{padding:[45,45],maxZoom:13}); }
function applyResults(result) {
  rides=result.records.map(record=>({...record.display, queryRecord:record}));
  filtered=rides;selected=null;$('detail').hidden=true;
  renderLists();drawMap();fitFiltered();$('fit').disabled=!filtered.length;
}
function showMode(next) {
  mode=next;$('workspace').hidden=mode!=='map';$('timeline').hidden=mode!=='timeline';
  $('map-view').setAttribute('aria-pressed',String(mode==='map'));$('timeline-view').setAttribute('aria-pressed',String(mode==='timeline'));
  if(mode==='map'&&map){map.invalidateSize();if(selected)map.fitBounds(paths.get(selected).getBounds(),{padding:[35,35]});else fitFiltered();}
}
function selectRide(id) {
  selected=id;renderLists();drawMap();
  const ride=rides.find(r=>r.id===id);renderDetail(ride);
  if(map&&mode==='map')map.fitBounds(paths.get(id).getBounds(),{padding:[40,40],maxZoom:14});
}
function renderDetail(r) {
  $('detail').hidden=false;
  $('detail').innerHTML=`<div class="detail-top"><div><p class="eyebrow">${entryDate(r)} · ${r.style}</p><h2>${escapeHTML(r.title)}</h2><p>${escapeHTML(r.context)}<br>${escapeHTML(r.bike)} · ${r.start ? timeLabel(r.start)+'–'+timeLabel(r.end)+' PDT' : 'No recorded timestamps'}</p></div><button class="quiet" id="close-detail">Close details</button></div><div class="detail-stats"><div class="metric"><strong>≈ ${r.miles} mi</strong><span>${r.kind==='route_reference'?'Supplied route length':'Recorded distance'}</span></div><div class="metric"><strong>${r.hours===null?'Unknown':duration(r.hours)}</strong><span>Elapsed recording span</span></div><div class="metric"><strong>${r.elevation[0].toLocaleString()}–${r.elevation[1].toLocaleString()} ft</strong><span>Elevation range as exported</span></div></div><p class="detail-caption">ELEVATION PROFILE · AS EXPORTED</p><svg class="profile" role="img" aria-label="Exported elevation against track distance"></svg><p class="quality">${escapeHTML(r.note)}</p><p class="detail-caption">${r.kind==='route_reference'?'Rally-supplied route':r.recorder?'Recorded on '+escapeHTML(r.recorder):'Recorder not specified'} · Exported by ${escapeHTML(r.exporter)} · ${r.points.toLocaleString()} points<br>${r.kind==='route_reference'?'Length follows supplied geometry, not your observed riding path.':'Distance uses adjacent recorded points, including gap connectors.'} Bike, title and trip context are user-confirmed; elevation is uncorrected.</p>`;
  $('close-detail').onclick=()=>{selected=null;$('detail').hidden=true;renderLists();drawMap();fitFiltered();};
  if(r.queryRecord) {
    const evidence=document.createElement('details');const summary=document.createElement('summary');summary.textContent='Query evidence & provenance';
    const pre=document.createElement('pre');pre.className='evidence';pre.textContent=JSON.stringify({matched_fields:r.queryRecord.matched_fields,provenance:r.queryRecord.provenance},null,2);
    evidence.append(summary,pre);$('detail').append(evidence);
  }
  drawProfile(r);
}
function drawProfile(r) {
  const svg=$('detail').querySelector('svg');if(!svg)return;
  const w=Math.max(260,svg.getBoundingClientRect().width),h=150,left=53,right=15,top=10,bottom=32;
  svg.setAttribute('viewBox',`0 0 ${w} ${h}`);
  const maxDistance=r.segments.flat().at(-1)[3],lo=Math.floor(r.elevation[0]/1000)*1000,hi=Math.ceil(r.elevation[1]/1000)*1000;
  const x=d=>left+d/maxDistance*(w-left-right), y=e=>h-bottom-(e-lo)/(hi-lo)*(h-top-bottom);
  const lines=[];
  for(let i=0;i<3;i++){const v=lo+(hi-lo)*i/2;lines.push(`<line x1="${left}" y1="${y(v)}" x2="${w-right}" y2="${y(v)}" stroke="#e5e9df"/><text x="${left-7}" y="${y(v)+4}" text-anchor="end">${Math.round(v).toLocaleString()}</text>`);}
  for(let i=0;i<4;i++){const v=maxDistance*i/3;lines.push(`<text x="${x(v)}" y="${h-14}" text-anchor="${i===3?'end':i===0?'start':'middle'}">${v.toFixed(0)} mi</text>`);}
  for(const s of r.segments) {
    let d='';s.forEach((p,i)=>{d+=(i===0||isGap(s[i-1],p)?'M':'L')+x(p[3]).toFixed(1)+','+y(p[2]).toFixed(1);});
    lines.push(`<path d="${d}" stroke="${r.color}" stroke-width="1.6" fill="none"/>`);
  }
  svg.innerHTML=lines.join('');
}
async function init() {
  try {

    if(window.L) {
      map=L.map('map',{scrollWheelZoom:true}).setView([46.3,-118.8],6);layers=L.featureGroup().addTo(map);
      let failures=0;L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19,attribution:'&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'}).on('tileerror',()=>{if(++failures>=3)$('tile-error').hidden=false;}).addTo(map);
      L.control.scale({imperial:true,metric:false}).addTo(map);
    } else { $('error').hidden=false;$('error').textContent='The online map library did not load. Your chronological ride archive is still available.';showMode('timeline');$('map-view').disabled=true; }
    $('fit').onclick=fitFiltered;
    $('map-view').onclick=()=>showMode('map');$('timeline-view').onclick=()=>showMode('timeline');
    window.addEventListener('resize',()=>{if(selected)drawProfile(rides.find(r=>r.id===selected));});
    await setupQueryUI();
  } catch(error) {$('error').hidden=false;$('error').textContent=error.message+' Start the local server described in the README.';}
}
