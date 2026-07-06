# Gimme Context

Gimme Context is a post-based incident coordination platform for humans and AI
agents. It gathers operational context, turns discussion into auditable
decisions and actions, and feeds confirmed learning back into permanent
channels.

## Documentation

- [Product requirements](docs/product-requirements.md)
- [Domain model](docs/domain-model.md)
- [Technical architecture](docs/technical-architecture.md)
- [MVP delivery plan](docs/mvp-delivery-plan.md)
- [Frontend decision: Elm and Material Design](docs/adr/0001-elm-material-design.md)

## Initial technology direction

- Backend: Go
- Frontend: Elm
- UI components: `aforemny/material-components-web-elm` 9.1.0
- Deployment: containers on Google Kubernetes Engine
- AI: Vertex AI
- Identity: OpenID Connect, tested with Keycloak and Dex
- Initial integrations: Prometheus, Alertmanager, Loki, and GitHub

## Local development

Prerequisites are Go 1.26, Elm 0.19.1, Node.js 24, and npm 11.

```sh
make test
make build
make web-build
```

Run the API with `go run ./cmd/api`; it listens on port 8080 by default and
exposes `/health/live`, `/health/ready`, and `/api/v1`. Run the asynchronous
worker separately with `go run ./cmd/worker`.

For a complete local stack, Docker Compose builds and starts the API, worker,
and web proxy. The UI is available at <http://localhost:8080>.

```sh
make compose-up
make compose-down
```

Set `APP_PORT` to use another host port, for example
`APP_PORT=18080 make compose-up`. Application data is held in API memory and is
discarded when the API container is replaced.

The initial Helm chart is under `deploy/helm/gimme-context`. Build the backend
container with target `api` or `worker`, and build the frontend with
`web/Dockerfile`.

## Minikube deployment

Prerequisites are Docker, Minikube, Helm 3, curl, and jq. Deploy all
three workloads, load their local images into Minikube, and wait for readiness:

```sh
make minikube-deploy
make minikube-smoke
minikube service --namespace gimme-context gimme-context-web
```

The smoke test reaches the UI and API through the web service, sets the
development principal, creates an incident, posts an update, and reads it back.
The UI's **Acting as** field and its `X-Principal-ID` request header are the
current development login boundary. They are suitable only for local testing;
verified OIDC login is not implemented yet.

Set `MINIKUBE_PROFILE`, `NAMESPACE`, or `RELEASE` to override their defaults.
Set `MINIKUBE_IMAGE_TAG` when a stable image tag is needed; otherwise every
deployment generates a tag so Kubernetes rolls out the new local build. The
smoke script also accepts `SMOKE_PORT` (default `18080`). Because the store is
currently in memory, application data is lost whenever the API pod restarts.

M1 through M6 are functionally complete against the in-memory contract store.
The API-backed Elm
interface supports permanent and incident channels, template-based incident
creation, typed posts and replies, author revisions, roles and ownership,
structured facts/decisions/actions/polls/approvals, and gated incident closure.
It also supports versioned context recipes, Alertmanager-created incidents,
Prometheus/Loki collection posts, visible retrieval failures and refreshes, and
similar-incident proposals. Backend and frontend tests cover both the manual
incident path and the alert-to-initial-context path without AI.

The M3 API adds approved Vertex AI agent definitions, incident activation,
time-bound capabilities, auditable runs, evidence-linked and redacted proposals,
human accept/reject review, and agent collaboration envelopes. Configure the
gateway with `VERTEX_AI_ENDPOINT` and a short-lived `VERTEX_AI_ACCESS_TOKEN`.

The later API slices add controlled workflows, sandboxed GitHub investigation,
versioned artifact promotion and runbook simulation, reviewed archival, audit
export, and pilot comparisons for time to context and first accepted decision.

The current store remains in-memory for contract iteration. PostgreSQL
durability, workspace administration, comprehensive read authorisation, and
verified OIDC claims are M0 production-readiness work; the development UI uses
the explicitly temporary `X-Principal-ID` boundary.

## License

[MIT](LICENSE)
