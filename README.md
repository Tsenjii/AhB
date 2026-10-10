# AhB｜輕量化 AI 帳號與 API 管理中心

**AhB（Android AI Hub）** 是一個以 Go 開發的輕量級本機 API Hub，整合已安裝的 AI Gateway，提供**多來源帳號管理、服務健康狀態、隨選啟動、模型路由、串流轉送與統一 API**。專案重點不是另一個模型搜尋平台，而是讓你管理自己已獲授權的帳號與本機服務，並透過統一端點使用它們。

- **操作介面：** [`http://127.0.0.1:8317/ui`](http://127.0.0.1:8317/ui)（桌面寬螢幕與手機版面）。
- **模型 Playground：** [`http://127.0.0.1:8317/playground`](http://127.0.0.1:8317/playground)（指定單一 Gateway，測試聊天、串流及工具格式）。
- **統一 API：** `http://127.0.0.1:8317/v1`。
- **原始碼：** [GitHub：Tsenjii/AhB](https://github.com/Tsenjii/AhB)。

> **注意：** Gateway「編譯成功／健康檢查成功」不代表帳號可用、具有足夠額度，或實際模型推論成功。每個來源都必須遵守其原有授權與使用條件；AhB 不會替你憑空提供免費模型額度。

## 最新版本與編譯狀態

**目前已發布的原生版本：`0bf0ba3727fd`（2026-10-11 台灣時間核對）**

| 驗證 | 狀態 | 連結 |
| --- | --- | --- |
| Go 測試、Race Test、Vet | 已通過 | [CI 紀錄](https://github.com/Tsenjii/AhB/actions/runs/38066931307) |
| Android ARM64 正式封包 | 已發布 | [Android 編譯](https://github.com/Tsenjii/AhB/actions/runs/38066931268) |
| Linux AMD64／ARM64 | 已發布 | [Linux 編譯](https://github.com/Tsenjii/AhB/actions/runs/38066931272) |
| 最新 Linux Release（含 SHA-256） | 已發布 | [`linux-0bf0ba3727fd`](https://github.com/Tsenjii/AhB/releases/tag/linux-0bf0ba3727fd) |
| Android `prebuilt` | 來源 SHA 與上述版本一致 | [`prebuilt` 分支](https://github.com/Tsenjii/AhB/tree/prebuilt) |

此版本已包含 **Agent2API v3.0.1**、新控制台、低記憶體防護、SSE 傳輸期間的 Gateway 保護，以及上游版本釘選整理。

如果你是後來才看到這份文件，請先查 [最新 Releases](https://github.com/Tsenjii/AhB/releases/latest) 和 [GitHub Actions](https://github.com/Tsenjii/AhB/actions)；此處的版本紀錄不會自動跟隨未來提交更新。

## Android 與 Linux 版本差異

AhB 維護兩種**原生封包**，共用 Go Hub 與九個 Gateway 定義。它們不是同一種執行檔，不能互相替換。

| 項目 | Android／Termux | Linux VPS |
| --- | --- | --- |
| 處理器 | ARM64 | AMD64、ARM64 |
| 啟動方式 | 隨選啟動 | 隨選啟動 |
| 預設最多同時常駐 Gateway | **3 個** | **1 個** |
| 預設閒置關閉 | **900 秒** | **120 秒** |
| 下載來源 | GitHub `prebuilt` 分支 | 獨立 Linux Release |
| 適合環境 | Android 手機 | 輕量伺服器、私有 VPS |

兩種版本都能從介面的**設定 → 資源**調整常駐數與閒置時間。Linux 可選擇「省記憶體」（1 個／120 秒）、「穩定優先」（2 個／600 秒）或「多來源」（3 個／900 秒）等預設；**選擇預設本身不會套用，必須明確儲存並重新啟動**。

**512 MiB 是部署目標，不是硬性 RAM 保證。** 一個 Gateway 仍可能超過實際可用記憶體；請利用 `/api/runtime` 與系統工具觀察峰值用量。正在執行的 SSE 串流不會因為閒置清理而被主動停止；若設定僅允許一個常駐 Gateway，同時要求其他來源可能回覆 HTTP 503。

## 內建的九個 Gateway

| API 前綴 | 來源 | 封包與使用狀態 |
| --- | --- | --- |
| `opencode/` | OpenCode2API（Go） | 內建，隨選啟動 |
| `freebuff/` | FreeBuff2API（Node.js） | 內建，需另外完成有效的 CLI／Bearer 登入 |
| `agent2api/` | Agent2API（Rust，**v3.0.1**） | 內建，保留上游原始程式與管理方式 |
| `deepseek/` | DeepSeek Web（實驗版 Go） | **預設停用**；尚未驗證真實推論 |
| `grok/` | Grok2API（Go） | 內建，需要完成帳號／金鑰初始化 |
| `kiro/` | Kiro-Go（Go） | 內建，需要有效帳號 |
| `copilot/` | Copilot2API（Go） | 內建，需 GitHub 授權與相應使用資格 |
| `geminiweb/` | Gemini Web2API（Go） | 內建，實際可用性需帳號驗證 |
| `duckai/` | Duck.ai2API（實驗版 Rust HTTP-only） | **預設停用**；曾觀察到上游 HTTP 418 |

這些 Gateway 由 GitHub Actions 針對不同架構編譯；除 FreeBuff 需要 Node.js 執行環境之外，原生 Go／Rust Gateway 不需要在手機或 VPS 上重新編譯。確切上游版本請參閱 [九個 Gateway 版本稽核](docs/UPSTREAM_AUDIT_2026-10-11.md)。

**不隨主封包安裝的服務：** Kimi Web、LMArena、Windsurf、CLIProxyAPI、其他獨立 Gemini／Qwen／Claude 等橋接器。AhB 可以連線至你已經安裝、啟動並授權的本機服務，**但不會假裝它們已經被打包**；現有自訂橋接器與帳號資料也不會因一般升級而故意刪除。

## Android／Termux 安裝

適用 ARM64 Android 手機。直接安裝官方已編譯封包，**不需要在手機編譯 Rust 或 Go，也不需要登入 GitHub**。

```bash
pkg update
pkg install -y curl
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/install-prebuilt-termux.sh | bash
cd ~/AhB
./scripts/run-termux.sh
```

安裝器會下載公開 `prebuilt` 封包、驗證 SHA-256，並準備本機設定與私有憑證。完成後開啟：

- 控制台：`http://127.0.0.1:8317/ui`
- 測試頁：`http://127.0.0.1:8317/playground`

### 更新既有 Android 安裝

**不要刪除 `~/AhB`，也不要用首次安裝腳本覆蓋既有資料。**

```bash
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/upgrade-prebuilt-termux.sh | bash
```

更新器會先驗證新封包，再保存 `config.json`、帳號、資料庫、私有金鑰與日誌，並保留 `~/AhB.backup-YYYYMMDD-HHMMSS` 備份目錄。完成後可執行：

```bash
cd ~/AhB
./scripts/run-termux.sh
./scripts/doctor-termux.sh
./scripts/smoke.sh
```

如果需要從原始碼建置 Termux 環境：

```bash
chmod +x scripts/*.sh
./scripts/bootstrap-termux.sh
./scripts/run-termux.sh
```

詳細說明：[Termux 部署](docs/DEPLOY_TERMUX.md)、[Android 實機驗收](docs/ANDROID_FINAL_ACCEPTANCE.md)。

## Linux VPS 安裝與更新

支援 **Linux AMD64／ARM64**，使用對應 Linux ELF 封包，請勿使用 Android 的二進位檔。建議使用一般使用者（不要以 root 執行），並準備 `curl`、`jq`、`tar`、`python3` 與 FreeBuff 所需的 Node.js 22。

**首次安裝：**

```bash
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/install-prebuilt-linux.sh -o "$HOME/ahb-linux-install.sh"
bash "$HOME/ahb-linux-install.sh"
cd "$HOME/AhB"
bash scripts/start-linux.sh
curl -fsS http://127.0.0.1:8317/healthz
```

**已安裝版本的升級：**

```bash
cd "$HOME/AhB"
bash scripts/upgrade-linux.sh
bash scripts/start-linux.sh
```

升級流程採分階段下載與驗證，保存帳號、SQLite、設定及原服務備份；若舊程序仍在執行，會拒絕在不安全情況下複製資料。**不要在確認備份可回復之前手動清除備份目錄。**

如要從自己的電腦管理遠端 VPS，請使用 SSH Tunnel，而不是公開 8317 管理埠：

```bash
ssh -N -L 8317:127.0.0.1:8317 USER@VPS
```

然後在電腦瀏覽器開啟 `http://127.0.0.1:8317/ui`。

更多資訊：[512 MiB VPS 部署教學](docs/DEPLOY_VPS_512MB.md)、[記憶體與程序策略](docs/512MB-DEPLOYMENT.md)。

## Docker／Compose（Linux 容器）

使用 Docker Desktop 的 macOS／Windows 可以執行 **Linux 容器版**，但**不代表專案有原生 macOS／Windows 執行檔**。

```bash
git clone https://github.com/Tsenjii/AhB.git
cd AhB
printf 'AHB_PUBLIC_TOKEN=%s\n' "$(openssl rand -hex 32)" > .env
chmod 600 .env
docker compose up -d --build
```

- Docker 控制台：`http://127.0.0.1:8080/ui`，使用 HTTP Basic（帳號 `ahb`；密碼取自私有 `.env` 的 `AHB_PUBLIC_TOKEN`）。
- API：`http://127.0.0.1:8080/v1`，同一 Token 作為 Bearer／`x-api-key` 憑證。
- 預設只繫結 `127.0.0.1`，私有資料保存在 `ahb-data` Volume。**不要使用 `docker compose down -v`，除非你確定要刪除 Volume 裡的資料。**

**Docker 版本提醒：** 為維持可重現的容器建置，目前 `Dockerfile`／`compose.yaml` **預設固定較舊的基準發布版** [`linux-15b2a47dea22`](https://github.com/Tsenjii/AhB/releases/tag/linux-15b2a47dea22)，不是本頁上方最新原生版。要使用其他**已發布**的 Linux 版本，需在私有 `.env` 設定 `AHB_RELEASE_TAG`，再重新建置；請先檢查對應 Release 的資產與 SHA-256。

詳情：[Docker 設定教學](docs/DOCKER.md)。

## 統一 API 與 Playground

AhB 透過明確的 `provider/model` 名稱路由請求，避免將不同帳號與 Gateway 的模型混淆。

```text
http://127.0.0.1:8317/v1

opencode/<model>
freebuff/<model>
agent2api/<model>
grok/<model>
kiro/<model>
copilot/<model>
geminiweb/<model>
deepseek/<model>    # 實驗版，預設停用
duckai/<model>      # 實驗版，預設停用
route/<alias>       # 必須自行設定並啟用路由別名
```

**目前轉送的 API 路徑：**

| 方法 | 路徑 |
| --- | --- |
| GET | `/v1/models` |
| POST | `/v1/chat/completions`、`/v1/completions` |
| POST | `/v1/responses` |
| POST | `/v1/messages`、`/v1/messages/count_tokens` |
| POST | `/v1/embeddings` |
| POST | `/v1/images/generations`、`/v1/audio/speech` |
| POST | `/v1/systemone` |

**支援路徑不代表所有 Gateway 都支援該能力。** 圖片／語音等請求採 JSON 轉送，仍要由上游實際實作；工具呼叫、Responses、Anthropic 相容性及 SSE 必須逐一實測。

在 [Playground](http://127.0.0.1:8317/playground) 指定單一 Gateway 與模型，可測試一般對話、串流、HTTP 狀態及部分工具格式。Playground **不會偷偷改用另一個 Gateway**；正式的多輪工具調用需要另外驗證工具結果回傳。

**模型清單與節省資源：**

- `GET /v1/models` 不會為了列出模型，就把所有睡眠中的 Gateway 一次喚醒。
- 控制台的「逐一載入全部模型」會依序喚醒並查詢各來源，再釋放程序。
- 已成功取得的模型目錄可在本機私有快取中保留最多 6 小時。快取資料會標記 `x_cached: true`，**不代表目前帳號或模型真的可使用**。
- 512 MiB 模式採全 Hub 共用的模型查詢並行上限，避免多個手機／電腦頁面同時刷新時占用過多資源。

## 帳號登入與原生管理介面

AhB 以整合**上游原有登入機制**為原則，不會把不同 Gateway 的管理台假裝成同一套通用帳密。

| Gateway | 原本的管理位置或設定方式 |
| --- | --- |
| OpenCode | `http://127.0.0.1:8404/` |
| Agent2API | `http://127.0.0.1:8403/` |
| Grok | `http://127.0.0.1:8407/` |
| Kiro | `http://127.0.0.1:8408/admin` |
| FreeBuff | 新版 Node Gateway 沒有舊版 Rust 的 `/ui` |
| DeepSeek Web | **DeepSeek Web: no native admin WebUI**；實驗版沒有舊 `/admin`，須使用私有且已授權的帳號檔案 |
| Copilot | 需執行 GitHub Device Flow，並具備有效資格 |

### FreeBuff（新版 Node Gateway）

舊 Rust FreeBuff Gateway 已換為 [yutian81/freebuff2api](https://github.com/yutian81/freebuff2api)。**舊 Web Cookie 不能自動轉成新版 CLI Token**，但升級不會刻意清除原本備份中的資料。

在 Android 控制台進入**帳號與登入 → FreeBuff**，使用官方 Codebuff CLI 的瀏覽器授權流程，或使用：

```bash
cd ~/AhB
./scripts/check-freebuff-login.sh
./scripts/freebuff-login-termux.sh
./scripts/stop-termux.sh && ./scripts/start-termux.sh
```

最後用 `./scripts/provider-status-termux.sh` 檢查設定狀態。它不會列印個別帳號 Token，也不會消耗推論額度；**UNKNOWN／HEALTHY 都不等於可用餘額**。

詳細教學：[FreeBuff 登入與遷移](docs/FREEBUFF_ANDROID_LOGIN.md)。

### 其他 Gateway 初始化

```bash
cd ~/AhB

# 完成 Grok 私有管理設定與本機 Client Key
./scripts/enable-grok2api.sh

# 完成 Kiro-Go 管理設定與帳號初始化
./scripts/enable-kiro-go.sh

# GitHub Copilot：依官方 Device Flow 完成授權後再啟用
./scripts/login-copilot2api.sh
./scripts/enable-copilot2api.sh
```

**DeepSeek Web：** 新實驗版 [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api) 為 Go 實作，沒有舊版 `/admin` 或通用自動 OAuth。必須先將自己合法取得的帳號憑證放進權限 `0600` 的 `data/deepseek2api/accounts.txt`（每行一組），再視需要執行 `./scripts/enable-deepseek2api.sh`。上游模型能否真正推論仍未實測通過。

**Duck.ai：** 目前是 [Rust HTTP-only 實驗版](https://github.com/desktop-tools-which-may-be-useful/duckai2api)，預設停用；觀察到的 HTTP 418 問題不能靠本機 `/health` 成功來宣稱修復。

來源限制與待測項目：[DeepSeek／Duck.ai 現況](docs/DEEPSEEK_DUCK_AVAILABILITY_2026-10-10.md)、[Gateway 穩定性](docs/PROVIDER_STABILITY_2026-10-09.md)。

## 連接其他已安裝的本機 Gateway

AhB 的 `connect-bridge.sh` 可以把你**自行部署、已經授權且正在運行**的 OpenAI 相容 API 接到統一路由，提供可選的本機 Key 設定與模型目錄驗證。

```bash
cd ~/AhB
./scripts/connect-bridge.sh --list
./scripts/connect-bridge.sh lmarena http://127.0.0.1:5102
./scripts/connect-bridge.sh windsurf http://127.0.0.1:3003
./scripts/connect-bridge.sh kimi http://127.0.0.1:8000
./scripts/connect-bridge.sh gemini http://127.0.0.1:5918
./scripts/connect-bridge.sh cliproxy http://127.0.0.1:8416
```

以上是**連線預設，不是已內建的上游程式**。例如 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 必須另行安裝與設定；若其預設使用 8317，會與 AhB 衝突，請先將其綁在其他本機連接埠（範例為 8416）。

Kimi Web 有額外的歷史 Termux Python 安裝方案，但不在基本封包內：

```bash
./scripts/install-kimiweb-termux.sh
./scripts/enable-kimiweb-termux.sh
```

其他外部橋接器、`codex` 等預設與限制：[外部 Gateway 接入說明](docs/BRIDGES.md)。專案不提供 CAPTCHA 繞過、批量註冊或規避上游帳號限制的功能。

## 健康檢查、記憶體與除錯

```bash
cd ~/AhB

# 本機 Hub 健康狀態
curl -fsS http://127.0.0.1:8317/healthz

# Hub / Gateway 記憶體、常駐數量及系統 RAM
curl -fsS http://127.0.0.1:8317/api/runtime

# Android：九個來源的非敏感狀態
./scripts/provider-status-termux.sh

# Android：環境與安裝健檢
./scripts/doctor-termux.sh

# 原生 Linux：查看近期 Hub 訊息
tail -n 40 logs/hubd.log
```

- Gateway 健康檢查不會因為一次錯誤就任意中斷有活動租約的 SSE 請求。
- 上游如果長時間沒有送出 HTTP 回應標頭，最多等待 120 秒；送出標頭後，長時間 SSE 可繼續傳送。
- 每個由 AhB 管理的 Gateway 日誌最多約 **4 MiB**，避免無限制增加磁碟占用。
- 模型快取只保存限定的非敏感中繼資料，不保存帳號 Token、訊息內容或任意上游欄位。
- 需要真正確認工具相容性時，可使用下列雙輪測試（**會消耗上游額度**）：

```bash
AIHUB_TEST_MODEL='opencode/YOUR_MODEL_ID' ./scripts/test-tool-roundtrip.sh
```

如果服務顯示 HEALTHY，但聊天回應 HTTP 503、418 或工具呼叫失敗，請先檢查**原始 Gateway 是否有有效帳號、可用額度與真實回應**，不要只看本機程序是否仍在運行。

## 安全與資料保留

1. **預設只監聽 `127.0.0.1`。** 不要將 Hub 控制台、原始管理頁、帳號資料庫或 8317 直接公開到 Internet。
2. 帳號 Cookie、Token、管理密碼與 `.env` 都應存放在私有位置；**不要提交 GitHub，也不要直接貼到 Issue 或聊天紀錄**。
3. 更新前應保留自動建立的版本備份；Linux、Android 的安裝包及重啟流程彼此獨立。
4. 啟動某個 Gateway 不代表你已有它的授權；模型、額度與權限仍以實際來源帳號為準。

## 更多文件

- [系統架構](docs/ARCHITECTURE.md)
- [所有來源與上游管理方式](docs/PROVIDERS.md)
- [九個上游來源版本稽核](docs/UPSTREAM_AUDIT_2026-10-11.md)
- [API 規格](docs/V1_SPEC.md)
- [外部 Gateway 接入](docs/BRIDGES.md)
- [Android 部署](docs/DEPLOY_TERMUX.md)
- [Linux 512 MiB VPS 部署](docs/DEPLOY_VPS_512MB.md)
- [Docker／Compose](docs/DOCKER.md)
- [Android 實機驗收](docs/ANDROID_FINAL_ACCEPTANCE.md)
- [故障排除與網路出口](docs/NETWORK_EGRESS.md)
- [開發與交接紀錄](docs/NEXT_AI_HANDOFF.md)

---

本專案以**繁體中文**為主要說明語言；API 路徑、設定鍵、專有名詞及終端機指令保留原始英文，以利直接複製使用。
