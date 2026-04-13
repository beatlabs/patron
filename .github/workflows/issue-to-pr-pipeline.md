---
description: Analyze auto-fix labeled issues and create targeted fix PRs.
on:
  label_command:
    label: "auto-fix"
tools: github[issues, pull_requests, repos], bash, edit, cache-memory
imports:
  - shared/mood.md
  - shared/go-ci.md
engine: copilot
strict: true
timeout-minutes: 20
permissions:
  contents: write
  pull-requests: write
  issues: write
safe-outputs:
  create-pull-request:
    title-prefix: "[auto-fix]"
    labels: ["automated", "needs-review"]
    max: 1
    skip-if-match: "\\[auto-fix\\].*open"
  add-comment:
    title-prefix: "[auto-fix]"
    max: 2
  update-issue:
    max: 1
---

## Goal

Handle issues labeled `auto-fix` by producing a narrow, low-risk fix PR or by explaining why the issue is out of scope for automation.

## Analyze the Issue

Read the issue title, body, and comments before taking action.

Classify the request as in scope only if it is a narrow, low-risk fix such as:
- typo or docs-adjacent code typo in source comments or strings
- nil guard or defensive check
- missing validation edge case
- small bug fix with a clear local cause
- simple refactor with no behavior or API surface expansion

If the issue is broad, ambiguous, risky, architectural, or requires product judgment, add a comment explaining that it cannot be auto-fixed safely and stop.

## Scope Assessment

Proceed only if all of the following are true:
- the fix is limited to a single file or very few files (maximum 3)
- the fix does not change any exported function, method, interface, struct, constant, variable, or type signature
- the fix does not require any new dependency
- the fix does not modify test infrastructure

If any condition fails, add a comment explaining which scope check failed and stop.

## Safety Rails

These rules are mandatory:
- NEVER modify files matching: `go.mod`, `go.sum`, `Makefile`, `Taskfile.yml`, `.github/**`, `*.lock.yml`
- NEVER add new dependencies
- NEVER modify more than 3 files
- NEVER change exported function, method, or type signatures
- If any safety rail would be violated, stop and comment why

## Implementation

If the issue is in scope:

1. Create a branch from `master` named `auto-fix/issue-{number}`.
2. Make the minimal code change required to address the issue.
3. Run `task test`.
4. Run `task lint`.

If `task test` or `task lint` fails, discard the attempted fix, restore the branch to its pre-change state, add a comment that auto-fix could not produce a clean fix, and stop.

## Pull Request

If the change is clean and verified, create exactly one PR with:
- title: `[auto-fix] {concise description from issue}`
- body containing `Fixes #{number}`
- a short explanation of what changed and why
- labels: `automated` and `needs-review`

Keep the PR tightly scoped to the issue only.

## Issue Follow-Up

After creating the PR, add a comment on the original issue linking to the PR.

If the issue is out of scope or the fix cannot be validated cleanly, add a concise comment explaining why no PR was created.

## Constraints

- Use `master` as the base branch. Do not use `main`.
- Do not create lock files.
- Do not widen scope beyond the labeled issue.
- Prefer no action over a risky or unclear change.
