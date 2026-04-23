---
on:
  pull_request:
    types: [opened, synchronize]
    paths: ["**.go"]
    branches: [main]

tools:
  github:
    toolsets: [pull_requests, repos]
  bash: true

imports:
  - shared/mood.md
  - shared/go-ci.md
  - shared/semver-policy.md

engine: copilot
strict: true
timeout-minutes: 15

permissions:
  contents: read
  pull-requests: read

safe-outputs:
  add-comment:
    max: 1
  add-labels:
    max: 2
  create-pull-request-review-comment:
    max: 5
---

# Breaking Change Checker

Review this pull request for high-confidence Go API surface changes compared with the base branch.

## Goal

Detect breaking changes for downstream consumers of patron. Version one must focus on HIGH-CONFIDENCE detections only.

## Scope

Compare the PR diff against the base branch to inspect exported API changes in Go packages.

Focus on:

- Removed exported functions, methods, types, constants, or variables.
- Changed function or method signatures, including parameter lists and return types.
- Changed or removed exported struct fields.
- Changed interface method sets.

Also detect non-breaking exported API additions so they can be labeled as `api-change`.

## Ignore

Do NOT flag:

- Internal or unexported changes.
- Test file changes.
- Changes in `_test.go` files.
- Vendored dependency changes.
- Low-confidence behavioral speculation without a clear exported API delta.

Keep the false positive rate LOW. Only report changes you are confident about.

## How to Inspect

Use GitHub pull request metadata and repository contents to understand the changed Go files.

Use `bash` with `go list` and `go doc` to inspect exported symbols and compare the base branch against the PR branch. Use exported-symbol comparison only; do not rely on heuristics about internal code.

Recommended approach:

1. Identify changed Go files in the PR, excluding `_test.go` files and vendored paths.
2. Determine the affected packages.
3. Compare exported API surface between the base branch and the PR branch.
4. Record only clear, consumer-visible API changes.

## Required Outputs

If you find confirmed breaking changes:

- Post a review comment on the specific changed line when possible.
- Each review comment must name the symbol and describe exactly what changed.
- Explain the likely impact on downstream consumers.

Post exactly ONE summary comment with the `[breaking-change]` prefix listing all detected breaking changes.

## Labels

- Add `breaking-change` if any confirmed removals or signature changes are detected.
- Add `api-change` if there are confirmed non-breaking API surface additions such as new exported symbols.

## Reporting Rules

- Prefer precise symbol-level findings.
- If a symbol is removed, state that it is no longer available to consumers.
- If a signature changed, state the old vs new shape when confidently determinable.
- If an exported struct field changed or was removed, state how consumer code may fail to compile or behave differently.
- If an interface method set changed, state that implementations or consumers may no longer compile.

If no high-confidence exported API changes are found, do not force a finding.
