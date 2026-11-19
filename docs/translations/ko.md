# catude glossary: Korean (ko)

Localized Claude Code docs exist for Korean at `https://code.claude.com/docs/ko/…` (fetched as `.md`, 2026-11-18).
Abbreviations used below:

- **CC-ko** = Anthropic Claude Code docs, `https://code.claude.com/docs/ko/<page>`
- **VSC-ko** = VS Code language pack, `https://github.com/microsoft/vscode-loc/blob/main/i18n/vscode-language-pack-ko/translations/main.i18n.json` (module :: key)
- **LG-ko** = lazygit, `https://github.com/jesseduffield/lazygit/blob/master/pkg/i18n/translations/ko.json` (key)

## Claude concepts (source 1: Anthropic docs)

| English | Korean | Source | Exact string seen |
|---|---|---|---|
| skill / Skills | 스킬 | CC-ko `costs` | 「팀원은 CLAUDE.md, MCP 서버, 스킬을 자동으로 로드하지만」 (CC-ko also leaves "skills" in English 253×; 스킬 727×) |
| subagent | 서브에이전트 | CC-ko `costs` | 「주 대화만 다루며, 서브에이전트는 다루지 않습니다」 |
| agents / Agents | 에이전트 | VSC-ko customizationsToolbar.contribution :: agents | 「에이전트」 (CC-ko 993×) |
| hook / Hooks | 훅 | CC-ko `costs` | 「### 훅 및 스킬로 처리 오프로드」 |
| MCP server | MCP 서버 | CC-ko `costs`; VSC-ko aiCustomizationTreeViewViews :: mcpServers | 「MCP 서버」 |
| plugin / Plugins | 플러그인 | CC-ko `plugins` | 「# 플러그인 개요」 |
| marketplace | 마켓플레이스 | CC-ko `mcp` | 「…로 마켓플레이스를 추가한 다음 설치를 다시 시도하세요」 |
| command / Commands | 명령 | CC-ko `costs` | 「### `/usage` 명령 사용」 |
| settings / Settings | 설정 | CC-ko `settings` | 「설정은 Claude Code의 동작을 변경하는 JSON 키입니다」 |
| session / Sessions | 세션 | CC-ko `hooks` | 「세션의 모델이 변경된 후」 |
| conversation | 대화 | CC-ko `costs` | 「주 대화만 다루며」 |
| transcript | 트랜스크립트 | CC-ko `hooks` | 「트랜스크립트 파일은 비동기적으로 기록되며」 |
| resume (a session) | 재개 | CC-ko `hooks` | 「세션을 재개할 때 모델 복원」 |
| context / Context | 컨텍스트 | CC-ko `costs` | 「각 팀원은 자체 컨텍스트 윈도우를 실행하므로」 |
| model / Models | 모델 | CC-ko `hooks` | 「세션의 모델이 변경된 후」 |
| usage / Usage | 사용량 | CC-ko `costs` | 「개별 MCP 서버에 귀속된 최근 사용량이며」 |
| audit / Audit | 감사 | CC-ko `memory` | 「기본적으로 감사는 CLAUDE.md, CLAUDE.local.md 및 AGENTS.md 파일과…」 |
| health (claude mcp list) / check health | 상태 확인 | CC-ko `mcp` | 「`claude mcp list`는 나열하는 각 서버 옆에 `✔ Connected`… 와 같은 상태를 표시합니다」 |
| to-do (task list of a session) | 할 일 | CC-ko `interactive-mode` | 「## 작업 목록」「작업 목록은 Claude의 할 일 체크리스트입니다」 |
| reasoning (kind of work) | 추론 | CC-ko `costs` | 「복잡한 계획 및 추론 작업의 성능을 크게 향상시키기 때문입니다」 |
| tools / Tools | 도구 | CC-ko `costs`; VSC-ko customizationsToolbar.contribution :: tools | 「추적 도구」; 「도구」 |

## General UI words (source 2: VS Code)

| English | Korean | Source | Exact string seen |
|---|---|---|---|
| Preferences | 기본 설정 | VSC-ko actionCommonCategories :: preferences | 「기본 설정」 |
| Theme | 테마 | VSC-ko themes.contribution :: themes.category.light | 「밝은 테마」 (색 테마 for selectTheme.label) |
| light / dark / high-contrast | 밝음 / 어두움 / 고대비 | VSC-ko themes.contribution :: themes.category.light, .dark, .hc | 「밝은 테마」「어두운 테마」「고대비 테마」 (adjectives turned into value nouns) |
| auto | 자동 | VSC-ko scmHistoryViewPane :: auto | 「자동」 |
| off / on | 끄기 / 켜기 | VSC-ko log :: off | 「끄기」 (켜기 by symmetry) |
| hidden / shown | 숨겨짐 / 표시됨 | VSC-ko editor.contribution :: hidden | 「숨겨짐」 (표시됨 by symmetry) |
| View | 보기 | VSC-ko actionCommonCategories :: view | 「보기」 |
| Search / find | 검색 / 찾기 | VSC-ko search.contribution :: search; localizedStrings :: find | 「검색」「찾기」 |
| Filter / filter: | 필터 / 필터: | VSC-ko abstractTree :: filter | 「필터」 |
| Help | 도움말 | VSC-ko actionCommonCategories :: help | 「도움말」 |
| Keyboard shortcuts | 바로 가기 키 | VSC-ko preferences.contribution :: keyboardShortcuts | 「바로 가기 키」 |
| Language | 언어 | VSC-ko configurationExtensionPoint :: language | 「언어」 |
| Accounts | 계정 | VSC-ko globalCompositeBar :: accounts | 「계정」 |
| Profiles (work profile) | 프로필 (작업 프로필) | VSC-ko userDataProfile :: profiles | 「프로필」; 작업 + 프로필 = no source: translated by analogy |
| Open folder | 폴더 열기 | VSC-ko dialogMainService :: openFolder | 「폴더 열기」 |
| Editor | 편집기 | VSC-ko textEditor :: editor | 「편집기」 |
| Install | 설치 | VSC-ko agentPluginActions :: install | 「설치」 |
| Delete / remove | 삭제 / 제거 | VSC-ko aiCustomizationTreeView.contribution :: delete; manageDynamicAuthenticationProvidersAction :: remove | 「삭제」「제거」 |
| Debug | 디버그 | VSC-ko debugCommands :: debug | 「디버그」 |
| Security | 보안 | VSC-ko settingsLayout :: security | 「보안」 |
| Overview | 개요 | VSC-ko customizationsToolbar.contribution :: overview | 「개요」 |
| Available | 사용 가능 | VSC-ko localizationsActions :: available | 「사용 가능」 |
| Default | 기본값 | VSC-ko configurationExtensionPoint :: default | 「기본값」 |
| Expand all / Collapse all | 모두 확장 / 모두 축소 | VSC-ko comments.contribution :: expandAll; treeView :: collapseAll | 「모두 확장」「모두 축소」 |
| Refresh | 새로 고침 | VSC-ko treeView :: refresh | 「새로 고침」 |
| Reload | 다시 로드 | VSC-ko window :: reload | 「다시 로드(&&R)」 |
| Export | 내보내기 | VSC-ko userDataProfilesEditorModel :: export | 「내보내기...」 (CC-ko also 내보내기) |
| Trash | 휴지통 | VSC-ko fileActions :: trashFailed | 「휴지통을 사용하여 삭제하지 못했습니다…」 |
| Terminal | 터미널 | VSC-ko terminal.contribution :: terminal | 「터미널」 |
| Storage | 스토리지 | VSC-ko runScriptCustomTaskWidget :: storageAriaLabel | 「작업 스토리지 대상」 |
| machine | 컴퓨터 | VSC-ko developerActions :: machine | 「컴퓨터」 |
| Animations | 애니메이션 | VSC-ko workbench.contribution :: workbench.reduceMotion | 「워크벤치가 더 적은 애니메이션으로 렌더링되어야 하는지…」 |
| Glyphs | 문자 모양 | VSC-ko editorOptions :: glyphMargin | 「편집기에서 세로 문자 모양 여백을 렌더링할지 여부를 제어합니다」 |
| lite (reduced view) | 간단히 (간단히 보기) | VSC-ko sessionsList :: showLessCompact | 「간단히 표시」 |
| full (view) | 전체 | no source: translated by analogy (counterpart of 간단히) | — |
| shell integration | 셸 통합 | VSC-ko terminalInstance :: shellIntegration | 「셸 통합」 |
| running | 실행 중 | VSC-ko callStackView :: running | 「실행 중」 |
| yes / no | 예 / 아니요 | VSC-ko mainThreadAuthentication :: yes / no | 「예」「아니요」 |
| next / previous | 다음 / 이전 | VSC-ko inlineCompletionsHintsWidget :: next / previous | 「다음」「이전」 |
| compact ages %ds %dm %dh %dd | %d초 %d분 %d시간 %d일 | VSC-ko vs/base/common/date :: date.fromNow.seconds.plural etc. | 「{0}초」「{0}분」「{0}시간」「{0}일」 |
| %s ago | %s 전 | VSC-ko date :: date.fromNow.minutes.plural.ago | 「{0}분 전」 |

## Terminal-UI key hints (source 3: lazygit)

| English | Korean | Source | Exact string seen |
|---|---|---|---|
| close | 닫기 | LG-ko Close; VSC-ko abstractTree :: close | 「닫기」 |
| quit | 종료 | LG-ko Quit; VSC-ko menubar :: quit | 「종료」「종료(&&Q)」 |
| scroll | 스크롤 | LG-ko ScrollDown | 「아래로 스크롤」 |
| page, top, bottom | 페이지, 맨 위, 맨 아래 | LG-ko PrevPage, GotoTop, GotoBottom | 「이전 페이지」「맨 위로 스크롤」「맨 아래로 스크롤」 |
| previous / next tab | 이전 / 다음 탭 | LG-ko PrevTab / NextTab | 「다음 탭」 / 「이전 탭」 (lazygit has these two swapped; the meaning was used, not the swap) |
| cancel | 취소 | LG-ko Cancel | 「취소」 |
| edit | 편집 | LG-ko EditFile | 「파일 편집」 |

## No source: translated by analogy

| English | Korean | Note |
|---|---|---|
| work profile | 작업 프로필 | VS Code 프로필 + 작업 |
| Presets | 프리셋 | — |
| Discover (tab) | 탐색 | — |
| Live (tab) / Live sessions | 라이브 / 라이브 세션 | — |
| Inventory | 인벤토리 | — |
| Graph / node / diagram | 그래프 / 노드 / 다이어그램 | — |
| Accent colour | 강조 색 | Windows term; not in VS Code pack |
| amber / blue / cyan / green / magenta / red | 호박색 / 파랑 / 청록 / 초록 / 자홍 / 빨강 | — |
| mono (theme) | 단색 | — |
| safe (glyph set) | 안전 | — |
| Config (tab) / configuration / setup | 구성 | VS Code style (구성 폴더, 현재 구성) |
| Sources (references) | 출처 | — |
| Process (steps) | 프로세스 | — |
| Key hints / Keys | 키 힌트 / 키 | — |
| Everywhere (help section) | 모든 화면 | — |
| kind of work; coding; math & science; knowledge | 작업 유형; 코딩; 수학 및 과학; 지식 | benchmark category labels |
| waiting for you / working | 입력 대기 중 / 작업 중 | — |
| verify | 검증 | — |
| mark | 표시 | — |
| clean (safe item) | 정리 | — |
| measure | 측정 | — |
| rebuild | 다시 빌드 | — |
| spinners | 스피너 | — |

## Added 2026-11-19 (Agents tab, token drain, switching off per project)

| English | Korean | Source | Exact string seen |
|---|---|---|---|
| token / tokens | 토큰 | CC-ko `prompt-caching` | 「캐시된 토큰 요금」 |
| prompt cache / cache | 프롬프트 캐시 / 캐시 | CC-ko `prompt-caching` | 「# Claude Code가 프롬프트 캐싱을 사용하는 방법」 |
| invalidate the cache | 캐시를 무효화 | CC-ko `prompt-caching` | 「## 캐시를 무효화하는 작업」 |
| cache writes / reads | 캐시 쓰기 / 캐시 읽기 | CC-ko `prompt-caching` | 「캐시 쓰기 요금이 더 낮은 5분 TTL」 (읽기 by analogy) |
| (cache) expires | 만료 | CC-ko `prompt-caching` | 「그 시간이 지나면 캐시가 만료되므로」 |
| compaction | 압축 | CC-ko `prompt-caching`, `costs` | 「### 대화 압축」 |
| system prompt | 시스템 프롬프트 | CC-ko `prompt-caching` | 「* **시스템 프롬프트**: …」 |
| main conversation ("your conversation") | 주 대화 | CC-ko `costs` | 「주 대화만 다루며, 서브에이전트는 다루지 않습니다」 |
| headless runs (claude -p) | 비대화형 실행 | CC-ko `headless` | 「Claude Code를 비대화형 모드에서 실행하려면」 |
| Agent SDK | Agent SDK (kept) | CC-ko `headless` | 「Agent SDK를 사용하여 CLI, …」 |
| changelog | 변경 로그 | VSC-ko extensionEditor :: changelog | 「변경 로그」 |
| Notification | 알림 | VSC-ko notificationsCenter :: notifications | 「알림」 |
| run / runs (of a subagent) | 실행 | no source: translated by analogy | — |
| Agent map | 에이전트 맵 | no source: translated by analogy | — |
| built in | 기본 제공 | no source: translated by analogy | — |

Kept untranslated (per brief): ascii, unicode, catude, ctlite, Claude, Claude Code, MCP, Markdown, Windows Terminal, claude-swap, claude mcp list, claude /context, catude claude, all key names and the filter examples `security, mcp, tdd`.
