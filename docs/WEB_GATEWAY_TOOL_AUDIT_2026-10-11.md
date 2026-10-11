# AhB 網頁版 Gateway 換源與工具呼叫驗收（2026-10-11）

## 實機回報與本輪結論

使用者實測：舊 `deepseek` 程序無法啟動、Duck.ai 發生上游拒絕，FreeBuff 在網頁控制台和 Termux 均無法完成授權；要求增加 Kimi／Qwen Web-to-API，並確認所有服務的真正 Tool Calling。

**不可把編譯成功、健康檢查 HTTP 200、靜態模型清單、登入檔案存在、或模擬環境下的工具解析測試寫成「帳號實機通過」。**

| 服務 | 目前 AhB 整合方式 | 工具調用的證據等級 | 下一步 |
| --- | --- | --- | --- |
| OpenCode | 原生 Android/Linux 包；已回報部分聊天能用 | 過往僅有初次工具呼叫輸出紀錄，完整雙輪尚未實機驗收 | 以 `audit-tool-calling.sh` 實測 |
| Agent2API v3.0.1 | 內建 Rust，來源自己的多帳號管理台 | 來源具工具協定處理，未驗證使用者帳號的雙輪工具結果 | 實機實測 |
| FreeBuff Node | 內建；官方 CLI OAuth 授權 | 上游提供工具／tool_result 轉換，並非帳號實測 | 先完成授權，再實機實測 |
| Grok2API | 內建 | 已有本機 Chat/SSE/Tools fixture，不代表實際帳號都通過 | 實機實測 |
| Kiro-Go | 內建 | 尚無實機雙輪驗收 | 實機實測 |
| Copilot2API | 內建，需 GitHub 授權及資格 | 尚無實機雙輪驗收 | 實機實測 |
| Gemini Web Go | 內建 | 尚無實機雙輪驗收 | 實機實測 |
| DeepSeek 舊 Go | 內建但預設停用 | 進程未能實機啟動，`/v1/models` 是靜態模型目錄 | 不作為正式可用來源；先改用獨立已登入的替代 Web gateway |
| Duck.ai Rust | 內建但預設停用 | 實機上游 418；健康檢查不代表推論成功 | 維持停用，沒有真實成功證據前不宣稱修復 |
| Kimi Web | **選裝** Python 服務，已有 `install-kimiweb-termux.sh`／`enable-kimiweb-termux.sh` | [chopper1026/kimi2api](https://github.com/chopper1026/kimi2api) 作者明言非 Tool Calling 編程場景最佳化 | 可先驗證聊天；工具需實機雙輪實測 |
| Qwen Web Go | 可用獨立本機來源 `qwenweb/` 或既有 `qwen/` 接入 [XxxXTeam/Qwen2API_Go](https://github.com/XxxXTeam/Qwen2API_Go) | 其內部單元測試包含**提示詞注入、工具回應解析**，不保證是真正原生上游工具協定 | 確認服務可啟動／登入後執行雙輪實測 |
| DeepSeek Web 替代 | 以獨立 `deepseekweb/` 接入 [PruhaNLP/web-to-api](https://github.com/PruhaNLP/web-to-api) | 專案提供**提示詞模擬工具呼叫**，不是保證真正原生 Tool Calling | 需 Chromium；Android/512 MB VPS 資源占用與實際登入尚未驗證 |

### DeepSeek 為什麼啟動失敗

舊 Go 專案 [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api) 在 `config.Load()` 發現沒有 `DEEPSEEK_TOKEN` 或有效的 `accounts.txt` 帳號時，就直接回傳 `no DeepSeek account configured`，在 `main.go` 使用 `log.Fatalf` 結束程序。因此 **「沒有完成帳號設定」本身就會造成無法啟動**；即使設定正確仍須實際推論驗證，不保證上游 Web 協定仍相容。

與其再把已知無法用的實驗版標成主要推薦，本輪新增**獨立來源** `deepseekweb`，使用外部已安裝的 Web-to-API。它是瀏覽器型方案，資源用量和手機支援**尚未**經驗證，絕不能把它宣傳成現有 Go 版的無縫替換。舊 `deepseek` 與 `duckai` 保持預設關閉，不擅自移除使用者的任何 Token、帳號或資料。

### 接入現有 Kimi／Qwen／DeepSeek Web

以上服務都**必須先由使用者自行部署、登入並在相同裝置的 localhost 回覆合法 OpenAI 模型目錄**，AhB 只是檢查可連線並加上前綴。不會偷偷安裝瀏覽器、繞過網站人機驗證、替服務註冊帳號，亦不表示它們已納入 AhB 原生二進位檔。

```bash
cd ~/AhB
./scripts/connect-bridge.sh qwenweb http://127.0.0.1:3000
./scripts/connect-bridge.sh kimi http://127.0.0.1:8000
./scripts/connect-bridge.sh deepseekweb http://127.0.0.1:3456
./scripts/stop-termux.sh && ./scripts/start-termux.sh
```

- `qwenweb/`：上游選擇 [Qwen2API_Go](https://github.com/XxxXTeam/Qwen2API_Go)，Go 1.26 且帶帳號／API key 管理。以本機端點 `3000` 為例；若自己使用不同連接埠請跟著修改。**非已預裝二進位檔。**
- `kimi/`：如用 AhB 既有**內建選裝** [Kimi2API](https://github.com/chopper1026/kimi2api)，請使用 `scripts/install-kimiweb-termux.sh`、`scripts/enable-kimiweb-termux.sh`，而不是連線到不存在的 8000 服務。它會註冊成 `kimiweb/`（本機 8412）；`kimi/` 只用於你另外啟動的橋接器。
- `deepseekweb/`：需要 Node.js + Chromium；不建議在記憶體吃緊的 VPS 或尚未有 Chromium 的 Termux 上直接承諾可用。此選項是**備選**，與舊 `deepseek/` 分開；登入方式、工具呼叫及實機可用性以來源自己的文件與測試為準。

### FreeBuff 登入與 HTTP 502

`freebuff/` 的 [yutian81/freebuff2api](https://github.com/yutian81/freebuff2api) 使用 Codebuff 官方 CLI 授權碼輪詢，必須依次成功：
1. `POST /api/auth/cli/code` 回傳合法官方授權網址。
2. 使用者在官方頁面完成授權；`GET /api/auth/cli/status` 回傳有效使用者憑證。
3. AhB 保存到 `data/freebuff/credentials`（私人權限），FreeBuff Node Gateway 重新載入該資料夾。
4. 用真正的 Chat Completions 進行單次測試，確認有效回答及帳號額度。

前兩步的上游 HTTP 502 或授權錯誤**不是**僅修改 AhB 就能保證修復。網頁控制台會顯示是哪一步與 HTTP 狀態，而不顯示 Token；Termux 腳本會列印一次性授權網址供使用者自行打開。

若已透過受支持的官方 CLI 方式合法取得有效 `authToken`，但腳本匯入遇到問題，可以**在本地 Termux 終端執行以下替代匯入**（隱藏輸入、檔案 0600、不列印 Token）：

```bash
cd ~/AhB
bash scripts/import-freebuff-token-termux.sh
```

這**不會取得新的官方登入憑證，也無法修復官方登入失敗**。已匯入的憑證不等於可用額度；透過 Dashboard 單獨重啟 FreeBuff 後再做一次真實推論驗證。不要將一次性授權網址、Cookie、Token 或個人資料貼到 Issue／聊天。

### 所有 Gateway：真實完整 Tool Calling 批量驗收

使用者**明確啟動**以下命令，每個符合條件的、已啟用的服務選取最多一個**非休眠快取**模型，最多八個服務，逐一發送一次包含 `get_time` 工具的請求、模擬其回傳，再要求模型完成第二輪回答。

```bash
cd ~/AhB
AHB_TOOL_AUDIT_CONFIRM=YES bash scripts/audit-tool-calling.sh
```

它調用已存在的 `scripts/test-tool-roundtrip.sh`，不執行模型產生的程式碼、不會批量註冊、更換 IP/帳號或自動重試；**每模型可能消耗至少兩次真實推論額度**。不會因為格式看似 JSON 或第一輪工具呼叫成功就標記 PASS；必須雙輪完整才算。

`deepseek`、`duckai` 會被略過，因為實機上游尚未通過；休眠快取模型也不會自動喚醒進行額度消耗。結果只在終端顯示每個模型的 PASS／FAIL，無法以 CI 或作者 README 宣稱任一個使用者帳號已通過。

## 仍需後續處理

- 在實際 Android 手機確認 FreeBuff CLI OAuth 失敗階段、來源 HTTP 502／401／429，以及真正可用的授權憑證。
- 驗證 Qwen／Kimi／DeepSeek 替代服務**自身**在 Android／Linux 的安裝、登入和推論是否能完成；這輪新增的是**本機連線選項**，而非憑空提供可用的帳號或雲端服務。
- 只有成功的實際 Chat、SSE、Tool Calling 雙輪資料可升級成「實機可用」狀態。
