# gsh Tutorial

Welcome to the gsh tutorial! This comprehensive guide will teach you how to use gsh as your interactive shell and write powerful automation scripts.

## What You'll Learn

By the end of this tutorial, you'll know how to:

- ✅ Use gsh as a POSIX-compatible interactive shell
- ✅ Configure your environment with `~/.gshrc` and `~/.gsh/repl.gsh`
- ✅ Set up generative command prediction
- ✅ Use AI agents directly in your shell
- ✅ Write and execute gsh scripts for automation

For deeper configuration topics (custom prompts, advanced SDK features, command middleware), see the **[SDK Guide](../sdk/README.md)**.

## Prerequisites

Before starting, you should:

- Have a basic understanding of shell commands (`ls`, `cd`, `echo`, etc.)
- Have gsh installed (see [main README](../../README.md) for installation)
- Have a text editor for creating configuration files

If you're brand new to shells, you might want to quickly learn bash basics first, then come back to this tutorial.

## Troubleshooting

### Something isn't working

1. **Check the logs**

   ```bash
   tail -f ~/.gsh/gsh.log
   ```

2. **Enable debug logging**

   ```gsh
   # In ~/.gsh/repl.gsh
   gsh.logging.level = "debug"
   ```

3. **Search the relevant chapter** - Most issues are covered in the detailed chapters

## Community and Support

- **GitHub Issues** - [Report bugs or ask questions](https://github.com/atinylittleshell/gsh/issues)
- **Contributing** - Help improve gsh! See [CONTRIBUTING.md](../../CONTRIBUTING.md)

---

Ready to get started? Begin with **[Chapter 01: Getting Started with gsh](01-getting-started-with-gsh.md)** →
