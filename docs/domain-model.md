# Domain model

## 1. Aggregate boundaries

The model uses peer channel types rather than nesting incident channels under
permanent channels.

```text
Workspace
├── Channel(type=permanent)
├── Channel(type=incident)
├── Template
├── AgentDefinition
├── Integration
└── WorkspacePolicy
```

The `Channel` aggregate owns discussion membership and presentation. Incident
operational state is held by an `IncidentState` aggregate keyed to an incident
channel. This prevents permanent channels from accumulating nullable incident
fields.

## 2. Principal entities

### Workspace

- ID, name, region, status
- identity-provider configuration
- data-classification and model policy
- workspace roles and platform limits
- created and updated timestamps

### Channel

- ID and workspace ID
- type: `permanent` or `incident`
- title, description, status
- owner user ID for incident channels
- template snapshot ID where applicable
- configuration and policy snapshot IDs
- created-by principal and creation trigger
- archive state

Channel relationships are typed references such as related-to, duplicate-of,
caused-by, follow-up-to, parent-of, and recurrence-of. A canonical incident may
redirect duplicates without deleting their history.

### Membership

- channel ID and principal ID
- principal type: user or agent
- role: owner, editor, participant, or viewer
- source: direct, workspace rule, template rule, or recommendation
- activation and expiration timestamps
- status and revocation reason

Workspace and channel administration are separate from incident ownership.

### Post and block

`Post` records channel, author principal, creation origin, timestamps, revision,
and presentation state. `PostBlock` records its type, schema version, structured
payload, source metadata, permissions, and display order.

Replies use an adjacency relationship and can target a post or block. Logical
depth is unlimited. Read models provide path, depth, descendant count, and branch
summaries for efficient navigation.

Post revisions are append-only. A current revision pointer provides efficient
reads without destroying history.

### Context source and snapshot

A context snapshot records:

- integration and source object identifiers
- retrieval operation and normalised query
- source time range and retrieval timestamp
- immutable excerpt or object reference
- source URL if available
- freshness and completeness
- transformation and visualisation metadata
- classification, redaction, and visibility
- agent run and principal responsible for retrieval

Reposting creates a new immutable snapshot with a provenance link. It does not
create a live copy or bypass destination access checks.

### IncidentState

- severity and lifecycle state
- optional structured scope dimensions
- proposed and accepted summaries
- inactivity and closure state
- current workflow run
- resolution checklist
- verification and review status

Scope is intentionally extensible. Fields such as systems, environment,
customers, geography, time window, and symptoms are optional typed dimensions,
not mandatory columns in every template.

### Evidence and fact

An `EvidenceReference` points to a post block or external snapshot. A `Fact`
contains a versioned statement and state: unverified, corroborated, disputed,
superseded, or invalidated. Facts carry supporting and opposing references.

### Decision

- immutable statement and rationale
- status: proposed, accepted, rejected, or superseded
- alternatives and evidence
- proposing and accepting principals
- approval or poll reference
- predecessor/successor relationship
- affected actions

### Poll and vote

Poll definitions include advisory/binding mode, options, eligibility rule,
quorum, deadline, vote-change policy, and resulting decision definition. Human
votes and agent assessments are separate entity types. Agent assessments include
confidence, evidence, model, and correlation metadata.

### Approval

An approval request references the immutable hash of an `ActionSpecification`.
It records rule version, eligible approvers, responses, re-authentication level,
deadline, escalation state, and final outcome. A changed specification requires
a new approval request.

### Action

- owner principal and human sponsor where applicable
- state: proposed, ready, in-progress, blocked, verification, completed, failed,
  or cancelled
- immutable action specification and current attempt
- risk assessment and autonomy envelope
- dependencies, deadline, result, and verification criteria
- decision, approval, workflow-step, and agent-run references

### WorkflowDefinition and WorkflowRun

A workflow definition is a versioned directed graph of typed steps and edges.
The checklist is a projection of the same definition. Definitions may exist at
platform, workspace, or template scope.

A workflow run holds immutable definition version, step states, variables,
timers, assignments, approvals, branches, and an append-only transition log.
Manual changes are commands recorded with principal and justification.

### Artifact

Artifacts are versioned reusable knowledge with types such as runbook, known
issue, ownership record, escalation policy, saved query, decision record, and
integration recipe. Promotion records the incident evidence and accepting human.

### AgentDefinition and AgentRun

Agent definitions record provider/model, purpose, capabilities, tools,
integration permissions, owner, and policy constraints. Channel activation
creates a scoped agent membership.

Agent runs record task, sponsor, model, policy snapshot, capability grants,
sandbox specification, tool operations, communication envelopes, artifacts,
usage, status, and termination reason. Internal chain-of-thought is not stored as
product content.

### Integration and capability grant

Integrations are workspace-owned. Channels reference allowed, scoped views of an
integration. A capability grant gives an agent or user a time-bound permission
to perform a precise operation; it is not a copy of a long-lived credential.

## 3. Configuration precedence

An incident channel is independent. Effective configuration is resolved when it
is created:

1. Non-overridable platform safety limits
2. Workspace policy
3. Selected template version or workspace defaults
4. Explicit incident override, where permitted

Relationships to permanent channels do not participate in precedence. The
resolved configuration is snapshotted for audit and stable execution.

## 4. Command and event semantics

State changes use explicit commands and append immutable domain events. Examples:

- `CreateIncidentChannel`
- `CollectContext`
- `PostContextSnapshot`
- `ProposeFact`
- `AcceptDecision`
- `RequestApproval`
- `StartAgentRun`
- `StopAutonomousAction`
- `TransferIncidentOwnership`
- `ResolveIncident`
- `PromoteArtifact`

Events support audit, projections, asynchronous integrations, and rebuilding
read models. This does not require full event sourcing for every entity. Current
state remains in transactional tables; security- and workflow-relevant changes
also append an outbox-backed audit event.

## 5. Permission checks

Every read and command evaluates:

- workspace and region
- caller identity and membership
- channel visibility
- object-level classification
- integration scope
- action capability and risk
- current workflow and approval state
- platform safety limits

Retrieved data is authorised twice: once for source retrieval and again before
publication to the channel audience.

## 6. Invariants

- A channel belongs to exactly one workspace.
- Only incident channels have an `IncidentState`.
- Every incident has exactly one current owner, though ownership may transfer.
- Accepted decisions and completed votes are immutable.
- Approval applies only to the exact action-specification hash.
- No autonomous action runs without a valid policy decision and human sponsor.
- Stopping an autonomous action is available to every editor.
- Restarting requires an authorised approver.
- Original authorship and prior revisions are never overwritten.
- Context reposting cannot broaden access implicitly.
- A template or policy change cannot silently modify a running incident.
- No external content is interpreted as an instruction source.
