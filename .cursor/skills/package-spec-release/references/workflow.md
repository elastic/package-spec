# package-spec release workflow

Portable steps to open a **`prepare-release-<version>`** pull request against `elastic/package-spec` `main`. Repository paths below are relative to the **repository root**.

## Prompting for choices

Whenever the workflow needs a **discrete choice** from the user (fork remote, Go minor vs patches-only, etc.):

1. Be **direct**: one short sentence of context, then the choice (no long setup).
2. Present a **numbered markdown list** (or two clear options **A** / **B**), one entry per concrete choice; put distinguishing detail in each line (e.g. remote name and fetch URL).
3. Ask for a **single reply** (the number, the letter, or the exact label).
4. If the client provides a **structured selectable list** UI, you may use it instead, with the **same** options as the numbered list.

## When the user skips required input

If the user **skips** a required prompt (dismisses structured choices without picking, leaves an answer empty, or otherwise does not provide **required** information): **stop** immediately—no branch creation, no repository edits.

**Reply style:** **Do not** extend the response. **Only** thank the user and, in **one brief sentence**, explain that a release needs **version** confirmation (`X.Y.Z`) and **fork remote** confirmation when that applies. If **version and fork remote are already settled** and the only missing piece is another required choice (e.g. Go minor vs patches), use **one** short sentence naming only that choice instead—still no extra paragraphs or tips.

## 0. Preconditions

- Clean working tree on a machine with `git` and network access.
- Resolve remotes with `git remote -v`:
  - **`UPSTREAM_REMOTE`**: the remote whose fetch URL is **`elastic/package-spec`** (common name `upstream`; sometimes `origin` if the clone is from the elastic org).
  - **`FORK_REMOTE`**: the remote pointing at **the user’s fork** (same repo name under their GitHub user or org), used for `git push` and PR head.
- If **`FORK_REMOTE` is unclear** (e.g. two remotes look like forks, `origin` is elastic not personal, or naming is nonstandard): follow **Prompting for choices**—e.g. *“Which remote is your fork?”* with **one option per remote**, each line showing **name and fetch URL** (e.g. `1. myfork — git@github.com:alice/package-spec.git`). Store the chosen remote name as `FORK_REMOTE`. If the user **skips** without choosing, follow **When the user skips required input**.
- Keep **base branch** = `main` on the elastic org repo (`elastic/main` in GitHub terms).

## 1. Version

1. Ask: **Which version are we releasing?** Expected: `X.Y.Z` (Semantic Versioning, no prerelease suffix in the answer).
2. If the user gives **no version** or empty input, **stop** (do not create branch or edits). Follow **When the user skips required input** for the reply.
3. Validate `X.Y.Z` (three dot-separated non-negative integers). If invalid, ask again or exit with a clear message.

Let `VERSION` = the agreed value (e.g. `3.6.1`).

## 2. Branch

1. `git fetch ${UPSTREAM_REMOTE}` (use the remote that tracks the elastic org repo).
2. Create from latest upstream main:

   `git checkout -b prepare-release-${VERSION} ${UPSTREAM_REMOTE}/main`

3. Push to the **fork**:

   `git push -u ${FORK_REMOTE} prepare-release-${VERSION}`

## 3. Changelog — `spec/changelog.yml`

1. Find a top-level entry whose `version` field is exactly **`${VERSION}-next`** (e.g. `3.6.1-next`).
2. If **no such entry**, **stop** and tell the user the version was not found (they may need a `-next` section added on `main` first).
3. If found, set `version` to **`${VERSION}`** (remove only the `-next` suffix). Do not reorder sections beyond what is needed for the edit.

## 4. Compliance CI — `.buildkite/pipeline.trigger.compliance.tests.sh`

1. Locate calls to `compliance_test <stack_version> <spec_version>`.
2. Update the **latest-stack** row (today the first call, pairing the newest `*-SNAPSHOT` or newest stack with the spec under release): set the **second argument** (`spec_version`) from the **previously released** spec on that row to **`${VERSION}`**.

   Example: releasing `3.6.1` when the first line is `compliance_test 9.4.0-SNAPSHOT 3.6.0` → `compliance_test 9.4.0-SNAPSHOT 3.6.1`.

3. Do **not** change other `compliance_test` rows unless the user explicitly asked to refresh those stack/spec pairs.

## 5. Go toolchain

1. Read `go.mod` `go` directive (repo minor; line may be `1.MM` or `1.MM.P`) and `.go-version` (full `1.MM.P`, e.g. `1.25.8`).
2. From **official Go releases**, determine the **latest stable** version (minor + patch).
3. **Newer minor than `go.mod`?** If latest stable’s **minor** is **greater** than the repo’s `go` minor:
   - State one line of fact (e.g. *Repo `go` is 1.25.x; latest stable Go is 1.26.y.*).
   - Follow **Prompting for choices**—e.g. *“Update Go for this release?”* with exactly two options: **Bump `go.mod` + `.go-version` to latest stable (`1.26.y`)** and **Stay on current minor; patch `.go-version` only**.
   - **If they choose the minor bump:** update `go` in `go.mod` to match this repo’s style (same `1.MM` vs `1.MM.P` pattern as today), set `.go-version` to the new minor’s latest patch, run `go mod tidy` / `make check` if needed, and **state rationale + alignment** with default branches of [elastic/elastic-package](https://github.com/elastic/elastic-package) and [elastic/integrations](https://github.com/elastic/integrations) in the PR.
   - **If they choose patches only:** do **not** change `go.mod` minor; continue with step 4 only. If those repos are already on a **higher** minor, **note the gap in the PR** (no automatic minor bump).
4. **Patch-only path** (no newer stable minor than `go.mod`, **or** user chose patches only in step 3): resolve the **latest patch** for **`go.mod`’s minor**. If that patch is **greater** than `.go-version`, update **only** `.go-version`. Optionally run `go mod tidy` / `make check`.
5. Do not churn unrelated `require` blocks; scope edits to the Go version lines unless `tidy` demands otherwise.

## 6. `AGENTS.md`

After changelog and CI edits, scan `AGENTS.md` for:

- Examples using `TEST_SPEC_VERSION`, `format_version`, validator `semver.MustParse`, or narrative **“currently `X.Y.Z-next`”** notes.
- Compliance / local run examples that should reference **`${VERSION}`** or the new **`-next`** line if a follow-on dev version appears in `changelog.yml`.

Update only what is **factually wrong** after the release edit (keep the doc accurate; avoid unrelated rewrites).

## 7. Commits

Create **one commit per topic**, short imperative subject, e.g.:

- `Update changelog for ${VERSION}`
- `Bump compliance CI spec version to ${VERSION}`
- `Bump Go patch version in .go-version` (if applicable)
- `Bump Go to 1.MM.P` (if `go.mod` / `.go-version` minor bump)
- `Update AGENTS.md for ${VERSION} release`

## 8. Push and pull request

1. `git push ${FORK_REMOTE}` (same branch as in step 2; add `--set-upstream` on first push if not already set).
2. Open a PR **from the fork** → **`elastic/package-spec` `main`** (GitHub UI or `gh pr create --repo elastic/package-spec --base main --head <fork-owner>:prepare-release-${VERSION} --title "Prepare release ${VERSION}"`). Derive `<fork-owner>` from the **`FORK_REMOTE`** URL if needed. If `gh` defaults the head correctly after `push`, `--head` may be omitted.

### PR body

Use [`.github/PULL_REQUEST_TEMPLATE.md`](../../../../.github/PULL_REQUEST_TEMPLATE.md): fill **What**, **Why**, and **Checklist** (strike items that do not apply, e.g. no new test packages for a pure release prep).

- Short **description of the release** (what `VERSION` contains at a high level).
- **Comprehensive summary of changes in this release:** bullet list derived from `spec/changelog.yml` for **`${VERSION}`** (every `changes[].description`, note breaking-change vs enhancement vs bugfix, include links already in the changelog).

Title suggestion: `Prepare release ${VERSION}`.

## Quick verification

- `spec/changelog.yml` contains `version: ${VERSION}` (no stray `${VERSION}-next` for that release block).
- First/latest `compliance_test` uses spec **`${VERSION}`** where intended.
- `go test ./code/go/...` or `make check` if Go files or version files changed.

## Optional reference

For changelog conventions and compliance tags, see existing sections in [`AGENTS.md`](../../../../AGENTS.md) (changelog management, compliance testing).
