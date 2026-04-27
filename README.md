# Comic Downloader

一个基于 Go + Playwright 的 Windows 漫画下载器。Win32 前端负责添加和管理任务，任务层使用 Playwright 打开页面，站点解析器负责提取图片 URL，通用资源层负责下载图片并生成缩略图。

## 当前状态

- 浏览器路线：公共 UI 和任务层统一使用 Firefox。
- 已接入站点：`zeri`、`hentai2`。
- 前端站点筛选：任务列表标题旁的下拉菜单支持 `显示全部`、`Zeri`、`Nyahentai`、`Hentai2`、`Hentaiaz`、`Hitomi`。
- `nyahentai`、`hentaiaz`、`hitomi` 当前只完成前端筛选占位和历史任务识别，下载流程尚未接入。
- 暂不支持站点：`myreadingmanga.info`，前端添加任务时会提示 `暂不支持此站点`。
- 下载与缩略图：所有站点统一走 `siteflow/assets` 的下载和缩略图逻辑。
- 便携版：单文件 `dist\portable.exe`，持久数据写入同级 `portable-data\`。
- Firefox 和 Playwright driver 设置成功后，会自动刷新未完成任务中的浏览器相关配置，之前失败的任务可以直接重试。

## 快速开始

运行测试：

```powershell
go test ./...
```

运行 Win32 前端：

```powershell
go run -tags playwright ./cmd/win32-frontend
```

运行任务探针：

```powershell
go run -tags playwright ./cmd/task-probe -url "https://www.zerobywzip.com/..." -headless=false -keep-open=false
```

构建便携版：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_portable.ps1
```

运行便携版：

```powershell
dist\portable.exe
```

## 运行环境

- Go：`go.mod` 声明 `go 1.24.0`。
- 平台：Windows，主前端使用 Win32 API。
- Playwright：通过 `github.com/playwright-community/playwright-go` 调用。
- 默认 Firefox 可执行文件：`C:\Program Files\Mozilla Firefox\firefox.exe`。
- 默认 Playwright 浏览器根目录：`runtime\playwright-browsers\`。
- 默认 Playwright driver 目录：`runtime\playwright-browsers\driver\`。

本地开发时可在前端菜单中设置 Firefox 可执行文件和 Playwright driver 目录。任务启动时会读取保存后的前端状态；如果未完成任务是在设置路径前创建的，设置成功后前端会自动刷新这些任务的浏览器和 driver 字段。

## 常用环境变量

- `COMIC_DOWNLOADER_WORKSPACE_ROOT`：覆盖工作区根目录，便携启动器会把它设为 `portable-data\`。
- `COMIC_DOWNLOADER_RUNTIME_ROOT`：覆盖运行时根目录。
- `COMIC_DOWNLOADER_DOWNLOAD_DIR`：覆盖默认下载目录。
- `COMIC_DOWNLOADER_FRONTEND_STATE_PATH`：覆盖前端设置文件路径。
- `COMIC_DOWNLOADER_STATE_PATH`：覆盖旧版历史/任务状态文件路径。
- `COMIC_FIREFOX_PROFILE_SOURCE_DIR`：覆盖 Firefox 源 profile 目录。

## 运行时目录

普通工作区默认写入 `runtime\`：

- `runtime\tasks\task-<id>\report.json`
- `runtime\logs\`
- `runtime\output\`
- `runtime\thumbnails\task-<id>\thumb.jpg`
- `runtime\browser-profiles\`
- `runtime\frontend_state.json`

便携版默认写入 `portable-data\`：

- `portable-data\tasks\task-<id>\report.json`
- `portable-data\logs\`
- `portable-data\output\`
- `portable-data\thumbnails\`
- `portable-data\browser-profiles\`
- `portable-data\frontend_state.json`
- `portable-data\comic_downloader_state.json`

## 当前任务路径

```text
前端 addPendingTask
-> ui.TodoList.RunImmediately
-> tasks.RunBrowserRequest
-> zeri / hentai2 按 URL 分发
-> siteflow/assets 统一下载和缩略图
-> BrowserRunResult 回填前端任务列表
```

## 文档索引

- [项目审计](docs/PROJECT_AUDIT.md)
- [界面与任务流](docs/INTERFACE_FLOW.md)
- [浏览器 Profile 流程](docs/browser_profile_flow.md)
- [Zeri 流程规则](docs/zeri_flow_rules.md)
- [Hentai2 流程规则](docs/hentai2_flow_rules.md)
- [手工冒烟测试](docs/SMOKE_TESTS.md)
