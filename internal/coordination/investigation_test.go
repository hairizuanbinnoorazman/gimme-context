package coordination

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeSandbox struct {
	requests  []SandboxRequest
	destroyed bool
}

func (f *fakeSandbox) Create(context.Context, string, time.Duration) (string, error) {
	return "sandbox-1", nil
}
func (f *fakeSandbox) Execute(_ context.Context, r SandboxRequest) (SandboxResult, error) {
	f.requests = append(f.requests, r)
	switch r.Operation {
	case "checkout":
		return SandboxResult{Output: "checked out abc123"}, nil
	case "diagnostic":
		return SandboxResult{Output: "reproduced: expected 200, got 500", ExitCode: 1}, nil
	case "patch":
		return SandboxResult{Output: "patched handler; commit deadbeef", ExitCode: 0}, nil
	case "test":
		return SandboxResult{Output: "PASS", ExitCode: 0}, nil
	case "browser":
		return SandboxResult{Output: "HTTP 200", ArtifactURI: "evidence://shot-1"}, nil
	}
	return SandboxResult{}, ErrInvalid
}
func (f *fakeSandbox) Destroy(context.Context, string) error { f.destroyed = true; return nil }

type fakeGitHub struct{ request PullRequestRequest }

func (*fakeGitHub) Protection(context.Context, string, string) (GitHubProtection, error) {
	return GitHubProtection{RequiredChecks: []string{"security"}, RequiredApprovals: 2, RequireConversationResolution: true, AllowsAutoMerge: true}, nil
}
func (*fakeGitHub) CreateBranch(context.Context, string, string, string) (string, error) {
	return "deadbeef", nil
}
func (f *fakeGitHub) CreatePullRequest(_ context.Context, r PullRequestRequest) (PullRequestResult, error) {
	f.request = r
	return PullRequestResult{Number: 42, URL: "https://github.com/acme/service/pull/42", HeadSHA: "deadbeef"}, nil
}

func investigationFixture(t *testing.T) (*Store, *fakeSandbox, *fakeGitHub, Incident) {
	t.Helper()
	s := NewStore()
	box := &fakeSandbox{}
	gh := &fakeGitHub{}
	s.SetSandboxProvider(box)
	s.SetGitHubService(gh)
	inc, err := s.CreateIncident("ws", "owner", "known issue", "", "SEV-2", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ConfigureRepository("ws", "owner", RepositoryConfig{Repository: "acme/service", DefaultBranch: "main", AllowedBrowserOrigins: []string{"https://staging.example.com"}, AllowedCommands: []string{"go", "make"}, MergePolicy: MergePolicy{RequiredChecks: []string{"unit"}, RequiredApprovals: 1, AllowAutoMerge: true}})
	if err != nil {
		t.Fatal(err)
	}
	return s, box, gh, inc
}

func TestKnownIssueToTraceablePullRequest(t *testing.T) {
	s, box, gh, inc := investigationFixture(t)
	ctx := context.Background()
	inv, err := s.StartInvestigation(ctx, "ws", inc.ID, "owner", "main", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !inv.ReadOnly || box.requests[0].Operation != "checkout" || !box.requests[0].ReadOnly {
		t.Fatal("checkout must begin read-only")
	}
	if _, err = s.ExecuteInvestigation(ctx, "ws", inc.ID, inv.ID, "owner", "diagnostic", "Known 500 reproduced", []string{"go", "test", "./..."}, ""); err != nil {
		t.Fatal(err)
	}
	inv, err = s.PreparePatch(ctx, "ws", inc.ID, inv.ID, "owner", "agent/fix-known-500")
	if err != nil {
		t.Fatal(err)
	}
	if inv.ReadOnly || inv.CommitSHA != "deadbeef" {
		t.Fatal("patch phase was not enabled")
	}
	if _, err = s.ExecuteInvestigation(ctx, "ws", inc.ID, inv.ID, "owner", "patch", "Correct status propagation", []string{"make", "patch"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ExecuteInvestigation(ctx, "ws", inc.ID, inv.ID, "owner", "test", "Regression suite", []string{"make", "test"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ExecuteInvestigation(ctx, "ws", inc.ID, inv.ID, "owner", "browser", "Patched staging response", nil, "https://staging.example.com/health"); err != nil {
		t.Fatal(err)
	}
	inv, err = s.CreateInvestigationPR(ctx, "ws", inc.ID, inv.ID, "owner", "Fix known 500", "Verified in disposable sandbox.")
	if err != nil {
		t.Fatal(err)
	}
	if inv.PullRequest == nil || inv.PullRequest.Number != 42 || inv.Status != "pull_request_open" {
		t.Fatalf("bad PR result: %#v", inv)
	}
	if inv.EffectiveMergePolicy.RequiredApprovals != 2 || !inv.EffectiveMergePolicy.RequireConversationResolution || !containsString(inv.EffectiveMergePolicy.RequiredChecks, "unit") || !containsString(inv.EffectiveMergePolicy.RequiredChecks, "security") {
		t.Fatalf("protections were weakened: %#v", inv.EffectiveMergePolicy)
	}
	if !strings.Contains(gh.request.Body, "sha256:") || !strings.Contains(gh.request.Body, "Known 500 reproduced") {
		t.Fatalf("PR lacks reproducible evidence: %s", gh.request.Body)
	}
	for _, e := range inv.Evidence {
		if len(e.SHA256) != 64 {
			t.Fatalf("evidence not hashed: %#v", e)
		}
	}
	if _, err = s.DestroyInvestigation(ctx, "ws", inc.ID, inv.ID, "owner"); err != nil || !box.destroyed {
		t.Fatalf("sandbox not destroyed: %v", err)
	}
}

func TestInvestigationSafetyBoundaries(t *testing.T) {
	s, _, _, inc := investigationFixture(t)
	ctx := context.Background()
	inv, err := s.StartInvestigation(ctx, "ws", inc.ID, "owner", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ExecuteInvestigation(ctx, "ws", inc.ID, inv.ID, "owner", "diagnostic", "", []string{"bash", "-c", "curl production"}, ""); err != ErrForbidden {
		t.Fatalf("shell should be denied: %v", err)
	}
	if _, err = s.ExecuteInvestigation(ctx, "ws", inc.ID, inv.ID, "owner", "browser", "", nil, "https://production.example.com"); err != ErrForbidden {
		t.Fatalf("production browser should be denied: %v", err)
	}
	if _, err = s.PreparePatch(ctx, "ws", inc.ID, inv.ID, "owner", "main"); err != ErrInvalid {
		t.Fatalf("unsafe branch should be denied: %v", err)
	}
	inv, err = s.PreparePatch(ctx, "ws", inc.ID, inv.ID, "owner", "agent/safe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateInvestigationPR(ctx, "ws", inc.ID, inv.ID, "owner", "Unverified", ""); err != ErrConflict {
		t.Fatalf("unverified PR should be denied: %v", err)
	}
	if _, err = s.StartInvestigation(ctx, "ws", inc.ID, "owner", "", 5*time.Hour); err != ErrInvalid {
		t.Fatalf("unbounded sandbox should be denied: %v", err)
	}
}

func TestInvestigationRequiresIncidentMembership(t *testing.T) {
	s, _, _, inc := investigationFixture(t)
	if _, err := s.StartInvestigation(context.Background(), "ws", inc.ID, "outsider", "", time.Hour); err != ErrForbidden {
		t.Fatalf("want forbidden, got %v", err)
	}
}
