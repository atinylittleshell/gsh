# Agent

gsh can act as an agent that invokes commands on your behalf. Commands starting with "@" are sent to the agent as a chat message.

![Agent](../assets/agent.gif)

gsh can even code for you!

![Agent Coding](../assets/agent_coding.gif)

## Chat Macros

You can use chat macros to quickly send frequently used messages to the agent.

A chat macro starts with "@/" followed by the macro name. The default configuration comes with a few chat macros:

```bash
# Summarize git changes
gsh> @/gitdiff

# Commit and push changes
gsh> @/gitpush

# Review changes and get suggestions
gsh> @/gitreview
```

You can customize your own macros by modifying the `GSH_AGENT_MACROS` configuration in your `.gshrc` file.
The value should be a JSON object mapping macro names to their corresponding chat messages.
See [Configuration](../README.md#configuration) for more details.

## Permission System

When the agent wants to execute commands on your behalf, gsh provides a flexible permission system to ensure you maintain control over what gets executed.

### Response Options

When prompted for permission to run a command, you have several response options:

- `y` or `yes`: Allow this command to run once
- `n` or `no`: Deny this command
- `m` or `manage`: Open an interactive menu to manage permissions for command prefixes
- Any other text: Provide custom feedback to the agent (treated as denial)

### "Manage" Functionality

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

### Examples

```bash
# First time running a git status command
gsh> @ check git status
Agent wants to run: git status
Do I have your permission to run the following command? (y/N/manage/freeform) m

# The permission menu opens, and you can approve "^git status.*" which is saved to ~/.config/gsh/authorized_commands
# Future git status commands will be auto-approved:

gsh> @ show git status with short format
Agent wants to run: git status -s
# This runs automatically without prompting because it matches the saved pattern
```

### Pattern Generation

gsh intelligently generates regex patterns based on the command structure:

- **Regular commands**: `ls -la` → `^ls.*` (matches any `ls` command)
- **Commands with subcommands**: `git commit -m "message"` → `^git commit.*` (matches any `git commit` command)
- **Special commands**: Commands like `git`, `npm`, `yarn`, `docker`, and `kubectl` include their subcommands in the pattern
- **Compound commands**: `ls && pwd` → `["^ls.*", "^pwd.*"]` (generates patterns for all individual commands)

### Managing Authorized Commands

The authorized commands are stored in `~/.config/gsh/authorized_commands` as regex patterns, one per line. You can:

- **View patterns**: `cat ~/.config/gsh/authorized_commands`
- **Edit patterns**: Manually edit the file to modify or remove patterns
- **Clear all patterns**: `rm ~/.config/gsh/authorized_commands`

This system works alongside the existing `GSH_AGENT_APPROVED_BASH_COMMAND_REGEX` configuration, providing both pre-configured and dynamically-generated command approval.

## Compound Command Security

gsh provides robust security for compound commands (commands using `;`, `&&`, `||`, `|`, or subshells) by analyzing each individual command separately:

### Security Model

- **Individual Validation**: Each command in a compound statement must be individually approved
- **No Bypass**: Malicious commands cannot hide behind approved commands
- **Comprehensive Parsing**: Handles all shell operators including pipes, subshells, and command substitution

### Examples

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

### Supported Compound Operators

- **Sequential**: `cmd1; cmd2` - Commands run in sequence
- **Conditional AND**: `cmd1 && cmd2` - cmd2 runs only if cmd1 succeeds
- **Conditional OR**: `cmd1 || cmd2` - cmd2 runs only if cmd1 fails
- **Pipes**: `cmd1 | cmd2` - Output of cmd1 becomes input of cmd2
- **Subshells**: `(cmd1 && cmd2)` - Commands run in isolated environment
- **Command Substitution**: `echo $(cmd1)` - Output of cmd1 used as argument

## Agent Controls

Agent controls are built-in commands that help you manage your interaction with the agent.
An agent control starts with "@!" followed by the control name.

Currently supported controls:

```bash
# Open the interactive configuration menu
gsh> @!config

# Reset the current chat session and start fresh
gsh> @!new

# Show token usage statistics for the current chat session
gsh> @!tokens
```

## Magic Fix

When a command fails, you can use `@?` to ask the agent to analyze the error and suggest a fix.

```bash
gsh> ls nonexistent_file
ls: nonexistent_file: No such file or directory

gsh> @?
gsh: The command failed because...

Command: ls "nonexistent_file"
Run this fix? [y/N]
```

The agent will:
1. Analyze the last failed command and its error output
2. Explain why it failed
3. Suggest a fixed command

If a fix is found, you can run it immediately with a single keypress (`y` to confirm, any other key to cancel).

## Default Confirmation Behavior

By default, confirmation prompts (including Magic Fix, command permissions, and app updates) default to "no" when Enter is pressed, displaying `[y/N]`.

You can change this behavior by setting `GSH_DEFAULT_TO_YES=1` in your `.gshrc` file. When enabled:
- Prompts will display `[Y/n]` instead of `[y/N]`
- Pressing Enter will confirm instead of cancel
