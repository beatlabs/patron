---
# Semver policy for Go libraries
# Usage: imports: [shared/semver-policy.md]
---

# Semantic Versioning Policy for Go Libraries

Apply semantic versioning using Go module conventions.

- Breaking API changes require a new major version.
- Additive, backward-compatible API changes are minor version changes.
- Backward-compatible fixes are patch version changes.
- For Go modules, a major version greater than v1 must follow Go module path conventions.

## Breaking Changes in Go

Treat the following as breaking changes when they affect exported API surface:

- Removed exported functions, methods, types, constants, or variables.
- Changed function or method signatures, including parameter lists, parameter types, return values, or return types.
- Removed or renamed exported struct fields.
- Changed exported struct field types in incompatible ways.
- Changed interface method sets.
- Changed exported type definitions in incompatible ways.

## Not Breaking

Do not treat the following as breaking on their own:

- Adding new exported functions, methods, types, constants, or variables.
- Adding new optional exported struct fields.
- Adding new implementations of existing interfaces.

## Labels

- Apply `breaking-change` for confirmed breaking API changes.
- Apply `api-change` for confirmed API surface changes that are not breaking, such as new exports.

## Severity

Classify confirmed changes with this priority:

1. Removal of existing exported API.
2. Signature or type-shape changes to existing exported API.
3. Behavioral changes that may alter consumer expectations.

When uncertain, prefer no finding over a speculative finding.
