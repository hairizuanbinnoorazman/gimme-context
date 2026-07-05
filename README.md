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

The initial Helm chart is under `deploy/helm/gimme-context`. Build the backend
container with target `api` or `worker`, and build the frontend with
`web/Dockerfile`.

M1 human incident coordination, M2 automatic operational context, and M3
AI-assisted synthesis are functionally complete. The API-backed Elm
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

The current store remains in-memory for contract iteration. PostgreSQL
durability, workspace administration, comprehensive read authorisation, and
verified OIDC claims are M0 production-readiness work; the development UI uses
the explicitly temporary `X-Principal-ID` boundary.

## License

[MIT](LICENSE)
