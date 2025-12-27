# Chapter 03: Values and Types

Welcome to the foundation of gsh programming! Every piece of data in a gsh script is a **value** with a **type**. Understanding types is crucial because gsh uses them to catch mistakes early and help you reason clearly about your code.

In this chapter, you'll learn:

- What types gsh supports
- How to write different values (literals)
- How type annotations work
- How types behave at runtime
- Type coercion and conversions

## The Basic Types

gsh has six fundamental types for everyday programming:

| Type      | Purpose                  | Examples                               |
| --------- | ------------------------ | -------------------------------------- |
| `string`  | Text and character data  | `"hello"`, `'world'`, `` `template` `` |
| `number`  | Integers and decimals    | `42`, `3.14`, `-5`                     |
| `boolean` | True or false            | `true`, `false`                        |
| `null`    | The absence of a value   | `null`                                 |
| `any`     | Any value (escape hatch) | Used for dynamic data like JSON        |
| `array`   | Ordered lists of values  | `[1, 2, 3]`, `["a", "b"]`              |

Let's explore each one.

## Strings: Working with Text

Strings represent text. You can create them with single quotes, double quotes, or backticks:

```gsh
message1 = "Hello, world!"
message2 = 'Hello, world!'
message3 = `Hello, world!`

print(message1)
print(message2)
print(message3)
```

Output:

```
Hello, world!
Hello, world!
Hello, world!
```

All three produce identical results. Use whichever feels natural to you.

### Multi-line Strings

For longer text—like prompts for AI models—use triple quotes:

```gsh
prompt = """
    Analyze the following document and extract key themes.
    Format your response as a JSON object with "themes" and "summary" fields.
    Be concise but thorough.
    """

print(prompt)
```

Output:

```
Analyze the following document and extract key themes.
Format your response as a JSON object with "themes" and "summary" fields.
Be concise but thorough.
```

Triple-quoted strings automatically remove common leading whitespace, so your code stays readable.

### String Interpolation

Use backticks and `${}` to embed expressions inside strings:

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

You can put any expression inside `${}`:

```gsh
x = 5
y = 10
result = `${x} + ${y} = ${x + y}`

print(result)
```

Output:

```
5 + 10 = 15
```

## Numbers: Integers and Decimals

In gsh, numbers cover both integers and floating-point values. There's no separate `int` or `float` type—just `number`:

```gsh
count = 42
price = 19.99
temperature = -5
tiny = 0.001

print(count)
print(price)
print(temperature)
print(tiny)
```

Output:

```
42
19.99
-5
0.001
```

Numbers support standard arithmetic operations (which we'll explore in detail in Chapter 05):

```gsh
a = 10
b = 3

print(a + b)
print(a - b)
print(a * b)
print(a / b)
print(a % b)
```

Output:

```
13
7
30
3.3333333333333335
1
```

### Number Methods

Numbers have a `toFixed()` method that formats the number as a string with a specified number of decimal places:

```gsh
pi = 3.14159

print(pi.toFixed(0))   # Round to integer
print(pi.toFixed(1))   # 1 decimal place
print(pi.toFixed(2))   # 2 decimal places

# Useful for display formatting
price = 19.5
print("Price: $" + price.toFixed(2))

# Works on expression results too
ratio = 1 / 3 * 100
print(ratio.toFixed(1) + "%")
```

Output:

```
3
3.1
3.14
Price: $19.50
33.3%
```

Note that `toFixed()` returns a **string**, not a number—perfect for display purposes.

## Booleans: True and False

Booleans are simple: they're either `true` or `false`. You'll use them most often in conditional statements (Chapter 08):

```gsh
isActive = true
isDeleted = false

print(isActive)
print(isDeleted)
```

Output:

```
true
false
```

You can create booleans through comparisons:

```gsh
x = 5
y = 10

isEqual = (x == y)
isLess = (x < y)
isGreater = (x > y)

print(isEqual)
print(isLess)
print(isGreater)
```

Output:

```
false
true
false
```

We'll explore comparisons and logical operators in Chapter 05.

## Null: The Absence of Value

`null` represents "no value." It's useful when a variable hasn't been initialized, when a function returns nothing, or when you explicitly want to represent absence:

```gsh
result = null

if (result == null) {
    print("Result is null")
}
```

Output:

```
Result is null
```

## Arrays: Lists of Values

An array is an ordered collection of values. You create arrays with square brackets:

```gsh
numbers = [1, 2, 3, 4, 5]
names = ["Alice", "Bob", "Charlie"]
mixed = [1, "hello", true, null]

print(numbers)
print(names)
print(mixed)
```

Output:

```
[1, 2, 3, 4, 5]
["Alice", "Bob", "Charlie"]
[1, hello, true, null]
```

### Accessing Array Elements

Access elements using bracket notation with a zero-based index:

```gsh
fruits = ["apple", "banana", "orange"]

print(fruits[0])
print(fruits[1])
print(fruits[2])
```

Output:

```
apple
banana
orange
```

### Array Length

Get the number of elements using the `.length` property:

```gsh
items = ["a", "b", "c", "d"]

print(items.length)
```

Output:

```
4
```

## Objects: Key-Value Data

An object is a collection of key-value pairs. Think of it like a dictionary or map:

```gsh
user = {
    name: "Alice",
    age: 30,
    email: "alice@example.com"
}

print(user)
```

Output:

```
{name: Alice, age: 30, email: alice@example.com}
```

### Accessing Object Properties

Access properties using dot notation or bracket notation:

```gsh
user = {
    name: "Alice",
    age: 30,
    city: "New York"
}

print(user.name)
print(user["age"])
print(user.city)
```

Output:

```
Alice
30
New York
```

### Creating Objects with Variables

You can use the short syntax where the key name matches the variable name:

```gsh
name = "Bob"
email = "bob@example.com"
active = true

user = {name, email, active}

print(user)
```

Output:

```
{name: Bob, email: bob@example.com, active: true}
```

## Advanced Collections: Sets and Maps

For more specialized needs, gsh provides `Set` and `Map`:

### Sets: Unique Values

A `Set` automatically removes duplicates:

```gsh
tags = Set(["javascript", "python", "javascript", "go", "python"])

print(tags)
```

Output:

```
{go, javascript, python}
```

Sets are useful when you only care about uniqueness, not order. We'll use them less frequently than arrays and objects, but they're helpful for deduplication.

### Maps: Key-Value with Non-String Keys

A `Map` is similar to an object but allows any type as a key. They're useful when you need flexible key types:

```gsh
ages = Map([["alice", 25], ["bob", 30], ["charlie", 35]])

print(ages)
```

Output:

```
Map({"alice": 25, "bob": 30, "charlie": 35})
```

Maps are particularly useful for cases where object notation won't work, such as when keys are numbers or other non-string values.

## Type Annotations

So far, we haven't explicitly declared types. gsh infers them automatically. But you can annotate types explicitly for clarity:

```gsh
name: string = "Alice"
age: number = 30
isActive: boolean = true

print(name)
print(age)
print(isActive)
```

Output:

```
Alice
30
true
```

For arrays, use bracket notation:

```gsh
numbers: number[] = [1, 2, 3]
names: string[] = ["Alice", "Bob"]

print(numbers)
print(names)
```

Output:

```
[1, 2, 3]
[Alice, Bob]
```

## The `any` Type

Sometimes you're working with dynamic data where you don't know the exact type in advance. Use `any`:

```gsh
data: any = { message: "hello", count: 42 }

print(data)
```

Output:

```
{message: hello, count: 42}
```

You'll use `any` most often when parsing JSON or working with data from external tools. We'll see `any` in action in later chapters when we work with MCP tools and APIs.

## Type Checking at Runtime

gsh doesn't force type safety at compile time (the language isn't statically typed), but types still matter at runtime. Values maintain their types throughout execution:

```gsh
x = 42
y = "hello"
z = true

if (x == 42) {
    print("x is the number 42")
}

if (y == "hello") {
    print("y is the string hello")
}

if (z == true) {
    print("z is boolean true")
}
```

Output:

```
x is the number 42
y is the string hello
z is boolean true
```

## Truthiness and Falsiness

In gsh, when you use a value in a boolean context (like an `if` condition), certain values are "truthy" and others are "falsy":

| Value                | Truthiness |
| -------------------- | ---------- |
| `true`               | Truthy     |
| `false`              | Falsy      |
| `null`               | Falsy      |
| `0`                  | Falsy      |
| Any non-zero number  | Truthy     |
| `""` (empty string)  | Falsy      |
| Any non-empty string | Truthy     |
| `[]` (empty array)   | Falsy      |
| Any non-empty array  | Truthy     |
| `{}` (empty object)  | Falsy      |
| Any non-empty object | Truthy     |

Here's truthiness in action:

```gsh
if (1) {
    print("1 is truthy")
}

if (0) {
    print("0 is truthy")
} else {
    print("0 is falsy")
}

if ("") {
    print("empty string is truthy")
} else {
    print("empty string is falsy")
}

if ([]) {
    print("empty array is truthy")
} else {
    print("empty array is falsy")
}
```

Output:

```
1 is truthy
0 is falsy
empty string is falsy
empty array is falsy
```

This is useful for checking if values are "empty" without explicit `null` or `== 0` checks. We'll use this pattern frequently in real scripts.

## Type Coercion in Operations

When you use values in operations, gsh sometimes converts them automatically. For example:

```gsh
result1 = "Hello" + " " + "World"
print(result1)

result2 = 5 + 3
print(result2)

result3 = "5" + "3"
print(result3)
```

Output:

```
Hello World
8
53
```

Notice the difference:

- `5 + 3` performs arithmetic (result is `8`)
- `"5" + "3"` concatenates strings (result is `"53"`)

The `+` operator's behavior depends on the types involved. We'll explore operators in depth in Chapter 05.

## Key Takeaways

- **Six basic types**: `string`, `number`, `boolean`, `null`, `any`, and `array`
- **Collections**: Arrays (ordered), Objects (key-value), Sets (unique), Maps (flexible keys)
- **String operations**: Single quotes, double quotes, backticks with interpolation
- **Type annotations**: Optional but helpful for clarity
- **Truthiness**: Understanding which values are truthy/falsy matters for conditionals
- **Type coercion**: Operations like `+` behave differently based on types

## What's Next?

Now that you understand types and values, Chapter 04 dives into **Variables and Assignment**—how to store these values, name them, and use them throughout your scripts.

---

**Previous Chapter:** [Chapter 02: Hello World](02-hello-world.md)

**Next Chapter:** [Chapter 04: Variables and Assignment](04-variables-and-assignment.md)
