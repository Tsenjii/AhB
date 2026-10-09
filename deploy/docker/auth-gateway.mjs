import http from 'node:http';
import { createHash, timingSafeEqual } from 'node:crypto';

const secret = process.env.AHB_PUBLIC_TOKEN || '';
const user = process.env.AHB_PUBLIC_USER || 'ahb';
const port = Number(process.env.PORT || 8080);
if (secret.length < 32 || /[\r\n]/.test(secret)) {
  console.error('AHB_PUBLIC_TOKEN must be a secret of at least 32 characters');
  process.exit(1);
}
if (!Number.isInteger(port) || port < 1 || port > 65535) {
  console.error('PORT must be a TCP port number');
  process.exit(1);
}
const expected = createHash('sha256').update(secret).digest();
function matches(candidate) {
  return typeof candidate === 'string' &&
    timingSafeEqual(createHash('sha256').update(candidate).digest(), expected);
}
function permitted(req) {
  const authorization = req.headers.authorization || '';
  if (authorization.startsWith('Bearer ')) return matches(authorization.slice(7));
  if (authorization.startsWith('Basic ')) {
    try {
      const raw = Buffer.from(authorization.slice(6), 'base64').toString('utf8');
      const colon = raw.indexOf(':');
      return colon >= 0 && raw.slice(0, colon) === user && matches(raw.slice(colon + 1));
    } catch { return false; }
  }
  // Anthropic Messages clients may use x-api-key instead of Authorization.
  return matches(req.headers['x-api-key']);
}
const gateway = http.createServer((req, res) => {
  let pathname;
  try {
    if (!req.url?.startsWith('/') || req.url.startsWith('//')) throw new Error('bad path');
    pathname = decodeURIComponent(new URL(req.url, 'http://localhost').pathname);
  } catch {res.writeHead(400);res.end('Invalid request target');return;}
  if ((req.method === 'GET' || req.method === 'HEAD') && pathname === '/healthz') {
    const probe = http.get('http://127.0.0.1:8317/healthz', {timeout: 2500}, up => {
      up.resume();
      const healthy = up.statusCode === 200;
      res.writeHead(healthy ? 200 : 503, {'Content-Type':'application/json','Cache-Control':'no-store'});
      res.end(req.method === 'HEAD' ? undefined : JSON.stringify({status: healthy ? 'ok' : 'starting'}));
    });
    probe.on('timeout', () => probe.destroy());
    probe.on('error', () => {if (!res.headersSent) {res.writeHead(503);res.end();}});
    return;
  }
  if (!permitted(req)) {
    res.writeHead(401, {'WWW-Authenticate':'Basic realm="AhB"','Cache-Control':'no-store'});
    res.end('Authentication required');
    return;
  }
  // AhB local management uses stricter CSRF semantics; never bypass them.
  if (pathname.startsWith('/api/control/')) {
    res.writeHead(403, {'Content-Type':'text/plain','Cache-Control':'no-store'});
    res.end('Administrative controls are available on trusted localhost only');
    return;
  }
  const headers = {...req.headers, host:'127.0.0.1:8317'};
  delete headers.authorization;
  delete headers['x-api-key'];
  delete headers.cookie;
  delete headers['proxy-authorization'];
  for (const key of Object.keys(headers)) {
    if (key.startsWith('x-forwarded-') || key === 'forwarded') delete headers[key];
  }
  const forward = http.request({
    hostname:'127.0.0.1', port:8317, method:req.method, path:req.url,
    headers, agent:false
  }, upstream => {
    const h = {...upstream.headers};
    delete h['set-cookie'];
    delete h['transfer-encoding'];
    delete h.connection;
    res.writeHead(upstream.statusCode || 502, h);
    upstream.pipe(res);
  });
  forward.on('error', () => {
    if (!res.headersSent) { res.writeHead(502); res.end('AhB internal service unavailable'); }
    else res.destroy();
  });
  req.on('aborted', () => forward.destroy());
  res.on('close', () => forward.destroy());
  req.pipe(forward);
});
gateway.headersTimeout = 30_000;
gateway.requestTimeout = 0; // SSE/tool streams may stay open for minutes.
gateway.listen(port, '0.0.0.0', () => console.log('AhB authenticated ingress listening on port '+port));
for (const sig of ['SIGINT','SIGTERM']) process.on(sig, () => gateway.close(() => process.exit(0)));
