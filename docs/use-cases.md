# Platform use cases

## 1. Purpose and scope

This document describes the use cases Gimme Context is expected to support. It
is a product-scope catalog: it covers the intended ways people, agents, and
integrations interact with the platform, including degraded and exceptional
paths. It does not imply that every use case is available in the current
in-memory implementation.

Gimme Context is an incident coordination and operational-knowledge platform.
It is designed to gather the smallest useful set of authoritative context,
support auditable decisions and controlled action, and turn reviewed incident
learning into reusable knowledge. It is not intended to be a general-purpose
chat system, unrestricted automation engine, telemetry store, or replacement
for source systems such as Prometheus, Loki, GitHub, or an identity provider.

## 2. Actors

- **Workspace administrator** configures identity, integrations, agents,
  workspace policy, templates, data handling, and platform-wide roles.
- **Channel administrator** configures a permanent channel or an incident's
  allowed agents, workflows, integrations, and approval rules.
- **Incident owner** is accountable for one incident, manages access and
  lifecycle, and completes or overrides closure gates.
- **Editor** maintains structured incident state and can stop autonomous work.
- **Participant** contributes posts, replies, evidence, votes, and assigned work.
- **Viewer** reads information permitted by membership and classification.
- **Approver** authorises a specific action under an approval policy.
- **Human sponsor** authorises and remains accountable for bounded autonomous
  agent work.
- **AI agent** synthesises context, proposes structured conclusions, investigates,
  or acts within explicit capabilities and policy.
- **Source integration** supplies alerts, metrics, logs, code, and other evidence.
- **External automation** creates or updates incidents through an authorised API
  or webhook.

One person may hold several roles. Every operation is evaluated against workspace
membership, channel role, object classification, integration scope, and current
policy; an actor name in this catalog never grants permission by itself.

## 3. Workspace and administration

### UC-ADM-01 — Create and configure a workspace

A platform operator or authorised administrator creates an organisational and
security boundary, selects its region, configures identity, and applies data,
model, risk, and platform-limit policies. The result is an isolated workspace
whose content, indexes, jobs, integrations, and agent execution cannot be read
from another workspace.

### UC-ADM-02 — Manage membership and roles

An administrator adds, removes, or changes workspace roles. Channel membership
may then be direct, inherited from a workspace rule, supplied by a template, or
accepted from a recommendation. Revocation takes effect immediately for future
reads and commands and is recorded in audit history.

### UC-ADM-03 — Configure identity and re-authentication

An administrator connects an OpenID Connect provider, maps verified claims to
workspace identity, and defines when sensitive approvals or actions require
fresh authentication. Development principal headers are not a production use
case.

### UC-ADM-04 — Register and scope integrations

An administrator registers Prometheus, Alertmanager, Loki, GitHub, or another
supported integration, using short-lived credentials where possible. A channel
administrator exposes only a scoped view to a channel. Retrieval permission and
permission to publish retrieved content are evaluated separately.

### UC-ADM-05 — Define and approve agents

An administrator records an agent's provider, model, purpose, owner,
capabilities, tools, integrations, autonomy limits, and policy constraints. A
channel administrator may allow the agent, but an incident owner must activate
it before incident work. Agent failure must remain visible and must not cause
silent substitution with another agent.

### UC-ADM-06 — Create and version incident templates

An administrator defines context recipes, agents, workflows, risk and approval
rules, default participants, severity naming, closure checks, and inactivity
rules. New incidents snapshot a specific template version. Changes do not alter
an active incident unless an authorised user explicitly migrates it.

### UC-ADM-07 — Configure risk and autonomy policy

Administrators classify actions as low, medium, high, or prohibited based on
data, availability, security, financial, external, reversibility, and blast
radius impact. Workspace rules can only narrow platform safety limits; channel
rules can only narrow the effective workspace envelope.

### UC-ADM-08 — Export and inspect audit history

An authorised user filters and exports append-only audit events for a workspace
and time range. The export supports incident review, compliance, and investigation
without exposing another workspace's events or internal agent reasoning.

## 4. Permanent channels and reusable knowledge

### UC-KNW-01 — Create a permanent operational channel

An authorised user creates a long-lived channel for a service, team, system, or
operational topic. The channel can contain discussion and curated artifacts but
has no incident severity or incident lifecycle.

### UC-KNW-02 — Discuss operational work

Members create typed posts, reply to a post or individual block, navigate nested
branches, revise their own posts with append-only history, and mark content as
important, pinned, required reading, superseded, or hidden from the default view.

### UC-KNW-03 — Maintain operational artifacts

Editors create and version runbooks, known issues, ownership records, escalation
policies, saved queries, decision records, and integration recipes. Ordinary
discussion remains distinct from reviewed, curated artifacts.

### UC-KNW-04 — Simulate a runbook or workflow

An editor runs a published definition with sample or historical inputs. The
simulation records inputs and results but does not modify external systems,
contact real people, or consume production action permissions.

### UC-KNW-05 — Convert discussion into an incident

A member turns a post or reply into an independent incident channel. The new
incident retains a backlink and copied or referenced context with provenance;
the permanent channel does not become its owner or permission source.

### UC-KNW-06 — Relate permanent knowledge and incidents

Members view active and historical incidents related to a permanent channel and
follow references from incidents to runbooks, saved queries, or known issues.
Relationships provide context only and do not implicitly broaden access.

## 5. Incident creation, routing, and access

### UC-INC-01 — Create an incident manually

An authorised human supplies a title, scope, initial severity, owner, and template.
The platform snapshots effective configuration and opens a chronological feed.
An incident does not require a parent permanent channel.

### UC-INC-02 — Create or update an incident from an alert

Alertmanager or another authorised detector routes an event using workspace and
channel rules. A new alert group creates an incident; a matching fingerprint or
group key updates the existing incident without duplicating it.

### UC-INC-03 — Create an incident through AI detection

An approved detector proposes or creates an incident only when configured
confidence and severity thresholds permit it. The creation trigger, model, rule,
and supporting evidence are recorded. Humans can cancel a false alarm.

### UC-INC-04 — Select, override, or migrate configuration

An authorised user selects a template at creation, applies permitted incident
overrides, or explicitly migrates a running incident to a newer version. The
platform validates the precedence of platform limits, workspace policy,
template defaults, and incident overrides and retains the prior snapshot.

### UC-INC-05 — Manage participants and subscriptions

The owner invites participants, accepts or rejects AI participant recommendations,
changes channel roles, revokes access, and configures role-based or per-incident
subscriptions. External participants are not currently expected.

### UC-INC-06 — Transfer or take ownership

The current owner explicitly hands the incident to another authorised user. If
the owner is unavailable, a suitably authorised user can take ownership. The
incident always has exactly one current owner and retains the transfer history.

### UC-INC-07 — Relate, deduplicate, or split incidents

An editor links incidents as related, duplicate, caused-by, follow-up-to,
parent-of, or recurrence-of. A duplicate may redirect readers to a canonical
incident while preserving its own record and access boundary.

## 6. Context collection and discovery

### UC-CTX-01 — Collect initial context automatically

Incident creation starts the template's context recipe. The platform retrieves
bounded snapshots of relevant alerts, metrics, logs, code changes, runbooks,
saved queries, and visible historical incidents, then posts one expandable
collection result with provenance.

### UC-CTX-02 — Collect context on demand

An authorised human or agent runs a recipe or scoped query for a stated time
range and incident variables. The result records source, normalised query,
retrieval time, freshness, completeness, transformations, redaction, and the
retrieving principal.

### UC-CTX-03 — Preview or simulate a context recipe

An administrator validates variables and previews the operations a recipe would
perform without publishing results or causing external side effects. This is
used before publishing a recipe or attaching it to a template.

### UC-CTX-04 — Refresh stale context

A member requests a new snapshot when evidence is stale or a different time
window is needed. The platform creates a new linked post and preserves the
original immutable snapshot and refresh lineage.

### UC-CTX-05 — Handle retrieval failure or partial data

Timeouts, permission failures, malformed responses, and partial output produce
visible failure results with the attempted operation, failure category, retries,
partial data, and required human action. Missing data is never presented as
proof that no issue exists.

### UC-CTX-06 — Search platform knowledge

A user searches posts, replies, structured state, attachments, snapshots, agent
findings, artifacts, and archived incidents. Results are filtered before return
according to the caller's current permissions and classification access.

### UC-CTX-07 — Find similar incidents

The platform or an agent proposes historical incidents based on systems,
symptoms, impact, changes, causes, and mitigations. Results show age,
verification state, and only content visible to the current participants. A
human may reject an irrelevant proposal.

### UC-CTX-08 — Repost or derive a view of evidence

A member shares authorised context in another channel or asks for a table,
chart, timeline, excerpt, or annotation. The platform creates a new versioned
snapshot with provenance and rechecks destination visibility instead of
mutating or live-copying the original.

## 7. Human coordination and structured incident state

### UC-COL-01 — Post and discuss evidence

Humans and authorised agents publish controlled blocks such as narrative, logs,
code, tables, charts, timelines, diffs, attachments, checklists, status panels,
or execution results. Replies may target a whole post or a block and may nest
without an artificial logical depth limit.

### UC-COL-02 — Navigate a large incident

A participant uses collapsed branches, breadcrumbs, focused branch views,
summaries, filters, and prioritised views. Chronology remains the canonical
official feed regardless of the selected view.

### UC-COL-03 — Maintain the incident header and summary

Editors keep severity, scope, owner, lifecycle, verified summary, active
decisions, approvals, actions, blockers, automation state, and related incidents
current. AI may propose evidence-linked summary sections; human acceptance, or
an explicit low-risk auto-accept rule, makes them official.

### UC-COL-04 — Record and review facts

A participant or agent proposes a fact linked to evidence. Editors mark it
unverified, corroborated, disputed, superseded, or invalidated and retain both
supporting and opposing references. Conflicting evidence triggers investigation,
not silent replacement.

### UC-COL-05 — Propose and accept a decision

A reply, form, poll, completed approval, or agent output proposes a decision
with rationale, alternatives, and evidence. An authorised human or configured
approval rule accepts or rejects it. Accepted decisions are immutable and can
only be superseded by a linked successor.

### UC-COL-06 — Create and track actions

An editor creates an action with an owner, due time, dependencies, related
decision, immutable specification, verification criteria, and risk. Progress,
attempts, output, blocking conditions, and final verification remain visible.
Autonomous actions also identify their human sponsor.

### UC-COL-07 — Run an advisory or binding poll

An authorised user defines options, eligible voters, quorum, deadline, and
whether votes may change. Human votes remain separate from evidence-linked AI
assessments. A completed binding poll creates a versioned decision but cannot
override required approvers, policy checks, or vetoes.

### UC-COL-08 — Request and complete an approval

An action requester submits an immutable action specification under a versioned
rule. Named users, roles, groups, quorum, unanimity, vetoes, deadlines, policy
checks, and re-authentication may apply. A material specification change
invalidates prior approval and requires a new request.

## 8. AI-assisted synthesis and collaboration

### UC-AI-01 — Activate an agent for an incident

The owner activates a workspace-approved and channel-allowed agent. The platform
creates a scoped membership and time-bound capability grants. Participants can
inspect its identity, purpose, model, owner, permissions, autonomy, sponsor, and
recent activity.

### UC-AI-02 — Generate an evidence-linked assessment

An agent analyses only authorised context and proposes a summary, fact,
decision, related incident, or visualisation. Every material claim links to
evidence visible in that run. A human accepts or rejects the proposal without
changing the source evidence.

### UC-AI-03 — Delegate between approved agents

An agent asks an approved specialist agent to perform a bounded task and shares
only authorised sources. The platform records a minimal communication envelope:
participants, task, shared sources, and resulting artifact or action. Internal
reasoning is not published or stored as product content.

### UC-AI-04 — Resist untrusted instructions and redact output

Retrieved pages, logs, code, and posts are treated as evidence rather than
instructions. Agent output is schema-validated and secret-like content is
redacted. Suspected credential exposure pauses execution and produces a visible
failure requiring human review.

### UC-AI-05 — Continue when AI is unavailable

If the model gateway times out, rejects a request, or is disabled, the platform
records the failed run and termination reason. Humans can continue posting,
collecting existing context, deciding, approving, acting, and closing the
incident.

## 9. Workflows, approvals, and controlled autonomy

### UC-WFL-01 — Define and publish a workflow

An administrator creates a versioned directed workflow containing human tasks,
agent tasks, approvals, conditions, timers, and parallel branches. A channel
administrator reviews and simulates an AI-proposed workflow before publishing
it. Flow and checklist views are projections of the same definition.

### UC-WFL-02 — Run a guided human workflow

The incident starts a workflow pinned to one definition version. People claim
and complete assigned steps, supply evidence, and satisfy verification checks.
The platform evaluates conditions and dependencies and records every transition.

### UC-WFL-03 — Run a mixed human and agent workflow

Individual steps operate in guided, approval-gated, or bounded-autonomous mode.
The platform activates only the capabilities needed for the current step and
does not allow an agent to infer permission from workflow text.

### UC-WFL-04 — Operate a running workflow

An authorised user pauses, resumes, cancels, retries, skips, reassigns, adds, or
revisits a step with a justification where policy requires it. Migration to a
new definition is explicit and preserves the old version and transition log.

### UC-WFL-05 — Execute a medium-risk countdown action

After policy and approval checks, a countdown post displays the change, reason,
expected impact, rollback, sponsor, and stop control. The countdown respects the
platform minimum and executes only if its evidence, permissions, and envelope
remain valid.

### UC-WFL-06 — Stop and restart autonomous work

Any incident editor stops an autonomous action immediately. Restart requires an
authorised approver and a fresh policy check; it cannot merely resume under a
stale approval or capability grant.

### UC-WFL-07 — Pause on unsafe or changed conditions

Execution pauses when evidence conflicts, monitoring fails, severity rises,
prerequisites fail, the sponsor loses authority, a secret may be exposed, or
the action would exceed targets, time, cost, credentials, network, or blast
radius. Prohibited actions never execute.

## 10. Sandboxed investigation and remediation

### UC-INV-01 — Investigate code in a disposable sandbox

An approved agent checks out an exact repository commit read-only, forms a
hypothesis, runs allowlisted diagnostic commands or tests, and records the
environment, method, output, limitations, and immutable evidence. Runtime,
resources, network, secrets, writable paths, and retention are bounded.

### UC-INV-02 — Reproduce a defect

An agent builds or runs the application locally or in an allowed staging
environment, reproduces the symptom, and records before-state evidence. Failed
reproduction is a valid finding and retains uncertainty rather than being
reported as absence of a defect.

### UC-INV-03 — Use a browser against an allowed environment

An agent uses browser automation only against allowlisted local or staging
origins, captures visual or other reproducible evidence, and cannot browse
production or arbitrary network destinations in the MVP.

### UC-INV-04 — Develop and verify a patch

After separate write authorisation, an agent creates an isolated `agent/*`
branch, changes code, reruns reproduction and verification, and records the
diff and before/after evidence. Investigation permission alone does not grant
code-remediation permission.

### UC-INV-05 — Create a traceable pull request

When reproduction and verification evidence exist, an agent creates a GitHub
pull request that links the incident, hypothesis, method, tests, and evidence.
Credentials and repository scope are limited to the approved operation.

### UC-INV-06 — Merge under effective protection policy

An agent or human merges only when channel policy explicitly permits it and all
live GitHub branch protections, required checks, reviews, and conversations are
satisfied. Platform policy may strengthen source-system protection but never
weaken it. Operational production changes remain separately authorised.

### UC-INV-07 — Destroy and audit the sandbox

At completion, timeout, cancellation, or policy violation, the platform destroys
the disposable environment and retains only permitted evidence and artifacts.
The run records the commit, image, grants, tool operations, usage, status, and
termination reason.

## 11. Incident lifecycle, closure, and learning

### UC-LIF-01 — Progress through the incident lifecycle

An editor moves an incident through Open, Investigating, Mitigating, Monitoring,
Resolved, Reviewed, and Archived as applicable. Severity changes can trigger new
retrieval, workflows, participant recommendations, or approval rules.

### UC-LIF-02 — Mark an incident dormant, cancelled, or reopened

Inactivity can prompt the owner and, under policy, make a low-severity incident
dormant or archived; it never implies resolution. False alarms, duplicates, and
irrelevant cases may be cancelled with a reason. New activity prompts a human
to reopen an incident but never reopens it automatically.

### UC-LIF-03 — Resolve through a closure checklist

The owner satisfies the template- and severity-specific checks for mitigation,
verification, actions, summary, and required approvals. The platform blocks
ordinary resolution while mandatory gates are incomplete.

### UC-LIF-04 — Force close with accountability

Where policy permits, the owner force closes an incident with an explicit
reason. The platform records the override and creates follow-up actions for
unresolved mandatory work rather than discarding it.

### UC-LIF-05 — Review a high-severity incident

Reviewers validate the timeline, facts, decisions, actions, contributing causes,
verification, and final summary. High-severity incidents cannot archive until
review requirements are complete.

### UC-LIF-06 — Promote incident learning

A human or agent proposes a new artifact or version linked to incident evidence.
An authorised human reviews and promotes it into a permanent channel. AI cannot
publish permanent knowledge autonomously, and the accepted version preserves
its provenance.

### UC-LIF-07 — Archive an incident

After resolution and any required review and knowledge promotion, an authorised
user archives the incident. Its feed, structured state, evidence, decisions,
agent runs, workflow transitions, and audit history remain searchable according
to retention and access policy.

### UC-LIF-08 — Evaluate operational improvement

Pilot owners define prior-process baselines and compare time to initial context
and time to first accepted decision for individual incidents and aggregates.
Metrics are workspace-isolated and should not reward message volume or
unverified automated output.

## 12. Cross-cutting failure and security cases

The platform must also support these non-happy paths across the catalog:

- An integration, model, worker, or source system is unavailable while human
  coordination remains usable.
- A webhook is retried or delivered out of order without creating duplicate
  incidents or duplicate actions.
- A user loses membership, an approver loses authority, or a capability expires
  during execution; affected work pauses or fails closed.
- Retrieved content has a higher classification than the destination channel;
  publication is denied or redacted even if retrieval was allowed.
- A query produces partial, stale, malformed, or contradictory data; its quality
  is explicit and no unsupported conclusion is inferred.
- A proposed action changes after approval; the action hash no longer matches
  and approval cannot be replayed.
- An agent attempts an ungranted tool, repository, secret, network target,
  writable path, or runtime; the operation is denied and audited.
- Concurrent users update ownership, lifecycle, decisions, or workflow state;
  invariants are preserved and stale commands receive a conflict response.
- A template, workflow, policy, post, artifact, or context view changes; the
  platform creates a version instead of erasing history.
- AI output contains unsupported claims, unsafe UI, prompt-injected instructions,
  or secret-like material; it is rejected, constrained, or redacted before
  publication or execution.

## 13. Expected end-to-end scenarios

These scenarios combine the atomic use cases into the principal product paths.

1. **Manual incident:** a human opens an incident, assembles a team, posts
   evidence, records facts and a decision, tracks mitigation, verifies recovery,
   completes closure checks, and archives after review.
2. **Alert to context:** Alertmanager creates or deduplicates an incident; a
   recipe posts bounded Prometheus and Loki context; the team accepts an initial
   decision within the incident feed.
3. **Deployment regression:** context links a recent commit to an error increase;
   an agent inspects the change and proposes a rollback, but a human approval is
   required before any operational action.
4. **Conflicting evidence:** metrics and logs disagree; the relevant fact is
   disputed, a targeted refresh is run, and the decision records the remaining
   uncertainty.
5. **Verified code remediation:** an agent reproduces a defect in a sandbox,
   creates and verifies a patch, captures browser evidence in staging, and opens
   a protected pull request.
6. **Controlled autonomous mitigation:** a policy-approved action enters a
   visible countdown; an editor stops it; restart requires a fresh approval and
   policy evaluation.
7. **Degraded manual operation:** Vertex AI or a telemetry integration fails;
   the failure is visible, while humans continue coordination, approval, action,
   verification, and closure.
8. **Learning loop:** closure produces a runbook or known-issue proposal; an
   editor reviews and promotes it; a future incident discovers it through
   search or similarity matching.

## 14. Explicitly deferred or unsupported use cases

The following are outside the current expected scope and must not be implemented
as unreviewed exceptions to the permission or data model:

- notification and paging policy;
- temporary or external participants;
- full Slack or Microsoft Teams synchronisation;
- dedicated native mobile applications;
- arbitrary graph-view semantics;
- detailed retention, deletion, legal-hold, and e-discovery workflows;
- physical-operations automation;
- production browser or general production computer control;
- unrestricted shell, network, credential, or repository access for agents;
- arbitrary agent-generated scripts, forms, event handlers, styles, or UI code;
- a public third-party agent marketplace;
- non-Google model hosting in the MVP;
- customer-hosted or dedicated single-tenant deployment; and
- use as the authoritative store for bulk telemetry, source code, identity, or
  long-lived credentials.

## 15. Coverage rule

A feature belongs in this platform when it materially improves at least one of
the following while preserving the security model:

1. acquiring relevant and attributable operational context;
2. coordinating people and approved agents around an incident;
3. producing explicit, evidence-linked facts, decisions, approvals, or actions;
4. executing investigation or remediation inside a bounded, auditable envelope;
5. verifying resolution and retaining reviewed operational learning.

If a proposed feature does not serve one of these outcomes, or requires bypassing
workspace isolation, immutable history, human accountability, exact-specification
approval, or capability boundaries, it is not an expected use case without a
product-scope decision.
