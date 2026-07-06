package coordination

import (
	"net/http"
	"time"
)

func registerAgentHTTP(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/agents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.Agents(r.PathValue("workspaceID"))})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/agents", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name         string   `json:"name"`
			Purpose      string   `json:"purpose"`
			Provider     string   `json:"provider"`
			Model        string   `json:"model"`
			Capabilities []string `json:"capabilities"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := store.CreateAgent(r.PathValue("workspaceID"), actor(r), in.Name, in.Purpose, in.Provider, in.Model, in.Capabilities)
		respond(w, http.StatusCreated, v, e)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/agent-activations", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			AgentID string `json:"agentId"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := store.ActivateAgent(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), in.AgentID)
		respond(w, http.StatusCreated, v, e)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/capability-grants", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			AgentID    string    `json:"agentId"`
			Capability string    `json:"capability"`
			ExpiresAt  time.Time `json:"expiresAt"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := store.GrantCapability(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), in.AgentID, in.Capability, in.ExpiresAt)
		respond(w, http.StatusCreated, v, e)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/agent-runs", func(w http.ResponseWriter, r *http.Request) {
		if !requireIncidentRead(w, r, store) {
			return
		}
		v, e := store.AgentRuns(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": v}, e)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/agent-runs", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			AgentID              string   `json:"agentId"`
			Task                 string   `json:"task"`
			Classification       string   `json:"classification"`
			EvidenceBlockIDs     []string `json:"evidenceBlockIds"`
			RequiredCapabilities []string `json:"requiredCapabilities"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := store.RunAgent(r.Context(), r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), in.AgentID, in.Task, in.Classification, in.EvidenceBlockIDs, in.RequiredCapabilities)
		respond(w, http.StatusCreated, v, e)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/ai-proposals", func(w http.ResponseWriter, r *http.Request) {
		if !requireIncidentRead(w, r, store) {
			return
		}
		v, e := store.AIProposals(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": v}, e)
	})
	mux.HandleFunc("PATCH /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/ai-proposals/{proposalID}", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Status string `json:"status"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := store.ReviewAIProposal(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("proposalID"), actor(r), in.Status)
		respond(w, http.StatusOK, v, e)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/collaboration-envelopes", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			FromRunID        string   `json:"fromRunId"`
			ToAgentID        string   `json:"toAgentId"`
			Task             string   `json:"task"`
			SourceBlockIDs   []string `json:"sourceBlockIds"`
			ResultArtifactID string   `json:"resultArtifactId"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := store.AddCollaborationEnvelope(r.PathValue("workspaceID"), r.PathValue("incidentID"), in.FromRunID, in.ToAgentID, in.Task, in.ResultArtifactID, in.SourceBlockIDs)
		respond(w, http.StatusCreated, v, e)
	})
}
