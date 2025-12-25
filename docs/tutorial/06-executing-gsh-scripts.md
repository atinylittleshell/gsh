# Chapter 06: Executing gsh Scripts

So far, we've focused on using gsh interactively. But gsh's real power shines when you write scripts—`.gsh` files that automate tasks, call agents, and integrate with tools. This chapter shows you how to write and execute gsh scripts.

## What Are gsh Scripts?

A **gsh script** is a file with a `.gsh` extension containing code written in the gsh scripting language. Unlike your `.gshrc.gsh` configuration file (which runs once at startup), scripts are standalone programs you execute on demand.

**gsh scripts vs. bash scripts:**

- **bash scripts** - Strings, limited types, exit codes for error handling
- **gsh scripts** - Type-safe, proper error handling, AI agents, MCP tools, structured data

## Your First gsh Script

Create a file called `hello.gsh`:

```gsh
#!/usr/bin/env gsh

# Your first gsh script
print("Hello from gsh!")
print("Current directory: " + env.PWD)
```

Make it executable:

```bash
chmod +x hello.gsh
```

Run it:

```bash
./hello.gsh
```

Output:

```
Hello from gsh!
Current directory: /Users/yourname/projects
```

Or run it explicitly with gsh:

```bash
gsh hello.gsh
```

## Script Structure

Here's a typical gsh script structure:

```gsh
#!/usr/bin/env gsh

# Imports (none in gsh, but you can include config)
# Define models and tools here if needed

# Helper functions (tools)
tool format_size(bytes: number): string {
    if (bytes < 1024) {
        return `${bytes} B`
    } else if (bytes < 1024 * 1024) {
        return `${(bytes / 1024).toFixed(2)} KB`
    } else {
        return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
    }
}

# Main logic
files = exec("ls -la").stdout.split("\n")
for (line in files) {
    if (line.length > 0) {
        print(line)
    }
}

# Exit
print("Done!")
```

## Common Script Patterns

### Pattern 1: Process Files

```gsh
#!/usr/bin/env gsh

# Find all .md files and count words in each
files = exec("find . -name '*.md' -type f").stdout.split("\n")

for (file in files) {
    if (file.length > 0) {
        result = exec(`wc -w ${file}`)
        print(result.stdout)
    }
}
```

### Pattern 2: Data Processing

```gsh
#!/usr/bin/env gsh

# Read JSON, process, write back
tool process_item(item: any): any {
    // Transform item
    return {
        id: item.id,
        name: item.name.toUpperCase(),
        processed: true,
    }
}

content = exec("cat data.json").stdout
data = JSON.parse(content)

processed = []
for (item in data) {
    processed.push(process_item(item))
}

print(JSON.stringify(processed, null, 2))
```

### Pattern 3: Error Handling

```gsh
#!/usr/bin/env gsh

try {
    // Try to read a file
    content = exec("cat important-file.txt").stdout
    print(`File has ${content.length} characters`)
} catch (error) {
    print(`Error reading file: ${error.message}`)
    // Exit with error status
    exec("exit 1")
}
```

### Pattern 4: Using Agents

```gsh
#!/usr/bin/env gsh

// Define a model and agent
model Claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-opus-4-mini",
}

agent CodeAnalyzer {
    model: Claude,
    system: "You are a code analysis expert.",
}

// Read code file
codeContent = exec("cat main.py").stdout

// Ask agent to review it
analysis = codeContent | CodeAnalyzer
print(analysis)
```

### Pattern 5: Conditional Logic

```gsh
#!/usr/bin/env gsh

// Check if we're in a git repo
status = exec("git status --porcelain")

if (status.exitCode == 0) {
    hasChanges = status.stdout.length > 0

    if (hasChanges) {
        print("Changes detected!")
        print("Modified files:")
        print(status.stdout)
    } else {
        print("Repository is clean")
    }
} else {
    print("Not a git repository")
}
```

## Passing Arguments to Scripts

> **Note:** Command-line argument passing to gsh scripts is not yet fully implemented. Scripts currently cannot access arguments passed on the command line. This feature is planned for a future release.

For now, you can work around this by using environment variables:

```bash
# Pass data via environment variable
MY_NAME="Alice" gsh greet.gsh
```

Then in your script:

```gsh
#!/usr/bin/env gsh

name = env.MY_NAME
if (name == null || name == "") {
    print("Usage: MY_NAME=name gsh greet.gsh")
    exec("exit 1")
}

print(`Hello, ${name}!`)
```

Run it:

```bash
MY_NAME="Alice" gsh greet.gsh
# Output: Hello, Alice!

MY_NAME="Bob" gsh greet.gsh
# Output: Hello, Bob!
```

## Accessing Environment Variables

Access environment variables with `env.NAME`:

```gsh
#!/usr/bin/env gsh

home = env.HOME
user = env.USER
shell = env.SHELL

print(`Home: ${home}`)
print(`User: ${user}`)
print(`Shell: ${shell}`)

// Set a new environment variable
exec(`export MY_VAR="hello"`)
```

## Running External Commands

Use `exec()` to run bash commands:

```gsh
#!/usr/bin/env gsh

// Simple command
result = exec("echo Hello")
print(result.stdout)  // "Hello"

// Command with arguments
result = exec("ls -la")
print(result.stdout)

// Command with pipes
result = exec("ps aux | grep node")
print(result.stdout)

// Check exit code
result = exec("grep pattern file.txt")
if (result.exitCode != 0) {
    print("Pattern not found")
}
```

## Building Complex Scripts

### Example: Backup Script

```gsh
#!/usr/bin/env gsh

// Backup important directories to a tar file
tool backup_directory(dir: string): string {
    timestamp = exec("date +%Y%m%d_%H%M%S").stdout.trim()
    filename = `backup_${timestamp}.tar.gz`

    result = exec(`tar -czf ${filename} ${dir}`)

    if (result.exitCode == 0) {
        return filename
    } else {
        throw "Backup failed"
    }
}

// Main
directories = [
    env.HOME + "/Documents",
    env.HOME + "/Projects",
    env.HOME + "/.ssh",
]

print("Starting backup...")

for (dir in directories) {
    try {
        filename = backup_directory(dir)
        print(`✓ Backed up to ${filename}`)
    } catch (error) {
        print(`✗ Failed to backup ${dir}: ${error}`)
    }
}

print("Backup complete!")
```

### Example: Log Analyzer

```gsh
#!/usr/bin/env gsh

// Analyze Apache logs

model Analyzer {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

agent LogExpert {
    model: Analyzer,
    system: "You are a log analysis expert. Analyze Apache logs and identify issues.",
}

// Get last 100 lines of logs
logContent = exec("tail -n 100 /var/log/apache2/access.log").stdout

// Ask agent to analyze
print("Analyzing logs...")
analysis = logContent | LogExpert
print(analysis)
```

### Example: Data Pipeline

```gsh
#!/usr/bin/env gsh

// Process CSV data

tool parse_csv(content: string): any[] {
    lines = content.split("\n")
    headers = lines[0].split(",")
    data = []

    for (i = 1; i < lines.length; i = i + 1) {
        values = lines[i].split(",")
        row = {}
        for (j = 0; j < headers.length; j = j + 1) {
            row[headers[j]] = values[j]
        }
        data.push(row)
    }

    return data
}

// Read CSV
csvContent = exec("cat data.csv").stdout
data = parse_csv(csvContent)

// Process each row
for (row in data) {
    // Your processing logic here
    print(row)
}
```

## Debugging Scripts

### Add Debug Output

```gsh
#!/usr/bin/env gsh

DEBUG = true

tool debug(message: string): null {
    if (DEBUG) {
        print(`[DEBUG] ${message}`)
    }
    return null
}

// Use it
debug("Starting process")
debug(`Processing ${items.length} items`)
```

### Check Variables

```gsh
#!/usr/bin/env gsh

tool inspect(name: string, value: any): null {
    print(`${name} = ${JSON.stringify(value, null, 2)}`)
    return null
}

data = {id: 1, name: "test"}
inspect("data", data)
```

### Enable Script Debugging

Run with debug logging:

```bash
GSH_LOG_LEVEL=debug gsh script.gsh
```

Check logs:

```bash
tail -f ~/.gsh.log
```

## Script Best Practices

1. **Always start with shebang**

   ```gsh
   #!/usr/bin/env gsh
   ```

2. **Handle errors explicitly**

   ```gsh
   try {
       // risky operation
   } catch (error) {
       print(`Error: ${error.message}`)
       exec("exit 1")
   }
   ```

3. **Validate inputs**

   ```gsh
   if ($ARGS.length < 2) {
       print("Usage: script.gsh <arg1> <arg2>")
       exec("exit 1")
   }
   ```

4. **Use meaningful tool names**

   ```gsh
   tool process_user_data(user: any): any {
       // ...
   }
   ```

5. **Comment complex sections**

   ```gsh
   // Convert timestamp to human-readable format
   dateStr = new Date(timestamp * 1000).toISOString()
   ```

6. **Exit with appropriate status codes**
   ```gsh
   if (success) {
       exec("exit 0")
   } else {
       exec("exit 1")
   }
   ```

## Integration with REPL

You can source a gsh script in the REPL to make its tools available:

```bash
gsh
gsh> # Tools from script.gsh are now available
```

Or pipe commands to scripts:

```bash
gsh> cat mydata.txt | gsh processor.gsh
```

## Learning More

For deeper dives into gsh scripting, see the [Script Documentation](../script/):

- [Chapter 11: Tool Declarations](../script/11-tool-declarations.md) - More on tools/functions
- [Chapter 14: MCP Servers](../script/14-mcp-servers.md) - Integrating external tools
- [Chapter 17-19: AI and Agents](../script/17-model-declarations.md) - Advanced AI integration
- [Chapter 21: Built-in Functions](../script/21-builtin-functions.md) - Complete function reference

## What's Next?

You've learned the full gsh tutorial! You can now:

1. ✅ Use gsh as an interactive shell
2. ✅ Configure it with `.gshrc` and `.gshrc.gsh`
3. ✅ Create beautiful prompts with Starship
4. ✅ Get AI command predictions
5. ✅ Use agents for complex tasks
6. ✅ Write and execute gsh scripts

Next steps:

- **Explore the [Script Documentation](../script/)** for deeper language features
- **Check out [AGENTS.md](../../AGENTS.md)** for more AI patterns
- **Visit the [GitHub repository](https://github.com/atinylittleshell/gsh)** for examples and community

Happy scripting! 🚀

---

**Previous Chapter:** [Chapter 05: Agents in the REPL](05-agents-in-the-repl.md)
