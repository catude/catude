# catude glossary: Traditional Chinese, Taiwan usage (zh-Hant)

Output: `internal/i18n/locales/zh-Hant.json` (197 messages). Written from the Taiwan sources below, not converted from zh-Hans: 工作階段 (not 會話), 設定檔 (not 配置文件), 檔案, 資料夾, 專案, 外掛程式, 市集, 伺服器, 索引標籤, 說明, 結束, 偵錯, 主控台, 處理序, 轉譯, 預設值, 佈景主題, 按一下, 選取, 依, 清單, 本機, 全域, 殼層, 作用中.

Sources (fetched 2026-11-18):
- **CC** = Anthropic's Claude Code docs in Traditional Chinese, `https://code.claude.com/docs/zh-TW/<page>` (read as `<page>.md`). Pages used: skills, sub-agents, hooks, mcp, plugins, discover-plugins, plugin-marketplaces, commands, interactive-mode, settings, model-config, context-window, costs, memory, terminal-config.
- **VS** = VS Code zh-hant language pack, `https://raw.githubusercontent.com/microsoft/vscode-loc/main/i18n/vscode-language-pack-zh-hant/translations/main.i18n.json` (cited as `file | key`).
- **LG** = lazygit `https://raw.githubusercontent.com/jesseduffield/lazygit/master/pkg/i18n/translations/zh-TW.json` (cited as key).

Address form: 您, as VS zh-hant (`chatSetupRunner | settings`: "一旦繼續，即表示您同意…") and Anthropic's zh-TW docs.

## Claude concepts (source 1: Anthropic)

| English | zh-Hant | Source | Exact string seen |
|---|---|---|---|
| Skills | Skills (kept in English) | CC /zh-TW/skills | "# 使用 skills 擴展 Claude" |
| Agents | 代理 | CC /zh-TW/discover-plugins | "安裝外掛程式會將其技能、代理、hooks 和 MCP 伺服器新增到您機器" |
| subagent | 子代理 | CC /zh-TW/context-window | "以及子代理如何將大型讀取保持在上下文之外" (title is "# 建立自訂 subagents") |
| Hooks | Hooks (kept in English) | CC /zh-TW/hooks | "# Hooks 參考" |
| Commands | 命令 | CC /zh-TW/skills | "請參閱[命令參考]" |
| MCP servers | MCP 伺服器 | CC /zh-TW/mcp | "## 安裝 MCP 伺服器" |
| Plugins | 外掛程式 | CC /zh-TW/discover-plugins | "# 安裝和管理外掛程式" |
| marketplaces | 市集 | CC /zh-TW/discover-plugins | "從任何使用介面上的市集安裝 Claude Code 外掛程式" |
| session | 工作階段 | CC /zh-TW/commands | "`/tasks` 列出目前工作階段的背景工作" |
| conversation | 對話 | CC /zh-TW/commands | "使用 `/resume` 恢復先前的對話" |
| resume | 恢復 | CC /zh-TW/commands | same as above |
| export (conversation) | 匯出 | CC /zh-TW/commands | "`/export [filename]` \| 將目前對話匯出為純文字" |
| context / context window | 上下文 | CC /zh-TW/context-window | "# 探索上下文視窗" |
| transcript | 逐字稿 | CC /zh-TW/commands | "掃描您的逐字稿以查找常見的唯讀 Bash 和 MCP 工具呼叫" |
| Usage | 用量 | CC /zh-TW/commands | "`/usage` \| 顯示工作階段成本、方案用量上限和活動統計" |
| in the background / background jobs | 在背景 / 背景工作 | CC /zh-TW/commands | "列出目前工作階段的背景工作"; "並在背景運行" |
| model / Models | 模型 | CC /zh-TW/model-config | "# 模型設定" |
| settings / configuration | 設定 | CC /zh-TW/settings | "# 設定檔案和優先順序" |
| terminal | 終端機 | CC /zh-TW/terminal-config | "# 為 Claude Code 設定您的終端機" |
| project | 專案 | CC /zh-TW/memory | "# Claude 如何記住您的專案" |
| Tools | 工具 | CC /zh-TW/mcp | "# 透過 MCP 將 Claude Code 連接到工具" |
| theme | 佈景主題 | CC /zh-TW/settings | "個人偏好設定：佈景主題、編輯器模式、預設模型" |
| debug | 偵錯 | CC /zh-TW/commands | "為目前工作階段啟用偵錯日誌記錄" |

## General interface words (source 2: VS Code)

| English | zh-Hant | Source | Exact string seen |
|---|---|---|---|
| Preferences | 喜好設定 | VS `actionCommonCategories \| preferences` | "喜好設定" |
| Settings / Config | 設定 | VS `preferences.contribution \| settings`; `layoutActions \| configureTabs` | "設定"; "設定索引標籤" |
| Help | 說明 | VS `actionCommonCategories \| help` | "說明" |
| quit | 結束 | VS `menubar \| quit`; LG `Quit` | "結束(&&Q)"; "結束" |
| tab(s) | 索引標籤 | VS `layoutActions \| tabBar`; LG `NextTab` | "索引標籤列"; "下一個索引標籤" |
| filter | 篩選 | VS `abstractTree \| filter`; LG `FilterPrefix` | "篩選"; "篩選：" |
| search | 搜尋 | VS `copilotToolDisplay \| toolName.search`; LG `SearchTitle` | "搜尋" |
| find | 尋找 | VS `findController \| findNextMatchAction` | "尋找下一個" |
| open / open folder | 開啟 / 開啟資料夾 | VS `dialogMainService \| openFolder` | "開啟資料夾" |
| folder | 資料夾 | VS `languages \| suggestWidget.kind.folder` | "資料夾" |
| file | 檔案 | LG `OpenFile`, `EditFile` | "開啟檔案", "編輯檔案" |
| close | 關閉 | VS `abstractTree \| close` | "關閉" |
| refresh | 重新整理 | VS `treeView \| refresh`; LG `Refresh` | "重新整理" |
| reload | 重新載入 | VS `workspaceWatcher \| reload` | "重新載入" |
| load / loading | 載入 | VS `multiDiffEditorWidgetImpl \| loading` | "正在載入..." |
| scroll | 捲動 | LG `ScrollDown`; VS `editorOptions \| smoothScrolling` | "向下捲動"; "控制編輯器是否會使用動畫捲動" |
| top / bottom / page | 頂部 / 底部 / 翻頁 | LG `GotoTop`, `GotoBottom`, `NextPage` | "捲動到頂部", "捲動到底部", "下一頁" |
| expand / collapse all | 全部展開 / 全部摺疊 | VS `comments.contribution \| expandAll`, `treeView \| collapseAll` | "全部展開", "全部摺疊" |
| install / upgrade | 安裝 / 升級 | VS `installActions \| install`, `chatActions \| upgradeChat` | "在 PATH 中安裝 '{0}' 命令", "升級 GitHub Copilot 方案" |
| delete / remove | 刪除 / 移除 | VS `aiCustomizationTreeView.contribution \| delete`; VS `developerActions \| removeLargeStorageDatabaseEntries` | "刪除"; "移除大型儲存體資料庫項目..." |
| restore / restorable | 還原 | VS `fileActions \| restore` | "您可以使用復原命令來還原此檔案。" |
| trash | 資源回收筒 | VS `fileActions \| deleteButtonLabelRecycleBin` | "移至資源回收筒(&&M)" (macOS variant: "移至垃圾筒") |
| default (value) | 預設值 | VS `layoutActions \| restore defaults` | "還原預設值" |
| preset | 預設 | VS `browserEditorEmulationFeatures \| browser.emulationToolbar.presets` | "套用預設..." |
| profile | 設定檔 | VS `developerActions \| profile`; `userDataProfile \| switchProfile` | "設定檔"; "切換設定檔..." |
| new (item) | 新增 | VS `desktop.contribution \| newTab` | "新增視窗索引標籤" |
| switch | 切換 | VS `userDataProfile \| switchProfile` | "切換設定檔..." |
| verify | 驗證 | VS `copilotToolDisplay \| toolName.mcpValidate` | "驗證 MCP 設定" |
| mark | 標記 | VS `sessionsViewActions \| markRead` | "標記為已讀取" |
| export | 匯出 | VS `userDataProfilesEditorModel \| export` | "匯出..." |
| Debug / debug panel | 偵錯 / 偵錯面板 | VS `log \| debug`; `debug.contribution \| debugPanel` | "偵錯"; "偵錯主控台" |
| console | 主控台 | VS `browserEditorChatFeatures \| consoleLogs` | "主控台記錄" |
| Accounts | 帳戶 | VS `globalCompositeBar \| accounts` | "帳戶" |
| Overview | 概觀 | VS `customizationsToolbar.contribution \| overview` | "概觀" |
| Security | 安全性 | VS `settingsLayout \| security` | "安全性" |
| Available | 可用 | VS `localizationsActions \| available` | "可用" |
| Sources | 來源 | VS `extensionEditor \| source` | "來源" |
| Graph | 圖表 | VS `scm.contribution \| scmGraph` | "圖表" |
| node | 節點 | VS `chatDebugFlowChartView \| chatDebug.flowChart.showMoreTitle` | "載入更多節點" |
| Live | 即時 | VS `codexAgent \| codex.sessionConfig.webSearchMode.live` | "即時" |
| Discover | 探索 | VS `agentHostChatDebugProvider \| agentHost.debug.agentDiscovery` | "代理程式探索" |
| health | 健康情況 | VS `chatDebugCacheExplorerView \| chatDebug.cache.sessionHealth` | "工作階段快取健康情況" |
| Theme | 佈景主題 | VS `themes.contribution \| miSelectTheme` | "佈景主題(&&T)" |
| light / dark / high contrast | 淺色 / 深色 / 高對比 | VS `themes.contribution \| themes.category.light`, `theme_picker \| HighContrast` | "淺色主題", "深色主題", "深色高對比" |
| Language | 語言 | VS `configurationExtensionPoint \| language` | "語言" |
| current (now: %s) | 目前 | VS `languagePacks \| currentDisplayLanguage` | " (目前)" |
| auto | 自動 | VS `codexAgent \| codex.sessionConfig.reasoningSummary.auto` | "自動" |
| on / off | 開啟 / 關閉 | VS `agentHostSessionConfigPicker \| …boolean.onLabel`, `log \| off` | "開啟", "關閉" |
| hidden / shown | 隱藏 / 顯示 | VS `toolbar \| hide`; `layoutActions \| sidebarVisible` | "隱藏"; "顯示的主要側邊欄" |
| lite (view) | 精簡 | VS `activitybarPart \| activityBarSizeCompact`; `auxiliaryEditorPart \| enableCompactAuxiliaryWindow` | "精簡"; "開啟精簡模式" |
| full | 完整 | VS `agentHostChatDebugProvider \| agentHost.debug.fullPrompt` | "完整提示" |
| View | 檢視 | VS `menubar \| mView` | "檢視(&&V)" |
| Glyphs | 字符 | VS `commentGlyphWidget \| editorGutterCommentGlyphForeground` | "用於註解字符的編輯器裝訂邊裝飾色彩。" (VS also has "繪製字元" for drawGlyph) |
| characters | 字元 | VS `contextmenu \| context.minimap.renderCharacters` | "轉譯字元" |
| font | 字型 | VS `settingsLayout \| font` | "字型" |
| Animations | 動畫 | VS `chat.shared.contribution \| chat.upvoteAnimation.off` | "未顯示動畫。" |
| keyboard shortcuts | 快速鍵 / 按鍵 | VS `helpActions \| keybindingsReference` | "鍵盤快速鍵參考" (LG `Keybindings`: "鍵盤快捷鍵"; VS preferred) |
| highlight | 反白顯示 | VS `colorThemeSchema \| schema.colors` | "反白顯示語法時的色彩" |
| selection / selected | 選取範圍 / 選取的 | VS `diffEditor.contribution \| revertSelection`; `extensionsActions \| select color theme` | "還原選取範圍"; "選取色彩佈景主題" |
| click | 按一下 | VS `foldingDecorations \| linesCollapsed` | "按一下以展開範圍。" |
| colour | 色彩 | VS `colors \| multiDiffEditor.border` | "多重檔案 Diff 編輯器的框線色彩" |
| blue / green / red | 藍色 / 綠色 / 紅色 | VS `chartsColors \| chartsBlue` etc. | "圖表視覺效果中所使用的藍色。" |
| browser | 瀏覽器 | VS `remoteAgentService \| directUrl` | "在瀏覽器中開啟" |
| editor | 編輯器 | VS `screenReaderSupport \| editor` | "編輯器" |
| terminal | 終端機 | VS `chatContext \| terminal` | "終端機" |
| process (OS) | 處理序 | VS `terminalTooltip \| shellProcessTooltip.processId` | "處理序識別碼 ({0}): {1}" |
| render | 轉譯 | VS `chatOutputItemRenderer \| chatOutputRenderer.viewType` | "轉譯器的唯一識別碼。" |
| Storage | 儲存體 | VS `developerActions \| logStorage` | "記錄儲存體資料庫內容" |
| backup / this machine | 備份 / 本機 | VS `userDataSync.contribution \| Open Backup folder` | "開啟本機備份資料夾" |
| Everywhere (global keys) | 全域 | VS `browserDataStorageFeatures \| browser.clearGlobalStorageAction` | "清除儲存體 (全域)" |
| shell | 殼層 | VS `agentHostCustomizationConfig \| agentHost.config.defaultShell.title` | "預設殼層" |
| integration | 整合 | VS `browserTabManagementFeatures \| browser.openFileAction` | "在整合瀏覽器中開啟" |
| active | 作用中 | VS `contextkeys \| activeEditorIsReadonly` | "作用中編輯器是否為唯讀" |
| list | 清單 | VS `notificationsList \| notificationsList` | "通知清單" |
| by (model/project) | 依 | VS `sessionsView \| groupByTime` | "依時間分組" |
| ages %ds %dm %dh %dd | %d 秒 / %d 分鐘 / %d 小時 / %d 天 | VS `date \| date.fromNow.seconds.singular` … | "{0} 秒", "{0} 分鐘", "{0} 小時", "{0} 天"; "{0} 小時前" for "%s ago" |

## Key-hint phrasing (source 3: lazygit)

| English | zh-Hant | Source | Exact string seen |
|---|---|---|---|
| quit | 結束 | LG `Quit` | "結束" |
| close | 關閉 | LG `Close` | "關閉" |
| edit file | 編輯檔案 | LG `EditFile` | "編輯檔案" |
| return / back | 返回 | LG `BackToParentRepo` | "返回父倉庫" |
| next / previous tab | 下一個 / 上一個索引標籤 | LG `NextTab`, `PrevTab` | "下一個索引標籤", "上一個索引標籤" |
| confirm / cancel | 確認 / 取消 | LG `Confirm`, `Cancel` | "確認", "取消" |

## No source: translated by analogy

| English | zh-Hant | Note |
|---|---|---|
| work profile | 工作設定檔 | catude concept, built from VS 設定檔 |
| Accent colour | 強調色 | not in VS; term used by macOS zh-TW (Windows zh-TW says 輔色) |
| Audit | 稽核 | not in VS; standard Taiwan IT term |
| Inventory | 清單 | heading listing item counts |
| Process (a practitioner's working process, gallery) | 流程 | not the OS process |
| measure (context/storage) | 測量 | — |
| kind of work: coding, reasoning, math & science, knowledge | 程式設計、推理、數學與科學、知識 | benchmark categories; 程式設計 is the Taiwan term for programming |
| clean (safe item) | 清理 | — |
| pinned files | 固定版本的檔案 | — |
| documented (setting) | 已記載 | — |
| practitioners | 從業人員 | — |
| to-do | 待辦 | VS `claudeToolDisplay \| claude.tool.todoWrite` has "更新待辦事項清單" |
| amber / cyan / magenta | 琥珀色 / 青色 / 洋紅色 | VS has only blue/green/red/yellow/orange/purple chart colours |
| mono (theme) | 單色 | — |
| spinners | 旋轉圖示 | 圖示 = icon in VS zh-hant ("搜尋圖示") |
| diagram (Mermaid) | 示意圖 | distinguishes it from 圖表 (Graph tab) |
| rebuild | 重建 | — |
| new value for | 新值 | sentence continues with the setting key, so kept as a short label |

## Where sources disagreed

| Term | Options seen | Chosen |
|---|---|---|
| Agents | CC 代理; VS 代理程式 (`customizationsToolbar.contribution \| agents`) | 代理 (Anthropic first for Claude concepts) |
| subagent | CC zh-TW uses both "subagent" (739×) and 子代理 (212×) | 子代理 (readable in a one-word label; same as zh-Hans) |
| Skills | CC title "使用 skills 擴展 Claude", prose also 技能 (269× vs 382× skills) | Skills, as in the page title |
| Preset vs default | VS uses 預設 for both "preset" ("套用預設...") and "default" ("預設殼層") | 預設 for Presets, 預設值 for default values |
| Theme | VS menu "佈景主題(&&T)", but "深色主題"/"淺色主題"; CC settings "佈景主題" | 佈景主題 for Theme, 淺色/深色 for values |
| Glyph | VS "字符" (gutter glyph) and "字元" (drawGlyph) | 字符; 字元 kept for "characters" |
| Keyboard shortcuts | VS "鍵盤快速鍵"; LG "鍵盤快捷鍵" | 快速鍵 (VS, Taiwan Microsoft term) |

## Added 2026-11-19 (Agents tab, token drain, switching off per project)

| English | zh-Hant | Source | Exact string seen |
|---|---|---|---|
| token / tokens | token (kept in English) | CC /zh-TW/costs | "Claude Code 按 API token 消耗量計費"; "減少 token 使用" |
| prompt cache / cache | 提示快取 / 快取 | CC /zh-TW/prompt-caching | "提示快取是 API 避免重新處理未變更部分的方式" |
| invalidate the cache | 使快取失效 | CC /zh-TW/prompt-caching | "## 使快取失效的動作" |
| cache reads / writes | 快取讀取 / 快取寫入 | CC /zh-TW/prompt-caching | "該請求會從快取讀取您的前綴" (名詞形式 by analogy with zh-CN 缓存写入/读取) |
| (cache) expires | 過期 | CC /zh-TW/prompt-caching | "快取就會過期" |
| compaction | 壓縮 | CC /zh-TW/prompt-caching | "### 壓縮對話" |
| system prompt / prompt | 系統提示詞 / 提示詞 | CC /zh-TW/prompt-caching | "* **系統提示詞**：下一個請求會從快取讀取此層" |
| headless runs (claude -p) | 非互動執行 | CC /zh-TW/headless | "若要以非互動模式執行 Claude Code，請傳遞 `-p`" |
| Agent SDK | Agent SDK (kept) | CC /zh-TW/headless | "使用 Agent SDK 從 CLI、Python 或 Typ…" |
| changelog | 變更記錄 | VS `extensionEditor \| changelog` | "變更記錄" |
| Notification | 通知 | VS `notificationsCenter \| notifications` | "通知" |
| script | 指令碼 | no source checked: standard Taiwan Microsoft term | — |
| run / runs (of a subagent) | 執行 | no source: translated by analogy | — |
| Agent map | 代理圖 | no source: translated by analogy | — |
| built in | 內建 | no source: translated by analogy | — |
| main conversation ("your conversation") | 主對話 | no source: translated by analogy (zh-CN docs: 主对话) | — |

## Kept in English

`Skills`, `Hooks` (as Anthropic's zh-TW docs do), `MCP`, `ascii`, `unicode` (preference values), the format string `" %d%% · %s"`, the filter examples `security, mcp, tdd` (search terms matched against English content), product/command names (catude, ctlite, Claude Code, claude-swap, Windows Terminal, Markdown, claude /context, claude mcp list) and all key names.

## Test workaround

`wheel scrolls · click a tab to open it` → `滾輪捲動 · 按一下索引標籤 (tab) 將其開啟`. The "(tab)" is only there because `TestCatalogsAreComplete` treats any id containing "tab " plus " · " as a key hint and requires "tab" in the translation. Natural form without it: `滾輪捲動 · 按一下索引標籤將其開啟`.
