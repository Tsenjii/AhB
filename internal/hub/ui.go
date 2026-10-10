package hub

import (
	"net/http"
	"strings"
)

const dashboardHTML = `<!doctype html>
<html lang="zh-TW">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<meta name="color-scheme" content="dark">
<title>AhB · 控制中心</title>
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
.provider-recovery-note{font-size:11px;color:var(--warn);margin-top:9px}
.proxy-config{margin-top:9px;border-top:1px solid var(--line2);padding-top:8px;color:var(--muted);font-size:11px}
.proxy-config summary{cursor:pointer;min-height:36px;display:flex;align-items:center;gap:8px}
.proxy-input-row{display:flex;align-items:center;gap:7px;flex-wrap:wrap;margin-top:8px}
.proxy-input-row input{flex:1;min-width:180px;background:#111419;color:var(--text);border:1px solid var(--line);border-radius:8px;padding:11px;font-size:13px}
.proxy-help{font-size:10px;color:var(--subtle);line-height:1.5;margin-top:6px}
.console-row{display:flex;gap:12px;align-items:center;justify-content:space-between;flex-wrap:wrap;border-bottom:1px solid var(--line2);padding:12px 14px}
.console-row:last-child{border-bottom:none}
.console-detail{flex:1;min-width:190px}
.console-title{font-size:13px;font-weight:670;margin-bottom:4px}
.console-help{font-size:11px;color:var(--muted);line-height:1.5}
.console-actions{display:flex;gap:7px;flex-wrap:wrap}
.console-actions .btn{min-height:42px}
@media(max-width:580px){.console-row{align-items:stretch;flex-direction:column}.console-actions{display:grid;grid-template-columns:1fr 1fr}.console-actions .btn{text-align:center;display:grid;place-items:center}}

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

/* AhB workbench: restrained dark, comfortable touch targets and clear hierarchy. */
:root{--bg:#101318;--panel:#181d24;--panel2:#1d232b;--line:#303741;--line2:#292f38;--text:#f3f5f7;--muted:#b0b9c5;--subtle:#86919f}
main{max-width:1160px;padding-top:28px}
.brand .mark{background:#e8eef6;color:#121821;border-color:#e8eef6;border-radius:10px;width:38px;height:38px}
h1{font-size:20px}
.kicker{font-size:12px}
button,.btn{font-size:13px;min-height:38px}
section{margin-top:27px}
h2{font-size:15px}
.section-head{margin-bottom:12px}
.overview{padding:24px 0}
.overview-title{font-size:20px;letter-spacing:-.03em}
.desc{font-size:12px}
.summary-item{padding:14px 17px}
.summary-item b{font-size:19px}
.providers,.table-wrap{box-shadow:0 1px 0 rgba(255,255,255,.015)}
.card{padding:17px 18px}
.provider-name{font-size:14px}
.metrics .metric b{font-size:13px}
.metrics .metric span{font-size:10px}
th,td{font-size:12px}
.section-note{font-size:12px;color:var(--muted);line-height:1.65}
.quick-connect{display:grid;grid-template-columns:minmax(0,.8fr) minmax(0,1.5fr);gap:20px;padding:22px;border:1px solid var(--line);border-radius:12px;background:var(--panel)}
.quick-connect p{margin:9px 0 0;font-size:12px;color:var(--muted);line-height:1.7}
.eyebrow{font-size:10px;font-weight:720;text-transform:uppercase;letter-spacing:.12em;color:var(--subtle)}
.quick-title{font-size:19px;font-weight:690;margin:9px 0 0;letter-spacing:-.02em}
.quick-form{display:grid;gap:11px}
.fieldrow{display:grid;grid-template-columns:minmax(0,.75fr) minmax(0,1.25fr);gap:11px}
.field{display:grid;gap:6px;min-width:0}
.field label{font-size:11px;font-weight:650;color:var(--muted)}
.field select,.field input{width:100%;height:41px;background:#10151b;color:var(--text);border:1px solid #3d4651;border-radius:8px;padding:8px 11px;font:13px ui-sans-serif,system-ui}
.field select:focus,.field input:focus{outline:2px solid #96b0cd;outline-offset:1px}
.command-panel{display:grid;gap:8px}
.command-box{border:1px solid var(--line);border-radius:8px;background:#0d1116;padding:11px;overflow-wrap:anywhere;font:12px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace;color:#d8e6f5}
.command-actions{display:flex;gap:9px;align-items:center;flex-wrap:wrap}
.command-help{font-size:11px;color:var(--subtle);line-height:1.6}
.tip-error{color:#eb9aa1}
.anchor-nav{display:flex;gap:8px;flex-wrap:wrap}
.anchor-nav a{color:var(--muted);font-size:12px;text-decoration:none;border:1px solid var(--line);padding:7px 11px;border-radius:7px}
.anchor-nav a:hover{color:var(--text);border-color:#687788}
@media(max-width:820px),(hover:none) and (pointer:coarse){
 main{padding-top:17px}
 .quick-connect{grid-template-columns:1fr;padding:16px;gap:17px}
 .overview-title{font-size:18px}
 .field select,.field input{min-height:45px;font-size:16px}
 .command-actions button{min-height:44px}
 .quick-title{font-size:18px}
}
@media(max-width:390px){.fieldrow{grid-template-columns:1fr}}
.anchor-nav{display:flex;gap:7px;overflow-x:auto;padding:5px 0 7px;scrollbar-width:none}
.anchor-nav button{white-space:nowrap;flex-shrink:0;background:transparent;border:1px solid transparent;color:var(--muted);padding:9px 12px;border-radius:9px;font-size:12px}
.anchor-nav button.selected{color:var(--text);background:var(--panel2);border-color:var(--line);font-weight:700}
.ahb-tab-hidden{display:none!important}
.ui-toggle{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-width:88px;border:1px solid var(--line);border-radius:8px;padding:8px 10px;font-size:12px;cursor:pointer;color:var(--text);background:var(--panel2)}
.ui-toggle input{accent-color:var(--ok);width:17px;height:17px;margin:0;cursor:pointer}
.ui-toggle:has(input:disabled){cursor:not-allowed;opacity:.45}
.quick-note{padding:11px 13px;border:1px solid var(--line);border-radius:10px;background:var(--panel);font-size:12px;line-height:1.7;color:var(--muted)}
@media(max-width:650px){.providers{grid-template-columns:1fr}.provider-actions{flex-wrap:wrap}.overview{grid-template-columns:1fr}.summary{max-width:none}main{padding:15px 12px 34px}}
@media(max-width:650px){
 header{flex-wrap:wrap;gap:12px}
 header>div:last-child{width:100%;display:grid!important;grid-template-columns:repeat(3,minmax(0,1fr));gap:6px!important}
 header>div:last-child button{font-size:12px;padding:11px 3px;min-width:0}
 .anchor-nav button{font-size:13px;min-height:44px}
 .ui-toggle{min-height:44px}
}
/* AhB mobile-first control surface, no external dependencies. */
:root{--bg:#10151c;--panel:#1b242f;--panel2:#283341;--line:#384657;--line2:#303b49;--text:#f2f6fa;--muted:#bac6d3;--subtle:#94a4b5;--accent:#a8c8ea;--ok:#80d2a6;--warn:#e5bd75;--bad:#ed959d}
body{line-height:1.5}main{max-width:1200px;padding-bottom:82px}.brand .mark{width:43px;height:43px;border-radius:12px;background:#2c4055;color:#e4f1ff;border-color:#58728f}h1{font-size:21px}h2{font-size:16px}
button,.btn{touch-action:manipulation}button:focus-visible,a:focus-visible,input:focus-visible,select:focus-visible,summary:focus-visible{outline:2px solid var(--accent);outline-offset:3px}
.anchor-nav{position:sticky;top:0;z-index:12;margin:0 -6px;padding:12px 6px;background:var(--bg);border-bottom:1px solid var(--line)}
.anchor-nav button,.anchor-nav .nav-play{display:inline-flex;align-items:center;justify-content:center;min-height:44px;padding:9px 17px;border-radius:10px;text-decoration:none;white-space:nowrap;font-size:13px;color:var(--muted);background:transparent;border:1px solid transparent}
.anchor-nav button.selected{background:#304257;color:#f2f7ff;border:1px solid #6d87a3}
.sync-strip{display:flex;gap:10px;align-items:center;padding:12px 0 0;color:var(--subtle);font-size:11px}.connection-pill{font-weight:680;color:var(--warn)}
.connection-pill:before{content:"";display:inline-block;width:7px;height:7px;border-radius:50%;background:currentColor;margin-right:7px}.connection-pill.online{color:var(--ok)}.connection-pill.offline{color:var(--bad)}
.overview{gap:22px;padding:22px 0}.overview-title{font-size:23px}.summary{grid-template-columns:repeat(4,minmax(0,1fr));min-width:min(520px,100%);border-radius:12px}.summary-item{padding:13px;min-width:0}.summary-item b{font-size:20px;font-variant-numeric:tabular-nums;overflow-wrap:anywhere}.summary-item span{font-size:10px}
.providers,.table-wrap,.quick-connect{border-radius:12px}.card[data-state="DEGRADED"]{box-shadow:inset 3px 0 var(--warn)}.card[data-state="DEAD"]{box-shadow:inset 3px 0 var(--bad)}.card[data-enabled="false"]{background:#17202a}
.provider-filter{display:flex;gap:7px;overflow:auto;scrollbar-width:none;margin:0 0 12px;padding:3px 0}.provider-filter button{white-space:nowrap;min-height:38px;background:transparent;color:var(--muted);border-color:var(--line)}.provider-filter button.selected{background:#30465c;border-color:#718faa;color:#f2f7ff;font-weight:700}
.empty-result{padding:25px;text-align:center;border:1px dashed var(--line);border-radius:12px;color:var(--muted);font-size:13px}.load-more-wrap{padding:11px;text-align:center}.load-more-wrap button[hidden]{display:none}
@media(max-width:960px){.overview{grid-template-columns:1fr}.summary{min-width:0;width:100%}}
@media(max-width:820px),(hover:none) and (pointer:coarse){main{padding-bottom:calc(116px + env(safe-area-inset-bottom))}.anchor-nav{position:fixed;top:auto;bottom:0;left:0;right:0;margin:0;padding:8px max(8px,env(safe-area-inset-left)) calc(8px + env(safe-area-inset-bottom)) max(8px,env(safe-area-inset-right));border-top:1px solid #4a596a;border-bottom:0;background:#1b242f;display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:3px;z-index:30;box-shadow:0 -4px 24px #0005}.anchor-nav button,.anchor-nav .nav-play{padding:9px 2px;min-height:46px;font-size:12px;min-width:0}.summary{grid-template-columns:repeat(2,minmax(0,1fr))}.summary-item:nth-child(3){border-left:0;border-top:1px solid var(--line)}.summary-item:nth-child(4){border-top:1px solid var(--line)}.card-top{grid-template-columns:1fr}.overview-title{font-size:21px}}
@media(max-width:390px){.anchor-nav button,.anchor-nav .nav-play{font-size:11px}}@media(prefers-reduced-motion:reduce){*,*::before,*::after{transition:none!important;animation:none!important}}
/* Desktop workspace: persistent sidebar and generous data density. */
@media(min-width:1024px){
 main{width:auto;max-width:1560px;margin:0 28px 0 258px;padding:26px 24px 76px}
 .anchor-nav{position:fixed;left:0;top:0;bottom:0;width:240px;margin:0;padding:100px 16px 80px;display:flex;flex-direction:column;gap:8px;overflow-y:auto;overflow-x:hidden;border:0;border-right:1px solid var(--line);border-radius:0;background:#141c26;box-shadow:none;z-index:30}
 .anchor-nav::before{content:'AhB  /  WORKSPACE';position:absolute;top:37px;left:26px;color:#b8d4f2;font-size:12px;font-weight:790;letter-spacing:.13em}
 .anchor-nav::after{content:'LOCALHOST  ·  PRIVATE';position:absolute;bottom:24px;left:26px;color:var(--subtle);font-size:10px;letter-spacing:.12em}
 .anchor-nav button,.anchor-nav .nav-play{display:flex;justify-content:flex-start;flex:0 0 auto;text-align:left;min-height:47px;width:100%;padding:12px 15px;border-radius:10px;font-size:14px}
 .anchor-nav button.selected{background:#32465c;border-color:#627e9c;box-shadow:inset 3px 0 #b6d4f4}
 .anchor-nav .nav-play{color:#c6dfff;background:#22364a;border-color:#364e67;margin-top:12px}
 .anchor-nav .nav-play:hover{background:#304761}
 header{padding-bottom:20px}.overview{grid-template-columns:minmax(320px,1fr) minmax(440px,.98fr)}
 .summary{width:100%;min-width:0}.summary-item{padding:17px 14px}.summary-item b{font-size:23px}
 .card-top{grid-template-columns:minmax(210px,1.6fr) minmax(160px,.8fr) minmax(260px,auto);gap:20px}
 .provider-actions{max-width:345px;flex-wrap:wrap;justify-content:flex-end}
 .provider-actions .btn{min-width:110px}.provider-name{font-size:15px}
 .quick-connect{grid-template-columns:minmax(250px,.85fr) minmax(380px,1.3fr);gap:32px;padding:25px}
 .table-wrap{max-height:560px;overflow:auto}th{z-index:1}
}
@media(min-width:1700px){main{margin-left:max(258px,calc((100vw - 1450px) / 2 + 90px));margin-right:auto;max-width:1450px}}
/* Stability presets are client-only until the user explicitly saves. */
.resource-presets{display:flex;gap:7px;flex-wrap:wrap;margin:11px 0}
.resource-presets button{min-height:39px;padding:8px 11px;font-size:11px;border-radius:9px;border:1px solid var(--line);background:var(--panel2);color:var(--text)}
.resource-presets button:hover{border-color:var(--accent)}
.resource-presets button:focus-visible{outline:2px solid var(--accent);outline-offset:2px}
</style>
</head>
<body>
<main>
<header>
 <div class="brand">
  <div class="mark">AhB</div>
  <div><h1>本機控制中心</h1><div id="runtime" class="kicker">正在讀取執行狀態…</div></div>
 </div>
 <div style="display:flex;gap:8px;align-items:center"><button id="updateAhB" type="button" disabled title="連 Agent2API 在內，從已發布 Android 封包安全更新">更新 AhB</button><button id="restartAhB" type="button" disabled title="只重新啟動 AhB，不會關閉 Termux">重新啟動 AhB</button><button id="refresh">重新整理</button></div>
</header>
<nav class="anchor-nav" aria-label="AhB 分頁" id="ahbTabs">
 <button type="button" data-tab="home" class="selected" aria-current="page">總覽</button>
 <a href="/playground" class="btn nav-play" aria-label="前往 Playground">實測</a>
 <button type="button" data-tab="accounts">帳號</button>
 <button type="button" data-tab="bridges">連接</button>
 <button type="button" data-tab="advanced">設定</button>
</nav>
<div class="sync-strip" role="status" aria-live="polite"><span class="connection-pill" id="connectionStatus">連線中</span><span id="lastSynced">讀取最新狀態…</span></div>
<div id="controlInfo" role="status" aria-live="polite" style="display:none;margin:12px 0;padding:12px;border:1px solid var(--line);border-radius:10px;background:var(--panel);font-size:12px"></div>

<div class="overview">
 <div>
  <div class="overview-title">統一 API 入口</div>
  <div class="desc">所有來源維持明確的 provider/model 命名；路由狀態可以在下方直接確認。</div>
  <div class="base-row"><div id="baseUrl" class="base">http://127.0.0.1:8317/v1</div><button id="copyBase">複製 Base URL</button></div>
 </div>
 <div class="summary">
  <div class="summary-item"><b id="providerTotal">—</b><span>已啟用來源</span></div>
  <div class="summary-item"><b id="modelTotal">—</b><span>可列出模型</span></div>
   <div class="summary-item"><b id="summaryReady">—</b><span>健康程序</span></div>
   <div class="summary-item"><b id="summaryRss">—</b><span>Hub + 子程序 RSS</span></div>
 </div>
</div>

<div id="error" class="notice"></div>
<div id="warnings" class="notice"></div>

<section id="resourceSettings">
 <div class="section-head"><div class="section-title"><h2>資源與常駐設定</h2><span class="count">Android / Linux 各自保留設定</span></div></div>
 <div class="quick-connect">
  <div>
   <div class="eyebrow">RAM monitor</div>
   <div class="quick-title" id="systemRamSummary">正在讀取整機 RAM…</div>
   <p id="ramSourceHint">Linux VPS 若有 cgroup 記憶體上限，優先顯示容器額度；否則顯示主機實體記憶體。AhB RSS 不等於整台機器的 RAM。</p>
   <div class="layers">
    <span class="layer">系統 RAM <b id="systemRamValue">—</b></span>
    <span class="layer">可用 RAM <b id="availableRamValue">—</b></span>
    <span class="layer">AhB + 轉接器 RSS <b id="ahbRamValue">—</b></span>
   </div>
   <div style="height:8px;border-radius:9px;background:var(--line);overflow:hidden;margin-top:12px"><div id="ramMeter" style="width:0%;height:100%;background:var(--accent);transition:width .2s"></div></div>
   <div class="command-help">上方「系統 RAM」包含其他程式；AhB RSS 為各程序估算相加，共用記憶體可能重複計算。Linux 512 MB 建議最多 1 個，Android 可設定較高。</div>
  </div>
  <div class="command-panel">
   <div class="eyebrow">On-demand sidecars</div>
   <div class="resource-presets" role="group" aria-label="快速資源設定"><button type="button" data-resource-preset="stable">穩定優先 · 2 個 / 10 分鐘</button><button type="button" data-resource-preset="lean">省資源 · 1 個 / 2 分鐘</button><button type="button" data-resource-preset="performance">多來源 · 3 個 / 15 分鐘</button><button type="button" data-resource-preset="all">全來源 · 9 個 / 24 小時</button></div>
   <div class="command-help">快速預設只填入建議值，按「儲存資源設定」才會生效。「全來源」允許九個 Gateway 同時運作，但不會自動啟動尚未使用的來源；你已在手機量到低占用，仍建議觀察實際推論峰值。Linux 小 VPS 可保留省資源模式。</div>
   <div class="field"><label for="maxResident">同時最多常駐幾個轉接器？</label><input id="maxResident" type="number" min="1" max="16" step="1" inputmode="numeric" value="1"></div>
   <div class="field"><label for="idleTimeout">閒置多久自動關閉？（秒）</label><input id="idleTimeout" type="number" min="30" max="86400" step="1" inputmode="numeric" value="120"></div>
   <div class="command-actions"><button type="button" id="saveResourceSettings" disabled>儲存資源設定</button></div>
   <div class="command-help">儲存後只重啟 AhB 自己的服務；不會刪除帳號或登入。正在進行的推理請求可能中斷。此數量不包含 AhB 主程式。若只有某個 Gateway 發生 503，先檢查登入與額度，再用首頁的「單獨重啟」，不必重啟整個 AhB。</div>
  </div>
 </div>
</section>

<section id="diagnostics">
  <div class="section-head"><div class="section-title"><h2>全來源狀態檢查</h2><span class="count">No inference / no secrets</span></div></div>
  <div class="quick-connect">
    <div>
      <div class="eyebrow">Built-in providers</div>
      <div class="quick-title">先看程序、帳號，再挑模型做實測。</div>
      <p>直接用首頁的平台開關啟用或停用來源。關閉後 AhB 會正常停止該服務，節省手機資源；啟用不代表帳號已登入或有額度。</p>
    </div>
    <div class="command-panel">
      <div class="eyebrow">Termux diagnostic</div>
      <div class="command-box">cd ~/AhB &amp;&amp; ./scripts/provider-status-termux.sh</div>
      <div class="command-help">此指令只讀取本機服務狀態與模型數量，顯示帳號筆數、最後 HTTP 狀態，不發送推理請求，也不輸出憑證；HEALTHY 與 200 不能證明有額度或能完成工具呼叫。</div>
    </div>
  </div>
</section>

<section id="freebuffLogin">
  <div class="section-head"><div class="section-title"><h2>FreeBuff 登入</h2><span class="count">Android Node.js CLI / Bearer</span></div></div>
  <div class="quick-connect">
    <div>
      <div class="eyebrow">FreeBuff account onboarding</div>
      <div class="quick-title">直接用手機瀏覽器完成官方 OAuth 授權。</div>
      <p>舊 Rust 版的 Web Cookie 不能直接匯入新版；原本的帳號資料會保留在備份和 data/freebuff/。新版以自己的 CLI/Bearer 帳號為來源，無額外管理網頁。</p>
      <div class="quick-actions">
        <a class="btn" href="https://github.com/yutian81/freebuff2api" target="_blank" rel="noopener noreferrer">新版原始碼與說明</a>
      </div>
    </div>
    <div class="command-panel">
      <div class="eyebrow">官方 Google / CLI OAuth 授權</div>
      <div class="quick-title">直接在 AhB 登入，不需要 Termux 指令</div>
      <p id="freebuffAccountCount">正在查詢本機帳號數…</p>
      <div class="command-actions">
       <button type="button" class="btn primary" id="startFreebuffLogin" disabled>以 Google 登入 FreeBuff</button>
       <button type="button" class="btn" id="cancelFreebuffLogin" style="display:none">取消登入</button>
      </div>
      <div id="freebuffLoginFeedback" class="quick-note" role="status" aria-live="polite" style="display:none;margin-top:10px">
       <span id="freebuffLoginMessage">正在檢查授權狀態…</span>
       <div id="freebuffLoginLinkArea" style="display:none;margin-top:12px">
        <a id="freebuffLoginLink" class="btn primary" target="_blank" rel="noopener noreferrer" href="#">開啟 FreeBuff 官方授權頁</a>
        <p class="command-help">這是一次性登入網址，請只在自己的手機開啟，不要分享截圖或網址。授權完成後，AhB 會自動安全保存到本機。</p>
       </div>
       <div style="margin-top:10px"><button type="button" class="btn" id="reloadFreebuff" style="display:none">重新載入 FreeBuff 帳號</button></div>
      </div>
      <details style="margin-top:14px"><summary class="muted" style="cursor:pointer">備用：使用 Termux 登入</summary>
       <div class="command-box" style="margin-top:7px">cd ~/AhB &amp;&amp; ./scripts/freebuff-login-termux.sh</div>
       <div class="command-help">只有本機 OAuth 按鈕無法使用時才需要。不要把授權連結或 Token 傳給其他人。</div>
      </details>
    </div>
  </div>
</section>

<section id="adminConsoles">
  <div class="section-head"><div class="section-title"><h2>原生控制台登入</h2><span class="count">本機密碼集中管理</span></div></div>
  <div class="quick-connect">
   <div>
    <div class="eyebrow">Native admin access</div>
    <div class="quick-title">不用再去 Termux 找每一個管理密碼</div>
    <p>進入各 Gateway 原本的管理頁面前，點「複製管理密碼」即可取得該來源目前的本機設定。只有你主動按下複製時才讀取；不修改管理員、上游 OAuth、帳號資料或 API 額度。</p>
    <p>這不是替上游強制登入的通行證。如果上游有自己的登入流程（例如 Agent2API），仍使用它原本的驗證。</p>
   </div>
   <div class="command-panel">
    <div class="eyebrow">Local credential vault</div>
    <div id="adminConsoleNotice" role="status" aria-live="polite" class="command-help">正在確認本機管理資料…</div>
    <button type="button" class="btn" id="reloadAdminConsoles" disabled>重新檢查本機登入資料</button>
   </div>
  </div>
  <div class="providers" id="adminConsoleRows" style="margin-top:10px">
   <div class="console-row muted">正在載入控制台清單…</div>
  </div>
  <p class="command-help">密碼複製後會暫時留在 Android 剪貼簿，請不要轉貼到聊天或截圖中。Grok 初始密碼在上游修改後可能不再有效。所有管理頁預設只在手機 localhost 開啟。</p>
 </section>

<section id="optional">
 <div class="section-head"><div class="section-title"><h2>選裝來源</h2><span class="count">Install locally, opt in</span></div></div>
 <div class="quick-connect">
  <div>
   <div class="eyebrow">Optional packages</div>
   <div class="quick-title">有帳號才啟用，減少手機負擔。</div>
   <p>安裝包包含九個獨立 Gateway；其他尚未整合的來源不會冒充為已安裝。Agent2API 的 Android 更新由 AhB 統一封包管理：在首頁按「更新 AhB」即可安全檢查並更新整套服務。</p>
   <p>Copilot 可以在這裡完成官方裝置授權；未安裝的 Kimi Web 不再列入預設來源。</p>
  </div>
  <div class="quick-form">
   <div class="command-panel">
    <div class="field"><label>GitHub Copilot · 本機裝置授權</label></div>
    <p>Copilot2API 沒有獨立 WebUI。先在這裡取得 GitHub 官方一次性授權碼並登入，再回首頁啟動 Gateway。已有 GitHub 授權資料不代表一定有 Copilot 額度。</p>
    <div class="command-actions"><button type="button" id="startCopilotLogin" class="btn primary">登入 GitHub Copilot</button><button type="button" id="copyCopilotInstall">複製備用指令</button></div>
    <div id="copilotLoginFeedback" class="quick-note" role="status" aria-live="polite" style="display:none;margin-top:10px">
     <span id="copilotLoginMessage">正在檢查登入狀態…</span>
     <div id="copilotLoginDevice" style="display:none;margin-top:8px">
      <a href="https://github.com/login/device" target="_blank" rel="noopener noreferrer" class="btn primary">開啟 GitHub 官方授權頁</a>
      <div style="margin-top:9px">一次性授權碼：<code id="copilotDeviceCode" style="font-size:19px;font-weight:700;user-select:all"></code></div>
      <p>只輸入在 GitHub 官方網站，不要傳給他人。</p>
     </div>
     <button type="button" id="enableCopilotAfterLogin" class="btn primary" style="display:none">啟用 Copilot 服務</button>
    </div>
    <div class="command-box" id="copilotInstallCommand" style="display:none">cd ~/AhB && ./scripts/login-copilot2api.sh</div>
    <div class="command-help">授權資料只存放在手機的私密帳號目錄，AhB 不會在網頁顯示永久金鑰。已有帳號時可直接回總覽啟用。</div>
   </div>

  </div>
 </div>
</section>
<section id="connect">
 <div class="section-head"><div class="section-title"><h2>連接外部 API</h2><span class="count">Local bridges</span></div></div>
 <div class="quick-connect">
  <div>
   <div class="eyebrow">Connect</div>
   <div class="quick-title">接入新的 API，不用改程式。</div>
   <p>選擇已在本機啟動的相容 API，填入連接埠，複製一條指令至 Termux 即可套用。既有帳號與原始管理介面不受影響。</p>
   <p>這是手動接入<strong>已另外安裝</strong>的本機服務，不代表 AhB 已內建它。沒有外部服務時不用設定。</p>
  </div>
  <div class="quick-form">
   <div class="fieldrow">
    <div class="field"><label for="bridgePreset">來源</label><select id="bridgePreset">
      <option value="cliproxy">CLIProxyAPI · Antigravity / Codex / Claude / Muse（需自行安裝）</option>
      <option value="custom">自訂已安裝的本機 API</option>
     </select></div>
    <div class="field"><label for="bridgeURL">本機 API 地址</label><input id="bridgeURL" type="url" inputmode="url" spellcheck="false" value="http://127.0.0.1:8418" placeholder="http://127.0.0.1:8418"></div>
   </div>
   <div class="field" id="customIdField" style="display:none"><label for="customBridgeId">來源代號（小寫英文）</label><input id="customBridgeId" value="mybridge" maxlength="30" spellcheck="false"></div>
   <div class="command-panel">
    <div id="bridgeCommand" class="command-box" aria-live="polite"></div>
    <div class="command-actions">
     <button class="btn primary" id="copyBridge" type="button">複製連接指令</button>
     <a class="btn" href="https://github.com/Tsenjii/AhB/blob/main/docs/BRIDGES.md" target="_blank" rel="noopener noreferrer">完整使用說明 ↗</a>
    </div>
    <div id="bridgeHint" class="command-help">前提：外部橋接服務已由你在相同手機上啟動並開放此連接埠。</div>
   </div>
  </div>
 </div>
</section>

<section id="providersSection">
 <div class="section-head"><div class="section-title"><h2>Gateway 服務</h2><span id="providerReady" class="count">—</span></div></div>
 <div class="provider-filter" role="group" aria-label="篩選 Gateway"><button type="button" class="selected" data-provider-filter="all" aria-pressed="true">全部</button><button type="button" data-provider-filter="enabled" aria-pressed="false">已啟用</button><button type="button" data-provider-filter="problems" aria-pressed="false">需注意</button><button type="button" data-provider-filter="asleep" aria-pressed="false">休眠</button></div>
 <div id="providers" class="providers"></div>
 <div id="providerFilterEmpty" class="empty-result" hidden>沒有符合此篩選的來源。</div>
</section>

<section id="modelsSection">
 <div class="section-head">
  <div><div class="section-title"><h2>模型清單</h2><span id="modelCount" class="count">—</span></div><div class="desc">已啟動來源顯示即時清單；休眠來源可顯示六小時內曾成功取得的模型快取（並標示「休眠快取」）。快取不保證帳號有效、模型未下架或有剩餘額度。可依序掃描全部來源而不常駐九個程序。</div></div>
  <div class="command-actions"><button type="button" id="scanAllModels" class="btn" disabled>逐一載入全部模型</button><input id="search" class="search" type="search" aria-label="搜尋模型或來源" placeholder="搜尋模型或來源…"></div>
 </div>
 <div class="table-wrap">
  <table><thead><tr><th>Model ID</th><th>Provider</th><th>Upstream</th></tr></thead><tbody id="models"></tbody></table><div class="load-more-wrap"><button type="button" id="showMoreModels" hidden>顯示更多模型</button></div>
 </div>
</section>

<div class="footer">Account 顯示的是帳號憑證或上游回報狀態，並非可用餘額；如 Agent2API 可能顯示 2/2 但實際回 503。Last API 只顯示本次啟動後最近一次上游 HTTP 狀態，不含對話或金鑰；HTTP 200 亦不保證完整串流或下一次有額度。Quota 請以來源管理介面或真實請求為準。</div>
</main>
<script>
// Ephemeral, loopback-only control. Never save this token to storage.
const ahbControlToken="__AHB_CONTROL_TOKEN__";
const ahbRestartAvailable=__AHB_RESTART_AVAILABLE__;
const ahbProviderControlAvailable=__AHB_PROVIDER_CONTROL_AVAILABLE__;
const ahbUpdateAvailable=__AHB_UPDATE_AVAILABLE__;
// The official FreeBuff CLI OAuth link is one-time data. Only the link (never
// a Bearer token) is sent to the browser; private credential files stay on
// the Android device and are never rendered or logged by this dashboard.
// Local-only read-on-click access to original upstream console passwords.
// No credential appears in the HTML source or the status inventory.
const consoleRows=document.getElementById('adminConsoleRows');
const consoleStatus=document.getElementById('adminConsoleNotice');
const consoleRefreshButton=document.getElementById('reloadAdminConsoles');
consoleRefreshButton.disabled=!ahbProviderControlAvailable;
async function nativeConsoleRequest(action,provider){
 const payload={action};if(provider)payload.provider=provider;
 const res=await fetch('/api/control/console-access',{
  method:'POST',credentials:'same-origin',cache:'no-store',
  headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},
  body:JSON.stringify(payload)
 });
 if(!res.ok)throw new Error('HTTP '+res.status+': '+(await res.text()).trim().slice(0,120));
 return res.json();
}
function nativeConsoleURL(raw){
 try{
  const u=new URL(raw);
  return u.protocol==='http:'&&u.hostname==='127.0.0.1'?u.href:'';
 }catch(_){return ''}
}
async function loadNativeConsoles(){
 consoleStatus.textContent='正在確認每個 Gateway 的本機管理方式…';
 consoleRows.replaceChildren();
 try{
  const doc=await nativeConsoleRequest('list');
  for(const item of doc.consoles||[]){
   const row=document.createElement('div');row.className='console-row';
   const detail=document.createElement('div');detail.className='console-detail';
   const title=document.createElement('div');title.className='console-title';title.textContent=item.name;
   const helper=document.createElement('div');helper.className='console-help';
   const mode=item.mode==='username_password'||item.mode==='bootstrap_password'?'帳號／密碼':
    (item.mode==='admin_password'?'管理密碼':(item.mode==='admin_token'?'管理 Token':'原生驗證'));
   helper.textContent=mode+(item.username?' · 帳號 '+item.username:'')+' · '+(item.help||'');
   detail.append(title,helper);
   const actions=document.createElement('div');actions.className='console-actions';
   const safeURL=nativeConsoleURL(item.url);
   const visit=document.createElement('a');visit.className='btn';visit.textContent='開啟原生 UI';
   // Docker/remote clients have their *own* 127.0.0.1, not the container's
   // private Gateway ports. Never link to a misleading or unrelated service.
   if(safeURL&&ahbProviderControlAvailable){
    visit.href=safeURL;visit.target='_blank';visit.rel='noopener noreferrer';
    // Never open an empty new tab while the phone is waiting for a sleeping
    // Gateway to wake: Android may suspend the original tab and never
    // navigate the blank page. Running providers use the real user tap as a
    // normal link; sleeping providers first wake then offer a direct tap.
    visit.addEventListener('click',async event=>{
     const cached=lastProviders.find(p=>p.id===item.id);
     if(visit.dataset.ready==='yes'||(cached&&cached.enabled&&cached.process_alive))return;
     event.preventDefault();
     if(visit.dataset.waking==='yes')return;
     visit.dataset.waking='yes';
     visit.textContent='正在啟動…';
     try{
      const status=await getJSON('/api/providers');
      const provider=(status.providers||[]).find(p=>p.id===item.id);
      if(!provider||!provider.enabled)throw new Error('這個來源尚未啟用，請先回總覽開啟。');
      if(!provider.process_alive){
       if(provider.start_mode!=='on_demand')throw new Error('尚未運作，請先到總覽檢查服務狀態。');
       consoleStatus.textContent=item.name+' 休眠中，正在按需啟動…';
       const started=await fetch('/api/control/wake/'+encodeURIComponent(item.id),{
        method:'POST',credentials:'same-origin',cache:'no-store',
        headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},body:'{}'
       });
       if(!started.ok)throw new Error('無法啟動 Gateway：'+(await started.text()).trim().slice(0,130));
      }
      visit.dataset.ready='yes';
      visit.textContent='已啟動，點此開啟';
      consoleStatus.textContent=item.name+' 原生 UI 已準備好；請再次點擊按鈕開啟本機管理頁。';
      await refresh();
     }catch(err){
      visit.textContent='開啟原生 UI';
      consoleStatus.textContent=item.name+'：'+err.message;
     }finally{delete visit.dataset.waking}
    });
   }
   else{visit.classList.add('disabled');visit.removeAttribute('href')}
   actions.appendChild(visit);
   if(item.mode!=='native'){
    const copy=document.createElement('button');copy.className='btn';
    copy.type='button';copy.textContent=item.local_available?'複製管理密碼':'尚未設定';
    copy.disabled=!ahbProviderControlAvailable||!item.local_available;
    copy.addEventListener('click',async()=>{
     copy.disabled=true;copy.textContent='複製中…';
     try{
      const data=await nativeConsoleRequest('copy',item.id);
      if(!data.secret||!await copyText(data.secret))throw new Error('複製失敗，請確認瀏覽器允許剪貼簿');
      consoleStatus.textContent=item.name+' 管理憑證已複製到剪貼簿。請只貼入該來源的本機登入頁；不要貼到聊天。';
     }catch(err){consoleStatus.textContent=item.name+'：'+err.message}
     finally{copy.disabled=false;copy.textContent='複製管理密碼'}
    });
    actions.appendChild(copy);
   }
   row.append(detail,actions);consoleRows.appendChild(row);
  }
  consoleStatus.textContent='已載入原生控制台入口。只有主動按下「複製管理密碼」才會讀取憑證。';
 }catch(err){
  const notice=document.createElement('div');notice.className='console-row muted';
  notice.textContent='本機控制台狀態無法取得：'+err.message;
  consoleRows.appendChild(notice);
  consoleStatus.textContent='只有使用本機 AhB Dashboard 才能讀取管理資料。';
 }
}
consoleRefreshButton.addEventListener('click',loadNativeConsoles);
if(ahbProviderControlAvailable)loadNativeConsoles();
else consoleStatus.textContent='只有 localhost 受控 Dashboard 才能查看原生管理憑證。';

const freebuffStart=document.getElementById('startFreebuffLogin');
const freebuffCancel=document.getElementById('cancelFreebuffLogin');
const freebuffFeedback=document.getElementById('freebuffLoginFeedback');
const freebuffMessage=document.getElementById('freebuffLoginMessage');
const freebuffCount=document.getElementById('freebuffAccountCount');
const freebuffLinkArea=document.getElementById('freebuffLoginLinkArea');
const freebuffLink=document.getElementById('freebuffLoginLink');
const freebuffReload=document.getElementById('reloadFreebuff');
let freebuffTimer=null;
freebuffStart.disabled=!ahbProviderControlAvailable;
function freebuffOfficialURL(raw){
 try {
  const u=new URL(raw);
  return u.protocol==='https:'&&(u.hostname==='www.codebuff.com'||u.hostname==='codebuff.com')?u.href:'';
 }catch(_){return ''}
}
async function freebuffLoginAction(action){
 // Localhost requests should finish immediately; a browser that suspends
 // an old tab must never leave an unbounded spinner or a disabled button.
 const controller=new AbortController();
 const timeout=setTimeout(()=>controller.abort(),15000);
 try{
  const res=await fetch('/api/control/login/freebuff',{
   method:'POST',credentials:'same-origin',cache:'no-store',signal:controller.signal,
   headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},
   body:JSON.stringify({action})
  });
  if(!res.ok)throw new Error((await res.text()).trim().slice(0,140)||'HTTP '+res.status);
  return res.json();
 }catch(err){
  if(err.name==='AbortError')throw new Error('本機 AhB 回應逾時，請確認服務是否仍在執行');
  throw err;
 }finally{clearTimeout(timeout)}
}
function showFreebuffLogin(data){
 const state=data.state||'idle';
 if(freebuffTimer){clearTimeout(freebuffTimer);freebuffTimer=null}
 const count=Number.isInteger(data.accounts)?data.accounts:0;
 freebuffCount.textContent='本機已保存 '+count+' 個 CLI 帳號 · 不代表有可用額度';
 freebuffFeedback.style.display=state==='idle'?'none':'block';
 freebuffMessage.textContent=data.detail||({
  starting:'正在取得官方授權網址，請稍候…',
  waiting:'授權網址已準備好。請按下方「開啟 FreeBuff 官方授權頁」完成登入；AhB 會自動確認。',
  done:'已完成授權並安全保存到手機。',
  expired:'授權逾時，請重新開始。',
  cancelled:'授權已取消。',
  failed:'授權失敗，請稍後重試。'
 }[state]||'尚未開始');
 const url=state==='waiting'?freebuffOfficialURL(data.url||''):'';
 freebuffLinkArea.style.display=url?'block':'none';
 if(url){freebuffLink.href=url}
 else{freebuffLink.removeAttribute('href')}
 // Never create an empty popup. On Android, Chrome often switches focus to
 // that blank tab and freezes the original page before OAuth polling yields
 // the real official URL. A direct user-tapped HTTPS link is more reliable.
 freebuffCancel.style.display=(state==='starting'||state==='waiting')?'inline-flex':'none';
 freebuffReload.style.display=state==='done'?'inline-flex':'none';
 freebuffStart.disabled=!ahbProviderControlAvailable||state==='starting'||state==='waiting';
 freebuffStart.textContent=state==='starting'||state==='waiting'?'授權進行中…':(count?'新增另一個 Google 帳號':'以 Google 登入 FreeBuff');
 if(state==='starting'||state==='waiting'){
  freebuffTimer=setTimeout(async()=>{
   try{showFreebuffLogin(await freebuffLoginAction('status'))}
   catch(err){
    freebuffMessage.textContent='無法確認授權：'+err.message+'。可以重新整理頁面再查看狀態。';
    freebuffStart.disabled=false;
    // Keep the official link usable even if one status poll fails.
    // Do not start a fresh authorization without user action.
    freebuffTimer=setTimeout(async()=>{
     try{showFreebuffLogin(await freebuffLoginAction('status'))}
     catch(_){freebuffStart.disabled=false}
    },4000)
   }
  },1500);
 }
}
freebuffStart.addEventListener('click',async()=>{
 if(!ahbProviderControlAvailable)return;
 freebuffStart.disabled=true;
 // Step 1 obtains the official authorization URL. Step 2 is a direct
 // user-initiated HTTPS link; never open a blank tab during an async fetch.
 freebuffFeedback.style.display='block';
 freebuffMessage.textContent='正在向官方取得一次性登入網址…';
 try{showFreebuffLogin(await freebuffLoginAction('start'))}
 catch(err){
  freebuffStart.disabled=false;
  freebuffMessage.textContent='無法開始 FreeBuff 登入：'+err.message;
 }
});
freebuffCancel.addEventListener('click',async()=>{
 try{showFreebuffLogin(await freebuffLoginAction('cancel'))}
 catch(err){freebuffMessage.textContent='取消授權失敗：'+err.message}
});
freebuffReload.addEventListener('click',async()=>{
 freebuffReload.disabled=true;
 try{
  const data=await getJSON('/api/providers');
  const p=(data.providers||[]).find(x=>x.id==='freebuff');
  if(!p||!p.enabled){controlNotice('FreeBuff 尚未啟用；先到總覽開啟。');return}
  const endpoint=p.process_alive?'/api/control/recover/freebuff':'/api/control/wake/freebuff';
  const res=await fetch(endpoint,{
   method:'POST',credentials:'same-origin',cache:'no-store',
   headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},body:'{}'
  });
  if(!res.ok)throw new Error((await res.text()).trim().slice(0,160)||'HTTP '+res.status);
  controlNotice('已重新載入 FreeBuff。請透過模型清單和實際推論確認帳號／額度。');
  await refresh();
 }catch(err){controlNotice('FreeBuff 尚未完成重新載入：'+err.message)}
 finally{freebuffReload.disabled=false}
});
freebuffLoginAction('status').then(showFreebuffLogin).catch(()=>{freebuffCount.textContent='本機登入狀態目前無法取得';});

const startCopilotLoginBtn=document.getElementById('startCopilotLogin');
const copilotLoginFeedback=document.getElementById('copilotLoginFeedback');
const copilotLoginMessage=document.getElementById('copilotLoginMessage');
const copilotLoginDevice=document.getElementById('copilotLoginDevice');
const copilotDeviceCode=document.getElementById('copilotDeviceCode');
const enableCopilotAfterLogin=document.getElementById('enableCopilotAfterLogin');
let copilotLoginTimer=null;
startCopilotLoginBtn.disabled=!ahbProviderControlAvailable;
enableCopilotAfterLogin.addEventListener('click',()=>{
 switchTab('home');
 toggleProvider('copilot',true);
});
async function copilotLoginAction(action){
 const r=await fetch('/api/control/login/copilot',{
  method:'POST',cache:'no-store',credentials:'same-origin',
  headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},
  body:JSON.stringify({action})
 });
 if(!r.ok)throw new Error('HTTP '+r.status+': '+(await r.text()).trim().slice(0,120));
 return await r.json();
}
function showCopilotLogin(data){
 copilotLoginFeedback.style.display='block';
 const state=data.state||'idle';
 const text={idle:'尚未啟動授權',starting:'正在向 GitHub 要求裝置授權…',waiting:'請打開 GitHub 並輸入授權碼，授權後會自動完成。',done:'本機已有 GitHub 授權資料；啟用後需用 Playground 驗證可用額度。',failed:'登入未成功，請確認帳號或網路後再試。'}[state]||'正在登入…';
 copilotLoginMessage.textContent=data.detail||text;
 copilotLoginDevice.style.display=(state==='waiting'&&data.user_code)?'block':'none';
 copilotDeviceCode.textContent=state==='waiting'?(data.user_code||''):'';
 enableCopilotAfterLogin.style.display=state==='done'?'inline-flex':'none';
 startCopilotLoginBtn.disabled=!ahbProviderControlAvailable || state==='starting'||state==='waiting';
 startCopilotLoginBtn.textContent=state==='waiting'?'授權進行中…':(state==='done'?'已完成登入':'登入 GitHub Copilot');
 if(copilotLoginTimer){clearTimeout(copilotLoginTimer);copilotLoginTimer=null}
 if(state==='waiting'||state==='starting'){
  copilotLoginTimer=setTimeout(async()=>{
   try{showCopilotLogin(await copilotLoginAction('status'))}
   catch(e){copilotLoginMessage.textContent='無法取得登入狀態：'+e.message;startCopilotLoginBtn.disabled=false}
  },1800);
 }
}
// Restore a pending GitHub device code after Android browser refresh.
 if(ahbProviderControlAvailable)copilotLoginAction('status').then(showCopilotLogin).catch(()=>{});
 startCopilotLoginBtn.addEventListener('click',async()=>{
 startCopilotLoginBtn.disabled=true;
 copilotLoginFeedback.style.display='block';
 copilotLoginMessage.textContent='正在啟動 GitHub 授權…';
 try{showCopilotLogin(await copilotLoginAction('start'))}
 catch(e){
  copilotLoginMessage.textContent='未能開始授權：'+e.message;
  startCopilotLoginBtn.disabled=false;
 }
});

const restartAhBButton=document.getElementById('restartAhB');
restartAhBButton.disabled=!ahbRestartAvailable;
restartAhBButton.title=ahbRestartAvailable?'重新啟動 AhB 後端及其受控服務，不會關閉 Termux':'僅限本機 Android Termux 預編譯版提供此操作';
restartAhBButton.addEventListener('click',async()=>{
 if(!ahbRestartAvailable||restartAhBButton.disabled)return;
 if(!confirm('要重新啟動 AhB 嗎？進行中的 API 請求會暫時中斷，但帳號和設定會保留。'))return;
 restartAhBButton.disabled=true;
 restartAhBButton.textContent='準備重新啟動…';
 try{
  const r=await fetch('/api/control/restart',{
   method:'POST',cache:'no-store',credentials:'same-origin',
   headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},
   body:'{}'
  });
  if(!r.ok)throw new Error('排程失敗 (HTTP '+r.status+')');
  restartAhBButton.textContent='正在重新啟動…';
  // The previous HTTP connection disappears while hubd gracefully exits.
  // Loading the page again verifies the new daemon rather than guessing.
  setTimeout(()=>location.reload(),9500);
 }catch(e){
  restartAhBButton.disabled=false;
  restartAhBButton.textContent='重新啟動 AhB';
  alert(e.message+'。原本的 AhB 不會被強制停止。');
 }
});
const updateAhBButton=document.getElementById('updateAhB');
updateAhBButton.disabled=!ahbUpdateAvailable;
updateAhBButton.addEventListener('click',async()=>{
 if(!ahbUpdateAvailable||updateAhBButton.disabled)return;
 if(!confirm('檢查 GitHub 已發布的 AhB Android 封包，有新版才更新。這會連 Agent2API 等內建服務一起更新，舊帳號與資料保留，會中斷目前請求。確定嗎？'))return;
 updateAhBButton.disabled=true;
 updateAhBButton.textContent='正在檢查…';
 try{
  const r=await fetch('/api/control/update',{
   method:'POST',cache:'no-store',credentials:'same-origin',
   headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},body:'{}'
  });
  if(!r.ok)throw new Error('HTTP '+r.status);
  controlNotice('已開始檢查更新。若有新版，系統會驗證封包、備份既有資料、更新後重新啟動 AhB；手機 Termux 不會被關閉。');
  updateAhBButton.textContent='正在檢查更新…';
 }catch(e){
  updateAhBButton.disabled=false;updateAhBButton.textContent='更新 AhB';
  controlNotice('無法啟動更新：'+e.message);
 }
});
// The dashboard starts with only the essential service cards and model list.
const tabSections={
 home:['providersSection','modelsSection'],
 accounts:['adminConsoles','freebuffLogin','optional'],
 bridges:['connect'],
 advanced:['resourceSettings','diagnostics']
};
function switchTab(name){
 const active=tabSections[name]||tabSections.home;
 for(const id of ['providersSection','modelsSection','adminConsoles','freebuffLogin','optional','connect','resourceSettings','diagnostics']){
  const e=document.getElementById(id);
  if(e)e.classList.toggle('ahb-tab-hidden',!active.includes(id));
 }
 document.querySelectorAll('#ahbTabs button').forEach(b=>{
  const selected=b.dataset.tab===name;
  b.classList.toggle('selected',selected);
  if(selected)b.setAttribute('aria-current','page');else b.removeAttribute('aria-current');
 });
}
document.getElementById('ahbTabs').addEventListener('click',e=>{
 const b=e.target.closest('button[data-tab]');
 if(b){switchTab(b.dataset.tab);if(b.dataset.tab==='home')refresh()}
});
function controlNotice(message){
 const el=document.getElementById('controlInfo');
 el.textContent=message;el.style.display='block';
 window.scrollTo({top:0,behavior:'smooth'});
}
async function toggleProvider(id,enabled){
 if(!ahbProviderControlAvailable)return;
 if(!confirm((enabled?'啟用 ':'停用 ')+id+'？AhB 將重新啟動自己的服務，目前進行中的 API 請求會中斷。')){refresh();return}
 controlNotice('正在儲存 '+id+' 設定…');
 try{
  const r=await fetch('/api/control/provider/'+encodeURIComponent(id),{
   method:'POST',cache:'no-store',credentials:'same-origin',
   headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},
   body:JSON.stringify({enabled})
  });
  if(!r.ok){const error=await r.text();throw new Error(error.trim().slice(0,180)||'HTTP '+r.status)}
  const data=await r.json();
  if(data.status==='unchanged'){controlNotice('設定沒有變更');await refresh();return}
  controlNotice('已儲存 '+id+' 設定，正在重啟 AhB。Termux 不會關閉，帳號資料會保留。');
  setTimeout(()=>location.reload(),8500);
 }catch(e){controlNotice('未完成：'+e.message);refresh()}
}
document.getElementById('providers').addEventListener('change',e=>{
 const input=e.target.closest('input[data-provider-toggle]');
 if(input)toggleProvider(input.dataset.providerToggle,input.checked);
});
document.getElementById('providers').addEventListener('click',async e=>{
 const copilotAuth=e.target.closest('button[data-copilot-auth]');
 if(copilotAuth){
  switchTab('accounts');
  startCopilotLoginBtn.focus();
  startCopilotLoginBtn.scrollIntoView({behavior:'smooth',block:'center'});
  return;
 }
 const recover=e.target.closest('button[data-provider-recover]');
 if(recover){
  const id=recover.dataset.providerRecover;
  if(!confirm('只重新啟動 '+id+'？進行中的推論不會被中斷。503 可能是額度或登入問題，重啟不保證修好。'))return;
  recover.disabled=true;recover.textContent='重啟中…';
  try{
   const res=await fetch('/api/control/recover/'+encodeURIComponent(id),{
    method:'POST',credentials:'same-origin',cache:'no-store',
    headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},body:'{}'
   });
   if(!res.ok)throw new Error((await res.text()).trim().slice(0,170)||'HTTP '+res.status);
   const doc=await res.json();
   controlNotice(id+' 已重啟'+(doc.ready?'，健康檢查通過。':'，但帳號或健康狀態仍待驗證。')+' 不會自動重送之前的請求。');
  }catch(error){controlNotice(id+' 無法安全重啟：'+error.message)}
  await refresh();return;
 }
 const proxyButton=e.target.closest('button[data-provider-proxy]');
 if(proxyButton){
  const id=proxyButton.dataset.providerProxy;
  const area=proxyButton.closest('details');
  const input=area&&area.querySelector('input[data-provider-proxy-url]');
  if(!input)return;
  const proxy=input.value.trim();
  if(proxy&&!(proxy.startsWith('http://')||proxy.startsWith('https://')||proxy.startsWith('socks5://'))){
   controlNotice('Proxy URL 只接受 http://、https:// 或 socks5://，且不能包含帳密。');return
  }
  if(!confirm((proxy?'設定':'清除')+' '+id+' 的程序出站 Proxy？這會重啟整個 AhB，請先結束推論。帳號級 Proxy 應優先使用來源原生介面。'))return;
  proxyButton.disabled=true;
  try{
   const res=await fetch('/api/control/proxy/'+encodeURIComponent(id),{
    method:'POST',credentials:'same-origin',cache:'no-store',
    headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},
    body:JSON.stringify({proxy_url:proxy})
   });
   if(!res.ok)throw new Error((await res.text()).trim().slice(0,160)||'HTTP '+res.status);
   const doc=await res.json();
   if(doc.status==='unchanged'){controlNotice('出站 Proxy 設定沒有變更。');await refresh();return}
   controlNotice(id+' Proxy 政策已儲存，正在重新啟動 AhB。');
   setTimeout(()=>location.reload(),8500);
  }catch(err){proxyButton.disabled=false;controlNotice('儲存 Proxy 失敗：'+err.message)}
  return;
 }
 const button=e.target.closest('button[data-provider-wake]');
 if(!button)return;
 const id=button.dataset.providerWake;
 button.disabled=true;button.textContent='正在啟動…';
 try{
  const r=await fetch('/api/control/wake/'+encodeURIComponent(id),{
   method:'POST',credentials:'same-origin',cache:'no-store',
   headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},body:'{}'
  });
  if(!r.ok)throw new Error((await r.text()).trim().slice(0,160)||'HTTP '+r.status);
  controlNotice(id+' 已啟動；閒置後會自動停止，登入資格與免費額度需由實際請求驗證。');
  await refresh();
 }catch(err){controlNotice('無法啟動 '+id+'：'+err.message);await refresh()}
});
let resourceSettingsDirty=false;
document.querySelector(".resource-presets").addEventListener("click",e=>{
 const preset=e.target.closest("button[data-resource-preset]");if(!preset)return;
 const values={stable:[2,600],lean:[1,120],performance:[3,900],all:[9,86400]}[preset.dataset.resourcePreset];if(!values)return;
 document.getElementById("maxResident").value=values[0];document.getElementById("idleTimeout").value=values[1];
 resourceSettingsDirty=true;
 controlNotice("已套用資源預設（尚未儲存）。請確認 RAM 需求，再按『儲存資源設定』。");
});
document.getElementById('maxResident').addEventListener('input',()=>{resourceSettingsDirty=true});
document.getElementById('idleTimeout').addEventListener('input',()=>{resourceSettingsDirty=true});
document.getElementById('saveResourceSettings').addEventListener('click',async()=>{
 if(!ahbProviderControlAvailable)return;
 const max=Number(document.getElementById('maxResident').value);
 const idle=Number(document.getElementById('idleTimeout').value);
 if(!Number.isInteger(max)||max<1||max>16||!Number.isInteger(idle)||idle<30||idle>86400) {
  controlNotice('資源設定超出範圍：常駐 1～16 個；閒置 30～86400 秒。');return;
 }
 if(!confirm('儲存新常駐上限和閒置時間，並重新啟動 AhB？目前進行中的 API 請求會中斷。'))return;
 const button=document.getElementById('saveResourceSettings');
 button.disabled=true;
 try{
  const res=await fetch('/api/control/resources',{
   method:'POST',credentials:'same-origin',cache:'no-store',
   headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},
   body:JSON.stringify({max_running_sidecars:max,idle_stop_seconds:idle})
  });
  if(!res.ok)throw new Error((await res.text()).trim().slice(0,140)||'HTTP '+res.status);
  const data=await res.json();
  resourceSettingsDirty=false;
  if(data.status==='unchanged'){controlNotice('資源設定沒有變更');button.disabled=false;return}
  controlNotice('資源設定已儲存，正在重啟 AhB。登入資料及其他設定不變。');
  setTimeout(()=>location.reload(),8500);
 }catch(err){button.disabled=false;controlNotice('儲存失敗：'+err.message)}
});
switchTab('home');
const fmtBytes=n=>!n?'—':n<1048576?(n/1024).toFixed(1)+' KiB':(n/1048576).toFixed(1)+' MiB';
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let lastModels=[];
let lastProviders=[];
let modelLimit=120,providerFilter='all',refreshing=false,runtimePolling=false;
let scanningAllModels=false;
async function getJSON(path){const r=await fetch(path,{cache:'no-store'});if(!r.ok)throw new Error(path+' HTTP '+r.status);return r.json()}
function modelCounts(models){const out={};for(const m of models){const p=m.x_provider||'';out[p]=(out[p]||0)+1}return out}
function accountLabel(x){
 if(x.account_total!==null&&x.account_total!==undefined)return String(x.account_usable_count||0)+'/'+String(x.account_total);
 if(x.id==='opencode'&&x.account_usable===true)return 'ANON';
 return x.account_usable===true?'YES':(x.account_usable===false?'NO':'UNKNOWN');
}
function lastRequestLabel(x){
 if(!x.last_request_at)return 'NOT TESTED';
 if(x.last_request_transport_error)return 'NETWORK ERROR';
 if(x.id==='duckai'&&x.last_request_http_status===418)return 'HTTP 418 · Duck.ai 拒絕請求';
 return x.last_request_http_status===503?'HTTP 503 · 上游失敗／額度待查':(x.last_request_http_status?'HTTP '+x.last_request_http_status:'UNKNOWN');
}
function renderModels(){
 const q=(document.getElementById('search').value||'').trim().toLowerCase();
 const rows=lastModels.filter(m=>!q||String(m.id||'').toLowerCase().includes(q)||String(m.x_provider_name||m.x_provider||'').toLowerCase().includes(q));
 const shown=rows.slice(0,modelLimit);
 document.getElementById('modelCount').textContent=rows.length===lastModels.length?lastModels.length+' 個模型':rows.length+' / '+lastModels.length+' 個模型';
 document.getElementById('models').innerHTML=shown.length?shown.map(m=>
 '<tr><td><code>'+esc(m.id)+'</code>'+(m.x_cached?'<span class="muted"> · 休眠快取</span>':'')+'</td><td>'+esc(m.x_provider_name||m.x_provider||'—')+'</td><td><code>'+esc(m.x_upstream_id||'—')+'</code></td></tr>'
 ).join(''):'<tr><td colspan="3" class="muted">沒有符合的模型</td></tr>';
 const more=document.getElementById('showMoreModels');more.hidden=rows.length<=modelLimit;
 more.textContent='顯示更多模型 · '+shown.length+' / '+rows.length;
}
function applyProviderFilter(){
 let visible=0;
 document.querySelectorAll('#providers article[data-provider-id]').forEach(card=>{
 const x=lastProviders.find(p=>p.id===card.dataset.providerId);
 const problem=x&&x.enabled&&(x.state==='DEGRADED'||x.state==='DEAD'||Number(x.last_request_http_status)>=400);
 const match=providerFilter==='all'||!!(x&&(providerFilter==='enabled'&&x.enabled||providerFilter==='problems'&&problem||providerFilter==='asleep'&&x.enabled&&!x.process_alive));
 card.hidden=!match;if(match)visible++;
 });
 document.getElementById('providerFilterEmpty').hidden=visible>0;
 document.querySelectorAll('[data-provider-filter]').forEach(b=>{const selected=b.dataset.providerFilter===providerFilter;b.classList.toggle('selected',selected);b.setAttribute('aria-pressed',String(selected))});
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
 if(refreshing)return;
 refreshing=true;
 const error=document.getElementById('error');error.style.display='none';
 const warnings=document.getElementById('warnings');warnings.style.display='none';
 try{
  const [p,m,r]=await Promise.all([getJSON('/api/providers'),getJSON('/v1/models'),getJSON('/api/runtime')]);
  lastModels=m.data||[];
  const counts=modelCounts(lastModels);
  const providers=p.providers||[];
  lastProviders=providers;
  document.getElementById('scanAllModels').disabled=!ahbProviderControlAvailable||scanningAllModels;
  const sidecarRSS=providers.reduce((n,x)=>n+(x.rss_bytes||0),0);
  const ready=providers.filter(x=>x.enabled&&x.state==='HEALTHY').length;
  const ahbRSS=r.ahb_rss_bytes||((r.process_rss_bytes||0)+sidecarRSS);
  const used=r.system_used_bytes||0, total=r.system_total_bytes||0;
  const percent=total?Math.min(100,Math.max(0,Math.round(used/total*100))):0;
  document.getElementById('runtime').textContent='AhB RSS '+fmtBytes(ahbRSS)+' · 按需 '+(r.running_on_demand??0)+'/'+(r.max_running_sidecars||1)+' · '+(r.goos||'?')+'/'+(r.goarch||'?');
  document.getElementById('systemRamSummary').textContent=total?('RAM '+percent+'% 已使用'):'系統 RAM 尚無可用數據';
  document.getElementById('systemRamValue').textContent=total?(fmtBytes(used)+' / '+fmtBytes(total)):'未取得';
  document.getElementById('availableRamValue').textContent=total?fmtBytes(r.system_available_bytes||0):'未取得';
  document.getElementById('ahbRamValue').textContent=fmtBytes(ahbRSS);
  document.getElementById('ramMeter').style.width=percent+'%';
  document.getElementById('ramSourceHint').textContent=r.system_memory_source==='cgroup_v2'
   ?'此 Linux 容器有獨立記憶體上限，顯示整個 cgroup 用量（含其他程序及快取）。'
   :'顯示整台主機 /proc/meminfo 的實際可用記憶體（不是只有 AhB）。';
  if(!resourceSettingsDirty){
   document.getElementById('maxResident').value=r.max_running_sidecars||1;
   document.getElementById('idleTimeout').value=r.idle_stop_seconds||120;
  }
  document.getElementById('saveResourceSettings').disabled=!ahbProviderControlAvailable;
  document.getElementById('providerTotal').textContent=providers.filter(x=>x.enabled).length;
   document.getElementById('summaryReady').textContent=ready;
   document.getElementById('summaryRss').textContent=fmtBytes(ahbRSS);
   const conn=document.getElementById('connectionStatus');conn.textContent='Hub 已連線';conn.className='connection-pill online';
   document.getElementById('lastSynced').textContent='最近同步 '+new Date().toLocaleTimeString('zh-TW',{hour:'2-digit',minute:'2-digit',second:'2-digit'});
  document.getElementById('modelTotal').textContent=lastModels.length;
  document.getElementById('providerReady').textContent=ready+' endpoint healthy / '+providers.length+' configured';
  const previousCards=new Map([...document.querySelectorAll('#providers article[data-provider-id]')].map(card=>{const input=card.querySelector('input[data-provider-proxy-url]'),details=card.querySelector('details.proxy-config');return [card.dataset.providerId,{value:input?input.value:'',open:!!(details&&details.open),focused:document.activeElement===input}]}));
   document.getElementById('providers').innerHTML=providers.map(x=>{
   const manageable=x.state==='HEALTHY'||x.state==='DEGRADED';
   const enabled=!!x.enabled;
   return '<article class="card" data-provider-id="'+esc(x.id)+'" data-state="'+esc(x.state)+'" data-enabled="'+esc(x.enabled)+'"><div class="card-top">'+
    '<div class="provider-main"><div class="provider-name"><span class="state-dot '+esc(x.state)+'"></span>'+esc(x.display_name||x.id)+' <span class="badge">'+esc(x.state)+'</span></div><div class="desc">'+esc(x.description||x.id)+'</div>'+
     '<div class="layers"><span class="layer">Process <b>'+esc(x.kind==='external'?'N/A':(x.process_alive?'YES':'NO'))+'</b></span><span class="layer">Ready <b>'+esc(x.provider_ready?'YES':'NO')+'</b></span><span class="layer">Credentials <b>'+esc(accountLabel(x))+'</b></span><span class="layer" title="最近一次 API 上游回覆 HTTP 狀態；200 不保證串流完整或有可用額度">Last API <b>'+esc(lastRequestLabel(x))+'</b></span></div>'+
     (x.last_error?'<div class="provider-error">'+esc(x.last_error)+'</div>':'')+'</div>'+
    '<div class="metrics"><div class="metric"><b>'+esc(x.enabled&&x.start_mode==='on_demand'&&!x.process_alive&&!(counts[x.id]>0)?'SLEEP':(counts[x.id]||0))+'</b><span>Models</span></div><div class="metric"><b>'+esc(fmtBytes(x.rss_bytes))+'</b><span>RSS</span></div><div class="metric"><b>'+esc(x.restarts||0)+'</b><span>Restarts</span></div></div>'+
    '<div class="provider-actions">'+
      '<label class="ui-toggle"><input type="checkbox" data-provider-toggle="'+esc(x.id)+'" '+(x.enabled?'checked ':'')+(!ahbProviderControlAvailable?'disabled ':'')+' aria-label="'+esc(x.display_name||x.id)+' 啟用或停用"><span>'+(x.enabled?'已開啟':'已關閉')+'</span></label>'+
      (enabled&&x.start_mode==='on_demand'&&!x.process_alive?'<button type="button" class="btn" data-provider-wake="'+esc(x.id)+'" '+(!ahbProviderControlAvailable?'disabled':'')+'>啟動並載入模型</button>':'')+
      (enabled&&x.start_mode==='on_demand'&&x.process_alive?'<button type="button" class="btn" data-provider-recover="'+esc(x.id)+'" '+(!ahbProviderControlAvailable?'disabled':'')+'>單獨重啟</button>':'')+
      (x.id==='copilot'?'<button type="button" class="btn primary" data-copilot-auth>GitHub 授權登入</button>':actionLink(x.ui_url,'管理原本 UI',true,enabled&&manageable))+
       (enabled?'<a class="btn" href="/playground?provider='+encodeURIComponent(x.id)+'">直接測試</a>':'')+
       actionLink(x.docs_url,'上游文件',false,true)+'</div>'+
      (x.id==='copilot'?'<div class="provider-recovery-note">Copilot2API 沒有網頁管理台。請先使用「GitHub 授權登入」，完成後再啟動並在 Playground 實測；/v1/models 不代表有額度。</div>':'')+
       (x.id==='duckai'&&x.last_request_http_status===418?'<div class="provider-error">Duck.ai 最近的實際推論被上游拒絕（HTTP 418）。/ping 成功不代表可用；暫停使用並確認官方服務狀態，請勿連續重啟或反覆請求。這不代表 IP 已永久被封鎖。</div>':'')+
      (x.id==='deepseek'&&x.enabled&&(x.account_total===0||x.account_usable===false)?'<div class="provider-recovery-note">實驗版 DeepSeek Web 已移除舊 /admin 管理台。請將自己授權的 Web 帳號憑證放入本機私有 data/deepseek2api/accounts.txt（權限 0600），然後使用 Playground 檢查真實回覆。舊版管理密碼無法登入新版本。</div>':'')+
      ((x.last_request_http_status===503||x.last_request_http_status===502)?'<div class="provider-recovery-note">最近回應 '+esc(x.last_request_http_status)+'：先確認登入與額度，程序卡住時再嘗試單獨重啟（不自動重送）。</div>':'')+
      (x.kind==='sidecar'?'<details class="proxy-config"><summary>程序出站 Proxy · '+(x.proxy_configured?'已設定':'未設定')+'</summary><div class="proxy-input-row"><input type="url" data-provider-proxy-url placeholder="http://127.0.0.1:7890（留白清除）" spellcheck="false" autocomplete="off" aria-label="'+esc(x.id)+' 出站 Proxy URL"><button type="button" class="btn" data-provider-proxy="'+esc(x.id)+'" '+(!ahbProviderControlAvailable?'disabled':'')+'>儲存 Proxy</button></div><div class="proxy-help">只作用於該 Gateway 的 HTTP_PROXY / HTTPS_PROXY 等程序環境變數，可能受上游實作影響；Agent2API 等來源的帳號代理池仍由原生管理介面負責。儲存會重新啟動 AhB。</div></details>':'')+
   '</div></article>';
  }).join('');
  document.querySelectorAll('#providers article[data-provider-id]').forEach(card=>{const saved=previousCards.get(card.dataset.providerId);if(!saved)return;const details=card.querySelector('details.proxy-config'),input=card.querySelector('input[data-provider-proxy-url]');if(details)details.open=saved.open;if(input){input.value=saved.value;if(saved.focused)input.focus({preventScroll:true});}});
   applyProviderFilter();
   if(m.x_provider_warnings&&Object.keys(m.x_provider_warnings).length){
   warnings.textContent=Object.entries(m.x_provider_warnings).map(([k,v])=>k+': '+v).join('\n');warnings.style.display='block';
  }
  renderModels();
 }catch(e){error.textContent=e.message;error.style.display='block';const conn=document.getElementById('connectionStatus');conn.textContent='連線異常';conn.className='connection-pill offline';document.getElementById('lastSynced').textContent='同步失敗';}
 finally{refreshing=false;}
}
document.getElementById('refresh').addEventListener('click',refresh);
document.getElementById('search').addEventListener('input',()=>{modelLimit=120;renderModels()});
document.getElementById('showMoreModels').addEventListener('click',()=>{modelLimit+=120;renderModels()});
document.querySelector('.provider-filter').addEventListener('click',e=>{const b=e.target.closest('button[data-provider-filter]');if(!b)return;providerFilter=b.dataset.providerFilter;applyProviderFilter()});
// Explicit opt-in discovery: never wake nine gateways just because a client
// polls /v1/models. Requests are sequential to respect the 512 MiB ceiling.
document.getElementById('scanAllModels').addEventListener('click',async()=>{
 if(!ahbProviderControlAvailable||scanningAllModels)return;
 scanningAllModels=true;
 const button=document.getElementById('scanAllModels');
 button.disabled=true;
 const targets=lastProviders.filter(x=>x.enabled&&x.start_mode==='on_demand')
   .sort((a,b)=>Number(a.id==='opencode')-Number(b.id==='opencode'));
 let discovered=0, unavailable=0;
 try{
  for(let i=0;i<targets.length;i++){
   const id=targets[i].id;
   button.textContent='讀取中 '+(i+1)+' / '+targets.length+' · '+id;
   try{
    const res=await fetch('/api/control/wake/'+encodeURIComponent(id),{
     method:'POST',credentials:'same-origin',cache:'no-store',
     headers:{'X-AhB-Control-Token':ahbControlToken,'Content-Type':'application/json'},body:'{}'
    });
    if(!res.ok)throw new Error('HTTP '+res.status);
    const answer=await res.json();
    if(answer.model_discovery==='ok')discovered++;
    else unavailable++;
   }catch(_){unavailable++}
  }
  controlNotice('模型掃描完成：'+discovered+' 個來源已回傳模型資料，'+unavailable+' 個未確認（可能未登入、忙碌或無法讀取）。休眠快取不是額度驗證。');
  await refresh();
 }finally{
  scanningAllModels=false;
  button.textContent='逐一載入全部模型';
  button.disabled=!ahbProviderControlAvailable;
 }
});
for(const [buttonId,cmdId] of [['copyCopilotInstall','copilotInstallCommand']]){
 document.getElementById(buttonId).addEventListener('click',async function(){
  const original=this.textContent;
  this.textContent=(await copyText(document.getElementById(cmdId).textContent))?'已複製':'複製失敗';
  setTimeout(()=>this.textContent=original,1300);
 });
}
document.getElementById('copyBase').addEventListener('click',async()=>{
 const btn=document.getElementById('copyBase'),text=document.getElementById('baseUrl').textContent;
 btn.textContent=(await copyText(text))?'已複製':'複製失敗';setTimeout(()=>btn.textContent='複製 Base URL',1200);
});
// No secrets are requested in the browser: bridge keys are configured locally
// through the script's private environment, never embedded into a URL or page.
const bridgeDefaults={cliproxy:8416,custom:8418};
let previousPreset='lmarena';
function updateBridgeCommand(){
 const preset=document.getElementById('bridgePreset').value;
 const id=preset==='custom'?document.getElementById('customBridgeId').value.trim():preset;
 const address=document.getElementById('bridgeURL').value.trim();
 const cmd=document.getElementById('bridgeCommand');
 const hint=document.getElementById('bridgeHint');
 const copy=document.getElementById('copyBridge');
 const validID=/^[a-z][a-z0-9_-]{1,30}$/.test(id)&&id!=='route';
 const urlMatch=address.match(/^http:\/\/(127\.0\.0\.1|localhost|\[::1\]):([0-9]{1,5})\/?$/);
 const validURL=!!urlMatch&&Number(urlMatch[2])>0&&Number(urlMatch[2])<=65535;
 const valid=validID&&validURL;
 cmd.textContent=valid?'cd ~/AhB && ./scripts/connect-bridge.sh '+id+' '+address+' && ./scripts/stop-termux.sh && ./scripts/start-termux.sh':'請輸入有效的來源代號和 127.0.0.1 本機連接埠。';
 hint.textContent=valid?'這條指令會驗證並連接本機橋接服務，接著重新啟動 AhB 使設定生效；現有請求會中斷。':'代號只能含小寫英文、數字、- 或 _；網址須為本機 HTTP 地址。';
 hint.classList.toggle('tip-error',!valid);
 copy.disabled=!valid;
}
document.getElementById('bridgePreset').addEventListener('change',function(){
 const preset=this.value;
 document.getElementById('bridgeURL').value='http://127.0.0.1:'+bridgeDefaults[preset];
 document.getElementById('customIdField').style.display=preset==='custom'?'grid':'none';
 previousPreset=preset;
 updateBridgeCommand();
});
document.getElementById('bridgeURL').addEventListener('input',updateBridgeCommand);
document.getElementById('customBridgeId').addEventListener('input',updateBridgeCommand);
document.getElementById('copyBridge').addEventListener('click',async function(){
 if(this.disabled)return;
 const ok=await copyText(document.getElementById('bridgeCommand').textContent);
 this.textContent=ok?'已複製':'複製失敗';
 setTimeout(()=>this.textContent='複製連接指令',1400);
});
updateBridgeCommand();
document.getElementById('baseUrl').textContent=location.origin+'/v1';
refresh();
// Refresh RAM separately: avoid re-probing all upstream model lists merely
// to update the memory gauge on a 512 MiB VPS.
setInterval(()=>{
 if(document.visibilityState==='hidden'||runtimePolling)return;
 runtimePolling=true;
 getJSON('/api/runtime').then(r=>{
  const total=r.system_total_bytes||0,used=r.system_used_bytes||0;
  const pct=total?Math.min(100,Math.round(used/total*100)):0;
  document.getElementById('systemRamSummary').textContent=total?'RAM '+pct+'% 已使用':'系統 RAM 尚無可用數據';
  document.getElementById('systemRamValue').textContent=total?fmtBytes(used)+' / '+fmtBytes(total):'未取得';
  document.getElementById('availableRamValue').textContent=total?fmtBytes(r.system_available_bytes||0):'未取得';
  document.getElementById('ahbRamValue').textContent=fmtBytes(r.ahb_rss_bytes);
  document.getElementById('ramMeter').style.width=pct+'%';
  document.getElementById('runtime').textContent='AhB RSS '+fmtBytes(r.ahb_rss_bytes)+' · 按需 '+(r.running_on_demand??0)+'/'+(r.max_running_sidecars||1)+' · '+(r.goos||'?')+'/'+(r.goarch||'?');
 }).catch(()=>{}).finally(()=>{runtimePolling=false});
},10000);
setInterval(()=>{if(document.visibilityState!=='hidden'&&!document.getElementById('providersSection').classList.contains('ahb-tab-hidden'))refresh()},45000);
document.addEventListener('visibilitychange',()=>{if(document.visibilityState==='visible'&&!document.getElementById('providersSection').classList.contains('ahb-tab-hidden'))refresh()});
</script>
</body>
</html>`

func (h *Hub) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// This HTML contains a page-local control nonce and can request private
	// native-console credentials. Never allow another website to frame it
	// and trick users into clicking administrative buttons.
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	html := strings.Replace(dashboardHTML, "__AHB_CONTROL_TOKEN__", h.controlToken, 1)
	available := "false"
	if h.controlToken != "" && h.restartFn != nil && !h.cfg.AllowLAN {
		available = "true"
	}
	html = strings.Replace(html, "__AHB_RESTART_AVAILABLE__", available, 1)
	html = strings.Replace(html, "__AHB_PROVIDER_CONTROL_AVAILABLE__", available, 1)
	updater := "false"
	if available == "true" && h.updateFn != nil { updater = "true" }
	html = strings.Replace(html, "__AHB_UPDATE_AVAILABLE__", updater, 1)
	_, _ = w.Write([]byte(html))
}