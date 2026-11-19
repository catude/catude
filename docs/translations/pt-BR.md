# Glossário catude: português do Brasil (pt-BR)

Arquivo: `internal/i18n/locales/pt-BR.json` (197 mensagens).

Fontes:
- **CC** = documentação do Claude Code da Anthropic em português: `https://code.claude.com/docs/pt/<página>` (lida em `.md`, 18/11/2026). O caminho `/docs/pt-BR/` dá 404; o site usa `/docs/pt/`, e o texto é português do Brasil («arquivo» 81 vezes e «ficheiro» 0 em settings.md; «você»).
- **VS** = pacote de idioma do VS Code: https://github.com/microsoft/vscode-loc/blob/main/i18n/vscode-language-pack-pt-BR/translations/main.i18n.json (chave e valor citados).
- **LG** = lazygit: https://github.com/jesseduffield/lazygit/blob/master/pkg/i18n/translations/pt.json (chave e valor citados).

Registro: o VS Code em pt-BR trata por «você» e usa imperativo («Peça ao agente…», `tip.mermaid`). No catude as ações vão no infinitivo («abrir», «filtrar») e as frases dirigidas ao usuário usam «você». Caixa: o VS Code pt-BR usa muitas vezes Title Case («Abrir Pasta»); aqui manteve-se a caixa da frase do original em inglês.

## Conceitos do Claude (fonte 1: documentação da Anthropic)

| Inglês | Português | Fonte | Texto visto |
|---|---|---|---|
| skill / Skills | skill / Skills (sem tradução) | CC skills | `# Estender Claude com skills`; glossário: «Claude carrega uma skill automaticamente» |
| subagent | subagente | CC sub-agents | `# Criar subagentes personalizados` |
| agents (guia) | Agentes | VS | `agents => Agentes` |
| hook / Hooks | hook / Hooks (sem tradução) | CC hooks; VS | `# Referência de hooks`; `hooks => Hooks` |
| MCP servers | Servidores MCP | CC mcp | «Os servidores MCP dão ao Claude…» |
| plugin / Plugins | plugin / Plugins | CC plugins/overview; VS | `# Visão geral de plugins`; `plugins => Plugins` |
| marketplaces | marketplaces | CC glossary, plugins/install | «marketplaces de plugins» |
| commands (guia) | Comandos | CC commands | `# Comandos` |
| settings (do Claude) | configurações | CC settings | `# Arquivos de configurações e precedência` |
| context window / context | contexto (janela de contexto) | CC context-window | `# Explore a janela de contexto` |
| session / sessions | sessão / Sessões | CC sessions | `# Gerenciar sessões` |
| resume (sessão) | retomar | CC sessions | `## Retomar uma sessão` |
| export | exportar | CC sessions | `## Exportar e localizar dados de sessão` |
| transcript | transcrição | CC sessions | «arquivos de transcrição locais» (a doc também usa «transcript»/«transcritos») |
| to-do (lista do Claude) | tarefas | CC interactive-mode | «Alternar lista de tarefas do Claude» |
| tool / tools | ferramenta / Ferramentas | VS; CC subagent | `tools => Ferramentas`; «acesso a ferramentas específicas» |
| model / Models | modelo / Modelos | CC model-config; VS | «qual modelo e nível de esforço se adequam»; `models => Modelos` |
| keyboard shortcuts | atalhos | CC keybindings; LG | `# Personalizar atalhos de teclado`; `Keybindings => Atalhos do teclado` |

## Palavras de interface (fonte 2: VS Code; fonte 3: lazygit)

| Inglês | Português | Chave | Valor visto |
|---|---|---|---|
| Preferences | Preferências | VS `preferences` | `Preferências` |
| Settings (seção) | Configurações | VS `settings` | `Configurações` |
| Theme | Tema | VS `selectTheme.label` | `Tema de Cores` |
| light / dark | claro / escuro | VS `themes.category.light` / `.dark` | `temas claros` / `temas escuros` |
| high-contrast | alto contraste | VS `toggleHighContrast` | `Ativar/Desativar Tema de Alto Contraste` |
| Accent colour | Cor de destaque | VS `chat.inputWorkingBorderColor2` | `Cor de destaque secundária…` |
| highlight colour | cor de realce | VS `peekViewResultsMatchHighlight` | `…cor de realce…` |
| Glyphs | Glifos | VS `drawGlyph.label` | `Desenhar Glifo` |
| Animations | Animações | VS `workbench.reduceMotion` | `…com menos animações.` |
| on / off | ativado / desativado | VS `agentHostSessionConfig.boolean.onLabel` / `offLabel` | `Ativado` / `Desativado` |
| hidden | oculto | VS `hidden` | `Oculto` |
| Language | Idioma | VS `setDisplayLanguage` | `Definir Idioma de Exibição` |
| View | Exibição | VS `toggle` | `Ativar/Desativar Modo de Exibição Fixo` |
| tab / tabs | guia / guias | VS `workbench.editor.showTabs.multiple` | `Cada editor é exibido como uma guia…` |
| lite (exibição) / full | compacta / completa | VS `codiconFontSizeCompact`; `fullscreen` | `…fonte compacta…`; `Tela Inteira` |
| filter | filtrar / filtro | VS `filter` | `Filtrar` |
| search | pesquisar | VS `search` | `Pesquisar` |
| find | localizar | VS `find` | `Localizar` |
| Help | ajuda | VS `help` | `Ajuda` |
| quit | sair | VS `exit`; LG `Quit` | `&&Sair`; `Sair` |
| close | fechar | VS `close` | `Fechar` |
| back | voltar | VS `back` | `Voltar` |
| refresh / reload | atualizar / recarregar | VS `refresh` / `reload` | `Atualizar` / `&&Recarregar` |
| install / upgrade | instalar / atualizar | VS `install` / `upgrade` | `Instalar` / `Atualizar` |
| delete | excluir | VS `delete` | `Excluir` |
| expand / collapse all | expandir / recolher tudo | VS `expandAll` / `collapseAll` | `Expandir Tudo` / `Recolher Tudo` |
| folder | pasta | VS `openFolder` | `Abrir a Pasta` |
| trash | lixeira | VS `undoTrash` | `Você pode restaurar esse arquivo da Lixeira.` |
| backup | backup | VS `Open Backup folder` | `Abrir Pasta de Backups Local` |
| default | padrão | VS `default` | `Padrão` |
| editor | editor | VS `editor` | `Editor` |
| Debug | Depuração | VS `debug` | `Mostrar Visualização de Depuração do Chat` |
| browser | navegador | VS `browser` | `Navegador` |
| Accounts | Contas | VS `accounts` | `Contas` |
| Profiles | Perfis | VS `profiles` | `Perfis` |
| Overview | Visão geral | VS `overview` | `Visão geral` |
| Security | Segurança | VS `security` | `Segurança` |
| Usage | Uso | VS `usage` | `Uso` |
| Context (guia) | Contexto | VS `chatContext` | `Contexto` |
| Graph | Grafo | VS `scmGraph` | `Grafo` |
| running | em execução | VS `running` | `Em execução` |
| working | trabalhando | VS `working` | `Trabalhando` |
| font | fonte | VS `font` | `Fonte` |
| scroll | rolar | LG `ScrollDown`; VS `mouseWheelScrollSensitivity` | `Rolar para baixo`; `…eventos de rolagem…` |
| colours | vermelho, verde, azul | VS `chartsRed`, `chartsGreen`, `chartsBlue` | `A cor vermelha…`, `A cor verde…`, `A cor azul…` |
| %ds / %dmin | %ds / %dmin | VS `duration.s`, `update.timeRemainingMinutes` | `{0}s`, `{0} min` |
| %s ago | %s atrás | VS `date.fromNow.minutes.plural.ago` | `{0} minutos atrás` |

## Sem fonte: traduzido por analogia

| Inglês | Português | Motivo |
|---|---|---|
| %dh / %dd | %dh / %dd | O VS Code só tem `{0} horas` / `{0} dias`; «h» e «d» compactos como s/min |
| work profile | perfil de trabalho | Conceito do catude; «Perfil» vem do VS Code |
| Presets | Predefinições | Sem termo no VS Code nem na doc |
| cyan / magenta / amber | ciano / magenta / âmbar | Cores ausentes do VS Code |
| mono (tema) | monocromático | Sem fonte |
| safe (glifos) | seguro | Sem fonte |
| Key hints | Dicas de teclas | VS Code: `Hint => Dica` |
| spinners | indicadores de atividade | Sem fonte |
| Live | Ao vivo | O pacote pt-BR não tem o equivalente de «En directo» |
| Discover / Inventory / Storage / Audit | Descobrir / Inventário / Armazenamento / Auditoria | Sem fonte |
| Process / Sources | Processo / Fontes | Sem fonte |
| rebuild (grafo) | reconstruir | O VS Code usa «Recompilar» para *build* de código; aqui é reconstruir o grafo |
| check health (MCP) | verificar integridade | Sem fonte |
| waiting for you | aguardando resposta | Sem fonte (a doc usa «aguardando») |
| Config (guia) | Config | Abreviatura compreendida em português |

## Adicionados em 19/11/2026 (guia Agentes, consumo de tokens, desativar por projeto)

| Inglês | Português | Fonte | Texto visto |
|---|---|---|---|
| token / tokens | token / tokens | CC costs | «reduzir o uso de tokens»; «tokens de entrada servidos do cache» |
| prompt cache / cache | cache de prompt / o cache | CC prompt-caching | `# Como Claude Code usa o cache de prompt`; `## Ações que invalidam o cache` |
| cache reads / writes | leituras de cache / escritas no cache | CC costs | «principalmente leituras de cache mais alguns tokens de saída» (escritas: por analogia) |
| (cache) expires | expira | CC prompt-caching | «as entradas expiram após um período de inatividade» |
| compaction | compactação | CC costs | «compactação automática»; «compactação ociosa» |
| system prompt | system prompt (sem tradução) | CC prompt-caching, costs | `* **System prompt**: a próxima solicitação lê…`; «conteúdo repetido como system prompts» |
| headless runs (claude -p) | execuções não interativas | CC headless | «Para executar Claude Code em modo não interativo, passe `-p`…» |
| Agent SDK | Agent SDK (sem tradução) | CC headless | «Use o Agent SDK para executar Claude Code…» |
| changelog | log de mudanças | VS `extensionEditor :: changelog` | `Log de mudanças` |
| Notification | Notificação | VS `notificationsCenter :: notifications` | `Notificações` |
| run / runs (de um subagente) | execução / execuções (cabeçalho «exec.») | Sem fonte: por analogia | — |
| Agent map | Mapa de agentes | Sem fonte: por analogia | — |
| built in | integrado | Sem fonte: por analogia | — |
