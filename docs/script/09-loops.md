# Chapter 09: Loops

Loops are the secret weapon for processing data at scale. Instead of manually repeating code, loops let you say "do this for each item" or "keep doing this until a condition changes." In this chapter, you'll learn three kinds of loops and how to control them with `break` and `continue`.

## Why Loops Matter

Imagine you have a list of 1,000 usernames and you need to print each one. Without loops, you'd write `print(username1)`, `print(username2)`, ... You'd go crazy. With loops, you write the logic once and let the computer repeat it.

Loops are especially powerful in gsh because:

- You can iterate over arrays and strings seamlessly
- They combine with data processing tasks (reading files, calling tools)
- They're the foundation for building pipelines

## The `for-of` Loop: Iterate Over Collections

The most common loop in gsh is the **`for-of` loop**. It iterates over each element in a collection (array or string).

### Basic Syntax

```gsh
for (variable of collection) {
    # Code here runs once per element
    # Use 'variable' to access the current element
}
```

### Example: Sum Numbers

Let's process a list of numbers:

```gsh
numbers = [10, 20, 30, 40, 50]
total = 0

for (num of numbers) {
    total = total + num
}

print(total)
```

Output:

```
150
```

The loop runs 5 times, once for each number. Each iteration, `num` holds the current value: 10, then 20, then 30, etc.

### Example: Process Strings

Strings are iterable too! You can loop over each character:

```gsh
word = "hello"
count = 0

for (char of word) {
    count = count + 1
}

print(count)
```

Output:

```
5
```

This loops 5 times—once per character. Character-by-character processing is useful for validation, transformation, or analysis.

### Example: Build a New List

You can create new data structures inside loops:

```gsh
names = ["Alice", "Bob", "Charlie"]
greetings = []

for (name of names) {
    greeting = "Hello, " + name + "!"
    greetings.push(greeting)
}

print(greetings)
```

Output:

```
["Hello, Alice!", "Hello, Bob!", "Hello, Charlie!"]
```

Notice: We use the `.push()` method to add items to an array. This modifies the array in place.

### Example: Filter Data

Loops let you selectively collect items:

```gsh
scores = [45, 78, 92, 65, 88, 100]
passing = []

for (score of scores) {
    if (score >= 70) {
        passing.push(score)
    }
}

print(passing)
```

Output:

```
[78, 92, 88, 100]
```

This pattern—iterate, check a condition, collect results—is called **filtering**.

## The `while` Loop: Repeat Until Condition Changes

A `while` loop keeps executing **while** a condition is true. It's perfect when you don't know in advance how many iterations you need.

### Basic Syntax

```gsh
while (condition) {
    # Code here runs as long as condition is true
    # Usually, condition changes inside the loop
}
```

### Example: Count Down

```gsh
n = 5

while (n > 0) {
    print(n)
    n = n - 1
}

print("Blastoff!")
```

Output:

```
5
4
3
2
1
Blastoff!
```

The loop checks `n > 0` before each iteration. When `n` becomes 0, the condition is false, and the loop stops.

### Example: Accumulate Until Threshold

Imagine you're collecting donations and want to know when you hit a goal:

```gsh
donations = [25, 50, 100, 75, 60]
total = 0
index = 0
goal = 150

while (total < goal) {
    total = total + donations[index]
    print("Collected: " + total)
    index = index + 1
}

print("Goal reached!")
```

Output:

```
Collected: 25
Collected: 75
Collected: 175
Goal reached!
```

This loop processes donations from the array until the goal is met.

### Example: Validate User Input

While loops are useful when you need to retry something:

```gsh
attempts = 0
max_attempts = 3

while (attempts < max_attempts) {
    attempts = attempts + 1
    print("Attempt " + attempts)

    if (attempts == 2) {
        print("Success!")
        break
    }
}
```

Output:

```
Attempt 1
Attempt 2
Success!
```

Here, we use `break` to exit the loop early (more on `break` next).

## Controlling Loops: `break` and `continue`

### `break`: Exit Immediately

`break` stops the loop right now, skipping any remaining iterations.

```gsh
numbers = [1, 2, 3, 4, 5]

for (num of numbers) {
    if (num == 3) {
        break
    }
    print(num)
}

print("Done")
```

Output:

```
1
2
Done
```

When `num` equals 3, we `break` out. We never print 3, 4, or 5.

### `continue`: Skip to Next Iteration

`continue` skips the rest of the current iteration and jumps to the next one.

```gsh
numbers = [1, 2, 3, 4, 5]

for (num of numbers) {
    if (num == 3) {
        continue
    }
    print(num)
}
```

Output:

```
1
2
4
5
```

When `num` equals 3, we `continue` to the next iteration. We skip printing 3 but continue with 4 and 5.

### Example: Filter with `continue`

Here's a practical pattern—skip even numbers:

```gsh
sum = 0

for (i of [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]) {
    if (i % 2 == 0) {
        continue
    }
    sum = sum + i
}

print(sum)
```

Output:

```
25
```

We add only odd numbers: 1 + 3 + 5 + 7 + 9 = 25.

### Example: Early Exit with `break`

Find the first item matching a condition:

```gsh
usernames = ["alice", "bob", "charlie", "bob"]
target = "bob"
found_at = -1

for (i of [0, 1, 2, 3]) {
    if (usernames[i] == target) {
        found_at = i
        break
    }
}

print(found_at)
```

Output:

```
1
```

We stop searching as soon as we find "bob" at index 1.

## Nested Loops

Loops can contain other loops. This is useful for processing multi-dimensional data.

### Example: Multiplication Table

```gsh
for (i of [1, 2, 3]) {
    for (j of [1, 2, 3]) {
        product = i * j
        print(i + " × " + j + " = " + product)
    }
}
```

Output:

```
1 × 1 = 1
1 × 2 = 2
1 × 3 = 3
2 × 1 = 2
2 × 2 = 4
2 × 3 = 6
3 × 1 = 3
3 × 2 = 6
3 × 3 = 9
```

The inner loop runs completely for each iteration of the outer loop. This creates 9 combinations (3 × 3).

### Example: Nested `break`

When you `break` in a nested loop, it only breaks the **innermost** loop:

```gsh
for (i of [1, 2, 3]) {
    print("Outer: " + i)
    for (j of [1, 2, 3]) {
        if (j == 2) {
            break
        }
        print("  Inner: " + j)
    }
}
```

Output:

```
Outer: 1
  Inner: 1
Outer: 2
  Inner: 1
Outer: 3
  Inner: 1
```

Even though we `break` from the inner loop, the outer loop continues. Notice how the outer loop still runs 3 times, but the inner loop always stops after 1 iteration.

## Common Loop Patterns

### Pattern 1: Accumulate (Sum, Concatenate)

Process each item and combine results:

```gsh
words = ["Hello", "world", "from", "gsh"]
sentence = ""

for (word of words) {
    sentence = sentence + word + " "
}

print(sentence)
```

Output:

```
Hello world from gsh
```

### Pattern 2: Transform

Create a new collection with modified items:

```gsh
numbers = [1, 2, 3, 4, 5]
doubled = []

for (num of numbers) {
    doubled.push(num * 2)
}

print(doubled)
```

Output:

```
[2, 4, 6, 8, 10]
```

### Pattern 3: Count or Aggregate

Count occurrences or compute statistics:

```gsh
scores = [85, 92, 78, 92, 88, 92]
count_92 = 0

for (score of scores) {
    if (score == 92) {
        count_92 = count_92 + 1
    }
}

print(count_92)
```

Output:

```
3
```

### Pattern 4: Search

Find an item meeting criteria:

```gsh
items = ["apple", "banana", "cherry", "date"]
target = "cherry"
found = false

for (item of items) {
    if (item == target) {
        found = true
        break
    }
}

print(found)
```

Output:

```
true
```

## Real-World Example: Process Files

Let's combine loops with tool calls. Suppose you have multiple files to process:

```gsh
# This example shows the structure; it requires the filesystem MCP server
# mcp filesystem {
#     command: "npx",
#     args: ["-y", "@modelcontextprotocol/server-filesystem"],
# }

files = ["data1.txt", "data2.txt", "data3.txt"]
total_lines = 0

# Note: In a real script, you'd call filesystem.read_file here
# For now, we'll simulate with loop patterns

for (filename of files) {
    # In a real script: content = filesystem.read_file(filename)
    # Then count lines: lines = content.split("\n")
    # Then: total_lines = total_lines + lines.length
}

# This demonstrates loop structure for tool integration
print("Processed files with loops")
```

This pattern—iterate over resources, call tools for each, accumulate results—is common in gsh scripts.

## Key Takeaways

1. **`for-of` loops** are best when you know the collection upfront (arrays, strings)
2. **`while` loops** are best when you don't know how many iterations you need
3. **`break`** exits the loop immediately
4. **`continue`** skips to the next iteration
5. **Nested loops** let you process multi-dimensional data (but break/continue only affect the innermost loop)
6. **Loop patterns** (accumulate, transform, filter, search) solve real problems
7. Loops combine with conditionals to build sophisticated data processing

## What's Next?

Now that you can iterate over data, the next chapter is **Chapter 10: Error Handling**. You'll learn how to handle problems gracefully with `try`/`catch`/`finally` blocks. This is crucial because in real scripts, things go wrong—files aren't found, network calls fail, tools return errors. Error handling lets your scripts be robust and production-ready.

---

**Navigation:**

- [← Chapter 08: Conditionals](08-conditionals.md)
- [Chapter 10: Error Handling →](10-error-handling.md)
