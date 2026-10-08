package hub

import "net/http"

const dashboardHTML = `<!doctype html>
<html lang="zh-TW">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<meta name="color-scheme" content="dark">
<title>Android AI Hub</title>
<style>
:root{color-scheme:dark;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#090b0f;color:#eef2f7}
*{box-sizing:border-box}
body{margin:0;min-height:100vh;background:radial-gradient(circle at top,#172033 0,#090b0f 42rem);color:#eef2f7}
main{max-width:1100px;margin:0 auto;padding:20px 16px 40px}
header{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin:6px 0 18px}
h1{font-size:24px;line-height:1.15;margin:0 0 7px}
h2{font-size:17px;margin:0}
.sub,.muted{color:#94a3b8}
.toolbar{display:flex;gap:8px;flex-wrap:wrap}
button,.btn{appearance:none;border:1px solid #334155;background:#18202d;color:#f8fafc;border-radius:10px;padding:9px 12px;font:inherit;font-size:13px;text-decoration:none;cursor:pointer}
button:hover,.btn:hover{background:#233047}.btn.primary{background:#2563eb;border-color:#3b82f6}.btn.disabled{opacity:.45;pointer-events:none}
.hero{border:1px solid #263244;background:rgba(16,22,32,.88);border-radius:16px;padding:16px;margin-bottom:14px}
.base-row{display:flex;gap:10px;align-items:center;margin-top:10px}
.base{flex:1;min-width:0;background:#090d13;border:1px solid #263244;border-radius:10px;padding:10px 12px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;overflow:auto;white-space:nowrap}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(270px,1fr));gap:12px}
.card{border:1px solid #263244;background:rgba(16,22,32,.92);border-radius:16px;padding:15px;min-width:0}
.row{display:flex;align-items:center;justify-content:space-between;gap:10px}.row.start{align-items:flex-start}
.provider-name{font-weight:750;font-size:16px}.desc{font-size:12px;color:#94a3b8;margin-top:5px;line-height:1.5}
.badge{padding:4px 8px;border-radius:999px;font-size:11px;font-weight:700;background:#29313b;white-space:nowrap}
.HEALTHY{background:#123b2b;color:#7ef0b0}.STARTING{background:#413412;color:#ffd56b}.DEGRADED{background:#493315;color:#ffc56b}.DEAD{background:#491b23;color:#ff9aa5}.DISABLED,.STOPPED{background:#28303b;color:#aab5c5}
.metrics{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:14px 0}
.metric{background:#0c1119;border:1px solid #202a39;border-radius:11px;padding:9px;min-width:0}
.metric b{display:block;font-size:13px;overflow:hidden;text-overflow:ellipsis}.metric span{display:block;color:#7f8da2;font-size:10px;margin-top:3px}
.actions{display:flex;gap:8px;flex-wrap:wrap}
section{margin-top:20px}
.section-head{display:flex;justify-content:space-between;align-items:center;gap:12px;margin-bottom:9px}
.search{width:min(340px,55vw);background:#0c1119;color:#eef2f7;border:1px solid #263244;border-radius:10px;padding:9px 11px}
.table-wrap{border:1px solid #263244;border-radius:14px;overflow:auto;background:#0f151e}
table{width:100%;border-collapse:collapse;min-width:560px}
th,td{text-align:left;padding:10px 12px;border-bottom:1px solid #202a39;font-size:12px}
th{color:#94a3b8;background:#111925;position:sticky;top:0}tr:last-child td{border-bottom:0}
code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}
#error{display:none;background:#43191d;color:#fecaca;border:1px solid #7f1d1d;padding:10px;border-radius:11px;margin:12px 0}
#warnings{display:none;background:#3c2c0b;color:#fde68a;border:1px solid #6d5315;padding:10px;border-radius:11px;margin:12px 0;font-size:12px;white-space:pre-wrap}
.footer{margin-top:18px;font-size:11px;color:#66758b}
@media(max-width:820px){
 main{padding:14px 12px calc(28px + env(safe-area-inset-bottom))}
 header{align-items:center;margin:2px 0 14px}
 h1{font-size:21px}.sub{font-size:12px;line-height:1.45}
 .hero{padding:13px;border-radius:14px}
 .base-row{align-items:stretch;flex-direction:column}
 .base-row button{width:100%;min-height:44px}
 .grid{grid-template-columns:1fr;gap:10px}
 .card{padding:14px;border-radius:14px}
 .provider-name{font-size:17px}
 .desc{font-size:12px}
 .metrics{grid-template-columns:repeat(3,1fr);margin:12px 0}
 .metric{padding:10px}.metric b{font-size:14px}
 .actions{display:grid;grid-template-columns:1fr 1fr}
 .actions .btn{text-align:center;min-height:44px;display:flex;align-items:center;justify-content:center}
 section{margin-top:18px}
 .section-head{align-items:stretch;flex-direction:column}
 .search{width:100%;min-height:44px;font-size:16px}
 .table-wrap{overflow:hidden}
 table{min-width:0;table-layout:fixed}
 th,td{padding:11px 10px;font-size:12px}
 th:first-child,td:first-child{width:68%}
 th:nth-child(2),td:nth-child(2){width:32%}
 th:nth-child(3),td:nth-child(3){display:none}
 td code{white-space:normal;overflow-wrap:anywhere;word-break:break-word}
 .footer{font-size:10px;line-height:1.5}
 button,.btn{font-size:13px}
}
@media(max-width:390px){
 main{padding-left:10px;padding-right:10px}
 .metrics{gap:6px}.metric{padding:8px}
 .actions{grid-template-columns:1fr}
}
</style>
</head>
<body>
<main>
<header>
  <div>
    <h1>Android AI Hub</h1>
    <div id="runtime" class="sub">正在讀取執行狀態…</div>
  </div>
  <div class="toolbar"><button id="refresh">重新整理</button></div>
</header>

<div class="hero">
  <div class="row"><div><strong>統一 API</strong><div class="desc">所有模型都用 provider/model 命名，不會偷偷切換來源。</div></div><span id="modelTotal" class="badge">— models</span></div>
  <div class="base-row"><div id="baseUrl" class="base">http://127.0.0.1:8317/v1</div><button id="copyBase">複製 Base URL</button></div>
</div>

<div id="error"></div>
<div id="warnings"></div>
<div id="providers" class="grid"></div>

<section>
  <div class="section-head">
    <div><h2>模型</h2><div class="desc">目前健康 Provider 回報的模型目錄。</div></div>
    <input id="search" class="search" placeholder="搜尋模型 / Provider">
  </div>
  <div class="table-wrap">
    <table>
      <thead><tr><th>Model ID</th><th>Provider</th><th>Upstream</th></tr></thead>
      <tbody id="models"></tbody>
    </table>
  </div>
</section>

<div class="footer">管理 Provider 時會直接打開各上游原本的 WebUI；Hub 不重做它們已經有的帳號、Proxy、Quota、Playground 與診斷功能。</div>
</main>
<script>
const fmtBytes=n=>!n?'—':n<1048576?(n/1024).toFixed(1)+' KiB':(n/1048576).toFixed(1)+' MiB';
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let lastModels=[];
async function getJSON(path){const r=await fetch(path,{cache:'no-store'});if(!r.ok)throw new Error(path+' HTTP '+r.status);return r.json()}
function modelCounts(models){const out={};for(const m of models){const p=m.x_provider||'';out[p]=(out[p]||0)+1}return out}
function renderModels(){
 const q=(document.getElementById('search').value||'').trim().toLowerCase();
 const rows=lastModels.filter(m=>!q||String(m.id||'').toLowerCase().includes(q)||String(m.x_provider_name||m.x_provider||'').toLowerCase().includes(q));
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
  document.getElementById('runtime').textContent='總 RSS '+fmtBytes((r.process_rss_bytes||0)+sidecarRSS)+' · Hub '+fmtBytes(r.process_rss_bytes)+' · '+(r.goos||'?')+'/'+(r.goarch||'?');
  document.getElementById('modelTotal').textContent=lastModels.length+' models';
  document.getElementById('providers').innerHTML=providers.map(x=>{
    const manageable=x.state==='HEALTHY'||x.state==='DEGRADED';
    const enabled=!!x.enabled;
    return '<article class="card">'+
      '<div class="row start"><div><div class="provider-name">'+esc(x.display_name||x.id)+'</div><div class="desc">'+esc(x.description||x.id)+'</div></div><span class="badge '+esc(x.state)+'">'+esc(x.state)+'</span></div>'+
      '<div class="metrics">'+
        '<div class="metric"><b>'+esc(counts[x.id]||0)+'</b><span>Models</span></div>'+
        '<div class="metric"><b>'+esc(fmtBytes(x.rss_bytes))+'</b><span>RSS</span></div>'+
        '<div class="metric"><b>'+esc(x.restarts||0)+'</b><span>Restarts</span></div>'+
      '</div>'+
      (x.last_error?'<div class="desc" style="color:#fda4af;margin-bottom:10px">'+esc(x.last_error)+'</div>':'')+
      '<div class="actions">'+actionLink(x.ui_url,'管理原本 UI',true,enabled&&manageable)+actionLink(x.docs_url,'上游文件',false,true)+'</div>'+
    '</article>';
  }).join('');
  if(m.x_provider_warnings&&Object.keys(m.x_provider_warnings).length){
    warnings.textContent=Object.entries(m.x_provider_warnings).map(([k,v])=>k+': '+v).join('\n');
    warnings.style.display='block';
  }
  renderModels();
 }catch(e){error.textContent=e.message;error.style.display='block'}
}
document.getElementById('refresh').addEventListener('click',refresh);
document.getElementById('search').addEventListener('input',renderModels);
document.getElementById('copyBase').addEventListener('click',async()=>{
 const btn=document.getElementById('copyBase');const text=document.getElementById('baseUrl').textContent;
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