# gsh Tutorial

Welcome to the gsh tutorial! This comprehensive guide will teach you how to use gsh as your interactive shell and write powerful automation scripts.

## What You'll Learn

By the end of this tutorial, you'll know how to:

- ✅ Use gsh as a POSIX-compatible interactive shell
- ✅ Configure your environment with `.gshrc` and `.gshrc.gsh`
- ✅ Create beautiful, informative prompts with Starship
- ✅ Set up generative command prediction
- ✅ Use agents directly in your shell
- ✅ Write and execute gsh scripts for automation

## Prerequisites

Before starting, you should:

- Have a basic understanding of shell commands (`ls`, `cd`, `echo`, etc.)
- Have gsh installed (see [main README](../../README.md) for installation)
- Have a text editor for creating configuration files

If you're brand new to shells, you might want to quickly learn bash basics first, then come back to this tutorial.

## Key Concepts

### REPL vs. Scripting

gsh has two modes:

- **REPL (Read-Eval-Print Loop)** - The interactive shell you get when typing `gsh`. Behaves like bash.
- **Scripts** - `.gsh` files with gsh language code. More powerful than bash scripts.

This tutorial covers both, starting with the REPL.

### Configuration Files

gsh uses two configuration files:

- **`.gshrc`** - Standard bash syntax, for shell aliases and functions
- **`.gshrc.gsh`** - gsh script language, for advanced configuration

Both are optional and loaded at startup.

### Models and Agents

- **Model** - An AI language model (local Ollama, OpenAI, Anthropic, etc.)
- **Agent** - An AI assistant configured with a model and system prompt
- **Prediction** - Fast command suggestions based on history and AI

Later chapters explain these in depth.

## Troubleshooting

### Something isn't working

1. **Check the logs**

   ```bash
   tail -f ~/.gsh.log
   ```

2. **Enable debug mode**

   ```gsh
   # In ~/.gshrc.gsh
   GSH_CONFIG = {
       logLevel: "debug",
   }
   ```

3. **Search the relevant chapter** - Most issues are covered in the detailed chapters

### I'm stuck

- Re-read the relevant chapter carefully
- Try the example exactly as shown
- Check for typos
- Post an issue on [GitHub](https://github.com/atinylittleshell/gsh/issues)

## What's Different from Other Shells?

gsh combines the familiarity of bash with modern features:

| Aspect             | bash    | zsh     | gsh           |
| ------------------ | ------- | ------- | ------------- |
| POSIX Compatible   | ✅      | ✅      | ✅            |
| Scripting Language | Limited | Limited | Full-featured |
| Type Safety        | ❌      | ❌      | ✅            |
| Native AI Support  | ❌      | ❌      | ✅            |
| Command Prediction | Plugins | Plugins | Built-in      |
| Tool Integration   | Manual  | Plugins | MCP servers   |
| Single Binary      | Yes     | Yes     | Yes           |

## Community and Support

- **GitHub Issues** - [Report bugs or ask questions](https://github.com/atinylittleshell/gsh/issues)
- **Contributing** - Help improve gsh! See [CONTRIBUTING.md](../../CONTRIBUTING.md)

---

Ready to get started? Begin with **[Chapter 01: Getting Started with gsh](01-getting-started-with-gsh.md)** →
