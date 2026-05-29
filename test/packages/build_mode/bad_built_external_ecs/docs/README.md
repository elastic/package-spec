# Bad Built Package - External ECS

This test fixture contains `external: ecs` field references that must be rejected
in build mode (issue `#549`). Built packages must materialize all ECS fields rather
than reference them via `external: ecs`.
