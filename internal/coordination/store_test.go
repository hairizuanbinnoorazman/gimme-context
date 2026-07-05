package coordination

import (
	"errors"
	"testing"
)

func TestManualIncidentLifecycle(t *testing.T) {
	store := NewStore()
	incident, err := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", []string{"checkout"})
	if err != nil {
		t.Fatal(err)
	}
	if incident.OwnerID != "alice" || incident.Lifecycle != "open" {
		t.Fatalf("unexpected incident: %+v", incident)
	}

	_, _ = store.AddMembership("workspace-a", incident.ID, "alice", "bob", "editor")
	incident, err = store.TransferOwnership("workspace-a", incident.ID, "alice", "bob")
	if err != nil {
		t.Fatal(err)
	}
	incident, err = store.UpdateIncident("workspace-a", incident.ID, "bob", "investigating", "", "")
	if incident.OwnerID != "bob" || incident.Lifecycle != "investigating" {
		t.Fatalf("unexpected update: %+v", incident)
	}
	if _, err = store.UpdateIncident("workspace-a", incident.ID, "alice", "mitigating", "", ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("former owner update error = %v, want forbidden", err)
	}
	if _, err = store.UpdateIncident("workspace-a", incident.ID, "bob", "resolved", "", ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("skipped transition error = %v, want conflict", err)
	}
}

func TestWorkspaceIsolation(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	if _, err := store.Incident("workspace-b", incident.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-workspace read error = %v, want not found", err)
	}
	if got := store.ListIncidents("workspace-b"); len(got) != 0 {
		t.Fatalf("cross-workspace list leaked %d incidents", len(got))
	}
	if _, err := store.AddPost("workspace-b", incident.ID, "mallory", "", "", []Block{{Type: "markdown", Payload: map[string]any{"text": "leak"}}}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-workspace post error = %v, want not found", err)
	}
}

func TestMembershipAndOwnershipTransfer(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	members, err := store.Memberships("workspace-a", incident.ID)
	if err != nil || len(members) != 1 || members[0].Role != "owner" {
		t.Fatalf("initial members = %+v, err = %v", members, err)
	}
	if _, err = store.AddMembership("workspace-a", incident.ID, "mallory", "bob", "editor"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner add error = %v, want forbidden", err)
	}
	if _, err = store.TransferOwnership("workspace-a", incident.ID, "alice", "bob"); !errors.Is(err, ErrConflict) {
		t.Fatalf("non-member transfer error = %v, want conflict", err)
	}
	if _, err = store.AddMembership("workspace-a", incident.ID, "alice", "bob", "editor"); err != nil {
		t.Fatal(err)
	}
	incident, err = store.TransferOwnership("workspace-a", incident.ID, "alice", "bob")
	if err != nil || incident.OwnerID != "bob" {
		t.Fatalf("transferred incident = %+v, err = %v", incident, err)
	}
	members, _ = store.Memberships("workspace-a", incident.ID)
	if members[0].Role != "editor" || members[1].Role != "owner" {
		t.Fatalf("transferred roles = %+v", members)
	}
	if _, err = store.UpdateMembership("workspace-a", incident.ID, "bob", "bob", "", true); !errors.Is(err, ErrConflict) {
		t.Fatalf("owner revocation error = %v, want conflict", err)
	}
}

func TestPostsRepliesAndRevisions(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	post, err := store.AddPost("workspace-a", incident.ID, "alice", "", "", []Block{{Type: "fact", Payload: map[string]any{"text": "errors started at 10:00"}}})
	if err != nil {
		t.Fatal(err)
	}
	if post.Revision != 1 || post.Blocks[0].ID == "" {
		t.Fatalf("unexpected post: %+v", post)
	}

	reply, err := store.AddPost("workspace-a", incident.ID, "bob", post.ID, post.Blocks[0].ID, []Block{{Type: "markdown", Payload: map[string]any{"text": "confirmed"}}})
	if err != nil {
		t.Fatal(err)
	}
	if reply.ReplyToBlockID != post.Blocks[0].ID {
		t.Fatalf("reply target = %q", reply.ReplyToBlockID)
	}
	if _, err = store.revisePost("workspace-a", incident.ID, post.ID, "bob", post.Blocks); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other author revision error = %v, want forbidden", err)
	}
	revised, err := store.revisePost("workspace-a", incident.ID, post.ID, "alice", []Block{{Type: "fact", Payload: map[string]any{"text": "errors started at 09:58"}}})
	if err != nil {
		t.Fatal(err)
	}
	if revised.Revision != 2 {
		t.Fatalf("revision = %d, want 2", revised.Revision)
	}
	feed, err := store.Feed("workspace-a", incident.ID)
	if err != nil || len(feed) != 2 {
		t.Fatalf("feed len = %d, err = %v", len(feed), err)
	}
}

func TestRejectsUnknownBlockType(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	_, err := store.AddPost("workspace-a", incident.ID, "alice", "", "", []Block{{Type: "script", Payload: map[string]any{"src": "bad"}}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want invalid", err)
	}
}

func TestFactsRequireValidEvidenceAndOwnerControlsState(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	post, _ := store.AddPost("workspace-a", incident.ID, "bob", "", "", []Block{{Type: "log", Payload: map[string]any{"text": "timeout"}}})
	if _, err := store.AddFact("workspace-a", incident.ID, "bob", "timeouts increased", []string{"missing"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing evidence error = %v, want invalid", err)
	}
	fact, err := store.AddFact("workspace-a", incident.ID, "bob", "timeouts increased", []string{post.Blocks[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpdateFact("workspace-a", incident.ID, fact.ID, "bob", "corroborated"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("participant fact update error = %v, want forbidden", err)
	}
	fact, err = store.UpdateFact("workspace-a", incident.ID, fact.ID, "alice", "corroborated")
	if err != nil || fact.State != "corroborated" {
		t.Fatalf("fact = %+v, err = %v", fact, err)
	}
}

func TestAcceptedDecisionIsImmutable(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	decision, err := store.AddDecision("workspace-a", incident.ID, "bob", "roll back", "errors followed deploy", nil)
	if err != nil {
		t.Fatal(err)
	}
	decision, err = store.Decide("workspace-a", incident.ID, decision.ID, "alice", "accepted")
	if err != nil || decision.Status != "accepted" {
		t.Fatalf("decision = %+v, err = %v", decision, err)
	}
	if _, err = store.Decide("workspace-a", incident.ID, decision.ID, "alice", "rejected"); !errors.Is(err, ErrConflict) {
		t.Fatalf("second decision error = %v, want conflict", err)
	}
}

func TestResolutionRequiresSummaryAndCompletedChecklist(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	for _, state := range []string{"investigating", "mitigating", "monitoring"} {
		incident, _ = store.UpdateIncident("workspace-a", incident.ID, "alice", state, "", "")
	}
	if _, err := store.UpdateIncident("workspace-a", incident.ID, "alice", "resolved", "", ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("unready resolution error = %v, want conflict", err)
	}
	incident, _ = store.UpdateResolution("workspace-a", incident.ID, "alice", "Rollback restored checkout.", "", nil)
	completed := true
	for _, item := range incident.ClosureChecklist {
		incident, _ = store.UpdateResolution("workspace-a", incident.ID, "alice", "", item.ID, &completed)
	}
	incident, err := store.UpdateIncident("workspace-a", incident.ID, "alice", "resolved", "", "")
	if err != nil || incident.Lifecycle != "resolved" {
		t.Fatalf("incident = %+v, err = %v", incident, err)
	}
}

func TestActionLifecycleAndOwnership(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	action, err := store.AddAction("workspace-a", incident.ID, "alice", "Roll back", "bob", "deploy.rollback", map[string]any{"service": "checkout", "version": "v2"}, "Error rate recovers")
	if err != nil || len(action.SpecificationHash) != 64 {
		t.Fatalf("action = %+v, err = %v", action, err)
	}
	if _, err = store.UpdateAction("workspace-a", incident.ID, action.ID, "mallory", "ready"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("unauthorized update = %v", err)
	}
	action.Specification.Parameters["version"] = "tampered"
	stored, _ := store.Actions("workspace-a", incident.ID)
	if stored[0].Specification.Parameters["version"] != "v2" {
		t.Fatal("returned action mutated its immutable stored specification")
	}
	for _, state := range []string{"ready", "in-progress", "verification", "completed"} {
		action, err = store.UpdateAction("workspace-a", incident.ID, action.ID, "bob", state)
		if err != nil || action.Status != state {
			t.Fatalf("state %s: action = %+v, err = %v", state, action, err)
		}
	}
	if _, err = store.UpdateAction("workspace-a", incident.ID, action.ID, "bob", "in-progress"); !errors.Is(err, ErrConflict) {
		t.Fatalf("terminal transition = %v", err)
	}
}

func TestPollEligibilityAndVoteChangePolicy(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	poll, err := store.AddPoll("workspace-a", incident.ID, "alice", "Roll back?", "advisory", []string{"Yes", "No"}, []string{"alice", "bob"}, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Vote("workspace-a", incident.ID, poll.ID, "mallory", poll.Options[0].ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ineligible vote = %v", err)
	}
	poll, err = store.Vote("workspace-a", incident.ID, poll.ID, "bob", poll.Options[0].ID)
	if err != nil || len(poll.Votes) != 1 {
		t.Fatalf("poll = %+v, err = %v", poll, err)
	}
	if _, err = store.Vote("workspace-a", incident.ID, poll.ID, "bob", poll.Options[1].ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed vote = %v", err)
	}
}

func TestApprovalBindsActionSpecificationAndRejectsReplay(t *testing.T) {
	store := NewStore()
	incident, _ := store.CreateIncident("workspace-a", "alice", "API errors", "", "SEV-2", nil)
	action, _ := store.AddAction("workspace-a", incident.ID, "alice", "Roll back", "bob", "deploy.rollback", map[string]any{"version": "v2"}, "")
	approval, err := store.RequestApproval("workspace-a", incident.ID, action.ID, "alice", []string{"carol", "dave"}, 2)
	if err != nil || approval.SpecificationHash != action.SpecificationHash {
		t.Fatalf("approval = %+v, err = %v", approval, err)
	}
	if _, err = store.RespondApproval("workspace-a", incident.ID, approval.ID, "mallory", "approve"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ineligible response = %v", err)
	}
	approval, _ = store.RespondApproval("workspace-a", incident.ID, approval.ID, "carol", "approve")
	if approval.Outcome != "pending" {
		t.Fatalf("outcome = %s", approval.Outcome)
	}
	approval, err = store.RespondApproval("workspace-a", incident.ID, approval.ID, "dave", "approve")
	if err != nil || approval.Outcome != "approved" {
		t.Fatalf("approval = %+v, err = %v", approval, err)
	}
	if _, err = store.RespondApproval("workspace-a", incident.ID, approval.ID, "carol", "approve"); !errors.Is(err, ErrConflict) {
		t.Fatalf("approval replay = %v", err)
	}
}

func TestTemplateVersionsAndIncidentSnapshotAreImmutable(t *testing.T) {
	store := NewStore()
	checklist := []ChecklistItem{{ID: "customer-impact", Label: "Customer impact documented"}}
	v1, err := store.CreateTemplateVersion("workspace-a", "alice", "", "Service incident", "Initial response", "SEV-2", []string{"api"}, checklist)
	if err != nil || v1.Version != 1 {
		t.Fatalf("v1 = %+v, err = %v", v1, err)
	}
	incident, err := store.CreateIncidentFromTemplate("workspace-a", "bob", v1.ID, 1, "API errors", "", "", nil)
	if err != nil || incident.Severity != "SEV-2" || incident.TemplateSnapshot.Version != 1 {
		t.Fatalf("incident = %+v, err = %v", incident, err)
	}
	v2, err := store.CreateTemplateVersion("workspace-a", "alice", v1.ID, "Service incident", "Revised response", "SEV-1", []string{"api", "edge"}, []ChecklistItem{{ID: "recovery", Label: "Recovery verified"}})
	if err != nil || v2.Version != 2 {
		t.Fatalf("v2 = %+v, err = %v", v2, err)
	}
	stored, _ := store.Incident("workspace-a", incident.ID)
	if stored.TemplateSnapshot.Version != 1 || stored.Severity != "SEV-2" || stored.ClosureChecklist[0].ID != "customer-impact" {
		t.Fatalf("template update changed incident snapshot: %+v", stored)
	}
	latest, _ := store.Template("workspace-a", v1.ID, 0)
	if latest.Version != 2 {
		t.Fatalf("latest version = %d", latest.Version)
	}
}

func TestTemplateWorkspaceIsolationAndValidation(t *testing.T) {
	store := NewStore()
	template, _ := store.CreateTemplateVersion("workspace-a", "alice", "", "Service incident", "", "SEV-2", nil, defaultClosureChecklist())
	if _, err := store.Template("workspace-b", template.ID, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-workspace template read = %v, want not found", err)
	}
	if _, err := store.CreateIncidentFromTemplate("workspace-b", "mallory", template.ID, 1, "Leak", "", "", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-workspace template use = %v, want not found", err)
	}
	if _, err := store.CreateTemplateVersion("workspace-a", "alice", "", "Bad", "", "SEV-2", nil, []ChecklistItem{{ID: "same", Label: "One"}, {ID: "same", Label: "Two"}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate checklist id = %v, want invalid", err)
	}
}
