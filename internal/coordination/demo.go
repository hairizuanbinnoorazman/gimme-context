package coordination

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// EnableDemoWorkspace installs deterministic local adapters and seed definitions.
// It is intended only for the explicitly enabled Docker Compose demo environment.
func EnableDemoWorkspace(s *Store, workspaceID string) error {
	s.SetContextService(ContextService{Prometheus: demoTelemetry{source: "prometheus"}, Loki: demoTelemetry{source: "loki"}})
	s.SetModelGateway(demoModel{})
	s.SetSandboxProvider(&demoSandbox{})
	s.SetGitHubService(demoGitHub{})
	if _, err := s.ConfigureRepository(workspaceID, "demo-bootstrap", RepositoryConfig{
		Repository: "acme/gimme-context", DefaultBranch: "main",
		AllowedBrowserOrigins: []string{"https://staging.example.com"},
		AllowedCommands:       []string{"go", "make"},
		MergePolicy: MergePolicy{RequiredChecks: []string{"unit"}, RequiredApprovals: 1,
			RequireConversationResolution: true},
	}); err != nil {
		return err
	}
	if _, err := s.CreateContextRecipe(workspaceID, "demo-bootstrap", "", "Service health", []ContextQuery{
		{Name: "Error rate", Source: "prometheus", Query: `rate(http_requests_total{status=~"5.."}[5m])`, Lookback: "1h", Step: "30s", Required: true},
		{Name: "Recent errors", Source: "loki", Query: `{service="gimme-context"} |= "error"`, Lookback: "1h", Limit: 100},
	}); err != nil {
		return err
	}
	if _, err := s.CreateAgent(workspaceID, "demo-bootstrap", "Incident synthesizer", "Evidence-linked incident synthesis", "vertex-ai", "local-deterministic", []string{"synthesis"}); err != nil {
		return err
	}
	_, err := s.CreateWorkflowVersion(workspaceID, "demo-bootstrap", "", "Guided mitigation", []WorkflowStep{
		{ID: "investigate", Name: "Investigate evidence", Type: "human", Mode: "guided", Risk: "low"},
		{ID: "mitigate", Name: "Apply approved mitigation", Type: "human", Mode: "approval-gated", Risk: "medium", DependsOn: []string{"investigate"}, AuthorisedApproverIDs: []string{"alice"}, CountdownSeconds: PlatformMinimumCountdownSeconds},
		{ID: "verify", Name: "Verify recovery", Type: "human", Mode: "guided", Risk: "low", DependsOn: []string{"mitigate"}},
	})
	return err
}

type demoTelemetry struct{ source string }

func (d demoTelemetry) Query(_ context.Context, query string, start, end time.Time, _ ContextQuery) (any, string, error) {
	return map[string]any{"source": d.source, "query": query, "start": start, "end": end, "samples": []float64{0.02, 0.03, 0.01}}, "demo://" + d.source, nil
}

type demoModel struct{}

func (demoModel) Generate(_ context.Context, r ModelRequest) (ModelResponse, error) {
	return ModelResponse{Proposals: []ModelProposal{{Kind: "summary", Content: "Evidence reviewed: " + r.Task, Rationale: "Deterministic Compose demo synthesis", EvidenceBlockIDs: append([]string(nil), r.EvidenceBlockIDs...)}}, InputTokens: 32, OutputTokens: 12}, nil
}

type demoSandbox struct{ sequence atomic.Uint64 }

func (d *demoSandbox) Create(context.Context, string, time.Duration) (string, error) {
	return fmt.Sprintf("demo-sandbox-%d", d.sequence.Add(1)), nil
}
func (*demoSandbox) Execute(_ context.Context, r SandboxRequest) (SandboxResult, error) {
	switch r.Operation {
	case "checkout":
		return SandboxResult{Output: "checked out " + r.Ref}, nil
	case "diagnostic":
		return SandboxResult{Output: "reproduced failure in isolated demo sandbox", ExitCode: 1}, nil
	case "patch":
		return SandboxResult{Output: "applied bounded demo patch", ExitCode: 0}, nil
	case "test":
		return SandboxResult{Output: "PASS", ExitCode: 0}, nil
	case "browser":
		return SandboxResult{Output: "staging check passed", ArtifactURI: "evidence://demo-browser", ExitCode: 0}, nil
	default:
		return SandboxResult{}, ErrInvalid
	}
}
func (*demoSandbox) Destroy(context.Context, string) error { return nil }

type demoGitHub struct{}

func (demoGitHub) Protection(context.Context, string, string) (GitHubProtection, error) {
	return GitHubProtection{RequiredChecks: []string{"unit", "security"}, RequiredApprovals: 1, RequireConversationResolution: true}, nil
}
func (demoGitHub) CreateBranch(context.Context, string, string, string) (string, error) {
	return "demo-commit-sha", nil
}
func (demoGitHub) CreatePullRequest(_ context.Context, r PullRequestRequest) (PullRequestResult, error) {
	return PullRequestResult{Number: 1, URL: "https://github.com/" + r.Repository + "/pull/1", HeadSHA: "demo-commit-sha"}, nil
}
