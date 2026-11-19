# catude glossary: German (de)

Sources used (downloaded 2026-11-18):

- **A** = Anthropic Claude Code docs in German, `https://code.claude.com/docs/de/<page>` (Markdown at `<page>.md`). German docs exist for every page checked (settings, hooks, skills, sub-agents, mcp, plugins, discover-plugins, plugin-marketplaces, memory, model-config, interactive-mode, costs, commands, context-window, checkpointing, statusline, output-styles, permissions).
- **V** = VS Code German language pack, `https://raw.githubusercontent.com/microsoft/vscode-loc/main/i18n/vscode-language-pack-de/translations/main.i18n.json` (cited as `module :: key = value`).
- **L** = lazygit: **there is no German translation** (`pkg/i18n/translations` has ja, ko, nl, pl, pt, ru, zh-CN, zh-TW only), so key-hint phrasing follows V (infinitive verb at the end, as in "Ordner öffnen").

Register: VS Code German uses formal **Sie** ("Drücken Sie die EINGABETASTE…"); followed ("Starten Sie eine…", "wartet auf Sie", "Sie verwenden").

| English | German | Source | Exact string seen |
|---|---|---|---|
| Skills | Skills | A skills | `# Claude mit Skills erweitern` |
| Agents (tab) / agents | Agenten | A hooks; V | `in welchem Agenten ein Hook ausgelöst wurde`; V `vs/sessions/common/categories :: agents = Agenten` |
| subagent | Subagent | A sub-agents, checkpointing | `# Benutzerdefinierte Subagenten erstellen`; `Ein [Subagent](…) nimmt Bearb…` |
| Commands | Befehle | A commands | `# Befehle` |
| Hooks | Hooks | A hooks | `# Hooks-Referenz` |
| MCP servers | MCP-Server | A mcp | `## MCP-Server installieren` |
| Plugins | Plugins | A discover-plugins | `# Plugins installieren und verwalten` (V uses `Plug-Ins`; A chosen) |
| marketplaces | Marketplaces | A discover-plugins | `Hinzufügen von Marketplaces und Aktualisierung von Plugins` |
| Settings | Einstellungen | A settings; V | `# Einstellungsdateien und Priorität`; V `preferences.contribution :: settings = Einstellungen` |
| Preferences | Einstellungen | V | `actionCommonCategories :: preferences = Einstellungen`, `menubar :: miPreferences = &&Einstellungen` |
| session / Sessions | Sitzung / Sitzungen | A commands, checkpointing; V | `eine andere Sitzung fortsetzen`; V `chatViewPane :: sessions = Sitzungen` |
| conversation | Konversation | A checkpointing | `Claude Code speichert Checkpoints zusammen mit der Konversation` (V uses `Unterhaltung`; A chosen) |
| transcript | Transkript | A commands | `Die Notes erscheinen in Ihrem Transkript` |
| context / Context | Kontext | A checkpointing; V | `komprimiert den Kontext`; V `settingsLayout :: chatContext = Kontext` |
| context window (not in UI, checked) | Kontextfenster | A context-window | `# Erkunden Sie das Kontextfenster` |
| Models | Modelle | A commands; V | `Modelle zu wechseln`; V `aiCustomizationManagementEditor :: models = Modelle` |
| Tools | Tools | A checkpointing; V | `durch seine Tools zur Dateibearbeitung`; V `aiCustomizationManagementEditor :: tools = Tools` |
| Usage | Nutzung | A costs | `Verfolgen Sie die Token-Nutzung` |
| resume | fortsetzen | A checkpointing; V | `eine andere Sitzung fortsetzen`; V `editSessions.contribution :: resume = Fortsetzen` |
| to-do | Aufgaben | A interactive-mode | `## Aufgabenliste`, `Claudes Aufgabenliste umschalten` |
| tab | Registerkarte | V; A discover-plugins | V `layoutActions :: configureTabs = Registerkarten konfigurieren`; A `auf der Registerkarte **Discover**` |
| Overview | Übersicht | V | `customizationsToolbar.contribution :: overview = Übersicht` |
| Search / find | suchen / durchsuchen | V | `settingsLayout :: search = Suchen` |
| Filter / filter: | filtern / Filter: | V | `abstractTree :: filter = Filter`; `sessionsViewActions :: filter = Filtern` |
| Help | Hilfe | V | `actionCommonCategories :: help = Hilfe` |
| quit | beenden | V | `menubar :: exit = &&Beenden` (V `quit = &&Aufhören` is odd; `Beenden` chosen) |
| close | schließen | V | `abstractTree :: close = Schließen` |
| back | zurück | V | `quickInput :: quickInput.back = Zurück` |
| Debug | Debuggen | V | `log :: debug = Debuggen` |
| install | installieren | V | `extensionsActions :: install = Installieren` |
| delete | löschen | V | `fileActions :: delete = Löschen` |
| remove | entfernen | V | `manageDynamicAuthenticationProvidersAction :: remove = Entfernen` |
| edit | bearbeiten | V | `mcpLanguageFeatures :: edit = Bearbeiten` |
| refresh | aktualisieren | V | `treeView :: refresh = Aktualisieren` |
| reload | neu laden | V | `workspaceWatcher :: reload = Neu laden` |
| export | exportieren | V | `userDataProfilesEditorModel :: export = Exportieren...` |
| expand / collapse everything | alles aufklappen / zuklappen | V | `comments.contribution :: expandAll = Alle aufklappen`, `collapseAll = Alle zuklappen` |
| scroll | Bildlauf | V | `hoverActions :: scrollDownHover = Bildlauf nach unten beim Daraufzeigen` |
| mark | markieren | V | `sessionsViewActions :: markRead = Als gelesen markieren` |
| trash (move to) | in den Papierkorb verschieben | V | `diskFileSystemProviderServer :: trashFailed = Fehler beim Verschieben von „{0}“ in den Papierkorb ({1})` |
| restorable | wiederherstellbar | V | `userDataSyncViews :: workbench.actions.sync.replaceCurrent = Wiederherstellen` |
| open folder | Ordner öffnen | V | `dialogMainService :: openFolder = Ordner öffnen` |
| Default | Standard | V | `configurationExtensionPoint :: default = Standard` |
| Theme | Design | V | `themes.contribution :: selectTheme.label = Farbdesign` |
| light / dark / high-contrast | hell / dunkel / hoher Kontrast | V | `theme_picker :: light = Hell modern`, `dark = Dunkel modern`, `HighContrast = Dunkel hoher Kontrast` |
| on / off | ein / aus | V | `agentHostSessionConfigPicker :: …onLabel = Ein`, `…offLabel = Aus` |
| hidden | ausgeblendet | V | `editor.contribution :: hidden = Ausgeblendet` |
| lite (view) | kompakt / Kompaktansicht | V | `activitybarPart :: activityBarSizeCompact = Kompakt` |
| View | Ansicht | V | `viewsExtensionPoint :: views = Ansichten` |
| Language | Sprache | V | `localizationsActions :: configureLocale = Anzeigesprache konfigurieren` |
| Glyphs | Glyphen | V | `editorOptions :: glyphMargin = …Glyphenrand…` |
| Accent colour | Akzentfarbe | V | `chatColors :: chat.inputWorkingBorderColor3 = Akzentfarbe, die von …` |
| Animations | Animationen | V | `editorOptions :: cursorBlinking = Steuert den Cursoranimationsstil.` |
| Keyboard shortcuts | Tastenkombinationen | V | `preferences.contribution :: keyboardShortcuts = Tastenkombinationen` (A interactive-mode uses `## Tastaturkürzel`; V chosen for a general UI word) |
| Accounts | Konten | V | `globalCompositeBar :: accounts = Konten` |
| Profiles / profile | Profile / Profil | V | `userDataProfilesEditor :: profiles = Profile`, `newProfile = Neues Profil` |
| switch (profile) | wechseln | V | `userDataProfile :: switchProfile = Profil wechseln...` |
| Security | Sicherheit | V | `settingsLayout :: security = Sicherheit` |
| terminal | Terminal | V | `terminal.contribution :: terminal = Terminal` |
| machine | Computer | V | `remote.contribution :: workspace = …auf dem Remotecomputer…` |
| %ds | %ds | V | `date :: duration.s = {0}s` |
| %dm | %d Min. | V | `date :: duration.m = {0} Min.` |
| %dh | %d Std. | V | `date :: duration.h = {0} Std.` |
| %dd | %d Tg. | V (adapted) | V has only `date :: duration.d = {0} Tage`; abbreviated because the value is inserted into "vor %s" / "seit %s", where `Tage` would need the dative `Tagen` |
| work profile | Arbeitsprofil | no source: translated by analogy | (VS Code `Profil` + "Arbeit") |
| Presets | Vorlagen | no source: translated by analogy | |
| Discover (tab) | Entdecken | no source: translated by analogy | A keeps Claude Code's own English tab label: `auf der Registerkarte **Discover**` |
| Audit | Audit | no source: translated by analogy | |
| Inventory | Bestand | no source: translated by analogy | |
| Storage | Speicher | no source: translated by analogy | |
| Graph | Graph | no source: translated by analogy | |
| Live / Live sessions | Live / Live-Sitzungen | V (partial) | `codexAgent :: codex.sessionConfig.webSearchMode.live = Live`, `editorConfiguration :: livePreview = Livevorschau` |
| check health | Status prüfen | A mcp (analogy) | `hängt claude mcp list die Fehlerdetails an diese Statuszeile an` |
| setup | Setup | no source: translated by analogy | |
| reasoning / coding / knowledge / math & science | Schlussfolgern / Programmieren / Wissen / Mathematik & Wissenschaft | no source: translated by analogy | |
| safe (glyph set) | sicher | no source: translated by analogy | |
| colours amber, blue, cyan, green, magenta, red | bernstein, blau, cyan, grün, magenta, rot | no source: translated by analogy (V `chartsColors :: chartsBlue = Die in Diagrammvisualisierungen verwendete Farbe Blau.`) | |

## Added 2026-11-19 (Agents tab, token drain, switching off per project)

| English | German | Source | Exact string seen |
|---|---|---|---|
| token / tokens | Token | A costs | `Verfolgen Sie die Token-Nutzung`; `Anteil der Eingabe-Token, die aus dem Cache bereitgestellt werden` |
| prompt cache / cache | Prompt-Cache / Cache | A prompt-caching, costs | `# Wie Claude Code Prompt-Caching nutzt`; `Diese Anfrage nutzt den Prompt-Cache des Gesprächs wieder` |
| invalidate the cache | den Cache ungültig machen | A prompt-caching | `## Aktionen, die den Cache ungültig machen` |
| cache reads / writes | aus dem Cache gelesen / in den Cache geschrieben | A prompt-caching, costs | `wird das unveränderte Präfix aus dem Cache gelesen`; `wie viele Token diese Anfragen zurück in den Cache geschrieben haben` |
| (cache) expires | läuft ab | A prompt-caching | `fragte auch nach Ablauf des Cache` |
| compaction | Komprimierung | A prompt-caching, costs | `### Konversation komprimieren`; `die automatische Komprimierung` |
| system prompt | System-Prompt | A prompt-caching | `* **System-Prompt**: Die nächste Anfrage liest diese Schicht…` |
| headless runs (claude -p) | nicht interaktive Ausführungen | A headless | `Der nicht interaktive Modus liest stdin`; `einen nicht interaktiven Aufruf in einem Skript` |
| Agent SDK | Agent SDK (kept) | A headless | `Verwenden Sie das Agent SDK, um Claude Code programm…` |
| marketplace (gender) | der Marketplace | A settings | `keine Plugins aus einem Marketplace, den die Datei deklariert` |
| changelog | Änderungsprotokoll | V | `extensionEditor :: changelog = Änderungsprotokoll` |
| Notification | Benachrichtigung | V | `notificationsCenter :: notifications = Benachrichtigungen` |
| run / runs (of a subagent) | Ausführung / Ausführungen (column header `Läufe`) | no source: translated by analogy | |
| Agent map | Agentenkarte | no source: translated by analogy | |
| built in | integriert | no source: translated by analogy | |
