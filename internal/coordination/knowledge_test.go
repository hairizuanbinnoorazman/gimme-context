package coordination

import (
	"testing"
	"time"
)

func TestClosedIncidentProducesSimulatedAcceptedKnowledgeAndArchives(t *testing.T) {
	s := NewStore()
	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	channel, err := s.CreatePermanentChannel("w1", "alice", "Operations", "")
	if err != nil {
		t.Fatal(err)
	}
	incident, err := s.CreateIncident("w1", "alice", "API errors", "", "SEV-2", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Knowledge can be proposed while review is ongoing, but not published yet.
	post, err := s.AddPost("w1", incident.ID, "alice", "", "", []Block{{Type: "markdown", SchemaVersion: 1, Payload: map[string]any{"text": "rollback worked"}}})
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.agentRuns[incident.ID] = []AgentRun{{ID: "agent-run-1", WorkspaceID: "w1", IncidentID: incident.ID, Status: "completed"}}
	s.mu.Unlock()
	artifact, err := s.CreateArtifactVersion("w1", channel.ID, "", "alice", "runbook", "API recovery", "1. Inspect metrics\n2. Roll back", incident.ID, "agent-run-1", []string{post.Blocks[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.PromoteArtifact("w1", channel.ID, artifact.ID, "alice", 1); err != ErrConflict {
		t.Fatalf("promotion before review = %v", err)
	}

	simulation, err := s.SimulateRunbook("w1", channel.ID, artifact.ID, "alice", 1, map[string]string{"service": "api"})
	if err != nil || !simulation.Passed {
		t.Fatalf("simulation = %#v, %v", simulation, err)
	}

	s.mu.Lock()
	stored := s.incidents[incident.ID]
	stored.Lifecycle = "reviewed"
	s.incidents[incident.ID] = stored
	s.mu.Unlock()
	published, err := s.PromoteArtifact("w1", channel.ID, artifact.ID, "alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != "published" || published.AcceptedBy != "alice" || published.AgentRunID != "agent-run-1" {
		t.Fatalf("published = %#v", published)
	}
	archived, err := s.ArchiveIncident("w1", incident.ID, "alice")
	if err != nil || archived.Lifecycle != "archived" {
		t.Fatalf("archive = %#v, %v", archived, err)
	}

	items, err := s.Artifacts("w1", channel.ID)
	if err != nil || len(items) != 1 || items[0].Version != 1 {
		t.Fatalf("artifacts = %#v, %v", items, err)
	}
	events := s.AuditExport("w1", time.Time{}, time.Time{})
	if len(events) < 6 {
		t.Fatalf("expected complete audit, got %d events", len(events))
	}
	for _, event := range events {
		if event.WorkspaceID != "w1" {
			t.Fatalf("cross-workspace event: %#v", event)
		}
	}
}

func TestArtifactVersionsAreImmutableAndArchiveRequiresKnowledge(t *testing.T) {
	s := NewStore()
	channel, _ := s.CreatePermanentChannel("w", "owner", "SRE", "")
	incident, _ := s.CreateIncident("w", "owner", "incident", "", "SEV-3", nil)
	s.mu.Lock()
	value := s.incidents[incident.ID]
	value.Lifecycle = "reviewed"
	s.incidents[incident.ID] = value
	s.mu.Unlock()
	if _, err := s.ArchiveIncident("w", incident.ID, "owner"); err != ErrConflict {
		t.Fatalf("archive without knowledge = %v", err)
	}
	v1, err := s.CreateArtifactVersion("w", channel.ID, "", "owner", "runbook", "Recovery", "old", incident.ID, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := s.CreateArtifactVersion("w", channel.ID, v1.ID, "owner", "runbook", "Recovery", "new", incident.ID, "", nil)
	if err != nil || v2.Version != 2 || v1.Content != "old" {
		t.Fatalf("versions: %#v %#v %v", v1, v2, err)
	}
	if _, err := s.CreateArtifactVersion("w", channel.ID, v1.ID, "owner", "saved-query", "Recovery", "changed identity", incident.ID, "", nil); err != ErrConflict {
		t.Fatalf("identity mutation = %v", err)
	}
}

func TestPilotAnalyticsComparesContextAndDecisionWithBaseline(t *testing.T) {
	s := NewStore()
	start := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return start }
	incident, _ := s.CreateIncident("w", "owner", "incident", "", "SEV-2", nil)
	contextSeconds, decisionSeconds := int64(60), int64(120)
	s.mu.Lock()
	s.collections[incident.ID] = []ContextCollection{{ID: "c", Status: "completed", CompletedAt: start.Add(time.Duration(contextSeconds) * time.Second)}}
	s.decisions[incident.ID] = []Decision{{ID: "d", Status: "accepted", UpdatedAt: start.Add(time.Duration(decisionSeconds) * time.Second)}}
	s.mu.Unlock()
	if _, err := s.SetPilotBaseline("w", "owner", PilotBaseline{TimeToContextSeconds: 120, TimeToDecisionSeconds: 240}); err != nil {
		t.Fatal(err)
	}
	a := s.PilotAnalytics("w")
	if a.AverageTimeToContextSeconds == nil || *a.AverageTimeToContextSeconds != 60 || a.ContextImprovementPercent == nil || *a.ContextImprovementPercent != 50 {
		t.Fatalf("analytics = %#v", a)
	}
	if a.AverageTimeToDecisionSeconds == nil || *a.AverageTimeToDecisionSeconds != 120 || a.DecisionImprovementPercent == nil || *a.DecisionImprovementPercent != 50 {
		t.Fatalf("analytics = %#v", a)
	}
}
