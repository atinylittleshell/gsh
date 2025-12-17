# Configuration

gsh_prime is configurable via simple dotfiles and environment variables. This guide explains where configuration lives, default values, and common customization tips.

Upstream project: https://github.com/atinylittleshell/gsh  
Fork repository: https://github.com/robottwo/gsh_prime

## Files and Load Order

The shell loads configuration in this order:

1. If launched as a login shell `gsh -l`, sources:
   - `/etc/profile`
   - `~/.gsh_profile`
2. Always loads:
   - `~/.gshrc`
   - `~/.gshenv`

Reference implementation for file discovery is in [cmd/gsh/main.go](../cmd/gsh/main.go).

Default templates you can copy and customize:
- [.gshrc.default](../cmd/gsh/.gshrc.default)
- [.gshrc.starship](../cmd/gsh/.gshrc.starship)

## ~/.gshrc

Primary runtime configuration file. Recommended setup:

```bash
# Example: configure models and behavior
export GSH_FAST_MODEL_ID="qwen2.5:3b"
export GSH_AGENT_CONTEXT_WINDOW_TOKENS=6000
export GSH_MINIMUM_HEIGHT=10

# Optional: pre-approve safe patterns for agent-executed commands
# Regex, one-per-line in ~/.config/gsh/authorized_commands is managed automatically
# You can still provide defaults via env if desired:
# export GSH_AGENT_APPROVED_BASH_COMMAND_REGEX='^ls.*|^cat.*|^git status.*'

# Enable chat macros (JSON string)
export GSH_AGENT_MACROS='{
  "gitdiff": "summarize the changes in the current git diff",
  "gitpush": "create a concise commit message and push",
  "gitreview": "review my recent changes and suggest improvements"
}'
```

Tip: Keep sensitive values (e.g., API keys) in `~/.gshenv` rather than `~/.gshrc`.

## ~/.gshenv

Environment-only overrides that load after `~/.gshrc`. Useful for secrets or per-machine toggles:

```bash
# Example: OpenAI-compatible endpoint via OpenRouter or your own gateway
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://openrouter.ai/api/v1"

# Ollama for local models
export OLLAMA_HOST="http://127.0.0.1:11434"
```

## Interactive Configuration Menu

gsh provides an interactive configuration menu accessible via the `@!config` command:

```bash
gsh> @!config
```

The configuration menu allows you to:
- Configure slow model settings (API key, model ID, base URL) for chat and agent operations
- Configure fast model settings for auto-completion and suggestions
- Set the assistant box height
- Toggle safety checks for command approval

Changes made through the configuration menu are persisted to `~/.gsh_config_ui` and automatically sourced in your shell.

## Common Environment Variables

- `GSH_FAST_MODEL_ID`: Model ID for the fast LLM (default: qwen2.5).
- `GSH_FAST_MODEL_PROVIDER`: LLM provider for fast model (ollama, openai, openrouter).
- `GSH_MINIMUM_HEIGHT`: Minimum number of lines reserved for prompt and UI rendering.
- `GSH_AGENT_CONTEXT_WINDOW_TOKENS`: Context window size for agent chats and tools; messages are pruned beyond this.
- `GSH_AGENT_APPROVED_BASH_COMMAND_REGEX`: Optional regex to pre-approve read-only or safe command families.
- `HTTP(S)_PROXY`, `NO_PROXY`: Standard proxy variables respected by network calls.

See defaults and comments in [.gshrc.default](../cmd/gsh/.gshrc.default).

## Prompt Customization with Starship

You can use Starship to render a custom prompt.

1. Install Starship: https://starship.rs
2. Copy the example config and adapt it:
   - [.gshrc.starship](../cmd/gsh/.gshrc.starship)

In your `~/.gshrc`:

```bash
# Example Starship integration
export STARSHIP_CONFIG="$HOME/.config/starship.toml"
eval "$(starship init bash)"  # or zsh if you prefer
```

Notes:
- The example includes prompt sections for exit code, duration, and gsh build version in dev mode.
- Adjust symbols, colors, and modules per your preference.

## Login Shell Setup

To make gsh your login shell (not recommended yet; experimental):

```bash
which gsh
echo "/path/to/gsh" | sudo tee -a /etc/shells
chsh -s "/path/to/gsh"
```

If you choose to run as a login shell, `gsh -l` will source `/etc/profile` and `~/.gsh_profile` before `~/.gshrc`.

## Authorized Commands Store

When you approve commands during agent operations, gsh stores regex patterns in:

- `~/.config/gsh/authorized_commands`

Manage them with standard file operations:

```bash
# View
cat ~/.config/gsh/authorized_commands

# Edit
$EDITOR ~/.config/gsh/authorized_commands

# Reset
rm ~/.config/gsh/authorized_commands
```

These patterns complement any defaults you provide via environment variables.

## Troubleshooting

- Unexpected prompt size: verify `GSH_MINIMUM_HEIGHT`.
- Missing macros: ensure `GSH_AGENT_MACROS` is valid JSON.
- API errors: confirm `OPENAI_BASE_URL` and `OPENAI_API_KEY` or Ollama connectivity.
- Login shell confusion: confirm whether you started gsh as a login shell and which profile files are being sourced.

## Related Docs

- Quick start: [GETTING_STARTED.md](GETTING_STARTED.md)
- Features: [FEATURES.md](FEATURES.md)
- Agent: [AGENT.md](AGENT.md)
- Subagents overview: [SUBAGENTS.md](SUBAGENTS.md)