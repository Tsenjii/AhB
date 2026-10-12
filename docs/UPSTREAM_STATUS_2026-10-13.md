# AhB 上游版本核對與實機證據 — 2026-10-13（台灣）

本文件記錄 GitHub Repo 主分支實際提交及原始碼比對，**不將模型清單、健康檢查或 CI 當成推論通過**。AhB 的職責仍是已授權帳號管理和統一 API，不取代 ModelAtlas 的模型發掘功能。

## 九個內建 Gateway 的版本核對

| Gateway | 當前上游 HEAD | AhB 策略 | 上游連結 |
|---|---|---|---|
| OpenCode2API | `1e2702f6` | 維持現有版本，上游無新提交 | [opencode2api](https://github.com/jasonxu114514/opencode2api) |
| FreeBuff Node.js | `93bf2a9d` | **維持釘選 `c4db1a35`**：比較結果 8 個新增提交均僅更新 MODELS.md，沒有新的核心程式碼 | [yutian81/freebuff2api](https://github.com/yutian81/freebuff2api) |
| Agent2API | `fd869a46` | 維持現有 v3.0.1 | [aimod-cc/agent2api](https://github.com/aimod-cc/agent2api) |
| Copilot2API | `a4aac95d` | 維持 | [whtsky/copilot2api](https://github.com/whtsky/copilot2api) |
| Grok2API | `7c889a96` | 維持 | [chenyme/grok2api](https://github.com/chenyme/grok2api) |
| Kiro-Go | `f8f6071c` | 維持 | [Quorinex/Kiro-Go](https://github.com/Quorinex/Kiro-Go) |
| DeepSeek2API（實驗性） | `41b924e7` | 維持；有模型目錄不代表 upstream chat 可用 | [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api) |
| Gemini Web2API Go | `93d9b384` | **更新 Android 與 Linux 釘選至 4.21.0**；較舊 `bc09c717` 多 5 筆上游提交（temporary_chat 配置、傳輸選項、前端開關、多輪互斥測試） | [zexadev/gemini-web2api-go](https://github.com/zexadev/gemini-web2api-go) |
| Duck.ai2API（實驗性） | `d8c6888e` | 維持；此前實際推論曾出現 HTTP 418，無法以編譯宣稱修好 | [duckai2api](https://github.com/desktop-tools-which-may-be-useful/duckai2api) |

更新時選擇**經差異審核的固定 commit**，而非每次 Actions 都拉取 HEAD，以避免突然引進不同 API 契約、互斥設定或破壞 Termux 相容性。

## Z.ai Web（選裝，不屬於內建九個來源）

- 研究來源：[XxxXTeam/zai2api](https://github.com/XxxXTeam/zai2api)，固定 `2a2ddd0f955434fd545542ec41dd2bf02fd1ff05`，Go 1.25、GPLv3。其專案表明無法再取得訪客 Token，因此**需要來源支援的合法帳號**。
- AhB 新增 `zaiweb` 選裝本機連接預設（示意埠 8460）；使用者需自行合法設定、啟動來源。原來源範例埠為 8000，建議為避免撞其他本機服務而另設埠。
- 源碼內包含訪客 Token 池等其他行為；AhB **不提供**人機驗證、偽造身分、規避上游限流的安裝或設定流程，也不啟動此源碼；只以使用者獨立管理之既有 OpenAI 相容本機 API 連接。
- GitHub Actions 的 [verify-zaiweb-bridge.yml](../.github/workflows/verify-zaiweb-bridge.yml) 執行**固定 SHA 的 Go 編譯**與 AhB 的自動化相容測試，不會登入線上網站，亦無真實上游推論資格。Actions 通過後，還需本機測試聊天、多輪、SSE [DONE]、原生工具循環。
- 連接需要**下游 client API Key** 時，用 Termux 既有 `scripts/connect-bridge.sh` 私密輸入；不要在公開 Actions、PR、issue 或 UI 上貼真實憑證。

## FreeBuff 問題：目前能確認什麼？

**截至此次檢查，還沒有 FreeBuff 端到端實機成功的證據。** AhB 已經有官方 CLI OAuth URL / polling、本機 credential import、Gateway 啟動驗收及測試 fixture；這些功能的完成不代表 user credentials 真正能在台灣手機拿到有效推論。

新增「帳號 → FreeBuff → 診斷 FreeBuff」按鈕，會讀取**本機統計**：

1. CLI Bearer 帳號候選數（只計數不讀出內容）
2. 是否啟用以及 sidecar 程序是否啟動
3. 若程序在執行，最多 3 秒訪問 **loopback `/healthz`**，僅解析 accounts / alive_accounts / unknown_accounts，並顯示程序報的總數
4. 最近一次 HTTP 推論狀態（已有 AhB Flight Recorder 的上游記錄）
5. 依結果給中文下一步：完成官方登入、啟動程序、重新載入已授權帳號、或進 Playground 真實測試

不會讀出帳號 Token，不會擅自把 Cookie 與 Bearer 互換、不中斷推論，也不會為了檢測而執行高頻推論。GitHub Actions 沒有使用者私密 Credential，不能替代手機授權。

## 發布門檻

1. `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./cmd/hubd`
2. `scripts/test-connect-bridge.sh`、`scripts/test-dashboard.sh` 與 FreeBuff 診斷授權測試
3. Android ARM64、Linux AMD64 / ARM64 原生編譯成功
4. **真正的 API 成功聲明**需手機 / 目標部署端真實帳號測試（合法登入、推論內容、SSE 結尾、Tool Calling 第二輪）
