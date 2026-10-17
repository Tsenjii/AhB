#!/usr/bin/env node
// Execute the actual Playground reload function in a small DOM/fetch fixture.
// This is a source-extracted regression test, NOT a real Safari smoke test.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const src=fs.readFileSync('internal/hub/playground_ui.go','utf8');
const start=src.indexOf('async function reload(){');
const end=src.indexOf('function chooseModels(){',start);
assert.ok(start>=0&&end>start,'reload() source anchors changed');
const fn=src.slice(start,end);
assert.match(fn,/catalogGeneration/);
assert.match(src,/catalogGeneration\+\+;[^\n]*\n\s*batchActive=true;/);
assert.match(src,/if\(provider\.value!==id\|\|batchActive\)return;/);

function deferred(){
 let resolve,reject;
 const promise=new Promise((yes,no)=>{resolve=yes;reject=no});
 return {promise,resolve,reject};
}
function harness(){
 const queue=[];
 const provider={
  value:'first',options:[],
  replaceChildren(){this.options=[];this.value='';},
  appendChild(option){this.options.push(option);if(this.options.length===1)this.value=option.value;}
 };
 const statusBox={className:'status',textContent:''},send={disabled:false};
 const env={provider,statusBox,send,controlsAvailable:true,location:{search:''},
  URLSearchParams,document:{createElement:()=>({})},chooseModels:()=>{},
  getJSON:()=>{const p=deferred();queue.push(p);return p.promise;}};
 vm.createContext(env);
 vm.runInContext("let providers=[],models=[],active=null,batchActive=false,catalogGeneration=0;\n"+
  fn+"\nthis.getModels=()=>models;this.setBatch=v=>{batchActive=v;};this.invalidate=()=>catalogGeneration++;",env);
 return {provider,statusBox,send,queue,reload:()=>env.reload(),
  models:()=>env.getModels(),setBatch:v=>env.setBatch(v),invalidate:()=>env.invalidate()};
}
{
 const h=harness(),old=h.reload(),recent=h.reload();
 h.queue[2].resolve({providers:[{id:'new',enabled:true}]});
 h.queue[3].resolve({data:[{x_upstream_id:'new'}]});await recent;
 h.queue[0].resolve({providers:[{id:'old',enabled:true}]});
 h.queue[1].resolve({data:[{x_upstream_id:'old'}]});await old;
 assert.equal(h.models()[0]?.x_upstream_id,'new');
}
{
 const h=harness(),old=h.reload(),recent=h.reload();
 h.queue[2].resolve({providers:[{id:'good',enabled:true}]});
 h.queue[3].resolve({data:[]});await recent;
 h.queue[0].reject(new Error('stale 503'));h.queue[1].resolve({data:[]});await old;
 assert.doesNotMatch(h.statusBox.textContent,/503/);
 assert.equal(h.send.disabled,false);
}
{
 const h=harness(),old=h.reload();h.provider.value='second';
 h.queue[0].resolve({providers:[{id:'first',enabled:true},{id:'second',enabled:true}]});
 h.queue[1].resolve({data:[]});await old;
 assert.equal(h.provider.value,'second');
}
{
 const h=harness(),old=h.reload();h.invalidate();h.setBatch(true);h.setBatch(false);
 h.queue[0].resolve({providers:[{id:'stale',enabled:true}]});
 h.queue[1].resolve({data:[]});await old;
 assert.equal(h.models().length,0);
}
console.log('PASS: source-extracted Playground reload race regression (4/4)');
