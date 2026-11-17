package boot

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
)

// A dispatch role enters the system prompt as an identity prefix, the same way
// a persona does. An ordinary session built without one has byte-identical
// prompt bytes to the same build before this field existed.
func TestRoleTextFoldsIntoSystemPromptOnlyForRoleSessions(t *testing.T) {
	home := testenv.TempDir(t)
	t.Setenv("REASONIX_HOME", home)
	dir := testenv.TempDir(t)
	writeFile(t, dir, "reasonix.toml", `
default_model = "test-model"

[agent]
system_prompt = "BASE SYSTEM PROMPT"

[[providers]]
name = "test-model"
kind = "openai"
base_url = "https://example.invalid"
model = "x"
api_key_env = "REASONIX_TEST_KEY_UNSET"
`)
	approveWorkspace(t, dir)

	plain, err := Build(context.Background(), Options{WorkspaceRoot: dir, SessionDir: filepath.Join(dir, "sessions")})
	if err != nil {
		t.Fatalf("Build plain: %v", err)
	}
	defer plain.Close()
	plainSys := systemMessage(plain.History())
	if strings.Contains(plainSys, "Dispatch role:") {
		t.Fatalf("ordinary session carries a dispatch role:\n%s", plainSys)
	}

	role, err := Build(context.Background(), Options{WorkspaceRoot: dir, SessionDir: filepath.Join(dir, "sessions"), RoleText: "reviewer"})
	if err != nil {
		t.Fatalf("Build role: %v", err)
	}
	defer role.Close()
	roleSys := systemMessage(role.History())
	if !strings.Contains(roleSys, "Dispatch role: reviewer") {
		t.Fatalf("role session's system prompt missing its role:\n%s", roleSys)
	}
	if !strings.HasPrefix(roleSys, "Dispatch role: reviewer") {
		t.Fatalf("role is not an identity prefix:\n%s", roleSys)
	}
}
