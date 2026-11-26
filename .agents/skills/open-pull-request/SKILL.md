---
name: open-pull-request
description: Draft a pull request body that follows .github/PULL_REQUEST_TEMPLATE.md, with concise What/Why and related issue links. Use when the user asks to open a PR, create a pull request, or write a PR description.
---

# Open pull request

1. **Read the template** at `.github/PULL_REQUEST_TEMPLATE.md` from the repository root. The PR description **must** use the same section headings and checklist structure as that file (do not rename sections or drop checklist items).

2. **Gather facts (short):**
   - Summarize **what** changed (scope, key files or areas) in a few sentences or tight bullets.
   - Summarize **why** in 1–3 sentences (motivation, risk, or user impact).
   - Infer the default base branch from repo conventions if unclear (often `main`).

3. **Related issues / PRs / other repos:**
   - Parse the **user’s request** for URLs, `#123`, `owner/repo#123`, `Closes` / `Fixes` / `Relates`, or explicit issue/PR numbers.
   - Parse **recent commits** on the current branch (e.g. `git log --oneline -20` and, if needed, `git log -1 --format=%B` / `git log --grep` for issue keywords) for the same patterns.
   - If **no** related references are found in the prompt or commits, **ask once in natural language** (or offer two numbered options): whether to include links to related issues, PRs, or other repositories; if yes, ask for a single reply listing URLs or `#refs`. Merge answers into the **Related issues** section using the template’s bullet style (`Closes #…`, `Relates #…`, etc.).

4. **Checklist:** Copy the checklist from the template verbatim. Mark items `[x]` only when verified from the diff or file tree; otherwise leave `[ ]`. For items that do not apply, **strike through** the line with `~~...~~` per the template comment (do not delete rows).

5. **Length:** Keep **What does this PR do?** and **Why is it important?** brief. Prefer pointers (file paths, spec areas) over long narratives. Put deep detail in review comments or commits if needed.

6. **Proposed commit message:** Fill the template's **Proposed commit message** section, following its comment, with a fenced block the maintainer can paste when squash-merging. Wrap the body at ~72 characters and describe the change as a whole, not the list of branch commits.

7. **Output:** a single Markdown body (no title line) usable with `gh pr create --body-file`. Open the PR with `gh` only if the user asks; otherwise hand over the Markdown.

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

**Proposed commit message:**

````markdown
## Proposed commit message

```
Tighten validation for foo

Reject packages that set foo to an unsupported value, so invalid
packages fail elastic-package checks instead of at install time.
```
````
