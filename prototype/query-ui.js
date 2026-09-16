/* No simulated NL: examples are explicitly structured queries. */
const makeQuery=(all=[],entity='archive_entries')=>({version:1,entity,where:{all},order_by:[{field:'started_at',direction:'desc'}],limit:100});
const eq=(field,value)=>({field,op:'eq',value});
const EXAMPLES=[
 ['All entries',makeQuery()],
 ['2025 rally routes',makeQuery([eq('trip_id','touratech-rally-2025')])],
 ['TE 300 recordings',makeQuery([eq('bike_id','te-300')],'recordings')],
 ['Missing exact dates',makeQuery([{field:'started_at',op:'is_null'}])],
 ['890 · September · over 50 mi',makeQuery([eq('bike_id','ktm-890'),{field:'started_at',op:'gte',value:'2026-09-01T00:00:00-07:00'},{field:'started_at',op:'lt',value:'2026-10-01T00:00:00-07:00'},{field:'raw_distance_m',op:'gt',value:80467.2}],'recordings')],
 ['No matches: TE 300 over 50 mi',makeQuery([eq('bike_id','te-300'),{field:'raw_distance_m',op:'gt',value:80467.2}],'recordings')],
];
let executedQuery=null, requestGeneration=0, savedQueries=[];
function setEditor(query){$('structured-query').value=JSON.stringify(query,null,2);invalidateQuery();}
function invalidateQuery(){requestGeneration++;executedQuery=null;$('export-results').disabled=true;$('query-status').textContent='Query changed. Run to update results; the map still shows the previous result.';}
async function runQuery(){
 const generation=++requestGeneration;executedQuery=null;$('export-results').disabled=true;$('query-error').hidden=true;
 try{
  const query=JSON.parse($('structured-query').value);
  $('query-status').textContent='Running local database query…';
  const response=await fetch('/api/query',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(query)});
  const result=await response.json();if(generation!==requestGeneration)return;
  if(!response.ok)throw new Error(result.error||'Query failed.');
  executedQuery=result.query;applyResults(result);$('export-results').disabled=false;
  $('query-status').textContent=`${result.total_count} matches; ${result.records.length} shown. ${result.excluded_unknown_count} entries cannot be evaluated because query-relevant values are unknown.${result.truncated?' Results are limited; export includes shown entries only.':''} Interpretation: ${result.interpretation}`;
 }catch(error){if(generation!==requestGeneration)return;$('query-error').hidden=false;$('query-error').textContent=error.message;$('query-status').textContent='Query did not run. The map still shows the previous successful result.';}
}
function refreshSaved(){
 $('saved-queries').replaceChildren(new Option('Load a saved query',''));
 savedQueries.forEach((item,i)=>$('saved-queries').add(new Option(item.name,String(i))));
}
async function setupQueryUI(){
 for(const [label,query] of EXAMPLES){const button=document.createElement('button');button.textContent=label;button.onclick=()=>{setEditor(query);runQuery();};$('examples').append(button);}
 $('structured-query').oninput=invalidateQuery;$('run-query').onclick=runQuery;
 try{const stored=JSON.parse(localStorage.getItem('ride-archive-queries-v1')||'[]');if(Array.isArray(stored))savedQueries=stored.filter(x=>x&&typeof x.name==='string'&&x.query).slice(0,30);}catch{}
 refreshSaved();
 $('saved-queries').onchange=()=>{const item=savedQueries[Number($('saved-queries').value)];if($('saved-queries').value!==''&&item)setEditor(item.query);};
 $('save-query').onclick=()=>{
  if(!executedQuery){$('query-status').textContent='Run a valid query before saving it.';return;}
  const name=`Query ${savedQueries.length+1} · ${executedQuery.entity} · ${executedQuery.where.all.length} conditions`;
  try{const next=[...savedQueries,{name,query:executedQuery}].slice(-30);localStorage.setItem('ride-archive-queries-v1',JSON.stringify(next));savedQueries=next;refreshSaved();$('query-status').textContent='Saved in this browser. Result exports preserve the executed query; the archive bundle does not include browser-saved queries.';}catch{$('query-status').textContent='Browser storage unavailable. Export shown results to preserve the executed query.';}
 };
 $('export-results').onclick=async()=>{
  if(!executedQuery)return;
  const snapshot=executedQuery;
  try{const response=await fetch('/api/export-results',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(snapshot)});if(!response.ok)throw new Error('Export failed.');const url=URL.createObjectURL(await response.blob());const link=document.createElement('a');link.href=url;link.download='ride-archive-results.zip';link.click();setTimeout(()=>URL.revokeObjectURL(url),1000);}catch(error){$('query-error').hidden=false;$('query-error').textContent=error.message;}
 };
 setEditor(makeQuery());await runQuery();
}

init();
