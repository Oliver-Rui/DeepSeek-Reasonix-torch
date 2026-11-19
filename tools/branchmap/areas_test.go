package main

import "testing"

func TestClassifyOwnsEachArea(t *testing.T) {
	cases := map[string]string{
		"docs/STUDIO_TWEAKS_HANDOFF.md":                             "docs:",
		"tools/repolint/baseline.json":                              "docs:",
		"internal/frontend/serve/hub.go":                            "kernel:",
		"cmd/reasonix/main.go":                                      "kernel:",
		"desktop/frontend-next/src/port/hub.ts":                     "core:",
		"desktop/frontend-next/src/i18n/msg/chat.ts":                "core:",
		"desktop/frontend-next/src/styles/studio.css":               "core:",
		"desktop/frontend-next/src/ui/SubagentRun.tsx":              "collab:",
		"desktop/frontend-next/src/ui/FeedbackForm.tsx":             "collab:",
		"desktop/frontend-next/src/styles/memory.css":               "collab:",
		"desktop/frontend-next/src/ui/Settings.tsx":                 "settings:",
		"desktop/frontend-next/src/ui/ModelUsage.tsx":               "settings:",
		"desktop/frontend-next/src/design/settings.css":             "settings:",
		"desktop/frontend-next/src/ui/WorkbenchPanel.tsx":           "workbench:",
		"desktop/frontend-next/src/ui/BrowserPanel.tsx":             "workbench:",
		"desktop/frontend-next/src/ui/document/DocumentPreview.tsx": "workbench:",
		"desktop/frontend-next/src/ui/Sidebar.tsx":                  "sidebar:",
		"desktop/frontend-next/src/ui/Workspaces.tsx":               "sidebar:",
		"desktop/frontend-next/src/ui/RemoteProjectMenu.tsx":        "sidebar:",
		"desktop/frontend-next/src/styles/session-batch.css":        "sidebar:",
		"desktop/frontend-next/src/ui/Pane.tsx":                     "conversation:",
		"desktop/frontend-next/src/ui/MeterRail.tsx":                "conversation:",
		"desktop/frontend-next/src/ui/cards/SayCard.tsx":            "conversation:",
		"desktop/frontend-next/src/styles/composer-menu.css":        "conversation:",
		"desktop/frontend-next/src/ui/App.tsx":                      "chrome:",
		"desktop/frontend-next/src/ui/SessionOverviewCard.tsx":      "chrome:",
		"desktop/electron/src/main.ts":                              "chrome:",
		"desktop/frontend-next/src/ui/SomeNewThing.tsx":             "core:",
		"desktop\\frontend-next\\src\\ui\\Sidebar.tsx":              "sidebar:",
	}
	for file, want := range cases {
		area, _ := Classify(file)
		if area.Prefix != want {
			t.Errorf("Classify(%q) = %s, want %s", file, area.Prefix, want)
		}
	}
}

func TestClassifyReportsUnmatched(t *testing.T) {
	area, matched := Classify("desktop/frontend-next/src/ui/SomeNewThing.tsx")
	if matched {
		t.Fatal("expected an unclaimed path so the caller can report it")
	}
	if area.Prefix != "core:" {
		t.Fatalf("fallback = %s, want core:", area.Prefix)
	}
}
