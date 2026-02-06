# AGENTS.md - Knowledge Base for AI Agents

This document contains important information learned while working on the package-spec repository to help future AI agents working on this codebase.

## Repository Structure

```
package-spec/
├── spec/                         # Package specifications
│   ├── integration/              # Integration package specs
│   │   └── manifest.spec.yml     # Main integration manifest spec (shared definitions go here)
│   ├── input/                    # Input package specs
│   ├── content/                  # Content package specs
│   └── changelog.yml             # Spec version changelog
├── code/go/                      # Go implementation
│   ├── internal/validator/semantic/  # Custom validation logic beyond JSON schema
│   └── pkg/validator/
│       └── validator_test.go     # Test cases for validation
├── test/packages/                # Test packages for validation
└── Makefile                      # Build and test commands
```

## Spec File Structure

The spec files use JSON Schema (https://json-schema.org/) written in YAML for readability:

```yaml
spec:
  type: object
  definitions:
    my_definition:
      description: Reusable definition
      type: string
  properties:
    my_property:
      $ref: "#/definitions/my_definition"
```

### Key Concepts

1. **Definitions**: Reusable schema components in `definitions` section
2. **Properties**: Fields referencing definitions via `$ref`
3. **Cross-file References**: Use relative paths (e.g., `$ref: "../integration/manifest.spec.yml#/definitions/my_field"`)
4. **AdditionalProperties**: Set to `false` to disallow undeclared fields

### Sharing Definitions

When a field is used in multiple locations, define it once in `spec/integration/manifest.spec.yml` and reference it:

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

**Important**: Remove property references before the definitions they depend on.

## Test Packages

### Creating Test Packages

⚠️ **Always use elastic-package when available!**

```bash
cd test/packages
elastic-package create package
```

The tool prompts for: package type, name, title, description, license (Apache-2.0), Kibana version (^8.0.0), subscription (basic), owner (elastic/foobar), and categories.

**After creation**: Adjust `format_version` and modify manifest.yml to test specific fields.

### Required Files

```
test/packages/my_package/
├── manifest.yml          # Package manifest
├── changelog.yml         # Version changelog
└── docs/
    └── README.md         # Documentation
```

**For data streams**, include valid ingest pipelines:
- Processors must have `tag` field
- on_failure must set `event.kind` to `pipeline_error`
- on_failure error.message must include `_ingest.on_failure_processor_type`, `_ingest.on_failure_processor_tag`, and `_ingest.pipeline`

Example pipeline:
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

### Test Case Format (validator_test.go)

```go
tests := map[string]struct {
    invalidPkgFilePath  string
    expectedErrContains []string
}{
    "good_package": {},  // Valid package, no errors expected
    "bad_package": {
        "manifest.yml",
        []string{`field my_field: validation error message`},
    },
}
```

**Note**: Use tabs for indentation in Go files.

## Testing Commands

```bash
# Validate spec files (DO THIS FIRST after any spec changes!)
go test ./code/go/internal

# Run specific test
go test -v -run "TestValidateFile/my_test" ./code/go/pkg/validator/...

# Run all tests
go test ./code/go/...

# Add license headers to Go files
make -C code/go update

# Format Go files
gofmt -w code/go/
```

## Changelog Management

Location: `spec/changelog.yml`

```yaml
- version: 3.6.0-next
  changes:
    - description: Brief description of the change.
      type: enhancement|breaking-change|bugfix
      link: https://github.com/elastic/package-spec/pull/NUMBER
```

Add new entries at the BOTTOM of the current version's changes list. Use "TBD" for PR link if not yet created.

## Common Pitfalls

1. **Forgetting version patches**: New features must be removed for older versions
2. **Not using shared definitions**: Define common fields once and reference with `$ref`
3. **Wrong patch order**: Remove property references before shared definitions
4. **Missing test package files**: changelog.yml and docs/README.md are required
5. **Invalid pipelines**: Data streams need proper ingest pipelines with on_failure handlers and processor tags
6. **Wrong conditions format**: Use `conditions.kibana.version` not `conditions.kibana:version` for spec 3.0+
7. **Creating test packages manually**: Always use `elastic-package create package`

## Example: Adding a New Field

1. **Add definition** in `spec/integration/manifest.spec.yml` (if used in multiple places):
   ```yaml
   definitions:
     my_field:
       description: Description
       type: string
   ```

2. **Reference it** where needed:
   ```yaml
   properties:
     my_field:
       $ref: "#/definitions/my_field"  # same file
       # or: $ref: "../integration/manifest.spec.yml#/definitions/my_field"
   ```

3. **Add version patch** (remove references first, then definition):
   ```yaml
   versions:
     - before: 3.X.0
       patch:
         - op: remove
           path: "/properties/my_field"
         - op: remove
           path: "/definitions/my_field"
   ```

4. **Check for custom validation** in `code/go/internal/validator/semantic/`

5. **Create test packages**: `elastic-package create package`, then modify manifest

6. **Add test cases** in `validator_test.go`

7. **Update changelog** in `spec/changelog.yml`

8. **Run tests**: `go test ./code/go/internal` then `go test ./code/go/...`

## Key Files

- `spec/integration/manifest.spec.yml` - Main spec with shared definitions
- `spec/changelog.yml` - Spec version changelog
- `code/go/pkg/validator/validator_test.go` - Test case definitions
- `code/go/internal/validator/semantic/` - Custom validation logic
- `test/packages/` - Test package examples

## Notes

- Spec versions with `-next` suffix are in development (3.6.0-next)
- Use `-count=1` flag to bypass test cache
- Custom validation logic exists beyond JSON schema in `code/go/internal/validator/semantic/`
- For real package examples, see [elastic/integrations](https://github.com/elastic/integrations)
