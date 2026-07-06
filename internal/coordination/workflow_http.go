package coordination

import "net/http"

func registerWorkflowHTTP(mux *http.ServeMux, store *Store) {
	base := "/api/v1/workspaces/{workspaceID}/workflow-definitions"
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.WorkflowDefinitions(r.PathValue("workspaceID"))})
	})
	mux.HandleFunc("POST "+base, workflowVersionHandler(store, false))
	mux.HandleFunc("POST "+base+"/{definitionID}/versions", workflowVersionHandler(store, true))
	mux.HandleFunc("POST "+base+"/{definitionID}/simulations", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Version   int            `json:"version"`
			Variables map[string]any `json:"variables"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.SimulateWorkflow(r.PathValue("workspaceID"), r.PathValue("definitionID"), input.Version, input.Variables)
		respond(w, http.StatusOK, item, err)
	})
	incidentBase := "/api/v1/workspaces/{workspaceID}/incidents/{incidentID}/workflow-runs"
	mux.HandleFunc("GET "+incidentBase, func(w http.ResponseWriter, r *http.Request) {
		if !requireIncidentRead(w, r, store) {
			return
		}
		items, err := store.WorkflowRuns(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST "+incidentBase, func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			DefinitionID      string         `json:"definitionId"`
			DefinitionVersion int            `json:"definitionVersion"`
			Variables         map[string]any `json:"variables"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.StartWorkflow(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.DefinitionID, input.DefinitionVersion, input.Variables)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("GET "+incidentBase+"/{runID}", func(w http.ResponseWriter, r *http.Request) {
		if !requireIncidentRead(w, r, store) {
			return
		}
		item, err := store.WorkflowProjection(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("runID"))
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("POST "+incidentBase+"/{runID}/commands", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Command       string `json:"command"`
			StepID        string `json:"stepId"`
			Justification string `json:"justification"`
			Output        string `json:"output"`
			TargetVersion int    `json:"targetVersion"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.CommandWorkflow(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("runID"), actor(r), input.Command, input.StepID, input.Justification, input.Output, input.TargetVersion)
		respond(w, http.StatusOK, item, err)
	})
}

func workflowVersionHandler(store *Store, existing bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name  string         `json:"name"`
			Steps []WorkflowStep `json:"steps"`
		}
		if !decode(w, r, &input) {
			return
		}
		id := ""
		if existing {
			id = r.PathValue("definitionID")
		}
		item, err := store.CreateWorkflowVersion(r.PathValue("workspaceID"), actor(r), id, input.Name, input.Steps)
		respond(w, http.StatusCreated, item, err)
	}
}
