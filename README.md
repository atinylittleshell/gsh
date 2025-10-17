# gsh_prime

[![License](https://img.shields.io/github/license/robottwo/gsh_prime.svg)](https://github.com/robottwo/gsh_prime/blob/main/LICENSE)
[![Release](https://img.shields.io/github/release/robottwo/gsh_prime.svg)](https://github.com/robottwo/gsh_prime/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/robottwo/gsh_prime/ci.yml?branch=main)](https://github.com/robottwo/gsh_prime/actions)

<p align="center">
A modern, POSIX-compatible, Generative Shell — fast-paced fork of gsh.
</p>

## About this fork

gsh_prime is an actively maintained fork of the original project, gsh.

- Upstream: https://github.com/atinylittleshell/gsh
- Fork: https://github.com/robottwo/gsh_prime

Focus areas:
- Faster development cadence and iteration
- Compatibility with upstream features and APIs
- Regular contribution of improvements back to upstream

Attribution: All credit for the original project goes to the upstream author and contributors.

## Quick start

For installation, building from source, and first run, see:
- docs/GETTING_STARTED.md

Example build from source:

```bash
git clone https://github.com/robottwo/gsh_prime.git
cd gsh_prime
make build
./bin/gsh
```

## Key features


## Overview

- POSIX-compatible shell with AI enhancements
- Generative assistance that predicts and explains commands
- Agent capabilities with permission controls
- Specialized AI assistants via Subagents
- Works with local or remote LLMs
- Built-in model evaluation using your command history

---

## Generative Command Suggestion

gsh automatically suggests the next command you are likely to run based on your history and context.

![Generative Suggestion](assets/prediction.gif)

Key points:
- Suggestions are lightweight and fast
- Privacy-aware when using local models
- You stay in control: suggestions are previews until you accept

---

## Command Explanation

gsh can explain the command you are about to run so you can validate effects and options quickly.

![Command Explanation](assets/explanation.gif)

Benefits:
- Prevents mistakes
- Speeds up learning of unfamiliar flags or tools
- Aids in review before execution

---

## Agent

The Agent can perform tasks for you by executing commands with your approval, previewing file edits, and providing rich summaries.

![Agent](assets/agent.gif)
![Agent Coding](assets/agent_coding.gif)

Highlights:
- Interactive permission workflow with granular controls
- Preview of code edits and diffs before applying changes
- Chat macros for common tasks

Full guide: [docs/AGENT.md](docs/AGENT.md)

---

## Subagents

Specialized assistants focused on particular tasks, tools, or workflows. Subagents improve security and quality by scoping capabilities and expertise.

Capabilities:
- Directory-aware discovery and auto-reload on `cd`
- Supports Claude-style and Roo Code-style configurations
- Intelligent auto-selection based on your prompt

See: [docs/SUBAGENTS.md](docs/SUBAGENTS.md)

Details and screenshots:
- [docs/FEATURES.md](docs/FEATURES.md)

--- 

--- 

## Documentation

- Getting started: [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md)
- Configuration: [docs/CONFIGURATION.md](docs/CONFIGURATION.md)
- Features: [docs/FEATURES.md](docs/FEATURES.md)
- Agent: [docs/AGENT.md](docs/AGENT.md)
- Subagents: [docs/SUBAGENTS.md](docs/SUBAGENTS.md)
- Roadmap: [ROADMAP.md](ROADMAP.md)
- Changelog: [CHANGELOG.md](CHANGELOG.md)

## Contributing

Contributions are welcome. Please read:
- [CONTRIBUTING.md](CONTRIBUTING.md)

Contribution flow:
- Open issues and pull requests against this repository
- Maintainers periodically propose relevant changes upstream to keep work aligned
- Keep changes focused and upstream-friendly where possible

## Status

This project is under active development. Expect rapid iteration and occasional breaking changes. Feedback and PRs are appreciated.

## Acknowledgements

Built on top of fantastic open-source projects, including but not limited to:
- mvdan/sh
- charmbracelet/bubbletea
- uber-go/zap
- go-gorm/gorm
- sashabaranov/go-openai

See [CHANGELOG.md](CHANGELOG.md) for recent updates and [ROADMAP.md](ROADMAP.md) for planned work.

## License

GPLv3 License. See [LICENSE](LICENSE).
