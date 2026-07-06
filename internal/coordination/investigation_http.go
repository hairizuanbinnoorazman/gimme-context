package coordination

import (
	"net/http"
	"time"
)

func registerInvestigationHTTP(mux *http.ServeMux, store *Store) {
	repos := "/api/v1/workspaces/{workspaceID}/repository-config"
	mux.HandleFunc("PUT "+repos, func(w http.ResponseWriter, r *http.Request) {
		var input RepositoryConfig
		if !decode(w, r, &input) {
			return
		}
		item, err := store.ConfigureRepository(r.PathValue("workspaceID"), actor(r), input)
		respond(w, http.StatusOK, item, err)
	})
	base := "/api/v1/workspaces/{workspaceID}/incidents/{incidentID}/investigations"
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		items, err := store.Investigations(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST "+base, func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Ref        string `json:"ref"`
			TTLSeconds int    `json:"ttlSeconds"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.StartInvestigation(r.Context(), r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.Ref, time.Duration(input.TTLSeconds)*time.Second)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("POST "+base+"/{investigationID}/executions", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Kind    string   `json:"kind"`
			Summary string   `json:"summary"`
			Command []string `json:"command"`
			URL     string   `json:"url"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.ExecuteInvestigation(r.Context(), r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("investigationID"), actor(r), input.Kind, input.Summary, input.Command, input.URL)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("POST "+base+"/{investigationID}/patch", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Branch string `json:"branch"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.PreparePatch(r.Context(), r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("investigationID"), actor(r), input.Branch)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("POST "+base+"/{investigationID}/pull-request", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.CreateInvestigationPR(r.Context(), r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("investigationID"), actor(r), input.Title, input.Body)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("POST "+base+"/{investigationID}/destroy", func(w http.ResponseWriter, r *http.Request) {
		item, err := store.DestroyInvestigation(r.Context(), r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("investigationID"), actor(r))
		respond(w, http.StatusOK, item, err)
	})
}
