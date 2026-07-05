package coordination

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ContextRecipe is immutable once created. Updating a recipe creates a new
// version so an incident collection can always identify the configuration used.
type ContextRecipe struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Name        string         `json:"name"`
	Version     int            `json:"version"`
	Queries     []ContextQuery `json:"queries"`
	CreatedBy   string         `json:"createdBy"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type ContextQuery struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	Query    string `json:"query"`
	Lookback string `json:"lookback,omitempty"`
	Step     string `json:"step,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type ContextSnapshot struct {
	ID              string    `json:"id"`
	Source          string    `json:"source"`
	Query           string    `json:"query"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	RetrievedAt     time.Time `json:"retrievedAt"`
	Freshness       string    `json:"freshness"`
	Complete        bool      `json:"complete"`
	SourceURL       string    `json:"sourceUrl,omitempty"`
	Transformations []string  `json:"transformations"`
	Redacted        bool      `json:"redacted"`
	RetrievedBy     string    `json:"retrievedBy"`
	Data            any       `json:"data,omitempty"`
}

type RetrievalFailure struct {
	Source      string `json:"source"`
	Query       string `json:"query"`
	Category    string `json:"category"`
	Message     string `json:"message"`
	Retries     int    `json:"retries"`
	Partial     any    `json:"partialOutput,omitempty"`
	HumanAction string `json:"requiredHumanAction"`
}

type ContextCollection struct {
	ID            string             `json:"id"`
	WorkspaceID   string             `json:"workspaceId"`
	IncidentID    string             `json:"incidentId"`
	RecipeID      string             `json:"recipeId,omitempty"`
	RecipeVersion int                `json:"recipeVersion,omitempty"`
	RefreshOf     string             `json:"refreshOf,omitempty"`
	Status        string             `json:"status"`
	Snapshots     []ContextSnapshot  `json:"snapshots"`
	Failures      []RetrievalFailure `json:"failures"`
	PostID        string             `json:"postId,omitempty"`
	RequestedBy   string             `json:"requestedBy"`
	StartedAt     time.Time          `json:"startedAt"`
	CompletedAt   time.Time          `json:"completedAt"`
}

type Alert struct {
	Fingerprint  string            `json:"fingerprint"`
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

type AlertWebhook struct {
	Receiver     string            `json:"receiver"`
	Status       string            `json:"status"`
	GroupKey     string            `json:"groupKey"`
	CommonLabels map[string]string `json:"commonLabels"`
	Alerts       []Alert           `json:"alerts"`
}

type AlertResult struct {
	Incident Incident `json:"incident"`
	Created  bool     `json:"created"`
	DedupKey string   `json:"dedupKey"`
}

type TelemetryClient interface {
	Query(context.Context, string, time.Time, time.Time, ContextQuery) (data any, sourceURL string, err error)
}

type ContextService struct {
	Prometheus TelemetryClient
	Loki       TelemetryClient
	Retries    int
}

func (s *Store) CreateContextRecipe(workspaceID, actorID, recipeID, name string, queries []ContextQuery) (ContextRecipe, error) {
	name = strings.TrimSpace(name)
	if workspaceID == "" || actorID == "" || name == "" || len(queries) == 0 {
		return ContextRecipe{}, ErrInvalid
	}
	for _, q := range queries {
		if strings.TrimSpace(q.Name) == "" || strings.TrimSpace(q.Query) == "" || (q.Source != "prometheus" && q.Source != "loki") || q.Limit < 0 {
			return ContextRecipe{}, ErrInvalid
		}
		if q.Lookback != "" {
			if d, err := time.ParseDuration(q.Lookback); err != nil || d <= 0 {
				return ContextRecipe{}, ErrInvalid
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	creating := recipeID == ""
	if creating {
		recipeID = newID()
	}
	versions := s.contextRecipes[recipeID]
	if !creating && len(versions) == 0 {
		return ContextRecipe{}, ErrNotFound
	}
	if len(versions) > 0 && versions[0].WorkspaceID != workspaceID {
		return ContextRecipe{}, ErrNotFound
	}
	now := s.now().UTC()
	r := ContextRecipe{ID: recipeID, WorkspaceID: workspaceID, Name: name, Version: len(versions) + 1, Queries: append([]ContextQuery(nil), queries...), CreatedBy: actorID, CreatedAt: now}
	s.contextRecipes[recipeID] = append(versions, r)
	s.record(workspaceID, actorID, "context_recipe.version_created", recipeID, now)
	return r, nil
}

func (s *Store) ContextRecipes(workspaceID string) []ContextRecipe {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []ContextRecipe{}
	for _, versions := range s.contextRecipes {
		if len(versions) > 0 && versions[0].WorkspaceID == workspaceID {
			result = append(result, versions[len(versions)-1])
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (s *Store) ContextRecipe(workspaceID, id string, version int) (ContextRecipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions := s.contextRecipes[id]
	if len(versions) == 0 || versions[0].WorkspaceID != workspaceID {
		return ContextRecipe{}, ErrNotFound
	}
	if version == 0 {
		version = len(versions)
	}
	if version < 1 || version > len(versions) {
		return ContextRecipe{}, ErrNotFound
	}
	return versions[version-1], nil
}

func SimulateRecipe(recipe ContextRecipe, labels map[string]string, at time.Time) (ContextCollection, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	c := ContextCollection{ID: "simulation", WorkspaceID: recipe.WorkspaceID, RecipeID: recipe.ID, RecipeVersion: recipe.Version, Status: "simulated", StartedAt: at, CompletedAt: at}
	for _, q := range recipe.Queries {
		rendered, err := renderQuery(q.Query, labels)
		if err != nil {
			return ContextCollection{}, err
		}
		lookback := time.Hour
		if q.Lookback != "" {
			lookback, _ = time.ParseDuration(q.Lookback)
		}
		c.Snapshots = append(c.Snapshots, ContextSnapshot{Source: q.Source, Query: rendered, Start: at.Add(-lookback), End: at})
	}
	return c, nil
}

func renderQuery(query string, labels map[string]string) (string, error) {
	for {
		start := strings.Index(query, "{{.")
		if start < 0 {
			return query, nil
		}
		end := strings.Index(query[start:], "}}")
		if end < 0 {
			return "", ErrInvalid
		}
		end += start
		key := query[start+3 : end]
		value, ok := labels[key]
		if !ok {
			return "", fmt.Errorf("%w: missing recipe variable %s", ErrInvalid, key)
		}
		query = query[:start] + value + query[end+2:]
	}
}

func (s *Store) IncidentForAlert(workspaceID, ownerID string, payload AlertWebhook) (AlertResult, error) {
	if workspaceID == "" || ownerID == "" || len(payload.Alerts) == 0 {
		return AlertResult{}, ErrInvalid
	}
	alert := payload.Alerts[0]
	key := strings.TrimSpace(alert.Fingerprint)
	if key == "" {
		key = strings.TrimSpace(payload.GroupKey)
	}
	if key == "" {
		return AlertResult{}, ErrInvalid
	}
	key = workspaceID + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	if id := s.alertIncidents[key]; id != "" {
		if incident, ok := s.incidents[id]; ok {
			return AlertResult{Incident: cloneIncident(incident), DedupKey: key}, nil
		}
	}
	now := s.now().UTC()
	title := alert.Annotations["summary"]
	if strings.TrimSpace(title) == "" {
		title = alert.Labels["alertname"]
	}
	if strings.TrimSpace(title) == "" {
		title = "Alertmanager incident"
	}
	severity := normalizeAlertSeverity(alert.Labels["severity"])
	scope := []string{}
	for _, k := range []string{"service", "namespace", "cluster"} {
		if v := alert.Labels[k]; v != "" {
			scope = append(scope, v)
		}
	}
	incident := Incident{ID: newID(), WorkspaceID: workspaceID, Title: title, Description: alert.Annotations["description"], OwnerID: ownerID, Severity: severity, Lifecycle: "open", Scope: scope, ClosureChecklist: defaultClosureChecklist(), CreatedAt: now, UpdatedAt: now}
	s.incidents[incident.ID] = incident
	s.memberships[incident.ID] = []Membership{{WorkspaceID: workspaceID, IncidentID: incident.ID, PrincipalID: ownerID, Role: "owner", Source: "alert_rule", Status: "active", AddedBy: "alertmanager", CreatedAt: now, UpdatedAt: now}}
	s.alertIncidents[key] = incident.ID
	s.record(workspaceID, "alertmanager", "incident.created_from_alert", incident.ID, now)
	return AlertResult{Incident: cloneIncident(incident), Created: true, DedupKey: key}, nil
}

func normalizeAlertSeverity(v string) string {
	switch strings.ToLower(v) {
	case "critical", "sev-1", "sev1":
		return "SEV-1"
	case "warning", "sev-2", "sev2":
		return "SEV-2"
	case "sev-3", "sev3":
		return "SEV-3"
	case "info", "sev-4", "sev4":
		return "SEV-4"
	}
	return "unclassified"
}

func (svc ContextService) Collect(ctx context.Context, store *Store, workspaceID, incidentID, actorID string, recipe ContextRecipe, labels map[string]string, refreshOf string) (ContextCollection, error) {
	if _, err := store.Incident(workspaceID, incidentID); err != nil {
		return ContextCollection{}, err
	}
	if actorID == "" {
		return ContextCollection{}, ErrForbidden
	}
	store.mu.RLock()
	canCollect := store.canParticipate(incidentID, actorID)
	store.mu.RUnlock()
	if !canCollect {
		return ContextCollection{}, ErrForbidden
	}
	now := store.now().UTC()
	c := ContextCollection{ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID, RecipeID: recipe.ID, RecipeVersion: recipe.Version, RefreshOf: refreshOf, Status: "running", RequestedBy: actorID, StartedAt: now}
	for _, q := range recipe.Queries {
		rendered, err := renderQuery(q.Query, labels)
		if err != nil {
			c.Failures = append(c.Failures, failure(q, "invalid_query", err, 0))
			continue
		}
		lookback := time.Hour
		if q.Lookback != "" {
			lookback, _ = time.ParseDuration(q.Lookback)
		}
		start, end := now.Add(-lookback), now
		client := svc.Prometheus
		if q.Source == "loki" {
			client = svc.Loki
		}
		if client == nil {
			c.Failures = append(c.Failures, failure(q, "not_configured", errorsNew(q.Source+" integration is not configured"), 0))
			continue
		}
		var data any
		var url string
		attempts := svc.Retries + 1
		for attempt := 0; attempt < attempts; attempt++ {
			data, url, err = client.Query(ctx, rendered, start, end, q)
			if err == nil {
				break
			}
		}
		if err != nil {
			c.Failures = append(c.Failures, failure(q, classifyRetrievalError(err), err, svc.Retries))
			continue
		}
		c.Snapshots = append(c.Snapshots, ContextSnapshot{ID: newID(), Source: q.Source, Query: rendered, Start: start, End: end, RetrievedAt: store.now().UTC(), Freshness: "current", Complete: true, SourceURL: url, Transformations: []string{}, RetrievedBy: actorID, Data: data})
	}
	c.CompletedAt = store.now().UTC()
	c.Status = "complete"
	if len(c.Failures) > 0 {
		c.Status = "partial"
	}
	if len(c.Snapshots) == 0 {
		c.Status = "failed"
	}
	return store.saveCollection(c)
}

func errorsNew(message string) error { return fmt.Errorf("%s", message) }
func classifyRetrievalError(err error) string {
	if err == context.DeadlineExceeded || err == context.Canceled {
		return "timeout"
	}
	return "upstream_error"
}
func failure(q ContextQuery, category string, err error, retries int) RetrievalFailure {
	return RetrievalFailure{Source: q.Source, Query: q.Query, Category: category, Message: err.Error(), Retries: retries, HumanAction: "Check integration access and query, then refresh the collection."}
}

func (s *Store) saveCollection(c ContextCollection) (ContextCollection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[c.IncidentID]
	if !ok || incident.WorkspaceID != c.WorkspaceID {
		return ContextCollection{}, ErrNotFound
	}
	blocks := []Block{{ID: newID(), Type: "retrieval", SchemaVersion: 1, Payload: map[string]any{"collectionId": c.ID, "status": c.Status, "snapshots": c.Snapshots, "failures": c.Failures}}}
	post := Post{ID: newID(), WorkspaceID: c.WorkspaceID, IncidentID: c.IncidentID, AuthorID: "context-collector", Revision: 1, Blocks: blocks, CreatedAt: c.CompletedAt, UpdatedAt: c.CompletedAt}
	c.PostID = post.ID
	s.posts[c.IncidentID] = append(s.posts[c.IncidentID], post)
	s.postHistory[post.ID] = []Post{clonePost(post)}
	s.collections[c.IncidentID] = append(s.collections[c.IncidentID], c)
	s.record(c.WorkspaceID, c.RequestedBy, "context_collection.completed", c.ID, c.CompletedAt)
	return c, nil
}

func (s *Store) ContextCollections(workspaceID, incidentID string) ([]ContextCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return nil, ErrNotFound
	}
	return append([]ContextCollection(nil), s.collections[incidentID]...), nil
}
func (s *Store) ContextCollection(workspaceID, incidentID, id string) (ContextCollection, error) {
	items, err := s.ContextCollections(workspaceID, incidentID)
	if err != nil {
		return ContextCollection{}, err
	}
	for _, c := range items {
		if c.ID == id {
			return c, nil
		}
	}
	return ContextCollection{}, ErrNotFound
}

type IncidentSearchResult struct {
	Incident Incident `json:"incident"`
	Score    int      `json:"score"`
	Reasons  []string `json:"reasons"`
}

func (s *Store) SearchIncidents(workspaceID, principalID, query, excludeID string) []IncidentSearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	terms := strings.Fields(strings.ToLower(query))
	results := []IncidentSearchResult{}
	for id, in := range s.incidents {
		if in.WorkspaceID != workspaceID || id == excludeID {
			continue
		}
		if _, ok := s.activeRole(id, principalID); !ok {
			continue
		}
		hay := strings.ToLower(in.Title + " " + in.Description + " " + in.VerifiedSummary + " " + strings.Join(in.Scope, " "))
		score := 0
		reasons := []string{}
		for _, term := range terms {
			if strings.Contains(hay, term) {
				score++
				reasons = append(reasons, "matched "+term)
			}
		}
		if score > 0 {
			results = append(results, IncidentSearchResult{Incident: cloneIncident(in), Score: score, Reasons: reasons})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Incident.CreatedAt.After(results[j].Incident.CreatedAt)
		}
		return results[i].Score > results[j].Score
	})
	return results
}
