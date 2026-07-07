package coordination

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"time-series": true, "retrieval": true,
}

type Block struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	SchemaVersion int            `json:"schemaVersion"`
	Payload       map[string]any `json:"payload"`
}

type Post struct {
	ID                string    `json:"id"`
	WorkspaceID       string    `json:"workspaceId"`
	IncidentID        string    `json:"incidentId"`
	AuthorID          string    `json:"authorId"`
	ReplyToPostID     string    `json:"replyToPostId,omitempty"`
	ReplyToBlockID    string    `json:"replyToBlockId,omitempty"`
	DerivedFromPostID string    `json:"derivedFromPostId,omitempty"`
	Revision          int       `json:"revision"`
	Blocks            []Block   `json:"blocks"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type IncidentRelationship struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspaceId"`
	SourceIncidentID string    `json:"sourceIncidentId"`
	TargetIncidentID string    `json:"targetIncidentId"`
	Kind             string    `json:"kind"`
	CreatedBy        string    `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
}

type IncidentSplitResult struct {
	Incident     Incident             `json:"incident"`
	Relationship IncidentRelationship `json:"relationship"`
	CopiedPosts  []Post               `json:"copiedPosts"`
}

type PermanentChannel struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Incident struct {
	ID                   string             `json:"id"`
	WorkspaceID          string             `json:"workspaceId"`
	Title                string             `json:"title"`
	Description          string             `json:"description,omitempty"`
	OwnerID              string             `json:"ownerId"`
	CreatedBy            string             `json:"createdBy"`
	Severity             string             `json:"severity"`
	Lifecycle            string             `json:"lifecycle"`
	Scope                []string           `json:"scope"`
	VerifiedSummary      string             `json:"verifiedSummary,omitempty"`
	ClosureChecklist     []ChecklistItem    `json:"closureChecklist"`
	TemplateSnapshot     *TemplateSnapshot  `json:"templateSnapshot,omitempty"`
	ConfigurationHistory []TemplateSnapshot `json:"configurationHistory"`
	Detection            *IncidentDetection `json:"detection,omitempty"`
	CreatedAt            time.Time          `json:"createdAt"`
	UpdatedAt            time.Time          `json:"updatedAt"`
}

type IncidentDetection struct {
	DetectorID         string   `json:"detectorId"`
	Trigger            string   `json:"trigger"`
	Model              string   `json:"model"`
	Rule               string   `json:"rule"`
	Confidence         float64  `json:"confidence"`
	ConfidenceGate     float64  `json:"confidenceGate"`
	MinimumSeverity    string   `json:"minimumSeverity"`
	SupportingEvidence []string `json:"supportingEvidence"`
}

type DetectionResult struct {
	Created  bool      `json:"created"`
	Reason   string    `json:"reason,omitempty"`
	Incident *Incident `json:"incident,omitempty"`
}

type IncidentTemplate struct {
	ID               string          `json:"id"`
	WorkspaceID      string          `json:"workspaceId"`
	Name             string          `json:"name"`
	Version          int             `json:"version"`
	Description      string          `json:"description,omitempty"`
	DefaultSeverity  string          `json:"defaultSeverity"`
	DefaultScope     []string        `json:"defaultScope"`
	ClosureChecklist []ChecklistItem `json:"closureChecklist"`
	CreatedBy        string          `json:"createdBy"`
	CreatedAt        time.Time       `json:"createdAt"`
}

type TemplateSnapshot struct {
	TemplateID       string          `json:"templateId"`
	Version          int             `json:"version"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	DefaultSeverity  string          `json:"defaultSeverity"`
	DefaultScope     []string        `json:"defaultScope"`
	ClosureChecklist []ChecklistItem `json:"closureChecklist"`
}

type ChecklistItem struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Completed bool   `json:"completed"`
}

type Membership struct {
	WorkspaceID string     `json:"workspaceId"`
	IncidentID  string     `json:"incidentId"`
	PrincipalID string     `json:"principalId"`
	Role        string     `json:"role"`
	Source      string     `json:"source"`
	Status      string     `json:"status"`
	AddedBy     string     `json:"addedBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
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

type ActionSpecification struct {
	Kind       string         `json:"kind"`
	Parameters map[string]any `json:"parameters"`
}

type Action struct {
	ID                   string              `json:"id"`
	WorkspaceID          string              `json:"workspaceId"`
	IncidentID           string              `json:"incidentId"`
	Title                string              `json:"title"`
	OwnerID              string              `json:"ownerId"`
	Status               string              `json:"status"`
	Specification        ActionSpecification `json:"specification"`
	SpecificationHash    string              `json:"specificationHash"`
	VerificationCriteria string              `json:"verificationCriteria,omitempty"`
	CreatedBy            string              `json:"createdBy"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type PollOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type Vote struct {
	VoterID  string    `json:"voterId"`
	OptionID string    `json:"optionId"`
	CastAt   time.Time `json:"castAt"`
}
type Poll struct {
	ID               string       `json:"id"`
	WorkspaceID      string       `json:"workspaceId"`
	IncidentID       string       `json:"incidentId"`
	Question         string       `json:"question"`
	Mode             string       `json:"mode"`
	Options          []PollOption `json:"options"`
	EligibleVoterIDs []string     `json:"eligibleVoterIds"`
	Quorum           int          `json:"quorum"`
	AllowVoteChanges bool         `json:"allowVoteChanges"`
	Votes            []Vote       `json:"votes"`
	CreatedBy        string       `json:"createdBy"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

type ApprovalResponse struct {
	ApproverID  string    `json:"approverId"`
	Decision    string    `json:"decision"`
	RespondedAt time.Time `json:"respondedAt"`
}
type Approval struct {
	ID                  string             `json:"id"`
	WorkspaceID         string             `json:"workspaceId"`
	IncidentID          string             `json:"incidentId"`
	ActionID            string             `json:"actionId"`
	SpecificationHash   string             `json:"specificationHash"`
	EligibleApproverIDs []string           `json:"eligibleApproverIds"`
	Quorum              int                `json:"quorum"`
	Responses           []ApprovalResponse `json:"responses"`
	Outcome             string             `json:"outcome"`
	CreatedBy           string             `json:"createdBy"`
	CreatedAt           time.Time          `json:"createdAt"`
	UpdatedAt           time.Time          `json:"updatedAt"`
}

type AuditEvent struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	ActorID     string    `json:"actorId"`
	Action      string    `json:"action"`
	TargetID    string    `json:"targetId"`
	At          time.Time `json:"at"`
}

type Store struct {
	mu                  sync.RWMutex
	channels            map[string]PermanentChannel
	incidents           map[string]Incident
	posts               map[string][]Post
	postHistory         map[string][]Post
	facts               map[string][]Fact
	decisions           map[string][]Decision
	actions             map[string][]Action
	polls               map[string][]Poll
	approvals           map[string][]Approval
	templates           map[string][]IncidentTemplate
	memberships         map[string][]Membership
	relationships       []IncidentRelationship
	audit               []AuditEvent
	contextRecipes      map[string][]ContextRecipe
	collections         map[string][]ContextCollection
	alertIncidents      map[string]string
	contextService      ContextService
	agents              map[string]AgentDefinition
	activations         map[string][]AgentActivation
	capabilityGrants    map[string][]CapabilityGrant
	agentRuns           map[string][]AgentRun
	aiProposals         map[string][]AIProposal
	collaboration       map[string][]CollaborationEnvelope
	workflowDefinitions map[string][]WorkflowDefinition
	workflowRuns        map[string][]WorkflowRun
	repositoryConfigs   map[string]RepositoryConfig
	investigations      map[string][]Investigation
	artifacts           map[string][]Artifact
	artifactSimulations map[string][]ArtifactSimulation
	pilotBaselines      map[string]PilotBaseline
	sandbox             SandboxProvider
	github              GitHubService
	modelGateway        ModelGateway
	now                 func() time.Time
}

var validRelationshipKind = map[string]bool{
	"related": true, "duplicate": true, "caused-by": true, "follow-up-to": true,
	"parent-of": true, "recurrence-of": true,
}

func NewStore() *Store {
	return &Store{
		channels:            make(map[string]PermanentChannel),
		incidents:           make(map[string]Incident),
		posts:               make(map[string][]Post),
		postHistory:         make(map[string][]Post),
		facts:               make(map[string][]Fact),
		decisions:           make(map[string][]Decision),
		actions:             make(map[string][]Action),
		polls:               make(map[string][]Poll),
		approvals:           make(map[string][]Approval),
		templates:           make(map[string][]IncidentTemplate),
		memberships:         make(map[string][]Membership),
		contextRecipes:      make(map[string][]ContextRecipe),
		collections:         make(map[string][]ContextCollection),
		alertIncidents:      make(map[string]string),
		agents:              make(map[string]AgentDefinition),
		activations:         make(map[string][]AgentActivation),
		capabilityGrants:    make(map[string][]CapabilityGrant),
		agentRuns:           make(map[string][]AgentRun),
		aiProposals:         make(map[string][]AIProposal),
		collaboration:       make(map[string][]CollaborationEnvelope),
		workflowDefinitions: make(map[string][]WorkflowDefinition),
		workflowRuns:        make(map[string][]WorkflowRun),
		repositoryConfigs:   make(map[string]RepositoryConfig),
		investigations:      make(map[string][]Investigation),
		artifacts:           make(map[string][]Artifact),
		artifactSimulations: make(map[string][]ArtifactSimulation),
		pilotBaselines:      make(map[string]PilotBaseline),
		sandbox:             UnavailableSandbox{},
		github:              UnavailableGitHub{},
		now:                 time.Now,
	}
}

func (s *Store) SetContextService(service ContextService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contextService = service
}

func (s *Store) CreatePermanentChannel(workspaceID, actorID, title, description string) (PermanentChannel, error) {
	title = strings.TrimSpace(title)
	if workspaceID == "" || actorID == "" || title == "" {
		return PermanentChannel{}, ErrInvalid
	}
	now := s.now().UTC()
	channel := PermanentChannel{ID: newID(), WorkspaceID: workspaceID, Title: title, Description: strings.TrimSpace(description), CreatedBy: actorID, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels[channel.ID] = channel
	s.record(workspaceID, actorID, "permanent_channel.created", channel.ID, now)
	return channel, nil
}

func (s *Store) PermanentChannels(workspaceID string) []PermanentChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]PermanentChannel, 0)
	for _, channel := range s.channels {
		if channel.WorkspaceID == workspaceID {
			items = append(items, channel)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	return items
}

func (s *Store) PermanentChannel(workspaceID, channelID string) (PermanentChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channel, ok := s.channels[channelID]
	if !ok || channel.WorkspaceID != workspaceID {
		return PermanentChannel{}, ErrNotFound
	}
	return channel, nil
}

func (s *Store) AddPermanentPost(workspaceID, channelID, actorID, replyPostID, replyBlockID string, blocks []Block) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, ok := s.channels[channelID]
	if !ok || channel.WorkspaceID != workspaceID {
		return Post{}, ErrNotFound
	}
	if actorID == "" || !validateBlocks(blocks) {
		return Post{}, ErrInvalid
	}
	if replyPostID != "" && !s.replyTargetExists(channelID, replyPostID, replyBlockID) {
		return Post{}, ErrInvalid
	}
	blocks = prepareBlocks(blocks)
	now := s.now().UTC()
	post := Post{ID: newID(), WorkspaceID: workspaceID, IncidentID: channelID, AuthorID: actorID, ReplyToPostID: replyPostID, ReplyToBlockID: replyBlockID, Revision: 1, Blocks: blocks, CreatedAt: now, UpdatedAt: now}
	s.posts[channelID] = append(s.posts[channelID], post)
	s.postHistory[post.ID] = []Post{clonePost(post)}
	s.record(workspaceID, actorID, "permanent_channel.post_created", post.ID, now)
	return clonePost(post), nil
}

func (s *Store) PermanentFeed(workspaceID, channelID string) ([]Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channel, ok := s.channels[channelID]
	if !ok || channel.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	return clonePosts(s.posts[channelID]), nil
}

func (s *Store) RevisePermanentPost(workspaceID, channelID, postID, actorID string, blocks []Block) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, ok := s.channels[channelID]
	if !ok || channel.WorkspaceID != workspaceID {
		return Post{}, ErrNotFound
	}
	if !validateBlocks(blocks) {
		return Post{}, ErrInvalid
	}
	for i := range s.posts[channelID] {
		post := &s.posts[channelID][i]
		if post.ID != postID {
			continue
		}
		if actorID == "" || post.AuthorID != actorID {
			return Post{}, ErrForbidden
		}
		post.Blocks, post.Revision, post.UpdatedAt = prepareBlocks(blocks), post.Revision+1, s.now().UTC()
		s.postHistory[post.ID] = append(s.postHistory[post.ID], clonePost(*post))
		s.record(workspaceID, actorID, "permanent_channel.post_revised", postID, post.UpdatedAt)
		return clonePost(*post), nil
	}
	return Post{}, ErrNotFound
}

func (s *Store) PostHistory(workspaceID, channelID, postID string) ([]Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if incident, ok := s.incidents[channelID]; ok {
		if incident.WorkspaceID != workspaceID {
			return nil, ErrNotFound
		}
	} else if channel, ok := s.channels[channelID]; !ok || channel.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	history, ok := s.postHistory[postID]
	if !ok {
		return nil, ErrNotFound
	}
	return clonePosts(history), nil
}

func (s *Store) CreateTemplateVersion(workspaceID, actorID, templateID, name, description, severity string, scope []string, checklist []ChecklistItem) (IncidentTemplate, error) {
	name, description = strings.TrimSpace(name), strings.TrimSpace(description)
	if workspaceID == "" || actorID == "" || name == "" || !validSeverity[severity] || !validChecklist(checklist) {
		return IncidentTemplate{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	creating := templateID == ""
	if templateID == "" {
		templateID = newID()
	}
	versions := s.templates[templateID]
	if len(versions) == 0 && !creating {
		// A caller-supplied ID denotes a new version of an existing template.
		return IncidentTemplate{}, ErrNotFound
	}
	if len(versions) > 0 && versions[0].WorkspaceID != workspaceID {
		return IncidentTemplate{}, ErrNotFound
	}
	now := s.now().UTC()
	t := IncidentTemplate{
		ID: templateID, WorkspaceID: workspaceID, Name: name,
		Version: len(versions) + 1, Description: description,
		DefaultSeverity: severity, DefaultScope: append([]string(nil), scope...),
		ClosureChecklist: cloneChecklist(checklist), CreatedBy: actorID, CreatedAt: now,
	}
	s.templates[templateID] = append(versions, t)
	s.record(workspaceID, actorID, "template.version_created", templateID, now)
	return cloneTemplate(t), nil
}

func (s *Store) Templates(workspaceID string) []IncidentTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []IncidentTemplate{}
	for _, versions := range s.templates {
		if len(versions) > 0 && versions[0].WorkspaceID == workspaceID {
			items = append(items, cloneTemplate(versions[len(versions)-1]))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (s *Store) Template(workspaceID, templateID string, version int) (IncidentTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions := s.templates[templateID]
	if len(versions) == 0 || versions[0].WorkspaceID != workspaceID {
		return IncidentTemplate{}, ErrNotFound
	}
	if version == 0 {
		version = len(versions)
	}
	if version < 1 || version > len(versions) {
		return IncidentTemplate{}, ErrNotFound
	}
	return cloneTemplate(versions[version-1]), nil
}

func (s *Store) CreateIncidentFromTemplate(workspaceID, actorID, templateID string, version int, title, description, severity string, scope []string) (Incident, error) {
	t, err := s.Template(workspaceID, templateID, version)
	if err != nil {
		return Incident{}, err
	}
	if severity == "" {
		severity = t.DefaultSeverity
	}
	if len(scope) == 0 {
		scope = append([]string(nil), t.DefaultScope...)
	}
	incident, err := s.CreateIncident(workspaceID, actorID, title, description, severity, scope)
	if err != nil {
		return Incident{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	incident = s.incidents[incident.ID]
	incident.ClosureChecklist = cloneChecklist(t.ClosureChecklist)
	incident.TemplateSnapshot = &TemplateSnapshot{
		TemplateID: t.ID, Version: t.Version, Name: t.Name,
		Description: t.Description, DefaultSeverity: t.DefaultSeverity,
		DefaultScope:     append([]string(nil), t.DefaultScope...),
		ClosureChecklist: cloneChecklist(t.ClosureChecklist),
	}
	s.incidents[incident.ID] = incident
	return cloneIncident(incident), nil
}

func (s *Store) Memberships(workspaceID, incidentID string) ([]Membership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return nil, ErrNotFound
	}
	return append([]Membership{}, s.memberships[incidentID]...), nil
}

func (s *Store) AddMembership(workspaceID, incidentID, actorID, principalID, role string) (Membership, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Membership{}, ErrNotFound
	}
	if actorID != incident.OwnerID {
		return Membership{}, ErrForbidden
	}
	principalID = strings.TrimSpace(principalID)
	if principalID == "" || !validMemberRole(role) || role == "owner" {
		return Membership{}, ErrInvalid
	}
	for _, member := range s.memberships[incidentID] {
		if member.PrincipalID == principalID && member.Status == "active" {
			return Membership{}, ErrConflict
		}
	}
	now := s.now().UTC()
	member := Membership{WorkspaceID: workspaceID, IncidentID: incidentID, PrincipalID: principalID, Role: role, Source: "direct", Status: "active", AddedBy: actorID, CreatedAt: now, UpdatedAt: now}
	s.memberships[incidentID] = append(s.memberships[incidentID], member)
	s.record(workspaceID, actorID, "membership.added", principalID, now)
	return member, nil
}

func (s *Store) UpdateMembership(workspaceID, incidentID, actorID, principalID, role string, revoke bool) (Membership, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Membership{}, ErrNotFound
	}
	if actorID != incident.OwnerID {
		return Membership{}, ErrForbidden
	}
	if principalID == incident.OwnerID {
		return Membership{}, ErrConflict
	}
	if !revoke && (!validMemberRole(role) || role == "owner") {
		return Membership{}, ErrInvalid
	}
	for i := range s.memberships[incidentID] {
		member := &s.memberships[incidentID][i]
		if member.PrincipalID != principalID || member.Status != "active" {
			continue
		}
		now := s.now().UTC()
		member.UpdatedAt = now
		if revoke {
			member.Status, member.RevokedAt = "revoked", &now
		} else {
			member.Role = role
		}
		s.record(workspaceID, actorID, "membership.updated", principalID, now)
		return *member, nil
	}
	return Membership{}, ErrNotFound
}

func (s *Store) TransferOwnership(workspaceID, incidentID, actorID, newOwnerID string) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if actorID != incident.OwnerID {
		return Incident{}, ErrForbidden
	}
	if newOwnerID == "" || newOwnerID == actorID {
		return Incident{}, ErrInvalid
	}
	memberIndex := -1
	for i, member := range s.memberships[incidentID] {
		if member.PrincipalID == newOwnerID && member.Status == "active" {
			memberIndex = i
			break
		}
	}
	if memberIndex < 0 {
		return Incident{}, ErrConflict
	}
	now := s.now().UTC()
	for i := range s.memberships[incidentID] {
		if s.memberships[incidentID][i].PrincipalID == actorID && s.memberships[incidentID][i].Status == "active" {
			s.memberships[incidentID][i].Role, s.memberships[incidentID][i].UpdatedAt = "editor", now
		}
	}
	s.memberships[incidentID][memberIndex].Role, s.memberships[incidentID][memberIndex].UpdatedAt = "owner", now
	incident.OwnerID, incident.UpdatedAt = newOwnerID, now
	s.incidents[incidentID] = incident
	s.record(workspaceID, actorID, "incident.ownership_transferred", incidentID, now)
	return cloneIncident(incident), nil
}

func validMemberRole(role string) bool {
	return role == "owner" || role == "editor" || role == "participant" || role == "viewer"
}

// activeRole is called while the store lock is held. Membership is the
// authorization boundary for incident mutations; revoked entries never grant
// access even when an older entry exists for the same principal.
func (s *Store) activeRole(incidentID, principalID string) (string, bool) {
	for i := len(s.memberships[incidentID]) - 1; i >= 0; i-- {
		member := s.memberships[incidentID][i]
		if member.PrincipalID == principalID && member.Status == "active" {
			return member.Role, true
		}
	}
	return "", false
}

func (s *Store) canParticipate(incidentID, principalID string) bool {
	role, ok := s.activeRole(incidentID, principalID)
	return ok && role != "viewer"
}

func (s *Store) canEdit(incidentID, principalID string) bool {
	role, ok := s.activeRole(incidentID, principalID)
	return ok && (role == "owner" || role == "editor")
}

func (s *Store) CreateIncident(workspaceID, actorID, title, description, severity string, scope []string) (Incident, error) {
	title = strings.TrimSpace(title)
	if workspaceID == "" || actorID == "" || title == "" || !validSeverity[severity] {
		return Incident{}, ErrInvalid
	}
	now := s.now().UTC()
	incident := Incident{
		ID: newID(), WorkspaceID: workspaceID, Title: title,
		Description: strings.TrimSpace(description), OwnerID: actorID, CreatedBy: actorID,
		Severity: severity, Lifecycle: "open", Scope: scope,
		ClosureChecklist: defaultClosureChecklist(), CreatedAt: now, UpdatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.incidents[incident.ID] = incident
	s.memberships[incident.ID] = []Membership{{WorkspaceID: workspaceID, IncidentID: incident.ID, PrincipalID: actorID, Role: "owner", Source: "creator", Status: "active", AddedBy: actorID, CreatedAt: now, UpdatedAt: now}}
	s.record(workspaceID, actorID, "incident.created", incident.ID, now)
	return cloneIncident(incident), nil
}

func (s *Store) Incident(workspaceID, id string) (Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[id]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	return cloneIncident(incident), nil
}

func (s *Store) ListIncidents(workspaceID string) []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Incident, 0)
	for _, incident := range s.incidents {
		if incident.WorkspaceID == workspaceID {
			items = append(items, cloneIncident(incident))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items
}

func (s *Store) ListIncidentsForPrincipal(workspaceID, principalID string) []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []Incident{}
	for _, incident := range s.incidents {
		if incident.WorkspaceID != workspaceID {
			continue
		}
		if _, ok := s.activeRole(incident.ID, principalID); ok {
			result = append(result, cloneIncident(incident))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result
}

func (s *Store) CanReadIncident(workspaceID, incidentID, principalID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return ErrNotFound
	}
	if principalID == "" {
		return ErrForbidden
	}
	if _, ok := s.activeRole(incidentID, principalID); !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Store) RelateIncidents(workspaceID, sourceID, targetID, actorID, kind string) (IncidentRelationship, error) {
	kind = strings.TrimSpace(kind)
	if sourceID == targetID || !validRelationshipKind[kind] {
		return IncidentRelationship{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	source, sourceOK := s.incidents[sourceID]
	target, targetOK := s.incidents[targetID]
	if !sourceOK || !targetOK || source.WorkspaceID != workspaceID || target.WorkspaceID != workspaceID {
		return IncidentRelationship{}, ErrNotFound
	}
	if !s.canEdit(sourceID, actorID) {
		return IncidentRelationship{}, ErrForbidden
	}
	if _, ok := s.activeRole(targetID, actorID); !ok {
		return IncidentRelationship{}, ErrForbidden
	}
	for _, item := range s.relationships {
		if item.SourceIncidentID == sourceID && item.TargetIncidentID == targetID && item.Kind == kind {
			return IncidentRelationship{}, ErrConflict
		}
	}
	now := s.now().UTC()
	item := IncidentRelationship{ID: newID(), WorkspaceID: workspaceID, SourceIncidentID: sourceID, TargetIncidentID: targetID, Kind: kind, CreatedBy: actorID, CreatedAt: now}
	s.relationships = append(s.relationships, item)
	s.record(workspaceID, actorID, "incident.related."+kind, item.ID, now)
	return item, nil
}

func (s *Store) DetectIncident(workspaceID, sponsorID, detectorID, title, severity, trigger, rule string, confidence, confidenceGate float64, minimumSeverity string, evidence []string) (DetectionResult, error) {
	title, trigger, rule = strings.TrimSpace(title), strings.TrimSpace(trigger), strings.TrimSpace(rule)
	if workspaceID == "" || sponsorID == "" || detectorID == "" || title == "" || trigger == "" || rule == "" || confidence < 0 || confidence > 1 || confidenceGate < 0 || confidenceGate > 1 || !validSeverity[severity] || !validSeverity[minimumSeverity] || len(evidence) == 0 {
		return DetectionResult{}, ErrInvalid
	}
	s.mu.RLock()
	detector, ok := s.agents[detectorID]
	s.mu.RUnlock()
	if !ok || detector.WorkspaceID != workspaceID || detector.Status != "approved" {
		return DetectionResult{}, ErrForbidden
	}
	if confidence < confidenceGate {
		return DetectionResult{Created: false, Reason: "confidence_below_gate"}, nil
	}
	severityRank := map[string]int{"SEV-1": 1, "SEV-2": 2, "SEV-3": 3, "SEV-4": 4, "unclassified": 5}
	if severityRank[severity] > severityRank[minimumSeverity] {
		return DetectionResult{Created: false, Reason: "severity_below_gate"}, nil
	}
	incident, err := s.CreateIncident(workspaceID, sponsorID, title, "Created by approved AI detection", severity, nil)
	if err != nil {
		return DetectionResult{}, err
	}
	detection := &IncidentDetection{DetectorID: detectorID, Trigger: trigger, Model: detector.Model, Rule: rule, Confidence: confidence, ConfidenceGate: confidenceGate, MinimumSeverity: minimumSeverity, SupportingEvidence: append([]string(nil), evidence...)}
	s.mu.Lock()
	stored := s.incidents[incident.ID]
	stored.Detection = detection
	stored.CreatedBy = detectorID
	s.incidents[incident.ID] = stored
	s.record(workspaceID, detectorID, "incident.created_from_ai_detection", incident.ID, stored.CreatedAt)
	s.mu.Unlock()
	incident.Detection = detection
	incident.CreatedBy = detectorID
	return DetectionResult{Created: true, Incident: &incident}, nil
}

func (s *Store) MigrateIncidentTemplate(workspaceID, incidentID, actorID, templateID string, version int, severityOverride string, scopeOverride []string) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if !s.canEdit(incidentID, actorID) {
		return Incident{}, ErrForbidden
	}
	versions := s.templates[templateID]
	if len(versions) == 0 || versions[0].WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if version == 0 {
		version = len(versions)
	}
	if version < 1 || version > len(versions) {
		return Incident{}, ErrNotFound
	}
	template := versions[version-1]
	if incident.TemplateSnapshot != nil {
		incident.ConfigurationHistory = append(incident.ConfigurationHistory, *incident.TemplateSnapshot)
	}
	snapshot := TemplateSnapshot{TemplateID: template.ID, Version: template.Version, Name: template.Name, Description: template.Description, DefaultSeverity: template.DefaultSeverity, DefaultScope: append([]string{}, template.DefaultScope...), ClosureChecklist: cloneChecklist(template.ClosureChecklist)}
	incident.TemplateSnapshot = &snapshot
	incident.Severity, incident.Scope, incident.ClosureChecklist = template.DefaultSeverity, append([]string{}, template.DefaultScope...), cloneChecklist(template.ClosureChecklist)
	if severityOverride != "" {
		if !validSeverity[severityOverride] {
			return Incident{}, ErrInvalid
		}
		incident.Severity = severityOverride
	}
	if scopeOverride != nil {
		incident.Scope = append([]string{}, scopeOverride...)
	}
	incident.UpdatedAt = s.now().UTC()
	s.incidents[incidentID] = incident
	s.record(workspaceID, actorID, "incident.configuration_migrated", incidentID, incident.UpdatedAt)
	return cloneIncident(incident), nil
}

func (s *Store) CancelDetectedIncident(workspaceID, incidentID, actorID string) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if !s.canEdit(incidentID, actorID) {
		return Incident{}, ErrForbidden
	}
	if incident.Detection == nil || incident.Lifecycle == "cancelled" {
		return Incident{}, ErrConflict
	}
	incident.Lifecycle = "cancelled"
	incident.UpdatedAt = s.now().UTC()
	s.incidents[incidentID] = incident
	s.record(workspaceID, actorID, "incident.ai_false_alarm_cancelled", incidentID, incident.UpdatedAt)
	return cloneIncident(incident), nil
}

func (s *Store) IncidentRelationships(workspaceID, incidentID, actorID string) ([]IncidentRelationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	if _, ok = s.activeRole(incidentID, actorID); !ok {
		return nil, ErrForbidden
	}
	result := []IncidentRelationship{}
	for _, item := range s.relationships {
		if item.WorkspaceID != workspaceID || (item.SourceIncidentID != incidentID && item.TargetIncidentID != incidentID) {
			continue
		}
		other := item.SourceIncidentID
		if other == incidentID {
			other = item.TargetIncidentID
		}
		if _, visible := s.activeRole(other, actorID); visible {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Store) SplitIncident(workspaceID, sourceID, actorID, title string, postIDs []string) (IncidentSplitResult, error) {
	title = strings.TrimSpace(title)
	if title == "" || len(postIDs) == 0 {
		return IncidentSplitResult{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.incidents[sourceID]
	if !ok || source.WorkspaceID != workspaceID {
		return IncidentSplitResult{}, ErrNotFound
	}
	if !s.canEdit(sourceID, actorID) {
		return IncidentSplitResult{}, ErrForbidden
	}
	wanted := map[string]bool{}
	for _, id := range postIDs {
		wanted[id] = true
	}
	selected := []Post{}
	for _, post := range s.posts[sourceID] {
		if wanted[post.ID] {
			selected = append(selected, clonePost(post))
			delete(wanted, post.ID)
		}
	}
	if len(wanted) != 0 {
		return IncidentSplitResult{}, ErrInvalid
	}
	now := s.now().UTC()
	incident := Incident{ID: newID(), WorkspaceID: workspaceID, Title: title, Description: "Split from incident " + sourceID, OwnerID: actorID, Severity: source.Severity, Lifecycle: "open", Scope: append([]string{}, source.Scope...), ClosureChecklist: defaultClosureChecklist(), CreatedAt: now, UpdatedAt: now}
	s.incidents[incident.ID] = incident
	s.memberships[incident.ID] = []Membership{{WorkspaceID: workspaceID, IncidentID: incident.ID, PrincipalID: actorID, Role: "owner", Source: "split", Status: "active", AddedBy: actorID, CreatedAt: now, UpdatedAt: now}}
	postMap, blockMap := map[string]string{}, map[string]string{}
	for _, post := range selected {
		postMap[post.ID] = newID()
		for _, block := range post.Blocks {
			blockMap[block.ID] = newID()
		}
	}
	copied := []Post{}
	for _, original := range selected {
		copy := clonePost(original)
		copy.ID, copy.IncidentID, copy.DerivedFromPostID = postMap[original.ID], incident.ID, original.ID
		copy.Revision, copy.CreatedAt, copy.UpdatedAt = 1, now, now
		if mapped := postMap[original.ReplyToPostID]; mapped != "" {
			copy.ReplyToPostID = mapped
		} else {
			copy.ReplyToPostID = ""
		}
		if mapped := blockMap[original.ReplyToBlockID]; mapped != "" {
			copy.ReplyToBlockID = mapped
		} else {
			copy.ReplyToBlockID = ""
		}
		for i := range copy.Blocks {
			copy.Blocks[i].ID = blockMap[original.Blocks[i].ID]
		}
		s.posts[incident.ID] = append(s.posts[incident.ID], copy)
		s.postHistory[copy.ID] = []Post{clonePost(copy)}
		copied = append(copied, clonePost(copy))
	}
	rel := IncidentRelationship{ID: newID(), WorkspaceID: workspaceID, SourceIncidentID: sourceID, TargetIncidentID: incident.ID, Kind: "parent-of", CreatedBy: actorID, CreatedAt: now}
	s.relationships = append(s.relationships, rel)
	s.record(workspaceID, actorID, "incident.split", incident.ID, now)
	return IncidentSplitResult{Incident: cloneIncident(incident), Relationship: rel, CopiedPosts: copied}, nil
}

func (s *Store) DerivePost(workspaceID, sourceIncidentID, postID, destinationIncidentID, actorID string) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source, sourceOK := s.incidents[sourceIncidentID]
	destination, destinationOK := s.incidents[destinationIncidentID]
	if !sourceOK || !destinationOK || source.WorkspaceID != workspaceID || destination.WorkspaceID != workspaceID {
		return Post{}, ErrNotFound
	}
	if _, ok := s.activeRole(sourceIncidentID, actorID); !ok {
		return Post{}, ErrForbidden
	}
	if !s.canParticipate(destinationIncidentID, actorID) {
		return Post{}, ErrForbidden
	}
	var original *Post
	for i := range s.posts[sourceIncidentID] {
		if s.posts[sourceIncidentID][i].ID == postID {
			copy := clonePost(s.posts[sourceIncidentID][i])
			original = &copy
			break
		}
	}
	if original == nil {
		return Post{}, ErrNotFound
	}
	now := s.now().UTC()
	blocks := prepareBlocks(original.Blocks)
	for i := range blocks {
		blocks[i].Payload["derivedFromIncidentId"] = sourceIncidentID
		blocks[i].Payload["derivedFromPostId"] = postID
	}
	post := Post{ID: newID(), WorkspaceID: workspaceID, IncidentID: destinationIncidentID, AuthorID: actorID, DerivedFromPostID: postID, Revision: 1, Blocks: blocks, CreatedAt: now, UpdatedAt: now}
	s.posts[destinationIncidentID] = append(s.posts[destinationIncidentID], post)
	s.postHistory[post.ID] = []Post{clonePost(post)}
	s.record(workspaceID, actorID, "post.derived", post.ID, now)
	return clonePost(post), nil
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
		if lifecycle == "archived" {
			hasPublishedKnowledge := false
			for _, versions := range s.artifacts {
				for _, artifact := range versions {
					if artifact.SourceIncidentID == id && artifact.Status == "published" {
						hasPublishedKnowledge = true
					}
				}
			}
			if !hasPublishedKnowledge {
				return Incident{}, ErrConflict
			}
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
		return Incident{}, ErrInvalid
	}
	incident.UpdatedAt = s.now().UTC()
	s.incidents[id] = incident
	s.record(workspaceID, actorID, "incident.updated", id, incident.UpdatedAt)
	return cloneIncident(incident), nil
}

func (s *Store) UpdateResolution(workspaceID, incidentID, actorID, summary, checklistItemID string, completed *bool) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Incident{}, ErrNotFound
	}
	if !s.canEdit(incidentID, actorID) {
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
	return cloneIncident(incident), nil
}

func (s *Store) AddFact(workspaceID, incidentID, actorID, statement string, evidence []string) (Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Fact{}, ErrNotFound
	}
	if !s.canParticipate(incidentID, actorID) {
		return Fact{}, ErrForbidden
	}
	statement = strings.TrimSpace(statement)
	if actorID == "" || statement == "" || !s.evidenceExists(incidentID, evidence) {
		return Fact{}, ErrInvalid
	}
	now := s.now().UTC()
	fact := Fact{
		ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID,
		Statement: statement, State: "unverified", EvidenceBlockIDs: evidence,
		ProposedBy: actorID, UpdatedBy: actorID, CreatedAt: now, UpdatedAt: now,
	}
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
	if !s.canEdit(incidentID, actorID) {
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
	return append([]Fact{}, s.facts[incidentID]...), nil
}

func (s *Store) AddDecision(workspaceID, incidentID, actorID, statement, rationale string, evidence []string) (Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Decision{}, ErrNotFound
	}
	if !s.canParticipate(incidentID, actorID) {
		return Decision{}, ErrForbidden
	}
	statement = strings.TrimSpace(statement)
	if actorID == "" || statement == "" || !s.evidenceExists(incidentID, evidence) {
		return Decision{}, ErrInvalid
	}
	now := s.now().UTC()
	d := Decision{
		ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID,
		Statement: statement, Rationale: strings.TrimSpace(rationale),
		Status: "proposed", EvidenceBlockIDs: evidence, ProposedBy: actorID,
		CreatedAt: now, UpdatedAt: now,
	}
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
	if !s.canEdit(incidentID, actorID) {
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
	return append([]Decision{}, s.decisions[incidentID]...), nil
}

func (s *Store) AddAction(workspaceID, incidentID, actorID, title, ownerID, kind string, parameters map[string]any, verification string) (Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Action{}, ErrNotFound
	}
	if !s.canEdit(incidentID, actorID) {
		return Action{}, ErrForbidden
	}
	title, ownerID, kind = strings.TrimSpace(title), strings.TrimSpace(ownerID), strings.TrimSpace(kind)
	if actorID == "" || title == "" || ownerID == "" || kind == "" || parameters == nil {
		return Action{}, ErrInvalid
	}
	if !s.canParticipate(incidentID, ownerID) {
		return Action{}, ErrInvalid
	}
	encodedParameters, err := json.Marshal(parameters)
	if err != nil {
		return Action{}, ErrInvalid
	}
	var immutableParameters map[string]any
	if err := json.Unmarshal(encodedParameters, &immutableParameters); err != nil {
		return Action{}, ErrInvalid
	}
	spec := ActionSpecification{Kind: kind, Parameters: immutableParameters}
	hash, err := specificationHash(spec)
	if err != nil {
		return Action{}, ErrInvalid
	}
	now := s.now().UTC()
	action := Action{
		ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID,
		Title: title, OwnerID: ownerID, Status: "proposed",
		Specification: spec, SpecificationHash: hash,
		VerificationCriteria: strings.TrimSpace(verification),
		CreatedBy:            actorID, CreatedAt: now, UpdatedAt: now,
	}
	s.actions[incidentID] = append(s.actions[incidentID], action)
	s.record(workspaceID, actorID, "action.created", action.ID, now)
	return cloneAction(action), nil
}

func (s *Store) UpdateAction(workspaceID, incidentID, actionID, actorID, status string) (Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Action{}, ErrNotFound
	}
	for i := range s.actions[incidentID] {
		a := &s.actions[incidentID][i]
		if a.ID != actionID {
			continue
		}
		if !s.canParticipate(incidentID, actorID) || (actorID != a.OwnerID && !s.canEdit(incidentID, actorID)) {
			return Action{}, ErrForbidden
		}
		allowed := map[string]map[string]bool{
			"proposed": {"ready": true, "cancelled": true}, "ready": {"in-progress": true, "cancelled": true},
			"in-progress": {"blocked": true, "verification": true, "failed": true, "cancelled": true},
			"blocked":     {"in-progress": true, "failed": true, "cancelled": true}, "verification": {"completed": true, "in-progress": true, "failed": true},
		}
		if !allowed[a.Status][status] {
			return Action{}, ErrConflict
		}
		a.Status, a.UpdatedAt = status, s.now().UTC()
		s.record(workspaceID, actorID, "action."+status, actionID, a.UpdatedAt)
		return cloneAction(*a), nil
	}
	return Action{}, ErrNotFound
}

func (s *Store) Actions(workspaceID, incidentID string) ([]Action, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return nil, ErrNotFound
	}
	items := make([]Action, len(s.actions[incidentID]))
	for i, action := range s.actions[incidentID] {
		items[i] = cloneAction(action)
	}
	return items, nil
}

func (s *Store) AddPoll(workspaceID, incidentID, actorID, question, mode string, labels, eligible []string, quorum int, allowChanges bool) (Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Poll{}, ErrNotFound
	}
	if !s.canEdit(incidentID, actorID) {
		return Poll{}, ErrForbidden
	}
	question = strings.TrimSpace(question)
	if actorID == "" || question == "" || (mode != "advisory" && mode != "binding") || len(labels) < 2 || len(eligible) == 0 || quorum < 1 || quorum > len(eligible) || hasDuplicates(eligible) {
		return Poll{}, ErrInvalid
	}
	for _, voterID := range eligible {
		if !s.canParticipate(incidentID, voterID) {
			return Poll{}, ErrInvalid
		}
	}
	options := make([]PollOption, len(labels))
	for i, label := range labels {
		if strings.TrimSpace(label) == "" {
			return Poll{}, ErrInvalid
		}
		options[i] = PollOption{ID: newID(), Label: strings.TrimSpace(label)}
	}
	now := s.now().UTC()
	poll := Poll{
		ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID,
		Question: question, Mode: mode, Options: options,
		EligibleVoterIDs: append([]string(nil), eligible...), Quorum: quorum,
		AllowVoteChanges: allowChanges, Votes: []Vote{},
		CreatedBy: actorID, CreatedAt: now, UpdatedAt: now,
	}
	s.polls[incidentID] = append(s.polls[incidentID], poll)
	s.record(workspaceID, actorID, "poll.created", poll.ID, now)
	return poll, nil
}

func (s *Store) Vote(workspaceID, incidentID, pollID, actorID, optionID string) (Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Poll{}, ErrNotFound
	}
	if !s.canParticipate(incidentID, actorID) {
		return Poll{}, ErrForbidden
	}
	for i := range s.polls[incidentID] {
		p := &s.polls[incidentID][i]
		if p.ID != pollID {
			continue
		}
		if !contains(p.EligibleVoterIDs, actorID) {
			return Poll{}, ErrForbidden
		}
		if !pollHasOption(*p, optionID) {
			return Poll{}, ErrInvalid
		}
		now := s.now().UTC()
		for j := range p.Votes {
			if p.Votes[j].VoterID == actorID {
				if !p.AllowVoteChanges {
					return Poll{}, ErrConflict
				}
				p.Votes[j] = Vote{VoterID: actorID, OptionID: optionID, CastAt: now}
				p.UpdatedAt = now
				s.record(workspaceID, actorID, "poll.vote_changed", pollID, now)
				return *p, nil
			}
		}
		p.Votes = append(p.Votes, Vote{VoterID: actorID, OptionID: optionID, CastAt: now})
		p.UpdatedAt = now
		s.record(workspaceID, actorID, "poll.voted", pollID, now)
		return *p, nil
	}
	return Poll{}, ErrNotFound
}

func (s *Store) Polls(workspaceID, incidentID string) ([]Poll, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return nil, ErrNotFound
	}
	return append([]Poll{}, s.polls[incidentID]...), nil
}

func (s *Store) RequestApproval(workspaceID, incidentID, actionID, actorID string, eligible []string, quorum int) (Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Approval{}, ErrNotFound
	}
	if !s.canEdit(incidentID, actorID) {
		return Approval{}, ErrForbidden
	}
	if actorID == "" || len(eligible) == 0 || quorum < 1 || quorum > len(eligible) || hasDuplicates(eligible) {
		return Approval{}, ErrInvalid
	}
	for _, approverID := range eligible {
		if !s.canParticipate(incidentID, approverID) {
			return Approval{}, ErrInvalid
		}
	}
	var action *Action
	for i := range s.actions[incidentID] {
		if s.actions[incidentID][i].ID == actionID {
			action = &s.actions[incidentID][i]
			break
		}
	}
	if action == nil {
		return Approval{}, ErrNotFound
	}
	now := s.now().UTC()
	approval := Approval{
		ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID,
		ActionID: actionID, SpecificationHash: action.SpecificationHash,
		EligibleApproverIDs: append([]string(nil), eligible...), Quorum: quorum,
		Responses: []ApprovalResponse{}, Outcome: "pending",
		CreatedBy: actorID, CreatedAt: now, UpdatedAt: now,
	}
	s.approvals[incidentID] = append(s.approvals[incidentID], approval)
	s.record(workspaceID, actorID, "approval.requested", approval.ID, now)
	return approval, nil
}

func (s *Store) RespondApproval(workspaceID, incidentID, approvalID, actorID, decision string) (Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return Approval{}, ErrNotFound
	}
	if !s.canParticipate(incidentID, actorID) {
		return Approval{}, ErrForbidden
	}
	if decision != "approve" && decision != "reject" {
		return Approval{}, ErrInvalid
	}
	for i := range s.approvals[incidentID] {
		a := &s.approvals[incidentID][i]
		if a.ID != approvalID {
			continue
		}
		if a.Outcome != "pending" {
			return Approval{}, ErrConflict
		}
		if !contains(a.EligibleApproverIDs, actorID) {
			return Approval{}, ErrForbidden
		}
		for _, response := range a.Responses {
			if response.ApproverID == actorID {
				return Approval{}, ErrConflict
			}
		}
		now := s.now().UTC()
		a.Responses = append(a.Responses, ApprovalResponse{ApproverID: actorID, Decision: decision, RespondedAt: now})
		a.UpdatedAt = now
		if decision == "reject" {
			a.Outcome = "rejected"
		} else {
			approved := 0
			for _, response := range a.Responses {
				if response.Decision == "approve" {
					approved++
				}
			}
			if approved >= a.Quorum {
				a.Outcome = "approved"
			}
		}
		s.record(workspaceID, actorID, "approval."+decision, approvalID, now)
		return *a, nil
	}
	return Approval{}, ErrNotFound
}

func (s *Store) Approvals(workspaceID, incidentID string) ([]Approval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.incidentExists(workspaceID, incidentID) {
		return nil, ErrNotFound
	}
	return append([]Approval{}, s.approvals[incidentID]...), nil
}

func specificationHash(spec ActionSpecification) (string, error) {
	value, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:]), nil
}
func cloneAction(action Action) Action {
	encoded, _ := json.Marshal(action.Specification.Parameters)
	var parameters map[string]any
	_ = json.Unmarshal(encoded, &parameters)
	action.Specification.Parameters = parameters
	return action
}
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func hasDuplicates(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}
func pollHasOption(p Poll, id string) bool {
	for _, option := range p.Options {
		if option.ID == id {
			return true
		}
	}
	return false
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

func validChecklist(items []ChecklistItem) bool {
	if len(items) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Label) == "" || item.Completed || seen[item.ID] {
			return false
		}
		seen[item.ID] = true
	}
	return true
}
func cloneChecklist(items []ChecklistItem) []ChecklistItem {
	return append([]ChecklistItem{}, items...)
}
func cloneTemplate(t IncidentTemplate) IncidentTemplate {
	t.DefaultScope = append([]string(nil), t.DefaultScope...)
	t.ClosureChecklist = cloneChecklist(t.ClosureChecklist)
	return t
}
func cloneIncident(incident Incident) Incident {
	// API collection fields must remain JSON arrays even when callers omit them.
	// A nil slice serializes as null, which breaks clients expecting a stable
	// list-shaped contract.
	incident.Scope = append([]string{}, incident.Scope...)
	incident.ClosureChecklist = cloneChecklist(incident.ClosureChecklist)
	incident.ConfigurationHistory = append([]TemplateSnapshot{}, incident.ConfigurationHistory...)
	for i := range incident.ConfigurationHistory {
		incident.ConfigurationHistory[i].DefaultScope = append([]string(nil), incident.ConfigurationHistory[i].DefaultScope...)
		incident.ConfigurationHistory[i].ClosureChecklist = cloneChecklist(incident.ConfigurationHistory[i].ClosureChecklist)
	}
	if incident.TemplateSnapshot != nil {
		snapshot := *incident.TemplateSnapshot
		snapshot.DefaultScope = append([]string(nil), snapshot.DefaultScope...)
		snapshot.ClosureChecklist = cloneChecklist(snapshot.ClosureChecklist)
		incident.TemplateSnapshot = &snapshot
	}
	if incident.Detection != nil {
		detection := *incident.Detection
		detection.SupportingEvidence = append([]string(nil), detection.SupportingEvidence...)
		incident.Detection = &detection
	}
	return incident
}

func (s *Store) AddPost(workspaceID, incidentID, actorID, replyPostID, replyBlockID string, blocks []Block) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Post{}, ErrNotFound
	}
	if !s.canParticipate(incidentID, actorID) {
		return Post{}, ErrForbidden
	}
	if actorID == "" || len(blocks) == 0 || !validateBlocks(blocks) {
		return Post{}, ErrInvalid
	}
	if replyPostID != "" && !s.replyTargetExists(incidentID, replyPostID, replyBlockID) {
		return Post{}, ErrInvalid
	}
	now := s.now().UTC()
	blocks = prepareBlocks(blocks)
	post := Post{ID: newID(), WorkspaceID: workspaceID, IncidentID: incidentID, AuthorID: actorID, ReplyToPostID: replyPostID, ReplyToBlockID: replyBlockID, Revision: 1, Blocks: blocks, CreatedAt: now, UpdatedAt: now}
	s.posts[incidentID] = append(s.posts[incidentID], post)
	s.postHistory[post.ID] = []Post{clonePost(post)}
	s.record(workspaceID, actorID, "post.created", post.ID, now)
	return clonePost(post), nil
}

func (s *Store) Feed(workspaceID, incidentID string) ([]Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	return clonePosts(s.posts[incidentID]), nil
}

func (s *Store) revisePost(workspaceID, incidentID, postID, actorID string, blocks []Block) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[incidentID]
	if !ok || incident.WorkspaceID != workspaceID {
		return Post{}, ErrNotFound
	}
	if !s.canParticipate(incidentID, actorID) {
		return Post{}, ErrForbidden
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
		for _, block := range post.Blocks {
			if block.Type == "retrieval" || block.Type == "time-series" {
				return Post{}, ErrConflict
			}
		}
		blocks = prepareBlocks(blocks)
		post.Blocks, post.Revision, post.UpdatedAt = blocks, post.Revision+1, s.now().UTC()
		s.postHistory[post.ID] = append(s.postHistory[post.ID], clonePost(*post))
		s.record(workspaceID, actorID, "post.revised", postID, post.UpdatedAt)
		return clonePost(*post), nil
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

func prepareBlocks(blocks []Block) []Block {
	prepared := make([]Block, len(blocks))
	for i, block := range blocks {
		prepared[i] = block
		prepared[i].ID = newID()
		if prepared[i].SchemaVersion == 0 {
			prepared[i].SchemaVersion = 1
		}
		prepared[i].Payload = clonePayload(block.Payload)
	}
	return prepared
}

func clonePayload(payload map[string]any) map[string]any {
	encoded, _ := json.Marshal(payload)
	var result map[string]any
	_ = json.Unmarshal(encoded, &result)
	return result
}

func clonePost(post Post) Post {
	post.Blocks = append([]Block(nil), post.Blocks...)
	for i := range post.Blocks {
		post.Blocks[i].Payload = clonePayload(post.Blocks[i].Payload)
	}
	return post
}

func clonePosts(posts []Post) []Post {
	result := make([]Post, len(posts))
	for i, post := range posts {
		result[i] = clonePost(post)
	}
	return result
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
