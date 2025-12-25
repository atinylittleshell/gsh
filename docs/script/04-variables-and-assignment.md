# Chapter 04: Variables and Assignment

Now that you understand types and values, it's time to learn how to store data in variables so you can use it throughout your scripts. Variables are the foundation of every program—they let you name values, change them, and build more complex logic.

In this chapter, you'll learn:

- What variables are and why they matter
- How to declare and name variables
- How to assign and reassign values
- Type annotations for clarity
- Variable scope and when variables are accessible
- Best practices for naming and organizing variables

## What Are Variables?

A variable is a named storage location that holds a value. Think of it like a labeled box: you put a value inside, give the box a name, and later you can look inside the box to get the value.

```gsh
message = "Hello, world!"
print(message)
```

Output:

```
Hello, world!
```

Here, `message` is a variable that holds the string `"Hello, world!"`. When we `print(message)`, we're asking gsh to look inside the box labeled `message` and display its contents.

## Declaring Variables

In gsh, you declare a variable simply by assigning a value to it:

```gsh
name = "Alice"
age = 30
isActive = true
score = 98.5
items = [1, 2, 3, 4, 5]
user = {name: "Bob", email: "bob@example.com"}

print(name)
print(age)
print(isActive)
print(score)
print(items)
print(user)
```

Output:

```
Alice
30
true
98.5
[1, 2, 3, 4, 5]
{name: Bob, email: bob@example.com}
```

Each line creates a new variable by assigning a value. gsh infers the type from the value you assign, so:

- `name = "Alice"` creates a `string` variable
- `age = 30` creates a `number` variable
- `isActive = true` creates a `boolean` variable
- `items = [1, 2, 3, 4, 5]` creates an `array` variable
- `user = {...}` creates an `object` variable

## Reassigning Variables

Variables in gsh are mutable, meaning you can change their values. Just assign a new value:

```gsh
count = 0
print(count)

count = 1
print(count)

count = count + 1
print(count)

count = count + 1
print(count)
```

Output:

```
0
1
2
3
```

Notice how `count = count + 1` works: it evaluates the expression on the right (`count + 1`), then assigns the result back to `count`. This is a very common pattern for incrementing a counter.

### Updating Collections

You can also update values inside arrays and objects:

```gsh
fruits = ["apple", "banana", "orange"]
print(fruits)

fruits[0] = "apricot"
print(fruits)

user = {name: "Alice", age: 25}
print(user)

user.age = 26
print(user)

user["name"] = "Amy"
print(user)
```

Output:

```
[apple, banana, orange]
[apricot, banana, orange]
{name: Alice, age: 25}
{name: Alice, age: 26}
{name: Amy, age: 26}
```

You can update array elements with bracket notation (`fruits[0] = ...`) and object properties with either dot notation (`user.age = ...`) or bracket notation (`user["name"] = ...`).

## Type Annotations

While gsh infers types automatically, you can explicitly annotate types for clarity or documentation:

```gsh
name: string = "Alice"
age: number = 30
isActive: boolean = true
score: number = 98.5

print(name)
print(age)
print(isActive)
print(score)
```

Output:

```
Alice
30
true
98.5
```

The type annotation comes before the `=` sign, using TypeScript-like syntax:

- `name: string` declares that `name` must hold a string
- `age: number` declares that `age` must hold a number
- `isActive: boolean` declares that `isActive` must hold a boolean
- `score: number` declares that `score` must hold a number

Type annotations are optional—you can always rely on type inference. But they're useful for:

1. **Self-documenting code** - A reader immediately knows what type is expected
2. **Intent clarification** - You're being explicit about what you expect
3. **Catching mistakes** - The interpreter can validate at runtime

## Understanding Scope

Variables exist within a scope, which is a region of your code where the variable is accessible. In gsh, scopes are created by blocks of code (inside `if` statements, loops, tools, etc.):

```gsh
x = "global"
print(x)

if (true) {
    x = "in if block"
    print(x)
}

print(x)
```

Output:

```
global
in if block
in if block
```

Here, when we assign `x = "in if block"` inside the `if` block, it updates the same `x` variable from the outer scope. That's why the final `print(x)` shows `"in if block"`.

### Block Scope and New Variables

When you create a new variable inside a block, it's local to that block:

```gsh
outer = "outside"
print(outer)

if (true) {
    inner = "inside"
    print(inner)
    print(outer)
}

print(outer)
```

Output:

```
outside
inside
outside
outside
```

The variable `inner` only exists inside the `if` block. After the block ends, `inner` is no longer accessible. But `outer` is accessible both inside and outside the block because it was defined in the outer scope.

### Scope in Loops

Variables created inside loops are also scoped to that loop:

```gsh
items = ["apple", "banana", "orange"]

for (item of items) {
    print(item)
}
```

Output:

```
apple
banana
orange
```

The loop variable `item` is automatically created for each iteration and is local to the loop body.

## Common Patterns

### Using Variables to Build Complex Expressions

Variables make complex computations readable:

```gsh
# Without variables - hard to understand
result = (5 + 3) * (10 - 2) / 2

# With variables - clear intent
base = 5 + 3
modifier = 10 - 2
result = base * modifier / 2

print(result)
```

Output:

```
48
```

### Swapping Values

A classic programming pattern:

```gsh
a = "first"
b = "second"

print(a)
print(b)

temp = a
a = b
b = temp

print(a)
print(b)
```

Output:

```
first
second
second
first
```

### Using `null` for Optional Values

When a value might not exist, use `null` as a placeholder:

```gsh
result = null

if (false) {
    result = "found something"
}

print(result)

if (result == null) {
    print("result is null")
}
```

Output:

```
null
result is null
```

## Naming Conventions

Here are some best practices for naming variables:

1. **Use descriptive names** - Avoid single-letter names except for loop counters

   ```gsh
   # Good
   user_email = "alice@example.com"
   total_items = 42

   # Avoid
   x = "alice@example.com"
   n = 42
   ```

2. **Use camelCase or snake_case consistently** - Pick a style and stick with it

   ```gsh
   # camelCase (common in gsh)
   firstName = "Alice"
   emailAddress = "alice@example.com"

   # snake_case (also valid)
   first_name = "Alice"
   email_address = "alice@example.com"
   ```

3. **Use nouns for variables** - Variables hold data, so they should be named like nouns

   ```gsh
   # Good - nouns
   user = "Alice"
   count = 42
   items = [1, 2, 3]

   # Avoid - verbs
   process = "Alice"
   calculate = 42
   ```

4. **Use meaningful names for booleans** - Prefix with `is` or `has`

   ```gsh
   # Good
   isActive = true
   hasPermission = false

   # Less clear
   active = true
   permission = false
   ```

## Variable Assignment and Expression Evaluation

When you assign to a variable, the right-hand side is fully evaluated before the assignment happens:

```gsh
x = 5
y = x + 10
print(y)

x = 100
print(y)
```

Output:

```
15
15
```

Notice that changing `x` afterward doesn't affect `y`. The expression `x + 10` was evaluated to `15` when `y` was assigned, and `y` keeps that value.

## Key Takeaways

- **Variables store values** - Use them to name data and make scripts readable
- **Assignment is simple** - Just use `name = value`
- **Variables are mutable** - You can reassign and update them
- **Type annotations are optional** - Use them for clarity or leave them out to rely on inference
- **Scope matters** - Variables exist within blocks and parent scopes can access child scopes
- **Naming conventions** - Use descriptive names, consistent style, nouns, and meaningful boolean names
- **One assignment per line** - Remember, gsh statements must be on separate lines (no semicolons)

## What's Next?

Now that you can store and use values in variables, Chapter 05 teaches you **Operators and Expressions**—how to perform calculations, comparisons, and logical operations with your variables.

---

**Previous Chapter:** [Chapter 03: Values and Types](03-values-and-types.md)

**Next Chapter:** [Chapter 05: Operators and Expressions](05-operators-and-expressions.md)
