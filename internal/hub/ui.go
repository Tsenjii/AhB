package hub

import "net/http"

const dashboardHTML = `<!doctype html>
<html lang="zh-TW">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<meta name="color-scheme" content="dark">
<title>AhB</title>
<style>
:root{
 color-scheme:dark;
 font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
 --bg:#0d0f12;--panel:#13161a;--panel2:#171b20;--line:#292e35;--line2:#22272d;
 --text:#eef1f4;--muted:#929aa5;--subtle:#6f7883;--accent:#d7e2f0;
 --ok:#68c58b;--warn:#d6aa5b;--bad:#df777f;--off:#77808b
}
*{box-sizing:border-box}
html{background:var(--bg)}
body{margin:0;min-height:100vh;background:var(--bg);color:var(--text)}
button,input{font:inherit}
main{max-width:1120px;margin:0 auto;padding:22px 18px 44px}
header{display:flex;align-items:center;justify-content:space-between;gap:18px;padding:4px 0 18px;border-bottom:1px solid var(--line)}
.brand{display:flex;align-items:center;gap:11px;min-width:0}
.mark{width:32px;height:32px;border:1px solid #3b424b;border-radius:8px;display:grid;place-items:center;font-size:13px;font-weight:800;letter-spacing:-.03em;background:#181c21}
h1{font-size:18px;line-height:1.1;margin:0;font-weight:720;letter-spacing:-.02em}
.kicker{font-size:11px;color:var(--muted);margin-top:3px}
button,.btn{appearance:none;border:1px solid #353b44;background:#181c21;color:var(--text);border-radius:8px;padding:8px 11px;font-size:12px;text-decoration:none;cursor:pointer;transition:background .12s,border-color .12s}
button:hover,.btn:hover{background:#20252b;border-color:#48515c}
.btn.primary{background:#e7ebef;color:#111418;border-color:#e7ebef;font-weight:650}
.btn.primary:hover{background:#fff;border-color:#fff}
.btn.disabled{opacity:.38;pointer-events:none}
.overview{display:grid;grid-template-columns:minmax(0,1.4fr) auto;gap:14px;align-items:center;padding:16px 0 17px;border-bottom:1px solid var(--line)}
.overview-title{font-size:13px;font-weight:650}
.desc,.muted{color:var(--muted)}
.desc{font-size:11px;line-height:1.5;margin-top:4px}
.base-row{display:flex;gap:8px;align-items:center;margin-top:10px;max-width:680px}
.base{flex:1;min-width:0;background:#111419;border:1px solid var(--line);border-radius:7px;padding:9px 11px;font:12px ui-monospace,SFMono-Regular,Menlo,monospace;overflow:auto;white-space:nowrap;color:#d8dde3}
.summary{display:grid;grid-template-columns:repeat(2,minmax(82px,1fr));border:1px solid var(--line);border-radius:9px;overflow:hidden}
.summary-item{padding:10px 12px;background:var(--panel)}
.summary-item+.summary-item{border-left:1px solid var(--line)}
.summary-item b{display:block;font-size:15px}
.summary-item span{display:block;color:var(--subtle);font-size:10px;margin-top:2px}
.notice{display:none;margin-top:12px;border:1px solid var(--line);border-left-width:3px;padding:9px 11px;border-radius:7px;font-size:11px;white-space:pre-wrap;background:var(--panel)}
#error{border-left-color:var(--bad);color:#f2b1b6}
#warnings{border-left-color:var(--warn);color:#e4c78e}
section{margin-top:22px}
.section-head{display:flex;align-items:end;justify-content:space-between;gap:14px;margin-bottom:9px}
.section-title{display:flex;align-items:baseline;gap:8px}
h2{font-size:13px;margin:0;font-weight:680}
.count{font-size:10px;color:var(--subtle)}
.providers{border:1px solid var(--line);border-radius:10px;overflow:hidden;background:var(--panel)}
.card{padding:14px 15px;border-bottom:1px solid var(--line2)}
.card:last-child{border-bottom:0}
.card-top{display:grid;grid-template-columns:minmax(150px,1.25fr) minmax(180px,1fr) auto;gap:16px;align-items:center}
.provider-main{min-width:0}
.provider-name{font-weight:680;font-size:13px;display:flex;align-items:center;gap:8px}
.state-dot{width:7px;height:7px;border-radius:50%;background:var(--off);box-shadow:0 0 0 2px rgba(119,128,139,.12)}
.state-dot.HEALTHY{background:var(--ok)}.state-dot.STARTING,.state-dot.DEGRADED{background:var(--warn)}.state-dot.DEAD{background:var(--bad)}.state-dot.DISABLED,.state-dot.STOPPED{background:var(--off)}
.badge{font-size:10px;color:var(--muted);white-space:nowrap;border:1px solid var(--line);border-radius:6px;padding:3px 6px;background:#111419}
.metrics{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:4px}
.metric{min-width:0}
.metric b{display:block;font-size:12px;font-weight:620;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.metric span{display:block;color:var(--subtle);font-size:9px;margin-top:2px;text-transform:uppercase;letter-spacing:.04em}
.layers{display:flex;gap:5px;flex-wrap:wrap;margin-top:9px}
.layer{font-size:9px;color:var(--subtle);border:1px solid var(--line2);border-radius:5px;padding:3px 5px;background:#101318}
.layer b{color:#c9d0d8;font-weight:620}
.provider-actions{display:flex;gap:6px;justify-content:flex-end}
.provider-error{font-size:10px;color:#e69ca2;margin-top:8px;line-height:1.45}
.search{width:min(310px,48vw);background:#111419;color:var(--text);border:1px solid var(--line);border-radius:8px;padding:8px 10px;font-size:12px;outline:none}
.search:focus{border-color:#535d69}
.table-wrap{border:1px solid var(--line);border-radius:10px;overflow:auto;background:var(--panel)}
table{width:100%;border-collapse:collapse;min-width:570px}
th,td{text-align:left;padding:9px 11px;border-bottom:1px solid var(--line2);font-size:11px}
th{color:var(--subtle);font-size:9px;text-transform:uppercase;letter-spacing:.05em;font-weight:650;background:#111419;position:sticky;top:0}
tr:last-child td{border-bottom:0}
tbody tr:hover{background:#171b20}
code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;color:#d8dde3}
.footer{margin-top:17px;padding-top:13px;border-top:1px solid var(--line);font-size:10px;line-height:1.5;color:var(--subtle)}
@media(max-width:820px),(hover:none) and (pointer:coarse){
 main{padding:14px 12px calc(30px + env(safe-area-inset-bottom))}
 header{padding-bottom:13px}.mark{width:30px;height:30px}h1{font-size:17px}
 .overview{grid-template-columns:1fr;padding:14px 0}
 .base-row{align-items:stretch;flex-direction:column;max-width:none}
 .base-row button{min-height:42px}
 .summary{width:100%}
 .providers{border-radius:9px}
 .card{padding:13px}
 .card-top{grid-template-columns:1fr;gap:11px}
 .provider-actions{justify-content:stretch;display:grid;grid-template-columns:1fr 1fr}
 .provider-actions .btn{min-height:42px;display:flex;align-items:center;justify-content:center;text-align:center}
 .metrics{border-top:1px solid var(--line2);padding-top:10px}
 .section-head{align-items:stretch;flex-direction:column}
 .search{width:100%;font-size:16px;min-height:42px}
 .table-wrap{overflow:hidden}
 table{min-width:0;table-layout:fixed}
 th,td{padding:10px 9px}
 th:first-child,td:first-child{width:68%}
 th:nth-child(2),td:nth-child(2){width:32%}
 th:nth-child(3),td:nth-child(3){display:none}
 td code{white-space:normal;overflow-wrap:anywhere;word-break:break-word}
}
@media(max-width:390px){
 main{padding-left:10px;padding-right:10px}
 .provider-actions{grid-template-columns:1fr}
}
</style>
</head>
<body>
<main>
<header>
 <div class="brand">
  <div class="mark">AhB</div>
  <div><h1>Local Gateway</h1><div id="runtime" class="kicker">正在讀取執行狀態…</div></div>
 </div>
 <button id="refresh">重新整理</button>
</header>

<div class="overview">
 <div>
  <div class="overview-title">統一 API</div>
  <div class="desc">所有來源維持明確的 provider/model 命名；路由狀態可以在下方直接確認。</div>
  <div class="base-row"><div id="baseUrl" class="base">http://127.0.0.1:8317/v1</div><button id="copyBase">複製 Base URL</button></div>
 </div>
 <div class="summary">
  <div class="summary-item"><b id="providerTotal">—</b><span>Providers</span></div>
  <div class="summary-item"><b id="modelTotal">—</b><span>Models</span></div>
 </div>
</div>

<div id="error" class="notice"></div>
<div id="warnings" class="notice"></div>

<section>
 <div class="section-head"><div class="section-title"><h2>Providers</h2><span id="providerReady" class="count">—</span></div></div>
 <div id="providers" class="providers"></div>
</section>

<section>
 <div class="section-head">
  <div><div class="section-title"><h2>Models</h2><span id="modelCount" class="count">—</span></div><div class="desc">目前可路由 Provider 回報的模型目錄。</div></div>
  <input id="search" class="search" placeholder="搜尋模型或 Provider">
 </div>
 <div class="table-wrap">
  <table><thead><tr><th>Model ID</th><th>Provider</th><th>Upstream</th></tr></thead><tbody id="models"></tbody></table>
 </div>
</section>

<div class="footer">Provider 的帳號、Proxy、Quota 與進階設定保留在各自原生管理介面；AhB 只負責本機狀態、統一模型目錄與路由。</div>
</main>
<script>
const fmtBytes=n=>!n?'—':n<1048576?(n/1024).toFixed(1)+' KiB':(n/1048576).toFixed(1)+' MiB';
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let lastModels=[];
async function getJSON(path){const r=await fetch(path,{cache:'no-store'});if(!r.ok)throw new Error(path+' HTTP '+r.status);return r.json()}
function modelCounts(models){const out={};for(const m of models){const p=m.x_provider||'';out[p]=(out[p]||0)+1}return out}
function accountLabel(x){
 if(x.account_total!==null&&x.account_total!==undefined)return String(x.account_usable_count||0)+'/'+String(x.account_total);
 if(x.id==='opencode'&&x.account_usable===true)return 'ANON';
 return x.account_usable===true?'YES':(x.account_usable===false?'NO':'UNKNOWN');
}
function renderModels(){
 const q=(document.getElementById('search').value||'').trim().toLowerCase();
 const rows=lastModels.filter(m=>!q||String(m.id||'').toLowerCase().includes(q)||String(m.x_provider_name||m.x_provider||'').toLowerCase().includes(q));
 document.getElementById('modelCount').textContent=rows.length===lastModels.length?lastModels.length+' available':rows.length+' of '+lastModels.length;
 document.getElementById('models').innerHTML=rows.length?rows.map(m=>
  '<tr><td><code>'+esc(m.id)+'</code></td><td>'+esc(m.x_provider_name||m.x_provider||'—')+'</td><td><code>'+esc(m.x_upstream_id||'—')+'</code></td></tr>'
 ).join(''):'<tr><td colspan="3" class="muted">沒有符合的模型</td></tr>';
}
function actionLink(url,label,primary,enabled){
 if(!url||!enabled)return '<span class="btn disabled">'+esc(label)+'</span>';
 return '<a class="btn'+(primary?' primary':'')+'" href="'+esc(url)+'" target="_blank" rel="noopener noreferrer">'+esc(label)+'</a>';
}
async function copyText(text){
 try{await navigator.clipboard.writeText(text);return true}catch(_){}
 const area=document.createElement('textarea');area.value=text;area.style.position='fixed';area.style.opacity='0';document.body.appendChild(area);area.select();
 let ok=false;try{ok=document.execCommand('copy')}catch(_){}
 area.remove();return ok;
}
async function refresh(){
 const error=document.getElementById('error');error.style.display='none';
 const warnings=document.getElementById('warnings');warnings.style.display='none';
 try{
  const [p,m,r]=await Promise.all([getJSON('/api/providers'),getJSON('/v1/models'),getJSON('/api/runtime')]);
  lastModels=m.data||[];
  const counts=modelCounts(lastModels);
  const providers=p.providers||[];
  const sidecarRSS=providers.reduce((n,x)=>n+(x.rss_bytes||0),0);
  const ready=providers.filter(x=>x.enabled&&(x.state==='HEALTHY'||x.state==='DEGRADED')).length;
  document.getElementById('runtime').textContent='RSS '+fmtBytes((r.process_rss_bytes||0)+sidecarRSS)+' · '+(r.goos||'?')+'/'+(r.goarch||'?');
  document.getElementById('providerTotal').textContent=providers.filter(x=>x.enabled).length;
  document.getElementById('modelTotal').textContent=lastModels.length;
  document.getElementById('providerReady').textContent=ready+' active / '+providers.length+' configured';
  document.getElementById('providers').innerHTML=providers.map(x=>{
   const manageable=x.state==='HEALTHY'||x.state==='DEGRADED';
   const enabled=!!x.enabled;
   return '<article class="card"><div class="card-top">'+
    '<div class="provider-main"><div class="provider-name"><span class="state-dot '+esc(x.state)+'"></span>'+esc(x.display_name||x.id)+' <span class="badge">'+esc(x.state)+'</span></div><div class="desc">'+esc(x.description||x.id)+'</div>'+
     '<div class="layers"><span class="layer">Process <b>'+esc(x.kind==='external'?'N/A':(x.process_alive?'YES':'NO'))+'</b></span><span class="layer">Ready <b>'+esc(x.provider_ready?'YES':'NO')+'</b></span><span class="layer">Account <b>'+esc(accountLabel(x))+'</b></span></div>'+
     (x.last_error?'<div class="provider-error">'+esc(x.last_error)+'</div>':'')+'</div>'+
    '<div class="metrics"><div class="metric"><b>'+esc(counts[x.id]||0)+'</b><span>Models</span></div><div class="metric"><b>'+esc(fmtBytes(x.rss_bytes))+'</b><span>RSS</span></div><div class="metric"><b>'+esc(x.restarts||0)+'</b><span>Restarts</span></div></div>'+
    '<div class="provider-actions">'+actionLink(x.ui_url,'管理原本 UI',true,enabled&&manageable)+actionLink(x.docs_url,'上游文件',false,true)+'</div>'+
   '</div></article>';
  }).join('');
  if(m.x_provider_warnings&&Object.keys(m.x_provider_warnings).length){
   warnings.textContent=Object.entries(m.x_provider_warnings).map(([k,v])=>k+': '+v).join('\n');warnings.style.display='block';
  }
  renderModels();
 }catch(e){error.textContent=e.message;error.style.display='block'}
}
document.getElementById('refresh').addEventListener('click',refresh);
document.getElementById('search').addEventListener('input',renderModels);
document.getElementById('copyBase').addEventListener('click',async()=>{
 const btn=document.getElementById('copyBase'),text=document.getElementById('baseUrl').textContent;
 btn.textContent=(await copyText(text))?'已複製':'複製失敗';setTimeout(()=>btn.textContent='複製 Base URL',1200);
});
document.getElementById('baseUrl').textContent=location.origin+'/v1';
refresh();setInterval(refresh,10000);
</script>
</body>
</html>`

func (h *Hub) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(dashboardHTML))
}