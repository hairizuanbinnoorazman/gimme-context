# ADR-0001: Elm frontend with Material Components for Elm

- Status: Accepted
- Date: 2026-07-05

## Context

The selected frontend language is Elm, and the intended visual system is
Material Design. `aforemny/material-components-web-elm` provides Material
Components for Elm by wrapping Material Components for the Web with custom
elements. Version 9.1.0 is available through the Elm package registry and uses
matching JavaScript and CSS assets distributed through npm.

This package is distinct from React's `@mui/material`. Its release cadence and
underlying Material Components version are behind the current Material Design
ecosystem, but it provides sufficient initial component coverage. The project
owner intends to update or replace it later.

## Decision

Adopt `aforemny/material-components-web-elm` 9.1.0 for the initial frontend,
including its version-matched JavaScript and CSS assets. Do not use React's
`@mui/material`.

Application modules must not depend directly on the third-party component API.
A project-owned Elm view layer wraps the package and exposes the components,
design tokens, and interaction contracts required by the application. This
boundary supports a future package upgrade, maintained fork, or replacement.

The package's required custom elements, JavaScript, and CSS are approved frontend
runtime dependencies. Other JavaScript ports or custom elements remain limited
to isolated capabilities that lack a practical Elm implementation, such as an
advanced code editor, specialised graph canvas, or media recorder. These
integrations receive typed, validated data and do not own application state.

The controlled post renderer supports trusted Elm components for tables, charts,
polls, approvals, checklists, diffs, timelines, and status panels. Sanitised
Markdown and limited HTML are content formats, not application-component APIs.

## Consequences

- The frontend retains Elm's application state model while using the package's
  custom-element runtime for component behaviour.
- Initial Material components are available without implementing the full design
  system from scratch.
- The package JavaScript and CSS versions must match the Elm package version.
- Project-owned wrappers add implementation work but contain upgrade risk.
- The package's age and older Material foundation create acknowledged technical
  debt. Updating, forking, or replacing it requires a later ADR.
- Accessibility, keyboard behaviour, focus management, and responsive layouts
  must be verified by this project rather than assumed from the dependency.
