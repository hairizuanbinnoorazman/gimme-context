package coordination

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("conflict")
	ErrInvalid   = errors.New("invalid input")
)

var lifecycleOrder = map[string]int{
	"open": 0, "investigating": 1, "mitigating": 2, "monitoring": 3,
	"resolved": 4, "reviewed": 5, "archived": 6,
}

var validSeverity = map[string]bool{
	"unclassified": true, "SEV-1": true, "SEV-2": true, "SEV-3": true, "SEV-4": true,
}

var validBlockType = map[string]bool{
	"markdown": true, "code": true, "log": true, "table": true, "checklist": true,
	"fact": true, "decision": true, "action": true, "poll": true, "approval": true, "status": true,
}

type Block struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	SchemaVersion int            `json:"schemaVersion"`
	Payload       map[string]any `json:"payload"`
}

type Post struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspaceId"`
	IncidentID     string    `json:"incidentId"`
	AuthorID       string    `json:"authorId"`
	ReplyToPostID  string    `json:"replyToPostId,omitempty"`
	ReplyToBlockID string    `json:"replyToBlockId,omitempty"`
	Revision       int       `json:"revision"`
	Blocks         []Block   `json:"blocks"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Incident struct {
	ID               string          `json:"id"`
	WorkspaceID      string          `json:"workspaceId"`
	Title            string          `json:"title"`
	Description      string          `json:"description,omitempty"`
	OwnerID          string          `json:"ownerId"`
	Severity         string          `json:"severity"`
	Lifecycle        string          `json:"lifecycle"`
	Scope            []string        `json:"scope"`
	VerifiedSummary  string          `json:"verifiedSummary,omitempty"`
	ClosureChecklist []ChecklistItem `json:"closureChecklist"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

type ChecklistItem struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Completed bool   `json:"completed"`
}

type Fact struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspaceId"`
	IncidentID       string    `json:"incidentId"`
	Statement        string    `json:"statement"`
	State            string    `json:"state"`
	EvidenceBlockIDs []string  `json:"evidenceBlockIds"`
	ProposedBy       string    `json:"proposedBy"`
	UpdatedBy        string    `json:"updatedBy"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Decision struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspaceId"`
	IncidentID       string    `json:"incidentId"`
	Statement        string    `json:"statement"`
	Rationale        string    `json:"rationale"`
	Status           string    `json:"status"`
	EvidenceBlockIDs []string  `json:"evidenceBlockIds"`
	ProposedBy       string    `json:"proposedBy"`
	DecidedBy        string    `json:"decidedBy,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type AuditEvent struct {
	ID, WorkspaceID, ActorID, Action, TargetID string
	At                                         time.Time
}

type Store struct {
	mu        sync.RWMutex
	incidents map[string]Incident
	posts     map[string][]Post
	facts     map[string][]Fact
	decisions map[string][]Decision
	audit     []AuditEvent
	now       func() time.Time
}

func NewStore() *Store {
	return &Store{incidents: make(map[string]Incident), posts: make(map[string][]Post), facts: make(map[string][]Fact), decisions: make(map[string][]Decision), now: time.Now}
}

func (s *Store) CreateIncident(workspaceID, actorID, title, description, severity string, scope []string) (Incident, error) {
	title = strings.TrimSpace(title)
	if workspaceID == "" || actorID == "" || title == "" || !validSeverity[severity] {
		return Incident{}, ErrInvalid
	}
	now := s.now().UTC()
	incident := Incident{ID: newID(), WorkspaceID: workspaceID, Title: title, Description: strings.TrimSpace(description), OwnerID: actorID, Severity: severity, Lifecycle: "open", Scope: scope, ClosureChecklist: defaultClosureChecklist(), CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.incidents[incident.ID] = incident
	s.record(workspaceID, actorID, "incident.created", incident.ID, now)
	return incident, nil
}

func (s *Store) Incident(workspaceID, id string) (Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[id]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	return incident, nil
}

func (s *Store) ListIncidents(workspaceID string) []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Incident, 0)
	for _, incident := range s.incidents {
		if incident.WorkspaceID == workspaceID {
			items = append(items, incident)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items
}

func (s *Store) UpdateIncident(workspaceID, id, actorID, lifecycle, severity, ownerID string) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[id]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if actorID != incident.OwnerID {
		return Incident{}, ErrForbidden
	}
	if lifecycle != "" {
		current, currentOK := lifecycleOrder[incident.Lifecycle]
		next, nextOK := lifecycleOrder[lifecycle]
		validBranch := lifecycle == "dormant" || lifecycle == "cancelled" || (incident.Lifecycle == "dormant" && lifecycle == "investigating")
		if !validBranch && (!nextOK || !currentOK || next != current+1) {
			return Incident{}, ErrConflict
		}
		if lifecycle == "resolved" && (strings.TrimSpace(incident.VerifiedSummary) == "" || !checklistComplete(incident.ClosureChecklist)) {
			return Incident{}, ErrConflict
		}
		incident.Lifecycle = lifecycle
	}
	if severity != "" {
		if !validSeverity[severity] {
			return Incident{}, ErrInvalid
		}
		incident.Severity = severity
	}
	if ownerID != "" {
		incident.OwnerID = ownerID
	}
	incident.UpdatedAt = s.now().UTC()
	s.incidents[id] = incident
	s.record(workspaceID, actorID, "incident.updated", id, incident.UpdatedAt)
	return incident, nil
}

func (s *Store) UpdateResolution(workspaceID, incidentID, actorID, summary, checklistItemID string, completed *bool) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if actorID != incident.OwnerID {
		return Incident{}, ErrForbidden
	}
	if summary != "" {
		incident.VerifiedSummary = strings.TrimSpace(summary)
	}
	if checklistItemID != "" {
		if completed == nil {
			return Incident{}, ErrInvalid
		}
		found := false
		for i := range incident.ClosureChecklist {
			if incident.ClosureChecklist[i].ID == checklistItemID {
				incident.ClosureChecklist[i].Completed = *completed
				found = true
				break
			}
		}
		if !found {
			return Incident{}, ErrNotFound
		}
	}
	if summary == "" && checklistItemID == "" {
		return Incident{}, ErrInvalid
	}
	incident.UpdatedAt = s.now().UTC()
	s.incidents[incidentID] = incident
	s.record(workspaceID, actorID, "incident.resolution_updated", incidentID, incident.UpdatedAt)
	return incident, nil
}

func (s *Store) AddFact(workspaceID, incidentID, actorID, statement string, evidence []string) (Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Fact{}, ErrNotFound
	}
	statement = strings.TrimSpace(statement)
	if actorID == "" || statement == "" || !s.evidenceExists(incidentID, evidence) {
		return Fact{}, ErrInvalid
	}
	now := s.now().UTC()
	fact := Fact{ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID, Statement: statement, State: "unverified", EvidenceBlockIDs: evidence, ProposedBy: actorID, UpdatedBy: actorID, CreatedAt: now, UpdatedAt: now}
	s.facts[incidentID] = append(s.facts[incidentID], fact)
	s.record(workspaceID, actorID, "fact.proposed", fact.ID, now)
	return fact, nil
}

func (s *Store) UpdateFact(workspaceID, incidentID, factID, actorID, state string) (Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Fact{}, ErrNotFound
	}
	if actorID != incident.OwnerID {
		return Fact{}, ErrForbidden
	}
	valid := map[string]bool{"unverified": true, "corroborated": true, "disputed": true, "superseded": true, "invalidated": true}
	if !valid[state] {
		return Fact{}, ErrInvalid
	}
	for i := range s.facts[incidentID] {
		if s.facts[incidentID][i].ID == factID {
			s.facts[incidentID][i].State, s.facts[incidentID][i].UpdatedBy, s.facts[incidentID][i].UpdatedAt = state, actorID, s.now().UTC()
			s.record(workspaceID, actorID, "fact.state_changed", factID, s.facts[incidentID][i].UpdatedAt)
			return s.facts[incidentID][i], nil
		}
	}
	return Fact{}, ErrNotFound
}

func (s *Store) Facts(workspaceID, incidentID string) ([]Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return nil, ErrNotFound
	}
	return append([]Fact(nil), s.facts[incidentID]...), nil
}

func (s *Store) AddDecision(workspaceID, incidentID, actorID, statement, rationale string, evidence []string) (Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Decision{}, ErrNotFound
	}
	statement = strings.TrimSpace(statement)
	if actorID == "" || statement == "" || !s.evidenceExists(incidentID, evidence) {
		return Decision{}, ErrInvalid
	}
	now := s.now().UTC()
	d := Decision{ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID, Statement: statement, Rationale: strings.TrimSpace(rationale), Status: "proposed", EvidenceBlockIDs: evidence, ProposedBy: actorID, CreatedAt: now, UpdatedAt: now}
	s.decisions[incidentID] = append(s.decisions[incidentID], d)
	s.record(workspaceID, actorID, "decision.proposed", d.ID, now)
	return d, nil
}

func (s *Store) Decide(workspaceID, incidentID, decisionID, actorID, status string) (Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Decision{}, ErrNotFound
	}
	if actorID != incident.OwnerID {
		return Decision{}, ErrForbidden
	}
	if status != "accepted" && status != "rejected" {
		return Decision{}, ErrInvalid
	}
	for i := range s.decisions[incidentID] {
		d := &s.decisions[incidentID][i]
		if d.ID != decisionID {
			continue
		}
		if d.Status != "proposed" {
			return Decision{}, ErrConflict
		}
		d.Status, d.DecidedBy, d.UpdatedAt = status, actorID, s.now().UTC()
		s.record(workspaceID, actorID, "decision."+status, decisionID, d.UpdatedAt)
		return *d, nil
	}
	return Decision{}, ErrNotFound
}

func (s *Store) Decisions(workspaceID, incidentID string) ([]Decision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return nil, ErrNotFound
	}
	return append([]Decision(nil), s.decisions[incidentID]...), nil
}

func (s *Store) incidentExists(workspaceID, incidentID string) bool {
	incident, ok := s.incidents[incidentID]
	return ok && incident.WorkspaceID == workspaceID
}

func (s *Store) evidenceExists(incidentID string, ids []string) bool {
	for _, id := range ids {
		found := false
		for _, post := range s.posts[incidentID] {
			for _, block := range post.Blocks {
				if block.ID == id {
					found = true
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func defaultClosureChecklist() []ChecklistItem {
	return []ChecklistItem{{ID: "impact-understood", Label: "Impact understood"}, {ID: "mitigation-verified", Label: "Mitigation verified"}, {ID: "follow-ups-assigned", Label: "Follow-ups assigned"}}
}

func checklistComplete(items []ChecklistItem) bool {
	for _, item := range items {
		if !item.Completed {
			return false
		}
	}
	return len(items) > 0
}

func (s *Store) AddPost(workspaceID, incidentID, actorID, replyPostID, replyBlockID string, blocks []Block) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Post{}, ErrNotFound
	}
	if actorID == "" || len(blocks) == 0 || !validateBlocks(blocks) {
		return Post{}, ErrInvalid
	}
	if replyPostID != "" && !s.replyTargetExists(incidentID, replyPostID, replyBlockID) {
		return Post{}, ErrInvalid
	}
	now := s.now().UTC()
	for i := range blocks {
		blocks[i].ID = newID()
		if blocks[i].SchemaVersion == 0 {
			blocks[i].SchemaVersion = 1
		}
	}
	post := Post{ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID, AuthorID: actorID, ReplyToPostID: replyPostID, ReplyToBlockID: replyBlockID, Revision: 1, Blocks: blocks, CreatedAt: now, UpdatedAt: now}
	s.posts[incidentID] = append(s.posts[incidentID], post)
	s.record(workspaceID, actorID, "post.created", post.ID, now)
	return post, nil
}

func (s *Store) Feed(workspaceID, incidentID string) ([]Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	return append([]Post(nil), s.posts[incidentID]...), nil
}

func (s *Store) revisePost(workspaceID, incidentID, postID, actorID string, blocks []Block) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Post{}, ErrNotFound
	}
	if !validateBlocks(blocks) {
		return Post{}, ErrInvalid
	}
	for i := range s.posts[incidentID] {
		post := &s.posts[incidentID][i]
		if post.ID != postID {
			continue
		}
		if post.AuthorID != actorID {
			return Post{}, ErrForbidden
		}
		for j := range blocks {
			blocks[j].ID = newID()
			if blocks[j].SchemaVersion == 0 {
				blocks[j].SchemaVersion = 1
			}
		}
		post.Blocks, post.Revision, post.UpdatedAt = blocks, post.Revision+1, s.now().UTC()
		s.record(workspaceID, actorID, "post.revised", postID, post.UpdatedAt)
		return *post, nil
	}
	return Post{}, ErrNotFound
}

func validateBlocks(blocks []Block) bool {
	if len(blocks) == 0 {
		return false
	}
	for _, block := range blocks {
		if !validBlockType[block.Type] || block.Payload == nil {
			return false
		}
	}
	return true
}

func (s *Store) replyTargetExists(incidentID, postID, blockID string) bool {
	for _, post := range s.posts[incidentID] {
		if post.ID == postID {
			if blockID == "" {
				return true
			}
			for _, block := range post.Blocks {
				if block.ID == blockID {
					return true
				}
			}
		}
	}
	return false
}

func (s *Store) record(workspaceID, actorID, action, targetID string, at time.Time) {
	s.audit = append(s.audit, AuditEvent{ID: newID(), WorkspaceID: workspaceID, ActorID: actorID, Action: action, TargetID: targetID, At: at})
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}
