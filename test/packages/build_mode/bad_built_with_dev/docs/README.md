# Bad Built Package - With Dev Directory

This fixture contains a `_dev/` directory, which is a source-only artifact that
must be rejected in build mode (issue `#549`). Used to verify that `ModeBuild`
correctly flags packages containing `_dev/` directories.
