# FreeBuff 在 Android / Termux 的登入與驗證

審查上游：[Freebuff2API v0.10.3](https://github.com/lza6/Freebuff-2API/tree/v0.10.3)、[中文說明](https://github.com/lza6/Freebuff-2API/blob/v0.10.3/README_zh.md)、[平台登入入口](https://github.com/lza6/Freebuff-2API/blob/v0.10.3/src/login_window.rs)。

## 能直接使用什麼？

AhB 預編譯包包含 FreeBuff Rust 閘道與 WebUI；不必再安裝 Rust、Electron 或 Windows WebView2 才能在 Termux 執行。但程式能啟動不代表已有登入的帳號。

- **Android / Termux**：原生內嵌一鍵登入程式使用 Windows WebView2，其他平台為 login_window_stub.rs，不會彈出相同登入視窗。請在 FreeBuff 自身的管理頁「帳號 → 匯入」依照上游支援的方式匯入自己有權使用的憑證。
- **Windows 桌面安裝版**：上游桌面程式提供自己的登入視窗。
- **桌面 Chrome／Edge 網頁面板**：上游有選裝的瀏覽器擴充套件，需自行審閱權限；擴充套件及本機閘道必須在同一台電腦。電腦瀏覽器中的 localhost 並不是 Android 手機的 localhost。

Android Chrome 不會因 AhB 已安裝就支援桌面擴充套件；AhB 不讀取其他 App 的 Cookie。切勿把 Cookie、Token、HAR、帳號匯出或含私密金鑰的截圖貼進聊天／GitHub issue。

## 手機上檢查帳號（不消耗模型額度）

```sh
cd ~/AhB
./scripts/start-termux.sh
./scripts/check-freebuff-login.sh
```

檢查程式只讀取 FreeBuff 本機 /api/accounts/health 的帳號筆數；沒有登入顯示 NO ACCOUNTS；存在 N 筆帳號也不代表可推理或有額度。HTTP 錯誤則代表本機閘道連線或認證需要檢查。

在**手機本機**瀏覽器開 http://127.0.0.1:8402/ui，使用 FreeBuff 原生「帳號／匯入」流程。完成後再次檢查帳號筆數，並在模型清單中查詢實際的 freebuff/ 模型 ID，選擇已授權模型做真實 Chat / SSE / Tool Calling 驗收。真實推理會消耗上游原有配額。

## 狀態、安全與架構

FreeBuff 的憑證、帳號池與配額管理由 Freebuff2API 負責；AhB 只監督程序、顯示無敏感資料的摘要、提供模型前綴與 API 轉發。資料預設保留在 data/freebuff/，不要將真實憑證寫入 GitHub、config.example.json 或 AhB 網頁 JavaScript。升級時使用既有的非破壞性更新程序，保留舊版備份。

在未確認安全、可靠的 Android 原生登入方案前，應清楚呈現手動匯入，而不是提供不會成功的「一鍵登入」假按鈕。
