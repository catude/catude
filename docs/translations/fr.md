# Glossaire catude : français (fr)

Fichier : `internal/i18n/locales/fr.json` (197 messages).

Sources :
- **CC** = documentation Claude Code d'Anthropic en français : `https://code.claude.com/docs/fr/<page>` (lue au format `.md`, 18/11/2026).
- **VS** = module linguistique VS Code : https://github.com/microsoft/vscode-loc/blob/main/i18n/vscode-language-pack-fr/translations/main.i18n.json (clé et valeur citées).
- **LG** = lazygit n'a pas de traduction française ; non utilisé pour cette langue.

Registre : VS Code en français vouvoie (« Demandez à l'assistant… », `tip.mermaid`). Dans catude, les actions sont à l'infinitif (« ouvrir », « filtrer ») et les phrases adressées à l'utilisateur vouvoient. Typographie VS Code : espace avant « : » (`Aperçu :`).

## Concepts Claude (source 1 : documentation Anthropic)

| Anglais | Français | Source | Texte vu |
|---|---|---|---|
| skill / Skills | skill / Skills (non traduit) | CC skills | `# Étendre Claude avec des skills` ; glossaire : « Claude charge un skill automatiquement » |
| subagent | sous-agent | CC sub-agents, glossary | `# Créer des sous-agents personnalisés` ; `### Sous-agent` |
| agents (onglet) | Agents | VS | `agents => Agents` |
| hook / Hooks | hook / Hooks (non traduit) | CC hooks | `# Référence des hooks` |
| MCP servers | Serveurs MCP | CC mcp | « Les serveurs MCP donnent à Claude… » |
| plugin / Plugins | plugin / Plugins | CC plugins/overview | `# Aperçu des plugins` |
| marketplaces | marketplaces | CC glossary, plugins/install | « marketplaces de plugins » |
| commands (onglet) | Commandes | CC commands | `# Commandes` |
| settings (de Claude) | paramètres | CC settings | `# Fichiers de paramètres et précédence` |
| context window / context | contexte (fenêtre de contexte) | CC context-window | `# Explorez la fenêtre de contexte` |
| session / sessions | session / Sessions | CC glossary, sessions | `### Session` ; `# Gérer les sessions` |
| resume (session) | reprendre | CC sessions | `## Reprendre une session` |
| export | exporter | CC sessions | `## Exporter et localiser les données de session` |
| transcript | transcription | CC glossary, sessions | « la transcription est cette conversation conservée sous forme de fichier » ; `### Où les transcriptions sont stockées` |
| to-do (liste de Claude) | tâches | CC interactive-mode | « Afficher/masquer la liste de tâches de Claude » |
| tool / tools | outil / Outils | CC glossary ; VS | « Chaque utilisation d'outil retourne un résultat » ; `tools => Outils` |
| model / Models | modèle / Modèles | CC model-config ; VS | « le modèle et le niveau d'effort qui conviennent » ; `models => Modèles` |
| keyboard shortcuts | raccourcis | CC keybindings ; VS | `# Personnaliser les raccourcis clavier` ; `keyboardShortcuts => Raccourcis clavier` |

## Mots d'interface (source 2 : VS Code)

| Anglais | Français | Clé VS Code | Valeur vue |
|---|---|---|---|
| Preferences | Préférences | `preferences` | `Préférences` |
| Settings (section) | Paramètres | `settings` | `Paramètres` |
| Theme | Thème | `selectTheme.label` | `Thème de couleur` |
| light / dark | clair / sombre | `themes.category.light` / `.dark` | `thèmes clairs` / `thèmes sombres` |
| high-contrast | contraste élevé | `toggleHighContrast` | `Activer/désactiver le thème à contraste élevé` |
| Accent colour | Couleur d'accent | `chat.inputWorkingBorderColor2` | `Couleur d’accentuation secondaire…` (raccourci, voir plus bas) |
| Glyphs | Glyphes | `drawGlyph.label` | `Dessiner un glyphe` |
| Animations | Animations | `workbench.reduceMotion` | `…avec moins d’animations.` |
| on / off | activé / désactivé | `agentHostSessionConfig.boolean.onLabel` / `offLabel` | `Activé` / `Désactivé` |
| hidden | masqué | `hidden` | `Masqué` |
| tab / tabs | onglet / onglets | `wrapTabs`, `zenMode.showTabs` | `…les onglets doivent être enroulés…` |
| lite (vue) | compacte | `codiconFontSizeCompact` | `Taille de police compacte…` |
| full (vue) | complète | — | voir « par analogie » |
| filter | filtrer / filtre | `filter` | `Filtrer` |
| search / find | rechercher | `search`, `find` | `Rechercher` |
| Help | aide | `help` | `Aide` |
| quit | quitter | `quit` | `&&Quitter` |
| close | fermer | `close` | `Fermer` |
| back | retour | `back` | `Retour` |
| refresh / reload | actualiser / recharger | `refresh` / `reload` | `Actualiser` / `&&Recharger` |
| install / upgrade | installer / mettre à niveau | `install` / `upgrade` | `Installer` / `Mettre à niveau` |
| delete | supprimer | `delete` | `Supprimer` |
| expand / collapse all | tout développer / tout réduire | `expandAll` / `collapseAll` | `Tout développer` / `Tout réduire` |
| folder | dossier | `openFolder` | `Ouvrir un dossier` |
| trash | corbeille | `deleteButtonLabelTrash` | `&&Déplacer vers la Corbeille` |
| backup | sauvegarde | `Open Backup folder` | `Ouvrir le dossier des sauvegardes locales` |
| default | par défaut | `default` | `Par défaut` |
| editor | éditeur | `editor` | `Éditeur` |
| Debug | Débogage | `debug` | `Afficher la vue Débogage de conversation` |
| browser | navigateur | `browser` | `Navigateur` |
| Accounts | Comptes | `accounts` | `Comptes` |
| Profiles | Profils | `profiles` | `Profils` |
| Overview | Présentation | `overview` | `Présentation` |
| Security | Sécurité | `security` | `Sécurité` |
| Usage | Utilisation | `usage` | `Utilisation` |
| Context (onglet) | Contexte | `chatContext` | `Contexte` |
| Live | En direct | `codex.sessionConfig.webSearchMode.live` | `En direct` |
| running | en cours | `running` | `En cours d'exécution` (raccourci) |
| font | police | `font` | `Police` |
| colours | rouge, vert, bleu | `chartsRed`, `chartsGreen`, `chartsBlue` | `Couleur rouge…`, `Couleur verte…`, `Couleur bleue…` |
| %ds / %dmin / %dh | %ds / %dmin / %dh | `seconds`, `duration.m`, `date.fromNow.hours.*` | `{0}s`, `{0} min`, `{0} heure(s)` → « h » |

## Sans source : traduit par analogie

| Anglais | Français | Raison |
|---|---|---|
| %dj | %dj | VS Code n'a que `{0} jours` ; « j » est l'abréviation française du jour |
| Accent colour | Couleur d'accent | VS Code dit « couleur d'accentuation » (22 caractères) ; raccourci pour tenir dans la colonne de 18 caractères des préférences |
| work profile | profil de travail | Concept de catude ; « Profil » vient de VS Code |
| Presets | Préréglages | Aucun terme dans VS Code ni dans la doc |
| cyan / magenta / amber | cyan / magenta / ambre | Couleurs absentes de VS Code |
| mono (thème) | monochrome | Sans source |
| safe (glyphes) | sûr | Sans source |
| Key hints | Aide des touches | VS Code traduit *hint* par « Conseil », peu naturel ici |
| spinners | indicateurs d'activité | Sans source |
| working (état) | en cours | VS Code dit `En état de fonctionnement`, trop long et peu naturel |
| full (vue) | complète | Sans source |
| Discover / Inventory / Storage / Audit / Graph | Découvrir / Inventaire / Stockage / Audit / Graphe | Sans source |
| Process / Sources | Processus / Sources | Sans source |
| check health (MCP) | vérifier l'état | Sans source |
| waiting for you | en attente de réponse | Sans source (la doc emploie « en attente ») |
| Config (onglet) | Config | Abréviation comprise en français |

## Ajouts du 19/11/2026 (onglet Agents, consommation de tokens, désactivation par projet)

| Anglais | Français | Source | Texte vu |
|---|---|---|---|
| token / tokens | token / tokens | CC costs | « réduire l'utilisation des tokens » ; « tokens d'entrée servis à partir du cache » |
| prompt cache / cache | cache de prompt / le cache | CC prompt-caching, costs | « réutilise le cache de prompt de la conversation » ; `## Actions qui invalident le cache` |
| cache reads / writes | lectures du cache / écritures en cache | CC prompt-caching | « ont réécrits dans le cache » (formes nominales : par analogie) |
| (cache) expires | expire | CC prompt-caching | « les entrées expirent après une période d'inactivité » |
| compaction | compaction | CC prompt-caching | `### Compaction de la conversation` |
| system prompt | prompt système | CC prompt-caching | « Au tour quatre, le prompt système a changé » |
| headless runs (claude -p) | exécutions non interactives | CC headless | « Pour exécuter Claude Code en mode non interactif, passez `-p`… » |
| Agent SDK | Agent SDK (non traduit) | CC headless | « Utilisez l'Agent SDK pour exécuter Claude Code… » |
| marketplace (genre) | une marketplace | CC plugins/install | « à partir d'une marketplace » ; « ajoutez d'abord la marketplace » |
| changelog | journal des modifications | VS `extensionEditor :: changelog` | `Journal des modifications` |
| Notification | Notification | VS `notificationsCenter :: notifications` | `Notifications` |
| run / runs (d'un sous-agent) | exécution / exécutions (en-tête « exéc. ») | Sans source : par analogie | — |
| Agent map | Carte des agents | Sans source : par analogie | — |
| built in | intégré | Sans source : par analogie | — |
