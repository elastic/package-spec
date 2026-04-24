---
name: open-pull-request
description: Drafts or opens a GitHub pull request body that matches this repository’s .github/PULL_REQUEST_TEMPLATE.md, keeps What/Why concise, and fills Related issues with links to issues, PRs, or other repos. Use when the user asks to open a PR, create a pull request, write PR description, or prepare gh pr create.
license: Elastic License (see LICENSE.txt in repository root)
compatibility: Requires repository checkout; optional network for gh. GitHub CLI (gh) optional for creating the PR from the terminal.
metadata:
  spec: https://agentskills.io/specification.md
---

# Open pull request (Agent Skill)

Portable **[Agent Skills](https://agentskills.io/specification.md)** skill: core steps use git, repo files, and plain chat only.

## Instructions

1. **Read the template** at `.github/PULL_REQUEST_TEMPLATE.md` from the repository root. The PR description **must** use the same section headings and checklist structure as that file (do not rename sections or drop checklist items).

2. **Gather facts (short):**
   - Summarize **what** changed (scope, key files or areas) in a few sentences or tight bullets.
   - Summarize **why** in 1–3 sentences (motivation, risk, or user impact).
   - Infer the default base branch from repo conventions if unclear (often `main`).

3. **Related issues / PRs / other repos:**
   - Parse the **user’s request** for URLs, `#123`, `owner/repo#123`, `Closes` / `Fixes` / `Relates`, or explicit issue/PR numbers.
   - Parse **recent commits** on the current branch (e.g. `git log --oneline -20` and, if needed, `git log -1 --format=%B` / `git log --grep` for issue keywords) for the same patterns.
   - If **no** related references are found in the prompt or commits, **ask once in natural language** (or offer two numbered options): whether to include links to related issues, PRs, or other repositories; if yes, ask for a single reply listing URLs or `#refs`. Merge answers into the **Related issues** section using the template’s bullet style (`Closes #…`, `Relates #…`, etc.).

   **Cursor:** If the structured **`AskQuestion`** tool is available, you may use it instead of freeform numbering for the same yes/no + follow-up flow (e.g. “No related items” vs “Yes — I will send references in chat”).

4. **Checklist:** Copy the checklist from the template verbatim. Mark items `[x]` only when verified from the diff or file tree; otherwise leave `[ ]`. For items that do not apply, **strike through** the line with `~~...~~` per the template comment (do not delete rows).

5. **Length:** Keep **What does this PR do?** and **Why is it important?** brief. Prefer pointers (file paths, spec areas) over long narratives. Put deep detail in review comments or commits if needed.

6. **Open the PR:** Prefer `gh pr create` with a body file when `gh` is available and authenticated; otherwise output the final Markdown for the user to paste. Ensure the body matches the template sections exactly before submitting.

## Output shape

Produce a single Markdown document that could be saved and passed to `gh pr create --body-file`, with no extra title line unless the hosting workflow requires it—sections should match `.github/PULL_REQUEST_TEMPLATE.md`.

## Examples

**Related issues line when user said “fixes observability backlog” but no number:**

After the agent asked for references, the user sends `Closes elastic/package-spec#999`. Body includes:

```markdown
## Related issues

- Closes #999
```

**Short What / Why:**

```markdown
## What does this PR do?

- Tightens validation for `foo` in `spec/…`
- Updates `spec/changelog.yml` for the next minor

## Why is it important?

Prevents invalid packages from passing `elastic-package` checks and documents the behavior change for integrators.
```

## Validation

After substantive edits to this skill, run (if installed):

```bash
skills-ref validate ./.cursor/skills/open-pull-request
```
