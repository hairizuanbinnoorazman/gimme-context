package coordination

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMixedWorkflowCountdownStopRestartAndAudit(t *testing.T) {
	s := NewStore()
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	incident, _ := s.CreateIncident("acme", "alice", "Checkout failure", "", "SEV-2", nil)
	_, _ = s.AddMembership("acme", incident.ID, "alice", "bob", "editor")
	steps := []WorkflowStep{
		{ID: "triage", Name: "Human triage", Type: "human", Mode: "guided", Risk: "low", AssigneeID: "alice"},
		{ID: "rollback", Name: "Autonomous rollback", Type: "agent", Mode: "autonomous", Risk: "medium", DependsOn: []string{"triage"}, SponsorID: "alice", AuthorisedApproverIDs: []string{"alice"}, CountdownSeconds: PlatformMinimumCountdownSeconds, Envelope: AutonomyEnvelope{Targets: []string{"staging/checkout"}, Scope: []string{"deployment"}, MaxDurationSecs: 300}},
	}
	def, err := s.CreateWorkflowVersion("acme", "alice", "", "Checkout response", steps)
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.StartWorkflow("acme", incident.ID, "alice", def.ID, 1, map[string]any{"region": "sg"})
	if err != nil {
		t.Fatal(err)
	}
	run, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "start-step", "triage", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	run, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "complete-step", "triage", "", "confirmed", 0)
	if err != nil {
		t.Fatal(err)
	}
	run, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "start-step", "rollback", "", "", 0)
	if err != nil || run.Steps[1].Status != "countdown" {
		t.Fatalf("countdown run=%+v err=%v", run, err)
	}
	if _, err = s.CommandWorkflow("acme", incident.ID, run.ID, "bob", "stop-autonomy", "rollback", "safety concern", "", 0); err != nil {
		t.Fatalf("editor stop: %v", err)
	}
	if _, err = s.CommandWorkflow("acme", incident.ID, run.ID, "bob", "restart-autonomy", "rollback", "", "", 0); err != ErrForbidden {
		t.Fatalf("unauthorised restart err=%v", err)
	}
	run, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "restart-autonomy", "rollback", "approved restart", "", 0)
	if err != nil || run.Steps[1].Status != "pending" {
		t.Fatalf("restart run=%+v err=%v", run, err)
	}
	run, _ = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "start-step", "rollback", "", "", 0)
	if _, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "complete-step", "rollback", "", "done", 0); err != ErrConflict {
		t.Fatalf("early countdown completion err=%v", err)
	}
	now = now.Add(time.Duration(PlatformMinimumCountdownSeconds) * time.Second)
	run, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "complete-step", "rollback", "", "done", 0)
	if err != nil || run.Status != "completed" {
		t.Fatalf("completed run=%+v err=%v", run, err)
	}
	if len(run.Transitions) != 8 {
		t.Fatalf("transitions=%d want 8", len(run.Transitions))
	}
}

func TestWorkflowVersionMigrationIsExplicitAndSimulationDoesNotPersist(t *testing.T) {
	s := NewStore()
	incident, _ := s.CreateIncident("acme", "alice", "Latency", "", "SEV-2", nil)
	v1, _ := s.CreateWorkflowVersion("acme", "alice", "", "Response", []WorkflowStep{{ID: "inspect", Name: "Inspect", Type: "human", Risk: "low"}})
	sim, err := s.SimulateWorkflow("acme", v1.ID, 1, map[string]any{"sample": true})
	if err != nil || len(sim.ExecutionOrder) != 1 {
		t.Fatalf("simulation=%+v err=%v", sim, err)
	}
	if len(s.workflowRuns[incident.ID]) != 0 {
		t.Fatal("simulation persisted a run")
	}
	v2, err := s.CreateWorkflowVersion("acme", "alice", v1.ID, "Response", []WorkflowStep{{ID: "inspect", Name: "Inspect", Type: "human", Risk: "low"}, {ID: "verify", Name: "Verify", Type: "approval", Risk: "low", DependsOn: []string{"inspect"}}})
	if err != nil {
		t.Fatal(err)
	}
	run, _ := s.StartWorkflow("acme", incident.ID, "alice", v1.ID, 1, nil)
	if _, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "migrate", "", "add verification", "", v2.Version); err != ErrConflict {
		t.Fatalf("running migration err=%v", err)
	}
	run, _ = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "pause", "", "", "", 0)
	run, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "migrate", "", "add verification", "", v2.Version)
	if err != nil || run.DefinitionVersion != 2 || len(run.Steps) != 2 {
		t.Fatalf("migration=%+v err=%v", run, err)
	}
}

func TestWorkflowHTTPAcceptance(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	do := func(method, path, body string, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("X-Principal-ID", "alice")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s=%d %s", method, path, rec.Code, rec.Body.String())
		}
		return rec.Body.Bytes()
	}
	var incident Incident
	_ = json.Unmarshal(do("POST", "/api/v1/workspaces/acme/incidents", `{"title":"Failure","severity":"SEV-2"}`, 201), &incident)
	var def WorkflowDefinition
	_ = json.Unmarshal(do("POST", "/api/v1/workspaces/acme/workflow-definitions", `{"name":"Response","steps":[{"id":"triage","name":"Triage","type":"human","mode":"guided","risk":"low","dependsOn":[],"autonomyEnvelope":{}}]}`, 201), &def)
	base := "/api/v1/workspaces/acme/incidents/" + incident.ID + "/workflow-runs"
	var run WorkflowRun
	_ = json.Unmarshal(do("POST", base, `{"definitionId":"`+def.ID+`","definitionVersion":1,"variables":{}}`, 201), &run)
	do("POST", base+"/"+run.ID+"/commands", `{"command":"start-step","stepId":"triage"}`, 200)
	projection := do("GET", base+"/"+run.ID, "", 200)
	if !bytes.Contains(projection, []byte(`"checklist"`)) || !bytes.Contains(projection, []byte(`"flow"`)) {
		t.Fatalf("projection=%s", projection)
	}
}

func TestWorkflowPolicyValidationAndProhibitedExecution(t *testing.T) {
	s := NewStore()
	if _, err := s.CreateWorkflowVersion("acme", "alice", "", "Unsafe countdown", []WorkflowStep{{ID: "x", Name: "X", Type: "agent", Mode: "autonomous", Risk: "medium", SponsorID: "alice", CountdownSeconds: PlatformMinimumCountdownSeconds - 1, Envelope: AutonomyEnvelope{Targets: []string{"staging"}, MaxDurationSecs: 60}}}); err != ErrInvalid {
		t.Fatalf("short countdown err=%v", err)
	}
	if _, err := s.CreateWorkflowVersion("acme", "alice", "", "Cycle", []WorkflowStep{{ID: "a", Name: "A", Type: "human", DependsOn: []string{"b"}}, {ID: "b", Name: "B", Type: "human", DependsOn: []string{"a"}}}); err != ErrInvalid {
		t.Fatalf("cycle err=%v", err)
	}
	incident, _ := s.CreateIncident("acme", "alice", "Unsafe", "", "SEV-2", nil)
	def, err := s.CreateWorkflowVersion("acme", "alice", "", "Classified", []WorkflowStep{{ID: "delete", Name: "Delete production", Type: "human", Risk: "prohibited"}})
	if err != nil {
		t.Fatal(err)
	}
	run, _ := s.StartWorkflow("acme", incident.ID, "alice", def.ID, 1, nil)
	if _, err = s.CommandWorkflow("acme", incident.ID, run.ID, "alice", "start-step", "delete", "", "", 0); err != ErrForbidden {
		t.Fatalf("prohibited execution err=%v", err)
	}
}
