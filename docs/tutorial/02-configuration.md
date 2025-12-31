# Chapter 02: Configuration Basics

gsh uses multiple configuration files.

## Quick Overview

- `~/.gshrc`, `~/.gshenv`, `~/.gsh_profile` — Bash-compatible aliases, functions, and environment variables
- `~/.gsh/repl.gsh` — gsh scripting language for configuring the REPL experience

## Configuration Loading Order

When gsh starts, it loads configuration files in this specific order:

1. `~/.gshrc` (POSIX-compatible configuration, if it exists)
2. `~/.gshenv` (environment variables, if it exists)
3. `~/.gsh/repl.gsh` (REPL configuration, if it exists)

### Login Shell Behavior

When gsh is launched as a login shell (`gsh --login` or `gsh -l`), additional files are loaded before the standard sequence:

1. `/etc/profile` (system profile, if it exists)
2. `~/.gsh_profile` (user profile, if it exists)
3. Then the standard loading order continues above

## Getting Started

Start with a basic `~/.gshrc`:

```bash
# ~/.gshrc
alias ll='ls -la'
export PATH="$HOME/.local/bin:$PATH"
```

Then create your `~/.gsh/repl.gsh`:

```gsh
# ~/.gsh/repl.gsh
tool greet(name: string) {
    print("Hello, " + name + "!")
}

greet("World")
```

After saving, start a new gsh session and try:

```bash
Hello, World!
gsh>
```

## Learn More

This chapter covers just the basics. For comprehensive REPL configuration guides, see the **[SDK Guide](../sdk/README.md)**.

---

**Previous Chapter:** [Chapter 01: Getting Started](01-getting-started-with-gsh.md)

**Next Chapter:** [Chapter 03: Command Prediction](03-command-prediction.md)
