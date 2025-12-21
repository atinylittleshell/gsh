# Changelog

## [0.28.0](https://github.com/robottwo/gsh_prime/compare/v0.27.0...v0.28.0) (2025-12-21)


### Features

* Add macOS DMG installers to release workflow ([#76](https://github.com/robottwo/gsh_prime/issues/76)) ([ef7ff20](https://github.com/robottwo/gsh_prime/commit/ef7ff2083e9b076ff4a7cf6574cc7619551b3638))
* Add Nix package building to release workflow ([#78](https://github.com/robottwo/gsh_prime/issues/78)) ([8dd2c33](https://github.com/robottwo/gsh_prime/commit/8dd2c330bc24c02d4399eadde18e743f3bc18209))


### Bug Fixes

* Rename FileExists to SourceFileExists to prevent infinite recursion ([#77](https://github.com/robottwo/gsh_prime/issues/77)) ([2c0852f](https://github.com/robottwo/gsh_prime/commit/2c0852faecac85df35b148381bdd95633c81a43e))

## [0.27.0](https://github.com/robottwo/gsh_prime/compare/v0.26.1...v0.27.0) (2025-12-20)


### Features

* add dynamic coaching tips based on user behavior ([#70](https://github.com/robottwo/gsh_prime/issues/70)) ([5a71bd3](https://github.com/robottwo/gsh_prime/commit/5a71bd3f9687853d6b5ee0eb7c582e5db815b53c))
* Add dynamic terminal title updates based on command history ([#73](https://github.com/robottwo/gsh_prime/issues/73)) ([ecf06fa](https://github.com/robottwo/gsh_prime/commit/ecf06fa1007d7c715f7ae8b48d8de33c9440f93c))
* Add idle summary feature for command prompt ([#69](https://github.com/robottwo/gsh_prime/issues/69)) ([2d8052e](https://github.com/robottwo/gsh_prime/commit/2d8052e715307a0ddf2daebbfd5da97c5716a43e))
* Right-align coach tips in assistant box ([#66](https://github.com/robottwo/gsh_prime/issues/66)) ([ee20bfd](https://github.com/robottwo/gsh_prime/commit/ee20bfd02450531078a95bacdee58a3b18b194f9))


### Bug Fixes

* Adjust tab completion to use common prefix ([#64](https://github.com/robottwo/gsh_prime/issues/64)) ([3c6b9aa](https://github.com/robottwo/gsh_prime/commit/3c6b9aa7abe39582db8ce885c52139a6f77e018b))
* Correct path handling for ./. and ~/. tab completions ([#72](https://github.com/robottwo/gsh_prime/issues/72)) ([3b43eee](https://github.com/robottwo/gsh_prime/commit/3b43eee2500d86f384bfefc0f5ffa7b9aaf434e2))
* Prevent LLM path hallucinations during completion ([#62](https://github.com/robottwo/gsh_prime/issues/62)) ([87d6e65](https://github.com/robottwo/gsh_prime/commit/87d6e65d3a2ccb3b6c245fdf2e66d77d010953cd))
* prevent shell exit on Ctrl+C ([#61](https://github.com/robottwo/gsh_prime/issues/61)) ([315f634](https://github.com/robottwo/gsh_prime/commit/315f63494c0f564d7663b6fa9cd6445eafdbfa2f))
* Treat variation selectors and zero-width chars as width 0 ([#74](https://github.com/robottwo/gsh_prime/issues/74)) ([a336d5e](https://github.com/robottwo/gsh_prime/commit/a336d5e96182c9a0a23ad112077e557555c44e1f))
* Use custom wordwrap with accurate Unicode width for coach tips ([#67](https://github.com/robottwo/gsh_prime/issues/67)) ([528ff15](https://github.com/robottwo/gsh_prime/commit/528ff158971506634507c7792ebbf386da209070))

## [0.26.1](https://github.com/robottwo/gsh_prime/compare/v0.26.0...v0.26.1) (2025-12-17)


### Features

* Claude/gamified productivity coach ([#58](https://github.com/robottwo/gsh_prime/issues/58)) ([ce7c3ff](https://github.com/robottwo/gsh_prime/commit/ce7c3ff4902f4d821bedfc80da31745f50e048ae))
* Assistant Box Border Status UI ([#57](https://github.com/robottwo/gsh_prime/issues/57)) ([137fad6](https://github.com/robottwo/gsh_prime/commit/137fad678624eda803b24372cee2cca03dd35b16))
* Add history search sort toggle (Ctrl+O) ([#56](https://github.com/robottwo/gsh_prime/issues/56)) ([45881a2](https://github.com/robottwo/gsh_prime/commit/45881a28a1be08651dd7111684328d14391b3fca))

### Bug Fixes

* Fix DST transitions and improve coach robustness ([#59](https://github.com/robottwo/gsh_prime/issues/59)) ([4e5e6e7](https://github.com/robottwo/gsh_prime/commit/4e5e6e79cef7eab83dadb9bc0c18b76ae37b4a9d))
* detect Unicode character width at runtime for Assistant Box ([#54](https://github.com/robottwo/gsh_prime/issues/54)) ([bd1ed47](https://github.com/robottwo/gsh_prime/commit/bd1ed479483f97176cddfcfb7e6521f14f3777f5))
* fix provider integration test ([#53](https://github.com/robottwo/gsh_prime/issues/53)) ([bafae69](https://github.com/robottwo/gsh_prime/commit/bafae699c6ec93f500af0104a49f38efc32aac31))

## [0.26.0](https://github.com/robottwo/gsh_prime/compare/0.25.10...v0.26.0) (2025-12-01)


### Features

* Implement comprehensive subagent system with Claude and Roo Code compatibility ([2fb7a6e](https://github.com/atinylittleshell/gsh/commit/2fb7a6e))
* Add support for Roo YAML mode files ([314941a](https://github.com/atinylittleshell/gsh/commit/314941a))
* Add interactive permissions menu system ([a5c7083](https://github.com/atinylittleshell/gsh/commit/a5c7083))
* Improve command regex generation with heuristic-based subcommand detection ([8beff60](https://github.com/atinylittleshell/gsh/commit/8beff60))
* Add context-sensitive completions for macro and builtin command prefixes ([57466dc](https://github.com/atinylittleshell/gsh/commit/57466dc))
* Improve Ctrl+C handling and interrupt detection in gline app ([cada44d](https://github.com/atinylittleshell/gsh/commit/cada44d))
* Add basic Nix flake support ([9cc2e03](https://github.com/atinylittleshell/gsh/commit/9cc2e03))
* add LLM loading indicators to Assistant Box ([#45](https://github.com/robottwo/gsh_prime/issues/45)) ([426a385](https://github.com/robottwo/gsh_prime/commit/426a3851f428b22b93942f6c2306abbffaec8689))
* add LLM-based tab completion ([#22](https://github.com/robottwo/gsh_prime/issues/22)) ([05511ae](https://github.com/robottwo/gsh_prime/commit/05511ae12f6c8637c8c7a49fecad3ab3ab27b6d8))
* **bash:** add typeset/declare command compatibility ([#10](https://github.com/robottwo/gsh_prime/issues/10)) ([15a8fe9](https://github.com/robottwo/gsh_prime/commit/15a8fe902357b5e15079eaa2e0429d012792bdfb))
* Improve agent output styling and order ([#26](https://github.com/robottwo/gsh_prime/issues/26)) ([535e324](https://github.com/robottwo/gsh_prime/commit/535e32476394f2ce80831622fa5d915d6fb95758))
* improve subagent syntax and prompt feedback ([#18](https://github.com/robottwo/gsh_prime/issues/18)) ([eb40d7f](https://github.com/robottwo/gsh_prime/commit/eb40d7fd4edb4d0f8ac5397d7b94a3a0f73ac174))
* rich TUI history search ([#43](https://github.com/robottwo/gsh_prime/issues/43)) ([02b2fb0](https://github.com/robottwo/gsh_prime/commit/02b2fb051cf184e38cb98c5640c65ebb1d0164a6))

### Bug Fixes

* Fix test dependencies on the local system ([52efcf4](https://github.com/atinylittleshell/gsh/commit/52efcf4))
* Improve command regex generation and test reliability ([d01934f](https://github.com/atinylittleshell/gsh/commit/d01934f))
* Remove manage permissions from file operations ([c7367ed](https://github.com/atinylittleshell/gsh/commit/c7367ed))
* Improve Ctrl+C handling and testability in completion system ([4a5afeb](https://github.com/atinylittleshell/gsh/commit/4a5afeb))

### Refactor

* Remove legacy 'always' workflow, consolidate on 'manage' menu system ([494a105](https://github.com/atinylittleshell/gsh/commit/494a105))
* Simplify user confirmation logic and remove retry mechanism ([8711cdd](https://github.com/atinylittleshell/gsh/commit/8711cdd))

### Documentation

* Add comprehensive documentation suite ([fb8b7b8](https://github.com/atinylittleshell/gsh/commit/fb8b7b8))
* Move agent documentation to separate file ([172c350](https://github.com/atinylittleshell/gsh/commit/172c350))
* Remove COMMAND_REGEX_IMPROVEMENTS documentation ([a7d0bd7](https://github.com/atinylittleshell/gsh/commit/a7d0bd7))
* Update documentation structure and content ([e88ca9a](https://github.com/atinylittleshell/gsh/commit/e88ca9a), [97a466a](https://github.com/atinylittleshell/gsh/commit/97a466a), [74ead62](https://github.com/atinylittleshell/gsh/commit/74ead62), [1effd42](https://github.com/atinylittleshell/gsh/commit/1effd42), [60b6d66](https://github.com/atinylittleshell/gsh/commit/60b6d66), [e35fc4d](https://github.com/atinylittleshell/gsh/commit/e35fc4d), [8a42f0a](https://github.com/atinylittleshell/gsh/commit/8a42f0a), [c4a9fc2](https://github.com/atinylittleshell/gsh/commit/c4a9fc2), [cf5322a](https://github.com/atinylittleshell/gsh/commit/cf5322a), [e3a79c3](https://github.com/atinylittleshell/gsh/commit/e3a79c3))

## [0.22.2](https://github.com/atinylittleshell/gsh/compare/v0.22.1...v0.22.2) (2025-02-08)


### Bug Fixes

* allow nil temperature and parallel tool calls config ([0fbab29](https://github.com/atinylittleshell/gsh/commit/0fbab29345d049be0c18ffd300094d42940ff062))

## [0.22.1](https://github.com/atinylittleshell/gsh/compare/v0.22.0...v0.22.1) (2025-02-03)


### Bug Fixes

* force version update ([550a59c](https://github.com/atinylittleshell/gsh/commit/550a59c5485d761653bdd48878ccaf3bb01d5a65))

## [0.22.0](https://github.com/atinylittleshell/gsh/compare/v0.21.1...v0.22.0) (2025-02-03)


### Features

* add support for multiple iterations in model evaluation ([73f2941](https://github.com/atinylittleshell/gsh/commit/73f2941fe0bb19b0ae94cbede736c2206792abb7))

## [0.21.1](https://github.com/atinylittleshell/gsh/compare/v0.21.0...v0.21.1) (2025-02-03)


### Bug Fixes

* skip evaluation UI when not in terminal ([e733243](https://github.com/atinylittleshell/gsh/commit/e7332435c305497c22b3cc307c6dbd0cb2af77be))

## [0.21.0](https://github.com/atinylittleshell/gsh/compare/v0.20.1...v0.21.0) (2025-02-03)


### Features

* add command evaluation functionality ([dfdda6d](https://github.com/atinylittleshell/gsh/commit/dfdda6dcaed4acd10f3f4629f6689e8a4f0f700f))
* add total count support to analytics command ([7f6484f](https://github.com/atinylittleshell/gsh/commit/7f6484fce90c7da7cd0774c4f0b94825c3224b9d))

## [0.20.1](https://github.com/atinylittleshell/gsh/compare/v0.20.0...v0.20.1) (2025-02-02)


### Bug Fixes

* analytics file path and null pointer handling ([a8ab90b](https://github.com/atinylittleshell/gsh/commit/a8ab90bf992eadb9987cc514cb74ea8e25bc4cc7))

## [0.20.0](https://github.com/atinylittleshell/gsh/compare/v0.19.4...v0.20.0) (2025-02-01)


### Features

* track prediction history locally ([f1a0c89](https://github.com/atinylittleshell/gsh/commit/f1a0c89ac8ed3896240514daaf581db11faae692))

## [0.19.4](https://github.com/atinylittleshell/gsh/compare/v0.19.3...v0.19.4) (2025-02-01)


### Bug Fixes

* improve file completion to handle home dir, absolute, and relative paths consistently ([0283834](https://github.com/atinylittleshell/gsh/commit/02838348152710f726afc8fc2139454c02999fb7))

## [0.19.3](https://github.com/atinylittleshell/gsh/compare/v0.19.2...v0.19.3) (2025-01-29)


### Bug Fixes

* improve file completion to preserve path prefix and earlier arguments ([6d107cb](https://github.com/atinylittleshell/gsh/commit/6d107cbdbb0b0a196da058c8074870003d73282b))

## [0.19.2](https://github.com/atinylittleshell/gsh/compare/v0.19.1...v0.19.2) (2025-01-27)


### Bug Fixes

* improve message pruning to keep early and recent context ([777be1f](https://github.com/atinylittleshell/gsh/commit/777be1f8b677e9a84f0f37620e688a631d3dadf3))

## [0.19.1](https://github.com/atinylittleshell/gsh/compare/v0.19.0...v0.19.1) (2025-01-27)


### Bug Fixes

* improve file diff preview when creating existing files ([1e865f9](https://github.com/atinylittleshell/gsh/commit/1e865f9c7c01fe6c3db74043dba12a995075bdba))

## [0.19.0](https://github.com/atinylittleshell/gsh/compare/v0.18.2...v0.19.0) (2025-01-26)


### Features

* add history-based command prefix prediction ([d6b7e83](https://github.com/atinylittleshell/gsh/commit/d6b7e838654ddcb5e5d1b14c05d7a21be266db74))

## [0.18.2](https://github.com/atinylittleshell/gsh/compare/v0.18.1...v0.18.2) (2025-01-26)


### Bug Fixes

* remove source files from gsh-bin aur release script ([1f42fb2](https://github.com/atinylittleshell/gsh/commit/1f42fb2c429eead0175965fd37a5d9156defb156))

## [0.18.1](https://github.com/atinylittleshell/gsh/compare/v0.18.0...v0.18.1) (2025-01-26)


### Bug Fixes

* improved rendering for token stats ([d513e49](https://github.com/atinylittleshell/gsh/commit/d513e494d0efa683edcced9bb6a9fd2ce52906f1))

## [0.18.0](https://github.com/atinylittleshell/gsh/compare/v0.17.0...v0.18.0) (2025-01-26)


### Features

* implement agent controls and add token tracking ([6793455](https://github.com/atinylittleshell/gsh/commit/679345538a793505b96cf809a39ba24abc8cfa01))

## [0.17.0](https://github.com/atinylittleshell/gsh/compare/v0.16.1...v0.17.0) (2025-01-26)


### Features

* add compgen command implementation and tests ([0f4f983](https://github.com/atinylittleshell/gsh/commit/0f4f983a81b922cc8eaf541cb3cc8a908ed3d367))
* basic bash completion support ([f2b4f19](https://github.com/atinylittleshell/gsh/commit/f2b4f1986e9eb04cd3914e535246547494b94a03))
* basic support for complete command ([6f9d1a8](https://github.com/atinylittleshell/gsh/commit/6f9d1a82e0520e32cb6d26a848a1dc601ae00908))

## [0.16.1](https://github.com/atinylittleshell/gsh/compare/v0.16.0...v0.16.1) (2025-01-25)


### Bug Fixes

* optimize verbose history format by grouping by directory ([a49432f](https://github.com/atinylittleshell/gsh/commit/a49432f17a4166067d2c608a7be574b9e7d81854))

## [0.16.0](https://github.com/atinylittleshell/gsh/compare/v0.15.9...v0.16.0) (2025-01-21)


### Features

* add support for chat macros ([527b2c1](https://github.com/atinylittleshell/gsh/commit/527b2c106dc6fc134918c68823c0372a8852f268))

## [0.15.9](https://github.com/atinylittleshell/gsh/compare/v0.15.8...v0.15.9) (2025-01-21)


### Bug Fixes

* continue fixing aur sources release ([b42ee5c](https://github.com/atinylittleshell/gsh/commit/b42ee5c240e7dac2e69ced66f162da3d9e13b749))

## [0.15.8](https://github.com/atinylittleshell/gsh/compare/v0.15.7...v0.15.8) (2025-01-21)


### Bug Fixes

* continue fixing aur sources release ([eb17ad3](https://github.com/atinylittleshell/gsh/commit/eb17ad3b9355eff4fa41ba473d5ad1897768f8bd))

## [0.15.7](https://github.com/atinylittleshell/gsh/compare/v0.15.6...v0.15.7) (2025-01-21)


### Bug Fixes

* continue fixing aur sources release ([3b08fdf](https://github.com/atinylittleshell/gsh/commit/3b08fdfe688806b33b91accc8dd496c48c9f3d7a))

## [0.15.6](https://github.com/atinylittleshell/gsh/compare/v0.15.5...v0.15.6) (2025-01-21)


### Bug Fixes

* continue fixing aur sources release ([648b11a](https://github.com/atinylittleshell/gsh/commit/648b11a2783566adcfdea90ae33d0c86ef012351))

## [0.15.5](https://github.com/atinylittleshell/gsh/compare/v0.15.4...v0.15.5) (2025-01-21)


### Bug Fixes

* force new release ([e6aefd3](https://github.com/atinylittleshell/gsh/commit/e6aefd3111354691f0d57ca3eef3c11ff1ec2307))

## [0.15.4](https://github.com/atinylittleshell/gsh/compare/v0.15.3...v0.15.4) (2025-01-21)


### Bug Fixes

* attempt to fir aur sources release ([762c1b5](https://github.com/atinylittleshell/gsh/commit/762c1b5db76c83899750709c71ab4dc1498714cc))

## [0.15.3](https://github.com/atinylittleshell/gsh/compare/v0.15.2...v0.15.3) (2025-01-21)


### Bug Fixes

* fix homebrew tap formula release ([2e6106c](https://github.com/atinylittleshell/gsh/commit/2e6106ce6ac9ebd26eee4b92367c7a15f8e90d6b))

## [0.15.2](https://github.com/atinylittleshell/gsh/compare/v0.15.1...v0.15.2) (2025-01-21)


### Bug Fixes

* try fixing goreleaser config ([e17024a](https://github.com/atinylittleshell/gsh/commit/e17024af199bc9db91377076ab696292e0a98c44))

## [0.15.1](https://github.com/atinylittleshell/gsh/compare/v0.15.0...v0.15.1) (2025-01-21)


### Bug Fixes

* add source archive to goreleaser ([08c3c8c](https://github.com/atinylittleshell/gsh/commit/08c3c8c7990a622ba07aeb5e81179fd720448d54))

## [0.15.0](https://github.com/atinylittleshell/gsh/compare/v0.14.0...v0.15.0) (2025-01-20)


### Features

* release aur sources ([af0aeb7](https://github.com/atinylittleshell/gsh/commit/af0aeb7bea9e6873cd855559419020cdb4a0cd48))

## [0.14.0](https://github.com/atinylittleshell/gsh/compare/v0.13.2...v0.14.0) (2025-01-20)


### Features

* improve pre-approved command patterns ([8c60662](https://github.com/atinylittleshell/gsh/commit/8c60662ac66bba90364d29c01b788eb396bad461))

## [0.13.2](https://github.com/atinylittleshell/gsh/compare/v0.13.1...v0.13.2) (2025-01-20)


### Bug Fixes

* make bash output buffer thread-safe ([4ab74ab](https://github.com/atinylittleshell/gsh/commit/4ab74ab747e4e1f6e058135d5f7e285dc7838459))

## [0.13.1](https://github.com/atinylittleshell/gsh/compare/v0.13.0...v0.13.1) (2025-01-20)


### Bug Fixes

* rollback shellopts change ([961720a](https://github.com/atinylittleshell/gsh/commit/961720acdeaf1103af986a45e988d4c270b1b2ab))

## [0.13.0](https://github.com/atinylittleshell/gsh/compare/v0.12.0...v0.13.0) (2025-01-19)


### Features

* add more read-only commands to pre-approved list ([c05851c](https://github.com/atinylittleshell/gsh/commit/c05851cec47d95fb9cfdc7eb54aca18af9d45b30))
* support history built-in command ([c9710cb](https://github.com/atinylittleshell/gsh/commit/c9710cb3407727e7507a76391180de0544b28fc1))
* support shell opts ([d335891](https://github.com/atinylittleshell/gsh/commit/d33589131ce7a38dacd0f88de8911bd1768a7f56))

## [0.12.0](https://github.com/atinylittleshell/gsh/compare/v0.11.3...v0.12.0) (2025-01-18)


### Features

* show build version in prompt when in dev mode ([bad7a27](https://github.com/atinylittleshell/gsh/commit/bad7a270561563fdb98d3136992300f792846a9f))


### Bug Fixes

* replace charmbracelet/x/ansi with muesli/reflow for text wrapping ([aaf4fe5](https://github.com/atinylittleshell/gsh/commit/aaf4fe5f1853f21309b663cd8af7e34f3ab49d90))

## [0.11.3](https://github.com/atinylittleshell/gsh/compare/v0.11.2...v0.11.3) (2025-01-18)


### Bug Fixes

* fix rendering for multi-line agent response ([cef8dc5](https://github.com/atinylittleshell/gsh/commit/cef8dc51f332eabac8fbd371b105c2b2aa9f7a15))

## [0.11.2](https://github.com/atinylittleshell/gsh/compare/v0.11.1...v0.11.2) (2025-01-18)


### Bug Fixes

* removed chain of thought when doing prediction to reduce token usage and improve latency ([1f5ee65](https://github.com/atinylittleshell/gsh/commit/1f5ee654b6e266225b0b40d72cc0b65a383ee99a))

## [0.11.1](https://github.com/atinylittleshell/gsh/compare/v0.11.0...v0.11.1) (2025-01-18)


### Bug Fixes

* force a new version release ([6446be4](https://github.com/atinylittleshell/gsh/commit/6446be4a47f0e706e016e6988d261e9cbc879b69))

## [0.11.0](https://github.com/atinylittleshell/gsh/compare/v0.10.0...v0.11.0) (2025-01-15)


### Features

* add interrupt handling for chat sessions ([657c27c](https://github.com/atinylittleshell/gsh/commit/657c27cba094bf6fd061965a1bfa017db351f961))

## [0.10.0](https://github.com/atinylittleshell/gsh/compare/v0.9.4...v0.10.0) (2025-01-15)


### Features

* add pre-approved command patterns to skip confirmation ([a765e43](https://github.com/atinylittleshell/gsh/commit/a765e43164dc164f31d636e67c591ce88d556a2d))


### Bug Fixes

* improve privacy for printed diffs ([6545c29](https://github.com/atinylittleshell/gsh/commit/6545c2936041e9533fd6cdd835a7c417b544e1e1))

## [0.9.4](https://github.com/atinylittleshell/gsh/compare/v0.9.3...v0.9.4) (2025-01-15)


### Bug Fixes

* improve file creation confirmation UI ([1cded2c](https://github.com/atinylittleshell/gsh/commit/1cded2c011102f868f1ac7d06627419db2f1836e))
* improve file edit confirmation UI ([d850c97](https://github.com/atinylittleshell/gsh/commit/d850c97898acc5a5ab5dcec90a315ac5c725b1ca))

## [0.9.3](https://github.com/atinylittleshell/gsh/compare/v0.9.2...v0.9.3) (2025-01-13)


### Bug Fixes

* remove done tool and add end_turn finish reason ([c5819f4](https://github.com/atinylittleshell/gsh/commit/c5819f4eb9fb7420c753fd75263219dac15281ff))
* tweak agent prompts ([0ef096c](https://github.com/atinylittleshell/gsh/commit/0ef096cf5eee74021f1bed28d7c2d158b3a289bb))

## [0.9.2](https://github.com/atinylittleshell/gsh/compare/v0.9.1...v0.9.2) (2025-01-13)


### Bug Fixes

* improve privacy of printed paths ([309f2a6](https://github.com/atinylittleshell/gsh/commit/309f2a6de234065f419bed131664f256fa5acf54))
* improve viewfile tool default behavior ([ef45e89](https://github.com/atinylittleshell/gsh/commit/ef45e89a1867b1cf6a4ad17dbc3f9606f20d9fd1))
* update JSON schema descriptions and add unit test for bash tool ([23caedb](https://github.com/atinylittleshell/gsh/commit/23caedb5dd43df9b0f15920bf9015722db76c702))
* **viewfile:** adjust line indexing to be 1-based and handle edge cases ([a4c535e](https://github.com/atinylittleshell/gsh/commit/a4c535e96965101010b9dc0ccd3fc57767b39a1b))

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
