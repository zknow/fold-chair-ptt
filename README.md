# Fold Chair PTT

Windows 11 專用的 PTT 推文監控桌面工具。貼上文章網址，即可開啟永遠置頂的小視窗，即時追蹤新推文。

## 功能特色

- **即時監控**：透過 PTT 長輪詢機制，自動取得新推文。
- **永遠置頂**：可自由切換視窗置頂狀態。
- **自動捲動**：自動顯示最新推文，往上捲動即可回看歷史訊息。
- **新推文累積**：瀏覽舊訊息時，新推文會暫存在「回到底部」按鈕，返回底部後再補上。
- **自訂設定**：支援淺色／深色主題、字級、輪詢間隔及訊息保留數量。
- **設定持久化**：設定自動儲存，重新啟動後沿用上次設定；不保留上次監控的文章。

## 系統需求

- Windows 11
- WebView2 Runtime（Windows 11 已內建）

## 使用方式

1. 啟動 `fold-chair-ptt.exe`。
2. 貼上 PTT 文章網址（支援 `www.ptt.cc` 與 `m.ptt.cc`），按 Enter 或點擊「開始監控」。
3. 推文會自動更新。往上捲動可瀏覽舊訊息，點擊「回到底部」查看累積的新推文。

視窗操作：
- 📌 切換永遠置頂
- ⚙ 開啟設定
- ✕ 關閉視窗
- 拖曳標題列移動視窗

設定儲存位置：`%APPDATA%\fold-chair-ptt\config.json`

## 建置方式

目前尚未提供安裝程式或 Release，需自行建置。

**環境需求：**
- Windows 11
- Go 1.26+
- [Wails CLI v2](https://wails.io)

安裝 Wails CLI 並執行建置：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build
```

建置完成後，執行檔位於：

```text
build/bin/fold-chair-ptt.exe
```

> 請使用 `wails build`，不要使用 `go build`，以確保圖示、版本資訊及 Windows manifest 正確打包。

## 開發與測試

執行單元測試（離線）：

```powershell
go test ./...
```

連線真實 PTT 文章進行測試：

```powershell
$env:FCP_LIVE="1"
go test ./internal/ptt/ -run TestLiveArticle -v
```

## 技術架構

- **桌面框架**：[Wails v2](https://wails.io)，使用 WebView2。
- **前端**：純 HTML、CSS、JavaScript，透過 `go:embed` 嵌入執行檔，無需 Node.js。
- **文章解析**：[goquery](https://github.com/PuerkitoBio/goquery)。
- **即時更新**：使用 PTT 長輪詢，從文章頁面的 `data-longpollurl` 與 `data-pollurl` 取得輪詢端點。
- **錯誤處理**：長輪詢遇到 HTTP 錯誤時立即重新抓取文章；其他錯誤累計 5 次後重新抓取，並補回中斷期間遺漏的推文。