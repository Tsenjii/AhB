package hub

import (
 "net/http"
 "strings"
)

// Separate mobile workbench from the 1000-line administrative dashboard to
// avoid coupling OAuth/control UX with long-running SSE test interactions.
// Both pages use the same local, ephemeral management nonce.
const playgroundHTML = `<!doctype html>
<html lang="zh-TW"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<meta name="color-scheme" content="dark"><title>AhB · Playground</title>
<style>
:root{color-scheme:dark;font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;--bg:#0d0f12;--panel:#13161a;--line:#303741;--text:#eff3f8;--muted:#a4acb7;--accent:#d7e2f0;--ok:#68c58b;--bad:#e68890}
*{box-sizing:border-box}body{margin:0;min-height:100vh;background:var(--bg);color:var(--text)}
main{max-width:1090px;margin:auto;padding:18px 16px 54px}
header{display:flex;gap:10px;align-items:center;justify-content:space-between;padding:6px 0 19px;border-bottom:1px solid var(--line);margin-bottom:20px}
.brand{font-size:19px;font-weight:750;letter-spacing:-.03em}.brand small{display:block;color:var(--muted);font-size:11px;font-weight:450;margin-top:5px;letter-spacing:normal}
h1{font-size:23px;margin:2px 0 8px}p{font-size:12px;color:var(--muted);line-height:1.65;margin:4px 0 13px}
.layout{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:17px}
.card{border:1px solid var(--line);border-radius:13px;background:var(--panel);padding:18px;min-width:0}
.field{margin-bottom:14px}.field label{display:block;margin-bottom:7px;font-size:12px;font-weight:650;color:var(--muted)}
input,select,textarea,button{font:inherit}
.field input,.field select,.field textarea{width:100%;border:1px solid #47505c;background:#0d1117;color:var(--text);padding:11px;border-radius:9px;font-size:15px;min-height:44px}
textarea{resize:vertical;min-height:148px!important;line-height:1.5}
.row{display:grid;grid-template-columns:1fr 1fr;gap:10px}
label.check{display:flex;align-items:center;gap:9px;margin:12px 0;color:var(--muted);font-size:12px}
.check input{width:19px;height:19px;accent-color:var(--accent)}
.actions{display:flex;gap:9px;flex-wrap:wrap;margin-top:17px}
button,.btn{display:inline-grid;place-items:center;border:1px solid #47505c;background:#1b2028;color:var(--text);padding:11px 13px;border-radius:9px;font-size:12px;cursor:pointer;text-decoration:none;min-height:43px}
button.primary{background:#d8e4f4;color:#172230;border-color:#d8e4f4;font-weight:740}
button:disabled{opacity:.45;cursor:not-allowed}
.output{background:#0d1117;border:1px solid var(--line);padding:14px;border-radius:9px;min-height:170px;max-height:440px;overflow:auto;white-space:pre-wrap;overflow-wrap:anywhere;font:13px/1.7 ui-monospace,SFMono-Regular,Consolas,monospace}
.status{color:var(--muted);font-size:13px;line-height:1.65;margin-bottom:12px}
.status.good{color:var(--ok)}.status.bad{color:var(--bad)}
.metrics{display:flex;gap:7px;flex-wrap:wrap;margin-bottom:13px}
.metrics span{font-size:11px;color:var(--muted);border:1px solid var(--line);border-radius:7px;padding:6px 9px}
details{margin-top:14px}summary{font-size:12px;color:var(--muted);cursor:pointer;padding:10px 0}
.note{border-top:1px solid var(--line);margin-top:15px;padding-top:13px}
@media(max-width:820px){.layout{grid-template-columns:1fr}main{padding:14px 12px 45px}.card{padding:14px}.row{grid-template-columns:1fr 1fr}}
@media(max-width:390px){.row{grid-template-columns:1fr}}
</style></head><body>
<main>
<header><div class="brand">AhB <small>Direct Gateway Playground</small></div><a href="/ui" class="btn">返回管理介面</a></header>
<h1>Playground</h1>
<p>對指定 Gateway 發出真正推論。**不經路由 fallback**，所以 Duck.ai 回 418 或 DeepSeek 回 503，不會偷偷換其他來源回答。只有按下「送出測試」才會使用帳號額度。</p>
<div class="layout">
<section class="card" aria-label="測試輸入">
 <div class="row">
  <div class="field"><label for="provider">Gateway</label><select id="provider"><option value="">載入來源中…</option></select></div>
  <div class="field"><label for="model">模型（可自行輸入）</label><input id="model" list="modelHints" spellcheck="false" autocomplete="off" placeholder="輸入上游模型名稱"><datalist id="modelHints"></datalist></div>
 </div>
 <div class="field"><label for="prompt">測試問題</label><textarea id="prompt" maxlength="6000" spellcheck="false">請用一句話介紹自己，然後回答 2 + 3 = ?</textarea></div>
 <div class="row">
  <div class="field"><label for="maxTokens">最大 Tokens</label><input id="maxTokens" type="number" min="1" max="1024" value="256"></div>
  <div><label class="check"><input id="stream" type="checkbox" checked>測試 SSE 串流</label><label class="check"><input id="toolTest" type="checkbox">傳送模擬 Tool Calling</label></div>
 </div>
 <div class="actions"><button id="send" type="button" class="primary" disabled>送出測試</button><button id="cancel" type="button" disabled>停止</button><button id="reload" type="button">更新來源／模型</button></div>
 <p class="note">睡眠來源的模型清單可能是快取；不知道模型時可先回管理頁面掃描。Playground 會按需啟動已啟用 Gateway，但**不會替你登入、不會自動重試、不會執行模擬工具**。</p>
</section>
<section class="card" aria-label="測試結果">
 <div class="status" id="status" role="status" aria-live="polite">尚未測試</div>
 <div class="metrics" id="metrics"><span>HTTP —</span><span>TTFT —</span><span>總耗時 —</span></div>
 <div class="output" id="output">模型的真實回答會顯示在這裡。</div>
 <details><summary>原始回應與 Tool Calls（僅本機顯示）</summary><pre id="raw" class="output" style="min-height:55px;max-height:260px"></pre></details>
 <p class="note">HTTP 200 不等於成功：必須有實際文字或結構化 tool_calls，SSE 還必須收到 [DONE]。HTTP 418 代表該來源拒絕請求；502／503 可能是帳號、上游或程序問題，不能靠重新啟動保證修復。</p>
</section></div>
</main>
<script>
const controlToken="__AHB_CONTROL_TOKEN__";
const controlsAvailable=__AHB_PROVIDER_CONTROL_AVAILABLE__;
const el=id=>document.getElementById(id);
const provider=el('provider'),model=el('model'),hints=el('modelHints'),statusBox=el('status');
const output=el('output'),raw=el('raw'),send=el('send'),cancel=el('cancel');
let providers=[],models=[],active=null;
function metric(code,ttft,total){
 el('metrics').replaceChildren();
 for(const label of ['HTTP '+code,'TTFT '+ttft,'總耗時 '+total]){
  const span=document.createElement('span');span.textContent=label;el('metrics').appendChild(span);
 }
}
async function getJSON(path){const r=await fetch(path,{cache:'no-store'});if(!r.ok)throw new Error('HTTP '+r.status);return r.json()}
async function reload(){
 const last=provider.value;
 statusBox.className='status';
 statusBox.textContent='讀取模型中（不會喚醒休眠來源）…';
 try{
  const data=await Promise.all([getJSON('/api/providers'),getJSON('/v1/models')]);
  providers=(data[0].providers||[]).filter(p=>p.enabled&&p.id!=='route');
  models=data[1].data||[];
  provider.replaceChildren();
  for(const p of providers){
   const option=document.createElement('option');option.value=p.id;
   option.textContent=(p.display_name||p.id)+' · '+p.id;provider.appendChild(option);
  }
  if(providers.some(p=>p.id===last))provider.value=last;
  chooseModels();
  send.disabled=!controlsAvailable||!providers.length||!!active;
  statusBox.textContent=providers.length?'選擇 Gateway 與模型後送出測試。':'沒有已啟用的 Gateway';
 }catch(err){statusBox.textContent='來源清單讀取失敗：'+err.message;send.disabled=true}
}
function chooseModels(){
 const items=models.filter(m=>m.x_provider===provider.value&&m.x_upstream_id);
 hints.replaceChildren();
 for(const item of items){
  const option=document.createElement('option');option.value=item.x_upstream_id;
  option.label=(item.x_cached?'休眠快取 · ':'')+item.x_upstream_id;hints.appendChild(option);
 }
 if(!items.some(m=>m.x_upstream_id===model.value))model.value=items.length?items[0].x_upstream_id:'';
 model.placeholder=items.length?'選擇或輸入模型':'請手動輸入真實模型 ID';
}
provider.addEventListener('change',chooseModels);
el('reload').addEventListener('click',reload);
cancel.addEventListener('click',()=>{if(active)active.abort()});
function contentText(v){
 if(typeof v==='string')return v;
 if(Array.isArray(v))return v.map(x=>typeof x?.text==='string'?x.text:'').join('');
 return '';
}
function hint(code){
 if(code===418)return '418：來源拒絕請求（Duck.ai 常見），重啟不等於修復。';
 if(code===429)return '429：來源限流或額度不足。';
 if(code===401||code===403)return code+'：登入或授權遭到拒絕。';
 if(code===503)return '503：來源或帳號未就緒，也可能是上游故障。';
 return code+'：Gateway 或上游回傳錯誤。';
}
function deltaEvent(packet,test){
 if(packet.error)throw new Error(typeof packet.error==='object'?packet.error.message||'串流錯誤':String(packet.error));
 for(const choice of packet.choices||[]){
  if(choice.finish_reason!==undefined&&choice.finish_reason!==null)test.finish=String(choice.finish_reason);
  const delta=choice.delta||{};
  const text=contentText(delta.content);
  if(text){
   test.text+=text;
   if(test.ttft===null)test.ttft=performance.now()-test.start;
   output.textContent=test.text;
  }
  for(const d of delta.tool_calls||[]){
   const idx=Number.isInteger(d.index)?d.index:0;
   if(!test.tools[idx])test.tools[idx]={id:'',name:'',arguments:''};
   const tool=test.tools[idx];
   if(d.id)tool.id=d.id;
   if(d.function?.name)tool.name+=d.function.name;
   if(d.function?.arguments)tool.arguments+=d.function.arguments;
  }
 }
}
async function readSSE(response,test){
 if(!response.body?.getReader)throw new Error('瀏覽器不支援 SSE 流式讀取');
 const reader=response.body.getReader(),dec=new TextDecoder();
 let buff='',size=0,done=false;
 function event(block){
  const payload=block.split('\n').filter(x=>x.startsWith('data:')).map(x=>x.slice(5).trimStart()).join('\n').trim();
  if(!payload)return;
  if(payload==='[DONE]'){done=true;return}
  let packet;
  try{packet=JSON.parse(payload)}catch(_){throw new Error('串流格式不是有效 JSON')}
  deltaEvent(packet,test);
 }
 while(true){
  const part=await reader.read();
  if(part.done)break;
  size+=part.value.byteLength;
  if(size>2*1024*1024)throw new Error('串流超過 2 MiB 測試限制');
  buff+=dec.decode(part.value,{stream:true});
  buff=buff.replace(/\r\n/g,'\n');
  let pos;
  while((pos=buff.indexOf('\n\n'))>=0){
   event(buff.slice(0,pos));
   buff=buff.slice(pos+2);
  }
 }
 buff+=dec.decode();
 if(!done&&buff.trim())event(buff.replace(/\r\n/g,'\n'));
 if(!done)throw new Error('串流沒有收到 [DONE]，不視為成功');
}
send.addEventListener('click',async()=>{
 if(active)return;
 const id=provider.value,upstream=model.value.trim(),prompt=el('prompt').value.trim();
 const max=Number(el('maxTokens').value);
 if(!id||!upstream||!prompt){statusBox.textContent='請選擇來源、模型和輸入問題。';return}
 if(upstream.length>200||prompt.length>6000||!Number.isInteger(max)||max<1||max>1024){
  statusBox.textContent='模型名稱、問題長度或 Tokens 不符合限制。';return
 }
 const ctrl=new AbortController();active=ctrl;send.disabled=true;cancel.disabled=false;
 let http='—';
 const test={start:performance.now(),ttft:null,text:'',tools:[],finish:''};
 output.textContent='';raw.textContent='';metric('—','—','—');
 statusBox.className='status';statusBox.textContent='直接向 '+id+' 發送真實請求…';
 const timeout=setTimeout(()=>ctrl.abort(),120000);
 try{
  const streaming=el('stream').checked;
  const response=await fetch('/api/control/playground',{
   method:'POST',credentials:'same-origin',cache:'no-store',signal:ctrl.signal,
   headers:{'Content-Type':'application/json','X-AhB-Control-Token':controlToken},
   body:JSON.stringify({model:id+'/'+upstream,prompt,stream:streaming,tool_test:el('toolTest').checked,max_tokens:max})
  });
  http=String(response.status);
  if(!response.ok){
   const reason=(await response.text()).slice(0,1200);
   throw new Error(hint(response.status)+'\n'+reason);
  }
  if(streaming)await readSSE(response,test);
  else{
   let packet;
   try{packet=JSON.parse((await response.text()).slice(0,2*1024*1024))}
   catch(_){throw new Error('HTTP 200，但回傳不是合法 Chat Completions JSON')}
   raw.textContent=JSON.stringify(packet,null,2).slice(0,10000);
   if(packet.error)throw new Error(packet.error.message||'上游錯誤');
   const choice=(packet.choices||[])[0]||{};
   test.text=contentText((choice.message||{}).content);
   test.tools=Array.isArray(choice.message?.tool_calls)?choice.message.tool_calls:[];
   test.finish=choice.finish_reason||'';
   if(test.text||test.tools.length)test.ttft=performance.now()-test.start;
   output.textContent=test.text;
  }
  const tools=test.tools.filter(Boolean);
  if(tools.length){
   raw.textContent=JSON.stringify({finish_reason:test.finish,tool_calls:tools},null,2).slice(0,10000);
   if(!test.text)output.textContent='模型回傳 '+tools.length+' 個 Tool Call（未執行工具）。';
  }
  if(!test.text&&!tools.length)throw new Error('HTTP 200 但沒有實際回答或工具呼叫，不算成功');
  statusBox.className='status good';
  statusBox.textContent='成功 · '+id+' 直接路由 · finish_reason: '+(test.finish||'未提供');
 }catch(error){
  statusBox.className='status bad';
  statusBox.textContent='失敗 · '+id+' 直接路由';
  output.textContent=(ctrl.signal.aborted?'測試已取消／逾時。':error.message)
   +(test.text?'\n\n先前部分輸出（整體仍失敗）：\n'+test.text:'');
 }finally{
  metric(http,test.ttft===null?'—':Math.round(test.ttft)+' ms',Math.round(performance.now()-test.start)+' ms');
  clearTimeout(timeout);active=null;cancel.disabled=true;send.disabled=!controlsAvailable||!provider.value;
 }
});
reload();
</script></body></html>`

func (h *Hub) handlePlaygroundUI(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodGet{
  w.Header().Set("Allow","GET");http.Error(w,"GET required",405);return
 }
 w.Header().Set("Content-Type","text/html; charset=utf-8")
 w.Header().Set("Cache-Control","no-store")
 w.Header().Set("X-Content-Type-Options","nosniff")
 w.Header().Set("Referrer-Policy","no-referrer")
 w.Header().Set("X-Frame-Options","DENY")
 w.Header().Set("Content-Security-Policy","frame-ancestors 'none'")
 html:=strings.Replace(playgroundHTML,"__AHB_CONTROL_TOKEN__",h.controlToken,1)
 available:="false"
 if h.controlToken!=""&&h.restartFn!=nil&&!h.cfg.AllowLAN{available="true"}
 html=strings.Replace(html,"__AHB_PROVIDER_CONTROL_AVAILABLE__",available,1)
 _,_=w.Write([]byte(html))
}
