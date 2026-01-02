# Chapter 01: Getting Started with gsh

Welcome to gsh! If you're new to the shell world, you might be wondering what gsh is and why you should care. In this chapter, we'll get you up and running with the basics.

## What is gsh?

**gsh** is an interactive shell with two distinct personalities:

1. **Interactive REPL** - When you run `gsh` without arguments, you get a shell that behaves like bash or zsh. It's POSIX-compatible, so all your familiar shell commands work exactly as expected.

2. **Scripting Language** - When you write `.gsh` files, you're using a powerful scripting language with type safety, AI integration, and modern features.

This tutorial focuses on the **interactive REPL**—your daily shell experience. If you want to learn scripting, check out the [Script Documentation](../script/).

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

### Building from Source

To build gsh from source, ensure you have Go installed and run the following command:

```bash
make build
```

This will compile the project and place the binary in the `./bin` directory.

### Verify Installation

Once installed, verify it works:

```bash
gsh --version
```

You should see a version number like `v1.0.0` or similar.

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

Your history is automatically saved in `~/.gsh/history.db`.

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

This is standard POSIX shell syntax, just like bash.

## Exiting gsh

To leave the shell, type:

```bash
gsh> exit
```

Or press **Ctrl+D**.

## Troubleshooting

### Command Not Found

If you get "command not found":

```bash
gsh> unknowncmd
"unknowncmd": executable file not found in $PATH
```

This means the command isn't in your `$PATH`. Check:

```bash
gsh> echo $PATH
```

### Slow Startup

If gsh takes a long time to start, check your `~/.gshrc` or `~/.gsh/repl.gsh` files. Loading large configuration files or connecting to remote MCP servers can slow things down.

### History Not Saving

History is stored in `~/.gsh/history.db`. If it's not working, check file permissions:

```bash
gsh> ls -la ~/.gsh/history.db
```

## What's Next?

Now that you're comfortable with the basics, Chapter 02 covers **Configuration**—how to customize your shell environment.

---

**Next Chapter:** [Chapter 02: Configuration](02-configuration.md)
