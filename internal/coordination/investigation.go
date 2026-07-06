package coordination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MergePolicy struct {
	RequiredChecks                []string `json:"requiredChecks"`
	RequiredApprovals             int      `json:"requiredApprovals"`
	RequireConversationResolution bool     `json:"requireConversationResolution"`
	AllowAutoMerge                bool     `json:"allowAutoMerge"`
}

type RepositoryConfig struct {
	WorkspaceID           string      `json:"workspaceId"`
	Repository            string      `json:"repository"`
	DefaultBranch         string      `json:"defaultBranch"`
	AllowedBrowserOrigins []string    `json:"allowedBrowserOrigins"`
	AllowedCommands       []string    `json:"allowedCommands"`
	MergePolicy           MergePolicy `json:"mergePolicy"`
	UpdatedBy             string      `json:"updatedBy"`
	UpdatedAt             time.Time   `json:"updatedAt"`
}

type SandboxRequest struct {
	SandboxID        string   `json:"sandboxId"`
	Operation        string   `json:"operation"`
	Repository       string   `json:"repository"`
	Ref              string   `json:"ref,omitempty"`
	Command          []string `json:"command,omitempty"`
	URL              string   `json:"url,omitempty"`
	ReadOnly         bool     `json:"readOnly"`
	NetworkAllowlist []string `json:"networkAllowlist"`
}
type SandboxResult struct {
	Output      string `json:"output"`
	ArtifactURI string `json:"artifactUri,omitempty"`
	ExitCode    int    `json:"exitCode"`
}
type SandboxProvider interface {
	Create(context.Context, string, time.Duration) (string, error)
	Execute(context.Context, SandboxRequest) (SandboxResult, error)
	Destroy(context.Context, string) error
}
type UnavailableSandbox struct{}

func (UnavailableSandbox) Create(context.Context, string, time.Duration) (string, error) {
	return "", ErrConflict
}
func (UnavailableSandbox) Execute(context.Context, SandboxRequest) (SandboxResult, error) {
	return SandboxResult{}, ErrConflict
}
func (UnavailableSandbox) Destroy(context.Context, string) error { return nil }

type GitHubProtection struct {
	RequiredChecks                []string `json:"requiredChecks"`
	RequiredApprovals             int      `json:"requiredApprovals"`
	RequireConversationResolution bool     `json:"requireConversationResolution"`
	AllowsAutoMerge               bool     `json:"allowsAutoMerge"`
}
type PullRequestRequest struct {
	Repository, Base, Head, Title, Body string
	Policy                              MergePolicy
}
type PullRequestResult struct {
	Number  int    `json:"number"`
	URL     string `json:"url"`
	HeadSHA string `json:"headSha"`
}
type GitHubService interface {
	Protection(context.Context, string, string) (GitHubProtection, error)
	CreateBranch(context.Context, string, string, string) (string, error)
	CreatePullRequest(context.Context, PullRequestRequest) (PullRequestResult, error)
}
type UnavailableGitHub struct{}

func (UnavailableGitHub) Protection(context.Context, string, string) (GitHubProtection, error) {
	return GitHubProtection{}, ErrConflict
}
func (UnavailableGitHub) CreateBranch(context.Context, string, string, string) (string, error) {
	return "", ErrConflict
}
func (UnavailableGitHub) CreatePullRequest(context.Context, PullRequestRequest) (PullRequestResult, error) {
	return PullRequestResult{}, ErrConflict
}

type InvestigationEvidence struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Summary     string    `json:"summary"`
	Output      string    `json:"output"`
	ArtifactURI string    `json:"artifactUri,omitempty"`
	SHA256      string    `json:"sha256"`
	ExitCode    int       `json:"exitCode"`
	Command     []string  `json:"command,omitempty"`
	CapturedAt  time.Time `json:"capturedAt"`
}
type Investigation struct {
	ID                   string                  `json:"id"`
	WorkspaceID          string                  `json:"workspaceId"`
	IncidentID           string                  `json:"incidentId"`
	ActorID              string                  `json:"actorId"`
	SandboxID            string                  `json:"sandboxId"`
	Repository           string                  `json:"repository"`
	Ref                  string                  `json:"ref"`
	Status               string                  `json:"status"`
	ReadOnly             bool                    `json:"readOnly"`
	ExpiresAt            time.Time               `json:"expiresAt"`
	Evidence             []InvestigationEvidence `json:"evidence"`
	Branch               string                  `json:"branch,omitempty"`
	CommitSHA            string                  `json:"commitSha,omitempty"`
	PatchStartedAt       *time.Time              `json:"patchStartedAt,omitempty"`
	PullRequest          *PullRequestResult      `json:"pullRequest,omitempty"`
	EffectiveMergePolicy *MergePolicy            `json:"effectiveMergePolicy,omitempty"`
	CreatedAt            time.Time               `json:"createdAt"`
	UpdatedAt            time.Time               `json:"updatedAt"`
}

func (s *Store) SetSandboxProvider(v SandboxProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v != nil {
		s.sandbox = v
	}
}
func (s *Store) SetGitHubService(v GitHubService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v != nil {
		s.github = v
	}
}

func (s *Store) ConfigureRepository(workspaceID, actorID string, in RepositoryConfig) (RepositoryConfig, error) {
	if workspaceID == "" || actorID == "" || !validRepo(in.Repository) || strings.TrimSpace(in.DefaultBranch) == "" || !validOrigins(in.AllowedBrowserOrigins) {
		return RepositoryConfig{}, ErrInvalid
	}
	if len(in.AllowedCommands) == 0 {
		in.AllowedCommands = []string{"go", "npm", "make"}
	}
	for _, c := range in.AllowedCommands {
		if c == "" || filepath.Base(c) != c {
			return RepositoryConfig{}, ErrInvalid
		}
	}
	in.WorkspaceID = workspaceID
	in.Repository = strings.TrimSpace(in.Repository)
	in.DefaultBranch = strings.TrimSpace(in.DefaultBranch)
	in.UpdatedBy = actorID
	in.UpdatedAt = s.now().UTC()
	in.AllowedBrowserOrigins = uniqueSorted(in.AllowedBrowserOrigins)
	in.AllowedCommands = uniqueSorted(in.AllowedCommands)
	in.MergePolicy.RequiredChecks = uniqueSorted(in.MergePolicy.RequiredChecks)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repositoryConfigs[workspaceID] = in
	s.record(workspaceID, actorID, "repository.configured", in.Repository, in.UpdatedAt)
	return in, nil
}
func validRepo(v string) bool {
	p := strings.Split(v, "/")
	return len(p) == 2 && p[0] != "" && p[1] != "" && !strings.ContainsAny(v, " .\\")
}
func validOrigins(values []string) bool {
	for _, raw := range values {
		u, e := url.Parse(raw)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
			return false
		}
	}
	return true
}
func uniqueSorted(in []string) []string {
	m := map[string]bool{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" {
			m[v] = true
		}
	}
	out := make([]string, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func (s *Store) StartInvestigation(ctx context.Context, workspaceID, incidentID, actorID, ref string, ttl time.Duration) (Investigation, error) {
	if ttl <= 0 || ttl > 4*time.Hour {
		return Investigation{}, ErrInvalid
	}
	s.mu.RLock()
	inc, ok := s.incidents[incidentID]
	cfg, configured := s.repositoryConfigs[workspaceID]
	member := s.activeMember(incidentID, actorID)
	provider := s.sandbox
	s.mu.RUnlock()
	if !ok || inc.WorkspaceID != workspaceID {
		return Investigation{}, ErrNotFound
	}
	if !member {
		return Investigation{}, ErrForbidden
	}
	if !configured {
		return Investigation{}, ErrConflict
	}
	if ref == "" {
		ref = cfg.DefaultBranch
	}
	id, err := provider.Create(ctx, workspaceID, ttl)
	if err != nil {
		return Investigation{}, err
	}
	res, err := provider.Execute(ctx, SandboxRequest{SandboxID: id, Operation: "checkout", Repository: cfg.Repository, Ref: ref, ReadOnly: true, NetworkAllowlist: []string{"github.com"}})
	if err != nil {
		_ = provider.Destroy(ctx, id)
		return Investigation{}, err
	}
	now := s.now().UTC()
	inv := Investigation{ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID, ActorID: actorID, SandboxID: id, Repository: cfg.Repository, Ref: ref, Status: "investigating", ReadOnly: true, ExpiresAt: now.Add(ttl), CreatedAt: now, UpdatedAt: now}
	inv.Evidence = append(inv.Evidence, newEvidence("checkout", "Read-only repository checkout", res, nil, now))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.investigations[incidentID] = append(s.investigations[incidentID], inv)
	s.record(workspaceID, actorID, "investigation.started", inv.ID, now)
	return cloneInvestigation(inv), nil
}
func newEvidence(kind, summary string, res SandboxResult, cmd []string, at time.Time) InvestigationEvidence {
	h := sha256.Sum256([]byte(kind + "\x00" + summary + "\x00" + res.Output + "\x00" + res.ArtifactURI))
	return InvestigationEvidence{ID: newID(), Kind: kind, Summary: summary, Output: res.Output, ArtifactURI: res.ArtifactURI, SHA256: hex.EncodeToString(h[:]), ExitCode: res.ExitCode, Command: append([]string(nil), cmd...), CapturedAt: at}
}

func (s *Store) ExecuteInvestigation(ctx context.Context, workspaceID, incidentID, investigationID, actorID, kind, summary string, command []string, targetURL string) (Investigation, error) {
	s.mu.RLock()
	inv, idx, err := s.findInvestigation(workspaceID, incidentID, investigationID)
	cfg := s.repositoryConfigs[workspaceID]
	provider := s.sandbox
	member := s.activeMember(incidentID, actorID)
	s.mu.RUnlock()
	if err != nil {
		return Investigation{}, err
	}
	if !member {
		return Investigation{}, ErrForbidden
	}
	if s.now().UTC().After(inv.ExpiresAt) || inv.Status == "destroyed" {
		return Investigation{}, ErrConflict
	}
	req := SandboxRequest{SandboxID: inv.SandboxID, Repository: inv.Repository, Ref: inv.Ref, ReadOnly: inv.ReadOnly, NetworkAllowlist: []string{"github.com"}}
	switch kind {
	case "diagnostic", "test", "patch":
		if len(command) == 0 || !containsString(cfg.AllowedCommands, filepath.Base(command[0])) {
			return Investigation{}, ErrForbidden
		}
		if kind == "patch" && inv.ReadOnly {
			return Investigation{}, ErrConflict
		}
		req.Operation = kind
		req.Command = append([]string(nil), command...)
	case "browser":
		if !allowedBrowser(targetURL, cfg.AllowedBrowserOrigins) {
			return Investigation{}, ErrForbidden
		}
		req.Operation = "browser"
		req.URL = targetURL
		req.NetworkAllowlist = append([]string(nil), cfg.AllowedBrowserOrigins...)
	default:
		return Investigation{}, ErrInvalid
	}
	res, err := provider.Execute(ctx, req)
	if err != nil {
		return Investigation{}, err
	}
	if kind == "browser" && res.ArtifactURI == "" {
		return Investigation{}, ErrConflict
	}
	now := s.now().UTC()
	inv.Evidence = append(inv.Evidence, newEvidence(kind, strings.TrimSpace(summary), res, command, now))
	inv.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	s.investigations[incidentID][idx] = inv
	s.record(workspaceID, actorID, "investigation."+kind, inv.ID, now)
	return cloneInvestigation(inv), nil
}
func allowedBrowser(raw string, origins []string) bool {
	u, e := url.Parse(raw)
	if e != nil {
		return false
	}
	origin := u.Scheme + "://" + u.Host
	return u.Scheme == "https" && containsString(origins, origin)
}
func containsString(v []string, x string) bool {
	for _, s := range v {
		if s == x {
			return true
		}
	}
	return false
}

func (s *Store) PreparePatch(ctx context.Context, workspaceID, incidentID, id, actorID, branch string) (Investigation, error) {
	s.mu.RLock()
	inv, idx, err := s.findInvestigation(workspaceID, incidentID, id)
	gh := s.github
	member := s.activeMember(incidentID, actorID)
	s.mu.RUnlock()
	if err != nil {
		return Investigation{}, err
	}
	if !member {
		return Investigation{}, ErrForbidden
	}
	branch = strings.TrimSpace(branch)
	if !strings.HasPrefix(branch, "agent/") || strings.ContainsAny(branch, " ~^:?*[\\") {
		return Investigation{}, ErrInvalid
	}
	sha, err := gh.CreateBranch(ctx, inv.Repository, inv.Ref, branch)
	if err != nil {
		return Investigation{}, err
	}
	now := s.now().UTC()
	inv.ReadOnly = false
	inv.Status = "patching"
	inv.Branch = branch
	inv.CommitSHA = sha
	inv.PatchStartedAt = &now
	inv.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	s.investigations[incidentID][idx] = inv
	s.record(workspaceID, actorID, "investigation.patch_prepared", id, now)
	return cloneInvestigation(inv), nil
}
func (s *Store) CreateInvestigationPR(ctx context.Context, workspaceID, incidentID, id, actorID, title, body string) (Investigation, error) {
	s.mu.RLock()
	inv, idx, err := s.findInvestigation(workspaceID, incidentID, id)
	cfg := s.repositoryConfigs[workspaceID]
	gh := s.github
	member := s.activeMember(incidentID, actorID)
	s.mu.RUnlock()
	if err != nil {
		return Investigation{}, err
	}
	if !member {
		return Investigation{}, ErrForbidden
	}
	if inv.Branch == "" || strings.TrimSpace(title) == "" || !hasEvidence(inv, "diagnostic") || !hasEvidence(inv, "patch") || !hasPostPatchTest(inv) {
		return Investigation{}, ErrConflict
	}
	protection, err := gh.Protection(ctx, inv.Repository, cfg.DefaultBranch)
	if err != nil {
		return Investigation{}, err
	}
	effective := mergePolicies(cfg.MergePolicy, protection)
	body = strings.TrimSpace(body) + "\n\nEvidence\n" + evidenceMarkdown(inv.Evidence)
	pr, err := gh.CreatePullRequest(ctx, PullRequestRequest{Repository: inv.Repository, Base: cfg.DefaultBranch, Head: inv.Branch, Title: strings.TrimSpace(title), Body: body, Policy: effective})
	if err != nil {
		return Investigation{}, err
	}
	now := s.now().UTC()
	inv.PullRequest = &pr
	inv.EffectiveMergePolicy = &effective
	inv.Status = "pull_request_open"
	inv.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	s.investigations[incidentID][idx] = inv
	s.record(workspaceID, actorID, "investigation.pull_request_created", id, now)
	return cloneInvestigation(inv), nil
}
func mergePolicies(p MergePolicy, g GitHubProtection) MergePolicy {
	return MergePolicy{RequiredChecks: uniqueSorted(append(append([]string(nil), p.RequiredChecks...), g.RequiredChecks...)), RequiredApprovals: max(p.RequiredApprovals, g.RequiredApprovals), RequireConversationResolution: p.RequireConversationResolution || g.RequireConversationResolution, AllowAutoMerge: p.AllowAutoMerge && g.AllowsAutoMerge}
}
func hasEvidence(i Investigation, k string) bool {
	for _, e := range i.Evidence {
		if e.Kind == k {
			return true
		}
	}
	return false
}
func hasPostPatchTest(i Investigation) bool {
	if i.PatchStartedAt == nil {
		return false
	}
	for _, e := range i.Evidence {
		if e.Kind == "test" && e.ExitCode == 0 && !e.CapturedAt.Before(*i.PatchStartedAt) {
			return true
		}
	}
	return false
}
func evidenceMarkdown(es []InvestigationEvidence) string {
	var b strings.Builder
	for _, e := range es {
		b.WriteString("- ")
		b.WriteString(e.Kind)
		b.WriteString(": ")
		b.WriteString(e.Summary)
		b.WriteString(" (`sha256:")
		b.WriteString(e.SHA256)
		b.WriteString("`)\n")
	}
	return b.String()
}
func (s *Store) DestroyInvestigation(ctx context.Context, w, i, id, actor string) (Investigation, error) {
	s.mu.RLock()
	inv, idx, err := s.findInvestigation(w, i, id)
	p := s.sandbox
	member := s.activeMember(i, actor)
	s.mu.RUnlock()
	if err != nil {
		return Investigation{}, err
	}
	if !member {
		return Investigation{}, ErrForbidden
	}
	if err = p.Destroy(ctx, inv.SandboxID); err != nil {
		return Investigation{}, err
	}
	now := s.now().UTC()
	inv.Status = "destroyed"
	inv.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	s.investigations[i][idx] = inv
	s.record(w, actor, "investigation.destroyed", id, now)
	return cloneInvestigation(inv), nil
}
func (s *Store) Investigations(w, i string) ([]Investigation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inc, ok := s.incidents[i]
	if !ok || inc.WorkspaceID != w {
		return nil, ErrNotFound
	}
	out := make([]Investigation, len(s.investigations[i]))
	for x := range out {
		out[x] = cloneInvestigation(s.investigations[i][x])
	}
	return out, nil
}
func (s *Store) findInvestigation(w, i, id string) (Investigation, int, error) {
	inc, ok := s.incidents[i]
	if !ok || inc.WorkspaceID != w {
		return Investigation{}, -1, ErrNotFound
	}
	for x, v := range s.investigations[i] {
		if v.ID == id {
			return cloneInvestigation(v), x, nil
		}
	}
	return Investigation{}, -1, ErrNotFound
}
func cloneInvestigation(v Investigation) Investigation {
	v.Evidence = append([]InvestigationEvidence(nil), v.Evidence...)
	for i := range v.Evidence {
		v.Evidence[i].Command = append([]string(nil), v.Evidence[i].Command...)
	}
	if v.PullRequest != nil {
		x := *v.PullRequest
		v.PullRequest = &x
	}
	if v.PatchStartedAt != nil {
		x := *v.PatchStartedAt
		v.PatchStartedAt = &x
	}
	if v.EffectiveMergePolicy != nil {
		x := *v.EffectiveMergePolicy
		x.RequiredChecks = append([]string(nil), x.RequiredChecks...)
		v.EffectiveMergePolicy = &x
	}
	return v
}
