# Chapter 07: String Manipulation

Strings are everywhere in scripts. You use them for user messages, file paths, commands, prompts, and more. This chapter teaches you to work with text like a pro—from basic operations to advanced patterns.

---

## Why Master String Manipulation?

Before diving into the "how," let's see why this matters:

**Scenario:** You're writing a script that processes user input, formats log messages, and constructs API calls.

```gsh
# Without good string skills, you'd do this:
greeting = "Hello, " + name + "!"
message = "[INFO] " + "Processing " + filename + " at " + timestamp

# With string skills, you can do this:
greeting = `Hello, ${name}!`
message = `[INFO] Processing ${filename} at ${timestamp}`
logs = message.split("\n")
```

Strings are fundamental. Let's master them.

---

## String Literals: Three Ways

gsh gives you three ways to write strings, each with a purpose.

### Double-Quoted Strings

The standard string, like you've seen before:

```gsh
message = "Hello, world!"
path = "C:\\Users\\Alice\\Documents"
print(message)
```

Output:

```
Hello, world!
```

Double-quoted strings support **escape sequences** for special characters:

| Escape | Meaning            |
|--------|--------------------|
| `\n`   | Newline            |
| `\t`   | Tab                |
| `\r`   | Carriage return    |
| `\\`   | Backslash          |
| `\"`   | Double quote       |
| `\'`   | Single quote       |
| `\uXXXX` | Unicode character (4 hex digits) |

```gsh
newline = "Line 1\nLine 2"
tab = "Column 1\tColumn 2"
quote = "She said \"Hello\""
backslash = "Path: C:\\Users"
smiley = "\u263A"

print(newline)
print(tab)
print(quote)
print(backslash)
print(smiley)
```

Output:

```
Line 1
Line 2
Column 1	Column 2
She said "Hello"
Path: C:\Users
☺
```

Unicode escapes are particularly useful for ANSI color codes in terminal output:

```gsh
yellow = "\u001b[38;5;11m"
reset = "\u001b[0m"
print(yellow + "This text is yellow!" + reset)
```

### Single-Quoted Strings

Single quotes create raw strings where escape sequences are **not interpreted**:

```gsh
path = 'C:\Users\Alice\Documents'
regex = 'pattern\d+'
print(path)
print(regex)
```

Output:

```
C:\Users\Alice\Documents
pattern\d+
```

This is useful when you have lots of backslashes (like Windows paths or regex patterns) and don't want to escape them all.

### Triple-Quoted Strings

Triple quotes create **multi-line strings** for long text, prompts, or formatted content:

```gsh
prompt = """
    You are a helpful coding assistant.
    Your task is to analyze code and suggest improvements.
    Be concise and focus on clarity.
    """

print(prompt)
```

Output:

```
You are a helpful coding assistant.
Your task is to analyze code and suggest improvements.
Be concise and focus on clarity.
```

Notice that triple-quoted strings automatically **remove common leading whitespace**. This keeps your code clean while preserving the text formatting.

---

## Template Literals: Bring Data into Strings

Template literals let you embed expressions directly into strings using `${...}` syntax. They use backticks:

```gsh
name = "Alice"
age = 30
greeting = `Hello, ${name}! You are ${age} years old.`
print(greeting)
```

Output:

```
Hello, Alice! You are 30 years old.
```

### Expressions in Templates

You can put any expression inside `${...}`, not just variables:

```gsh
x = 10
y = 20
result = `${x} + ${y} = ${x + y}`
print(result)
```

Output:

```
10 + 20 = 30
```

Accessing properties:

```gsh
user = {name: "Bob", email: "bob@example.com"}
message = `User: ${user.name} (${user.email})`
print(message)
```

Output:

```
User: Bob (bob@example.com)
```

Array indexing:

```gsh
colors = ["red", "green", "blue"]
favorite = `My favorite color is ${colors[0]}`
print(favorite)
```

Output:

```
My favorite color is red
```

---

## The `.length` Property

Every string has a `.length` property that tells you how many characters it contains:

```gsh
message = "Hello"
print(message.length)

empty = ""
print(empty.length)

name = "Alice"
print(`${name} has ${name.length} characters`)
```

Output:

```
5
0
Alice has 5 characters
```

---

## Essential String Methods

String methods are functions you call on a string to do useful work. Here are the most common ones.

### `.split(separator)` — Break Strings Apart

Split a string into an array using a separator:

```gsh
csv = "apple,banana,orange"
fruits = csv.split(",")
print(fruits)
```

Output:

```
[apple, banana, orange]
```

Split on whitespace:

```gsh
text = "one   two   three"
words = text.split(" ")
print(words)
```

Output:

```
[one, , , two, , , three]
```

Note: If you have multiple spaces, you get empty strings between them. Use `.trim()` first if needed (see below).

### `.trim()`, `.trimStart()`, `.trimEnd()` — Remove Whitespace

Remove leading and trailing whitespace:

```gsh
message = "  Hello, world!  "
cleaned = message.trim()
print(`"${cleaned}"`)
```

Output:

```
"Hello, world!"
```

Use `.trimStart()` or `.trimEnd()` to remove whitespace from only one side:

```gsh
message = "  Hello, world!  "
print(`"${message.trimStart()}"`)
print(`"${message.trimEnd()}"`)
```

Output:

```
"Hello, world!  "
"  Hello, world!"
```

These are great for cleaning up user input or text from files.

### `.toUpperCase()` and `.toLowerCase()`

Convert to upper or lowercase:

```gsh
text = "Hello, World!"
print(text.toUpperCase())
print(text.toLowerCase())
```

Output:

```
HELLO, WORLD!
hello, world!
```

### `.includes(search)` — Check for Substring

Check if a string contains another string:

```gsh
email = "user@example.com"
if (email.includes("@")) {
    print("Valid email format")
}

filename = "document.pdf"
if (filename.includes(".pdf")) {
    print("This is a PDF")
}
```

Output:

```
Valid email format
This is a PDF
```

### `.startsWith(prefix)` and `.endsWith(suffix)`

Check what a string starts or ends with:

```gsh
filename = "document.pdf"
if (filename.endsWith(".pdf")) {
    print("It's a PDF file")
}

url = "https://example.com"
if (url.startsWith("https://")) {
    print("Secure connection")
}
```

Output:

```
It's a PDF file
Secure connection
```

### `.replace(search, replacement)` — Replace First Occurrence

Replace the first occurrence of a substring:

```gsh
message = "The quick brown fox jumps over the lazy dog"
updated = message.replace("quick", "slow")
print(updated)
```

Output:

```
The slow brown fox jumps over the lazy dog
```

Notice only the first "quick" is replaced. Use `.replaceAll()` to replace all occurrences.

### `.replaceAll(search, replacement)` — Replace All Occurrences

```gsh
message = "foo bar foo baz foo"
updated = message.replaceAll("foo", "hello")
print(updated)
```

Output:

```
hello bar hello baz hello
```

### `.indexOf(search)` — Find Position

Get the index (position) of a substring. Returns `-1` if not found:

```gsh
text = "Hello, World!"
pos = text.indexOf("World")
print(pos)

notFound = text.indexOf("xyz")
print(notFound)
```

Output:

```
7
-1
```

### `.substring(start, end)` — Extract Part of String

Extract a portion of a string by index. The end index is exclusive:

```gsh
text = "Hello, World!"
part1 = text.substring(0, 5)
print(part1)

part2 = text.substring(7, 12)
print(part2)

rest = text.substring(7)
print(rest)
```

Output:

```
Hello
World
World!
```

### `.slice(start, end)` — Like Substring, with Negative Indices

Similar to `.substring()` but supports negative indices to count from the end:

```gsh
text = "Hello, World!"
print(text.slice(0, 5))      # First 5 characters
print(text.slice(-6))         # Last 6 characters
print(text.slice(0, -1))      # All but the last character
```

Output:

```
Hello
World!
Hello, World
```

### `.charAt(index)` — Get Character at Position

Get a single character at an index:

```gsh
text = "Hello"
print(text.charAt(0))
print(text.charAt(1))
print(text.charAt(10))  # Out of bounds returns empty string
```

Output:

```
H
e

```

### `.repeat(count)` — Repeat String

Repeat a string multiple times:

```gsh
star = "*"
line = star.repeat(10)
print(line)

dash = "-"
print(dash.repeat(20))
```

Output:

```
**********
--------------------
```

### `.padStart(length)` and `.padEnd(length)` — Add Padding

Pad a string to a target length:

```gsh
price = "99"
padded = price.padStart(5, "0")
print(padded)

label = "Name"
padded2 = label.padEnd(10)
print(`"${padded2}"`)
```

Output:

```
00099
"Name      "
```

This is useful for formatting tables or creating fixed-width output.

---

## Real-World Example: Parse CSV

Let's combine what we've learned to parse a CSV line:

```gsh
line = "Alice,alice@example.com,30,engineer"
parts = line.split(",")

name = parts[0]
email = parts[1]
age = parts[2]
role = parts[3]

print(`Name: ${name}`)
print(`Email: ${email}`)
print(`Age: ${age}`)
print(`Role: ${role}`)
```

Output:

```
Name: Alice
Email: alice@example.com
Age: 30
Role: engineer
```

---

## Real-World Example: Clean and Format Log Output

```gsh
logLine = "  ERROR: Connection timeout after 30s  \n"

# Clean it up
cleaned = logLine.trim()

# Check what level it is
if (cleaned.startsWith("ERROR")) {
    print("[CRITICAL]" + cleaned)
} else if (cleaned.startsWith("WARN")) {
    print("[WARNING]" + cleaned)
} else {
    print("[INFO]" + cleaned)
}
```

Output:

```
[CRITICAL]ERROR: Connection timeout after 30s
```

---

## Real-World Example: Validate and Reformat User Input

```gsh
username = "  john_doe  "

# Clean whitespace
username = username.trim()

# Validate format
if (username.length < 3) {
    print("Username too short")
} else if (!username.includes("_") && !username.includes("-")) {
    print("Warning: Username has no separators")
}

# Convert to lowercase for storage
normalized = username.toLowerCase()
print(`Normalized username: ${normalized}`)
```

Output:

```
Warning: Username has no separators
Normalized username: john_doe
```

---

## Combining with Arrays: Join

Remember from Chapter 06 that arrays have a `.join()` method? Here's the complement—string's `.split()`:

```gsh
# Split a string into an array
sentence = "The quick brown fox jumps"
words = sentence.split(" ")
print(words)

# Process the array
words[0] = words[0].toUpperCase()
words[1] = words[1].toUpperCase()

# Join back into a string
result = words.join(" ")
print(result)
```

Output:

```
[The, quick, brown, fox, jumps]
The Quick brown fox jumps
```

---

## Key Takeaways

1. **Three string literal types**: Double-quoted (with escapes), single-quoted (raw), and triple-quoted (multi-line)
2. **Template literals** (`${...}`) embed expressions into strings for readable code
3. **`.length`** tells you how many characters are in a string
4. **String methods** do most of the work: split, trim, includes, replace, slice, and many more
5. **Combine methods**: chain them together like `text.trim().toLowerCase().split(",")`
6. **For data**: use `.split()` to break strings apart, and `.join()` (from arrays) to put them back together

---

## What's Next?

Now that you can manipulate strings fluently, you're ready to **make decisions** with them.

[Next Chapter: Chapter 08 - Conditionals →](08-conditionals.md)

You'll learn `if`, `else if`, and `else` to make your scripts respond intelligently to different inputs and conditions.

---

**Previous Chapter:** [Chapter 06: Arrays and Objects](06-arrays-and-objects.md)

**Next Chapter:** [Chapter 08: Conditionals](08-conditionals.md)
