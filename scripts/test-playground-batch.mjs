// Browser-free regression for the embedded Playground batch runner.
// Runs the real script in a minimal DOM with mocked local AhB endpoints.
// No accounts, network access, or model quota are needed.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const source=fs.readFileSync(process.argv[2],'utf8')
 .replace('__AHB_CONTROL_TOKEN__','fixture-page-nonce')
 .replace('__AHB_PROVIDER_CONTROL_AVAILABLE__','true');

function mockNode(id=''){
 return {
  id,value:'',disabled:false,checked:false,textContent:'',className:'',children:[],handlers:{},
  style:{},options:[],classList:{add(){},remove(){},toggle(){}},
  addEventListener(type,fn){this.handlers[type]=fn},
  replaceChildren(...items){this.children=[...items];this.options=[...items];if(id==='provider')this.value=''},
  appendChild(x){this.children.push(x);this.options.push(x);if(id==='provider'&&!this.value)this.value=x.value},
  setAttribute(){},removeAttribute(){},querySelector(){return null}
 };
}
function createWorkbench({statuses={},replyFor,providers,models,onModelsFetch}={}){
 const ids=new Map(),requests=[],seenWake=[];
 const node=id=>{if(!ids.has(id))ids.set(id,mockNode(id));return ids.get(id)};
 const document={
  getElementById:node,createElement:()=>mockNode(),
  querySelector:()=>mockNode('presets'),
 };
 const p=providers||[
  {id:'opencode',enabled:true,start_mode:'always',process_alive:true,display_name:'OpenCode'},
  {id:'freebuff',enabled:true,start_mode:'always',process_alive:true},
  {id:'deepseek',enabled:true,start_mode:'always',process_alive:true},
  {id:'duckai',enabled:false,start_mode:'always',process_alive:false},
 ];
 const m=models||[
  {x_provider:'opencode',x_upstream_id:'first'},
  {x_provider:'opencode',x_upstream_id:'second'},
  {x_provider:'opencode',x_upstream_id:'third'},
  {x_provider:'freebuff',x_upstream_id:'model'},
  {x_provider:'deepseek',x_upstream_id:'deepseek-chat'},
  {x_provider:'duckai',x_upstream_id:'unavailable'}
 ];
 const fetch=async(path,options={})=>{
  if(path==='/api/providers')return {ok:true,json:async()=>({providers:p})};
  if(path==='/v1/models')return onModelsFetch?onModelsFetch(options):{ok:true,json:async()=>({data:m})};
  if(path.startsWith('/api/control/wake/')){
   seenWake.push(path);
   return {ok:true,status:200};
  }
  if(path==='/api/control/playground'){
   assert.equal(options.method,'POST');
   assert.equal(options.headers['X-AhB-Control-Token'],'fixture-page-nonce');
   assert.equal(options.credentials,'same-origin');
   const request=JSON.parse(options.body);requests.push(request);
   assert.equal(request.stream,false);
   assert.equal(request.tool_test,false);
   assert.equal(request.max_tokens,64);
   assert.equal(request.prompt,'請只回答 OK。');
   if(replyFor)return replyFor(request,options);
   const status=statuses[request.model]||200;
   return {
    ok:status===200,status,
    text:async()=>status===200?'{"choices":[{"message":{"content":"OK"}}]}':'{"error":"unused"}'
   };
  }
  throw new Error('Unexpected route '+path);
 };
 node('batchLimit').value='20';
 node('batchScope').value='current';
 const context={document,fetch,navigator:{clipboard:{writeText:async()=>{}}},
  URLSearchParams,location:{search:''},AbortController,performance:{now:()=>Date.now()},
  setTimeout,clearTimeout,console};
 vm.runInNewContext(source,context,{timeout:3000});
 const ready=new Promise(resolve=>setImmediate(resolve));
 return {node,ready,requests,seenWake,context};
}

async function testStopsProviderAfter429And502(){
 const app=createWorkbench({statuses:{'opencode/second':429,'freebuff/model':502}});
 await app.ready;
 app.node('batchScope').value='all';
 await app.node('batchStart').handlers.click();
 assert.deepEqual(app.requests.map(r=>r.model),['opencode/first','opencode/second','freebuff/model']);
 const labels=app.node('batchRows').children.map(row=>row.children[1].textContent);
 assert.equal(labels.filter(x=>x==='成功').length,1);
 assert.equal(labels.filter(x=>x==='失敗').length,2);
 assert.ok(labels.filter(x=>x==='略過').length>=2);
 assert.match(app.node('batchSummary').textContent,/已送出 3/);
 assert.equal(app.node('batchStop').disabled,true);
 assert.equal(app.node('batchCopy').disabled,false);
 assert.equal(app.node('send').disabled,false);
}
async function testCapsAndSkipsDisabledProviders(){
 const app=createWorkbench();
 await app.ready;
 app.node('batchLimit').value='1';
 app.node('batchScope').value='all';
 await app.node('batchStart').handlers.click();
 assert.deepEqual(app.requests.map(x=>x.model),['opencode/first']);
 assert.equal(app.requests.length,1);
 assert.equal(app.seenWake.length,0);
}
async function testAbortStopsRemainingModels(){
 let entered;
 const awaited=new Promise(resolve=>{entered=resolve});
 const app=createWorkbench({replyFor:(_req,opts)=>{
  entered();
  return new Promise((_resolve,reject)=>{
   opts.signal.addEventListener('abort',()=>reject(new Error('aborted')),{once:true});
  });
 }});
 await app.ready;
 const job=app.node('batchStart').handlers.click();
 await awaited;
 app.node('batchStop').handlers.click();
 await job;
 assert.equal(app.requests.length,1,'cancel must prevent sending a second model');
 assert.match(app.node('batchStatus').textContent,/已停止/);
 assert.equal(app.node('batchStart').disabled,false);
}
async function testExperimentalOptInAndDisabledStillOff(){
 const app=createWorkbench({providers:[
  {id:'deepseek',enabled:true,start_mode:'always',process_alive:true},
  {id:'duckai',enabled:false,start_mode:'always',process_alive:false},
 ],models:[
  {x_provider:'deepseek',x_upstream_id:'deepseek-chat'},
  {x_provider:'duckai',x_upstream_id:'duckai-disabled'},
 ]});
 await app.ready;
 await app.node('batchStart').handlers.click();
 assert.equal(app.requests.length,0,'experimental must remain opt-in');
 app.node('batchExperimental').checked=true;
 await app.node('batchStart').handlers.click();
 assert.deepEqual(app.requests.map(r=>r.model),['deepseek/deepseek-chat']);
}
async function testAbortWhileLoadingModelCatalog(){
 let started;
 const entered=new Promise(resolve=>{started=resolve});
 const app=createWorkbench({onModelsFetch:opts=>{
  // Initial page load is a normal read; the batch catalog request must
  // carry an AbortSignal so Stop does not wait for a hung /v1/models.
  if(!opts.signal)return {ok:true,json:async()=>({data:[{x_provider:'opencode',x_upstream_id:'first'}]})};
  started();
  return new Promise((_resolve,reject)=>{
   opts.signal.addEventListener('abort',()=>reject(new Error('aborted')),{once:true});
  });
 }});
 await app.ready;
 const job=app.node('batchStart').handlers.click();
 await entered;
 app.node('batchStop').handlers.click();
 await job;
 assert.equal(app.requests.length,0,'cancelling catalog fetch must not consume quota');
 assert.match(app.node('batchStatus').textContent,/已停止/,'cancellation is not a generic error');
 assert.equal(app.node('batchStart').disabled,false,'start button should recover after cancel');
}

await testStopsProviderAfter429And502();
await testCapsAndSkipsDisabledProviders();
await testAbortStopsRemainingModels();
await testAbortWhileLoadingModelCatalog();
await testExperimentalOptInAndDisabledStillOff();
console.log('batch Playground fixture PASS: cap, real requests, fault gating, opt-in, request + catalog cancellation');
