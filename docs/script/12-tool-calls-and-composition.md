# Chapter 12: Tool Calls and Composition

In the previous chapter, you learned how to **declare** tools—how to package code into reusable functions. Now it's time to learn how to **use** them effectively. This chapter is about calling tools, capturing their results, and building complex scripts by composing smaller tools together.

Tool composition is the heart of gsh's design. By chaining simple tools together, you can build powerful scripts that are easy to understand, test, and maintain. This is especially important for agentic workflows—when you compose tools well, AI agents can discover and use them to accomplish complex tasks.

---

## Calling Tools: The Basics

You already know the syntax for calling a tool: write the tool name followed by parentheses with arguments inside.

```gsh
tool greet(name: string): string {
    return "Hello, " + name + "!"
}

message = greet("Alice")
print(message)
```

Output:

```
Hello, Alice!
```

That's it. The tool is called, its body executes, and it returns a value. You can then use that value in your script—assign it to a variable, pass it to another tool, print it, or anything else.

### Capturing Return Values

Tools always return values, and you should capture them. When a tool finishes executing, the last expression in its body is automatically returned (or you can use an explicit `return` statement):

```gsh
tool calculateTax(amount: number): number {
    rate = 0.08
    return amount * rate
}

price = 100
tax = calculateTax(price)
total = price + tax

print("Price: " + price)
print("Tax: " + tax)
print("Total: " + total)
```

Output:

```
Price: 100
Tax: 8
Total: 108
```

The return value is captured in the `tax` variable, and then we use it immediately in the calculation.

### Tools That Return Different Types

Tools can return any type—strings, numbers, arrays, objects, booleans, or even `null`. The return type annotation helps document what you expect:

```gsh
tool getUser(id: number): any {
    # Imagine this queries a database
    return {
        id: id,
        name: "Alice",
        email: "alice@example.com",
    }
}

user = getUser(1)
print("User: " + user.name)
print("Email: " + user.email)
```

Output:

```
User: Alice
Email: alice@example.com
```

Notice how we declared the return type as `any` because we're returning an object. The return value is an object with properties we can access immediately.

---

## Composing Tools: Building from Building Blocks

The real power of tools emerges when you call one tool from inside another. This is called **composition**, and it's how you build complex scripts from simple, understandable pieces.

### Simple Composition

Let's build a series of tools where each one does one thing, and they call each other:

```gsh
tool celsiusToFahrenheit(celsius: number): number {
    return (celsius * 9 / 5) + 32
}

tool formatTemperature(celsius: number): string {
    fahrenheit = celsiusToFahrenheit(celsius)
    return celsius + "°C = " + fahrenheit + "°F"
}

result = formatTemperature(25)
print(result)
```

Output:

```
25°C = 77°F
```

Here, `formatTemperature` calls `celsiusToFahrenheit` inside its body, and then uses the result to build a formatted string. Each tool does one thing well, and they work together.

### Composition with Collections

Composition becomes especially useful when processing collections. You can build tools that transform data step by step:

```gsh
tool getScores(): any {
    return [85, 92, 78, 95, 88]
}

tool filterPassing(scores: any): any {
    result = []
    for (score of scores) {
        if (score >= 80) {
            result.push(score)
        }
    }
    return result
}

tool calculateAverage(scores: any): number {
    sum = 0
    for (score of scores) {
        sum = sum + score
    }
    return sum / scores.length
}

scores = getScores()
print("All scores: " + scores)

passing = filterPassing(scores)
print("Passing scores: " + passing)

average = calculateAverage(passing)
print("Average of passing: " + average)
```

Output:

```
All scores: [85,92,78,95,88]
Passing scores: [85,92,95,88]
Average of passing: 90
```

See how each tool solves one problem:

- `getScores()` retrieves the data
- `filterPassing()` filters it
- `calculateAverage()` computes a summary

Then we call them in sequence, passing results between them.

### Composition with Error Handling

When composing tools, errors in one tool propagate to the caller. You can handle this by wrapping composition in try-catch:

```gsh
tool parseJSON(jsonStr: string): any {
    return JSON.parse(jsonStr)
}

tool extractField(jsonStr: string, field: string): any {
    data = parseJSON(jsonStr)
    return data[field]
}

tool safeExtractField(jsonStr: string, field: string): any {
    try {
        return extractField(jsonStr, field)
    } catch (error) {
        print("Error extracting field: " + error.message)
        return null
    }
}

result = safeExtractField("{\"name\": \"Alice\", \"age\": 30}", "name")
print("Result: " + result)

result2 = safeExtractField("{invalid json}", "name")
print("Result2: " + result2)
```

Output:

```
Result: Alice
Error extracting field: JSON.parse error: invalid character 'i' looking for beginning of object key string
Result2: null
```

Here, `safeExtractField` calls `extractField`, which calls `parseJSON`. If `parseJSON` fails, the error bubbles up through `extractField` and is caught in `safeExtractField`, where we handle it gracefully.

---

## Tool Organization Patterns

As your scripts grow, you'll want to organize tools in ways that make sense. Here are some common patterns:

### Data Pipeline Pattern

Chain tools where each one transforms data:

```gsh
tool readInput(prompt: string): string {
    return input(prompt)
}

tool validateEmail(email: string): boolean {
    return email.includes("@") && email.includes(".")
}

tool normalizeEmail(email: string): string {
    return email.toLowerCase().trim()
}

tool processEmail(): any {
    email = readInput("Enter email: ")

    if (!validateEmail(email)) {
        return {success: false, error: "Invalid email format"}
    }

    normalized = normalizeEmail(email)
    return {success: true, email: normalized}
}

result = processEmail()
print(result)
```

Output (if user enters "ALICE@EXAMPLE.COM"):

```
{success:true,email:alice@example.com}
```

Each tool does one thing: read input, validate, normalize. When composed together, they form a data pipeline.

### Helper Tool Pattern

Create specialized helper tools that other tools use:

```gsh
tool isEven(num: number): boolean {
    return num % 2 == 0
}

tool filterEvenNumbers(numbers: any): any {
    result = []
    for (num of numbers) {
        if (isEven(num)) {
            result.push(num)
        }
    }
    return result
}

tool calculateStats(numbers: any): any {
    evens = filterEvenNumbers(numbers)
    total = 0
    for (num of numbers) {
        total = total + num
    }
    return {
        all: numbers.length,
        evens: evens.length,
        evenPercentage: (evens.length / numbers.length) * 100,
    }
}

numbers = [1, 2, 3, 4, 5, 6, 7, 8]
stats = calculateStats(numbers)
print(stats)
```

Output:

```
{all:8,evens:4,evenPercentage:50}
```

The `isEven` helper is available for other tools to use. Helper tools make your code more modular and easier to test.

### Strategy Pattern

Create different implementations of similar tasks and choose which to use:

```gsh
tool summarizeShort(text: string): string {
    lines = text.split("\n")
    if (lines.length > 0) {
        return lines[0]
    }
    return ""
}

tool summarizeMedium(text: string): string {
    lines = text.split("\n")
    count = 0
    result = []
    for (line of lines) {
        if (count < 3) {
            result.push(line)
            count = count + 1
        }
    }
    return result.join(" ")
}

tool summarize(text: string, strategy: string): string {
    if (strategy == "short") {
        return summarizeShort(text)
    } else if (strategy == "medium") {
        return summarizeMedium(text)
    }
    return text
}

document = `This is a long document.
It has multiple lines.
And lots of information.
We want to summarize it.`

short = summarize(document, "short")
print("Short: " + short)

medium = summarize(document, "medium")
print("Medium: " + medium)
```

Output:

```
Short: This is a long document.
Medium: This is a long document. It has multiple lines. And lots of information.
```

---

## Tool Composition with MCP Tools

You can compose user-defined tools with MCP tools. This is where gsh really shines—your custom logic works seamlessly with external tools:

```gsh
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem", "."],
}

tool processFile(filename: string): any {
    try {
        content = filesystem.read_file(filename)
        lines = content.split("\n")
        lineCount = lines.length
        return {
            success: true,
            filename: filename,
            lineCount: lineCount,
        }
    } catch (error) {
        return {
            success: false,
            error: error.message,
        }
    }
}

tool processMultipleFiles(filenames: any): any {
    results = []
    for (filename of filenames) {
        result = processFile(filename)
        results.push(result)
    }
    return results
}

files = ["file1.txt", "file2.txt"]
results = processMultipleFiles(files)
print(results)
```

Output (depends on file contents):

```
[{success:true,filename:file1.txt,lineCount:5},{success:true,filename:file2.txt,lineCount:3}]
```

Here, the `processFile` tool uses the MCP `filesystem.read_file` tool, and `processMultipleFiles` composes `processFile` with itself in a loop. You're mixing custom logic with external capabilities.

---

## Avoiding Common Pitfalls

### Tool Scope: Return values for changes

Tools can access variables from their outer scope, but modifications inside a tool will affect the outer scope. To avoid unintended side effects, it's best practice to **return modified values instead of relying on mutations**:

```gsh
counter = 0

tool incrementCounter(value: number): number {
    return value + 1
}

counter = incrementCounter(counter)
print(counter)
```

Output:

```
1
```

Here, instead of letting the tool modify `counter` directly, we pass it as a parameter and reassign the returned value. This makes the data flow explicit and your code easier to understand.

If you need to work with outer scope variables inside a tool, you can read them, but be aware that any assignments will affect the outer scope:

```gsh
message = "Hello"

tool appendToMessage(suffix: string): string {
    result = message + " " + suffix
    return result
}

result = appendToMessage("World")
print(result)
print(message)
```

Output:

```
Hello World
Hello
```

Here, we read `message` from the outer scope but return a new value instead of modifying `message` directly.

### Parameter Mismatch: Pass the right number of arguments

If you call a tool with the wrong number of arguments, you'll get an error:

```gsh
tool add(a: number, b: number): number {
    return a + b
}

result = add(5)  # Error: expects 2 arguments, got 1
```

Always count your parameters and match them to your arguments.

### Return Type Mismatches: Handle what you get

If a tool declares a return type, ensure it actually returns that type. The type is checked at runtime:

```gsh
tool getNumber(): number {
    return "not a number"  # This will cause a type error
}
```

---

## Real-World Example: Processing User Data

Let's build a complete example that shows tool composition in action:

```gsh
tool parseUser(userData: string): any {
    # Parse user data from a string
    parts = userData.split(",")
    return {
        name: parts[0],
        email: parts[1],
        age: parts[2],
    }
}

tool validateUser(user: any): any {
    if (!user.email.includes("@")) {
        return {valid: false, reason: "Invalid email"}
    }
    if (user.name.length == 0) {
        return {valid: false, reason: "Empty name"}
    }
    return {valid: true}
}

tool formatUser(user: any): string {
    return user.name + " <" + user.email + ">"
}

tool processUsers(userDataList: any): any {
    results = []
    for (userData of userDataList) {
        user = parseUser(userData)
        validation = validateUser(user)

        if (!validation.valid) {
            results.push({
                user: user,
                status: "invalid",
                reason: validation.reason,
            })
        } else {
            formatted = formatUser(user)
            results.push({
                user: user,
                status: "valid",
                formatted: formatted,
            })
        }
    }
    return results
}

users = ["Alice,alice@example.com,30", "Bob,bob@example.com,25", "Charlie,invalid-email,35"]

for (userData of users) {
    user = parseUser(userData)
    validation = validateUser(user)

    if (!validation.valid) {
        print("INVALID: " + validation.reason)
    } else {
        formatted = formatUser(user)
        print("VALID: " + formatted)
    }
}
```

Output:

```
VALID: Alice <alice@example.com>
VALID: Bob <bob@example.com>
INVALID: Invalid email
```

This example shows:

1. **Decomposition** - Each tool has a single responsibility
2. **Reusability** - Each tool can be called independently or as part of a larger pipeline
3. **Error handling** - `validateUser` provides feedback about failures
4. **Composition** - `processUsers` orchestrates the other tools

---

## Key Takeaways

- **Tools are values**: When you define a tool, you create a value that can be called. You can pass tools around and store them in variables if needed.
- **Composition is powerful**: Build complex scripts by chaining simple tools together. Each tool should do one thing well.
- **Return values are key**: Tools are most useful when they return meaningful values that other tools or code can use.
- **Scope is isolated**: Each tool call gets its own environment. This prevents side effects and makes tools predictable.
- **Errors propagate**: When a called tool encounters an error, that error propagates to the caller. Handle it with try-catch if needed.
- **Mix custom and external tools**: User-defined tools and MCP tools work seamlessly together. This is where gsh shines.

---

## What's Next?

You now know how to write, call, and compose tools. You have the foundation to build scripts that are clean, reusable, and powerful.

Next, we'll move to **Part 5: External Integration**, where you'll learn how to interact with the broader system beyond gsh itself. We'll start with **Chapter 13: Environment Variables**, which teaches you how to access and use the system's environment—secrets, configuration, and system state that your scripts need.

Then you'll learn about **MCP servers** and **shell commands**, which give you access to the full power of external tools and systems.

---

**Next Chapter: [Chapter 13: Environment Variables](13-environment-variables.md)**
