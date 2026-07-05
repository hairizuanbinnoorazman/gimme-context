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

	incident, err = store.UpdateIncident("workspace-a", incident.ID, "alice", "investigating", "", "bob")
	if err != nil {
		t.Fatal(err)
	}
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
