package coordination

import (
	"sort"
	"strings"
	"time"
)

var artifactTypes = map[string]bool{"runbook": true, "known-issue": true, "ownership-record": true, "escalation-policy": true, "saved-query": true, "decision-record": true, "integration-recipe": true}

type Artifact struct {
	ID                 string     `json:"id"`
	WorkspaceID        string     `json:"workspaceId"`
	PermanentChannelID string     `json:"permanentChannelId"`
	Type               string     `json:"type"`
	Title              string     `json:"title"`
	Version            int        `json:"version"`
	Content            string     `json:"content"`
	Status             string     `json:"status"`
	SourceIncidentID   string     `json:"sourceIncidentId,omitempty"`
	EvidenceBlockIDs   []string   `json:"evidenceBlockIds,omitempty"`
	ProposedBy         string     `json:"proposedBy"`
	AgentRunID         string     `json:"agentRunId,omitempty"`
	AcceptedBy         string     `json:"acceptedBy,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	PublishedAt        *time.Time `json:"publishedAt,omitempty"`
}

type ArtifactSimulation struct {
	ID          string            `json:"id"`
	ArtifactID  string            `json:"artifactId"`
	Version     int               `json:"version"`
	WorkspaceID string            `json:"workspaceId"`
	Inputs      map[string]string `json:"inputs"`
	Result      string            `json:"result"`
	Passed      bool              `json:"passed"`
	SimulatedBy string            `json:"simulatedBy"`
	SimulatedAt time.Time         `json:"simulatedAt"`
}

type PilotBaseline struct {
	TimeToContextSeconds  int64 `json:"timeToContextSeconds"`
	TimeToDecisionSeconds int64 `json:"timeToFirstAcceptedDecisionSeconds"`
}
type PilotMetric struct {
	IncidentID            string `json:"incidentId"`
	TimeToContextSeconds  *int64 `json:"timeToContextSeconds,omitempty"`
	TimeToDecisionSeconds *int64 `json:"timeToFirstAcceptedDecisionSeconds,omitempty"`
}
type PilotAnalytics struct {
	Baseline                     PilotBaseline `json:"baseline"`
	Incidents                    []PilotMetric `json:"incidents"`
	AverageTimeToContextSeconds  *int64        `json:"averageTimeToContextSeconds,omitempty"`
	AverageTimeToDecisionSeconds *int64        `json:"averageTimeToFirstAcceptedDecisionSeconds,omitempty"`
	ContextImprovementPercent    *float64      `json:"timeToContextImprovementPercent,omitempty"`
	DecisionImprovementPercent   *float64      `json:"timeToFirstAcceptedDecisionImprovementPercent,omitempty"`
}

func (s *Store) CreateArtifactVersion(workspaceID, channelID, artifactID, actorID, kind, title, content, incidentID, agentRunID string, evidence []string) (Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, ok := s.channels[channelID]
	if !ok || channel.WorkspaceID != workspaceID {
		return Artifact{}, ErrNotFound
	}
	title, content = strings.TrimSpace(title), strings.TrimSpace(content)
	if actorID == "" || title == "" || content == "" || !artifactTypes[kind] {
		return Artifact{}, ErrInvalid
	}
	if incidentID != "" {
		inc, ok := s.incidents[incidentID]
		if !ok || inc.WorkspaceID != workspaceID {
			return Artifact{}, ErrNotFound
		}
		if !s.canEdit(incidentID, actorID) {
			return Artifact{}, ErrForbidden
		}
		if len(evidence) > 0 && !s.evidenceExists(incidentID, evidence) {
			return Artifact{}, ErrInvalid
		}
		if agentRunID != "" {
			found := false
			for _, run := range s.agentRuns[incidentID] {
				if run.ID == agentRunID && run.Status == "completed" {
					found = true
					break
				}
			}
			if !found {
				return Artifact{}, ErrInvalid
			}
		}
	}
	versions := s.artifacts[artifactID]
	if artifactID == "" {
		artifactID = newID()
	} else if len(versions) == 0 || versions[0].WorkspaceID != workspaceID || versions[0].PermanentChannelID != channelID {
		return Artifact{}, ErrNotFound
	}
	if len(versions) > 0 && (versions[0].Type != kind || versions[0].Title != title) {
		return Artifact{}, ErrConflict
	}
	now := s.now().UTC()
	item := Artifact{ID: artifactID, WorkspaceID: workspaceID, PermanentChannelID: channelID, Type: kind, Title: title, Version: len(versions) + 1, Content: content, Status: "proposed", SourceIncidentID: incidentID, EvidenceBlockIDs: append([]string(nil), evidence...), ProposedBy: actorID, AgentRunID: agentRunID, CreatedAt: now}
	s.artifacts[artifactID] = append(versions, item)
	s.record(workspaceID, actorID, "artifact.version_proposed", artifactID, now)
	return item, nil
}

func (s *Store) Artifacts(workspaceID, channelID string) ([]Artifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if c, ok := s.channels[channelID]; !ok || c.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	items := []Artifact{}
	for _, versions := range s.artifacts {
		if len(versions) > 0 && versions[0].PermanentChannelID == channelID {
			items = append(items, versions[len(versions)-1])
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	return items, nil
}

func (s *Store) PromoteArtifact(workspaceID, channelID, artifactID, actorID string, version int) (Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions := s.artifacts[artifactID]
	if len(versions) == 0 || versions[0].WorkspaceID != workspaceID || versions[0].PermanentChannelID != channelID {
		return Artifact{}, ErrNotFound
	}
	if version == 0 {
		version = len(versions)
	}
	if version < 1 || version > len(versions) {
		return Artifact{}, ErrNotFound
	}
	item := &s.artifacts[artifactID][version-1]
	if item.Status != "proposed" {
		return Artifact{}, ErrConflict
	}
	if item.SourceIncidentID != "" {
		inc := s.incidents[item.SourceIncidentID]
		if inc.Lifecycle != "reviewed" && inc.Lifecycle != "archived" {
			return Artifact{}, ErrConflict
		}
		if !s.canEdit(item.SourceIncidentID, actorID) {
			return Artifact{}, ErrForbidden
		}
	} else if actorID == "" {
		return Artifact{}, ErrForbidden
	}
	now := s.now().UTC()
	item.Status = "published"
	item.AcceptedBy = actorID
	item.PublishedAt = &now
	s.record(workspaceID, actorID, "artifact.promoted", artifactID, now)
	return *item, nil
}

func (s *Store) SimulateRunbook(workspaceID, channelID, artifactID, actorID string, version int, inputs map[string]string) (ArtifactSimulation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions := s.artifacts[artifactID]
	if len(versions) == 0 || versions[0].WorkspaceID != workspaceID || versions[0].PermanentChannelID != channelID {
		return ArtifactSimulation{}, ErrNotFound
	}
	if version == 0 {
		version = len(versions)
	}
	if version < 1 || version > len(versions) || versions[version-1].Type != "runbook" || actorID == "" {
		return ArtifactSimulation{}, ErrInvalid
	}
	now := s.now().UTC()
	passed := !strings.Contains(strings.ToLower(versions[version-1].Content), "{{missing}}")
	result := "simulation passed without external side effects"
	if !passed {
		result = "unresolved placeholder: missing"
	}
	sim := ArtifactSimulation{ID: newID(), ArtifactID: artifactID, Version: version, WorkspaceID: workspaceID, Inputs: inputs, Result: result, Passed: passed, SimulatedBy: actorID, SimulatedAt: now}
	s.artifactSimulations[artifactID] = append(s.artifactSimulations[artifactID], sim)
	s.record(workspaceID, actorID, "runbook.simulated", artifactID, now)
	return sim, nil
}

func (s *Store) ArchiveIncident(workspaceID, incidentID, actorID string) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inc, ok := s.incidents[incidentID]
	if !ok || inc.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if inc.OwnerID != actorID {
		return Incident{}, ErrForbidden
	}
	if inc.Lifecycle != "reviewed" {
		return Incident{}, ErrConflict
	}
	accepted := false
	for _, vs := range s.artifacts {
		for _, a := range vs {
			if a.SourceIncidentID == incidentID && a.Status == "published" {
				accepted = true
			}
		}
	}
	if !accepted {
		return Incident{}, ErrConflict
	}
	inc.Lifecycle = "archived"
	inc.UpdatedAt = s.now().UTC()
	s.incidents[incidentID] = inc
	s.record(workspaceID, actorID, "incident.archived", incidentID, inc.UpdatedAt)
	return cloneIncident(inc), nil
}

func (s *Store) AuditExport(workspaceID string, from, to time.Time) []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []AuditEvent{}
	for _, e := range s.audit {
		if e.WorkspaceID == workspaceID && (from.IsZero() || !e.At.Before(from)) && (to.IsZero() || !e.At.After(to)) {
			out = append(out, e)
		}
	}
	return out
}
func (s *Store) SetPilotBaseline(workspaceID, actorID string, b PilotBaseline) (PilotBaseline, error) {
	if actorID == "" || b.TimeToContextSeconds <= 0 || b.TimeToDecisionSeconds <= 0 {
		return PilotBaseline{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pilotBaselines[workspaceID] = b
	s.record(workspaceID, actorID, "pilot.baseline_set", workspaceID, s.now().UTC())
	return b, nil
}
func (s *Store) PilotAnalytics(workspaceID string) PilotAnalytics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := PilotAnalytics{Baseline: s.pilotBaselines[workspaceID], Incidents: []PilotMetric{}}
	var cs, ds int64
	var cn, dn int64
	for _, inc := range s.incidents {
		if inc.WorkspaceID != workspaceID {
			continue
		}
		m := PilotMetric{IncidentID: inc.ID}
		var cAt, dAt time.Time
		for _, c := range s.collections[inc.ID] {
			if c.Status == "completed" && (cAt.IsZero() || c.CompletedAt.Before(cAt)) {
				cAt = c.CompletedAt
			}
		}
		for _, d := range s.decisions[inc.ID] {
			if d.Status == "accepted" && (dAt.IsZero() || d.UpdatedAt.Before(dAt)) {
				dAt = d.UpdatedAt
			}
		}
		if !cAt.IsZero() {
			v := int64(cAt.Sub(inc.CreatedAt).Seconds())
			if v < 0 {
				v = 0
			}
			m.TimeToContextSeconds = &v
			cs += v
			cn++
		}
		if !dAt.IsZero() {
			v := int64(dAt.Sub(inc.CreatedAt).Seconds())
			if v < 0 {
				v = 0
			}
			m.TimeToDecisionSeconds = &v
			ds += v
			dn++
		}
		out.Incidents = append(out.Incidents, m)
	}
	if cn > 0 {
		v := cs / cn
		out.AverageTimeToContextSeconds = &v
		if out.Baseline.TimeToContextSeconds > 0 {
			p := float64(out.Baseline.TimeToContextSeconds-v) * 100 / float64(out.Baseline.TimeToContextSeconds)
			out.ContextImprovementPercent = &p
		}
	}
	if dn > 0 {
		v := ds / dn
		out.AverageTimeToDecisionSeconds = &v
		if out.Baseline.TimeToDecisionSeconds > 0 {
			p := float64(out.Baseline.TimeToDecisionSeconds-v) * 100 / float64(out.Baseline.TimeToDecisionSeconds)
			out.DecisionImprovementPercent = &p
		}
	}
	return out
}
