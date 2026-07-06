package coordination

import (
	"net/http"
	"time"
)

func registerKnowledgeHTTP(mux *http.ServeMux, s *Store) {
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/artifacts", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.Artifacts(r.PathValue("workspaceID"), r.PathValue("channelID"))
		respond(w, http.StatusOK, map[string]any{"items": v}, e)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/artifacts", artifactVersionHandler(s, false))
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/artifacts/{artifactID}/versions", artifactVersionHandler(s, true))
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/artifacts/{artifactID}/promotions", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Version int `json:"version"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := s.PromoteArtifact(r.PathValue("workspaceID"), r.PathValue("channelID"), r.PathValue("artifactID"), actor(r), in.Version)
		respond(w, http.StatusOK, v, e)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/artifacts/{artifactID}/simulations", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Version int               `json:"version"`
			Inputs  map[string]string `json:"inputs"`
		}
		if !decode(w, r, &in) {
			return
		}
		v, e := s.SimulateRunbook(r.PathValue("workspaceID"), r.PathValue("channelID"), r.PathValue("artifactID"), actor(r), in.Version, in.Inputs)
		respond(w, http.StatusCreated, v, e)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/archive", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.ArchiveIncident(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r))
		respond(w, http.StatusOK, v, e)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/audit-export", func(w http.ResponseWriter, r *http.Request) {
		from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
		to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
		writeJSON(w, http.StatusOK, map[string]any{"items": s.AuditExport(r.PathValue("workspaceID"), from, to)})
	})
	mux.HandleFunc("PUT /api/v1/workspaces/{workspaceID}/pilot-baseline", func(w http.ResponseWriter, r *http.Request) {
		var in PilotBaseline
		if !decode(w, r, &in) {
			return
		}
		v, e := s.SetPilotBaseline(r.PathValue("workspaceID"), actor(r), in)
		respond(w, http.StatusOK, v, e)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/pilot-analytics", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.PilotAnalytics(r.PathValue("workspaceID")))
	})
}
func artifactVersionHandler(s *Store, existing bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Type             string   `json:"type"`
			Title            string   `json:"title"`
			Content          string   `json:"content"`
			SourceIncidentID string   `json:"sourceIncidentId"`
			AgentRunID       string   `json:"agentRunId"`
			EvidenceBlockIDs []string `json:"evidenceBlockIds"`
		}
		if !decode(w, r, &in) {
			return
		}
		id := ""
		if existing {
			id = r.PathValue("artifactID")
		}
		v, e := s.CreateArtifactVersion(r.PathValue("workspaceID"), r.PathValue("channelID"), id, actor(r), in.Type, in.Title, in.Content, in.SourceIncidentID, in.AgentRunID, in.EvidenceBlockIDs)
		respond(w, http.StatusCreated, v, e)
	}
}
