#!/usr/bin/env node
// Exercises the actual embedded chooseModels() and wake callback with fake DOM.
// This is a DOM fixture, not a physical browser test.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
const src=fs.readFileSync('internal/hub/playground_ui.go','utf8');
const chooseStart=src.indexOf('function chooseModels(){');
const chooseEnd=src.indexOf("provider.addEventListener('change',chooseModels);",chooseStart);
const wakeStart=src.indexOf("loadModels.addEventListener('click',async()=>{",chooseEnd);
const wakeEnd=src.indexOf('const presets=',wakeStart);
assert.ok(chooseStart>=0&&chooseEnd>chooseStart&&wakeStart>chooseEnd&&wakeEnd>wakeStart,'control anchors changed');
const choose=src.slice(chooseStart,chooseEnd),wake=src.slice(wakeStart,wakeEnd);
function deferred(){let resolve;const promise=new Promise(ok=>resolve=ok);return {promise,resolve}}
function harness(){
 const handlers={};
 const provider={value:'first',addEventListener:(event,cb)=>handlers['provider:'+event]=cb};
 const loadModels={disabled:true,addEventListener:(event,cb)=>handlers['load:'+event]=cb};
 const model={value:'',placeholder:''},modelSelect={value:'',replaceChildren(){},appendChild(){}};
 const statusBox={className:'status',textContent:''},pending=deferred();
 const env={provider,loadModels,model,modelSelect,statusBox,controlsAvailable:true,
  controlToken:'fixture',document:{createElement:()=>({})},el:()=>({textContent:''}),
  fetch:()=>pending.promise,encodeURIComponent,reload:async()=>true};
 vm.createContext(env);
 vm.runInContext("let providers=[{id:'first',enabled:true,start_mode:'on_demand'}],models=[],active=null,lastProvider='',batchActive=false;\n"+
  choose+"\n"+wake+"\nthis.setBatch=v=>batchActive=v;this.setActive=v=>active=v;",env);
 return {provider,loadModels,pending,choose:()=>env.chooseModels(),
  click:()=>handlers['load:click'](),setBatch:v=>env.setBatch(v),setActive:v=>env.setActive(v)};
}
{
 const h=harness();h.setBatch(true);h.choose();
 assert.equal(h.loadModels.disabled,true,'batch navigation re-enabled wake');
}
{
 const h=harness();h.choose();
 assert.equal(h.loadModels.disabled,false,'idle control should remain usable');
 h.setActive({});h.choose();
 assert.equal(h.loadModels.disabled,true,'active chat should disable wake');
}
{
 const h=harness();h.choose();const task=h.click();h.setBatch(true);
 h.pending.resolve({ok:true});await task;
 assert.equal(h.loadModels.disabled,true,'pending wake re-enabled control during batch');
}
{
 const h=harness();h.choose();const task=h.click();h.setActive({});
 h.pending.resolve({ok:true});await task;
 assert.equal(h.loadModels.disabled,true,'pending wake re-enabled control during chat');
}
console.log('PASS: Playground interaction guards (4/4)');
