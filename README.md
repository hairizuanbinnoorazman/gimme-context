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

Phase 5 extends the M1 manual incident path with versioned incident templates,
template defaults, and immutable per-incident template snapshots. Owned action
state machines, eligible-voter polls, and quorum approvals bound to immutable action hashes,
Evidence-linked facts, owner-accepted immutable decisions, verified summaries,
closure gates, workspace-scoped incidents, typed posts, replies, and revisions
remain available.
The current store is in-memory for contract iteration; PostgreSQL durability,
membership roles, and verified OIDC claims remain production-readiness work.

## License

[MIT](LICENSE)
