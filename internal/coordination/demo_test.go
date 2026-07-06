package coordination

import "testing"

func TestEnableDemoWorkspaceSeedsRunnableCapabilities(t *testing.T) {
	s := NewStore()
	if err := EnableDemoWorkspace(s, "acme"); err != nil {
		t.Fatal(err)
	}
	if len(s.ContextRecipes("acme")) != 1 || len(s.Agents("acme")) != 1 || len(s.WorkflowDefinitions("acme")) != 1 {
		t.Fatal("demo workspace was not fully seeded")
	}
}
