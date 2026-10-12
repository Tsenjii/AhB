# AhB 故障雷達（Flight Recorder）

此功能將統一 `/v1` API 從「只顯示最後一次錯誤」升級為可觀測的本機服務。開啟 AhB 控制台 `/ui` → **故障雷達**，不需外部監控平台、外部帳號或任何額外依賴。Android / Termux、Linux 及 Docker 採用同一份 Go 實作。

## 功能

- **無磁碟持久化**：固定長度 256 筆的環狀事件紀錄，重啟清空；僅保存成功/失敗 HTTP 狀態碼、固定 API 端點、AhB 已知來源 ID、請求耗時、首次回應本文時間與傳輸的 bytes。最近 48 筆事件可直接查看；HTTP 418、429、401/403、502/503/504 顯示不同排障建議。
- **即時來源比較**：以記憶體中的樣本計算總成功率、P95 完整請求時間、來源成功/失敗次數；適合找出最近失敗率偏高的 Gateway。P95 完整請求時間包含串流生成過程，非模型首 token 延遲。
- **串流相容**：只包裝標準 ResponseWriter，透過 `Flush` 立即透傳 SSE 串流，不讀取或儲存 payload。HTTP 200 不代表串流中途一定沒有中斷。
- **一鍵帶走診斷資料**：JSON 匯出只包含上述暫存的事件中繼資訊，方便排除 Android 實機遇到的 418、502、503 等問題；亦可複製不含內文的故障摘要。
- **不碰使用中的帳號**：不會改 Token、OAuth、Cookie、proxy 或帳號池，不會自動喚醒 Gateway / 發送額外推理 / 重試失敗請求。不包含 ModelAtlas 類型的模型聚合與搜尋功能。

## API

本機 `GET /api/insights` 需要當次儀表板 HTML 內的 `X-AhB-Control-Token` 標頭。除此之外還需符合 **loopback listen / Host / remote address** 限制，且當 `allow_lan=true` 時一律拒絕。未授權請求回 403、錯誤方法回 405。請不要把此頁面或控制台 nonce 截圖分享。

回應欄位：

```json
{
  "since": "2026-10-12T12:00:00Z",
  "sample_window": 256,
  "requests": 2,
  "successes": 1,
  "failures": 1,
  "success_rate": 50,
  "p95_ms": 1200,
  "by_provider": [
    {"provider": "opencode", "requests": 2, "successes": 1, "failures": 1, "last_status": 503, "p95_ms": 1200}
  ],
  "recent": [
    {
      "at": "2026-10-12T12:00:30Z",
      "endpoint": "/v1/chat/completions",
      "provider": "opencode",
      "status": 503,
      "category": "gateway_unavailable",
      "duration_ms": 1200,
      "first_byte_ms": 1200,
      "response_bytes": 80
    }
  ],
  "privacy": "Volatile, bounded metadata only..."
}
```

上方為**示意欄位**，不代表 opencode 已被真實測出這些數字。若無已知來源歸屬則使用 `unattributed`，不能據此斷定 Gateway 有故障。只支援固定白名單 API 路徑，query string 和模型名稱皆不收集。HTTP 請求數限最近 256 次、非歷史累積。

## 問題定位

- **418**：已收到上游拒絕回覆，不表示本機 DNS 壞掉，也不能武斷宣稱帳號或 IP 永久遭封鎖。
- **429**：檢查官方配額、限流與重置時間，不應短時間內重試。
- **401 / 403**：檢查授權與帳號權限，避免分享到 Token 或敏感資訊。
- **502 / 503 / 504**：可由 Playground 直接測試單一來源，分辨來源沒有啟動、上游返回 503、授權失敗；單純重啟不保證解決。
- **2xx**：僅表示 HTTP status 符合 2xx，不代表模型產生了有效內文、SSE 完整、工具呼叫可用或尚有額度。

## 本次升級的限制

1. 本次是軟體層的觀測升級，非手機端實測。對應上游服務仍需使用者自己的合法帳號完成授權與實際推理驗收。
2. 指標不會寫到磁碟，不會把帳號/模型/內容輸出到第三方監控。安裝後只有發生實際 API 請求才會累積數據。
3. 當 `allow_lan=true`，故障雷達頁面會顯示受限說明；`/v1` 正常保留其原本的 LAN 行為。
