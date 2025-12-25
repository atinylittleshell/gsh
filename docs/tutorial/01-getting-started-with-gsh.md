# Chapter 01: Getting Started with gsh

Welcome to gsh! If you're new to the shell world, you might be wondering what gsh is and why you should care. In this chapter, we'll get you up and running with the basics.

## What is gsh?

**gsh** is an interactive shell with two distinct personalities:

1. **Interactive REPL** - When you run `gsh` without arguments, you get a shell that behaves like bash or zsh. It's POSIX-compatible, so all your familiar shell commands work exactly as expected.

2. **Scripting Language** - When you write `.gsh` files, you're using a powerful scripting language with type safety, AI integration, and modern features.

This tutorial focuses on the **interactive REPL**—your daily shell experience. If you want to learn scripting, check out the [Script Documentation](../script/).

## Installation

Before we begin, you need the gsh binary. Visit the [official repository](https://github.com/atinylittleshell/gsh) for installation instructions.

Once installed, verify it works:

```bash
gsh --version
```

You should see a version number like `v0.1.0` or similar.

## Starting Your First gsh Session

Launch an interactive gsh session:

```bash
gsh
```

You should see a prompt like:

```
gsh>
```

Congratulations! You're now inside the gsh REPL.

## Basic Shell Experience

gsh works like any POSIX shell. Try some familiar commands:

```bash
gsh> echo "Hello, world!"
Hello, world!

gsh> pwd
/Users/yourname

gsh> ls
Documents  Downloads  Desktop  ...

gsh> cd Documents
gsh> pwd
/Users/yourname/Documents
```

All the commands you know from bash work here:

- **File operations**: `ls`, `cd`, `mkdir`, `rm`, `cp`, `mv`
- **Text processing**: `echo`, `cat`, `grep`, `sed`, `awk`
- **Process management**: `ps`, `kill`, `jobs`, `fg`, `bg`
- **Piping**: `command1 | command2`
- **Redirection**: `> file`, `< file`, `>> file`
- **Variables**: `NAME=value`, `$NAME`, `${NAME}`
- **Aliases**: `alias ll='ls -la'`

### Command History

gsh keeps a history of your commands. Use:

- **Up Arrow** / **Down Arrow** - Navigate through previous commands
- **Ctrl+R** - Search command history
- **Ctrl+A** - Jump to start of line
- **Ctrl+E** - Jump to end of line
- **Ctrl+U** - Clear line

Your history is automatically saved in `~/.gsh_history`.

### Tab Completion

Press **Tab** to complete file names, directory names, and commands:

```bash
gsh> cat /etc/pass[TAB]
gsh> cat /etc/passwd
```

Tab completion understands your shell context and suggests relevant options.

### Command Substitution

Use backticks or `$()` syntax to run commands and capture their output:

```bash
gsh> echo "Current date: $(date)"
Current date: Thu Dec 25 00:58:57 PST 2024

gsh> FILES=`ls -1`
gsh> echo "$FILES"
Documents
Downloads
Desktop
```

This is standard shell syntax, just like bash.

## Exiting gsh

To leave the shell, type:

```bash
gsh> exit
```

Or press **Ctrl+D**.

## Key Differences from Bash

While gsh aims for bash compatibility, there are a few differences you should know:

### `.gshrc` Configuration Files

Unlike bash which only reads `.bashrc`, gsh reads **two** configuration files (if they exist):

1. **`.gshrc`** - Standard bash commands (like bash's `.bashrc`)
2. **`.gshrc.gsh`** - gsh-specific configuration using the gsh language

We'll cover this in detail in Chapter 02.

### gsh-Specific Features in the REPL

The gsh REPL adds features that bash doesn't have:

- **LLM Predictions** - As you type, gsh can suggest commands using an AI model
- **Agent Interactions** - Call AI agents directly from the shell
- **Custom Prompts** - Write sophisticated prompts using tools (like Starship)

These are optional and don't interfere with normal shell usage. We'll explore them in later chapters.

### Case Sensitivity

Like bash, gsh is case-sensitive:

```bash
gsh> NAME="Alice"
gsh> echo $name
         # (empty—different variable)
gsh> echo $NAME
Alice
```

## Troubleshooting

### Command Not Found

If you get "command not found":

```bash
gsh> unknowncmd
-bash: unknowncmd: command not found
```

This means the command isn't in your `$PATH`. Check:

```bash
gsh> echo $PATH
```

### Slow Startup

If gsh takes a long time to start, check your `.gshrc` or `.gshrc.gsh` files. Loading large configuration files or connecting to remote MCP servers can slow things down.

### History Not Saving

History is stored in `~/.gsh_history`. If it's not working, check file permissions:

```bash
gsh> ls -la ~/.gsh_history
```

## What's Next?

Now that you're comfortable with the basics, Chapter 02 covers **Configuration**—how to customize your shell environment with `.gshrc` and `.gshrc.gsh` files, and how to access the full gsh scripting language documentation.

---

**Next Chapter:** [Chapter 02: Configuration](02-configuration.md)
