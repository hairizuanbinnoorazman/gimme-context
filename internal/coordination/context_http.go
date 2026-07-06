package coordination

import (
	"net/http"
	"strconv"
	"time"
)

func registerContextHTTP(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/context-recipes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.ContextRecipes(r.PathValue("workspaceID"))})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/context-recipes", recipeHandler(store, false))
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/context-recipes/{recipeID}/versions", recipeHandler(store, true))
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/context-recipes/{recipeID}/versions/{version}", func(w http.ResponseWriter, r *http.Request) {
		v, err := strconv.Atoi(r.PathValue("version"))
		if err != nil {
			respond(w, http.StatusOK, nil, ErrInvalid)
			return
		}
		item, err := store.ContextRecipe(r.PathValue("workspaceID"), r.PathValue("recipeID"), v)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/context-recipes/{recipeID}/simulate", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Version int               `json:"version"`
			Labels  map[string]string `json:"labels"`
			At      time.Time         `json:"at"`
		}
		if !decode(w, r, &input) {
			return
		}
		recipe, err := store.ContextRecipe(r.PathValue("workspaceID"), r.PathValue("recipeID"), input.Version)
		if err != nil {
			respond(w, http.StatusOK, nil, err)
			return
		}
		item, err := SimulateRecipe(recipe, input.Labels, input.At)
		respond(w, http.StatusOK, item, err)
	})

	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/integrations/alertmanager/webhook", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OwnerID       string `json:"ownerId"`
			RecipeID      string `json:"recipeId"`
			RecipeVersion int    `json:"recipeVersion"`
			AlertWebhook
		}
		if !decode(w, r, &input) {
			return
		}
		owner := input.OwnerID
		if owner == "" {
			owner = actor(r)
		}
		result, err := store.IncidentForAlert(r.PathValue("workspaceID"), owner, input.AlertWebhook)
		if err != nil {
			respond(w, http.StatusAccepted, nil, err)
			return
		}
		if result.Created && input.RecipeID != "" {
			recipe, e := store.ContextRecipe(r.PathValue("workspaceID"), input.RecipeID, input.RecipeVersion)
			if e == nil {
				labels := map[string]string{}
				for key, value := range input.CommonLabels {
					labels[key] = value
				}
				if len(input.Alerts) > 0 {
					for key, value := range input.Alerts[0].Labels {
						if labels[key] == "" {
							labels[key] = value
						}
					}
				}
				store.mu.RLock()
				svc := store.contextService
				store.mu.RUnlock()
				collection, e := svc.Collect(r.Context(), store, r.PathValue("workspaceID"), result.Incident.ID, owner, recipe, labels, "")
				if e == nil {
					respond(w, http.StatusCreated, map[string]any{"incident": result.Incident, "created": true, "dedupKey": result.DedupKey, "collection": collection}, nil)
					return
				}
			}
		}
		status := http.StatusOK
		if result.Created {
			status = http.StatusCreated
		}
		respond(w, status, result, nil)
	})

	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/context-collections", func(w http.ResponseWriter, r *http.Request) {
		if !requireIncidentRead(w, r, store) {
			return
		}
		items, err := store.ContextCollections(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/context-collections", collectHandler(store, false))
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/context-collections/{collectionID}/refresh", collectHandler(store, true))
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incident-search", func(w http.ResponseWriter, r *http.Request) {
		items := store.SearchIncidents(r.PathValue("workspaceID"), actor(r), r.URL.Query().Get("q"), r.URL.Query().Get("exclude"))
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/knowledge-search", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.SearchKnowledge(r.PathValue("workspaceID"), actor(r), r.URL.Query().Get("q"))})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/knowledge-search", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Query string `json:"query"`
		}
		if !decode(w, r, &input) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": store.SearchKnowledge(r.PathValue("workspaceID"), actor(r), input.Query)})
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/similar", func(w http.ResponseWriter, r *http.Request) {
		if !requireIncidentRead(w, r, store) {
			return
		}
		incident, err := store.Incident(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		if err != nil {
			respond(w, http.StatusOK, nil, err)
			return
		}
		q := incident.Title + " " + incident.Description + " " + stringJoin(incident.Scope)
		items := store.SearchIncidents(r.PathValue("workspaceID"), actor(r), q, incident.ID)
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	})
}

func recipeHandler(store *Store, existing bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name    string         `json:"name"`
			Queries []ContextQuery `json:"queries"`
		}
		if !decode(w, r, &input) {
			return
		}
		id := ""
		if existing {
			id = r.PathValue("recipeID")
		}
		item, err := store.CreateContextRecipe(r.PathValue("workspaceID"), actor(r), id, input.Name, input.Queries)
		respond(w, http.StatusCreated, item, err)
	}
}
func collectHandler(store *Store, refresh bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RecipeID      string            `json:"recipeId"`
			RecipeVersion int               `json:"recipeVersion"`
			Labels        map[string]string `json:"labels"`
		}
		if !decode(w, r, &input) {
			return
		}
		refreshOf := ""
		if refresh {
			refreshOf = r.PathValue("collectionID")
			old, err := store.ContextCollection(r.PathValue("workspaceID"), r.PathValue("incidentID"), refreshOf)
			if err != nil {
				respond(w, http.StatusCreated, nil, err)
				return
			}
			if input.RecipeID == "" {
				input.RecipeID = old.RecipeID
				input.RecipeVersion = old.RecipeVersion
			}
		}
		recipe, err := store.ContextRecipe(r.PathValue("workspaceID"), input.RecipeID, input.RecipeVersion)
		if err != nil {
			respond(w, http.StatusCreated, nil, err)
			return
		}
		store.mu.RLock()
		svc := store.contextService
		store.mu.RUnlock()
		item, err := svc.Collect(r.Context(), store, r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), recipe, input.Labels, refreshOf)
		respond(w, http.StatusCreated, item, err)
	}
}
func stringJoin(v []string) string {
	result := ""
	for _, s := range v {
		result += " " + s
	}
	return result
}
