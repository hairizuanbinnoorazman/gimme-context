package coordination

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIncidentHTTPFlow(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	body := []byte(`{"title":"Database latency","severity":"SEV-2","scope":["payments"]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents", bytes.NewReader(body))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var incident Incident
	if err := json.NewDecoder(recorder.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/acme/incidents/"+incident.ID, nil)
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get status = %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/other/incidents/"+incident.ID, nil)
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("cross-workspace status = %d", recorder.Code)
	}
}

func TestDecisionHTTPFlow(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents", bytes.NewBufferString(`{"title":"Latency","severity":"SEV-2"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	var incident Incident
	if err := json.NewDecoder(recorder.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents/"+incident.ID+"/decisions", bytes.NewBufferString(`{"statement":"roll back","rationale":"latency followed deployment"}`))
	request.Header.Set("X-Principal-ID", "bob")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("propose status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var decision Decision
	if err := json.NewDecoder(recorder.Body).Decode(&decision); err != nil {
		t.Fatal(err)
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/acme/incidents/"+incident.ID+"/decisions/"+decision.ID, bytes.NewBufferString(`{"status":"accepted"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("accept status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestMembershipOwnershipHTTPFlow(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents", bytes.NewBufferString(`{"title":"Latency","severity":"SEV-2"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	var incident Incident
	_ = json.NewDecoder(recorder.Body).Decode(&incident)

	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents/"+incident.ID+"/members", bytes.NewBufferString(`{"principalId":"bob","role":"editor"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("add member status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents/"+incident.ID+"/ownership-transfers", bytes.NewBufferString(`{"newOwnerId":"bob"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("transfer status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	_ = json.NewDecoder(recorder.Body).Decode(&incident)
	if incident.OwnerID != "bob" {
		t.Fatalf("owner = %q, want bob", incident.OwnerID)
	}
}

func TestActionApprovalHTTPFlow(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents", bytes.NewBufferString(`{"title":"Latency","severity":"SEV-2"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	var incident Incident
	if err := json.NewDecoder(recorder.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents/"+incident.ID+"/actions", bytes.NewBufferString(`{"title":"Roll back","ownerId":"bob","kind":"deploy.rollback","parameters":{"version":"v2"}}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("action status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var action Action
	if err := json.NewDecoder(recorder.Body).Decode(&action); err != nil {
		t.Fatal(err)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents/"+incident.ID+"/approvals", bytes.NewBufferString(`{"actionId":"`+action.ID+`","eligibleApproverIds":["carol"],"quorum":1}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("approval status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var approval Approval
	if err := json.NewDecoder(recorder.Body).Decode(&approval); err != nil {
		t.Fatal(err)
	}
	if approval.SpecificationHash != action.SpecificationHash {
		t.Fatalf("approval hash %q != action hash %q", approval.SpecificationHash, action.SpecificationHash)
	}
}
