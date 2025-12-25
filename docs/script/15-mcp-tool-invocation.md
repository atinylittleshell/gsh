# Chapter 15: MCP Tool Invocation

## Opening

You've learned how to declare MCP servers. Now comes the practical part: actually calling the tools those servers provide. This chapter teaches you how to invoke MCP tools, handle their results, and integrate them into your scripts.

In gsh, MCP tools are invoked using a simple dot notation syntax. Once you've declared a server, its tools are immediately available to call—just like methods on an object.

---

## Core Concepts

### Dot Notation Access

After declaring an MCP server, tools are accessed with dot notation:

```gsh
# After declaring: mcp filesystem { ... }
# You can call:
content = filesystem.read_file("data.json")
files = filesystem.list_directory("/home/user")
```

The pattern is: `serverName.toolName(arguments)`

### Tool Arguments

MCP tools accept arguments in two ways:

**1. Single Object Argument (Most Common)**

Tools typically accept a single object with named parameters:

```gsh
result = filesystem.write_file({
    path: "/tmp/greeting.txt",
    content: "Hello, world!",
})
```

**2. Multiple Arguments (Position-Based)**

For tools with a single parameter, you can pass it directly:

```gsh
# If the tool expects a single "path" parameter
content = filesystem.read_file("/path/to/file.txt")
```

### Tool Results

MCP tools return results as gsh values. The exact type depends on what the tool returns:

- **String results** are returned as `string` values
- **Structured results** are returned as `object` values
- **Multiple results** are returned as `array` values
- **No result** returns `null`

### Error Handling

Tool invocations can fail for various reasons (file not found, network error, permission denied, etc.). Errors should be caught with `try`/`catch` blocks.

---

## Examples

### Example 1: Reading a File

Let's start with a simple file read operation:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

try {
    result = filesystem.read_file({path: "/etc/hostname"})
    content = result.content
    print(`Hostname: ${content}`)
} catch (error) {
    print(`Error: ${error.message}`)
}
```

**Output:**

```
Hostname: mycomputer
```

When this script runs, `filesystem.read_file()` is called with an object containing the `path` parameter. The MCP server reads the file and returns an object with a `content` property containing the file's contents as a string.

### Example 2: Writing a File

Now let's write data to a file:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

message = "gsh script execution successful!"
result = filesystem.write_file({
    path: "/tmp/success.txt",
    content: message,
})

print("File written successfully")
print(result.content)
```

**Output:**

```
File written successfully
Successfully wrote to /tmp/success.txt
```

The `write_file` tool returns an object with a `content` property describing the outcome. The file is now available at `/tmp/success.txt` with the content you specified.

### Example 3: Listing a Directory

List files in a directory with error handling:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

try {
    result = filesystem.list_directory({path: "/home/user/documents"})
    listing = result.content
    print(`Files in directory:`)
    print(listing)
} catch (error) {
    print(`Failed to list directory: ${error.message}`)
}
```

**Output:**

```
Files in directory:
[FILE] report.pdf
[DIR] archive
[FILE] notes.txt
[FILE] spreadsheet.xlsx
```

The `list_directory` tool returns an object with a `content` property containing a formatted listing of the directory contents.

### Example 4: Processing File Contents

Reading and processing data from a file:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

try {
    # Read JSON data from file
    result = filesystem.read_file({path: "/tmp/data.json"})
    jsonContent = result.content
    data = JSON.parse(jsonContent)

    print(`Data contains ${data.length} items`)

    for (item of data) {
        print(`  - Item: ${item.name}`)
    }
} catch (error) {
    print(`Error: ${error.message}`)
}
```

**Output:**

```
Data contains 3 items
  - Item: Alice
  - Item: Bob
  - Item: Charlie
```

### Example 5: Chaining Multiple Tool Calls

Combine multiple tool calls to accomplish a task:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

tool processAndSave(inputPath: string, outputPath: string) {
    try {
        # Read source file
        readResult = filesystem.read_file({path: inputPath})
        content = readResult.content
        print(`Read ${content.length} bytes from ${inputPath}`)

        # Process the content
        lines = content.split("\n")
        processedLines = []
        for (line of lines) {
            if (line.length > 0) {
                processedLines = processedLines + [line.toUpperCase()]
            }
        }

        # Write processed content
        writeResult = filesystem.write_file({
            path: outputPath,
            content: processedLines.join("\n"),
        })

        print(`Wrote processed content to ${outputPath}`)
        return true
    } catch (error) {
        print(`Processing failed: ${error.message}`)
        return false
    }
}

# Use the tool
processAndSave("/tmp/input.txt", "/tmp/output.txt")
```

**Output:**

```
Read 45 bytes from /tmp/input.txt
Wrote processed content to /tmp/output.txt
```

### Example 6: Checking for Tool Existence

Access an MCP server's tools programmatically:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

# The filesystem object itself represents the MCP server
# Individual tools are accessed as properties
print(`Filesystem server: ${filesystem}`)

# Tools are available as callable objects
readTool = filesystem.read_file
print(`Read file tool: ${readTool}`)
```

**Output:**

```
Filesystem server: <mcp server: filesystem>
Read file tool: <mcp tool: filesystem.read_file>
```

### Example 7: Error Handling with Different Failure Modes

Handle various types of errors that can occur:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

tool safeRead(path: string): string {
    try {
        content = filesystem.read_file(path)
        return content
    } catch (error) {
        # Error could be: file not found, permission denied, etc.
        log.error(`Failed to read ${path}: ${error.message}`)
        return ""
    }
}

# Try reading different files
result1 = safeRead("/tmp/exists.txt")
result2 = safeRead("/tmp/nonexistent.txt")
result3 = safeRead("/etc/passwd")  # May fail due to permissions

print("All reads attempted")
```

**Output:**

```
[Error] Failed to read /tmp/nonexistent.txt: file not found
[Error] Failed to read /etc/passwd: permission denied
All reads attempted
```

### Example 8: Using Multiple MCP Servers

Working with different MCP servers in the same script:

```gsh
#!/usr/bin/env gsh

# Declare multiple MCP servers
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

mcp github {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-github"],
    env: {
        GITHUB_TOKEN: env.GITHUB_TOKEN,
    },
}

# Use tools from both servers
try {
    # Read configuration from local file
    config = filesystem.read_file("./config.json")
    log.info(`Loaded config: ${config}`)

    # Get repository information from GitHub
    repo = github.get_repository({
        owner: "myorg",
        repo: "myrepo",
    })
    log.info(`Repository: ${repo.name}`)
} catch (error) {
    log.error(`Operation failed: ${error.message}`)
}
```

**Output:**

```
[Info] Loaded config: {"timeout": 30000, "retries": 3}
[Info] Repository: myrepo
```

### Example 9: Tool Results as Input to Other Operations

Using tool results in downstream processing:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

# Get directory listing
listResult = filesystem.list_directory({path: "/tmp"})
listing = listResult.content

print(`Directory listing:`)
print(listing)

# Process the listing to find text files
# The content is a formatted string, so we split by lines
lines = listing.split("\n")
txtFileCount = 0

for (line of lines) {
    if (line.includes("[FILE]") && line.includes(".txt")) {
        txtFileCount = txtFileCount + 1
    }
}

print(`Found ${txtFileCount} .txt files in listing`)
```

**Output:**

```
Directory listing:
[FILE] notes.txt
[FILE] log.txt
[FILE] config.txt
Found 3 .txt files in listing
```

---

## Key Takeaways

1. **Dot notation** is the primary way to access MCP tools: `server.toolName(arguments)`

2. **Tool arguments** are typically passed as a single object with named properties, but simple values work too

3. **Results vary** depending on what the tool returns—could be strings, objects, arrays, or null

4. **Always use try/catch** when calling MCP tools since they can fail (file not found, permissions, network issues, etc.)

5. **Error messages** from tools provide debugging information—use them to understand what went wrong

6. **Multiple servers** can be declared and used in the same script—each provides its own set of tools

7. **Tool results** can be further processed—use string methods, loops, conditionals, and other tools to transform the data

8. **Chaining** multiple tool calls is the foundation for building powerful scripts that combine multiple MCP servers

---

## What's Next

You've learned how to call MCP tools and handle their results. In the next chapter, we'll explore **Shell Commands**, learning how to execute arbitrary shell commands and integrate their output into your scripts. This gives you the power to leverage existing command-line tools alongside MCP integration.

---

**Previous Chapter:** [Chapter 14: MCP Servers](14-mcp-servers.md)

**Next Chapter:** [Chapter 16: Shell Commands](16-shell-commands.md)
