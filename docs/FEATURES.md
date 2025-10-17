# Features

gsh_prime focuses on a fast development cadence while remaining compatible with upstream gsh. This document provides concise, actionable details on the core features.

Upstream project: https://github.com/atinylittleshell/gsh  
Fork repository: https://github.com/robottwo/gsh_prime

## Overview

- POSIX-compatible shell with AI enhancements
- Generative assistance that predicts and explains commands
- Agent capabilities with permission controls
- Specialized AI assistants via Subagents
- Works with local or remote LLMs
- Built-in model evaluation using your command history

Related docs:
- Getting started: [GETTING_STARTED.md](GETTING_STARTED.md)
- Configuration: [CONFIGURATION.md](CONFIGURATION.md)
- Agent guide: [AGENT.md](AGENT.md)
- Subagents: [../SUBAGENTS.md](../SUBAGENTS.md)

---

## Generative Command Suggestion

gsh automatically suggests the next command you are likely to run based on your history and context.

![Generative Suggestion](../assets/prediction.gif)

Key points:
- Suggestions are lightweight and fast
- Privacy-aware when using local models
- You stay in control: suggestions are previews until you accept

---

## Command Explanation

gsh can explain the command you are about to run so you can validate effects and options quickly.

![Command Explanation](../assets/explanation.gif)

Benefits:
- Prevents mistakes
- Speeds up learning of unfamiliar flags or tools
- Aids in review before execution

---

## Agent

The Agent can perform tasks for you by executing commands with your approval, previewing file edits, and providing rich summaries.

![Agent](../assets/agent.gif)
![Agent Coding](../assets/agent_coding.gif)

Highlights:
- Interactive permission workflow with granular controls
- Preview of code edits and diffs before applying changes
- Chat macros for common tasks

Full guide: [AGENT.md](AGENT.md)

---

## Subagents

Specialized assistants focused on particular tasks, tools, or workflows. Subagents improve security and quality by scoping capabilities and expertise.

Capabilities:
- Directory-aware discovery and auto-reload on `cd`
- Supports Claude-style and Roo Code-style configurations
- Intelligent auto-selection based on your prompt

See: [SUBAGENTS.md](SUBAGENTS.md)

---

## Local and Remote LLM Support

You can choose your model provider based on privacy and performance needs:

- Local: via [Ollama](https://ollama.com/)
- Remote: any OpenAI-compatible endpoint, e.g. [OpenRouter](https://openrouter.ai/)

Configure via environment variables and your `~/.gshrc`. See examples in:
- Defaults: [../cmd/gsh/.gshrc.default](../cmd/gsh/.gshrc.default)
- Starship prompt: [../cmd/gsh/.gshrc.starship](../cmd/gsh/.gshrc.starship)
- Loader reference: [../cmd/gsh/main.go](../cmd/gsh/main.go)

More details: [CONFIGURATION.md](CONFIGURATION.md)

---

## Model Evaluation

Evaluate how well different LLM models predict your recent commands using built-in tooling.

Example usage:

```bash
# Evaluate using the configured fast model
gsh> gsh_evaluate

# Evaluate using the configured fast model but change model id to mistral:7b
gsh> gsh_evaluate -m mistral:7b

# Control the number of recent commands to use for evaluation
gsh> gsh_evaluate -l 50  # evaluate with the most recent 50 commands you ran

# Run multiple iterations for more accurate results
gsh> gsh_evaluate -i 5  # run 5 iterations
```

Options:
- `-h, --help`: Display help message
- `-l, --limit <number>`: Limit number of entries to evaluate (default: 100)
- `-m, --model <model-id>`: Specify model to use (default: configured fast model)
- `-i, --iterations <number>`: Repeat evaluation for stability (default: 3)

Sample report:

```
┌────────────────────────┬──────────┬──────────┐
│Metric                  │Value     │Percentage│
├────────────────────────┼──────────┼──────────┤
│Model ID                │qwen2.5:3b│          │
│Current Iteration       │3/3       │          │
│Evaluated Entries       │300       │          │
│Prediction Errors       │0         │0.0%      │
│Perfect Predictions     │77        │25.7%     │
│Average Similarity      │0.38      │38.4%     │
│Average Latency         │0.9s      │          │
│Input Tokens Per Request│723.1     │          │
│Output Tokens Per Second│17.7      │          │
└────────────────────────┴──────────┴──────────┘
```

---

## Security and Permissions

- Granular approval per command or command prefix
- Interactive, immediate-keypress permission menu
- Compound command safety: each sub-command must be individually approved
- Authorized command patterns are stored in `~/.config/gsh/authorized_commands`

Management commands:

```bash
# List authorized patterns
cat ~/.config/gsh/authorized_commands

# Clear all patterns
rm ~/.config/gsh/authorized_commands
```

More in: [AGENT.md](AGENT.md)

---


## Roadmap

See [../ROADMAP.md](../ROADMAP.md) for planned improvements. Contributions welcome:
- Fork workflow and upstream PR strategy: [../CONTRIBUTING.md](../CONTRIBUTING.md)
- Issues: https://github.com/robottwo/gsh_prime/issues

---

## Acknowledgements

Built on top of great open source projects:
- [mvdan/sh](https://github.com/mvdan/sh)
- [bubbletea](https://github.com/charmbracelet/bubbletea)
- [zap](https://github.com/uber-go/zap)
- [gorm](https://github.com/go-gorm/gorm)
- [go-openai](https://github.com/sashabaranov/go-openai)