# Subagents in gsh

gsh now supports Claude-style subagents and Roo Code-style modes, allowing you to create specialized AI assistants with specific roles, tool access, and configurations.

## Overview

Subagents are specialized AI agents that can be invoked with specific commands and have their own:
- System prompts tailored for specific tasks
- Restricted tool access for security and focus
- File access patterns (for Roo Code modes)
- Model configurations

## Configuration Formats

### Claude-style Subagents (.md files)

Claude-style subagents use Markdown files with YAML frontmatter:

```yaml
---
name: code-reviewer
description: Review code for bugs, security issues, and best practices
tools: view_file, view_directory, bash
model: inherit
---

You are a senior code reviewer with extensive experience in software engineering...
```

### Roo Code-style Modes

Roo Code modes support multiple configuration formats:

#### YAML Files (.yaml)
```yaml
customModes:
  - slug: git-helper
    name: 🔧 Git Assistant
    description: Specialized Git workflow assistant
    roleDefinition: You are a Git expert who helps with version control...
    groups:
      - read
      - command
      - ["edit", {"fileRegex": "\\.(md|txt|gitignore)$"}]
```

#### Roomodes Files (.roomodes)
Both `./.roomodes` and `~/.roomodes` files use the same YAML format as above.

#### Rules Directories (.roo/rules-{slug}/)
The recommended Roo Code approach uses dedicated directories with markdown files:

```
.roo/rules-project-helper/
├── main-instructions.md
├── context.md
└── examples.md
```

All markdown files in the directory are combined to form the subagent's system prompt, with the directory name determining the slug (e.g., `rules-project-helper` → `project-helper`).

## Directory Structure

Subagent configurations are discovered in these locations (in priority order):

```
# Project-level (higher priority)
.claude/agents/          # Claude-style subagents
.roo/rules-{slug}/      # Roo Code rules directories (recommended)
.roo/*.yaml             # Roo Code YAML files
.roomodes               # Roo Code modes file

# User-level (lower priority)
~/.claude/agents/       # Claude-style subagents
~/.roo/rules-{slug}/    # Roo Code rules directories (recommended)
~/.roo/*.yaml          # Roo Code YAML files
~/.roomodes            # Roo Code modes file
```

Project-level configurations take precedence over user-level ones with the same ID.

## Automatic Directory Change Detection

gsh automatically detects when you change directories and rescans for subagent configurations, enabling seamless project-specific workflows:

### How It Works

- **Change Detection**: gsh monitors the current working directory (`PWD`)
- **Automatic Rescanning**: When you `cd` to a new directory, gsh automatically:
  - Updates the search paths to include the new directory
  - Rescans for subagent configurations
  - Makes new subagents immediately available
  - Updates tab completion suggestions
  - Clears cached executors to pick up configuration changes

### Practical Examples

#### Project-Specific Subagents

```bash
# Working in a web project
~/web-project $ @!subagents
Loaded 2 subagent(s):
  • Frontend Helper (frontend-dev): React/TypeScript development assistant
  • API Tester (api-test): API endpoint testing and validation

# Move to a data science project
~/web-project $ cd ~/ml-project
~/ml-project $ @!subagents
Loaded 3 subagent(s):
  • Data Analyst (data-viz): Data visualization and analysis expert
  • Model Trainer (ml-model): Machine learning model development
  • Jupyter Helper (notebook): Jupyter notebook assistance

# Tab completion shows current directory's subagents
~/ml-project $ @<TAB>
@data-viz    @ml-model    @notebook

# Intelligent selection uses current directory's subagents
~/ml-project $ # help me visualize this dataset
[data-viz]: I'll help you create visualizations for your dataset...
```

#### Hierarchical Configuration

```bash
# Parent directory subagents
~/my-company $ @!subagents
Loaded 2 subagent(s): company-standards, security-reviewer

# Subdirectory inherits parent + adds specific ones
~/my-company $ cd backend-api
~/my-company/backend-api $ @!subagents
Loaded 3 subagent(s): api-helper, company-standards, security-reviewer

# Deeper nesting for specialized contexts
~/my-company/backend-api $ cd auth-service
~/my-company/backend-api/auth-service $ @!subagents
Loaded 2 subagent(s): auth-specialist, security-reviewer
```

#### Dynamic Workflow Adaptation

The automatic detection enables fluid workflows where your AI assistants adapt to your current context:

1. **Context Switching**: Move between projects and immediately have relevant expertise available
2. **Tab Completion**: Always shows subagents available in your current location
3. **Intelligent Selection**: LLM-powered selection considers only current directory's subagents
4. **No Manual Management**: No need to manually reload or specify which subagents to use

### Performance Considerations

- **Efficient Detection**: Only rescans when directory actually changes
- **Fast Scanning**: Uses filesystem operations optimized for quick discovery
- **Cached Results**: Subagent configurations are cached until directory changes
- **Minimal Overhead**: Detection happens only during subagent operations, not on every command

## Usage

### Invoking Subagents

There are multiple ways to invoke subagents:

1. **Direct invocation (Claude style)**: `@subagent-name your prompt here`
2. **Direct invocation (Roo style)**: `@:mode-slug your prompt here`
3. **Intelligent auto-selection**: `@ your natural language request here`

### Examples

```bash
# Direct invocation with explicit syntax
gsh> @code-reviewer Please review the authentication logic in auth.go
gsh [code-reviewer]: I'll analyze the authentication logic for potential security issues...

# Roo Code mode invocation
gsh> @:git-helper Help me resolve this merge conflict
gsh [🔧 Git Assistant]: Let me guide you through resolving this merge conflict...

# Intelligent auto-selection (NEW)
gsh> # please review my code for bugs
gsh [code-reviewer]: I'll examine your code for potential bugs and issues...

gsh> # help me write unit tests for this function
gsh [test-writer]: I'll create comprehensive unit tests for your function...
```

### Intelligent Selection

gsh uses the fast LLM model to analyze your prompt and automatically select the most appropriate subagent based on:

- **Task context**: What type of work you're requesting
- **Tool requirements**: Which tools the task needs
- **Subagent expertise**: Each subagent's specialized domain
- **Available options**: All configured subagents and their capabilities

This provides a more natural interaction model where you can simply describe what you want to accomplish, and gsh will intelligently route your request to the most suitable specialist.

### Subagent Identification

When a subagent is active, gsh clearly identifies which specialist is responding:

```bash
# Before: Generic agent response
gsh: Here's my analysis of your code...

# After: Clear subagent identification
gsh [code-reviewer]: Here's my analysis of your code...
gsh [🐹 Go Developer]: Let me help optimize this Go function...
```

This helps you understand what type of expertise is being applied to your request.

### Agent Controls

New agent controls for managing subagents:

- `@!subagents` - List all subagents available in the current directory
- `@!subagents <name>` - Show detailed information about a specific subagent (including tools, file restrictions, and configuration)
- `@!reload-subagents` - Refresh configurations from disk (automatic on directory change)
- `@!reset-<subagent-name>` - Reset chat session for specific subagent

**Note**: All agent controls automatically reflect the current directory's subagent configuration. When you change directories, these commands will show subagents available in your new location.

### Tab Completion

gsh provides comprehensive tab completion for the subagent system:

- **Agent controls**: Type `@!s` + Tab → `@!subagents`
- **Subagent invocation**: Type `@` + Tab → Shows subagents available in current directory
- **Partial matching**: Type `@c` + Tab → Shows subagents starting with 'c' in current directory
- **Context preservation**: Works correctly even when completing in the middle of a line
- **Help information**: Tab completion includes help text for discovery
- **Directory-aware**: Completions automatically update when you change directories

```bash
# Tab completion examples
gsh> @<TAB>
@code-reviewer    @docs-writer    @test-writer

gsh> @c<TAB>
@code-reviewer

gsh> some command @c<TAB> more text
some command @code-reviewer more text
```

## Configuration Reference

### Claude Format Fields

- `name` (required): Unique identifier for the subagent
- `description` (required): Description of when to use this subagent
- `tools` (optional): Comma-separated list of allowed tools (defaults to all)
- `model` (optional): Model override or "inherit" to use main agent's model

### Roo Code Format Fields

- `slug` (required): Unique identifier for the mode
- `name` (optional): Display name (uses slug if not provided)
- `description` (optional): Description of the mode's purpose
- `roleDefinition` (required): System prompt for the mode
- `whenToUse` (optional): Additional context about when to use the mode
- `customInstructions` (optional): Additional instructions
- `groups` (optional): Tool access groups (read, edit, command, browser, mcp)
- `model` (optional): Model override

### Tool Groups Mapping

Roo Code tool groups are mapped to gsh tools:

- `read` → `view_file`, `view_directory`
- `edit` → `create_file`, `edit_file`, `view_file`, `view_directory`
- `command` → `bash`
- `browser` → (not applicable in gsh)
- `mcp` → (future extension point)

### File Access Restrictions

Roo Code modes support file access patterns using `fileRegex`:

```yaml
groups:
  - ["edit", {"fileRegex": "\\.(md|txt)$"}]
```

This restricts file editing to only markdown and text files.

## Example Configurations

See the included example configurations:

### Claude-style Examples
- `.claude/agents/code-reviewer.md` - Code review specialist
- `.claude/agents/test-writer.md` - Test writing assistant
- `.claude/agents/docs-writer.md` - Documentation specialist

### Roo Code Examples
- `.roo/dev_modes.yaml` - Development-focused modes (🔧 Git Assistant, 🐹 Go Developer, 🐛 Debug Assistant, 🔒 Security Auditor)
- `.roo/writing_modes.yaml` - Writing and documentation modes (📝 Technical Writer, 💬 Commit Message Helper, 📋 Changelog Writer)

### Rules Directory Example
You can also create Roo rules directories like:
```
.roo/rules-documentation-helper/
├── instructions.md      # Main role definition
├── style-guide.md      # Writing style guidelines
└── examples.md         # Example documentation formats
```

**Note**: The included examples are for demonstration and testing. For production use, create custom subagents tailored to your specific workflows, tools, and domain expertise.

## Benefits

- **Specialization**: Each subagent has a focused role and expertise
- **Security**: Tool access can be restricted per subagent
- **Context**: Specialized system prompts improve response quality
- **Flexibility**: Support for both popular subagent configuration formats
- **Project Organization**: Different subagents for different project needs
- **Automatic Adaptation**: Directory-specific subagents are automatically discovered and available
- **Seamless Workflow**: No manual configuration required when switching between projects
- **Context-Aware Assistance**: AI expertise adapts to your current working context

## Migration

If you're coming from Claude Code or Roo Code, you can use your existing subagent configurations directly:

1. Copy Claude subagents to `.claude/agents/`
2. Copy Roo Code modes to `.roo/` (as `.yaml` files, `.roomodes` files, or `rules-{slug}/` directories)
3. Place them in project directories for project-specific assistants or in `~/` for global access
4. Use `@!subagents` to see what's available (automatically reflects current directory)

The subagent system is fully backward compatible with gsh's existing agent functionality. All existing configurations will automatically benefit from directory change detection - no modifications needed.