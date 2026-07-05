package coordination

import (
	"sort"
	"strings"
	"time"
)

const PlatformMinimumCountdownSeconds = 30

var validWorkflowStepType = map[string]bool{
	"human": true, "agent": true, "condition": true, "timer": true, "parallel": true, "approval": true,
}

var validRisk = map[string]bool{"low": true, "medium": true, "high": true, "prohibited": true}

type AutonomyEnvelope struct {
	Targets          []string `json:"targets"`
	Scope            []string `json:"scope"`
	CredentialScopes []string `json:"credentialScopes"`
	ForbiddenActions []string `json:"forbiddenActions"`
	MaxDurationSecs  int      `json:"maxDurationSeconds"`
	MaxCostCents     int      `json:"maxCostCents"`
}

type WorkflowStep struct {
	ID                    string           `json:"id"`
	Name                  string           `json:"name"`
	Type                  string           `json:"type"`
	Mode                  string           `json:"mode"`
	DependsOn             []string         `json:"dependsOn"`
	AssigneeID            string           `json:"assigneeId,omitempty"`
	AgentID               string           `json:"agentId,omitempty"`
	Risk                  string           `json:"risk"`
	SponsorID             string           `json:"sponsorId,omitempty"`
	AuthorisedApproverIDs []string         `json:"authorisedApproverIds,omitempty"`
	CountdownSeconds      int              `json:"countdownSeconds,omitempty"`
	Envelope              AutonomyEnvelope `json:"autonomyEnvelope"`
}

type WorkflowDefinition struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Name        string         `json:"name"`
	Version     int            `json:"version"`
	Steps       []WorkflowStep `json:"steps"`
	CreatedBy   string         `json:"createdBy"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type WorkflowStepState struct {
	StepID       string     `json:"stepId"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Mode         string     `json:"mode"`
	Risk         string     `json:"risk"`
	Status       string     `json:"status"`
	Attempt      int        `json:"attempt"`
	CountdownEnd *time.Time `json:"countdownEndsAt,omitempty"`
	StoppedBy    string     `json:"stoppedBy,omitempty"`
	Output       string     `json:"output,omitempty"`
}

type WorkflowTransition struct {
	ID            string    `json:"id"`
	ActorID       string    `json:"actorId"`
	Command       string    `json:"command"`
	StepID        string    `json:"stepId,omitempty"`
	From          string    `json:"from,omitempty"`
	To            string    `json:"to,omitempty"`
	Justification string    `json:"justification,omitempty"`
	At            time.Time `json:"at"`
}

type WorkflowRun struct {
	ID                string               `json:"id"`
	WorkspaceID       string               `json:"workspaceId"`
	IncidentID        string               `json:"incidentId"`
	DefinitionID      string               `json:"definitionId"`
	DefinitionVersion int                  `json:"definitionVersion"`
	Status            string               `json:"status"`
	Variables         map[string]any       `json:"variables"`
	Steps             []WorkflowStepState  `json:"steps"`
	Transitions       []WorkflowTransition `json:"transitions"`
	CreatedBy         string               `json:"createdBy"`
	CreatedAt         time.Time            `json:"createdAt"`
	UpdatedAt         time.Time            `json:"updatedAt"`
}

type WorkflowProjection struct {
	Definition WorkflowDefinition  `json:"definition"`
	Run        WorkflowRun         `json:"run"`
	Flow       []WorkflowStep      `json:"flow"`
	Checklist  []WorkflowStepState `json:"checklist"`
}

type WorkflowSimulation struct {
	DefinitionID      string         `json:"definitionId"`
	DefinitionVersion int            `json:"definitionVersion"`
	Valid             bool           `json:"valid"`
	ExecutionOrder    []string       `json:"executionOrder"`
	BlockedSteps      []string       `json:"blockedSteps"`
	Variables         map[string]any `json:"variables"`
}

func (s *Store) CreateWorkflowVersion(workspaceID, actorID, definitionID, name string, steps []WorkflowStep) (WorkflowDefinition, error) {
	name = strings.TrimSpace(name)
	if workspaceID == "" || actorID == "" || name == "" || !validateWorkflow(steps) {
		return WorkflowDefinition{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	version := 1
	if definitionID == "" {
		definitionID = newID()
	} else {
		versions := s.workflowDefinitions[definitionID]
		if len(versions) == 0 || versions[0].WorkspaceID != workspaceID {
			return WorkflowDefinition{}, ErrNotFound
		}
		version = len(versions) + 1
	}
	now := s.now().UTC()
	d := WorkflowDefinition{ID: definitionID, WorkspaceID: workspaceID, Name: name, Version: version, Steps: cloneWorkflowSteps(steps), CreatedBy: actorID, CreatedAt: now}
	s.workflowDefinitions[definitionID] = append(s.workflowDefinitions[definitionID], d)
	s.record(workspaceID, actorID, "workflow_definition.version_created", definitionID, now)
	return cloneWorkflowDefinition(d), nil
}

func validateWorkflow(steps []WorkflowStep) bool {
	if len(steps) == 0 {
		return false
	}
	ids := map[string]bool{}
	for i := range steps {
		st := &steps[i]
		st.ID = strings.TrimSpace(st.ID)
		if st.ID == "" || strings.TrimSpace(st.Name) == "" || ids[st.ID] || !validWorkflowStepType[st.Type] {
			return false
		}
		ids[st.ID] = true
		if st.Mode == "" {
			st.Mode = "guided"
		}
		if st.Mode != "guided" && st.Mode != "approval-gated" && st.Mode != "autonomous" {
			return false
		}
		if st.Risk == "" {
			st.Risk = "low"
		}
		if !validRisk[st.Risk] {
			return false
		}
		if st.Mode == "autonomous" && (st.Type != "agent" || st.SponsorID == "" || len(st.Envelope.Targets) == 0 || st.Envelope.MaxDurationSecs <= 0) {
			return false
		}
		if st.Mode == "approval-gated" && len(st.AuthorisedApproverIDs) == 0 {
			return false
		}
		if st.Risk == "medium" && st.CountdownSeconds < PlatformMinimumCountdownSeconds {
			return false
		}
		if st.Risk == "high" && len(st.AuthorisedApproverIDs) == 0 {
			return false
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return false
		}
		if visited[id] {
			return true
		}
		visiting[id] = true
		for _, dep := range steps[indexStep(steps, id)].DependsOn {
			if !ids[dep] || !visit(dep) {
				return false
			}
		}
		visiting[id] = false
		visited[id] = true
		return true
	}
	for id := range ids {
		if !visit(id) {
			return false
		}
	}
	return true
}

func indexStep(steps []WorkflowStep, id string) int {
	for i := range steps {
		if steps[i].ID == id {
			return i
		}
	}
	return -1
}
func cloneWorkflowSteps(in []WorkflowStep) []WorkflowStep {
	out := append([]WorkflowStep(nil), in...)
	for i := range out {
		out[i].DependsOn = append([]string(nil), out[i].DependsOn...)
		out[i].AuthorisedApproverIDs = append([]string(nil), out[i].AuthorisedApproverIDs...)
		out[i].Envelope.Targets = append([]string(nil), out[i].Envelope.Targets...)
		out[i].Envelope.Scope = append([]string(nil), out[i].Envelope.Scope...)
		out[i].Envelope.CredentialScopes = append([]string(nil), out[i].Envelope.CredentialScopes...)
		out[i].Envelope.ForbiddenActions = append([]string(nil), out[i].Envelope.ForbiddenActions...)
	}
	return out
}
func cloneWorkflowDefinition(in WorkflowDefinition) WorkflowDefinition {
	in.Steps = cloneWorkflowSteps(in.Steps)
	return in
}

func (s *Store) WorkflowDefinitions(workspaceID string) []WorkflowDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []WorkflowDefinition{}
	for _, versions := range s.workflowDefinitions {
		if len(versions) > 0 && versions[0].WorkspaceID == workspaceID {
			out = append(out, cloneWorkflowDefinition(versions[len(versions)-1]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (s *Store) workflowDefinition(workspaceID, id string, version int) (WorkflowDefinition, error) {
	versions := s.workflowDefinitions[id]
	if len(versions) == 0 || versions[0].WorkspaceID != workspaceID {
		return WorkflowDefinition{}, ErrNotFound
	}
	if version == 0 {
		version = len(versions)
	}
	if version < 1 || version > len(versions) {
		return WorkflowDefinition{}, ErrNotFound
	}
	return cloneWorkflowDefinition(versions[version-1]), nil
}

func (s *Store) SimulateWorkflow(workspaceID, definitionID string, version int, variables map[string]any) (WorkflowSimulation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, err := s.workflowDefinition(workspaceID, definitionID, version)
	if err != nil {
		return WorkflowSimulation{}, err
	}
	order := topologicalOrder(d.Steps)
	return WorkflowSimulation{DefinitionID: d.ID, DefinitionVersion: d.Version, Valid: true, ExecutionOrder: order, BlockedSteps: []string{}, Variables: cloneMap(variables)}, nil
}

func topologicalOrder(steps []WorkflowStep) []string {
	done := map[string]bool{}
	out := []string{}
	for len(out) < len(steps) {
		for _, st := range steps {
			if done[st.ID] {
				continue
			}
			ready := true
			for _, d := range st.DependsOn {
				ready = ready && done[d]
			}
			if ready {
				done[st.ID] = true
				out = append(out, st.ID)
			}
		}
	}
	return out
}

func (s *Store) StartWorkflow(workspaceID, incidentID, actorID, definitionID string, version int, variables map[string]any) (WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.canEdit(incidentID, actorID) {
		return WorkflowRun{}, ErrForbidden
	}
	if inc, ok := s.incidents[incidentID]; !ok || inc.WorkspaceID != workspaceID {
		return WorkflowRun{}, ErrNotFound
	}
	d, err := s.workflowDefinition(workspaceID, definitionID, version)
	if err != nil {
		return WorkflowRun{}, err
	}
	now := s.now().UTC()
	states := make([]WorkflowStepState, len(d.Steps))
	for i, st := range d.Steps {
		states[i] = WorkflowStepState{StepID: st.ID, Name: st.Name, Type: st.Type, Mode: st.Mode, Risk: st.Risk, Status: "pending"}
	}
	r := WorkflowRun{ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID, DefinitionID: d.ID, DefinitionVersion: d.Version, Status: "running", Variables: cloneMap(variables), Steps: states, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now}
	r.Transitions = append(r.Transitions, transition(actorID, "start", "", "", "running", "", now))
	s.workflowRuns[incidentID] = append(s.workflowRuns[incidentID], r)
	s.record(workspaceID, actorID, "workflow_run.started", r.ID, now)
	return r, nil
}

func (s *Store) WorkflowRuns(workspaceID, incidentID string) ([]WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if inc, ok := s.incidents[incidentID]; !ok || inc.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	return append([]WorkflowRun(nil), s.workflowRuns[incidentID]...), nil
}

func (s *Store) WorkflowProjection(workspaceID, incidentID, runID string) (WorkflowProjection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, _, err := s.findRun(workspaceID, incidentID, runID)
	if err != nil {
		return WorkflowProjection{}, err
	}
	d, err := s.workflowDefinition(workspaceID, r.DefinitionID, r.DefinitionVersion)
	if err != nil {
		return WorkflowProjection{}, err
	}
	return WorkflowProjection{Definition: d, Run: *r, Flow: cloneWorkflowSteps(d.Steps), Checklist: append([]WorkflowStepState(nil), r.Steps...)}, nil
}

func (s *Store) CommandWorkflow(workspaceID, incidentID, runID, actorID, command, stepID, justification, output string, targetVersion int) (WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ri, err := s.findRun(workspaceID, incidentID, runID)
	if err != nil {
		return WorkflowRun{}, err
	}
	if !s.canEdit(incidentID, actorID) {
		return WorkflowRun{}, ErrForbidden
	}
	now := s.now().UTC()
	from := r.Status
	to := r.Status
	switch command {
	case "pause":
		if r.Status != "running" {
			return WorkflowRun{}, ErrConflict
		}
		r.Status = "paused"
	case "resume":
		if r.Status != "paused" {
			return WorkflowRun{}, ErrConflict
		}
		r.Status = "running"
	case "stop":
		if r.Status == "completed" || r.Status == "cancelled" {
			return WorkflowRun{}, ErrConflict
		}
		r.Status = "cancelled"
	case "migrate":
		if r.Status != "paused" || targetVersion <= r.DefinitionVersion || strings.TrimSpace(justification) == "" {
			return WorkflowRun{}, ErrConflict
		}
		nd, e := s.workflowDefinition(workspaceID, r.DefinitionID, targetVersion)
		if e != nil {
			return WorkflowRun{}, e
		}
		old := map[string]WorkflowStepState{}
		for _, st := range r.Steps {
			old[st.StepID] = st
		}
		r.Steps = make([]WorkflowStepState, len(nd.Steps))
		for i, st := range nd.Steps {
			if v, ok := old[st.ID]; ok {
				r.Steps[i] = v
			} else {
				r.Steps[i] = WorkflowStepState{StepID: st.ID, Name: st.Name, Type: st.Type, Mode: st.Mode, Risk: st.Risk, Status: "pending"}
			}
		}
		r.DefinitionVersion = targetVersion
	case "start-step", "complete-step", "fail-step", "retry-step", "skip-step", "stop-autonomy", "restart-autonomy":
		if si := indexStepState(r.Steps, stepID); si >= 0 {
			from = r.Steps[si].Status
		}
		if err := s.commandStep(r, actorID, command, stepID, justification, output, now); err != nil {
			return WorkflowRun{}, err
		}
		if si := indexStepState(r.Steps, stepID); si >= 0 {
			to = r.Steps[si].Status
		}
	default:
		return WorkflowRun{}, ErrInvalid
	}
	r.UpdatedAt = now
	if stepID == "" {
		to = r.Status
	}
	r.Transitions = append(r.Transitions, transition(actorID, command, stepID, from, to, justification, now))
	s.workflowRuns[incidentID][ri] = *r
	s.record(workspaceID, actorID, "workflow_run."+command, runID, now)
	return *r, nil
}

func (s *Store) commandStep(r *WorkflowRun, actorID, command, stepID, justification, output string, now time.Time) error {
	d, err := s.workflowDefinition(r.WorkspaceID, r.DefinitionID, r.DefinitionVersion)
	if err != nil {
		return err
	}
	si := indexStep(d.Steps, stepID)
	if si < 0 {
		return ErrNotFound
	}
	state := &r.Steps[si]
	def := d.Steps[si]
	if command != "stop-autonomy" && command != "restart-autonomy" && r.Status != "running" {
		return ErrConflict
	}
	switch command {
	case "start-step":
		if r.Status != "running" || state.Status != "pending" {
			return ErrConflict
		}
		for _, dep := range def.DependsOn {
			ds := r.Steps[indexStep(d.Steps, dep)].Status
			if ds != "completed" && ds != "skipped" {
				return ErrConflict
			}
		}
		if def.Risk == "prohibited" {
			return ErrForbidden
		}
		if def.Mode == "autonomous" && (def.SponsorID == "" || def.Envelope.MaxDurationSecs <= 0) {
			return ErrForbidden
		}
		if def.Mode == "approval-gated" && !contains(def.AuthorisedApproverIDs, actorID) {
			return ErrForbidden
		}
		if def.Risk == "high" && !contains(def.AuthorisedApproverIDs, actorID) {
			return ErrForbidden
		}
		if def.Risk == "medium" {
			end := now.Add(time.Duration(def.CountdownSeconds) * time.Second)
			state.CountdownEnd = &end
			state.Status = "countdown"
		} else {
			state.Status = "in-progress"
			state.Attempt++
		}
	case "complete-step":
		if state.Status == "countdown" {
			if state.CountdownEnd == nil || now.Before(*state.CountdownEnd) {
				return ErrConflict
			}
			state.Status = "in-progress"
			state.Attempt++
		}
		if state.Status != "in-progress" {
			return ErrConflict
		}
		state.Status = "completed"
		state.Output = output
	case "fail-step":
		if state.Status != "in-progress" && state.Status != "countdown" {
			return ErrConflict
		}
		state.Status = "failed"
		state.Output = output
	case "retry-step":
		if state.Status != "failed" {
			return ErrConflict
		}
		state.Status = "pending"
		state.CountdownEnd = nil
	case "skip-step":
		if state.Status != "pending" || strings.TrimSpace(justification) == "" {
			return ErrConflict
		}
		state.Status = "skipped"
	case "stop-autonomy":
		if def.Mode != "autonomous" || (state.Status != "in-progress" && state.Status != "countdown") {
			return ErrConflict
		}
		state.Status = "stopped"
		state.StoppedBy = actorID
	case "restart-autonomy":
		if state.Status != "stopped" || !contains(def.AuthorisedApproverIDs, actorID) {
			return ErrForbidden
		}
		state.Status = "pending"
		state.CountdownEnd = nil
		state.StoppedBy = ""
	}
	all := true
	for _, st := range r.Steps {
		if st.Status != "completed" && st.Status != "skipped" {
			all = false
		}
	}
	if all {
		r.Status = "completed"
	}
	return nil
}

func indexStepState(steps []WorkflowStepState, id string) int {
	for i := range steps {
		if steps[i].StepID == id {
			return i
		}
	}
	return -1
}

func (s *Store) findRun(workspaceID, incidentID, runID string) (*WorkflowRun, int, error) {
	if inc, ok := s.incidents[incidentID]; !ok || inc.WorkspaceID != workspaceID {
		return nil, -1, ErrNotFound
	}
	for i := range s.workflowRuns[incidentID] {
		if s.workflowRuns[incidentID][i].ID == runID {
			return &s.workflowRuns[incidentID][i], i, nil
		}
	}
	return nil, -1, ErrNotFound
}
func transition(actor, cmd, step, from, to, why string, at time.Time) WorkflowTransition {
	return WorkflowTransition{ID: newID(), ActorID: actor, Command: cmd, StepID: step, From: from, To: to, Justification: why, At: at}
}
func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
