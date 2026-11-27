# package-spec PR Review Checklist

Review items grouped by changed-file category. The rules, rationale, and examples live in
[AGENTS.md](../../../../AGENTS.md); each section links to the one to apply. Do not restate rules
here — update AGENTS.md instead.

## Contribution process

When: large or cross-product spec changes, or new package categories.
Rules: [Contribution Process](../../../../AGENTS.md#contribution-process).

- [ ] A substantial spec or validator change references a proposal or tracking issue; if not, ask the author to confirm the process was followed.

## Changelog (always)

Rules: [Changelog Management](../../../../AGENTS.md#changelog-management).

- [ ] `spec/changelog.yml` has a new entry (unless the PR is a Dependabot bump).
- [ ] Entry is at the bottom of the `-next` version's `changes` list.
- [ ] `description` is a complete sentence ending with a period; `type` is `enhancement`, `bugfix`, or `breaking-change`.
- [ ] `link` is a PR URL or `TBD`.
- [ ] Blocked features have a `# Pending on <url>` comment directly above the entry.

## Spec files (`spec/**/*.spec.yml`)

Rules: [Spec File Structure](../../../../AGENTS.md#spec-file-structure), [Version Patches](../../../../AGENTS.md#version-patches).

- [ ] New `properties`/`definitions` that older `format_version` values must not see have `remove` patches, or the change is scoped to the introducing version.
- [ ] New `versions` entries are at the top of the list; properties are removed before their definitions.
- [ ] `_dev` files have no patches.
- [ ] Patch comments follow the path-line convention.
- [ ] Shared fields are defined once in `spec/integration/manifest.spec.yml` and referenced with relative `$ref`.
- [ ] New object schemas set `additionalProperties: false` where appropriate.

## Folder spec files (`spec/**/spec.yml`)

Rules: [Folder Spec Files](../../../../AGENTS.md#folder-spec-files).

- [ ] New folders or files that only belong in source or built packages set `validationMode` (`source` or `build`).
- [ ] New items `$ref` their schema with a relative path.

## Semantic validators (`code/go/internal/validator/semantic/`, `code/go/internal/validator/spec.go`)

Rules: [Semantic Validators](../../../../AGENTS.md#semantic-validators).

- [ ] Every new `ValidateXxx` is registered in `rulesDef` with the right `since`, `until`, `types`, and `modes`.
- [ ] `warnOn()` is used only for rules being phased in.
- [ ] Error messages use `fsys.Path()` and `\"%s\"` (never `%q`) for file paths.
- [ ] Tests cover valid and invalid cases, either as a `t.TempDir()` unit test or as test packages plus `validator_test.go` cases.
- [ ] `expectedErrContains` substrings are specific.

## Test packages (`test/packages/**`, `compliance/testdata/packages/**`)

Rules: [Test Packages](../../../../AGENTS.md#test-packages), [Transform packages in compliance tests](../../../../AGENTS.md#transform-packages-in-compliance-tests).

- [ ] `manifest.yml`, `changelog.yml`, and `docs/README.md` are present.
- [ ] Ingest pipelines tag every processor and have a valid `on_failure` block.
- [ ] Compliance transform packages use a non-hidden dest index and `run_as_kibana_system: false`.

## Compliance features (`compliance/features/*.feature`)

Rules: [Compliance Testing](../../../../AGENTS.md#compliance-testing).

- [ ] Every `Scenario:` has an `@X.Y.Z` tag.
- [ ] Every `@skip` follows the version tag and has a `# Pending ...: <url>` comment.
- [ ] The matching changelog entry has a `# Pending on <url>` comment.

## Go files (`**/*.go`)

Rules: [Go Style](../../../../AGENTS.md#go-style).

- [ ] New files have the Elastic license header.
- [ ] Tabs for indentation; no abbreviated names.
