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

The project is currently in product and architecture definition.

## License

[MIT](LICENSE)
