# Bad Built Package - Stream Package Reference

This fixture is intentionally invalid for build-mode validation tests (issue `#549`).
It uses source-only `package:` references in stream definitions, which must be
rejected under `ModeBuild`.
