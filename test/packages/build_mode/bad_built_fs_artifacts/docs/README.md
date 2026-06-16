# Bad Built Package - Source-only FS Artifacts

This fixture contains both a `_dev/` directory and a `.link` file — both
source-only artifacts that must be rejected in build mode (issue `#549`).
Used to verify that `BuildMode` correctly flags packages containing either
`_dev/` directories or `.link` files.
