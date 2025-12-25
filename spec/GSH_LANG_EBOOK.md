# gsh Scripting Language Ebook - Structure & Plan

**Purpose:** A comprehensive, sequential learning guide for the gsh scripting language that builds from fundamentals to advanced agentic patterns.

**Philosophy:** This ebook teaches concepts in order of _when you need them_, not alphabetical order. Each chapter progressively builds on previous knowledge with hands-on examples and real motivation.

---

## Overall Structure

The ebook is organized into 8 parts, progressing from absolute basics through agentic patterns:

- **Part 1: Getting Started** (2 chapters) - Orientation and first steps
- **Part 2: Core Language Fundamentals** (5 chapters) - Sequential building blocks
- **Part 3: Control Flow** (3 chapters) - Decision making and loops
- **Part 4: Functions & Reusability** (2 chapters) - Custom tools and organization
- **Part 5: External Integration** (4 chapters) - Environment, MCP, shell commands
- **Part 6: AI Agents** (3 chapters) - LLMs, agents, and multi-turn conversations
- **Part 7: Advanced Topics** (1 chapter) - Debugging and troubleshooting
- **Part 8: Reference** (1 chapter) - Complete API reference and cheat sheets

**Total: 21 chapters**

---

## Detailed Chapter Plan

### Part 1: Getting Started

#### Chapter 01: Introduction

- **Status:** DONE
- **File:** `01-introduction.md`
- **Goal:** Orient readers to gsh scripting and why it matters
- **Key Topics:**
  - What is gsh scripting? (vs. bash, vs. other languages)
  - Why use gsh scripts? (type safety, AI integration, MCP tools)
  - Architecture overview (lexer → parser → interpreter)
  - Design philosophy (native Go interpreter, no external dependencies)
  - How this ebook is organized
- **Audience:** Someone new to gsh who wants to understand the "why"

#### Chapter 02: Hello World

- **Status:** DONE
- **File:** `02-hello-world.md`
- **Goal:** Get readers running their first script
- **Key Topics:**
  - Setting up your environment
  - Writing a minimal `.gsh` script
  - Running scripts with `gsh script.gsh`
  - Shebang support (`#!/usr/bin/env gsh`)
  - Basic output with `print()`
  - Structure of a gsh script
- **Audience:** Someone ready to write code

---

### Part 2: Core Language Fundamentals

#### Chapter 03: Values and Types

- **Status:** DONE
- **File:** `03-values-and-types.md`
- **Goal:** Understand gsh's type system
- **Key Topics:**
  - Basic types: `string`, `number`, `boolean`, `null`, `any`
  - Type annotations (TypeScript-like syntax)
  - How types work at runtime
  - Type coercion and conversions
- **Audience:** Foundation for all subsequent chapters

#### Chapter 04: Variables and Assignment

- **Status:** DONE
- **File:** `04-variables-and-assignment.md`
- **Goal:** Learn how to store and work with data
- **Key Topics:**
  - Variable declarations with type annotations
  - Assignment and reassignment
  - Naming conventions
  - Scope basics (will be expanded in later chapters)
  - Updating variables
- **Audience:** Next logical step after types

#### Chapter 05: Operators and Expressions

- **Status:** DONE
- **File:** `05-operators-and-expressions.md`
- **Goal:** Perform computations and comparisons
- **Key Topics:**
  - Arithmetic operators (`+`, `-`, `*`, `/`, `%`)
  - Comparison operators (`==`, `!=`, `<`, `>`, `<=`, `>=`)
  - Logical operators (`&&`, `||`, `!`)
  - String concatenation and `+` operator
  - Operator precedence
  - Parentheses for grouping
- **Audience:** Build on types and variables

#### Chapter 06: Arrays and Objects

- **Status:** DONE
- **File:** `06-arrays-and-objects.md`
- **Goal:** Work with structured data
- **Key Topics:**
  - Array literals and indexing
  - Object/map literals
  - Accessing properties (dot notation, bracket notation)
  - Collections: Arrays, Objects, Sets, Maps
  - Nesting and complex structures
  - When to use each collection type
- **Audience:** Essential for realistic data manipulation

#### Chapter 07: String Manipulation

- **Status:** DONE
- **File:** `07-string-manipulation.md`
- **Goal:** Master working with text
- **Key Topics:**
  - String literals (single, double, triple-quoted)
  - Template literals with `${...}` interpolation
  - String methods (length, split, join, etc.)
  - Multi-line strings for prompts and content
  - Escaping and special characters
- **Audience:** Strings are everywhere in scripts

---

### Part 3: Control Flow

#### Chapter 08: Conditionals

- **Status:** DONE
- **File:** `08-conditionals.md`
- **Goal:** Make decisions in scripts
- **Key Topics:**
  - `if` statements
  - `else if` chains
  - `else` blocks
  - Boolean logic in conditions
  - Common patterns and idioms
- **Audience:** Time to make programs interactive

#### Chapter 09: Loops

- **Status:** DONE
- **File:** `09-loops.md`
- **Goal:** Repeat actions and iterate over data
- **Key Topics:**
  - `while` loops
  - `for` loops (for-of specifically)
  - `break` and `continue`
  - Loop patterns (filtering, mapping, accumulating)
  - Nested loops
- **Audience:** Essential for data processing

#### Chapter 10: Error Handling

- **Status:** DONE
- **File:** `10-error-handling.md`
- **Goal:** Handle failures gracefully
- **Key Topics:**
  - `try`/`catch`/`finally` blocks
  - Error objects and `.message` property
  - Error propagation up the call stack
  - When to catch vs. let errors propagate
  - Best practices for error handling
- **Audience:** Make scripts robust and production-ready

---

### Part 4: Functions & Reusability

#### Chapter 11: Tool Declarations

- **Status:** DONE
- **File:** `11-tool-declarations.md`
- **Goal:** Define custom functions/tools
- **Key Topics:**
  - `tool` keyword and syntax
  - Parameters and parameter types
  - Return types and `return` statements
  - Scope inside tools
  - Recursion (brief mention)
- **Audience:** Organize code and avoid repetition

#### Chapter 12: Tool Calls and Composition

- **Status:** DONE
- **File:** `12-tool-calls-and-composition.md`
- **Goal:** Use tools effectively
- **Key Topics:**
  - Calling tools with arguments
  - Return values and using results
  - Composing tools (calling one from another)
  - Tool organization patterns
  - Library-like patterns
- **Audience:** Build complex scripts from simpler pieces

---

### Part 5: External Integration

#### Chapter 13: Environment Variables

- **Status:** DONE
- **File:** `13-environment-variables.md`
- **Goal:** Interact with the system environment
- **Key Topics:**
  - Accessing `env.VARIABLE_NAME`
  - Default values with `??` operator
  - Setting environment variables
  - Unsetting variables (setting to `null`)
  - Common environment variables (PATH, HOME, etc.)
  - Secrets and API keys in environment
- **Audience:** Bridge between gsh and the wider system

#### Chapter 14: MCP Servers

- **Status:** DONE
- **File:** `14-mcp-servers.md`
- **Goal:** Understand and declare MCP servers
- **Key Topics:**
  - What is MCP (Model Context Protocol)?
  - `mcp` keyword and server declarations
  - Local process-based servers
  - Remote HTTP/SSE servers
  - Server configuration (command, args, env, url, headers)
  - Error handling for server startup
- **Audience:** Gateway to external tool integration

#### Chapter 15: MCP Tool Invocation

- **Status:** DONE
- **File:** `15-mcp-tool-invocation.md`
- **Goal:** Call and use MCP tools
- **Key Topics:**
  - Dot notation for tool access (`filesystem.read_file`)
  - Tool parameters and arguments
  - Tool results and return values
  - Error handling for tool failures
  - Common MCP servers (filesystem, GitHub, etc.)
  - Chaining tool calls
- **Audience:** Practical use of MCP integration

#### Chapter 16: Shell Commands

- **Status:** DONE
- **File:** `16-shell-commands.md`
- **Goal:** Execute and integrate shell commands
- **Key Topics:**
  - `exec()` function syntax
  - Capturing stdout and stderr
  - Checking exit codes
  - Timeout options
  - String interpolation in commands
  - When to use exec vs. MCP tools
  - Practical examples (git, ls, etc.)
- **Audience:** Leverage existing shell tools

---

### Part 6: AI Agents

#### Chapter 17: Model Declarations

- **Status:** DONE
- **File:** `17-model-declarations.md`
- **Goal:** Configure LLM providers
- **Key Topics:**
  - `model` keyword and syntax
  - Providers: OpenAI, Anthropic, Ollama
  - Model names and versions
  - API keys and authentication
  - Temperature and other parameters
  - Local vs. cloud models
- **Audience:** Enable AI capabilities

#### Chapter 18: Agent Declarations

- **Status:** DONE
- **File:** `18-agent-declarations.md`
- **Goal:** Define intelligent agents
- **Key Topics:**
  - `agent` keyword and syntax
  - System prompts (role, personality, instructions)
  - Assigning tools to agents (MCP + user-defined)
  - Agent parameters and customization
  - Multiple agents for different roles
- **Audience:** Orchestrate AI for specific tasks

#### Chapter 19: Conversations and the Pipe Operator

- **Status:** DONE
- **File:** `19-conversations-and-pipes.md`
- **Goal:** Master agent interaction
- **Key Topics:**
  - Pipe operator (`|`) semantics
  - `String | Agent` (create conversation)
  - `Conversation | String` (add user message)
  - `Conversation | Agent` (execute with context)
  - Multi-turn conversations
  - Agent handoffs (`conv | Agent1 | "msg" | Agent2`)
  - Agents calling tools
- **Audience:** Build agentic workflows

---

### Part 7: Advanced Topics

#### Chapter 20: Debugging and Troubleshooting

- **Status:** DONE
- **File:** `20-debugging-and-troubleshooting.md`
- **Goal:** Solve problems effectively
- **Key Topics:**
  - Common errors and their meanings
  - Debug with `log.*` functions
  - Reading error messages and stack traces
  - Debugging agents (checking tool calls, prompts)
  - Debugging MCP integration
  - Performance troubleshooting
- **Audience:** Become self-sufficient

---

### Part 8: Reference

#### Chapter 21: Built-in Functions Reference

- **Status:** DONE
- **File:** `21-builtin-functions.md`
- **Goal:** Complete API documentation
- **Key Topics:**
  - Output: `print()`
  - Logging: `log.debug()`, `log.info()`, `log.warn()`, `log.error()`
  - JSON: `JSON.parse()`, `JSON.stringify()`
  - Shell: `exec()`
  - Complete signatures and examples
  - Error conditions
- **Audience:** Quick lookup reference

---

## Design Principles

1. **Progressive Complexity** - Each chapter builds on previous knowledge without backtracking
2. **Hands-On Examples** - Every concept has runnable, tested code samples
3. **Real Motivation** - Show _why_ each feature matters before diving into _how_
4. **Clear Distinction** - Clearly separate gsh script language from bash/shell/other languages
5. **Narrative Flow** - Present concepts as "You want to do X, here's how..."
6. **Practical Focus** - Emphasize what people actually build with this language
7. **Consistent Structure** - Each chapter follows: problem → concept → examples → exercises

---

## Relationship to Other Documentation

- **GSH_SCRIPT_SPEC.md** = Complete technical specification (reference, alphabetical)
- **GSH_LANG_EBOOK.md** (this file) = Learning journey and pedagogy (sequential, narrative)
- **Individual ebook chapters** = Specific learning modules with examples and exercises

The spec is comprehensive but dense. The ebook translates it into a digestible learning path.

---

## Writing Examples

When writing a code example that requires a custom model, always use a model from ollama like this:

```gsh
model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}
```

---

## Success Criteria

- [ ] All 25 chapters written and reviewed
- [ ] All code examples tested and runnable
- [ ] Progressive complexity maintained (no forward references)
- [ ] Each chapter ~1000-2000 words (readable in 10-15 minutes)
- [ ] Clear navigation between chapters
- [ ] Consistent style and tone throughout
