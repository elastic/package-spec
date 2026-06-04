# Bad Built Package - Link File Present

This built-package fixture contains `.link` files and is invalid in build mode
(issue `#549`). Build mode rejects packages with `.link` files, as they are
source-only artifacts that must be resolved during the build process.
