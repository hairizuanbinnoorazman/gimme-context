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
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get status = %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/other/incidents/"+incident.ID, nil)
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("cross-workspace status = %d", recorder.Code)
	}
}

func TestIncidentHTTPSerializesOmittedScopeAsArray(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents", bytes.NewBufferString(`{"title":"No scope","severity":"unclassified"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body["scope"]) != "[]" {
		t.Fatalf("scope = %s, want []", body["scope"])
	}
}

func TestEmptyCoordinationHTTPSerializesCollectionsAsArrays(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents", bytes.NewBufferString(`{"title":"Empty coordination","severity":"unclassified"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	var incident Incident
	if err := json.NewDecoder(recorder.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/acme/incidents/"+incident.ID+"/coordination", nil)
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	var body map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"facts", "decisions", "actions", "polls", "approvals"} {
		if string(body[field]) != "[]" {
			t.Errorf("%s = %s, want []", field, body[field])
		}
	}
}

func TestIncidentReadsRequireCurrentMembership(t *testing.T) {
	store := NewStore()
	incident, err := store.CreateIncident("acme", "alice", "Private incident", "", "SEV-2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.AddMembership("acme", incident.ID, "alice", "bob", "viewer"); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, store)

	doGet := func(path, principal string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if principal != "" {
			req.Header.Set("X-Principal-ID", principal)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	base := "/api/v1/workspaces/acme/incidents/" + incident.ID
	if got := doGet(base+"/posts", "mallory").Code; got != http.StatusForbidden {
		t.Fatalf("non-member read = %d", got)
	}
	if got := doGet(base+"/posts", "").Code; got != http.StatusForbidden {
		t.Fatalf("anonymous read = %d", got)
	}
	if got := doGet(base+"/posts", "bob").Code; got != http.StatusOK {
		t.Fatalf("member read = %d", got)
	}
	if _, err = store.UpdateMembership("acme", incident.ID, "alice", "bob", "viewer", true); err != nil {
		t.Fatal(err)
	}
	if got := doGet(base+"/posts", "bob").Code; got != http.StatusForbidden {
		t.Fatalf("revoked read = %d", got)
	}

	var list struct {
		Items []Incident `json:"items"`
	}
	if err = json.Unmarshal(doGet("/api/v1/workspaces/acme/incidents", "mallory").Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("non-member list leaked %d incidents", len(list.Items))
	}
}

func TestManualIncidentCanRunAndCloseOverHTTP(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	do := func(method, path, body string, want int) []byte {
		t.Helper()
		request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		request.Header.Set("X-Principal-ID", "alice")
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		if recorder.Code != want {
			t.Fatalf("%s %s status = %d, body = %s", method, path, recorder.Code, recorder.Body.String())
		}
		return recorder.Body.Bytes()
	}

	created := do(http.MethodPost, "/api/v1/workspaces/acme/incidents", `{"title":"Checkout errors","severity":"SEV-2","scope":["checkout"]}`, http.StatusCreated)
	var incident Incident
	if err := json.Unmarshal(created, &incident); err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/workspaces/acme/incidents/" + incident.ID
	do(http.MethodPost, base+"/posts", `{"blocks":[{"type":"status","schemaVersion":1,"payload":{"text":"Incident coordination started"}}]}`, http.StatusCreated)
	do(http.MethodPost, base+"/facts", `{"statement":"Checkout failures are elevated","evidenceBlockIds":[]}`, http.StatusCreated)
	do(http.MethodPost, base+"/decisions", `{"statement":"Roll back checkout","rationale":"Failures followed deployment","evidenceBlockIds":[]}`, http.StatusCreated)
	do(http.MethodPost, base+"/actions", `{"title":"Perform rollback","ownerId":"alice","kind":"manual.rollback","parameters":{},"verificationCriteria":"Error rate recovers"}`, http.StatusCreated)
	do(http.MethodPost, base+"/polls", `{"question":"Proceed with rollback?","mode":"advisory","options":["Yes","No"],"eligibleVoterIds":["alice"],"quorum":1,"allowVoteChanges":true}`, http.StatusCreated)
	for _, state := range []string{"investigating", "mitigating", "monitoring"} {
		do(http.MethodPatch, base, `{"lifecycle":"`+state+`"}`, http.StatusOK)
	}
	do(http.MethodPatch, base+"/resolution", `{"verifiedSummary":"Rollback restored checkout."}`, http.StatusOK)
	for _, item := range incident.ClosureChecklist {
		do(http.MethodPatch, base+"/resolution", `{"checklistItemId":"`+item.ID+`","completed":true}`, http.StatusOK)
	}
	resolved := do(http.MethodPatch, base, `{"lifecycle":"resolved"}`, http.StatusOK)
	if err := json.Unmarshal(resolved, &incident); err != nil || incident.Lifecycle != "resolved" {
		t.Fatalf("resolved incident = %+v, err = %v", incident, err)
	}
}

func TestPermanentChannelHTTPFlow(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, NewStore())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/permanent-channels", bytes.NewBufferString(`{"title":"Platform"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var channel PermanentChannel
	if err := json.NewDecoder(recorder.Body).Decode(&channel); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/permanent-channels/"+channel.ID+"/posts", bytes.NewBufferString(`{"blocks":[{"type":"markdown","payload":{"text":"Runbook discussion"}}]}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("post status = %d, body = %s", recorder.Code, recorder.Body.String())
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
	addMemberHTTP(t, mux, incident.ID, "bob", "participant")

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
	addMemberHTTP(t, mux, incident.ID, "bob", "participant")
	addMemberHTTP(t, mux, incident.ID, "carol", "participant")

	actionURL := "/api/v1/workspaces/acme/incidents/" + incident.ID + "/actions"
	actionBody := `{"title":"Roll back","ownerId":"bob","kind":"deploy.rollback","parameters":{"version":"v2"}}`
	request = httptest.NewRequest(http.MethodPost, actionURL, bytes.NewBufferString(actionBody))
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

func addMemberHTTP(t *testing.T, mux *http.ServeMux, incidentID, principalID, role string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/incidents/"+incidentID+"/members", bytes.NewBufferString(`{"principalId":"`+principalID+`","role":"`+role+`"}`))
	request.Header.Set("X-Principal-ID", "alice")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("add member %s status = %d, body = %s", principalID, recorder.Code, recorder.Body.String())
	}
}
