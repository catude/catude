# catude glossary: Japanese (ja)

Localized Claude Code docs exist for Japanese at `https://code.claude.com/docs/ja/…` (fetched as `.md`, 2026-11-18).
Abbreviations used below:

- **CC-ja** = Anthropic Claude Code docs, `https://code.claude.com/docs/ja/<page>`
- **VSC-ja** = VS Code language pack, `https://github.com/microsoft/vscode-loc/blob/main/i18n/vscode-language-pack-ja/translations/main.i18n.json` (module :: key)
- **LG-ja** = lazygit, `https://github.com/jesseduffield/lazygit/blob/master/pkg/i18n/translations/ja.json` (key)

## Claude concepts (source 1: Anthropic docs)

| English | Japanese | Source | Exact string seen |
|---|---|---|---|
| skill / Skills | スキル | CC-ja `costs` | 「スキル、サブエージェント、プラグイン、および個別の MCP サーバーに属性付けされた最近の使用量」 |
| subagent | サブエージェント | CC-ja `costs` | 「メイン会話のみをカバーし、サブエージェントはカバーしません」 |
| agents / Agents | エージェント | VSC-ja customizationsToolbar.contribution :: agents | 「エージェント」 (CC-ja uses エージェント 1058× across pages) |
| hook / Hooks | フック | CC-ja `costs` | 「### フックとスキルに処理をオフロードする」 |
| MCP server | MCP サーバー | CC-ja `costs`; VSC-ja aiCustomizationTreeViewViews :: mcpServers | 「個別の MCP サーバー」; 「MCP サーバー」 |
| plugin / Plugins | プラグイン | CC-ja `plugins` | 「# プラグインの概要」 |
| marketplace | マーケットプレイス | CC-ja `mcp` | 「…でマーケットプレイスを追加してから、インストールを再試行してください」 |
| command / Commands | コマンド | CC-ja `costs` | 「### `/usage` コマンドを使用する」 |
| settings / Settings | 設定 | CC-ja `settings` | 「設定は、Claude Code の動作を変更する JSON キーです。」 |
| session / Sessions | セッション | CC-ja `costs` | 「現在のセッションのトークン使用量を確認します」 |
| conversation | 会話 | CC-ja `costs` | 「メイン会話のみをカバーし…」 |
| transcript | トランスクリプト | CC-ja `hooks` | 「トランスクリプト ファイルは非同期に書き込まれ…」 |
| resume (a session) | 再開 | CC-ja `hooks` | 「セッションを再開するときのモデル復元など」 |
| context / Context | コンテキスト | CC-ja `costs` (299×) | 「各チームメイトは独自のコンテキストウィンドウを実行するため」 |
| model / Models | モデル | CC-ja `costs` | 「Opus がデフォルトモデルとして残されている」 |
| usage / Usage | 使用量 | CC-ja `costs` | 「現在のセッションのトークン使用量を確認します」 |
| audit / Audit | 監査 | CC-ja `memory` | 「デフォルトでは、監査は CLAUDE.md、CLAUDE.local.md、AGENTS.md ファイル…をカバーします」 |
| health (claude mcp list) / check health | ヘルスチェック | CC-ja `mcp` | 「`✔ Connected`、`! Needs authentication`、`✘ Failed to connect` などのヘルスステータスを表示します」 (ヘルスステータス → ヘルスチェック by analogy) |
| to-do (task list of a session) | タスク | CC-ja `interactive-mode` | 「## タスクリスト」「タスクリストは Claude のやることリストです」 |
| reasoning (kind of work) | 推論 | CC-ja `costs` | 「複雑な計画と推論タスクのパフォーマンスを大幅に向上させる」 |
| tools / Tools | ツール | CC-ja `costs`; VSC-ja customizationsToolbar.contribution :: tools | 「追跡ツール」; 「ツール」 |

## General UI words (source 2: VS Code)

| English | Japanese | Source | Exact string seen |
|---|---|---|---|
| Preferences | 基本設定 | VSC-ja actionCommonCategories :: preferences | 「基本設定」 |
| Theme | テーマ | VSC-ja themes.contribution :: themes.category.light / dark | 「ライト テーマ」「ダーク テーマ」 (配色テーマ for selectTheme.label; shortened to テーマ) |
| light / dark / high-contrast | ライト / ダーク / ハイ コントラスト | VSC-ja themes.contribution :: themes.category.light, .dark, .hc | 「ライト テーマ」「ダーク テーマ」「ハイ コントラスト テーマ」 |
| auto | 自動 | VSC-ja scmHistoryViewPane :: auto | 「自動」 |
| off / on | オフ / オン | VSC-ja log :: off | 「オフ」 (オン by symmetry) |
| hidden | 非表示 | VSC-ja editor.contribution :: hidden | 「非表示」 |
| View | 表示 | VSC-ja actionCommonCategories :: view | 「表示」 |
| Search / find | 検索 | VSC-ja search.contribution :: search; localizedStrings :: find | 「検索」 |
| Filter / filter: | フィルター / フィルター: | VSC-ja abstractTree :: filter; LG-ja FilterPrefix | 「フィルター」; 「フィルター: 」 |
| Help | ヘルプ | VSC-ja actionCommonCategories :: help | 「ヘルプ」 |
| Keyboard shortcuts | ショートカット | VSC-ja preferences.contribution :: keyboardShortcuts | 「キーボード ショートカット」 |
| Language | 言語 | VSC-ja configurationExtensionPoint :: language | 「言語」 |
| Accounts | アカウント | VSC-ja globalCompositeBar :: accounts | 「アカウント」 |
| Profiles (work profile) | プロファイル (作業プロファイル) | VSC-ja userDataProfile :: profiles | 「プロファイル」; 作業 + プロファイル = no source: translated by analogy |
| Open folder | フォルダーを開く | VSC-ja dialogMainService :: openFolder | 「フォルダーを開く」 |
| Editor | エディター | VSC-ja textEditor :: editor | 「エディター」 |
| Install | インストール | VSC-ja agentPluginActions :: install | 「インストール」 |
| Delete / remove | 削除 | VSC-ja aiCustomizationTreeView.contribution :: delete; manageDynamicAuthenticationProvidersAction :: remove | 「削除」 |
| Debug | デバッグ | VSC-ja debugCommands :: debug | 「デバッグ」 |
| Security | セキュリティ | VSC-ja settingsLayout :: security | 「セキュリティ」 |
| Overview | 概要 | VSC-ja customizationsToolbar.contribution :: overview | 「概要」 |
| Available | 利用可能 | VSC-ja localizationsActions :: available | 「利用可能」 |
| Default | 既定値 | VSC-ja activitybarPart :: default; layoutActions :: restore defaults | 「既定値」「既定値に戻す」 |
| Expand all / Collapse all | すべて展開 / すべて折りたたむ | VSC-ja comments.contribution :: expandAll; treeView :: collapseAll | 「すべて展開」「すべて折りたたむ」 |
| Reload | 再読み込み | VSC-ja window :: reload | 「再読み込み(&&R)」 |
| Export | エクスポート | VSC-ja userDataProfilesEditorModel :: export | 「エクスポート...」 |
| Trash | ごみ箱 | VSC-ja fileActions :: trashFailed | 「ごみ箱を使用した削除に失敗しました…」 |
| Restore(able) | 復元 | VSC-ja fileActions :: restore | 「…コマンドを使用して復元できます。」 |
| Terminal | ターミナル | VSC-ja terminal.contribution :: terminal | 「ターミナル」 |
| Storage | ストレージ | VSC-ja developerActions :: storageLogDialogMessage | 「ストレージ データベースの内容が…」 |
| machine | コンピューター | VSC-ja developerActions :: machine | 「コンピューター」 |
| Animations | アニメーション | VSC-ja workbench.contribution :: workbench.reduceMotion | 「…アニメーション数を減少させるかどうかを制御します。」 |
| Glyphs | グリフ | VSC-ja editorOptions :: glyphMargin | 「…縦のグリフ余白が表示されるかどうか…」 |
| lite (reduced view) | コンパクト | VSC-ja baseSizes :: codiconFontSizeCompact | 「Codicons 用のコンパクトなフォント サイズ。」 |
| full (view) | フル | no source: translated by analogy (counterpart of コンパクト) | — |
| shell integration | シェルの統合 | VSC-ja terminalInstance :: shellIntegration | 「シェルの統合」 |
| running | 実行中 | VSC-ja mcpListWidget :: running; callStackView :: running | 「実行中」 |
| selection / highlight | 選択範囲 / 強調表示 | VSC-ja editorOptions :: selectionHighlight | 「エディターが選択項目と類似の一致項目を強調表示するかどうか…」 |
| yes / no | はい / いいえ | VSC-ja mainThreadAuthentication :: yes / no | 「はい」「いいえ」 |
| compact ages %ds %dm %dh %dd | %d秒 %d分 %d時間 %d日 | VSC-ja vs/base/common/date :: date.fromNow.seconds.plural etc. | 「{0} 秒」「{0} 分」「{0} 時間」「{0} 日」「{0} 分前」 (space dropped for the compact form) |
| %s ago | %s前 | VSC-ja date :: date.fromNow.minutes.plural.ago | 「{0} 分前」 |

## Terminal-UI key hints (source 3: lazygit)

| English | Japanese | Source | Exact string seen |
|---|---|---|---|
| close | 閉じる | LG-ja Close; VSC-ja abstractTree :: close | 「閉じる」 |
| quit | 終了 | LG-ja Quit; VSC-ja menubar :: quit | 「終了」「終了(&&Q)」 |
| edit | 編集 | LG-ja Edit | 「編集」 |
| refresh | 更新 | LG-ja Refresh | 「更新」 |
| scroll | スクロール | LG-ja ScrollDown | 「下にスクロール」 |
| previous / next tab | 前のタブ / 次のタブ | LG-ja PrevTab / NextTab | 「前のタブ」「次のタブ」 |
| page, top, bottom | ページ、先頭、末尾 | LG-ja PrevPage, GotoTop, GotoBottom | 「前のページ」「先頭にスクロール」「末尾にスクロール」 |
| cancel | キャンセル | LG-ja Cancel | 「キャンセル」 |

## No source: translated by analogy

| English | Japanese | Note |
|---|---|---|
| work profile | 作業プロファイル | VS Code プロファイル + 作業 |
| Presets | プリセット | common katakana loanword |
| Discover (tab) | 見つける | no equivalent in sources |
| Live (tab) / Live sessions | ライブ / ライブ セッション | VS Code-style spacing (エージェント セッション) |
| Inventory | インベントリ | — |
| Graph / node / diagram | グラフ / ノード / 図 | — |
| Accent colour | アクセント カラー | Windows term; not in VS Code pack |
| amber / blue / cyan / green / magenta / red | アンバー / 青 / シアン / 緑 / マゼンタ / 赤 | — |
| mono (theme) | モノクロ | — |
| safe (glyph set) | 安全 | — |
| Config (tab) / configuration / setup | 構成 | VS Code 構成 (e.g. 「ツールの構成」 in chatSlashCommands :: tools) |
| Sources (references) | 出典 | — |
| Process (steps) | プロセス | — |
| Key hints / Keys | キーのヒント / キー | — |
| Everywhere (help section) | 全画面共通 | — |
| kind of work; coding; math & science; knowledge | 作業の種類; コーディング; 数学と科学; 知識 | benchmark category labels |
| waiting for you / working | 入力待ち / 作業中 | — |
| verify | 検証 | — |
| mark | マーク | — |
| clean (safe item) | クリーンアップ | — |
| measure | 測定 | — |
| rebuild | 再構築 | — |
| spinners | スピナー | — |

## Added 2026-11-19 (Agents tab, token drain, switching off per project)

| English | Japanese | Source | Exact string seen |
|---|---|---|---|
| token / tokens | トークン | CC-ja `costs`, `prompt-caching` | 「現在のセッションのトークン使用量を確認します」 |
| prompt cache / cache | プロンプトキャッシュ / キャッシュ | CC-ja `prompt-caching` | 「# Claude Code がプロンプトキャッシュを使用する方法」 |
| invalidate the cache | キャッシュを無効化 | CC-ja `prompt-caching` | 「## キャッシュを無効化するアクション」 |
| cache writes / reads | キャッシュ書き込み / キャッシュ読み取り | CC-ja `prompt-caching` | 「キャッシュ書き込みのレートがより低い 5 分の TTL」 (読み取り by analogy) |
| (cache) expires | 期限切れになる | CC-ja `prompt-caching` | 「その時間が経過すると、キャッシュは期限切れになるため」 |
| compaction | コンテキスト圧縮 | CC-ja `prompt-caching` | 「### 会話のコンテキスト圧縮」 |
| system prompt | システムプロンプト | CC-ja `prompt-caching` | 「* **システムプロンプト**: 次のリクエストはこのレイヤーをキャッシュから読み込みます」 |
| main conversation ("your conversation") | メイン会話 | CC-ja `costs` | 「メイン会話のみをカバーし、サブエージェントはカバーしません」 |
| headless runs (claude -p) | 非対話型の実行 | CC-ja `headless` | 「Claude Code を非対話型モードで実行するには」 |
| Agent SDK | Agent SDK (kept) | CC-ja `headless` | 「Agent SDK を使用して、CLI、…」 |
| changelog | 変更ログ | VSC-ja extensionEditor :: changelog | 「変更ログ」 |
| Notification | 通知 | VSC-ja notificationsCenter :: notifications | 「通知」 |
| run / runs (of a subagent) | 実行 | no source: translated by analogy | — |
| Agent map | エージェント マップ | no source: translated by analogy (VS Code spacing) | — |
| built in | 組み込み | no source: translated by analogy | — |

Kept untranslated (per brief): ascii, unicode, catude, ctlite, Claude, Claude Code, MCP, Markdown, Windows Terminal, claude-swap, claude mcp list, claude /context, catude claude, all key names and the filter examples `security, mcp, tdd`.
