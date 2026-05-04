# package-spec PR Review Checklist — Detailed Reference

This file contains the full rules, rationale, and examples for each review category.
It is referenced by `SKILL.md` and is loaded on demand when deeper detail is needed.

The human-oriented contributor guide ([CONTRIBUTING.md](../../../../CONTRIBUTING.md)) covers the same development workflows (Makefile targets, test packages, integrations testing) in narrative form. Prefer that document for onboarding; use this checklist for systematic PR review.

---

## 0. Contribution process

**When it applies:** Large or cross-product spec changes, or new package categories.

| Topic | Rule |
| --- | --- |
| **Change proposals** | Changes that impact the Elastic Stack or Elastic Packages beyond this repo should follow the **Change Proposals** process in [CONTRIBUTING.md](../../../../CONTRIBUTING.md#change-proposals) (GitHub issue, consensus, ordered checklist) before landing as a single drive-by PR. |
| **Category proposals** | New supported package categories require the **Category Proposals** flow in [CONTRIBUTING.md](../../../../CONTRIBUTING.md#category-proposals). |

**Review hint:** If the diff is a substantial spec or validator change and the PR description does not reference a proposal or tracking issue, add an extra-review note asking the author to confirm process was followed.

---

## 1. Changelog

**File:** `spec/changelog.yml`

### Changelog Format

```yaml
- version: 3.7.0-next        # in-development version always has -next suffix
  changes:
    - description: Brief description of the change ending with a period.
      type: enhancement        # enhancement | bugfix | breaking-change
      link: https://github.com/elastic/package-spec/pull/NUMBER
```

### Changelog Rules

| Rule | Detail |
| --- | --- |
| Always required | Every non-trivial PR must add an entry. Dependabot dependency bumps are the only exception. |
| Position | New entries go at the **bottom** of the `changes` list of the in-development version. |
| Version | Must be the version with a `-next` suffix, not a released version. |
| Description | Complete sentence ending with a period. Describes what was added/changed, not why. |
| Type | Exactly one of: `enhancement`, `bugfix`, `breaking-change`. |
| Link | Full GitHub PR URL. Use `TBD` only if the PR does not exist yet. Never use an issue URL here unless explicitly intended. |

### Pending features

When a spec feature is complete but blocked on upstream (Kibana, elastic-package):

```yaml
    # Pending on https://github.com/elastic/kibana/issues/NNNNNN
    - description: Add support for foo.
      type: enhancement
      link: https://github.com/elastic/package-spec/pull/NNN
```

- The `# Pending on <url>` comment must appear on the line **directly above** the entry.
- The corresponding compliance `.feature` file must also have a `@skip` tag with the same URL.
- Remove both when the blocker is resolved.

---

## 2. Version Patches

**Why they exist:** The spec supports multiple format versions. New fields added in, say, 3.7.0
must be invisible to packages using 3.6.x or earlier. Version patches achieve this by removing
the new fields from the schema when validating older packages.

**Where they live:** In the same `.spec.yml` file that defines the new field, under a `versions`
key at the top level.

### Patch Format

```yaml
versions:
  - before: 3.7.0
    patch:
      - op: remove
        path: "/properties/my_new_field"
      - op: remove
        path: "/definitions/my_new_definition"
```

### Patch Rules

| Rule | Detail |
| --- | --- |
| Required for all new fields | Every new `properties` entry and `definitions` entry in a `.spec.yml` needs a `versions.before` patch. |
| `versions` list order | Add new version entries at the **top** of the `versions` list (newer spec versions first), per [CONTRIBUTING.md](../../../../CONTRIBUTING.md#version-patches). |
| Remove order | **References before definitions.** Remove `/properties/my_field` before `/definitions/my_field`. Otherwise the patch fails because it tries to remove a definition that is still referenced. |
| `_dev` exception | Files under `_dev` directories do not need version patches. These are developer tooling files, not part of the distributed spec. |
| Patch comments | Comments should be placed on the `path:` line, not the `op:` line. Only add comments for non-obvious paths (e.g., array index paths like `required/-`). Omit comments for simple, self-explanatory paths. |

### Comment convention examples

```yaml
# GOOD — comment on path line, only for complex path
- op: add
  path: "/properties/policy_templates/items/required/-"  # re-add type as required
  value: type

# GOOD — no comment for simple path
- op: remove
  path: "/properties/my_field"

# BAD — comment on op line
- op: remove  # removes my_field
  path: "/properties/my_field"

# BAD — redundant comment for obvious path
- op: remove
  path: "/properties/my_field"  # removes my_field property
```

### Version Patches Checklist

- [ ] Every new `properties` key has a `remove` patch.
- [ ] Every new `definitions` key has a `remove` patch.
- [ ] Properties are removed before their definitions in the patch list.
- [ ] `_dev` files are excluded.
- [ ] Comments follow the path-line convention.

---

## 3. Schema Reuse

**Why it matters:** Duplicate inline definitions drift out of sync. The spec uses JSON Schema
`$ref` to define a field once and reference it everywhere.

### When to use `$ref`

If the same field definition appears in two or more spec files: define it once in
`spec/integration/manifest.spec.yml` under `definitions`, then reference it everywhere else.

### Cross-file reference syntax

```yaml
# Same file
$ref: "#/definitions/my_field"

# From spec/input/manifest.spec.yml referencing spec/integration/manifest.spec.yml
$ref: "../integration/manifest.spec.yml#/definitions/my_field"

# From spec/integration/data_stream/manifest.spec.yml
$ref: "../../integration/manifest.spec.yml#/definitions/my_field"
```

Always use **relative paths** (starting with `../`). Absolute or root-relative paths break
portability.

### `additionalProperties: false`

Objects should have `additionalProperties: false` to prevent undeclared fields from silently
passing validation. Flag if a new object type is missing this.

### Schema Reuse Checklist

- [ ] No field definition appears inline in two different `.spec.yml` files.
- [ ] Shared definitions live in `spec/integration/manifest.spec.yml`.
- [ ] All `$ref` uses relative paths.
- [ ] New object schemas include `additionalProperties: false` where appropriate.

---

## 4. Semantic Validators

**Location:** `code/go/internal/validator/semantic/`

### Registration

New validator functions must be registered in `code/go/internal/validator/spec.go` inside
the `rulesDef` slice within the `rules()` function:

```go
// Minimal — applies to all package types and all versions
{fn: semantic.ValidateMyRule},

// With version gate — only applies for spec >= 3.7.0
{fn: semantic.ValidateMyRule, since: semver.MustParse("3.7.0")},

// With package type filter
{fn: semantic.ValidateMyRule, types: []string{"integration", "input"}, since: semver.MustParse("3.7.0")},

// With upper bound — only applies before 3.0.0
{fn: warnOn(semantic.ValidateMyRule), until: semver.MustParse("3.0.0")},
```

Fields:

- `fn` — required, the validator function
- `since` — optional, minimum spec version (inclusive)
- `until` — optional, maximum spec version (exclusive)
- `types` — optional, package types this applies to; omit to apply to all types

### Semantic Validators Checklist

- [ ] New `ValidateXxx` function is present in the `rulesDef` slice.
- [ ] `since` is set when the rule corresponds to a new spec feature (should match the spec
  version being targeted).
- [ ] `types` is set when the rule is not universally applicable (e.g., only for `integration`
  packages).
- [ ] `warnOn()` is used only for rules that are being phased in (deprecation warnings before
  becoming hard errors in a future version).

---

## 5. Tests for Semantic Validators

Two testing strategies; choose based on complexity:

### Simple: unit test with `t.TempDir()`

Use when the validator reads a single file or a simple directory structure.

```go
func TestValidateMyRule(t *testing.T) {
    tests := map[string]struct {
        manifest      string
        expectError   bool
        errorContains string
    }{
        "valid": {
            manifest:    "name: test\nformat_version: 3.7.0\n",
            expectError: false,
        },
        "invalid_missing_field": {
            manifest:      "name: test\nformat_version: 3.7.0\nbad_field: value\n",
            expectError:   true,
            errorContains: "bad_field",
        },
    }
    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            pkgRoot := t.TempDir()
            err := os.WriteFile(filepath.Join(pkgRoot, "manifest.yml"),
                []byte(tc.manifest), 0644)
            require.NoError(t, err)
            fsys := fspath.DirFS(pkgRoot)
            errs := semantic.ValidateMyRule(fsys)
            if tc.expectError {
                require.NotEmpty(t, errs)
                require.Contains(t, errs[0].Error(), tc.errorContains)
            } else {
                require.Empty(t, errs)
            }
        })
    }
}
```

### Complex: test packages + `validator_test.go`

Use when the rule requires a full package structure (multiple data streams, pipelines, etc.).

1. Create the package: `cd test/packages && elastic-package create package`
2. Modify its `manifest.yml` to exercise the rule.
3. Add entries to `code/go/pkg/validator/validator_test.go`:

```go
tests := map[string]struct {
    invalidPkgFilePath  string
    expectedErrContains []string
}{
    "good_my_feature": {},   // valid — no errors expected
    "bad_my_feature": {
        "manifest.yml",
        []string{`expected error substring here`},
    },
}
```

### Validator Tests Checklist

- [ ] At least one of the above test strategies is present for every new validator.
- [ ] Tests cover both the valid and invalid cases.
- [ ] Error message substrings in `expectedErrContains` are specific enough to be meaningful.

---

## 6. Test Packages

**Location:** `test/packages/` (general), `compliance/testdata/packages/` (compliance-only)

### Required files

Every package directory must contain:

```text
my_package/
├── manifest.yml          # package manifest
├── changelog.yml         # at minimum: one version entry with at least one change
└── docs/
    └── README.md         # package documentation
```

### Valid ingest pipeline

Any data stream that includes an ingest pipeline must follow this structure:

```yaml
---
description: Pipeline description.
processors:
  - set:
      tag: set_field_name     # tag is required on every processor
      field: my.field
      value: test
on_failure:                   # on_failure block is required
  - set:
      field: event.kind
      value: pipeline_error
  - set:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        with tag '{{{ _ingest.on_failure_processor_tag }}}'
        in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
```

Required elements:

- Every processor must have a `tag` field.
- `on_failure` must set `event.kind` to `pipeline_error`.
- `error.message` must include all three Handlebars placeholders:
  `_ingest.on_failure_processor_type`, `_ingest.on_failure_processor_tag`, `_ingest.pipeline`.
  (A fourth, `_ingest.on_failure_message`, is also included by convention.)

### Transform packages (compliance only)

Transforms in `compliance/testdata/packages/` that are installed and uninstalled across
test runs must use:

```yaml
dest:
  index: "metrics-mypackage.my_dest_default"  # no leading dot — hidden indices cause 403 on uninstall
_meta:
  managed: true
  run_as_kibana_system: false  # use logged-in user credentials, not kibana_system
```

### Test Packages Checklist

- [ ] `manifest.yml`, `changelog.yml`, and `docs/README.md` all present.
- [ ] Data stream ingest pipelines have `tag` on each processor and a valid `on_failure` block.
- [ ] Compliance-only transform packages use non-hidden dest index and `run_as_kibana_system: false`.
- [ ] Package was created using `elastic-package create package` (not manually scaffolded) when possible.

---

## 7. Compliance Feature Files

**Location:** `compliance/features/*.feature`

### Version tagging

Every `Scenario:` must be tagged with the minimum spec version that introduced the feature:

```gherkin
  @3.6.0
  Scenario: Integration package with named input can be installed
   Given the "good_v3" package is installed
   ...
```

The tag controls whether the scenario runs: it is skipped if the `TEST_SPEC_VERSION`
environment variable is lower than the tag.

### Skipped scenarios

When a scenario cannot pass because upstream (Kibana or elastic-package) hasn't implemented
support yet:

```gherkin
  @3.6.0
  @skip
  # Pending support for qualified input names: https://github.com/elastic/kibana/pull/262138
  Scenario: Integration package with OTel input can be installed
   Given the "good_v3" package is installed
   ...
```

Rules:

- `@skip` must appear directly after the version tag.
- A `# Pending <description>: <url>` comment must appear on the line after `@skip`.
- The corresponding `spec/changelog.yml` entry must have a matching `# Pending on <url>`
  comment above it.
- Both the `@skip` and the `# Pending on` comment in the changelog must be removed together
  when the blocker is resolved.

### Compliance Features Checklist

- [ ] Every `Scenario:` has a `@X.Y.Z` tag.
- [ ] Every `@skip` is accompanied by a `# Pending ...` comment with a URL.
- [ ] Changelog entry for the same feature has a matching `# Pending on <url>` comment.
- [ ] No scenario has `@skip` without a tracking issue URL.

---

## 8. Go Style

### License header

Every new `.go` file must begin with:

```go
// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
```

Enforced by `make check`. Running `make -C code/go update` adds headers automatically.

### Naming conventions

| Bad (abbreviation) | Good (full name) |
| --- | --- |
| `pkgName` | `packageName` |
| `dsManifests` | `dataStreamManifests` |
| `ptIdx` | `templateIndex` |
| `errs` (in loops) | `validationErrors` or be specific |

Exception: loop variables (`i`, `j`) and widely accepted short names (`err`, `ok`) are fine.

### Indentation

Go files use **tabs**, not spaces. `gofmt` enforces this — run `make -C code/go format`.

### Error messages

When creating structured errors with file paths, use `fsys.Path("relative/path")` rather than
`file.Path()` to get the full package-relative path expected by the test framework:

```go
// Good
specerrors.NewStructuredErrorf("file %q is invalid: %s", fsys.Path("_dev/test/config.yml"), err)

// Bad — path is relative within fsys, not the full package path
specerrors.NewStructuredErrorf("file %q is invalid: %s", config.Path(), err)
```

### Go Style Checklist

- [ ] License header present in every new `.go` file.
- [ ] No abbreviations in variable/function names (outside `err`, `ok`, loop vars).
- [ ] Indentation uses tabs.
- [ ] Error messages use `fsys.Path()` for file path references.
