# Changelog

## [0.9.1](https://github.com/atinylittleshell/gsh/compare/v0.9.0...v0.9.1) (2025-01-11)


### Bug Fixes

* update default context types for explanation ([dc7d1d1](https://github.com/atinylittleshell/gsh/commit/dc7d1d1ac69d142a2e1009803575b55f30136ba2))

## [0.9.0](https://github.com/atinylittleshell/gsh/compare/v0.8.1...v0.9.0) (2025-01-11)


### Features

* make rag context types configurable ([11b8036](https://github.com/atinylittleshell/gsh/commit/11b8036e373ba0962b60e0506bd47c901dabb231))

## [0.8.1](https://github.com/atinylittleshell/gsh/compare/v0.8.0...v0.8.1) (2025-01-10)


### Bug Fixes

* standardize output formatting across tools using gline.RESET_CURSOR_COLUMN ([0249a45](https://github.com/atinylittleshell/gsh/commit/0249a456c606d595c33c513694c526c4deeabccf))

## [0.8.0](https://github.com/atinylittleshell/gsh/compare/v0.7.3...v0.8.0) (2025-01-08)


### Features

* enhance shell input with history navigation and improved utils ([d066369](https://github.com/atinylittleshell/gsh/commit/d066369b56a9a9ba1e1d761a0b015d88202c7005))

## [0.7.3](https://github.com/atinylittleshell/gsh/compare/v0.7.2...v0.7.3) (2025-01-07)


### Bug Fixes

* ensure relative paths are resolved to absolute paths in tools ([12c64e4](https://github.com/atinylittleshell/gsh/commit/12c64e431ebc691292a4d5fd5ad05362a7afacdb))

## [0.7.2](https://github.com/atinylittleshell/gsh/compare/v0.7.1...v0.7.2) (2025-01-07)


### Bug Fixes

* avoid structured output for predict and explain ([6ab1825](https://github.com/atinylittleshell/gsh/commit/6ab1825835a3a055d17906d4bafcf4028107d8a1))
* disable parallel tool calls for agent chat ([ac04d32](https://github.com/atinylittleshell/gsh/commit/ac04d3235c2eb36ba62a3d07bf22b31c9351f7da))

## [0.7.1](https://github.com/atinylittleshell/gsh/compare/v0.7.0...v0.7.1) (2025-01-06)


### Bug Fixes

* put context into agent system message ([0f28fc3](https://github.com/atinylittleshell/gsh/commit/0f28fc374bc166c3c3874853c41a39280e1c2606))
* tweak agent instructions to emphasize understanding diff before composing commit messages ([0314b67](https://github.com/atinylittleshell/gsh/commit/0314b67afdeea149f1430a383354bf5578037fa5))

## [0.7.0](https://github.com/atinylittleshell/gsh/compare/v0.6.0...v0.7.0) (2025-01-05)


### Features

* enhance LLM client with special headers for openrouter.ai and update roadmap ([290d0fc](https://github.com/atinylittleshell/gsh/commit/290d0fc2a15619a7ec8aca7b75422549891e7503))

## [0.6.0](https://github.com/atinylittleshell/gsh/compare/v0.5.4...v0.6.0) (2025-01-04)


### Features

* release to AUR ([ec9cf1c](https://github.com/atinylittleshell/gsh/commit/ec9cf1c0915811584748c2347c1d2cfb331a1a12))

## [0.5.4](https://github.com/atinylittleshell/gsh/compare/v0.5.3...v0.5.4) (2025-01-04)


### Bug Fixes

* log command line args ([03d6e3a](https://github.com/atinylittleshell/gsh/commit/03d6e3a32eac22a252870788481c2803b93ea7d8))

## [0.5.3](https://github.com/atinylittleshell/gsh/compare/v0.5.2...v0.5.3) (2025-01-03)


### Bug Fixes

* change error logs to warnings in env.go\n\n- Updated error logs to warnings for parsing environment variables. ([7f0efa1](https://github.com/atinylittleshell/gsh/commit/7f0efa10d803f9a26285c9660795ca421233c6aa))

## [0.5.2](https://github.com/atinylittleshell/gsh/compare/v0.5.1...v0.5.2) (2025-01-03)


### Bug Fixes

* correct login shell profile paths ([84b9437](https://github.com/atinylittleshell/gsh/commit/84b9437a4b956158ed5feb155cc0cdcb269b043c))

## [0.5.1](https://github.com/atinylittleshell/gsh/compare/v0.5.0...v0.5.1) (2025-01-03)


### Bug Fixes

* always start output messages with gsh: ([a7e3331](https://github.com/atinylittleshell/gsh/commit/a7e33317a2fcd3f247be26238f0317211f2cd9d1))
* improve login shell detection ([6623ad6](https://github.com/atinylittleshell/gsh/commit/6623ad64d5fe483514a331625fa76aa785d67394))

## [0.5.0](https://github.com/atinylittleshell/gsh/compare/v0.4.2...v0.5.0) (2025-01-03)


### Features

* add -ver flag to display BuildVersion\n\n- Implemented a new command-line flag '-ver' to print the current BuildVersion. ([a452224](https://github.com/atinylittleshell/gsh/commit/a452224a1ccf8210648204f1361e4324c6a850c9))
* self update ([f863a39](https://github.com/atinylittleshell/gsh/commit/f863a39cb39175f04651ec96b11eb001df922713))

## [0.4.2](https://github.com/atinylittleshell/gsh/compare/v0.4.1...v0.4.2) (2025-01-03)


### Bug Fixes

* fix goreleaser pipeline ([2e9ae6d](https://github.com/atinylittleshell/gsh/commit/2e9ae6ddb714a7aa8944bc05f7916bd57d202d23))

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
