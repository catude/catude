# Glosario catude: español (es)

Archivo: `internal/i18n/locales/es.json` (197 mensajes).

Fuentes:
- **CC** = documentación de Claude Code de Anthropic en español: `https://code.claude.com/docs/es/<página>` (consultada en formato `.md`, 18/11/2026).
- **VS** = paquete de idioma de VS Code: https://github.com/microsoft/vscode-loc/blob/main/i18n/vscode-language-pack-es/translations/main.i18n.json (se cita la clave y el valor).
- **LG** = lazygit no tiene traducción al español (solo ja, ko, nl, pl, pt, ru, zh-CN, zh-TW), así que no se usó para este idioma.

Registro: VS Code en español usa formas impersonales o de *usted* («Pida al agente…», `tip.mermaid`). En catude las acciones van en infinitivo («abrir», «filtrar») y las frases evitan el tuteo («Al iniciar una (claude)…», «Usado antes de escribir», «En uso: »).

## Conceptos de Claude (fuente 1: documentación de Anthropic)

| Inglés | Español | Fuente | Texto visto |
|---|---|---|---|
| skill / Skills | skill / Skills (sin traducir) | CC skills | `# Extender Claude con skills`; glosario: «Claude carga un skill automáticamente» |
| subagent | subagente | CC sub-agents, glossary | `# Crear subagentes personalizados`; `### Subagente` |
| agents (pestaña) | Agentes | VS | `agents => Agentes` |
| hook / Hooks | hook / Hooks (sin traducir) | CC hooks | `# Referencia de hooks` |
| MCP servers | Servidores MCP | CC mcp | «Los servidores MCP dan a Claude…» |
| plugin / Plugins | plugin / Plugins | CC plugins/overview | `# Descripción general de plugins` |
| marketplaces | marketplaces | CC glossary, plugins/install | «marketplaces de plugins» (50 apariciones de «marketplaces») |
| commands (pestaña) | Comandos | CC commands | `# Comandos` |
| settings (de Claude) | configuración | CC settings | `# Archivos de configuración y precedencia` |
| context window / context | contexto (ventana de contexto) | CC context-window | `# Explorar la ventana de contexto` |
| session / sessions | sesión / Sesiones | CC glossary, sessions | `### Sesión`; `# Gestionar sesiones` |
| resume (sesión) | reanudar | CC sessions | `## Reanuda una sesión` |
| export | exportar | CC sessions | `## Exporte y localice datos de sesión` |
| transcript | transcripción | CC glossary, sessions | «la transcripción es esa conversación guardada como un archivo»; `### Dónde se almacenan las transcripciones` |
| to-do (lista de Claude) | tareas | CC interactive-mode | «Alternar lista de tareas de Claude» |
| tool / tools | herramienta / Herramientas | CC glossary; VS | «Las herramientas son lo que hace que Claude Code sea agentic»; `tools => Herramientas` |
| model / Models | modelo / Modelos | CC model-config; VS | «qué modelo y nivel de esfuerzo se ajustan»; `models => Modelos` |
| keyboard shortcuts | atajos | CC keybindings | `# Personalizar atajos de teclado` |

## Palabras de interfaz (fuente 2: VS Code)

| Inglés | Español | Clave VS Code | Valor visto |
|---|---|---|---|
| Preferences | Preferencias | `preferences` | `Preferencias` |
| Settings (sección) | Configuración | `settings` | `Configuración` |
| Theme | Tema | `selectTheme.label` | `Tema de color` |
| light / dark | claro / oscuro | `themes.category.light` / `.dark` | `temas claros` / `temas oscuros` |
| high-contrast | contraste alto | `toggleHighContrast` | `Alternar tema de contraste alto` |
| Accent colour | Color de énfasis | `chat.inputWorkingBorderColor2` | `Color de énfasis secundario…` |
| highlight colour | color de resaltado | `peekViewResultsMatchHighlight` | `…el color de resaltado…` |
| Glyphs | Glifos | `drawGlyph.label` | `Glifo de dibujo` |
| Animations | Animaciones | `workbench.reduceMotion` | `…con menos animaciones.` |
| on / off | activado / desactivado | `agentHostSessionConfig.boolean.onLabel` / `offLabel` | `Activado` / `Desactivado` |
| hidden | oculto | `hidden` | `Oculto` |
| Language | Idioma | `setDisplayLanguage` | `Establecer idioma para mostrar` |
| tab / tabs | pestaña / pestañas | `zenMode.showTabs` | `…varias pestañas del editor…` |
| lite (vista) / full | compacta / completa | `activityBarSizeCompact`; `fullscreen` | `Compacto`; `Pantalla completa` |
| filter | filtrar / filtro | `filter` | `Filtrar`, `Filtro` |
| search / find | buscar | `search`, `find` | `Buscar` |
| Help | ayuda | `help` | `Ayuda` |
| quit | salir | `quit` | `&&Salir` |
| close | cerrar | `close` | `Cerrar` |
| back | volver | `back` | `Atrás` (se usa «volver», verbo, como en las demás acciones) |
| refresh / reload | actualizar / recargar | `refresh` / `reload` | `Actualizar` / `&&Recargar` |
| install / upgrade | instalar / actualizar | `install` / `upgrade` | `Instalar` / `Actualizar` |
| delete | eliminar | `delete` | `Eliminar` |
| expand / collapse all | expandir / contraer todo | `expandAll` / `collapseAll` | `Expandir todo` / `Contraer todo` |
| folder | carpeta | `openFolder` | `Abrir carpeta` |
| trash | papelera | `deleteButtonLabelTrash` | `&&Mover a la papelera` |
| restorable | se puede restaurar | `restore` | `Puede restaurar este archivo…` |
| backup | copia de seguridad | `Open Backup folder` | `Abrir carpeta de copias de seguridad locales` |
| default | predeterminado | `default` | `Predeterminado` |
| editor | editor | `editor` | `Editor` |
| Debug | Depuración | `debug` | `Depurar`; `Mostrar vista de depuración de chat` |
| Accounts | Cuentas | `accounts` | `Cuentas` |
| Profiles | Perfiles | `profiles` | `Perfiles` |
| Security | Seguridad | `security` | `Seguridad` |
| Usage | Uso | `usage` | `Uso` |
| Context (pestaña) | Contexto | `chatContext` | `Contexto` |
| Live | En directo | `codex.sessionConfig.webSearchMode.live` | `En directo` |
| running | en ejecución | `running` | `En ejecución` |
| working | trabajando | `working` | `Trabajando` |
| Graph | Grafo | VS pt-BR `scmGraph` (no hay valor en es) | `Grafo` (ver «por analogía») |
| mouse wheel | rueda del mouse | `mouseWheelScrollSensitivity` | `…desplazamiento de la rueda del mouse` |
| scroll | desplazar | `mouseWheelScrollSensitivity` | `…eventos de desplazamiento…` |
| colours | rojo, verde, azul | `chartsRed`, `chartsGreen`, `chartsBlue` | `Color rojo…`, `Color verde…`, `Color azul…` |
| %ds / %dh / %dmin | %ds / %dh / %dmin | `seconds`, `date.fromNow.hours.singular`, `date.fromNow.minutes.singular` | `{0}s`, `{0} h`, `{0} min` |

## Sin fuente: traducido por analogía

| Inglés | Español | Motivo |
|---|---|---|
| %dd | %dd | VS Code solo tiene `{0} días`; «d» de día, compacto como s/h/min |
| work profile | perfil de trabajo | Concepto de catude; «Perfil» viene de VS Code |
| Presets | Preajustes | No hay término en VS Code ni en la documentación |
| cyan / magenta / amber | cian / magenta / ámbar | VS Code no nombra estos colores |
| mono (tema) | monocromo | Sin fuente; «mono» suelto se presta a equívoco en español |
| safe (glifos) | seguro | Sin fuente |
| Key hints | Ayuda de teclas | VS Code traduce *hint* por «Sugerencia»; «Ayuda de teclas» es más claro y corto |
| spinners | indicadores de actividad | Sin fuente |
| Overview | Resumen | VS Code dice `Información general` (19 caracteres); se acorta para la barra de pestañas |
| Discover / Inventory / Storage / Audit | Descubrir / Inventario / Almacenamiento / Auditoría | Sin fuente |
| Process / Sources | Proceso / Fuentes | Sin fuente |
| Graph | Grafo | Valor tomado del paquete pt-BR (`scmGraph => Grafo`), mismo término en español |
| browser | navegador | VS Code es usa `Explorador`, que choca con el Explorador de archivos; se usa el término habitual |
| check health (MCP) | comprobar estado | Sin fuente |
| waiting for you | esperando respuesta | Sin fuente |
| Config (pestaña) | Config | Abreviatura que se entiende igual en español |

## Añadidos el 19/11/2026 (pestaña Agentes, consumo de tokens, desactivar por proyecto)

| Inglés | Español | Fuente | Texto visto |
|---|---|---|---|
| token / tokens | token / tokens | CC costs | «reducir el uso de tokens»; «tokens de entrada servidos desde caché» |
| prompt cache / cache | caché de prompts / la caché | CC prompt-caching | `# Cómo Claude Code utiliza la caché de prompts`; `## Acciones que invalidan la caché` |
| cache reads / writes | lecturas de caché / escrituras en caché | CC costs | «principalmente lecturas de caché más algunos tokens de salida» (escrituras: por analogía) |
| (cache) expires | expira | CC prompt-caching | «las entradas expiran después de un período de inactividad» |
| compaction / compact | compactación / compactar | CC prompt-caching, costs | `### Compactar la conversación`; «compactación automática» |
| system prompt | prompt del sistema | CC prompt-caching | `* **Prompt del sistema**: la siguiente solicitud lee esta capa…` |
| headless runs (claude -p) | ejecuciones no interactivas | CC headless | «Para ejecutar Claude Code en modo no interactivo, pase `-p`…» |
| Agent SDK | Agent SDK (sin traducir) | CC headless | «Utilice el Agent SDK para ejecutar Claude Code…» |
| changelog | registro de cambios | VS `extensionEditor :: changelog` | `Registro de cambios` |
| Notification | Notificación | VS `notificationsCenter :: notifications` | `Notificaciones` |
| run / runs (de un subagente) | ejecución / ejecuciones (cabecera «ejec.») | Sin fuente: por analogía | — |
| Agent map | Mapa de agentes | Sin fuente: por analogía | — |
| built in | integrado | Sin fuente: por analogía | — |
