# Chapter 11: Tool Declarations

You've learned how to work with data, make decisions, and loop through collections. Now it's time to take the next step: **organizing your code into reusable pieces**. In gsh, we call these pieces **tools**.

A tool is a function—a chunk of code that does one thing well and returns a result. We call them "tools" (not just "functions") because tools have a special purpose in gsh: **they can be assigned to AI agents**. Later, when you learn about agents, you'll see that agents can use tools to accomplish tasks. By writing tools now, you're not just organizing your own code—you're creating capabilities that AI agents can use. This is central to gsh's design as an agentic scripting language.

For now, think of tools as reusable chunks of code. Instead of copying the same logic throughout your script, you write it once in a tool, then call it from anywhere.

---

## Why Tools Matter

Let's say you're writing a script that processes user data multiple times:

```gsh
# Without tools - repetitive code
json1 = "{\"name\": \"Alice\", \"age\": 30}"
data1 = JSON.parse(json1)
print(data1.name + " is " + data1.age + " years old")

json2 = "{\"name\": \"Bob\", \"age\": 25}"
data2 = JSON.parse(json2)
print(data2.name + " is " + data2.age + " years old")

json3 = "{\"name\": \"Charlie\", \"age\": 35}"
data3 = JSON.parse(json3)
print(data3.name + " is " + data3.age + " years old")
```

This works, but you're typing the same thing over and over. If you need to change how you process data, you'd have to update it in three places. That's error-prone.

With tools, you write it once:

```gsh
tool printPerson(json: string) {
    data = JSON.parse(json)
    print(data.name + " is " + data.age + " years old")
}

printPerson("{\"name\": \"Alice\", \"age\": 30}")
printPerson("{\"name\": \"Bob\", \"age\": 25}")
printPerson("{\"name\": \"Charlie\", \"age\": 35}")
```

Output:

```
Alice is 30 years old
Bob is 25 years old
Charlie is 35 years old
```

Much cleaner! And if you need to change the logic, you only change it in one place.

---

## Anatomy of a Tool

Here's the basic structure:

```gsh
tool toolName(parameter1, parameter2) {
    # Tool body - do work here
    return result
}
```

Let's break it down:

- **`tool`** - The keyword that declares a tool
- **`toolName`** - The name of your tool (use `camelCase`)
- **`(parameter1, parameter2)`** - Parameters the tool accepts (can be empty)
- **`{ ... }`** - The body of the tool (the code that runs)
- **`return`** - Returns a value to the caller (optional; if you don't return, tools return `null`)

---

## Simple Tools: No Parameters, No Return

The simplest tool does work without taking input or producing output:

```gsh
tool greet() {
    print("Hello, gsh!")
}

greet()
```

Output:

```
Hello, gsh!
```

You call a tool just like you'd call a function: write the name followed by parentheses and arguments.

---

## Tools with Parameters

Tools become powerful when they accept parameters. Each parameter becomes a variable inside the tool:

```gsh
tool greet(name) {
    print("Hello, " + name + "!")
}

greet("Alice")
greet("Bob")
```

Output:

```
Hello, Alice!
Hello, Bob!
```

### Multiple Parameters

You can have as many parameters as you need, separated by commas:

```gsh
tool add(x, y) {
    result = x + y
    print(x + " + " + y + " = " + result)
    return result
}

sum = add(5, 3)
print("The sum is: " + sum)
```

Output:

```
5 + 3 = 8
The sum is: 8
```

---

## Return Values

Tools return values with the `return` statement. When the tool hits `return`, it immediately stops and sends the value back to the caller:

```gsh
tool multiply(x, y) {
    return x * y
}

result = multiply(6, 7)
print(result)
```

Output:

```
42
```

### Implicit Returns

If a tool doesn't have an explicit `return` statement, it returns the value of the last expression:

```gsh
tool getCount() {
    count = 10
    count
}

result = getCount()
print(result)
```

Output:

```
10
```

This works, but explicit returns are clearer. Prefer explicit returns in your tools.

---

## Type Annotations

You can annotate parameter and return types using TypeScript-like syntax. The interpreter **validates types at runtime**:

```gsh
tool calculateScore(points: number, multiplier: number): number {
    return points * multiplier
}

score = calculateScore(10, 5)
print("Score: " + score)
```

Output:

```
Score: 50
```

If you pass the wrong type, you get an error:

```gsh
tool calculateScore(points: number): number {
    return points * 2
}

score = calculateScore("invalid")
```

Error:

```
tool calculateScore parameter points expects type number, got string
```

### Return Type Validation

The interpreter also checks that the actual return value matches the declared return type:

```gsh
tool getValue(): number {
    return "this is not a number"
}

value = getValue()
```

Error:

```
tool getValue expected to return number, got string
```

---

## Practical Example: Data Validation

Let's build a real-world tool that validates user data:

```gsh
tool validateEmail(email: string): boolean {
    if (email.length < 5) {
        return false
    }
    if (!email.includes("@")) {
        return false
    }
    if (!email.includes(".")) {
        return false
    }
    return true
}

tool validateAge(age: number): boolean {
    if (age < 0 || age > 150) {
        return false
    }
    return true
}

tool validateUser(email: string, age: number): boolean {
    emailOk = validateEmail(email)
    ageOk = validateAge(age)
    return emailOk && ageOk
}

# Test it out
if (validateUser("alice@example.com", 30)) {
    print("User is valid")
} else {
    print("User validation failed")
}

if (validateUser("invalid-email", 200)) {
    print("User is valid")
} else {
    print("User validation failed")
}
```

Output:

```
User is valid
User validation failed
```

Notice how we composed tools—`validateUser` calls both `validateEmail` and `validateAge`. This is a powerful pattern for building complex logic from simple pieces.

---

## Tools Calling Tools

Tools can call other tools. This is called **composition** and it's one of the most important patterns in gsh:

```gsh
tool square(x: number): number {
    return x * x
}

tool sum(a: number, b: number): number {
    return a + b
}

tool calculate(x: number, y: number): number {
    xSquared = square(x)
    ySquared = square(y)
    total = sum(xSquared, ySquared)
    return total
}

result = calculate(3, 4)
print("3² + 4² = " + result)
```

Output:

```
3² + 4² = 25
```

This follows the mathematical principle: (3² + 4² = 9 + 16 = 25). By building small, focused tools and combining them, you create powerful scripts.

---

## Working with Collections

Tools work great with arrays and objects. Let's build a tool that processes a list:

```gsh
tool sumArray(numbers: any): number {
    total = 0
    for (num of numbers) {
        total = total + num
    }
    return total
}

tool averageArray(numbers: any): number {
    if (numbers.length == 0) {
        return 0
    }
    total = sumArray(numbers)
    return total / numbers.length
}

numbers = [10, 20, 30, 40, 50]
sum = sumArray(numbers)
avg = averageArray(numbers)

print("Sum: " + sum)
print("Average: " + avg)
```

Output:

```
Sum: 150
Average: 30
```

### Processing Objects

Tools are useful for transforming objects too:

```gsh
tool getUserInfo(user: any): any {
    return {
        name: user.name,
        email: user.email,
        isAdmin: user.role == "admin",
    }
}

user = {
    name: "Alice",
    email: "alice@example.com",
    role: "admin",
    password: "secret123",
    lastLogin: "2025-12-20",
}

info = getUserInfo(user)
print("Name: " + info.name)
print("Email: " + info.email)
print("Is Admin: " + info.isAdmin)
```

Output:

```
Name: Alice
Email: alice@example.com
Is Admin: true
```

Notice how we extracted only the fields we need and computed a derived field (`isAdmin`). This is a great use of tools—transforming data into the shape you need.

---

## Error Handling in Tools

Tools can throw errors, and callers can catch them with `try-catch`:

```gsh
tool divide(a: number, b: number): number {
    if (b == 0) {
        return null
    }
    return a / b
}

tool safeDivide(a: number, b: number): any {
    if (b == 0) {
        return {
            success: false,
            error: "Division by zero",
        }
    }
    return {
        success: true,
        result: a / b,
    }
}

result = safeDivide(10, 0)
if (result.success) {
    print("Result: " + result.result)
} else {
    print("Error: " + result.error)
}
```

Output:

```
Error: Division by zero
```

Alternatively, you can use `try-catch` to handle errors that occur inside tools:

```gsh
tool parseJSON(jsonStr: string) {
    data = JSON.parse(jsonStr)
    return data
}

tool safeParseJSON(jsonStr: string): any {
    try {
        data = parseJSON(jsonStr)
        return {
            success: true,
            data: data,
        }
    } catch (error) {
        return {
            success: false,
            error: error.message,
        }
    }
}

result = safeParseJSON("{invalid json}")
if (result.success) {
    print("Parsed: " + result.data)
} else {
    print("Parse error: " + result.error)
}
```

Output:

```
Parse error: JSON.parse error: invalid character 'i' looking for beginning of object key string
```

---

## Tool Scope and Variables

Each time you call a tool, it creates a new **scope** that is enclosed by the parent scope. This means tools can both **read and modify** variables from the outer scope, just like `if` blocks and `for` loops:

```gsh
x = "global"

tool changeX() {
    x = "inside tool"
    print("Inside tool: " + x)
}

print("Before: " + x)
changeX()
print("After: " + x)
```

Output:

```
Before: global
Inside tool: inside tool
After: inside tool
```

When the tool assigns to `x`, it modifies the outer variable because `x` already exists in the parent scope. This makes tools behave consistently with other block types in gsh.

### Creating Local Variables

If you assign to a variable that doesn't exist in the parent scope, a new local variable is created:

```gsh
globalValue = 100

tool localWork() {
    globalValue = 200
    localValue = 50
    print("Global modified in tool: " + globalValue)
    print("Local var: " + localValue)
}

localWork()
print("Global after tool: " + globalValue)
```

Output:

```
Global modified in tool: 200
Local var: 50
Global after tool: 200
```

Notice that `globalValue` is modified (it exists in the outer scope), but `localValue` only exists inside the tool.

### Practical Example: Counters and State

This scoping behavior is useful for maintaining state across tool calls:

```gsh
counter = 0

tool increment() {
    counter = counter + 1
}

tool getCount() {
    return counter
}

increment()
increment()
increment()
print("Count: " + getCount())
```

Output:

```
Count: 3
```

This makes tools flexible for both stateful operations (modifying outer variables) and pure functions (using only parameters and local variables).

---

## Common Patterns: Filtering and Mapping

### Filtering: Keep Only What You Need

```gsh
tool filterNumbers(numbers: any, threshold: number): any {
    result = []
    for (num of numbers) {
        if (num > threshold) {
            result.push(num)
        }
    }
    return result
}

numbers = [5, 12, 3, 18, 7, 25]
filtered = filterNumbers(numbers, 10)
print(filtered)
```

Output:

```
[12,18,25]
```

### Mapping: Transform Every Item

```gsh
tool doubleNumbers(numbers: any): any {
    result = []
    for (num of numbers) {
        result.push(num * 2)
    }
    return result
}

numbers = [1, 2, 3, 4, 5]
doubled = doubleNumbers(numbers)
print(doubled)
```

Output:

```
[2,4,6,8,10]
```

### Reducing: Combine Into One Value

```gsh
tool productNumbers(numbers: any): number {
    result = 1
    for (num of numbers) {
        result = result * num
    }
    return result
}

numbers = [2, 3, 4]
product = productNumbers(numbers)
print("Product: " + product)
```

Output:

```
Product: 24
```

---

## Building Blocks: Creating Reusable Logic

The power of tools is that you can build complex behaviors from simple building blocks:

```gsh
tool isEven(num: number): boolean {
    return num % 2 == 0
}

tool isOdd(num: number): boolean {
    return !isEven(num)
}

tool filterEven(numbers: any): any {
    result = []
    for (num of numbers) {
        if (isEven(num)) {
            result.push(num)
        }
    }
    return result
}

tool filterOdd(numbers: any): any {
    result = []
    for (num of numbers) {
        if (isOdd(num)) {
            result.push(num)
        }
    }
    return result
}

numbers = [1, 2, 3, 4, 5, 6, 7, 8]
evens = filterEven(numbers)
odds = filterOdd(numbers)

print("Evens: " + evens)
print("Odds: " + odds)
```

Output:

```
Evens: [2,4,6,8]
Odds: [1,3,5,7]
```

Each tool does one thing. By combining them, you create more complex behavior.

---

## Key Takeaways

1. **Tools are reusable chunks of code** - Write once, call many times
2. **Parameters pass data in** - Tools accept input through parameters
3. **Return statements pass data out** - Tools send results back to callers
4. **Type annotations add safety** - Declare what types you expect (validated at runtime)
5. **Tools share scope with their parent** - Tools can read and modify outer variables, just like `if`/`for` blocks
6. **Compose tools together** - Call tools from other tools to build complex logic
7. **Tools handle errors** - Use `try-catch` inside tools to handle failures gracefully
8. **Small, focused tools are powerful** - Each tool should do one thing well

---

## What's Next

Now that you can write tools, you're ready to use them with the rest of gsh's powerful features:

- **[Chapter 12: Tool Calls and Composition](12-tool-calls-and-composition.md)** - Master advanced patterns for using tools together
- **[Chapter 13: Environment Variables](13-environment-variables.md)** - Access system state from your tools
- **[Chapter 14: MCP Servers](14-mcp-servers.md)** - Connect your tools to external services

Tools are the foundation of gsh scripts. Everything you build from here on out will use them.

---

**Previous Chapter:** [Chapter 10: Error Handling](10-error-handling.md)

**Next Chapter:** [Chapter 12: Tool Calls and Composition](12-tool-calls-and-composition.md)
