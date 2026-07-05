package coordination

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

func Register(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/permanent-channels", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.PermanentChannels(r.PathValue("workspaceID"))})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/permanent-channels", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.CreatePermanentChannel(r.PathValue("workspaceID"), actor(r), input.Title, input.Description)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}", func(w http.ResponseWriter, r *http.Request) {
		item, err := store.PermanentChannel(r.PathValue("workspaceID"), r.PathValue("channelID"))
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/posts", func(w http.ResponseWriter, r *http.Request) {
		items, err := store.PermanentFeed(r.PathValue("workspaceID"), r.PathValue("channelID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/posts", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ReplyToPostID  string  `json:"replyToPostId"`
			ReplyToBlockID string  `json:"replyToBlockId"`
			Blocks         []Block `json:"blocks"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.AddPermanentPost(r.PathValue("workspaceID"), r.PathValue("channelID"), actor(r), input.ReplyToPostID, input.ReplyToBlockID, input.Blocks)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("PUT /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/posts/{postID}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Blocks []Block `json:"blocks"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.RevisePermanentPost(r.PathValue("workspaceID"), r.PathValue("channelID"), r.PathValue("postID"), actor(r), input.Blocks)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/permanent-channels/{channelID}/posts/{postID}/revisions", postHistoryHandler(store))
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incident-templates", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.Templates(r.PathValue("workspaceID"))})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incident-templates", templateVersionHandler(store, false))
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incident-templates/{templateID}/versions", templateVersionHandler(store, true))
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incident-templates/{templateID}/versions/{version}", func(w http.ResponseWriter, r *http.Request) {
		version, err := strconv.Atoi(r.PathValue("version"))
		if err != nil {
			respond(w, http.StatusOK, nil, ErrInvalid)
			return
		}
		item, err := store.Template(r.PathValue("workspaceID"), r.PathValue("templateID"), version)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": store.ListIncidents(r.PathValue("workspaceID"))})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title           string   `json:"title"`
			Description     string   `json:"description"`
			Severity        string   `json:"severity"`
			Scope           []string `json:"scope"`
			TemplateID      string   `json:"templateId"`
			TemplateVersion int      `json:"templateVersion"`
		}
		if !decode(w, r, &input) {
			return
		}
		var incident Incident
		var err error
		if input.TemplateID != "" {
			incident, err = store.CreateIncidentFromTemplate(r.PathValue("workspaceID"), actor(r), input.TemplateID, input.TemplateVersion, input.Title, input.Description, input.Severity, input.Scope)
		} else {
			incident, err = store.CreateIncident(r.PathValue("workspaceID"), actor(r), input.Title, input.Description, input.Severity, input.Scope)
		}
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
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/members", func(w http.ResponseWriter, r *http.Request) {
		items, err := store.Memberships(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/members", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			PrincipalID string `json:"principalId"`
			Role        string `json:"role"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.AddMembership(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.PrincipalID, input.Role)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("PATCH /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/members/{principalID}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Role   string `json:"role"`
			Revoke bool   `json:"revoke"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.UpdateMembership(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), r.PathValue("principalID"), input.Role, input.Revoke)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/ownership-transfers", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			NewOwnerID string `json:"newOwnerId"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.TransferOwnership(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.NewOwnerID)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("PATCH /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/resolution", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			VerifiedSummary string `json:"verifiedSummary"`
			ChecklistItemID string `json:"checklistItemId"`
			Completed       *bool  `json:"completed"`
		}
		if !decode(w, r, &input) {
			return
		}
		incident, err := store.UpdateResolution(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.VerifiedSummary, input.ChecklistItemID, input.Completed)
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
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/posts/{postID}/revisions", postHistoryHandler(store))
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/facts", func(w http.ResponseWriter, r *http.Request) {
		items, err := store.Facts(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/coordination", func(w http.ResponseWriter, r *http.Request) {
		workspaceID, incidentID := r.PathValue("workspaceID"), r.PathValue("incidentID")
		facts, err := store.Facts(workspaceID, incidentID)
		if err != nil {
			respond(w, http.StatusOK, nil, err)
			return
		}
		decisions, _ := store.Decisions(workspaceID, incidentID)
		actions, _ := store.Actions(workspaceID, incidentID)
		polls, _ := store.Polls(workspaceID, incidentID)
		approvals, _ := store.Approvals(workspaceID, incidentID)
		writeJSON(w, http.StatusOK, map[string]any{"facts": facts, "decisions": decisions, "actions": actions, "polls": polls, "approvals": approvals})
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/facts", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Statement        string   `json:"statement"`
			EvidenceBlockIDs []string `json:"evidenceBlockIds"`
		}
		if !decode(w, r, &input) {
			return
		}
		fact, err := store.AddFact(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.Statement, input.EvidenceBlockIDs)
		respond(w, http.StatusCreated, fact, err)
	})
	mux.HandleFunc("PATCH /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/facts/{factID}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			State string `json:"state"`
		}
		if !decode(w, r, &input) {
			return
		}
		fact, err := store.UpdateFact(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("factID"), actor(r), input.State)
		respond(w, http.StatusOK, fact, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/decisions", func(w http.ResponseWriter, r *http.Request) {
		items, err := store.Decisions(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/decisions", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Statement        string   `json:"statement"`
			Rationale        string   `json:"rationale"`
			EvidenceBlockIDs []string `json:"evidenceBlockIds"`
		}
		if !decode(w, r, &input) {
			return
		}
		decision, err := store.AddDecision(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.Statement, input.Rationale, input.EvidenceBlockIDs)
		respond(w, http.StatusCreated, decision, err)
	})
	mux.HandleFunc("PATCH /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/decisions/{decisionID}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Status string `json:"status"`
		}
		if !decode(w, r, &input) {
			return
		}
		decision, err := store.Decide(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("decisionID"), actor(r), input.Status)
		respond(w, http.StatusOK, decision, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/actions", func(w http.ResponseWriter, r *http.Request) {
		items, err := store.Actions(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/actions", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title                string         `json:"title"`
			OwnerID              string         `json:"ownerId"`
			Kind                 string         `json:"kind"`
			Parameters           map[string]any `json:"parameters"`
			VerificationCriteria string         `json:"verificationCriteria"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.AddAction(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.Title, input.OwnerID, input.Kind, input.Parameters, input.VerificationCriteria)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("PATCH /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/actions/{actionID}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Status string `json:"status"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.UpdateAction(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("actionID"), actor(r), input.Status)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/polls", func(w http.ResponseWriter, r *http.Request) {
		items, err := store.Polls(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/polls", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Question         string   `json:"question"`
			Mode             string   `json:"mode"`
			Options          []string `json:"options"`
			EligibleVoterIDs []string `json:"eligibleVoterIds"`
			Quorum           int      `json:"quorum"`
			AllowVoteChanges bool     `json:"allowVoteChanges"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.AddPoll(r.PathValue("workspaceID"), r.PathValue("incidentID"), actor(r), input.Question, input.Mode, input.Options, input.EligibleVoterIDs, input.Quorum, input.AllowVoteChanges)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/polls/{pollID}/votes", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OptionID string `json:"optionId"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.Vote(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("pollID"), actor(r), input.OptionID)
		respond(w, http.StatusOK, item, err)
	})
	mux.HandleFunc("GET /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/approvals", func(w http.ResponseWriter, r *http.Request) {
		items, err := store.Approvals(r.PathValue("workspaceID"), r.PathValue("incidentID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/approvals", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ActionID            string   `json:"actionId"`
			EligibleApproverIDs []string `json:"eligibleApproverIds"`
			Quorum              int      `json:"quorum"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.RequestApproval(r.PathValue("workspaceID"), r.PathValue("incidentID"), input.ActionID, actor(r), input.EligibleApproverIDs, input.Quorum)
		respond(w, http.StatusCreated, item, err)
	})
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/incidents/{incidentID}/approvals/{approvalID}/responses", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Decision string `json:"decision"`
		}
		if !decode(w, r, &input) {
			return
		}
		item, err := store.RespondApproval(r.PathValue("workspaceID"), r.PathValue("incidentID"), r.PathValue("approvalID"), actor(r), input.Decision)
		respond(w, http.StatusOK, item, err)
	})
	registerContextHTTP(mux, store)
	registerAgentHTTP(mux, store)
	registerWorkflowHTTP(mux, store)
}

func postHistoryHandler(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := r.PathValue("incidentID")
		if channelID == "" {
			channelID = r.PathValue("channelID")
		}
		items, err := store.PostHistory(r.PathValue("workspaceID"), channelID, r.PathValue("postID"))
		respond(w, http.StatusOK, map[string]any{"items": items}, err)
	}
}

func templateVersionHandler(store *Store, existing bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name             string          `json:"name"`
			Description      string          `json:"description"`
			DefaultSeverity  string          `json:"defaultSeverity"`
			DefaultScope     []string        `json:"defaultScope"`
			ClosureChecklist []ChecklistItem `json:"closureChecklist"`
		}
		if !decode(w, r, &input) {
			return
		}
		templateID := ""
		if existing {
			templateID = r.PathValue("templateID")
		}
		item, err := store.CreateTemplateVersion(r.PathValue("workspaceID"), actor(r), templateID, input.Name, input.Description, input.DefaultSeverity, input.DefaultScope, input.ClosureChecklist)
		respond(w, http.StatusCreated, item, err)
	}
}

func actor(r *http.Request) string {
	return r.Header.Get("X-Principal-ID")
}

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
