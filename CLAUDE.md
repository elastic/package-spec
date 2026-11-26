# Claude Code — package-spec

Contribution rules, repository layout, and testing commands are in **[AGENTS.md](AGENTS.md)**. Read it before changing specs, validators, test packages, or compliance tests.

Agent Skills live in `.agents/skills/` (also linked from `.claude/skills/`), e.g. `/review-pr` and `/open-pull-request`.

## Releases

When preparing a **package-spec release** (e.g. user mentions releasing package-spec, `prepare-release-*`, or finalizing `spec/changelog.yml` for a version), follow **[docs/releasing-package-spec.md](docs/releasing-package-spec.md)**. It links to the canonical workflow under `.agents/skills/package-spec-release/references/workflow.md`.
