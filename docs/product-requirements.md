# Product requirements

## 1. Product statement

Gimme Context is a coordination platform where humans and authorised AI agents
collect evidence, investigate incidents, make decisions, execute controlled
workflows, and retain reusable operational knowledge.

The product optimises for the smallest set of authoritative context required to
make the next correct decision. It does not optimise for message volume or for
collecting all available data.

## 2. Product objects

Permanent channels and incident channels are peer objects within a workspace.

- A **workspace** is an organisation and security boundary.
- A **permanent channel** is a long-lived area for discussion, operational
  knowledge, runbooks, saved queries, and incident templates.
- An **incident channel** is a temporary, independently configured case used to
  investigate and resolve an event.
- A **post** is an authored contribution or a collected context package.
- A **block** is a typed, collapsible unit within a post.
- A **reply** targets either a post or one of its blocks and may be nested.
- An **artifact** is reusable, versioned knowledge promoted from a discussion or
  incident.

Incident channels may reference permanent channels and other incidents. These
links provide context but do not confer ownership, permissions, or configuration.

## 3. Target user and initial use case

The initial user is a software operations or platform team running Kubernetes
and using Prometheus, Alertmanager, Loki, GitHub, and an OpenID Connect identity
provider.

The first complete workflow is:

1. Alertmanager or a human creates an incident channel.
2. A versioned incident template supplies context recipes, agents, workflow,
   risk rules, and approval rules.
3. The platform retrieves relevant metrics, logs, alerts, code changes, runbooks,
   and similar incidents.
4. AI produces a structured, evidence-linked assessment.
5. Humans and agents investigate using posts and block-level replies.
6. A decision is proposed, reviewed, and accepted.
7. An agent performs an approval-aware investigation or remediation in an
   isolated environment.
8. Results are verified and posted with reproducible evidence.
9. The owner resolves the incident through a checklist.
10. AI proposes reusable artifacts for human acceptance.

## 4. Functional requirements

### 4.1 Channel creation and templates

- Humans, alerts, AI detection, and conversion of an existing post may create an
  incident channel.
- AI creation is governed by configurable confidence and severity thresholds.
- Incidents do not require an owning permanent channel.
- Workspace administrators define versioned incident templates.
- Permanent channels may recommend templates and define alert-routing rules.
- Templates include context recipes, approved agents, workflows, risk policies,
  approval templates, participants, and inactivity rules.
- Active incidents retain their template snapshot. Later changes require an
  explicit migration.

### 4.2 Context collection

Channel administrators can pre-authorise read-only retrieval within defined
scopes. Retrieval and publication are separate permissions.

Context sources in the MVP are:

- Prometheus metrics
- Alertmanager alerts
- Loki logs
- GitHub repositories, commits, pull requests, and code snippets
- Published runbooks and saved queries
- Previous incidents visible to the current participants and agent

Each context item records its source, query, time range, retrieval time,
freshness, transformations, redaction state, and retrieving agent. Bulk
telemetry remains in its source system. The platform stores snapshots, excerpts,
and source links.

Retrieval failures are first-class results. They report the attempted operation,
failure category, partial output, retries, and required human action. Missing
data is never interpreted as evidence that no problem exists.

### 4.3 Posts, blocks, and views

Posts may contain these controlled block types:

- Markdown narrative
- Log or code excerpt
- Table
- Time-series or bar chart
- Timeline
- Dependency or evidence graph (semantics deferred)
- Image, screenshot, video, or attachment
- Diff
- Poll
- Approval form
- Checklist
- Decision, action, or status panel
- Agent execution result

Blocks are collapsible, and replies may target individual blocks. Logical reply
depth is not artificially limited. The interface provides collapsed branches,
breadcrumbs, focused branch views, and summaries.

Charts and other source-backed views are immutable snapshots. A request for a
different view produces a new linked post rather than rewriting the original.
Source-backed data is immutable; annotations and derived views are versioned.

Markdown and HTML use strict allowlists. Agent-provided scripts, forms, event
handlers, external resources, and unrestricted styles are not rendered.
Interactive controls are trusted platform components.

### 4.4 Feed and incident state

- Chronology is the canonical feed order.
- Optional filters and prioritised views do not change the official timeline.
- Editors may mark posts normal, important, pinned, required reading,
  superseded, or hidden from the default view.
- Retrieval jobs initially create one expandable collection post and split out
  findings only when discussion, permissions, actions, approvals, or size
  require it.
- The incident header shows status, severity, owner, scope, verified summary,
  active decisions, approvals, actions, blockers, automation, last meaningful
  update, and related incidents.
- AI maintains a proposed evidence-linked summary. Human acceptance makes it
  official. Templates may auto-accept low-risk summary sections.

### 4.5 Facts and decisions

Evidence may be unverified, corroborated, disputed, superseded, or invalidated.
Human participants with edit access may change these states. Original posts are
not silently edited by other users.

A reply, explicit form, AI extraction, completed approval, or poll may propose a
decision. Acceptance requires an explicit human action or configured approval
rule. Accepted decisions are immutable and may be superseded, retaining the
complete chain and rationale.

### 4.6 Polls and approvals

- Polls are advisory or binding.
- Polls specify eligible voters, quorum, deadline, and whether votes may change.
- Human votes and AI assessments are separate.
- AI assessments include confidence, evidence, and agent capability.
- Correlated agents must not be presented as independent consensus.
- Polls do not override required approvers, policy checks, or veto rules.
- Binding poll completion creates a versioned decision.
- Approval templates may require named people, roles, multiple groups, quorum,
  unanimity, vetoes, deadlines, or policy checks.
- Approval binds to an immutable action specification. Any material parameter
  change invalidates it.
- Re-authentication requirements are configurable by channel and risk.

### 4.7 Actions and workflows

Actions have an owner, status, due time, dependencies, related decision,
execution output, and verification criteria. An autonomous AI action also has a
named human sponsor who authorised its bounded autonomy.

Workflows have one structured definition with toggleable flow and checklist
views. Steps may be human tasks, agent tasks, approvals, conditions, timers, or
parallel branches. Users may pause, cancel, add, skip, reassign, or revisit
steps, subject to policy and audit.

Workflow modes may be mixed by step:

- Guided human execution
- Approval-gated agent execution
- Bounded autonomous execution

Workflows support simulation using sample or historical data without modifying
external systems or contacting real people. AI may propose new workflows, but a
channel administrator must review, simulate, and publish them.

### 4.8 Agent investigation and remediation

Agents may form hypotheses, inspect code, write diagnostic scripts and tests,
run applications, use a browser against local or staging environments, capture
evidence, create GitHub branches and pull requests, and merge where channel
policy explicitly permits it.

Execution occurs in disposable sandboxes with explicit repository, commit,
container image, network allowlist, secret scope, resource limits, writable
paths, runtime, and artifact retention. Production browser or code execution is
not part of the MVP.

An investigation result includes hypothesis, method, environment, commit,
before/after result, scripts or tests, output, visual evidence, limitations, and
remaining uncertainty. Investigation, code remediation, and operational changes
are separately authorised capabilities.

### 4.9 Agent identity and collaboration

- Workspace administrators approve agents and integrations.
- Channel administrators configure approved agents.
- Incident owners activate them.
- Each agent exposes its identity, purpose, provider/model, owner, capabilities,
  integrations, permissions, autonomy level, sponsor, and recent actions.
- Agents can discover and communicate with other agents.
- Internal reasoning is not published.
- Material findings are posted, and a minimal audit envelope records which
  agents communicated, for what task, what sources were shared, and what
  artifact or action resulted.
- Failed agents are not silently substituted.

### 4.10 Risk and autonomy

Actions are classified as low, medium, high, or prohibited according to data,
availability, security, financial, external, reversibility, and blast-radius
impact.

- Low-risk actions may proceed immediately within policy.
- Medium-risk actions may use a channel-configured countdown subject to a
  platform minimum.
- High-risk behaviour is channel-configurable within workspace and platform
  safety limits.
- Prohibited actions cannot execute.
- Any editor may stop an autonomous action; an authorised approver is required
  to restart it.

Autonomy operates inside a documented envelope of targets, duration, cost,
scope, credentials, and forbidden actions. Countdown posts show the change,
reason, expected impact, rollback, sponsor, and stop control. Execution pauses
when evidence conflicts, monitoring fails, severity rises, prerequisites fail,
the sponsor loses authority, or the envelope would be exceeded.

### 4.11 Participation and ownership

- Each incident has one transferable owner.
- Channel policy controls invitation, inherited workspace membership, automatic
  addition through ownership rules, and AI participant recommendations.
- Authors may edit their own posts with version history.
- Editors manage structured facts, decisions, actions, summaries, and post
  presentation states without rewriting another author’s content.
- Owners manage access and may reduce or revoke a participant’s access.
- Authorised users may take ownership during an incident; explicit handover is
  preferred.
- Selective subscriptions use role-based defaults and per-incident overrides.
- External participants are deferred.

### 4.12 Lifecycle

Default severities are `SEV-1` through `SEV-4` plus unclassified. Templates may
rename them. AI may set severity under template policy, and severity changes may
trigger workflows, retrieval, participant recommendations, and approval rules.

Default lifecycle:

`Open -> Investigating -> Mitigating -> Monitoring -> Resolved -> Reviewed -> Archived`

`Dormant` means inactive but not confirmed resolved. `Cancelled` represents a
false alarm, duplicate, or irrelevant case. Inactivity may prompt the owner and
move a low-severity incident to dormant or archived according to policy; it does
not imply resolution. New activity prompts reopening but never reopens an
incident automatically.

Resolution uses a template- and severity-specific checklist. Owners may force
close with a recorded reason and generated follow-up actions. High-severity
incidents require review before archival.

### 4.13 Permanent knowledge

Permanent channels support discussion, typed blocks, replies, polls, decisions,
agents, runbooks, saved queries, and other versioned artifacts. They do not use
incident severity or lifecycle.

A post or reply may create an independent incident channel with a backlink.
Incident findings become permanent artifacts only after human acceptance. AI may
propose artifact updates, but does not publish them autonomously. Permanent
channels separate ordinary discussion from curated artifacts and show related
active and historical incidents.

### 4.14 Search and related incidents

Platform search covers posts, replies, structured state, attachments, snapshots,
agent findings, and archived incidents within the caller's permissions.
Federated external search is a distinct, policy-gated operation.

AI may propose similar historical incidents using systems, symptoms, impact,
changes, causes, and mitigations. Results appear as a compact expandable
collection and show age, verification status, and permissions. Humans may reject
the proposed context.

## 5. Non-functional requirements

- Human posting, existing context, and approvals remain usable when AI or
  integrations are unavailable.
- Tenant isolation applies to relational data, objects, queues, caches, search,
  vector indexes, integrations, and agent execution.
- A workspace selects its Google Cloud region at creation.
- Model processing respects region and data classification.
- Retrieved content is untrusted evidence and cannot override platform,
  workspace, channel, workflow, or authorised human instructions.
- Audit history is append-only and exportable.
- Credentials are short-lived where possible and never rendered in posts or
  agent artifacts.
- Suspected secret exposure pauses execution.

## 6. MVP exclusions and deferred decisions

- Notification and paging behaviour
- External participants
- General graph-view semantics
- Detailed retention, deletion, and legal holds
- Physical-operations workflows
- Dedicated mobile applications; the web UI remains responsive
- Third-party agent marketplace
- Customer-hosted deployment
- Non-Google model hosting
- Full Slack or Teams synchronisation
- Arbitrary agent-generated UI code

## 7. Success criteria

- Alert-created incident channel within 30 seconds.
- Initial operational context available within two minutes.
- Every material AI finding links to reproducible evidence.
- Current facts, owner, decisions, and actions are understandable without reading
  the complete feed.
- Agent investigation and remediation are bounded and auditable.
- At least one incident class can be reproduced in a sandbox and result in a
  verified GitHub pull request.
- Closure produces a useful summary and proposed reusable artifact.
- Median time from incident creation to first accepted decision improves against
  the pilot team's existing process.
