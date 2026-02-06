# AGENTS.md - Knowledge Base for AI Agents

This document contains important information learned while working on the package-spec repository to help future AI agents working on this codebase.

## Repository Structure

```
package-spec/
├── spec/                         # Package specifications
│   ├── integration/              # Integration package specs
│   │   └── manifest.spec.yml     # Main integration manifest spec
│   ├── input/                    # Input package specs
│   ├── content/                  # Content package specs
│   └── changelog.yml             # Spec version changelog
├── code/go/                      # Go implementation
│   └── pkg/validator/
│       └── validator_test.go     # Test cases for validation
├── test/packages/                # Test packages for validation
└── Makefile                      # Build and test commands
```

## Spec Validation Command

After making ANY changes to spec files or changelog, always run:

```bash
go test ./code/go/internal
```

## Spec File Structure

### JSON Schema in YAML Format

The spec files use JSON Schema (https://json-schema.org/) written in YAML for readability:

```yaml
spec:
  type: object
  definitions:
    # Reusable definitions here
    my_definition:
      type: object
      properties:
        field: ...
  properties:
    # Top-level properties reference definitions
    my_property:
      $ref: "#/definitions/my_definition"
```

### Key Concepts

1. **Definitions**: Reusable schema components defined in `definitions` section
2. **Properties**: Top-level fields in the manifest, often referencing definitions via `$ref`
3. **Patterns**: Use regex patterns for field validation (e.g., `'^[a-z0-9_]+$'`)
4. **AdditionalProperties**: Set to `false` to disallow undeclared fields
5. **DRY Principle**: Define reusable fields once in definitions and reference them with `$ref` across multiple locations
6. **Cross-file References**: Use relative paths for references across spec files (e.g., `$ref: "../integration/manifest.spec.yml#/definitions/my_field"`)

### Sharing Definitions Across Spec Files

When a field is used in multiple locations, define it once and reference it:

```yaml
# In spec/integration/manifest.spec.yml
definitions:
  my_common_field:
    description: A field used in multiple places
    type: string

# In spec/input/manifest.spec.yml
properties:
  field:
    $ref: "../integration/manifest.spec.yml#/definitions/my_common_field"

# In spec/integration/data_stream/manifest.spec.yml
properties:
  field:
    $ref: "../../integration/manifest.spec.yml#/definitions/my_common_field"
```

**Benefits**:
- Single source of truth for field definitions
- Consistent descriptions and validation across all uses
- Easier maintenance and updates

### Version Patches

Version patches enable backward compatibility by removing features from older spec versions:

```yaml
versions:
  - before: 3.6.0
    patch:
      - op: remove
        path: "/properties/my_new_field"
      - op: remove
        path: "/definitions/my_new_definition"
```

**Important**:
- Patches go at the TOP of the versions list (newer versions first)
- Remove both the property references AND shared definition(s) if they are not used elsewhere
- When using shared definitions via `$ref`, remove the definition only after removing all references to it
- Order matters: remove property references before the definitions they depend on
- For shared definitions used across files, ensure the definition is removed in the file where it's defined

## Test Packages

### Creating Test Packages

⚠️ **IMPORTANT: Always use elastic-package when available!**

The `elastic-package` tool should be your first choice for creating test packages. It ensures correct file structure and content.

**Using elastic-package (REQUIRED METHOD)**:

```bash
cd test/packages
elastic-package create package
```

The tool will prompt interactively for package details:
- Package type (input/integration/content)
- Package name, title, description
- License (Apache-2.0 for test packages)
- Kibana version constraint (use ^8.0.0)
- Subscription level (basic)
- Owner (elastic/foobar for test packages)
- Categories

**After creation**:
- Adjust `format_version` in manifest.yml if testing features requiring a specific spec version
- Modify the manifest to include the specific field(s) you're testing

**Manual creation (ONLY if elastic-package unavailable)**: Create the required files or copy them from existing packages in the "test/packages" directory. This method is error-prone and should be avoided.

### Required Files for Integration Packages

Every test package must have:
```
test/packages/my_package/
├── manifest.yml          # Package manifest
├── changelog.yml         # Version changelog
└── docs/
    └── README.md         # Documentation
```

**For integration packages with data streams**, also ensure:
- Valid ingest pipelines with proper error handling (on_failure handler)
- Pipeline processors must have tags (SVR00006)
- on_failure must set `event.kind` to `pipeline_error` (SVR00008)
- on_failure error.message must include `_ingest.on_failure_processor_type`, `_ingest.on_failure_processor_tag`, and `_ingest.pipeline` (SVR00009)

Example minimal pipeline:
```yaml
---
description: Pipeline description
processors:
  - set:
      tag: set_field_name
      field: my.field
      value: test
on_failure:
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

### Manifest Requirements for Spec 3.6.0+

```yaml
format_version: 3.6.0      # Use 3.6.0 for new features
name: my_package            # Package name (a-z0-9_)
title: My Package           # Display title
description: Description    # Full description
version: 0.0.1              # Use 0.0.1 for test packages
type: integration           # Package type
source:
  license: "Apache-2.0"     # License identifier
conditions:
  kibana:                   # Note: nested structure for 3.0+
    version: '^8.0.0'
owner:
  github: elastic/foobar    # GitHub owner
  type: community           # Owner type: elastic, partner, community
```

### Changelog Format

```yaml
- version: 0.0.1
  changes:
    - description: Initial release
      type: enhancement
      link: https://github.com/elastic/package-spec/pull/1
```

## Adding Test Cases

### In validator_test.go

Test cases are defined in the `TestValidateFile` function:

```go
tests := map[string]struct {
    invalidPkgFilePath  string
    expectedErrContains []string
}{
    "good_package": {},  // Valid package, no errors expected
    "bad_package": {
        "manifest.yml",  // File with error
        []string{
            `field my_field: validation error message`,
        },
    },
}
```

**Pattern for adding tests**:
1. **Create test package** using `elastic-package create package` (see "Creating Test Packages" section)
2. Modify the generated manifest.yml to include/test the specific field(s)
3. Add entry to validator_test.go map
4. For good packages: empty struct `{}`
5. For bad packages: specify file and expected error messages

**Note**: Use tabs for indentation in Go files, not spaces!

## Testing Workflow

### Validate Spec Files (Critical!)

**Always run this first** to validate spec and changelog files:

```bash
go test ./code/go/internal
```

This validates:
- Spec file syntax (manifest.spec.yml, etc.)
- Changelog format and structure
- Schema definitions and references
- JSON patches for version compatibility

### Run Package Validation Tests

To test specific package validation:

```bash
go test -v -run "TestValidateFile/my_test" ./code/go/pkg/validator/...
```

### Run All Tests

```bash
go test ./code/go/...
```

### Format Go files

Add license headers:
```bash
make -C code/go update
```

Format with `gofmt`.

## Changelog Management

### Location
`spec/changelog.yml` - Documents changes to the package specification itself.

### Format

```yaml
- version: 3.6.0-next
  changes:
    - description: Brief description of the change.
      type: enhancement|breaking-change|bugfix
      link: https://github.com/elastic/package-spec/pull/NUMBER
```

### Guidelines

1. Add new entries to the BOTTOM of the current version's changes list
2. Keep descriptions concise but clear
3. Use "TBD" for PR link if not yet created
4. Types: `enhancement`, `breaking-change`, `bugfix`

## Common Patterns

### Defining Reusable Types

```yaml
definitions:
  my_type:
    description: Description of the type
    type: object
    additionalProperties: false
    properties:
      name:
        type: string
        pattern: '^[a-z0-9_]+$'
      version:
        type: string
    required:
      - name
      - version
```

### Using References

```yaml
properties:
  my_array:
    type: array
    items:
      $ref: "#/definitions/my_type"
```

### Validation Patterns

- Package names: `'^[a-z0-9_]+$'`
- Semantic versions: `'^([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+)?$'`

## Common Pitfalls

1. **Forgetting version patches**: New features must be removed for older versions
2. **Not using shared definitions**: Define common fields once and reference with `$ref` instead of duplicating
3. **Wrong patch order for shared definitions**: Remove all property references before removing the shared definition
4. **Missing test package files**: changelog.yml and docs/README.md are required
5. **Invalid pipelines**: Integration packages with data streams need proper ingest pipelines with on_failure handlers
6. **Missing pipeline tags**: All processors in pipelines must have a `tag` field
7. **Wrong conditions format**: Use nested structure for spec 3.0+ (`conditions.kibana.version` not `conditions.kibana:version`)
8. **Test version mismatch**: Use 0.0.1 for test packages to avoid GA version warnings
9. **Changelog placement**: Add entries at the BOTTOM of the current version section
10. **Creating test packages manually**: Always use `elastic-package create package` when available

## Git Workflow

The repository uses feature branches:
```bash
git status                   # Check current changes
git diff --stat              # Summary of changes
git log --oneline -5         # Recent commits
```

## Key Files to Remember

- `spec/integration/manifest.spec.yml` - Integration package specification
- `spec/changelog.yml` - Spec version changelog
- `code/go/pkg/validator/validator_test.go` - Test case definitions
- `test/packages/` - Test package examples

## Real Package Examples

For real-world examples of integration packages, see the [elastic/integrations](https://github.com/elastic/integrations) repository. This repository contains hundreds of production integration packages that demonstrate:
- Complex manifest structures
- Real-world data stream configurations
- Production-ready ingest pipelines
- Dashboards and visualizations
- Complete package documentation

When implementing new spec features, it's helpful to:
1. Look at similar features in existing packages
2. Test your spec changes against real packages
3. Study package patterns used across different integrations

## Useful Commands

```bash
# Validate spec and changelog files (DO THIS FIRST!)
go test ./code/go/internal

# Run all tests
go test ./code/go/...

# Update spec
make -C ./code/go update

# Check package structure
ls -la test/packages/good/
```

## Example: Adding a New Field

1. **Decide where to define it**:
   - If used in only one location: define it inline where it's used
   - If used in multiple locations: define once in `spec/integration/manifest.spec.yml#/definitions` and reference with `$ref`

2. **Add definition** (for shared fields):
   ```yaml
   # In spec/integration/manifest.spec.yml
   definitions:
     my_field_type:
       description: Description of the field
       type: object
       properties: ...
   ```

3. **Add property** referencing the definition:
   ```yaml
   # In the same file
   properties:
     my_field:
       $ref: "#/definitions/my_field_type"

   # In a different file (adjust path as needed)
   properties:
     my_field:
       $ref: "../integration/manifest.spec.yml#/definitions/my_field_type"
   ```

4. **Add version patch** (if feature is version-gated):
   ```yaml
   versions:
     - before: 3.X.0
       patch:
         # Remove all property references first
         - op: remove
           path: "/properties/my_field"
         - op: remove
           path: "/other/location/properties/my_field"
         # Then remove the shared definition (only if not used elsewhere)
         - op: remove
           path: "/definitions/my_field_type"
   ```

5. **Update validation code if needed**: Check if there's custom validation logic in `code/go/internal/validator/semantic/` that needs updating

6. **Create test packages** using `elastic-package create package`:
   - `test/packages/good_my_field/` - Valid usage
   - `test/packages/bad_my_field/` - Invalid usage
   - After creation, modify manifest.yml to include/test the new field

7. **Add test cases** in `validator_test.go`

8. **Update changelog** in `spec/changelog.yml`

9. **Validate spec files**: `go test ./code/go/internal`

10. **Run package tests**: `go test ./code/go/pkg/validator/...`

## Notes

- Spec versions with the -next suffix are currently in development (3.6.0-next)
- All integration tests must pass before changes are accepted
- The project uses Go modules for dependency management
- Test packages should follow the same structure as real packages
- Use `-count=1` flag instead of `go clean -testcache` to bypass test cache
- When adding fields used in multiple places, use shared definitions via `$ref` to avoid duplication
- Integration packages require valid ingest pipelines with proper error handling
- Custom validation logic may exist in Go code (`code/go/internal/validator/semantic/`) beyond JSON schema validation
