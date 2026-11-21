package main

import "testing"

func TestClassifyOwnsEachArea(t *testing.T) {
	cases := map[string]string{
		"docs/STUDIO_TWEAKS_HANDOFF.md":                             "area/docs",
		"tools/repolint/baseline.json":                              "area/docs",
		"internal/frontend/serve/hub.go":                            "area/kernel",
		"cmd/reasonix/main.go":                                      "area/kernel",
		"desktop/frontend-next/src/port/hub.ts":                     "area/core",
		"desktop/frontend-next/src/i18n/msg/chat.ts":                "area/core",
		"desktop/frontend-next/src/styles/studio.css":               "area/core",
		"desktop/frontend-next/src/ui/DispatchThreadsBar.tsx":       "area/collab",
		"desktop/frontend-next/src/ui/PeerMessages.tsx":             "area/collab",
		"desktop/frontend-next/src/styles/peerthreads.css":          "area/collab",
		"desktop/frontend-next/src/ui/Settings.tsx":                 "area/settings",
		"desktop/frontend-next/src/ui/ModelUsage.tsx":               "area/settings",
		"desktop/frontend-next/src/design/settings.css":             "area/settings",
		"desktop/frontend-next/src/ui/WorkbenchPanel.tsx":           "area/workbench",
		"desktop/frontend-next/src/ui/BrowserPanel.tsx":             "area/workbench",
		"desktop/frontend-next/src/ui/document/DocumentPreview.tsx": "area/workbench",
		"desktop/frontend-next/src/ui/Sidebar.tsx":                  "area/sidebar",
		"desktop/frontend-next/src/ui/Workspaces.tsx":               "area/sidebar",
		"desktop/frontend-next/src/ui/RemoteProjectMenu.tsx":        "area/sidebar",
		"desktop/frontend-next/src/styles/session-batch.css":        "area/sidebar",
		"desktop/frontend-next/src/ui/Pane.tsx":                     "area/conversation",
		"desktop/frontend-next/src/ui/MeterRail.tsx":                "area/conversation",
		"desktop/frontend-next/src/ui/cards/SayCard.tsx":            "area/conversation",
		"desktop/frontend-next/src/styles/composer-menu.css":        "area/conversation",
		"desktop/frontend-next/src/ui/App.tsx":                      "area/chrome",
		"desktop/frontend-next/src/ui/SessionOverviewCard.tsx":      "area/chrome",
		"desktop/electron/src/main.ts":                              "area/chrome",
		"desktop/frontend-next/src/ui/SomeNewThing.tsx":             "area/core",
		"desktop\\frontend-next\\src\\ui\\Sidebar.tsx":              "area/sidebar",
	}
	for file, want := range cases {
		area, _ := Classify(file)
		if area.Branch != want {
			t.Errorf("Classify(%q) = %s, want %s", file, area.Branch, want)
		}
	}
}

func TestClassifyReportsUnmatched(t *testing.T) {
	area, matched := Classify("desktop/frontend-next/src/ui/SomeNewThing.tsx")
	if matched {
		t.Fatal("expected an unclaimed path so the caller can report it")
	}
	if area.Branch != "area/core" {
		t.Fatalf("fallback = %s, want area/core", area.Branch)
	}
}
