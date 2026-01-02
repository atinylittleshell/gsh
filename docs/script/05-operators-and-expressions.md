# Chapter 05: Operators and Expressions

You now know how to create values and store them in variables. But a script that only stores data isn't very useful. You need to **do things** with that data—add numbers, compare values, make decisions. That's where operators come in.

An **operator** is a symbol that tells gsh to perform an operation on one or more values. An **expression** is a combination of values and operators that produces a result.

In this chapter, you'll learn:

- Arithmetic operators for calculations
- Comparison operators for testing relationships
- Logical operators for boolean reasoning
- Operator precedence (which operations happen first)
- How different types interact with operators

Let's start with the most familiar operators: arithmetic.

## Arithmetic Operators

Arithmetic operators work on numbers and perform calculations:

```gsh
a = 10
b = 3

print(a + b)    # Addition
print(a - b)    # Subtraction
print(a * b)    # Multiplication
print(a / b)    # Division
print(a % b)    # Remainder (modulo)
```

Output:

```
13
7
30
3.3333333333333335
1
```

Each operator does what you'd expect:

- `+` adds two numbers
- `-` subtracts the right from the left
- `*` multiplies two numbers
- `/` divides the left by the right
- `%` returns the remainder after division

### Division and Remainders

Let's explore division a bit more carefully:

```gsh
print(10 / 4)    # Regular division gives a decimal
print(10 % 4)    # Modulo gives the remainder
```

Output:

```
2.5
2
```

The modulo operator (`%`) is handy for checking if a number divides evenly:

```gsh
number = 15

if (number % 2 == 0) {
    print("15 is even")
} else {
    print("15 is odd")
}
```

Output:

```
15 is odd
```

### Unary Operators

You can also use `+` and `-` as unary operators (operators that work on a single value):

```gsh
x = 5
print(-x)       # Unary minus (negation)
print(+x)       # Unary plus (no change)

y = -10
print(-y)       # Double negation
```

Output:

```
-5
5
10
```

## String Concatenation with `+`

The `+` operator has a special behavior with strings: it concatenates them instead of adding numerically:

```gsh
firstName = "Alice"
lastName = "Smith"
fullName = firstName + " " + lastName

print(fullName)
```

Output:

```
Alice Smith
```

This is powerful because it means `+` adapts to the types it operates on:

```gsh
print("Count: " + "5")          # String + String → Concatenation
print(5 + 3)                     # Number + Number → Addition
print("Number: " + 42)           # String + Number → Concatenation
```

Output:

```
Count: 5
8
Number: 42
```

When one operand is a string and the other isn't, gsh converts the non-string to a string and concatenates.

## Comparison Operators

Comparison operators test relationships between values and always return a boolean (`true` or `false`):

```gsh
x = 10
y = 5

print(x == y)    # Equal to?
print(x != y)    # Not equal to?
print(x < y)     # Less than?
print(x <= y)    # Less than or equal?
print(x > y)     # Greater than?
print(x >= y)    # Greater than or equal?
```

Output:

```
false
true
false
false
true
true
```

Let's break these down:

| Operator | Meaning               | Example           |
| -------- | --------------------- | ----------------- |
| `==`     | Equal to              | `5 == 5` → `true` |
| `!=`     | Not equal to          | `5 != 3` → `true` |
| `<`      | Less than             | `3 < 5` → `true`  |
| `<=`     | Less than or equal    | `5 <= 5` → `true` |
| `>`      | Greater than          | `5 > 3` → `true`  |
| `>=`     | Greater than or equal | `5 >= 5` → `true` |

### Comparing Different Types

Equality comparisons work across types:

```gsh
print(5 == "5")       # Number vs String
print(true == 1)      # Boolean vs Number
print(null == null)   # Null vs Null
print(null == false)  # Null vs Boolean
```

Output:

```
false
false
true
false
```

Notice that `5 == "5"` is `false`—in gsh, a number and a string are different types, even if they have similar representations.

### Comparing Strings

Strings can be compared using all comparison operators (`<`, `<=`, `>`, `>=`). Comparison is lexicographic (dictionary order), character by character:

```gsh
print("apple" < "banana")    # true (a comes before b)
print("abc" < "abd")         # true (c comes before d)
print("ab" < "abc")          # true (prefix is smaller)
print("a" >= "a")            # true (equal strings)
```

Output:

```
true
true
true
true
```

This is especially useful for checking character ranges:

```gsh
c = "5"

# Check if c is a digit
if (c >= "0" && c <= "9") {
    print("'" + c + "' is a digit")
}

# Check if c is a lowercase letter
c = "m"
if (c >= "a" && c <= "z") {
    print("'" + c + "' is a lowercase letter")
}
```

Output:

```
'5' is a digit
'm' is a lowercase letter
```

## Logical Operators

Logical operators work with boolean values and combine conditions:

```gsh
x = 10
y = 5

print((x > y) && (x < 20))    # AND: both must be true
print((x > y) || (x > 20))    # OR: at least one must be true
print(!(x == y))               # NOT: reverses the boolean
```

Output:

```
true
true
true
```

### The `&&` Operator (AND)

`&&` returns `true` only if **both** operands are truthy:

```gsh
age = 25
hasLicense = true

canDrive = (age >= 18) && hasLicense

print(canDrive)
```

Output:

```
true
```

Let's see a case where it's `false`:

```gsh
age = 16
hasLicense = true

canDrive = (age >= 18) && hasLicense

print(canDrive)
```

Output:

```
false
```

Because `age >= 18` is `false`, the whole expression is `false`.

### The `||` Operator (OR)

`||` returns `true` if **at least one** operand is truthy:

```gsh
hasEmail = false
hasPhoneNumber = true

canContact = hasEmail || hasPhoneNumber

print(canContact)
```

Output:

```
true
```

### The `!` Operator (NOT)

`!` reverses a boolean:

```gsh
isActive = true
isInactive = !isActive

print(isActive)
print(isInactive)
```

Output:

```
true
false
```

`!` is a unary operator—it operates on just one value, placed to its left.

### Combining Logical Operators

You can combine multiple logical operators to build complex conditions:

```gsh
age = 25
score = 85
isPassed = true

eligible = (age >= 21) && (score >= 80) && isPassed

print(eligible)
```

Output:

```
true
```

## Operator Precedence

When you have multiple operators in an expression, which one executes first? Precedence determines the order:

```gsh
result = 2 + 3 * 4

print(result)
```

Output:

```
14
```

Multiplication happens before addition (as in math), so this is `2 + (3 * 4) = 2 + 12 = 14`, not `(2 + 3) * 4 = 20`.

Here's the precedence order in gsh (highest to lowest):

1. **Unary operators**: `!`, `-`, `+`
2. **Multiplicative**: `*`, `/`, `%`
3. **Additive**: `+`, `-`
4. **Comparison**: `<`, `<=`, `>`, `>=`
5. **Equality**: `==`, `!=`
6. **Logical AND**: `&&`
7. **Logical OR**: `||`

When in doubt, use parentheses to make your intent clear:

```gsh
# These are different!
result1 = 2 + 3 * 4
result2 = (2 + 3) * 4

print(result1)
print(result2)
```

Output:

```
14
20
```

### Building Complex Expressions

Parentheses let you override precedence and group operations:

```gsh
age = 25
score = 85
isVerified = true

# Check multiple conditions
eligible = (age >= 18 && age <= 65) && (score >= 80 || isVerified)

print(eligible)
```

Output:

```
true
```

Breaking this down:

1. `age >= 18` → `true`
2. `age <= 65` → `true`
3. `true && true` → `true`
4. `score >= 80` → `true`
5. `true || isVerified` → `true`
6. `true && true` → `true`

## Real-World Examples

Let's see operators in practical contexts:

### Example 1: Calculating Discounts

```gsh
originalPrice = 100
discountPercent = 20

discountAmount = originalPrice * (discountPercent / 100)
finalPrice = originalPrice - discountAmount

print(`Original: $${originalPrice}`)
print(`Discount: $${discountAmount}`)
print(`Final: $${finalPrice}`)
```

Output:

```
Original: $100
Discount: $20
Final: $80
```

### Example 2: Validating User Input

```gsh
username = "alice_123"
password = "SecurePass!"

isValidUsername = (username.length > 3) && (username.length < 20)
isValidPassword = (password.length >= 8) && (password != username)

canRegister = isValidUsername && isValidPassword

print(`Username valid: ${isValidUsername}`)
print(`Password valid: ${isValidPassword}`)
print(`Can register: ${canRegister}`)
```

Output:

```
Username valid: true
Password valid: true
Can register: true
```

### Example 3: Temperature Classification

```gsh
temperature = 72

isCold = temperature < 50
isMild = (temperature >= 50) && (temperature < 70)
isWarm = (temperature >= 70) && (temperature < 85)
isHot = temperature >= 85

if (isCold) {
    print("Wear a heavy coat")
} else if (isHot) {
    print("Drink water and stay cool")
} else if (isWarm) {
    print("Light clothing recommended")
} else {
    print("Comfortable temperature")
}
```

Output:

```
Light clothing recommended
```

## Key Takeaways

- **Arithmetic operators** (`+`, `-`, `*`, `/`, `%`) work on numbers
- **String concatenation** uses the `+` operator to join strings
- **Comparison operators** (`==`, `!=`, `<`, `>`, `<=`, `>=`) return booleans
- **Logical operators** (`&&`, `||`, `!`) combine boolean values
- **Operator precedence** determines execution order; use parentheses to be explicit
- **`??` operator** provides fallback values for `null`
- **Expressions combine** values and operators to produce new values

## What's Next?

You now understand how to perform operations and create expressions. But real programs need to work with collections of data. Chapter 06 covers **Arrays and Objects**—how to structure and manipulate multiple values together.

---

**Previous Chapter:** [Chapter 04: Variables and Assignment](04-variables-and-assignment.md)

**Next Chapter:** [Chapter 06: Arrays and Objects](06-arrays-and-objects.md)
