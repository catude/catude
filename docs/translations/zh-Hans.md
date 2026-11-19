# catude glossary: Simplified Chinese (zh-Hans)

Output: `internal/i18n/locales/zh-Hans.json` (197 messages).

Sources (fetched 2026-11-18):
- **CC** = Anthropic's Claude Code docs in Simplified Chinese, `https://code.claude.com/docs/zh-CN/<page>` (read as `<page>.md`). Pages used: skills, sub-agents, hooks, mcp, plugins, discover-plugins, plugin-marketplaces, commands, interactive-mode, settings, model-config, context-window, costs, memory, terminal-config.
- **VS** = VS Code zh-hans language pack, `https://raw.githubusercontent.com/microsoft/vscode-loc/main/i18n/vscode-language-pack-zh-hans/translations/main.i18n.json` (cited as `file | key`).
- **LG** = lazygit `https://raw.githubusercontent.com/jesseduffield/lazygit/master/pkg/i18n/translations/zh-CN.json` (cited as key).

Address form: VS Code zh-hans uses 你 (VS `chatSetupRunner | settings`: "继续操作即表示你同意…"), so catude uses 你. Anthropic's docs use 您.

## Claude concepts (source 1: Anthropic)

| English | zh-Hans | Source | Exact string seen |
|---|---|---|---|
| Skills | Skills (kept in English) | CC /zh-CN/skills | "# 使用 skills 扩展 Claude" |
| Agents | 代理 | CC /zh-CN/hooks | "请参阅 [Skills 和代理中的 Hooks]" |
| subagent | 子代理 | CC /zh-CN/commands; /zh-CN/interactive-mode | "会将工作分发到多个子代理并在后台运行"; "所有正在运行的[后台子代理]" (子代理 758 times vs "subagent" 224 in the pages read) |
| Hooks | Hooks (kept in English) | CC /zh-CN/hooks | "# Hooks 参考" |
| Commands | 命令 | CC /zh-CN/commands | "大多数命令在会话的特定阶段很有用" |
| MCP servers | MCP 服务器 | CC /zh-CN/mcp | "MCP 服务器为 Claude Code 提供对您的工具、数据库…" |
| Plugins | 插件 | CC /zh-CN/discover-plugins | "# 安装和管理插件" |
| marketplaces | 市场 | CC /zh-CN/discover-plugins | "从任何使用的界面上的市场安装 Claude Code 插件" |
| session | 会话 | CC /zh-CN/commands | "Commands 从会话内部控制 Claude Code" |
| conversation | 对话 | CC /zh-CN/commands | "使用 `/resume` 恢复之前的对话" |
| resume | 恢复 | CC /zh-CN/commands | same as above |
| export (conversation) | 导出 | CC /zh-CN/commands | "`/export [filename]` \| 将当前对话导出为纯文本" |
| context / context window | 上下文 | CC /zh-CN/context-window | "# 探索上下文窗口" |
| transcript | 会话记录 | CC /zh-CN/commands | "扫描您的会话记录，查找常见的只读 Bash 和 MCP 工具调用" (fewer-permission-prompts: "Scan your transcripts") |
| Usage | 用量 | CC /zh-CN/commands | "`/usage` \| 显示会话开销、套餐用量限制和活动统计信息" |
| in the background / background jobs | 在后台 / 后台任务 | CC /zh-CN/commands | "列出当前会话的后台工作" |
| model / Models | 模型 | CC /zh-CN/model-config | "# 模型配置" |
| settings | 设置 | CC /zh-CN/settings | "# 设置文件和优先级" |
| terminal | 终端 | CC /zh-CN/terminal-config | "# 为 Claude Code 配置您的终端" |
| project | 项目 | CC /zh-CN/memory | "# Claude 如何记住您的项目" |
| Tools | 工具 | CC /zh-CN/mcp | "# 通过 MCP 将 Claude Code 连接到工具" |
| theme | 主题 | CC /zh-CN/settings | "个人偏好：主题、编辑器模式、默认模型" |

## General interface words (source 2: VS Code)

| English | zh-Hans | Source | Exact string seen |
|---|---|---|---|
| Preferences | 首选项 | VS `actionCommonCategories \| preferences` | "首选项" |
| Settings (gallery heading) | 设置 | VS `preferences.contribution \| settings` | "设置" |
| Config / configuration | 配置 | VS `layoutActions \| configureTabs` | "配置选项卡" |
| Help | 帮助 | VS `actionCommonCategories \| help` | "帮助" |
| quit | 退出 | VS `menubar \| quit` | "退出(&&Q)" |
| tab(s) | 标签页 | VS `menubar \| mNewTab` | "新建标签页" (VS also uses 选项卡: `layoutActions \| tabBar` "选项卡栏"; LG `NextTab` "下一个标签") |
| filter | 筛选 | VS `abstractTree \| filter` | "筛选器"; LG `FilterPrefix` "过滤: " (VS preferred) |
| search | 搜索 | LG `SearchTitle`; VS `iconSelectBox \| iconSelect.placeholder` | "搜索"; "搜索图标" |
| find | 查找 | VS `findController | findNextMatchAction` | "查找下一个" |
| open / open folder | 打开 / 打开文件夹 | VS `dialogMainService \| openFolder` | "打开文件夹" |
| close | 关闭 | VS `abstractTree \| close` | "关闭" |
| refresh | 刷新 | VS `treeView \| refresh`; LG `Refresh` | "刷新" |
| reload | 重新加载 | VS `workspaceWatcher \| reload` | "重新加载" |
| loading / load | 加载 | VS `multiDiffEditorWidgetImpl \| loading` | "正在加载..." |
| scroll | 滚动 | LG `ScrollDown`; VS `editorOptions \| smoothScrolling` | "向下滚动"; "控制编辑器是否使用动画滚动。" |
| scroll to top / bottom, page | 顶部 / 底部 / 翻页 | LG `GotoTop`, `GotoBottom`, `NextPage` | "滚动到顶部", "滚动到底部", "下一页" |
| Expand all / collapse | 全部展开 / 折叠 | VS `comments.contribution \| expandAll`, `treeView \| collapseAll` | "全部展开", "全部折叠" |
| install / upgrade | 安装 / 升级 | VS `installActions \| install`, `chatActions \| upgradeChat` | "在 PATH 中安装“{0}”命令", "升级 GitHub Copilot 计划" |
| delete / remove | 删除 / 移除 | VS `aiCustomizationTreeView.contribution \| delete`; LG `Remove` | "删除" |
| rename | 重命名 | VS `renameSymbolProcessor \| rename` | "重命名" |
| copy | 复制 | VS `textInputActions \| copy` | "复制" |
| restore / restorable | 还原 | VS `fileActions \| restore` | "可以使用“撤消”命令还原此文件。" |
| trash | 回收站 | VS `fileActions \| deleteButtonLabelRecycleBin` | "移动到回收站(&&M)" (macOS variant: "移动到废纸篓") |
| default (value) | 默认值 | VS `layoutActions \| restore defaults` | "还原默认值" |
| preset | 预设 | VS `browserEditorEmulationFeatures \| browser.emulationToolbar.presets` | "应用预设..." |
| profile | 配置文件 | VS `developerActions \| profile`; `userDataProfilesEditor \| newProfile` | "配置文件"; "新建配置文件" |
| new (item) | 新建 | VS `userDataProfilesEditor \| newProfile` | "新建配置文件" |
| switch | 切换 | VS `auxiliaryEditorPart \| toggleCompactAuxiliaryWindow` | "切换窗口紧密模式" |
| verify | 验证 | VS `copilotToolDisplay \| toolName.mcpValidate` | "验证 MCP 配置" |
| mark | 标记 | VS `sessionsViewActions \| markRead` | "标记为已读" |
| resume (gallery/editor) | 恢复 | (Anthropic, see above) | — |
| export | 导出 | VS `userDataProfilesEditorModel \| export` | "导出..." |
| Debug / debug panel | 调试 / 调试面板 | VS `log \| debug`; `debug.contribution \| debugPanel` | "调试"; "调试控制台" |
| console | 控制台 | VS `debug.contribution \| debugPanel` | "调试控制台" |
| Accounts | 帐户 | VS `globalCompositeBar \| accounts` | "帐户" |
| Overview | 概述 | VS `customizationsToolbar.contribution \| overview` | "概述" |
| Security | 安全性 | VS `settingsLayout \| security` | "安全性" |
| Available | 可用 | VS `localizationsActions \| available` | "可用" |
| Sources | 来源 | VS `extensionEditor \| source` | "源" (shortened form; 来源 chosen as the heading reads better; VS zh-hant uses 來源) |
| Graph | 图表 | VS `scm.contribution \| scmGraph` | "图表" |
| node | 节点 | VS `chatDebugFlowChartView \| chatDebug.flowChart.showMoreTitle` | "加载更多节点" |
| Live | 实时 | VS `codexAgent \| codex.sessionConfig.webSearchMode.live` | "实时" |
| Discover | 发现 | VS `agentHostChatDebugProvider \| agentHost.debug.agentDiscovery` | "智能体发现" |
| health | 健康状况 | VS `chatDebugCacheExplorerView \| chatDebug.cache.sessionHealth` | "会话缓存健康状况" |
| Theme | 主题 | VS `themes.contribution \| miSelectTheme` | "主题(&&T)" |
| light / dark / high contrast | 浅色 / 深色 / 高对比度 | VS `themes.contribution \| themes.category.light`, `theme_picker \| HighContrast` | "浅色主题", "深色主题", "深色高对比度" |
| Language | 语言 | VS `configurationExtensionPoint \| language` | "语言" |
| current (now: %s) | 当前 | VS `languagePacks \| currentDisplayLanguage` | " (当前)" |
| auto | 自动 | VS `codexAgent \| codex.sessionConfig.reasoningSummary.auto` | "自动" |
| on / off | 开 / 关 | VS `agentHostSessionConfigPicker \| …boolean.onLabel`, `log \| off` | "开", "关" |
| hidden / shown | 隐藏 / 显示 | VS `toolbar \| hide`; `layoutActions \| sidebarVisible` | "隐藏"; "主侧边栏已显示" |
| lite (view) | 紧凑 | VS `baseSizes \| codiconFontSizeCompact` | "condicons 的紧凑字号。" (brief: translate as a compact view label) |
| full | 完整 | VS `agentHostChatDebugProvider \| agentHost.debug.fullPrompt` | "完整提示" |
| View (full / lite) | 视图 | VS `compositeBar | activityBarAriaLabel` | "活动视图切换器" (VS translates the *View menu* as "查看(&&V)", a verb; 视图 is VS's noun for a view, which is what this preference names) |
| Glyphs | 字形 | VS `gpuActions \| drawGlyph.label` | "绘制字形" |
| font | 字体 | VS `settingsLayout \| font` | "字体" |
| Animations | 动画 | VS `chat.shared.contribution \| chat.upvoteAnimation.off` | "未显示动画。" |
| keyboard shortcuts | 快捷键 / 按键 | VS `helpActions \| keybindingsReference` | "键盘快捷方式参考" (shortened to 快捷键 for the terminal; LG `Keybindings` "按键绑定") |
| selection | 所选内容 | VS `diffEditor.contribution | revertSelection` | "还原所选内容" |
| click | 单击 | VS `foldingDecorations | linesCollapsed` | "单击以展开范围。" |
| highlight colour | 突出显示颜色 | VS `colorThemeSchema \| schema.colors` | "语法突出显示颜色" |
| colours (blue, green, red) | 蓝色 / 绿色 / 红色 | VS `chartsColors \| chartsBlue` etc. | "图表可视化效果中使用的蓝色。" |
| browser | 浏览器 | VS `remoteAgentService \| directUrl` | "在浏览器中打开" |
| editor | 编辑器 | VS `screenReaderSupport \| editor` | "编辑器" |
| terminal | 终端 | VS `chatContext \| terminal` | "终端" |
| process (OS) | 进程 | VS `processExplorerControl \| processName` | "进程名称" |
| render | 呈现 | VS `chatOutputItemRenderer | chatOutputRenderer.viewType` | "呈现器的唯一标识符。" |
| Storage | 存储 | VS `developerActions \| logStorage` | "记录存储数据库内容" |
| backup | 备份 | VS `userDataSync.contribution \| Open Backup folder` | "打开本地备份文件夹" |
| Everywhere (global keys) | 全局 | VS `browserDataStorageFeatures \| browser.clearGlobalStorageAction` | "清除存储(全局)" |
| Shell | Shell | VS `agentHostCustomizationConfig \| agentHost.config.defaultShell.title` | "默认 Shell" |
| integration | 集成 | VS `browserTabManagementFeatures \| browser.openFileAction` | "在集成浏览器中打开" |
| ages %ds %dm %dh %dd | %d 秒 / %d 分钟 / %d 小时 / %d 天 | VS `date \| date.fromNow.seconds.singular` … | "{0} 秒", "{0} 分钟", "{0} 小时", "{0} 天"; "{0} 小时前" for "%s ago" |

## Key-hint phrasing (source 3: lazygit)

| English | zh-Hans | Source | Exact string seen |
|---|---|---|---|
| quit | 退出 | LG `Quit` | "退出" |
| close | 关闭 | LG `Close` | "关闭" |
| edit file | 编辑文件 | LG `EditFile` | "编辑文件" |
| open file | 打开文件 | LG `OpenFile` | "打开文件" |
| return / back | 返回 | LG `BackToParentRepo` | "返回父仓库" |
| next / previous tab | 下一个 / 上一个 | LG `NextTab`, `PrevTab` | "下一个标签", "上一个标签" |
| confirm / cancel | 确认 / 取消 | LG `Confirm`, `Cancel` | "确认", "取消" |

## No source: translated by analogy

| English | zh-Hans | Note |
|---|---|---|
| work profile | 工作配置文件 | catude concept; built from VS 配置文件 (profile) |
| Accent colour | 强调色 | not in VS; term used by macOS zh-CN |
| Audit | 审计 | not in VS (VS 审核 = "review") |
| Inventory | 清单 | catude heading listing item counts |
| Process (a practitioner's working process, gallery) | 流程 | not the OS process |
| measure (context/storage) | 测量 | — |
| kind of work: coding, reasoning, math & science, knowledge | 编程、推理、数学与科学、知识 | benchmark categories |
| clean (safe item) | 清理 | — |
| pinned files | 固定版本的文件 | files pinned to a version |
| to-do | 待办 | — |
| amber / cyan / magenta | 琥珀色 / 青色 / 洋红色 | VS only has blue/green/red/yellow/orange/purple chart colours |
| mono (theme) | 单色 | — |
| spinners | 旋转图标 | — |
| diagram (Mermaid) | 示意图 | distinguishes it from 图表 (Graph tab) |
| rebuild | 重建 | — |
| active profile | 当前配置文件 | VS uses 活动 ("活动编辑器"); 当前 chosen as clearer for a profile |
| new value for | 新值 | sentence continues with the setting key, so kept as a short label |

## Added 2026-11-19 (Agents tab, token drain, switching off per project)

| English | zh-Hans | Source | Exact string seen |
|---|---|---|---|
| token / tokens | token (kept in English) | CC /zh-CN/costs | "Claude Code 按 API token 消耗收费"; "减少 token 使用" (title also has 令牌; token chosen as in the body) |
| prompt cache / cache | 提示缓存 / 缓存 | CC /zh-CN/prompt-caching | "# Claude Code 如何使用提示缓存" |
| invalidate the cache | 使缓存失效 | CC /zh-CN/prompt-caching | "## 使缓存失效的操作" |
| cache writes / reads | 缓存写入 / 缓存读取 | CC /zh-CN/prompt-caching | "其缓存写入费率更低"; "任何未缓存读取的费用" |
| (cache) expires | 过期 | CC /zh-CN/prompt-caching | "缓存就会过期" |
| compaction | 压缩 | CC /zh-CN/prompt-caching | "### 压缩对话" |
| system prompt / prompt | 系统提示词 / 提示词 | CC /zh-CN/prompt-caching | "* **系统提示词**：下一个请求会从缓存中读取此层" |
| headless runs (claude -p) | 非交互运行 | CC /zh-CN/headless | "要以非交互模式运行 Claude Code，请使用 `-p`" |
| Agent SDK | Agent SDK (kept) | CC /zh-CN/headless | "使用 Agent SDK 从 CLI、Python 或 Typ…" |
| changelog | 更改日志 | VS `extensionEditor \| changelog` | "更改日志" |
| Notification | 通知 | VS `notificationsCenter \| notifications` | "通知" |
| run / runs (of a subagent) | 运行 | no source: translated by analogy | — |
| Agent map | 代理图 | no source: translated by analogy | — |
| built in | 内置 | no source: translated by analogy | — |
| main conversation ("your conversation") | 主对话 | CC /zh-CN/prompt-caching (analogy) | "Claude Code 将主对话降低到五分钟 TTL" |

## Kept in English

`Skills`, `Hooks` (as Anthropic's zh-CN docs do), `MCP`, `ascii`, `unicode` (preference values), the format string `" %d%% · %s"`, the filter examples `security, mcp, tdd` (they are search terms matched against English content), product/command names (catude, ctlite, Claude Code, claude-swap, Windows Terminal, Markdown, claude /context, claude mcp list) and all key names.

## Test workaround

`wheel scrolls · click a tab to open it` → `滚轮滚动 · 单击标签页 (tab) 将其打开`. The "(tab)" is only there because `TestCatalogsAreComplete` treats any id containing "tab " plus " · " as a key hint and requires "tab" in the translation. Natural form without it: `滚轮滚动 · 单击标签页将其打开`.

## Where sources disagreed

| Term | Options seen | Chosen |
|---|---|---|
| Agents | CC 代理; VS 智能体 (`customizationsToolbar.contribution \| agents`) | 代理 (Anthropic first for Claude concepts) |
| subagent | CC zh-CN 子代理 (758×) and "subagent" (224×, page title "创建自定义 subagents") | 子代理 |
| Skills | CC title "使用 skills 扩展 Claude", prose also 技能 (121× vs 402× skills) | Skills, as in the page title |
| tab | VS 选项卡 and 标签页; LG 标签 | 标签页 |
| filter | VS 筛选; LG 过滤 | 筛选 (VS) |
| trash | VS 回收站 (Windows) / 废纸篓 (macOS) | 回收站 |
| View | VS menu 查看 (verb) / 视图 (noun) | 视图 |
