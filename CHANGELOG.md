# Changelog

## [0.4.1](https://github.com/atinylittleshell/gsh/compare/v0.4.0...v0.4.1) (2025-01-03)


### Bug Fixes

* fix release pipeline ([b70b29c](https://github.com/atinylittleshell/gsh/commit/b70b29c3366455e6dfedb0537f7128f34b4c9221))

## [0.4.0](https://github.com/atinylittleshell/gsh/compare/v0.3.0...v0.4.0) (2025-01-03)


### Features

* add configurable minimum shell prompt height\n\nIntroduced a new environment variable GSH_MINIMUM_HEIGHT to configure the minimum number of lines the shell prompt occupies. Updated the shell and environment components to utilize this new configuration. ([5aa0abc](https://github.com/atinylittleshell/gsh/commit/5aa0abc77718705d1aa64b4c92ec5f21407558bb))
* allow backspace to clear prediction at empty input ([428330a](https://github.com/atinylittleshell/gsh/commit/428330a52e746e6cc5bc0c54ccdcb2bb57a9e7fb))
* attemp to produce homebrew tap ([983197b](https://github.com/atinylittleshell/gsh/commit/983197b45824ebcd0cf348c6d3923018a6383e84))
* enhance shell prompt and command execution tracking\n\nUpdated .gshrc.starship for richer prompt details including command status and duration. Improved command execution tracking in shell.go and bash.go with duration and exit code handling. ([e16cb84](https://github.com/atinylittleshell/gsh/commit/e16cb8489d9465e601fa6bfbbab3f6d51f2e343a))


### Bug Fixes

* read /etc/profile as login shell ([33370d0](https://github.com/atinylittleshell/gsh/commit/33370d09d7e9972a6c264668c02d991de48853f1))
* update shell.go to improve command execution handling ([727f904](https://github.com/atinylittleshell/gsh/commit/727f9049d4fa780f56a31bbe396b47e8128045eb))

## [0.3.0](https://github.com/atinylittleshell/gsh/compare/v0.2.0...v0.3.0) (2025-01-02)


### Features

* add help flag to main command\n\nAdded a help flag (-h) to the main command to display usage information. Updated ROADMAP.md to reflect the reordering of tasks. ([452e017](https://github.com/atinylittleshell/gsh/commit/452e01720f30a03b0707a1464059951acaa067f6))
* **agent:** add preview code edits feature\n\nImplemented a feature to preview code edits before applying them. Updated ROADMAP.md to reflect the completion of this task. ([826cd9e](https://github.com/atinylittleshell/gsh/commit/826cd9efb5ecb0588a88e233d31e4586180161d1))
* **core:** add system info retriever and update roadmap\n\nAdded a new SystemInfoContextRetriever to the shell core for retrieving system information. Updated the ROADMAP.md to reflect recent changes and future plans. ([959f80f](https://github.com/atinylittleshell/gsh/commit/959f80f149a8ea567d6a6a44b245488a88921fb8))
* implement message pruning for agent chat\n\nAdded a new function to prune agent messages based on a context window size defined by GSH_AGENT_CONTEXT_WINDOW_TOKENS. Updated .gshrc.default and added tests for the new functionality. ([909bb46](https://github.com/atinylittleshell/gsh/commit/909bb460f4ce8e9a68633d3e356630880ea86910))


### Bug Fixes

* set SHELL environment variable correctly ([88a2471](https://github.com/atinylittleshell/gsh/commit/88a2471656f3fa878940b0d66d4794c3f4312024))

## [0.2.0](https://github.com/atinylittleshell/gsh/compare/v0.1.0...v0.2.0) (2025-01-01)


### Features

* **agent:** integrate history manager into agent and bash tool ([012e4a3](https://github.com/atinylittleshell/gsh/commit/012e4a3b68c19bb132bba0e905ba8acedaff4d5f))


### Bug Fixes

* correctly clear preview after command execution ([8c0642e](https://github.com/atinylittleshell/gsh/commit/8c0642e9264807a442860e3e93f27da7cdd06d8d))
* improve user confirmation handling in tools ([7a11590](https://github.com/atinylittleshell/gsh/commit/7a115909695ee9ce0485a4c1ff7a57dd21bd2f44))
* update .gshrc.starship configuration ([9de5ec6](https://github.com/atinylittleshell/gsh/commit/9de5ec6c5df3ab25e3602619b15047bd870c2fd4))

## [0.1.0](https://github.com/atinylittleshell/gsh/compare/v0.0.1...v0.1.0) (2024-12-31)


### Features

* explain prediction ([d7feb37](https://github.com/atinylittleshell/gsh/commit/d7feb3767dd7e010253a1e715773e1cb996a857e))

## 0.0.1 (2024-12-31)
