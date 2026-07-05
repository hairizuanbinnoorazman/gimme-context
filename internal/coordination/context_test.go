package coordination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubTelemetry struct {
	data  any
	err   error
	calls int
}

func (s *stubTelemetry) Query(context.Context, string, time.Time, time.Time, ContextQuery) (any, string, error) {
	s.calls++
	return s.data, "https://telemetry.example/explore", s.err
}

func TestAlertCreatesIncidentCollectsContextAndDeduplicates(t *testing.T) {
	store := NewStore()
	prom := &stubTelemetry{data: map[string]any{"result": "metrics"}}
	loki := &stubTelemetry{data: map[string]any{"result": "logs"}}
	store.SetContextService(ContextService{Prometheus: prom, Loki: loki, Retries: 1})
	recipe, err := store.CreateContextRecipe("acme", "alice", "", "Initial context", []ContextQuery{{Name: "errors", Source: "prometheus", Query: `rate(http_errors_total{service="{{.service}}"}[5m])`, Lookback: "30m"}, {Name: "logs", Source: "loki", Query: `{service="{{.service}}"} |= "error"`, Limit: 100}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, store)
	body := map[string]any{"ownerId": "alice", "recipeId": recipe.ID, "commonLabels": map[string]string{"service": "checkout"}, "alerts": []map[string]any{{"fingerprint": "alert-1", "labels": map[string]string{"alertname": "HighErrors", "severity": "critical", "service": "checkout"}, "annotations": map[string]string{"summary": "Checkout errors"}}}}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/integrations/alertmanager/webhook", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Incident   Incident          `json:"incident"`
		Collection ContextCollection `json:"collection"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Incident.Severity != "SEV-1" || len(response.Collection.Snapshots) != 2 || response.Collection.PostID == "" {
		t.Fatalf("response=%+v", response)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/integrations/alertmanager/webhook", bytes.NewReader(raw))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("duplicate status=%d body=%s", rec.Code, rec.Body.String())
	}
	if prom.calls != 1 || loki.calls != 1 {
		t.Fatalf("duplicate recollected: prom=%d loki=%d", prom.calls, loki.calls)
	}
}

func TestCollectionPublishesFailuresAndRefreshLineage(t *testing.T) {
	store := NewStore()
	broken := &stubTelemetry{err: errors.New("unavailable")}
	store.SetContextService(ContextService{Prometheus: broken, Retries: 2})
	in, err := store.CreateIncident("acme", "alice", "Failure", "", "SEV-2", nil)
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := store.CreateContextRecipe("acme", "alice", "", "Context", []ContextQuery{{Name: "metric", Source: "prometheus", Query: "up", Required: true}})
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.contextService.Collect(context.Background(), store, "acme", in.ID, "alice", recipe, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "failed" || len(first.Failures) != 1 || first.Failures[0].Retries != 2 || broken.calls != 3 {
		t.Fatalf("first=%+v calls=%d", first, broken.calls)
	}
	second, err := store.contextService.Collect(context.Background(), store, "acme", in.ID, "alice", recipe, nil, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID || second.RefreshOf != first.ID || second.PostID == first.PostID {
		t.Fatalf("refresh=%+v", second)
	}
}

func TestSearchOnlyReturnsVisibleIncidents(t *testing.T) {
	store := NewStore()
	visible, _ := store.CreateIncident("acme", "alice", "Checkout latency", "", "SEV-2", []string{"payments"})
	hidden, _ := store.CreateIncident("acme", "bob", "Checkout errors", "", "SEV-2", []string{"payments"})
	results := store.SearchIncidents("acme", "alice", "checkout payments", " ")
	if len(results) != 1 || results[0].Incident.ID != visible.ID {
		t.Fatalf("results=%+v hidden=%s", results, hidden.ID)
	}
}
