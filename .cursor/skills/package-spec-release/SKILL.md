---
name: package-spec-release
description: Prepares a package-spec release branch from elastic/main, finalizes spec/changelog.yml, updates compliance CI matrix, reviews Go toolchain (prompts whether to adopt a newer Go minor when stable releases are ahead, otherwise patch-only .go-version), updates AGENTS.md, commits by topic, pushes to the contributor fork, and opens a PR to elastic/main. If git remotes do not clearly identify the user's fork, lists remotes and asks which remote to use. Use when the user asks to release package-spec, prepare a release, or run the release workflow.
license: Elastic License (see LICENSE.txt in repository root)
compatibility: Requires git, network access, and GitHub push access to a fork; GitHub CLI (gh) optional for opening pull requests.
metadata:
  spec: https://agentskills.io/specification.md
---

# package-spec release (Agent Skill)

This skill conforms to the **[Agent Skills](https://agentskills.io/specification.md)** open format ([agentskills.io](https://agentskills.io/)).

## Instructions

1. Read and follow the full procedure in **[references/workflow.md](references/workflow.md)** end-to-end.
2. **Cursor only:** when presenting discrete choices, prefer the **`AskQuestion`** tool when it is available; otherwise use the numbered list (or A/B) pattern described in that file.
