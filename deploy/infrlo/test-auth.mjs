import assert from 'node:assert/strict';
import http from 'node:http';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { setTimeout as sleep } from 'node:timers/promises';
import { test } from 'node:test';

const secret = 'test-token-only-0123456789abcdef-0123456789abcdef';
const port = 18777;
const service = http.createServer(async (req, res) => {
  if (req.url === '/healthz') {
    res.end(JSON.stringify({status:'ok',private:'never-share'}));
    return;
  }
  if (req.url?.startsWith('/v1/chat')) {
    res.writeHead(200, {'Content-Type':'text/event-stream'});
    res.write('data: {"choices":[{"delta":{"content":"ok"}}]}\n\n');
    await sleep(20);
    res.end('data: [DONE]\n\n');
    return;
  }
  res.setHeader('Content-Type','application/json');
  res.end(JSON.stringify({path:req.url,auth:req.headers.authorization||null,
    apiKey:req.headers['x-api-key']||null,forwarded:req.headers['x-forwarded-for']||null}));
});

await test('secure ingress: auth, safe health, stripped headers, control denied, SSE', async () => {
  await new Promise((resolve,reject)=>service.once('error',reject).listen(8317,'127.0.0.1',resolve));
  let child;
  try {
    child = spawn(process.execPath,[new URL('./auth-gateway.mjs',import.meta.url).pathname],{
      env:{...process.env,AHB_PUBLIC_TOKEN:secret,AHB_PUBLIC_USER:'ahb',PORT:String(port)},
      stdio:['ignore','pipe','pipe']
    });
    let ready=false;
    for(let i=0;i<35;i++){
      if(child.exitCode!==null)break;
      try{if((await fetch('http://127.0.0.1:'+port+'/healthz')).ok){ready=true;break;}}catch{}
      await sleep(60);
    }
    assert.ok(ready,'ingress did not start');
    let r=await fetch('http://127.0.0.1:'+port+'/healthz');
    assert.deepEqual(await r.json(),{status:'ok'});
    r=await fetch('http://127.0.0.1:'+port+'/v1/models');
    assert.equal(r.status,401);
    r=await fetch('http://127.0.0.1:'+port+'/v1/models',{headers:{Authorization:'Bearer bad'}});
    assert.equal(r.status,401);
    r=await fetch('http://127.0.0.1:'+port+'/v1/models',{
      headers:{Authorization:'Bearer '+secret,'X-Forwarded-For':'untrusted'}});
    assert.equal(r.status,200);
    assert.deepEqual(await r.json(),{path:'/v1/models',auth:null,apiKey:null,forwarded:null});
    r=await fetch('http://127.0.0.1:'+port+'/ui',{
      headers:{Authorization:'Basic '+Buffer.from('ahb:'+secret).toString('base64')}});
    assert.equal(r.status,200);
    r=await fetch('http://127.0.0.1:'+port+'/v1/messages',{headers:{'X-Api-Key':secret}});
    assert.equal(r.status,200);
    assert.deepEqual(await r.json(),{path:'/v1/messages',auth:null,apiKey:null,forwarded:null});
    r=await fetch('http://127.0.0.1:'+port+'/api/control/wake/opencode',{
      method:'POST',headers:{Authorization:'Bearer '+secret}});
    assert.equal(r.status,403);
    r=await fetch('http://127.0.0.1:'+port+'/v1/chat/completions',{
      method:'POST',headers:{Authorization:'Bearer '+secret,'Content-Type':'application/json'},body:'{}'});
    assert.equal(r.status,200);
    const sse=await r.text();
    assert.ok(sse.includes('data: [DONE]') && sse.includes('"content":"ok"'));
  } finally {
    if(child){child.kill('SIGTERM');await Promise.race([once(child,'exit'),sleep(1000)]).catch(()=>{});}
    await new Promise(resolve=>service.close(resolve));
  }
});
