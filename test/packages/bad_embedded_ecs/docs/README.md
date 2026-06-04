# Bad ECS Embedded in Source

Test fixture: integration package containing `_embedded_ecs` keys in `dynamic_templates`.

These keys are auto-injected by `elastic-package` at build time when `import_mappings` is
enabled. In source packages they must not appear. This package is valid in legacy mode but
rejected in source mode by `ValidateNoEmbeddedEcsInDynamicTemplates`.
