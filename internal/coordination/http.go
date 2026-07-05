package coordination

import (
	"encoding/json"
	"errors"
	"net/http"
)

func Register(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.ListIncidents(r.PathValue("workspaceID"))})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Severity    string   `json:"severity"`
			Scope       []string `json:"scope"`
		}
		if !decode(w, r, &input) {
			return
		}
		incident, err := store.CreateIncident(r.PathValue("workspaceID"), actor(r), input.Title, input.Description, input.Severity, input.Scope)
		respond(w, http.StatusCreated, incident, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}", func(w http.ResponseWriter, r *http.Request) {
		incident, err := store.Incident(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, incident, err)
	})
	mux.HandleFunc("PATCH /api/v1/workspaces/{workspaceID}/incidents/{incidentID}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Lifecycle string `json:"lifecycle"`
			Severity  string `json:"severity"`
			OwnerID   string `json:"ownerId"`
		}
		if !decode(w, r, &input) {
			return
		}
		incident, err := store.UpdateIncident(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.Lifecycle, input.Severity, input.OwnerID)
		respond(w, http.StatusOK, incident, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/posts", func(w http.ResponseWriter, r *http.Request) {
		posts, err := store.Feed(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": posts}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/posts", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ReplyToPostID  string  `json:"replyToPostId"`
			ReplyToBlockID string  `json:"replyToBlockId"`
			Blocks         []Block `json:"blocks"`
		}
		if !decode(w, r, &input) {
			return
		}
		post, err := store.AddPost(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.ReplyToPostID, input.ReplyToBlockID, input.Blocks)
		respond(w, http.StatusCreated, post, err)
	})
	mux.HandleFunc("PUT /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/posts/{postID}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Blocks []Block `json:"blocks"`
		}
		if !decode(w, r, &input) {
			return
		}
		post, err := store.revisePost(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("postID"), actor(r), input.Blocks)
		respond(w, http.StatusOK, post, err)
	})
}

func actor(r *http.Request) string { return r.Header.Get("X-Principal-ID") }

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_request", "message": err.Error()})
		return false
	}
	return true
}

func respond(w http.ResponseWriter, success int, value any, err error) {
	if err == nil {
		writeJSON(w, success, value)
		return
	}
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(err, ErrInvalid):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, ErrForbidden):
		status, code = http.StatusForbidden, "forbidden"
	case errors.Is(err, ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, ErrConflict):
		status, code = http.StatusConflict, "invalid_transition"
	}
	writeJSON(w, status, map[string]string{"code": code, "message": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
