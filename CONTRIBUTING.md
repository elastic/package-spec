# Contributing

## Change Proposals

Any changes to Elastic Stack products or Elastic Packages that require changes to the Elastic Package Specification
must be discussed first and then safely implemented across impacted products. At a high level, this process looks
something like this:

1. Propose your change via a [new Change Proposal issue](https://github.com/elastic/package-spec/issues/new/choose)
   in the `package-spec` repository (i.e. this repository). This gives us an opportunity to understand which parts of the
   Elastic Stack might be impacted by this change and pull in relevant experts to get their opinions. The initial proposal
   should cover these areas:
   - What problem the proposal is solving. This provides context and could help shape the solution.
   - Where the solution will need to be implemented, i.e. which parts, if any, of the Elastic Stack will be impacted. It's
     okay if the initial proposal doesn't get this 100% right; the discussion in the proposal issue should provide clarity.
     Feel free to tag relevant experts to get their opinions.

2. Discussion ensues and eventually we hope to reach some consensus.

3. Once there's consensus on the proposal issue, modify the issue description to include an **ordered** checklist of
   tasks that need to be resolved to make the change happen in a safe way.  For example, maybe Kibana needs to implement
   support for a new property in the Package Specification first, then only when that support is implemented, the Package
   Specification can itself be modified, which would then allow packages to define this property and have these packages
   still be valid. At this point the proposal issue serves as a meta issue for the safe implementation of the change.

4. File issues in each of the repositories corresponding to the checklist items and update the checklist with links to
   these issues.

5. No single PR may close the proposal issue. But as these PRs get resolved, the corresponding checklist item should be
   checked off. The proposal issue is closed when all items are checked off.

## Category Proposals

Any changes on the categories supported by the Elastic Package Specification has to be discussed. The process is:

1. A solutions team or an integration author creates a
[Category Proposal issue](https://github.com/elastic/package-spec/issues/new/choose)
in the package-spec repo requesting the addition of a new category.
2. Github issue provides details related to the category i.e. label, position, solution, etc. Answers to all the points in
the eligibility requirement and a plan or timeline to add a promised minimum set of integrations.
3. Ecosystem team assesses the request and makes a decision based on data points provided in the GitHub issue.
4. If we are not able to address and the category doesn’t meet eligibility requirements, we can assume that
this category is more harmful to the UX than the value it provides, and hence, it must not be added.
5. If the decision is in favor of adding the category, the Ecosystem team will prioritize and work on the GitHub issue in
one of the releases under planning (future releases).

## Folder Item spec

### Using predefined placeholders in filename patterns

There are predefined placeholders that can be used in filename patterns. Standard patterns are simple regular expressions
like `[a-z0-9]+\.json`. In some cases it might be useful to introduce a harder requirement (binding), e.g. a filename should
be prefixed with a package name (`{PACKAGE_NAME}-.+\.json`).

Currently, the following placeholders are available:

* `{PACKAGE_NAME}` - name of the package

## Folder Item schema

### Defining property format

The JSON Schema defines the basic structure of a JSON document (e.g. package manifests, ingest pipelines, etc.).
In some cases this might be insufficient as there are properties that require strict validation (not just type
consistency), e.g. format validation:

```yaml
src:
  description: Relative path to the screenshot's image file.
  type: string
  format: relative-path
  examples:
  - "/img/apache_httpd_server_status.png"
```

Currently, the following custom formats are available:

* `relative-path`: Relative path to the package root directory. The format checker verifies if the path is correct and
  the file exists.
* `data-stream-name`: Name of a data stream. The format checker verifies if the data stream exists.


## Development

Download the latest main of `package-spec` repository:
```bash
git clone https://github.com/elastic/package-spec.git
cd package-spec
make test
```

While developing on a new branch, there are some [Makefile targets](./Makefile) available
that will help you in this development phase:
- `make update`: add required license header in all the needed files.
- `make test`: run all the tests
- `make check`: run lint and ensures that license headers are in-place.

### Testing Your Changes

When modifying spec files or the changelog, always validate the syntax with:

```bash
go test ./code/go/internal
```

This command validates:
- Spec file syntax (manifest.spec.yml, etc.)
- Changelog format and structure
- Schema definitions and references
- JSON patches for version compatibility

This is **different** from package validation tests. The internal tests validate the specification
itself, while package tests validate that packages conform to the specification.

To run package validation tests:

```bash
# Test specific package validation
go test -v -run "TestValidateFile/my_test" ./code/go/pkg/validator/...

# Run all tests (includes both spec and package validation)
go test ./code/go/...
```

### Adding Test Packages

When adding new features to the specification, you should create test packages under the
`test/packages/` folder to validate the new behavior. Each test package must have at least:

```
test/packages/my_package/
├── manifest.yml          # Package manifest
├── changelog.yml         # Version changelog
└── docs/
    └── README.md         # Documentation
```

#### Using elastic-package Tool (Recommended)

If the `elastic-package` tool is available, you can use it to scaffold test packages interactively:

```bash
cd test/packages
elastic-package create package
```

The tool will prompt for package details. After creation, you may need to adjust the `format_version`
or any other content that you need for your test case.

#### Manual Creation

Alternatively, you can manually create the directory structure and files following the structure
of existing test packages.

After creating test packages (by either method), remember to add corresponding test cases in
`code/go/pkg/validator/validator_test.go`.

#### Manifest Requirements

For packages using spec version 3.6.0+, the manifest should follow this structure:

```yaml
format_version: 3.6.0      # Use the spec version you're targeting
name: my_package            # Package name (lowercase letters, numbers, underscores)
title: My Package           # Display title
description: Description    # Full description
version: 0.0.1              # Use 0.0.1 for test packages to avoid version warnings
type: integration           # Package type (integration, input, or content)
source:
  license: "Apache-2.0"     # License identifier
conditions:
  kibana:                   # Note: nested structure for spec 3.0+
    version: '^8.0.0'
owner:
  github: elastic/foobar    # GitHub owner
  type: community           # Owner type: elastic, partner, or community
```

### Version Patches

When adding new features to the specification, you must ensure backward compatibility by adding
version patches. Version patches remove new features from older spec versions, allowing packages
with older `format_version` values to continue validating correctly.

Version patches are defined at the bottom of spec files (e.g., `spec/integration/manifest.spec.yml`):

```yaml
versions:
  - before: 3.6.0
    patch:
      - op: remove
        path: "/properties/my_new_field"
      - op: remove
        path: "/definitions/my_new_definition"
```

Guidelines:
- Add patches at the top of the versions list (newer versions first).
- Remove both the property and its definition(s) if they are not used elsewhere.
- Order matters: remove properties before the definitions they reference.
- Only remove definitions that are not used by other features.

### Testing with integrations repository

While working on a new branch, it is interesting to test these changes
with all the packages defined in the [integrations repository](https://github.com/elastic/integrations).
This allows to test a much wider scenarios than the test packages that are defined in this repository.

This process can also be done automatically from your Pull Request by adding a comment `test integrations`. Example:
- Comment: https://github.com/elastic/package-spec/pull/540#issuecomment-1593491304
- Pull Request created in integrations repository: https://github.com/elastic/integrations/pull/6587

This comment triggers this [Buildkite pipeline](https://github.com/elastic/package-spec/blob/72f19e94c61cc5c590aeefbeddfa025a95025b4e/.buildkite/pipeline.test-with-integrations-repo.yml) ([Buildkite job](https://buildkite.com/elastic/package-spec-test-with-integrations))

This pipeline creates a new draft Pull Request in integration updating the required dependencies to test your own changes. As a new pull request is created, a CI
job will be triggered to test all the packages defined in this repository. A new comment with the link to this new Pull Request will be posted in your package-spec Pull Request.

**IMPORTANT**: Remember to close this PR in the integrations repository once you close the package-spec Pull Request.

Usually, this process would require the following manual steps:
1. Create your package-spec pull request and push all your commits
2. Get the SHA of the latest changeset of your PR:
   ```bash
    $ git show -s --pretty=format:%H
   a86c0814e30b6a9dede26889a67e7df1bf827357
   ```
3. Go to your clone of the [integrations repository](https://github.com/elastic/integrations), and update go.mod and go.sum with that changeset:
   ```bash
   cd /path/to/integrations/repostiory
   go mod edit -replace github.com/elastic/package-spec/v3=github.com/<your_github_user>/package-spec/v3@a86c0814e30b6a9dede26889a67e7df1bf827357
   go mod tidy
   ```
4. Push these changes into a branch and create a Pull Request
    - Creating this PR would automatically trigger a new Jenkins pipeline.

