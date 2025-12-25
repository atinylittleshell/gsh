# Chapter 08: Conditionals

Welcome to **control flow**! So far, your scripts have been straightforward: one statement after another, top to bottom. But real programs need to make decisions. "If the user is logged in, show the dashboard. Otherwise, show the login page." That's what this chapter is about.

In gsh, you make decisions with **conditionals**: `if`, `else if`, and `else` statements. They let your code branch based on conditions.

## The Simplest Conditional

Let's start with the simplest form:

```gsh
age = 18

if (age >= 18) {
    print("You are an adult")
}
```

Output:

```
You are an adult
```

Here's what happened:

1. We set `age = 18`
2. We wrote `if (age >= 18)` — this is a **condition**
3. If the condition is true, we execute the code inside the braces `{}`

Let's see what happens when the condition is false:

```gsh
age = 15

if (age >= 18) {
    print("You are an adult")
}

print("Done")
```

Output:

```
Done
```

Notice that the message "You are an adult" didn't print because `age >= 18` is false. But "Done" printed because it's outside the `if` block.

## The `else` Block

What if you want code to run when the condition is **false**? Use `else`:

```gsh
age = 15

if (age >= 18) {
    print("You are an adult")
} else {
    print("You are a minor")
}
```

Output:

```
You are a minor
```

Now we have two branches. The program picks one or the other, but never both.

## Chaining with `else if`

What if you need to check multiple conditions? Use `else if`:

```gsh
score = 85

if (score >= 90) {
    print("Grade: A")
} else if (score >= 80) {
    print("Grade: B")
} else if (score >= 70) {
    print("Grade: C")
} else {
    print("Grade: F")
}
```

Output:

```
Grade: B
```

Here's how it works:

1. Check: Is `score >= 90`? No.
2. Check: Is `score >= 80`? Yes! Execute this block and stop.
3. Don't check any remaining conditions.

This is crucial: **once a condition is true, the remaining `else if` blocks are skipped**.

Let's test with a different score:

```gsh
score = 72

if (score >= 90) {
    print("Grade: A")
} else if (score >= 80) {
    print("Grade: B")
} else if (score >= 70) {
    print("Grade: C")
} else {
    print("Grade: F")
}
```

Output:

```
Grade: C
```

## Conditions and Comparisons

The condition inside parentheses must evaluate to a **boolean** (true or false). In Chapter 05, we saw comparison operators. Let's review them in context:

```gsh
x = 10
y = 5

if (x > y) {
    print("x is greater than y")
}

if (x == y) {
    print("x equals y")
}

if (x != y) {
    print("x is not equal to y")
}
```

Output:

```
x is greater than y
x is not equal to y
```

### Comparison Operators

| Operator | Meaning               | Example           |
| -------- | --------------------- | ----------------- |
| `==`     | Equals                | `5 == 5` → `true` |
| `!=`     | Not equals            | `5 != 3` → `true` |
| `<`      | Less than             | `3 < 5` → `true`  |
| `>`      | Greater than          | `5 > 3` → `true`  |
| `<=`     | Less than or equal    | `5 <= 5` → `true` |
| `>=`     | Greater than or equal | `5 >= 3` → `true` |

## Logical Operators: AND, OR, NOT

You can combine conditions using logical operators:

```gsh
age = 25
hasLicense = true

if (age >= 18 && hasLicense) {
    print("You can drive")
}
```

Output:

```
You can drive
```

The `&&` operator means **AND**: both conditions must be true. Let's change it:

```gsh
age = 15
hasLicense = true

if (age >= 18 && hasLicense) {
    print("You can drive")
} else {
    print("You cannot drive")
}
```

Output:

```
You cannot drive
```

Now `age >= 18` is false, so even though `hasLicense` is true, the whole condition is false.

### The OR Operator (`||`)

Use `||` when you want **at least one** condition to be true:

```gsh
isWeekend = false
isHoliday = true

if (isWeekend || isHoliday) {
    print("No work today!")
}
```

Output:

```
No work today!
```

### The NOT Operator (`!`)

Use `!` to reverse a boolean:

```gsh
isRaining = true

if (!isRaining) {
    print("Let's go outside")
} else {
    print("Stay inside")
}
```

Output:

```
Stay inside
```

### Logical Operators Table

| Operator | Meaning                | Example           | Result  |
| -------- | ---------------------- | ----------------- | ------- |
| `&&`     | AND (both true)        | `true && false`   | `false` |
| `\|\|`   | OR (at least one true) | `true \|\| false` | `true`  |
| `!`      | NOT (reverse)          | `!true`           | `false` |

## Truthiness and Falsiness

We touched on this in Chapter 03, but it's important for conditionals. In gsh, not everything in an `if` condition needs to be an explicit boolean. Some values are automatically treated as true or false:

```gsh
if (1) {
    print("1 is truthy")
}

if (0) {
    print("0 is truthy")
} else {
    print("0 is falsy")
}
```

Output:

```
1 is truthy
0 is falsy
```

Here are the truthy and falsy values:

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

Let's use this in a practical example:

```gsh
name = "Alice"

if (name) {
    print(`Hello, ${name}!`)
} else {
    print("No name provided")
}
```

Output:

```
Hello, Alice!
```

The string `"Alice"` is truthy, so the condition is true. If `name` were an empty string `""`, it would be falsy:

```gsh
name = ""

if (name) {
    print(`Hello, ${name}!`)
} else {
    print("No name provided")
}
```

Output:

```
No name provided
```

## Nested Conditionals

You can put an `if` statement inside another `if` statement:

```gsh
age = 25
hasLicense = true
hasInsurance = true

if (age >= 18) {
    print("Old enough to drive")

    if (hasLicense) {
        print("Has a license")

        if (hasInsurance) {
            print("Can legally drive")
        } else {
            print("Needs insurance")
        }
    } else {
        print("Needs a license")
    }
}
```

Output:

```
Old enough to drive
Has a license
Can legally drive
```

Nested conditionals work, but they can become hard to read. Often, you can simplify using `&&`:

```gsh
age = 25
hasLicense = true
hasInsurance = true

if (age >= 18 && hasLicense && hasInsurance) {
    print("Can legally drive")
}
```

Output:

```
Can legally drive
```

Much clearer!

## Common Patterns

### Pattern 1: Checking for Empty Values

```gsh
data = null

if (data == null) {
    print("No data provided")
} else {
    print(`Data: ${data}`)
}
```

Output:

```
No data provided
```

Or using truthiness:

```gsh
data = ""

if (!data) {
    print("Data is empty")
} else {
    print(`Data: ${data}`)
}
```

Output:

```
Data is empty
```

### Pattern 2: Default Values with `??`

Chapter 13 will cover the `??` operator in detail, but here's a preview—it's useful for providing defaults:

```gsh
port = env.PORT ?? 3000

if (port == 3000) {
    print("Using default port")
} else {
    print(`Using port ${port}`)
}
```

Output (if `PORT` is not set):

```
Using default port
```

### Pattern 3: Early Returns in Tools

When writing tools, you often want to exit early if something is wrong:

```gsh
tool validateAge(age: number): boolean {
    if (age < 0) {
        print("Age cannot be negative")
        return false
    }

    if (age > 150) {
        print("Age seems unrealistic")
        return false
    }

    return true
}

result = validateAge(25)
print(result)
```

Output:

```
true
```

Let's test with invalid input:

```gsh
tool validateAge(age: number): boolean {
    if (age < 0) {
        print("Age cannot be negative")
        return false
    }

    if (age > 150) {
        print("Age seems unrealistic")
        return false
    }

    return true
}

result = validateAge(-5)
print(result)
```

Output:

```
Age cannot be negative
false
```

## Comparison to Loops

You've seen conditionals make decisions. In the next chapter, you'll see **loops** repeat actions. Here's the key difference:

- **`if` statement**: "Execute this block if a condition is true"
- **`while` loop** (next chapter): "Keep executing this block while a condition is true"

Both use conditions, but they behave very differently.

## Key Takeaways

- **`if` statements** make your code make decisions
- **Conditions** go in parentheses: `if (condition) { ... }`
- **`else if` chains** let you check multiple conditions in order
- **`else` blocks** handle cases when no condition is true
- **Logical operators** (`&&`, `||`, `!`) combine conditions
- **Truthiness** means non-boolean values can be used in conditions
- **Nested conditionals** work but can often be simplified with `&&` or `||`

## What's Next?

In Chapter 09, you'll learn **Loops**—how to repeat actions over and over. You'll use `while` loops and `for-of` loops to process data, build collections, and automate repetitive tasks.

---

**Previous Chapter:** [Chapter 07: String Manipulation](07-string-manipulation.md)

**Next Chapter:** [Chapter 09: Loops](09-loops.md)
