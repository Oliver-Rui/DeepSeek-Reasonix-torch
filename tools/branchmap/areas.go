package main

import (
	"path"
	"strings"
)

// Area names one long-lived module branch. Patterns are checked in order and
// the first match wins, so a specific component must appear before the shared
// fallback it would otherwise be swallowed by.
type Area struct {
	Name     string
	Branch   string
	Patterns []string
	Note     string
}

var areas = []Area{
	{
		Name:   "docs",
		Branch: "area/docs",
		Patterns: []string{
			"docs/", "release-notes/", "tools/", ".github/", "scripts/",
			"README", "README.zh-CN.md", "CONTRIBUTING.md", "SECURITY.md",
			"design-qa.md", "Makefile", ".gitignore",
			"desktop/frontend-next/perf/",
		},
		Note: "handoff, release notes, developer tools and CI",
	},
	{
		Name:   "collab",
		Branch: "area/collab",
		Patterns: []string{
			"desktop/frontend-next/src/ui/Feedback", "desktop/frontend-next/src/ui/feedback",
			"desktop/frontend-next/src/ui/Peer", "desktop/frontend-next/src/ui/peer",
			"desktop/frontend-next/src/ui/HeldPeerMessage",
			"desktop/frontend-next/src/ui/DispatchThreadsBar", "desktop/frontend-next/src/ui/dispatch-threads-bar",
			"desktop/frontend-next/src/ui/Remotes", "desktop/frontend-next/src/ui/RemoteHosts",
			"desktop/frontend-next/src/ui/RemoteDirs", "desktop/frontend-next/src/ui/RemoteNotice",
			"desktop/frontend-next/src/ui/RemoteAsk",
			"desktop/frontend-next/src/ui/Memory", "desktop/frontend-next/src/ui/MemoryDetail",
			"desktop/frontend-next/src/ui/MemoryProactiveRow", "desktop/frontend-next/src/ui/MemorySuggestRow",
			"desktop/frontend-next/src/ui/memory-", "desktop/frontend-next/src/ui/usePeers",
			"desktop/frontend-next/src/ui/peerthreads",
			"desktop/frontend-next/src/styles/peerthreads.css",
			"desktop/frontend-next/src/styles/peers.css",
			"desktop/frontend-next/src/styles/feedback.css",
			"desktop/frontend-next/src/styles/community.css",
			"desktop/frontend-next/src/styles/memory.css",
		},
		Note: "sessions talking to sessions, feedback, peers, remote and memory",
	},
	{
		Name:   "settings",
		Branch: "area/settings",
		Patterns: []string{
			"desktop/frontend-next/src/ui/Settings", "desktop/frontend-next/src/ui/settings-",
			"desktop/frontend-next/src/ui/setting", "desktop/frontend-next/src/ui/prefsnav",
			"desktop/frontend-next/src/ui/PageIndex", "desktop/frontend-next/src/ui/useSettingsSearch",
			"desktop/frontend-next/src/ui/Account", "desktop/frontend-next/src/ui/AgentSettings",
			"desktop/frontend-next/src/ui/Appearance", "desktop/frontend-next/src/ui/Backup",
			"desktop/frontend-next/src/ui/ChatDisplaySettings", "desktop/frontend-next/src/ui/CommonSettings",
			"desktop/frontend-next/src/ui/ConnectSettings", "desktop/frontend-next/src/ui/DataSettings",
			"desktop/frontend-next/src/ui/EditConn", "desktop/frontend-next/src/ui/Ext",
			"desktop/frontend-next/src/ui/ExtAddMenu", "desktop/frontend-next/src/ui/GlobalShortcutSection",
			"desktop/frontend-next/src/ui/Hero", "desktop/frontend-next/src/ui/InstalledLocation",
			"desktop/frontend-next/src/ui/Market", "desktop/frontend-next/src/ui/MarketCache",
			"desktop/frontend-next/src/ui/MarketConfirm", "desktop/frontend-next/src/ui/MarketOwn",
			"desktop/frontend-next/src/ui/MarketPublish", "desktop/frontend-next/src/ui/MarketVote",
			"desktop/frontend-next/src/ui/MessageTimesSetting", "desktop/frontend-next/src/ui/Model",
			"desktop/frontend-next/src/ui/Network", "desktop/frontend-next/src/ui/OpsSettings",
			"desktop/frontend-next/src/ui/Persona", "desktop/frontend-next/src/ui/Phone",
			"desktop/frontend-next/src/ui/Policy", "desktop/frontend-next/src/ui/Preset",
			"desktop/frontend-next/src/ui/Provider", "desktop/frontend-next/src/ui/SafetySettings",
			"desktop/frontend-next/src/ui/ServerRow", "desktop/frontend-next/src/ui/Skill",
			"desktop/frontend-next/src/ui/Storage", "desktop/frontend-next/src/ui/Theme",
			"desktop/frontend-next/src/ui/UsageSettings", "desktop/frontend-next/src/ui/Wallpaper",
			"desktop/frontend-next/src/ui/WindowSection", "desktop/frontend-next/src/ui/Shell.tsx",
			"desktop/frontend-next/src/ui/AddPlugin", "desktop/frontend-next/src/ui/AddProvider",
			"desktop/frontend-next/src/ui/AddServer", "desktop/frontend-next/src/ui/Packages",
			"desktop/frontend-next/src/ui/BalanceFields", "desktop/frontend-next/src/ui/BillingModeField",
			"desktop/frontend-next/src/ui/EndpointSwitches", "desktop/frontend-next/src/ui/Versions",
			"desktop/frontend-next/src/ui/Community", "desktop/frontend-next/src/ui/community",
			"desktop/frontend-next/src/ui/Sponsor",
			"desktop/frontend-next/src/ui/addplugin", "desktop/frontend-next/src/ui/addserver",
			"desktop/frontend-next/src/ui/extensions-page", "desktop/frontend-next/src/ui/extension-",
			"desktop/frontend-next/src/ui/ext-foot", "desktop/frontend-next/src/ui/packages-",
			"desktop/frontend-next/src/ui/plugin-commit-preview", "desktop/frontend-next/src/ui/market-",
			"desktop/frontend-next/src/ui/modelmenu", "desktop/frontend-next/src/ui/modelpicker",
			"desktop/frontend-next/src/ui/useModelCatalog", "desktop/frontend-next/src/ui/scope-row",
			"desktop/frontend-next/src/ui/serverrow-owner", "desktop/frontend-next/src/ui/skillrow-owner",
			"desktop/frontend-next/src/ui/theme-", "desktop/frontend-next/src/ui/CapabilityScope",
			"desktop/frontend-next/src/ui/hide-amounts",
			"desktop/frontend-next/src/styles/settings-type.css",
			"desktop/frontend-next/src/styles/providerpresets.css",
			"desktop/frontend-next/src/styles/modelmenu.css",
			"desktop/frontend-next/src/styles/mods.css",
			"desktop/frontend-next/src/styles/skillpin.css",
			"desktop/frontend-next/src/design/settings.css",
		},
		Note: "settings, providers, models, extensions and account surfaces",
	},
	{
		Name:   "workbench",
		Branch: "area/workbench",
		Patterns: []string{
			"desktop/frontend-next/src/ui/Workbench", "desktop/frontend-next/src/ui/workbench",
			"desktop/frontend-next/src/ui/Browser", "desktop/frontend-next/src/ui/browser",
			"desktop/frontend-next/src/ui/Terminal", "desktop/frontend-next/src/ui/terminal",
			"desktop/frontend-next/src/ui/CodeEditor", "desktop/frontend-next/src/ui/UnifiedDiff",
			"desktop/frontend-next/src/ui/ImagePreview", "desktop/frontend-next/src/ui/OpenWithMenu",
			"desktop/frontend-next/src/ui/ExplorerResize", "desktop/frontend-next/src/ui/document/",
			"desktop/frontend-next/src/ui/chart/", "desktop/frontend-next/src/ui/tree",
			"desktop/frontend-next/src/ui/useWorkbenchFiles", "desktop/frontend-next/src/ui/useGitRepos",
			"desktop/frontend-next/src/ui/recentFiles", "desktop/frontend-next/src/ui/ChangePreview",
			"desktop/frontend-next/src/ui/ChangedFilesPill", "desktop/frontend-next/src/ui/files-navigation",
			"desktop/frontend-next/src/ui/bring-result", "desktop/frontend-next/src/ui/changedpill",
			"desktop/frontend-next/src/ui/graphics-search",
			"desktop/frontend-next/src/styles/workbench-", "desktop/frontend-next/src/styles/treerows.css",
			"desktop/frontend-next/src/styles/explorer-resize.css", "desktop/frontend-next/src/styles/gitgraph.css",
			"desktop/frontend-next/src/styles/browserfull.css", "desktop/frontend-next/src/styles/chart.css",
			"desktop/frontend-next/src/styles/contentmatches.css", "desktop/frontend-next/src/styles/jsontext.css",
			"desktop/frontend-next/src/design/workbench.css", "desktop/frontend-next/src/design/terminal.css",
			"desktop/frontend-next/src/design/seltools.css",
		},
		Note: "workbench, files, browser, terminal and document preview",
	},
	{
		Name:   "sidebar",
		Branch: "area/sidebar",
		Patterns: []string{
			"desktop/frontend-next/src/ui/Sidebar", "desktop/frontend-next/src/ui/Nav",
			"desktop/frontend-next/src/ui/Workspaces", "desktop/frontend-next/src/ui/Workspace",
			"desktop/frontend-next/src/ui/ProjectsHeader", "desktop/frontend-next/src/ui/ProjectMenu",
			"desktop/frontend-next/src/ui/RemoteProjectMenu", "desktop/frontend-next/src/ui/RemoteSessionRow",
			"desktop/frontend-next/src/ui/SessionMenu", "desktop/frontend-next/src/ui/SessionFilter",
			"desktop/frontend-next/src/ui/SessionRowActions", "desktop/frontend-next/src/ui/SessionBatch",
			"desktop/frontend-next/src/ui/SessionAge", "desktop/frontend-next/src/ui/SessionQuickSection",
			"desktop/frontend-next/src/ui/SessionRef", "desktop/frontend-next/src/ui/MoveSessionDialog",
			"desktop/frontend-next/src/ui/RenameSession", "desktop/frontend-next/src/ui/OpenProject",
			"desktop/frontend-next/src/ui/BlankProjectDialog", "desktop/frontend-next/src/ui/AddWorkspacePrompt",
			"desktop/frontend-next/src/ui/Palette", "desktop/frontend-next/src/ui/railgeom",
			"desktop/frontend-next/src/ui/railfilter", "desktop/frontend-next/src/ui/railmarks",
			"desktop/frontend-next/src/ui/railsearch", "desktop/frontend-next/src/ui/rail-hierarchy",
			"desktop/frontend-next/src/ui/pinorder", "desktop/frontend-next/src/ui/recencySort",
			"desktop/frontend-next/src/ui/sessionswitch", "desktop/frontend-next/src/ui/sidebarFolds",
			"desktop/frontend-next/src/ui/sidebaropen", "desktop/frontend-next/src/ui/addws",
			"desktop/frontend-next/src/ui/workspace", "desktop/frontend-next/src/ui/remoteProjects",
			"desktop/frontend-next/src/ui/unread", "desktop/frontend-next/src/ui/RecentSessions",
			"desktop/frontend-next/src/ui/WsMark",
			"desktop/frontend-next/src/ui/sidebar-", "desktop/frontend-next/src/ui/session-batch",
			"desktop/frontend-next/src/ui/session-move", "desktop/frontend-next/src/ui/session-new",
			"desktop/frontend-next/src/ui/session-quick-setting", "desktop/frontend-next/src/ui/recent-sessions",
			"desktop/frontend-next/src/ui/nav-", "desktop/frontend-next/src/ui/remote-worktree",
			"desktop/frontend-next/src/ui/add-project-opens-session", "desktop/frontend-next/src/ui/projectchoice",
			"desktop/frontend-next/src/ui/useNewSessionPicker", "desktop/frontend-next/src/ui/useNewConversation",
			"desktop/frontend-next/src/ui/useSessionMove", "desktop/frontend-next/src/ui/useWorkspaceDrag",
			"desktop/frontend-next/src/ui/useScheduledCount", "desktop/frontend-next/src/ui/dragghost",
			"desktop/frontend-next/src/ui/timeBuckets", "desktop/frontend-next/src/ui/new-session-picker",
			"desktop/frontend-next/src/ui/worktree-project",
			"desktop/frontend-next/src/styles/session-batch.css", "desktop/frontend-next/src/styles/session-rename.css",
			"desktop/frontend-next/src/styles/sessionnotes.css", "desktop/frontend-next/src/styles/sessionpin.css",
			"desktop/frontend-next/src/styles/sidebardrop.css", "desktop/frontend-next/src/styles/workspace-path.css",
			"desktop/frontend-next/src/styles/worktree.css",
			"desktop/frontend-next/src/design/sidebar.css", "desktop/frontend-next/src/design/palette.css",
		},
		Note: "sidebar, projects, session rows, palette and navigation",
	},
	{
		Name:   "conversation",
		Branch: "area/conversation",
		Patterns: []string{
			"desktop/frontend-next/src/ui/Pane", "desktop/frontend-next/src/ui/Transcript",
			"desktop/frontend-next/src/ui/Composer", "desktop/frontend-next/src/ui/MeterRail",
			"desktop/frontend-next/src/ui/cards/", "desktop/frontend-next/src/ui/Plan",
			"desktop/frontend-next/src/ui/plan-", "desktop/frontend-next/src/ui/Queue",
			"desktop/frontend-next/src/ui/reply", "desktop/frontend-next/src/ui/rewind",
			"desktop/frontend-next/src/ui/drafts", "desktop/frontend-next/src/ui/usesubmit",
			"desktop/frontend-next/src/ui/turnrows", "desktop/frontend-next/src/ui/turn",
			"desktop/frontend-next/src/ui/blocks", "desktop/frontend-next/src/ui/ActivityHead",
			"desktop/frontend-next/src/ui/activity-reads", "desktop/frontend-next/src/ui/EarlierFold",
			"desktop/frontend-next/src/ui/NewAnswerLine", "desktop/frontend-next/src/ui/new-answer-line",
			"desktop/frontend-next/src/ui/WorkFold", "desktop/frontend-next/src/ui/fold-changes",
			"desktop/frontend-next/src/ui/thoughtfold", "desktop/frontend-next/src/ui/Markdown",
			"desktop/frontend-next/src/ui/LazyMarkdown", "desktop/frontend-next/src/ui/CodeBlock",
			"desktop/frontend-next/src/ui/ToolList", "desktop/frontend-next/src/ui/Metrics",
			"desktop/frontend-next/src/ui/Rail.tsx", "desktop/frontend-next/src/ui/Rail.test.tsx",
			"desktop/frontend-next/src/ui/Find", "desktop/frontend-next/src/ui/usefind",
			"desktop/frontend-next/src/ui/SidePanel", "desktop/frontend-next/src/ui/SideAskButton",
			"desktop/frontend-next/src/ui/BroughtResults", "desktop/frontend-next/src/ui/bring-",
			"desktop/frontend-next/src/ui/RuntimeBar", "desktop/frontend-next/src/ui/HistoryPageControl",
			"desktop/frontend-next/src/ui/useHistoryPages", "desktop/frontend-next/src/ui/useLiveGrace",
			"desktop/frontend-next/src/ui/panels/", "desktop/frontend-next/src/ui/context-",
			"desktop/frontend-next/src/ui/scroll-jump", "desktop/frontend-next/src/ui/paired-running",
			"desktop/frontend-next/src/ui/pane-", "desktop/frontend-next/src/ui/conversation-files",
			"desktop/frontend-next/src/ui/preview-truncated", "desktop/frontend-next/src/ui/queue",
			"desktop/frontend-next/src/ui/AskTimeout", "desktop/frontend-next/src/ui/Await",
			"desktop/frontend-next/src/ui/Compaction", "desktop/frontend-next/src/ui/compact-",
			"desktop/frontend-next/src/ui/Refine", "desktop/frontend-next/src/ui/refine",
			"desktop/frontend-next/src/ui/AgentRunView", "desktop/frontend-next/src/ui/AgentPageNote",
			"desktop/frontend-next/src/ui/AgentCompositionPicker", "desktop/frontend-next/src/ui/Message",
			"desktop/frontend-next/src/ui/user-message", "desktop/frontend-next/src/ui/Ask",
			"desktop/frontend-next/src/ui/Approval", "desktop/frontend-next/src/ui/Chat",
			"desktop/frontend-next/src/ui/transcript-", "desktop/frontend-next/src/ui/workfold",
			"desktop/frontend-next/src/ui/steer-chip", "desktop/frontend-next/src/ui/side-panel-",
			"desktop/frontend-next/src/ui/side-question-", "desktop/frontend-next/src/ui/user-images",
			"desktop/frontend-next/src/ui/panel-dead-ends", "desktop/frontend-next/src/ui/decision_source_pick",
			"desktop/frontend-next/src/ui/usePaneCallbacks", "desktop/frontend-next/src/ui/useSlotReorder",
			"desktop/frontend-next/src/ui/shell_outcome",
			"desktop/frontend-next/src/styles/composer-menu.css", "desktop/frontend-next/src/styles/composition-",
			"desktop/frontend-next/src/styles/line.css", "desktop/frontend-next/src/styles/rewind-notice.css",
			"desktop/frontend-next/src/styles/refine.css", "desktop/frontend-next/src/styles/scrolljump.css",
			"desktop/frontend-next/src/styles/compaction-live.css", "desktop/frontend-next/src/styles/compact-now.css",
			"desktop/frontend-next/src/styles/askskip.css", "desktop/frontend-next/src/styles/waitnote.css",
			"desktop/frontend-next/src/styles/stophint.css", "desktop/frontend-next/src/styles/pastekeep.css",
			"desktop/frontend-next/src/styles/bring.css", "desktop/frontend-next/src/styles/changedpill.css",
			"desktop/frontend-next/src/styles/runsignals.css", "desktop/frontend-next/src/styles/ctxitems.css",
			"desktop/frontend-next/src/styles/agentrun.css", "desktop/frontend-next/src/styles/mcpheld.css",
			"desktop/frontend-next/src/styles/overview.css", "desktop/frontend-next/src/styles/phonecomposer.css",
			"desktop/frontend-next/src/design/turn.css", "desktop/frontend-next/src/design/composer.css",
			"desktop/frontend-next/src/design/plan.css", "desktop/frontend-next/src/design/rail.css",
			"desktop/frontend-next/src/design/queue.css", "desktop/frontend-next/src/design/notices.css",
			"desktop/frontend-next/src/design/subagent.css", "desktop/frontend-next/src/design/jobs.css",
		},
		Note: "conversation, transcript, composer, plans and cards",
	},
	{
		Name:   "chrome",
		Branch: "area/chrome",
		Patterns: []string{
			"desktop/frontend-next/src/ui/App.tsx", "desktop/frontend-next/src/ui/Chrome",
			"desktop/frontend-next/src/ui/PaneTabs", "desktop/frontend-next/src/ui/Gutter",
			"desktop/frontend-next/src/ui/RailToggle", "desktop/frontend-next/src/ui/WindowControls",
			"desktop/frontend-next/src/ui/DeviceBar", "desktop/frontend-next/src/ui/ViewSwitch",
			"desktop/frontend-next/src/ui/Toast", "desktop/frontend-next/src/ui/Boundary",
			"desktop/frontend-next/src/ui/WallpaperStage", "desktop/frontend-next/src/ui/Update",
			"desktop/frontend-next/src/ui/Welcome", "desktop/frontend-next/src/ui/PhonePop",
			"desktop/frontend-next/src/ui/KeysSheet", "desktop/frontend-next/src/ui/SessionOverviewCard",
			"desktop/frontend-next/src/ui/ContextSummaryCard", "desktop/frontend-next/src/ui/DeckChips",
			"desktop/frontend-next/src/ui/AgentDeck", "desktop/frontend-next/src/ui/JobDeck",
			"desktop/frontend-next/src/ui/ProcessDeck", "desktop/frontend-next/src/ui/ServiceMark",
			"desktop/frontend-next/src/ui/ViewSwitch", "desktop/frontend-next/src/ui/chrome-",
			"desktop/frontend-next/src/ui/update-", "desktop/frontend-next/src/ui/versionFailure", "desktop/frontend-next/src/styles/dock.css",
			"desktop/frontend-next/src/styles/dialogcenter.css", "desktop/frontend-next/src/styles/notice.css",
			"desktop/frontend-next/src/styles/toasts.css", "desktop/frontend-next/src/styles/versions.css",
			"desktop/frontend-next/src/styles/narrow.css",
			"desktop/frontend-next/src/design/shell.css", "desktop/frontend-next/src/design/phone",
			"desktop/frontend-next/src/design/onboarding.css", "desktop/frontend-next/src/design/controls.css",
			"desktop/frontend-next/src/design/motion.css", "desktop/frontend-next/src/design/notices.css",
			"desktop/electron/",
		},
		Note: "app shell, tabs, top/bottom rails and window chrome",
	},
	{
		Name:   "core",
		Branch: "area/core",
		Patterns: []string{
			"desktop/frontend-next/src/port/", "desktop/frontend-next/src/i18n/",
			"desktop/frontend-next/src/state/", "desktop/frontend-next/src/styles/",
			"desktop/frontend-next/src/design/", "desktop/frontend-next/src/boot/",
			"desktop/frontend-next/src/data/", "desktop/frontend-next/src/e2e/",
			"desktop/frontend-next/src/assets/", "desktop/frontend-next/src/ui/kit/",
			"desktop/frontend-next/src/ui/StudioIcon", "desktop/frontend-next/src/ui/Clip",
			"desktop/frontend-next/src/ui/Backdrop", "desktop/frontend-next/src/ui/Spark",
			"desktop/frontend-next/src/ui/Sym", "desktop/frontend-next/src/ui/TariffChip",
			"desktop/frontend-next/src/ui/LevelBadge", "desktop/frontend-next/src/ui/ChipMirror",
			"desktop/frontend-next/src/ui/args", "desktop/frontend-next/src/ui/icons",
			"desktop/frontend-next/src/ui/menuKeys", "desktop/frontend-next/src/ui/motion",
			"desktop/frontend-next/src/ui/navdest", "desktop/frontend-next/src/ui/navrail",
			"desktop/frontend-next/src/ui/panetypes", "desktop/frontend-next/src/ui/railgeom",
			"desktop/frontend-next/src/ui/reorder", "desktop/frontend-next/src/ui/stable",
			"desktop/frontend-next/src/ui/slots", "desktop/frontend-next/src/ui/place",
			"desktop/frontend-next/src/ui/listen", "desktop/frontend-next/src/ui/keys",
			"desktop/frontend-next/src/ui/dismiss", "desktop/frontend-next/src/ui/viewport",
			"desktop/frontend-next/src/ui/fault", "desktop/frontend-next/src/ui/singleflight",
			"desktop/frontend-next/src/ui/sessionInfo", "desktop/frontend-next/src/ui/age",
			"desktop/frontend-next/src/ui/testkit", "desktop/frontend-next/src/ui/vendormarks",
			"desktop/frontend-next/src/main.tsx", "desktop/frontend-next/src/actions",
			"desktop/frontend-next/src/feedbackchannel", "desktop/frontend-next/src/testsetup",
			"desktop/frontend-next/src/errsink", "desktop/frontend-next/src/orphan",
			"desktop/frontend-next/src/pwa", "desktop/frontend-next/src/wording",
			"desktop/frontend-next/src/vendor-windows-names",
			"desktop/frontend-next/package.json", "desktop/frontend-next/pnpm-lock.yaml",
			"desktop/frontend-next/vite.config.ts", "desktop/frontend-next/tsconfig",
			"desktop/frontend-next/index.html",
		},
		Note: "shared port, i18n, state, styles, boot and primitives",
	},
	{
		Name:   "kernel",
		Branch: "area/kernel",
		Patterns: []string{
			"internal/", "cmd/", "sdk/", "benchmarks/", "workers/", "integrations/",
			"go.mod", "go.sum", "desktop/go.mod", "desktop/go.sum", "desktop/*.go",
		},
		Note: "Go kernel, commands and the desktop host",
	},
}

// Classify returns the area that owns rel, and whether a pattern claimed it.
// The caller reports unmatched paths as core rather than dropping them, so a
// new file never disappears from the next consolidated history.
func Classify(rel string) (Area, bool) {
	rel = normalize(rel)
	for _, a := range areas {
		for _, p := range a.Patterns {
			if matches(p, rel) {
				return a, true
			}
		}
	}
	return areaByName("core"), false
}

func areaByName(name string) Area {
	for _, a := range areas {
		if a.Name == name {
			return a
		}
	}
	return Area{Name: "core", Branch: "area/core"}
}

func normalize(rel string) string {
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = strings.TrimPrefix(rel, "./")
	return strings.TrimSpace(rel)
}

func matches(pattern, rel string) bool {
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(rel, pattern)
	}
	if strings.ContainsAny(pattern, "*?[") {
		ok, err := path.Match(pattern, rel)
		return err == nil && ok
	}
	return rel == pattern || strings.HasPrefix(rel, pattern)
}
