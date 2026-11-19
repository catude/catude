# catude glossary: Russian (ru)

Sources used (downloaded 2026-11-18):

- **A** = Anthropic Claude Code docs in Russian, `https://code.claude.com/docs/ru/<page>` (Markdown at `<page>.md`). Russian docs exist for every page checked (settings, hooks, skills, sub-agents, mcp, plugins, discover-plugins, plugin-marketplaces, memory, model-config, interactive-mode, costs, commands, context-window, checkpointing, statusline, output-styles, permissions). They mix Latin and Cyrillic forms ("skills"/"скиллы", "hooks"/"хуки"); the Cyrillic forms were used.
- **V** = VS Code Russian language pack, `https://raw.githubusercontent.com/microsoft/vscode-loc/main/i18n/vscode-language-pack-ru/translations/main.i18n.json` (cited as `module :: key = value`).
- **L** = lazygit Russian, `https://raw.githubusercontent.com/jesseduffield/lazygit/master/pkg/i18n/translations/ru.json` (cited as `Key = value`).

Register: VS Code Russian uses infinitives for commands ("Открыть папку") and formal **вы** in sentences; followed. VS Code writes "е" for "ё" ("темные темы", "еще {1}"); followed.

| English | Russian | Source | Exact string seen |
|---|---|---|---|
| Skills | Скиллы | A skills, commands | `### Скиллы, синхронизированные с claude.ai`; `как скиллы, которые вы пишете сами` (V: `Навыки`; A chosen) |
| Agents / agents | Агенты / агенты | V; A | V `categories :: agents = Агенты`; A `между множеством субагентов` |
| subagent | субагент | A commands | `попросить Claude создать [субагентов](…)` |
| Commands | Команды | A commands; V | `# Команды`; V `menusExtensionPoint :: commands = Команды` |
| Hooks | Хуки | A hooks | `Хуки определяются в JSON-файлах настроек.` (V: `Обработчики`; A chosen) |
| MCP servers | MCP-серверы | A costs, hooks | `до того, как MCP-серверы становятся доступны` |
| Plugins | Плагины | A discover-plugins; V | `Установите плагины Claude Code из маркетплейса`; V `chatPluginActions :: plugins = Плагины` |
| marketplaces | маркетплейсы | A discover-plugins | `добавление маркетплейсов и поддержание плагинов` (V keeps `Marketplace`) |
| Settings | Настройки | A hooks, settings | `## Файлы настроек и кого они затрагивают` (A title also says `Файлы параметров`; V uses `Параметры` for Settings) |
| Preferences | Параметры | V | `actionCommonCategories :: preferences = Параметры`, `menubar :: miPreferences = &&Параметры` |
| session / Sessions | сессия / Сессии | A checkpointing, commands | `в своей сессии`, `возобновить эту сессию` (A also uses `сеанс`; "сесси*" ≈1000 hits vs "сеанс*" ≈260; V uses `Сеансы`) |
| conversation | диалог | A checkpointing, commands | `Claude Code сохраняет чекпоинты вместе с диалогом`; `не попадая в диалог, который видит Claude` (V: `беседа`; A chosen) |
| transcript | транскрипт | A commands | `Примечания появляются в вашем транскрипте` |
| context | контекст | A checkpointing | `сжимает контекст` |
| context window (not in UI, checked) | контекстное окно | A context-window | `# Изучите контекстное окно` |
| Models | Модели | V; A | V `aiCustomizationManagementEditor :: models = Модели`; A `выбора модели` |
| Tools | Инструменты | V; A | V `aiCustomizationManagementEditor :: tools = Инструменты`; A `выполняются инструменты` |
| Usage | Использование | A costs; V | `## Снижение использования токенов`; V `argv :: usage = Использование` |
| resume | возобновить | A commands; V | `возобновить эту сессию`; V `editSessions.contribution :: resume = Возобновить` |
| to-do | задачи | A interactive-mode | `Когда список задач пуст` |
| tab | вкладка | A discover-plugins; V; L | A `на вкладке **Discover**`; V `terminalTabsList :: terminal.tabs = Вкладки терминала`; L `NextTab = Следующая вкладка` |
| previous / next tab | предыдущая / следующая вкладка | L | `PrevTab = Предыдущая вкладка`, `NextTab = Следующая вкладка` |
| Overview | Обзор | V | `customizationsToolbar.contribution :: overview = Обзор` |
| Search | поиск | V; L | V `settingsLayout :: search = Поиск`; L `SearchTitle = Поиск` |
| Filter / filter: | фильтр / фильтр: | V; L | V `abstractTree :: filter = Фильтр`; L `FilteringMenuTitle = Фильтрация` |
| Help | справка | V | `actionCommonCategories :: help = Справка` |
| quit | выход | V | `menubar :: quit = &&Выход` (L `Quit = Выйти`) |
| close | закрыть | V; L | V `abstractTree :: close = Закрыть`; L `Close = Закрыть` |
| back | назад | V | `quickInput :: quickInput.back = Назад` |
| Debug | Отладка | V | `log :: debug = Отладка` |
| install | установить | V | `extensionsActions :: install = Установить` |
| delete / remove | удалить | V | `fileActions :: delete = Удалить`; `…AuthenticationProvidersAction :: remove = Удалить` |
| edit | изменить | V | `chatToolPartUtilities :: edit = Изменить` |
| refresh | обновить | V; L | V `treeView :: refresh = Обновить`; L `Refresh = Обновить` |
| reload | перезагрузить | V | `workspaceWatcher :: reload = Перезагрузить` |
| export | экспорт / экспортировать | V | `userDataProfilesEditorModel :: export = Экспорт...` |
| expand / collapse everything | развернуть / свернуть все | V | `comments.contribution :: expandAll = Развернуть все`, `collapseAll = Свернуть все` |
| scroll | прокрутка | V; L | V `editorOptions :: scrollPredominantAxis = Прокрутка только вдоль…`; L `ScrollDown = Прокрутить вниз` |
| mark | пометить | V | `sessionsViewActions :: markRead = Пометить как прочитанное` |
| trash (move to) | переместить в корзину | V | `diskFileSystemProviderServer :: trashFailed = Не удалось переместить "{0}" в корзину ({1})` |
| restorable | можно восстановить | V | `userDataSyncViews :: workbench.actions.sync.replaceCurrent = Восстановить` |
| open folder | открыть папку | V | `dialogMainService :: openFolder = Открыть папку` |
| Default | по умолчанию | V | `configurationExtensionPoint :: default = По умолчанию` |
| Theme | Тема | V | `themes.contribution :: selectTheme.label = Цветовая тема` |
| light / dark / high-contrast | светлая / темная / высокий контраст | V | `themes.category.light = светлые темы`, `themes.category.dark = темные темы`, `themes.category.hc = темы с высоким контрастом` |
| on / off | вкл / выкл | V | `agentHostSessionConfigPicker :: …onLabel = Вкл`; `auxiliaryEditorPart :: disableCompactAuxiliaryWindow = Выключить компактный режим` |
| hidden | скрыто | V | `editor.contribution :: hidden = Скрыто` |
| lite (view) | компактный / компактный вид | V | `activitybarPart :: activityBarSizeCompact = Компактный` |
| View | Вид | V | `menubar :: mView = &&Вид` |
| Language | Язык | V | `localizationsActions :: configureLocale = Настройка языка интерфейса` |
| Glyphs | Глифы | V | `editorOptions :: glyphMargin = Управляет отображением вертикальных полей глифа…` |
| Accent colour | Акцентный цвет | no source: translated by analogy | (no V string for accent colour) |
| Animations | Анимации | V | `chat.shared.contribution :: chat.upvoteAnimation.off = Анимация не отображается.` |
| Keyboard shortcuts | сочетания клавиш | V; A | V `preferences.contribution :: keyboardShortcuts = Сочетания клавиш`; A `## Сочетания клавиш` |
| Accounts | Учетные записи | V | `globalCompositeBar :: accounts = Учетные записи` (A skills uses `Аккаунт claude.ai`) |
| Profiles / profile | Профили / профиль | V | `userDataProfilesEditor :: profiles = Профили`, `newProfile = Новый профиль` |
| switch (profile) | переключить | V | `userDataProfile :: switchProfile = Переключить профиль…` |
| Security | Безопасность | V | `settingsLayout :: security = Безопасность` |
| terminal | терминал | V | `terminal.contribution :: terminal = Терминал` |
| Storage | Хранилище | V | `runScriptAction :: workspaceStorageUnavailableTooltip = Хранилище рабочей области недоступно…` |
| %ds | %d с | V | `date :: duration.s = {0} с` |
| %dm | %d мин | V | `date :: duration.m = {0} мин` |
| %dh | %d ч | V | `date :: duration.h = {0} ч` |
| %dd | %d дн. | V | `date :: duration.d = {0} дн.` |
| … more | еще %d | V | `chatDebugFlowGraph :: discoveryMergedLabel = {0} + еще {1}` |
| work profile | рабочий профиль | no source: translated by analogy | (V `Профиль`) |
| Presets | Шаблоны | no source: translated by analogy | |
| Discover (tab) | Каталог | no source: translated by analogy | A keeps Claude Code's English tab label: `на вкладке **Discover**` |
| Live / Live sessions | Активные / Активные сессии | no source: translated by analogy | V `livePreview = Динамический просмотр` (not suitable for a tab) |
| Audit | Аудит | no source: translated by analogy | |
| Inventory | Состав | no source: translated by analogy | |
| Graph | Граф | no source: translated by analogy | |
| Config | Конфигурация | A | `конфигурации` (115 hits), e.g. `Если /init обнаружит конфигурацию` |
| setup | конфигурация | no source: translated by analogy | |
| check health | проверить состояние | no source: translated by analogy | |
| reasoning / coding / knowledge / math & science | рассуждения / программирование / знания / математика и наука | no source: translated by analogy | |
| safe (glyph set) | безопасные | no source: translated by analogy | |
| colours amber, blue, cyan, green, magenta, red | янтарный, синий, голубой, зеленый, пурпурный, красный | no source: translated by analogy | |

## Added 2026-11-19 (Agents tab, token drain, switching off per project)

| English | Russian | Source | Exact string seen |
|---|---|---|---|
| token / tokens | токен / токены | A costs | `## Снижение использования токенов` |
| prompt cache / cache | кэш промптов / кэш | A prompt-caching | `# Как Claude Code использует кэширование промптов`; `## Действия, которые инвалидируют кэш` |
| cache reads / writes | чтение кэша / запись в кэш | A costs | `состоит из чтений кэша плюс несколько…`; `запросы записали обратно в кэш` |
| (cache) expires | истекает | A prompt-caching | `срок действия кэша истекает` |
| cache rebuild | перестроение кэша | A prompt-caching | `читает более старую запись кэша вместо перестроения` |
| compaction | сжатие контекста | A prompt-caching, costs | `### Сжатие диалога`; `[Сжатие контекста](…)` |
| system prompt | системный промпт | A prompt-caching | `* **Системный промпт**: следующий запрос читает этот слой` |
| headless runs (claude -p) | неинтерактивные запуски | A headless | `Чтобы запустить Claude Code в неинтерактивном режиме, передайте -p` |
| Agent SDK | Agent SDK (kept) | A headless | `[Agent SDK](…)` |
| changelog | журнал изменений | V | `extensionEditor :: changelog = Журнал изменений` |
| Notification | Уведомление | V | `notificationsCenter :: notifications = Уведомления` |
| run / runs (of a subagent) | запуск / запуски | no source: translated by analogy | |
| Agent map | Карта агентов | no source: translated by analogy | |
| built in | встроенный | no source: translated by analogy | |

Counts with Russian plurals are written as "label: %d" ("запусков: %d") so that one form fits every number.
