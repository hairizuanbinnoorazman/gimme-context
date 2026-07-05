# Technical architecture

## 1. Architecture goals

- Preserve human coordination during AI and integration outages.
- Keep all consequential state changes auditable.
- Isolate tenants, agents, credentials, and execution environments.
- Add integrations and models behind stable internal contracts.
- Support asynchronous context gathering without turning the feed into a job log.
- Start as a modular system that can be split only when operational evidence
  justifies it.

## 2. Initial deployment

The application is deployed as containers to a regional Google Kubernetes
Engine cluster. Managed Google Cloud services hold durable state; databases and
object stores are not run inside Kubernetes.

Recommended initial components:

- GKE regional cluster for application workloads
- Cloud SQL for PostgreSQL for transactional state
- Cloud Storage for attachments, immutable snapshots, and execution artifacts
- Pub/Sub for durable asynchronous work
- Memorystore only when measured caching or coordination needs justify it
- Secret Manager and Workload Identity for credentials
- Vertex AI for platform-hosted models
- Cloud Logging, Monitoring, Trace, and Error Reporting for platform telemetry
- Artifact Registry and Cloud Build or GitHub Actions for container delivery

Workspace data is pinned to a selected region. Service configuration must prevent
implicit cross-region model or data processing.

## 3. Backend structure

The backend is written in Go as a modular monolith for the MVP. Modules own their
schema and interfaces but deploy together initially:

```text
identity       OIDC sessions, principals, workspace roles
channels       channels, memberships, posts, blocks, replies
incidents      scope, severity, lifecycle, summaries, closure
decisions      facts, polls, approvals, decisions, actions
workflows      definitions, runs, timers, transitions, simulation
context        retrieval recipes, snapshots, provenance, visual data
agents         definitions, runs, tools, capabilities, collaboration
integrations   Prometheus, Alertmanager, Loki, GitHub
artifacts      permanent knowledge, versions, promotion
search         authorised platform search and related incidents
audit          append-only security and domain audit records
```

Use ordinary Go packages with explicit dependencies rather than a framework that
hides transactions or request flow. HTTP APIs should be versioned and described
with OpenAPI. Generate Elm decoders/encoders where practical, but retain contract
tests because generated types do not validate behavioural compatibility.

### Transaction and async pattern

Commands that change durable state run in PostgreSQL transactions. The same
transaction writes an outbox record. Workers publish or process outbox records,
making context jobs, summaries, notifications, and integrations retryable.

Workers must be idempotent. Every external operation has an idempotency key and
attempt record. Webhook ingestion verifies signatures and deduplicates provider
delivery identifiers.

### API styles

- JSON HTTP API for commands, queries, and pagination
- Server-Sent Events for incident-feed and workflow updates in the MVP
- Signed, short-lived object upload/download URLs
- Provider webhooks for Alertmanager and GitHub events

SSE is simpler than bidirectional WebSockets for the initial mostly server-driven
update model. Add WebSockets only if collaborative features demonstrate a need.

## 4. Frontend

The frontend is an Elm single-page application using
`aforemny/material-components-web-elm` 9.1.0 as its initial Material component
library. It includes:

- channel and incident navigation
- chronological feed with typed, collapsible blocks
- block-level threaded replies and focused branch view
- incident state panel
- workflow flow/checklist toggle
- trusted forms for polls, approvals, actions, and risk countdowns
- responsive layouts for viewing, replying, approving, and stopping actions

Elm owns application state, routing, validation, and rendering. A project-owned
Elm view layer wraps the Material package so feature modules do not couple to its
API. The package's version-matched JavaScript and CSS assets are included in the
frontend build. Additional JavaScript interop is restricted to small audited
ports or custom elements for capabilities that are impractical in Elm, such as
advanced code editors or graph rendering.

React's `@mui/material` is not used. The accepted package, maintenance risk, and
future replacement boundary are documented in
[ADR-0001](adr/0001-elm-material-design.md).

## 5. Storage and search

PostgreSQL stores transactional entities, JSON payloads for versioned block
schemas, workflow state, and audit indexes. JSON does not replace relational
columns for tenancy, permissions, status, or other frequently constrained data.

Cloud Storage contains larger immutable artifacts. Object names are tenant
scoped, and access is mediated through the backend.

Start search with PostgreSQL full-text search and explicit permission filters.
Introduce a separate search or vector system only after corpus size, related-
incident quality, or latency demonstrates the requirement. Vector retrieval must
retain tenant, channel, classification, and artifact-state filters.

## 6. Integration adapters

All adapters implement scoped operations and return common provenance metadata.

### Prometheus

- read-only instant and range queries
- label, tenant, duration, timeout, and sample-count limits
- query text and endpoint recorded
- immutable chart/table snapshot output

### Alertmanager

- signed or network-restricted webhook ingestion
- routing rule maps alerts to incident templates
- deduplication and grouping preserve original alert identity

### Loki

- read-only range queries
- tenant, label, time, line-count, byte, and timeout limits
- redaction before publication
- bulk log data remains in Loki

### GitHub

- GitHub App installation rather than personal tokens
- repository and operation-scoped permissions
- webhook verification and delivery deduplication
- read operations for repository investigation
- separately authorised branch, pull-request, review-request, and merge actions
- incident, decision, sponsor, and agent-run references in generated pull requests

## 7. Agent orchestration and sandboxing

The agent control plane runs in the application backend. Untrusted investigation
and remediation execute in disposable Kubernetes jobs or a stronger isolated
runtime selected during implementation threat modelling.

Each run receives:

- immutable task and policy snapshot
- repository and commit
- approved tools and model
- network allowlist enforced outside the workload
- short-lived capability credentials
- CPU, memory, storage, time, and cost budgets
- artifact output location
- stop signal and termination deadline

Do not rely solely on prompts for safety. Admission validation, Kubernetes
policies, network controls, capability services, repository protections, and
action-specific approval checks enforce boundaries.

Browser sessions target only allowlisted local or staging URLs. Interaction
events and evidence are recorded while credentials and sensitive fields are
redacted. Investigation progress is represented by one live status block and a
final structured result, not one post per tool call.

## 8. AI architecture

Vertex AI is the initial model provider. Internal interfaces separate:

- model selection
- structured generation
- tool requests
- usage accounting
- safety and classification checks
- retry and availability handling

Model access is selected by workspace region, data classification, agent
capability, and task. Prompts and outputs are configured not to train shared
models. Provider and exact model identifiers are recorded on every run.

Retrieved posts, logs, code, email-like text, and web content are delimited and
labelled as untrusted evidence. They cannot introduce tools, permissions, or
instructions. Tool execution requires a server-issued capability independent of
model output.

## 9. Identity and authorisation

Authentication uses OpenID Connect, initially tested against Keycloak and Dex.
OAuth 2.0 alone is not treated as an authentication protocol.

Authorisation is enforced in the backend and combines:

- workspace role
- channel membership
- content classification
- integration and source scope
- capability grant
- workflow/approval state
- risk policy
- non-overridable platform policy

PostgreSQL tenant keys and optional row-level security provide defence in depth,
not a substitute for application checks. Automated tests attempt cross-tenant
access for every repository and API class.

High-risk actions may require recent authentication, a passkey, or multiple
approvers according to channel policy.

## 10. Secrets and data protection

- Workloads use Google Workload Identity.
- Long-lived secrets remain in Secret Manager.
- Agents receive short-lived, operation-scoped credentials where supported.
- Secrets are excluded from prompts and structured posts.
- Inputs and outputs pass through detection and redaction before publication.
- Suspected leakage pauses the run and emits a security audit event.
- Data and backups are encrypted with Google-managed keys initially; customer-
  managed keys may be added when required.

## 11. Audit and observability

Audit records are append-only and include actor, tenant, channel, command,
target, policy version, result, timestamp, and correlation identifiers. Sensitive
payloads are referenced or hashed rather than copied into audit rows.

Trace identifiers connect API commands, workflow transitions, retrieval jobs,
agent runs, model calls, GitHub activity, and resulting posts. Product telemetry
must distinguish useful time-to-context and time-to-decision from raw activity.

## 12. Availability and degradation

- Human posting and reads depend only on core application storage.
- AI and integration failures surface as explicit status blocks.
- Retrieval work is queued and retryable.
- Existing approvals remain inspectable if Vertex AI is unavailable.
- No missing model response automatically authorises or completes an action.
- Regional backups and recovery procedures are tested before production pilots.

## 13. Initial repository layout

```text
cmd/
  api/
  worker/
internal/
  identity/
  channels/
  incidents/
  decisions/
  workflows/
  context/
  agents/
  integrations/
  artifacts/
  search/
  audit/
web/
  elm.json
  src/
api/
  openapi.yaml
deploy/
  helm/
  environments/
docs/
```

The API and worker may initially share the same Go modules and container build,
while using separate entrypoints and Kubernetes workloads.
