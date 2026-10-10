# AhB 本機原生管理控制台

AhB 的 Dashboard → **帳號與登入 → 原生控制台登入**，列出各個原本的 Gateway 管理頁、其管理員驗證方式，以及按需複製管理密碼的按鈕。

這不會覆蓋上游的使用者帳號，也不會建立同一個不安全的預設密碼。對尚未支援單一登入協定的上游，瀏覽器仍必須使用原本管理員登入頁。建議交由瀏覽器密碼管理器記住各來源的管理資料；不要在專案原始碼中加入固定密碼「0000」。

| 來源 | URL (手機本機) | AhB 取得憑證方式 | 限制 |
|---|---|---|---|
| OpenCode | http://127.0.0.1:8404/ | 私人 `data/opencode/config.json` 的 webui 帳號密碼 | 如果在上游修改過，顯示以實際本機設定為準 |
| Gemini Web | http://127.0.0.1:8413/admin | 私人 `config.json` 裡 `geminiweb.env.ADMIN_TOKEN` | 來源原生 Admin Token；可能與 AhB local API key 共用 |
| Grok2API | http://127.0.0.1:8407/ | `data/grok2api/admin-password.txt` 初始密碼 | 資料庫建立後若曾在上游更換密碼，初始密碼可能失效 |
| Kiro-Go | http://127.0.0.1:8408/admin | 私人 `data/kiro-go/config.json` 的 password | 不會更動原生帳號 |
| Agent2API | http://127.0.0.1:8403/ | 使用原本管理頁 | AhB 不覆蓋它的帳號或代理池 |

實驗版 DeepSeek Web 沒有舊版 `/admin`；其網頁登入帳號資料必須放在私有 `data/deepseek2api/accounts.txt`，AhB 不會將原來的管理密碼誤認作網頁帳號，也不會顯示 Token。需經 Playground 真實驗證。

FreeBuff 的 OAuth 與 Copilot 的 GitHub 授權流程位於同一個 **帳號與登入** 分頁，但那是 AI 平台自己的官方授權，不是 Gateway 管理員密碼。

**Copilot 特別說明：** 上游 [whtsky/copilot2api](https://github.com/whtsky/copilot2api) 沒有獨立 WebUI。請從首頁 Copilot 卡片按「GitHub 授權登入」，再按「登入 GitHub Copilot」取得官方一次性授權碼，前往 [GitHub 官方裝置授權頁](https://github.com/login/device) 輸入代碼。手機重新整理後可恢復進行中的授權狀態；授權錯誤只顯示安全分類，不會暴露 Token 或原始上游日誌。授權資料存在仍不代表 Copilot 帳號具備模型資格或剩餘額度。

## 安全性與操作

- 只有直接在 localhost 打開 AhB Dashboard、並按下個別「複製管理密碼」時，後端才會把該秘密交給本機瀏覽器的剪貼簿。預設清單與網頁 HTML **不包含密碼或 Token**。
- 要求 loopback 監聽與來源、不可預測的頁面控制 Token，以及 POST。跨網站偽造請求會被拒絕。
- 讀取位置是固定白名單，並拒絕符號連結、非正規檔案、群組或其他使用者可讀的私密檔案，不提供任意檔案路徑。
- 密碼複製後會短暫出現在系統剪貼簿。不要貼到聊天、螢幕截圖、記錄檔或 GitHub Issue。
- 如果 Gateway 還在休眠，請先在 AhB 總覽啟動該服務；密碼複製不會喚醒 Gateway，也不會消耗推論額度。
- Docker 的公開 HTTP 入口明確阻擋所有 `/api/control/*` 管理操作，因此此功能不會在 Docker 的外部認證代理公開。

**限定與待驗收：** 此功能是「集中讀取／複製／開啟」管理入口，**不是**把不同上游 WebUI 改成支援 AhB 的共用 SSO。CI 可確認本機檔案和 UI 協定，但無法保證使用者目前的 Grok 管理資料庫、上游介面改版或瀏覽器剪貼簿權限。在 Android 真機更新前仍須驗收。