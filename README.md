# gsh

[![License](https://img.shields.io/github/license/atinylittleshell/gsh.svg)](https://github.com/atinylittleshell/gsh/blob/main/LICENSE)
[![Release](https://img.shields.io/github/release/atinylittleshell/gsh.svg)](https://github.com/atinylittleshell/gsh/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/atinylittleshell/gsh/ci.yml?branch=main)](https://github.com/atinylittleshell/gsh/actions)
[![Test Coverage](https://codecov.io/gh/atinylittleshell/gsh/branch/main/graph/badge.svg?token=U7HWPOEPTF)](https://codecov.io/gh/atinylittleshell/gsh)

<p align="center">
A Modern, POSIX-compatible, <ins>G</ins>enerative <ins>Sh</ins>ell.
</p>

## Status

This project is in early development stage. Use at your own risk!
Please expect bugs, incomplete features and breaking changes.

That said, if you can try it out and provide feedback,
that would help make gsh more useful!

## Goals

- **POSIX-compatibility**: you shouldn't need to learn a new shell language to adopt gsh
- **Embrace AI**: gsh is designed from the ground up for the generative AI era to provide you with intelligent assistance at the right place, right time
- **Privacy**: gsh should allow you to use local LLMs for guaranteed privacy
- **Customizability**: gsh is _your_ shell. You should be able to configure it to your liking
- **Extensibility**: gsh should allow the community to build and share extensions to make it more useful

## But what does being "generative" mean?

### Generative suggestion of shell commands

gsh will automatically suggest the next command you are likely want to run.

![Generative Suggestion](assets/prediction.gif)

### Command explanation

gsh will provide an explanation of the command you are about to run.

![Command Explanation](assets/explanation.gif)

### Agent

gsh can act as an agent that invoke commands on your behalf.
Commands starting with "#" are sent to the agent as a chat message.

![Agent](assets/agent.gif)

gsh can even code for you!

![Agent Coding](assets/agent_coding.gif)

#### Chat Macros

You can use chat macros to quickly send frequently used messages to the agent.

A chat macro starts with "#/" followed by the macro name. The default configuration comes with a few chat macros:

```bash
# Summarize git changes
gsh> #/gitdiff

# Commit and push changes
gsh> #/gitpush

# Review changes and get suggestions
gsh> #/gitreview
```

You can customize your own macros by modifying the `GSH_AGENT_MACROS` configuration in your `.gshrc` file.
The value should be a JSON object mapping macro names to their corresponding chat messages.
See [Configuration](#configuration) for more details.

#### Permission System

When the agent wants to execute commands on your behalf, gsh provides a flexible permission system to ensure you maintain control over what gets executed.

##### Response Options

When prompted for permission to run a command, you have several response options:

- **`y` or `yes`**: Allow this command to run once
- **`n` or `no`**: Deny this command
- **`m` or `manage`**: Open an interactive menu to manage permissions for command prefixes
- **Any other text**: Provide custom feedback to the agent (treated as denial)

##### "Manage" Functionality

The `m` (manage) response option opens an **interactive real-time permissions menu**:

1. When you respond with `m` or `manage`, gsh displays a clean menu with all command prefixes
2. Navigate and control the menu using **immediate keyboard input** (no Enter required):
   - **j/k** to move between options instantly
   - **SPACE** to toggle permissions for individual prefixes (shows ✓ when enabled)
   - **1-9** to jump directly to a specific option number
   - **ENTER** to apply your selections and save them to `~/.config/gsh/authorized_commands`
   - **ESC** to cancel without making changes
   - **y/n** for direct yes/no responses

For example, with the command `ls --foo bar`, you can individually manage permissions for:
- `ls` (allows any ls command)
- `ls --foo` (allows ls with --foo flag and any additional arguments)
- `ls --foo bar` (allows this exact command)

The menu provides **clear visual feedback** with clean formatting that matches the tab-completion display style.

**Example menu display:**
```
Managing permissions for: ls --foo bar

Permission Management - Toggle permissions for command prefixes:

> 1. [✓] ls
  2. [ ] ls --foo
  3. [ ] ls --foo bar

j/k=navigate  SPACE=toggle  ENTER=apply  ESC=cancel
(Keys respond immediately - no Enter needed)

Current selection: ls
Enabled permissions: ls
```

The system combines clean, reliable display with immediate keyboard responsiveness, providing an intuitive interface for managing granular command permissions.

**Note**: In non-interactive environments (like automated scripts), the system automatically falls back to line-based input for compatibility.

##### Legacy "Always Allow" Support

The legacy `a` (always) response is still supported for backward compatibility and works the same as the previous version.

##### Examples

```bash
# First time running a git status command
gsh> # check git status
Agent wants to run: git status
Do I have your permission to run the following command? (y/N/freeform/a) a

# The pattern "^git status.*" is now saved to ~/.config/gsh/authorized_commands
# Future git status commands will be auto-approved:

gsh> # show git status with short format
Agent wants to run: git status -s
# This runs automatically without prompting because it matches the saved pattern
```

##### Pattern Generation

gsh intelligently generates regex patterns based on the command structure:

- **Regular commands**: `ls -la` → `^ls.*` (matches any `ls` command)
- **Commands with subcommands**: `git commit -m "message"` → `^git commit.*` (matches any `git commit` command)
- **Special commands**: Commands like `git`, `npm`, `yarn`, `docker`, and `kubectl` include their subcommands in the pattern
- **Compound commands**: `ls && pwd` → `["^ls.*", "^pwd.*"]` (generates patterns for all individual commands)

##### Managing Authorized Commands

The authorized commands are stored in `~/.config/gsh/authorized_commands` as regex patterns, one per line. You can:

- **View patterns**: `cat ~/.config/gsh/authorized_commands`
- **Edit patterns**: Manually edit the file to modify or remove patterns
- **Clear all patterns**: `rm ~/.config/gsh/authorized_commands`

This system works alongside the existing `GSH_AGENT_APPROVED_BASH_COMMAND_REGEX` configuration, providing both pre-configured and dynamically-generated command approval.

#### Compound Command Security

gsh provides robust security for compound commands (commands using `;`, `&&`, `||`, `|`, or subshells) by analyzing each individual command separately:

##### Security Model

- **Individual Validation**: Each command in a compound statement must be individually approved
- **No Bypass**: Malicious commands cannot hide behind approved commands
- **Comprehensive Parsing**: Handles all shell operators including pipes, subshells, and command substitution

##### Examples

```bash
# ✅ SECURE: All commands approved
gsh: Do I have your permission to run the following command?
Command: ls && pwd && echo done
# If ls, pwd, and echo are all approved → auto-approved

# ❌ BLOCKED: Contains unapproved command
gsh: Do I have your permission to run the following command?
Command: ls; rm -rf /
# Even though ls is approved, rm is not → requires confirmation

# ❌ BLOCKED: Injection in subshell
gsh: Do I have your permission to run the following command?
Command: (ls && rm -rf /)
# rm command in subshell is not approved → requires confirmation

# ❌ BLOCKED: Injection in pipe
gsh: Do I have your permission to run the following command?
Command: ls | rm -rf /
# rm command in pipe is not approved → requires confirmation
```

##### Supported Compound Operators

- **Sequential**: `cmd1; cmd2` - Commands run in sequence
- **Conditional AND**: `cmd1 && cmd2` - cmd2 runs only if cmd1 succeeds
- **Conditional OR**: `cmd1 || cmd2` - cmd2 runs only if cmd1 fails
- **Pipes**: `cmd1 | cmd2` - Output of cmd1 becomes input of cmd2
- **Subshells**: `(cmd1 && cmd2)` - Commands run in isolated environment
- **Command Substitution**: `echo $(cmd1)` - Output of cmd1 used as argument

This security model ensures that command injection attacks are prevented while maintaining usability for legitimate compound commands.

#### Agent Controls

Agent controls are built-in commands that help you manage your interaction with the agent.
An agent control starts with "#!" followed by the control name.

Currently supported controls:

```bash
# Reset the current chat session and start fresh
gsh> #!new

# Show token usage statistics for the current chat session
gsh> #!tokens
```

### Supports both local and remote LLMs

gsh can run with either

- Local LLMs through [Ollama](https://ollama.com/)
- Or remote LLMs through an OpenAI API-compatible endpoint, such as [OpenRouter](https://openrouter.ai/)

## Installation

To install gsh:

```bash
# Linux and macOS through Homebrew
brew tap atinylittleshell/gsh https://github.com/atinylittleshell/gsh
brew install atinylittleshell/gsh/gsh

# You can use gsh on arch, btw
yay -S gsh-bin
```

Windows is not supported (yet).

### Upgrading

gsh can automatically detect newer versions and self update.

## Building from Source

To build gsh from source, ensure you have Go installed and run the following command:

```bash
make build
```

This will compile the project and place the binary in the `./bin` directory.

## Configuration

gsh can be configured through a configuration file located at `~/.gshrc`.
Configuration options and default values can be found in [.gshrc.default](./cmd/gsh/.gshrc.default).

gsh also loads a `~/.gshenv` file, right after loading `~/.gshrc`.
This file can be used to set environment variables that the gsh session will use.

When launched as a login shell (`gsh -l`),
gsh will also load `/etc/profile` and `~/.gsh_profile` at start (before `~/.gshrc`).

### Custom command prompt

You can use [Starship.rs](https://starship.rs/) to render a custom command line prompt.
See [.gshrc.starship](./cmd/gsh/.gshrc.starship) for an example configuration.

## Usage

### Manually

You can manually start gsh from an existing shell:

```bash
gsh
```

### Automatically, through an existing shell

You can also automatically launch gsh from another shell's configuration file:

```bash
# For bash
echo "gsh" | tee -a ~/.bashrc
```

```bash
# For zsh
echo "gsh" | tee -a ~/.zshrc

# Your zsh config may have set "gsh" as an alias for `git show`.
# In that case, you would need to use the full path to gsh.
echo "/full/path/to/gsh" | tee -a ~/.zshrc
```

### Automatically, as your default shell

Or, you can set gsh as your default shell.
This is not recommended at the moment as gsh is still in early development.
But if you know what you are doing, you can do so by:

```bash
# Get the absolute path to gsh by running `which gsh`
which gsh

# Add gsh to the list of approved shells
echo "/path/to/gsh" | sudo tee -a /etc/shells

# Change your default shell to gsh
chsh -s "/path/to/gsh"
```

## Default Key Bindings

gsh provides a set of default key bindings for navigating and editing text input.
These key bindings are designed to be familiar to users of traditional shells and text editors.
It's on the roadmap to allow users to customize these key bindings.

- **Character Forward**: `Right Arrow`, `Ctrl+F`
- **Character Backward**: `Left Arrow`, `Ctrl+B`
- **Word Forward**: `Alt+Right Arrow`, `Ctrl+Right Arrow`, `Alt+F`
- **Word Backward**: `Alt+Left Arrow`, `Ctrl+Left Arrow`, `Alt+B`
- **Delete Word Backward**: `Alt+Backspace`, `Ctrl+W`
- **Delete Word Forward**: `Alt+Delete`, `Alt+D`
- **Delete After Cursor**: `Ctrl+K`
- **Delete Before Cursor**: `Ctrl+U`
- **Delete Character Backward**: `Backspace`, `Ctrl+H`
- **Delete Character Forward**: `Delete`, `Ctrl+D`
- **Line Start**: `Home`, `Ctrl+A`
- **Line End**: `End`, `Ctrl+E`
- **Paste**: `Ctrl+V`

## Model Evaluation

gsh provides a built-in command to use your recent command history to evaluate how well different LLM models work for predicting your commands.
You can run the evaluation command with various options:

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

Available options:

- `-h, --help`: Display help message
- `-l, --limit <number>`: Limit the number of entries to evaluate (default: 100)
- `-m, --model <model-id>`: Specify the model to use (default: use the default fast model)
- `-i, --iterations <number>`: Number of times to repeat the evaluation (default: 3)

You will get a report like below on how well the model performed in predicting the commands you recently ran.

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

## Roadmap

See [ROADMAP.md](./ROADMAP.md) for what's already planned.
Feel free to suggest new features by opening an issue!

## Acknowledgements

gsh is built on top of many great open source projects. Most notably:

- [mvdan/sh](https://github.com/mvdan/sh) - A shell parser, formatter, and interpreter
- [bubbletea](https://github.com/charmbracelet/bubbletea) - A powerful little TUI framework
- [zap](https://github.com/uber-go/zap) - Blazing fast, structured, leveled logging in Go
- [gorm](https://github.com/go-gorm/gorm) - The fantastic ORM library for Golang
- [go-openai](https://github.com/sashabaranov/go-openai) - A Go client for the OpenAI API

## Support This Project

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/onelittleshell)
