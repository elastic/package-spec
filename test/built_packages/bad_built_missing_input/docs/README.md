# Bad Built Package - Stream Missing `input:` Field

This fixture is intentionally invalid for build-mode validation tests (issue `#549`).
It models a built package where a data stream stream entry does not have the required
`input:` field materialized. Build mode rejects streams that are missing `input:`,
as they represent an incompletely materialised composable package.
