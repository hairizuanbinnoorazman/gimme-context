# MVP delivery plan

## 1. Delivery principle

Build one complete incident path before broadening integrations or agent
behaviour. Each increment must leave human coordination functional even when its
AI feature is disabled.

## 2. Pilot scope

- One pilot organisation
- A small number of permanent channels
- One GKE environment and one regional data plane
- Prometheus, Alertmanager, Loki, GitHub, and OIDC integrations
- One or two versioned incident templates
- One primary agent with optional specialist agents
- Local or staging agent execution only
- No external participants or production computer use

## 3. Milestones

### M0: Contracts and foundations

- Confirm terminology and block schemas.
- Define OpenAPI conventions and error model.
- Establish Go and Elm builds, container images, and GKE deployment.
- Implement OIDC login with Keycloak and Dex test configurations.
- Create workspace isolation and cross-tenant security tests.
- Implement PostgreSQL migrations, transactional outbox, and append-only audit.

Exit condition: two isolated workspaces can authenticate and perform audited
operations without cross-tenant access.

### M1: Human incident coordination

Status: complete for the M1 functional slice. The acceptance coverage exercises
manual incident creation, discussion, structured coordination, resolution, and
closure without AI or external integrations. Durable PostgreSQL storage and
verified OIDC identity remain M0 production-readiness dependencies and are not
silently treated as M1 implementation.

- Permanent and incident channels
- Versioned incident templates
- Membership, incident owner, and ownership transfer
- Posts, typed blocks, block replies, revisions, and chronological feed
- Incident header, scope, severity, lifecycle, summaries, and closure checklist
- Facts, decisions, actions, basic polls, and approvals
- Responsive Elm interface with initial Material Design components

Exit condition: a team can run and close a manual incident without AI or
external integrations.

Implemented acceptance surface:

- Peer permanent and incident channel APIs and navigation
- Immutable incident-template snapshots and selectable template versions
- Incident roles, membership, explicit ownership transfer, and immediate revocation
- Typed posts, block/post replies, author revisions, append-only revision history,
  and chronological feeds
- Severity, scope, lifecycle gates, verified summary, and closure checklist
- Evidence-aware facts and decisions, owned actions, polls, and hash-bound approvals
- API-backed responsive Elm interface with trusted controls and Material assets
- Backend lifecycle/invariant tests, an HTTP manual-closure acceptance test, Elm
  lifecycle tests, and responsive/accessibility interface contract tests

### M2: Automatic operational context

Status: complete for the M2 functional slice. Integration transport, recipes,
collection provenance, failures, refresh lineage, search visibility, and the
alert-to-context acceptance path are implemented against the in-memory contract
store. Durable job execution and encrypted integration credentials remain M0
production-readiness dependencies.

- Alertmanager incident creation and deduplication
- Prometheus metrics snapshots
- Loki log excerpts
- Retrieval collection posts, provenance, failure reporting, and refresh
- Context recipe configuration and simulation
- Platform search and similar-incident proposals

Exit condition: an alert creates an incident and posts initial metrics and logs
within the target two minutes under pilot load.

Implemented acceptance surface:

- Fingerprint/group-key Alertmanager deduplication and routed incident ownership
- Read-only Prometheus range snapshots and bounded Loki excerpts with source links
- Immutable collection posts with query windows, retrieval time, freshness,
  completeness, transformation/redaction fields, retries, and failure guidance
- Versioned context recipes, variable validation, side-effect-free simulation,
  manual collection, and provenance-linked refresh
- Workspace search and similar-incident proposals filtered by active membership
- Elm controls for collection, status/failure inspection, refresh, and proposals
- Backend alert-to-context, retry/failure, refresh, and visibility acceptance tests

### M3: AI-assisted synthesis

Status: complete for the M3 functional slice. The Vertex transport boundary,
agent controls, evidence-linked proposals, review flow, safety boundaries, and
failure isolation are implemented against the in-memory contract store. Durable
run execution, workload-identity token acquisition, and encrypted persistence
remain M0 production-readiness dependencies.

- Vertex AI model gateway and classification rules
- Agent identity, activation, capability grants, and run records
- Proposed summaries, facts, decisions, related incidents, and visualisations
- Evidence links and human accept/reject controls
- Prompt-injection boundaries and output redaction
- Agent collaboration envelopes

Exit condition: every material AI claim is traceable to visible evidence, and AI
failure does not block human incident work.

Implemented acceptance surface:

- Vertex AI structured-generation gateway with recorded provider/model and usage
- Approved agent definitions, owner activation, scoped memberships, and expiring
  capability grants independent of model output
- Persistent success/failure run records and explicit termination reasons
- Proposed summaries, facts, decisions, related incidents, and visualisations,
  each constrained to evidence visible in its run
- Human accept/reject controls that promote accepted summaries, facts, and
  decisions without silently changing source posts
- Untrusted-evidence delimiters, server-owned system instructions, structured
  output allowlists, and secret-like output redaction
- Evidence- and artifact-aware agent collaboration envelopes
- Acceptance tests for capability denial, injection boundaries, evidence
  traceability, redaction, review, failure recording, and manual continuity

### M4: Workflow and controlled autonomy

Status: complete for the M4 functional slice. Immutable workflow versions,
side-effect-free simulation, mixed-step runs, audited operator commands, and
the controlled-autonomy safety path are implemented against the in-memory
contract store. Durable timer dispatch and PostgreSQL-backed workflow state
remain M0 production-readiness dependencies.

- Versioned workflow definition and flow/checklist projections
- Human, agent, condition, timer, parallel, and approval steps
- Simulation, pause, stop, retry, and explicit migration
- Risk classification and autonomy envelopes
- Medium-risk countdown with platform minimum
- Universal editor stop and authorised restart

Exit condition: a mixed human/agent workflow executes with complete audit and
cannot exceed its tested permissions.

Implemented acceptance surface:

- Immutable directed workflow versions with human, agent, condition, timer,
  parallel, and approval steps plus flow/checklist projections
- Side-effect-free simulation, definition-version pinning, explicit paused-run
  migration, pause/resume/stop, step failure/retry, skip, and completion
- Low, medium, high, and prohibited risk classification with validated autonomy
  envelopes and approval-gated execution
- Platform-minimum medium-risk countdowns that cannot be bypassed by callers
- Universal autonomous-step stop for incident editors and restart restricted to
  named authorised approvers, always returning through a fresh policy check
- Append-only per-run transition records with actor, command, target step,
  justification, and time, plus workspace audit events
- Elm workflow controls with checklist/flow toggle, run controls, countdown
  status, editor stop, authorised restart, and retry controls
- Backend mixed-run, policy, migration, simulation, safety, and HTTP acceptance
  tests plus a successful optimised Elm build

### M5: Sandboxed investigation and GitHub remediation

Status: complete for the M5 functional slice. Disposable sandbox and GitHub
transports, investigation state, evidence integrity, staging egress controls,
and protected pull-request creation are implemented against the in-memory
contract store. Production sandbox provisioning and GitHub App credential
acquisition remain deployment integration dependencies.

- Disposable execution environment
- Repository checkout and read-only investigation
- Diagnostic script and test execution
- Allowlisted staging browser use and evidence capture
- GitHub branch and pull-request creation
- Channel-configured merge policy integrated with GitHub protections
- Reproducible finding and verification blocks

Exit condition: an agent reproduces a known issue, proposes a verified patch, and
creates a traceable pull request without production access.

Implemented acceptance surface:

- Time-bounded disposable sandboxes with explicit destruction and read-only checkout
- Command allowlists for diagnostic/test execution and HTTPS origin allowlists for
  staging browser use, with production and arbitrary shell access denied
- Immutable SHA-256 finding, test, checkout, and browser evidence records
- Explicit transition to writable `agent/*` patch branches through GitHub
- Pull requests gated on both reproduction and verification evidence
- Effective merge policy that can only strengthen channel policy with live GitHub
  branch protections, never weaken required checks, reviews, or conversations
- Evidence-linked pull-request bodies, audit events, OpenAPI routes, Elm visibility
  and controls, and backend safety/known-issue acceptance coverage

### M6: Knowledge feedback and pilot evaluation

Status: complete for the M6 functional slice. Versioned knowledge, runbook
simulation, reviewed archival, workspace audit export, and baseline-comparative
pilot analytics are implemented against the in-memory contract store. Durable
artifact persistence and production analytics warehousing remain M0 deployment
dependencies.

- Versioned artifacts and promotion workflow
- Runbook editing and simulation in permanent channels
- AI-proposed artifact updates
- Reviewed incident archival
- Audit export and pilot analytics
- Time-to-context and time-to-first-accepted-decision comparison

Exit condition: closed incidents produce accepted reusable knowledge and the
pilot demonstrates measurable improvement over its prior process.

Implemented acceptance surface:

- Immutable reusable artifact versions for runbooks, known issues, ownership,
  escalation, saved queries, decisions, and integration recipes
- Incident- and evidence-linked human or AI proposals, with completed agent-run
  attribution and explicit human promotion into permanent channels
- Side-effect-free runbook simulation with recorded inputs, result, actor, and time
- Archival gated on reviewed state and accepted reusable knowledge
- Time-filtered, workspace-isolated audit export
- Configurable prior-process baselines and per-incident/aggregate comparison of
  time to initial context and first accepted decision
- Backend acceptance coverage for promotion gates, immutable versions, archival,
  audit isolation, and pilot improvement calculations

## 4. Testing strategy

### Backend

- Unit tests for policies and state transitions
- PostgreSQL integration tests for transactions, outbox, and tenant isolation
- Contract tests for Prometheus, Alertmanager, Loki, GitHub, OIDC, and Vertex AI
- Property tests for workflow transitions and approval/action hashes
- Failure-injection tests for retries, duplicate webhooks, timeouts, and partial
  external outages

### Frontend

- Elm unit and fuzz tests for update logic and decoders
- Accessibility and keyboard tests for every trusted interactive block
- Browser tests for incident creation, threaded discussion, approval, stop,
  closure, and workflow views
- Responsive tests for critical mobile-friendly actions

### Security

- Cross-tenant API and object-store tests
- Prompt-injection and tool-capability tests
- Sandbox escape and network-egress tests
- Secret leakage and artifact-redaction tests
- Approval replay and changed-action rejection tests
- GitHub permission and branch-protection tests

## 5. Pilot scenarios

1. Alert fires for elevated HTTP errors; context shows the relevant metric and
   logs; a human accepts the initial decision.
2. Recent deployment correlates with an error; agent checks the commit and
   proposes a rollback without executing it.
3. Agent reproduces a defect locally, patches it, verifies it in a browser, and
   creates a GitHub pull request with evidence.
4. Conflicting metrics create a disputed fact and a follow-up retrieval.
5. Vertex AI is unavailable while humans continue posting and approving manual
   actions.
6. An autonomous action is stopped by an editor and cannot restart without an
   authorised approval.
7. Closing an incident proposes a runbook update that an editor accepts into a
   permanent channel.

## 6. Deferred backlog

- Notification and paging policy
- External and temporary participants
- Graph-view product semantics
- Detailed retention and legal holds
- Physical-operations templates
- Dedicated mobile applications
- Third-party agent marketplace
- Bedrock and customer-hosted model providers
- Full Slack or Teams synchronisation
- Dedicated single-tenant or customer-hosted deployment

Deferred work must not be partially implemented through unreviewed exceptions to
the MVP permission or data model.
