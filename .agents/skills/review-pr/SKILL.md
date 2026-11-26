---
name: review-pr
description: Review a package-spec pull request or branch for completeness and correctness. Use when the user asks to review a PR, check their branch is ready to merge, verify they have not missed required changes, or wants a pre-submission checklist. Accepts a PR number, a branch name, or no argument (current branch vs main).
license: Elastic License (see LICENSE.txt in repository root)
compatibility: Requires git. For PR number resolution, also requires the gh CLI authenticated against elastic/package-spec. For local validation (make check, go test), requires a Go toolchain and make. Compatible with Claude Code (/review-pr), Cursor, GitHub Copilot, Gemini CLI, and other Agent Skills-compatible agents.
metadata:
  spec: https://agentskills.io/specification.md
---

# package-spec PR Review (Agent Skill)

This skill conforms to the **[Agent Skills](https://agentskills.io/specification.md)** open format ([agentskills.io](https://agentskills.io/)).

**This skill does not modify the codebase.** It does not edit source files, run any git
command that modifies state (no commit, push, checkout, reset, stash, merge, or rebase), or
change anything on the branch under review. If an issue is found, describe it and stop — do
not attempt to fix it. The only file it writes is the review report at `.agents/review-pr-output.md`.

Reviews a branch or pull request against the contribution rules for this repository.
See [references/checklist.md](references/checklist.md) for the checklist and [AGENTS.md](../../../AGENTS.md) for the rules behind it.

For narrative onboarding (Makefile targets, scaffolding test packages, testing against the integrations repo), see **[CONTRIBUTING.md](../../../CONTRIBUTING.md)** at the repository root. This skill and the checklist focus on **PR completeness** against those practices.

## How to Use This Skill

| Situation | What to do |
| --- | --- |
| **Claude Code** | `/review-pr`, `/review-pr <branch>`, or `/review-pr <PR-number>` (discovered via `.claude/skills/`) |
| **Cursor / Copilot** | Discovered via `.cursor/skills/` / `.github/skills/`; ask to "review my changes", or attach this file to the chat |
| **Any LLM** | Paste this file as a system message, then provide the output of `git diff main...HEAD` |

---

## Step 1 — Resolve the Target

Determine what to diff based on the input provided:

**If a PR number was given** (e.g., `1140`):

```bash
gh pr view 1140 --repo elastic/package-spec --json headRefName,baseRefName,title,url
git fetch origin <headRefName>
# diff: FETCH_HEAD vs <baseRefName>
git diff <baseRefName>...FETCH_HEAD --stat
git diff <baseRefName>...FETCH_HEAD
```

**If a branch name was given** (e.g., `my-feature`):

```bash
git diff main...<branch> --stat
git diff main...<branch>
```

**If no argument was given** (review current branch):

```bash
git branch --show-current          # confirm we are not on main
git diff main...HEAD --stat
git diff main...HEAD
```

If the current branch is `main` and no argument was given, report that there is nothing to review.

---

## Step 2 — Categorize Changed Files

Using the file list from `--stat`, bucket each changed file into one or more categories:

| Category | File pattern |
| --- | --- |
| **Spec files** | `spec/**/*.spec.yml` |
| **Changelog** | `spec/changelog.yml` |
| **Semantic validators** | `code/go/internal/validator/semantic/*.go` (excluding `*_test.go`) |
| **Validator registry** | `code/go/internal/validator/spec.go` |
| **Validator unit tests** | `code/go/internal/validator/semantic/*_test.go` |
| **Validator integration tests** | `code/go/pkg/validator/validator_test.go` |
| **Test packages** | `test/packages/**` |
| **Compliance packages** | `compliance/testdata/packages/**` |
| **Compliance features** | `compliance/features/*.feature` |
| **Go files** | `**/*.go` |

Use the categories internally to pick checklist sections; do not paste the full categorized file list into the report unless it clarifies an extra-review item.

---

## Step 3 — Run the Checklist

For each category found in Step 2, work through the matching section of
[references/checklist.md](references/checklist.md). Each section links to the rules in
[AGENTS.md](../../../AGENTS.md), the single source of truth for contribution rules; read the
linked AGENTS.md section when an item needs detail or examples.

Always run the **Changelog** and, for large or cross-product changes, the **Contribution process**
sections, whatever the categories.

---

## Step 4 — Run Local Validation

If a Go toolchain and `make` are available in the environment, run at minimum (same order as [CONTRIBUTING.md](../../../CONTRIBUTING.md#testing-your-changes)):

```bash
# Validate spec YAML syntax (fast, always do this after spec changes)
go test ./code/go/internal

# Lint and license headers
make check
```

**Optional (broader validation):** When the diff warrants it, also run `make test` for the full suite ([CONTRIBUTING.md](../../../CONTRIBUTING.md#testing-your-changes)). For changes that should be exercised across many real packages, see **Testing with integrations repository** in [CONTRIBUTING.md](../../../CONTRIBUTING.md#testing-with-integrations-repository) (e.g. `test integrations` PR comment and closing the spawned integrations PR when done).

Record validation outcomes only as described in **Step 5** (short summary per command; on failure, include a small excerpt only). If the toolchain is not available, note `skipped` for those commands.

---

## Step 5 — Write the Report

Write a **short** report to `.agents/review-pr-output.md`, overwriting any previous run. Use **plain text only** (no emoji or decorative symbols). Aim for roughly one screen of content.

Include **only** what follows. Omit empty sections entirely.

```markdown
# PR review: <branch or PR title>
<!-- reviewed: <ISO date> -->

## Scope

At most 2–4 bullets: what the diff changes, only if that context helps a reviewer. Omit this section if the PR title and `git diff --stat` are enough and nothing needs extra review.

## Extra review

Bullets **only** for items that need human attention: checklist gaps, blockers, open questions, or parts of the change that are easy to get wrong (for example new `versions` patches, new semantic validators, `spec/changelog.yml` entries, compliance `@skip`, or cross-file `$ref`). For each bullet use `path:approximate-line` (or `path` and a short region description) plus **one sentence**. Prefix with `blocker`, `suggestion`, or `question` when useful.

If there is nothing to flag here and validation passed, use a single line instead of a list:

No items flagged for extra review.

## Validation

One line per command you ran, for example: `go test ./code/go/internal: pass`. Use `fail` or `skipped` as appropriate. If a command failed, add a **short** excerpt (failing test name or error lines only), not the full log. If no commands ran, one line: `Validation: skipped (toolchain not available).`
```

**Do not** add per-topic sections (changelog, version patches, tests, etc.) when there is nothing to say. Do not paste full `go test` or `make check` output on success. Do not list files that look fine.

After writing the file, print this message to the user:

```
Review written to .agents/review-pr-output.md
```

## Validating this skill

After substantive edits to this skill, run (if installed):

```bash
skills-ref validate ./.agents/skills/review-pr
```
