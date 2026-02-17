# Add `++`, `--`, `+=`, `-=` and Other Compound Operators

## Summary

gsh currently does not support `++`, `--`, `+=`, `-=`, `*=`, `/=`, or `%=` operators. Users must use the verbose form (e.g. `x = x + 1`) instead.

## Current State

- **Lexer** (`internal/script/lexer/lexer.go`): Treats `+` and `-` as single-character tokens (`OP_PLUS`, `OP_MINUS`) with no lookahead for `++`, `--`, `+=`, or `-=`. Same for `*`, `/`, `%` — no lookahead for `*=`, `/=`, `%=`.
- **Token types** (`internal/script/lexer/token.go`): No `OP_INCREMENT`, `OP_DECREMENT`, `OP_PLUS_ASSIGN`, `OP_MINUS_ASSIGN`, etc. token types defined.
- **Parser** (`internal/script/parser/`): No AST node for postfix/prefix update expressions or compound assignment. Only prefix unary `!`, `-`, `+` are supported.
- **Interpreter** (`internal/script/interpreter/expressions.go`): `evalUnaryExpression` only handles `!`, `-`, `+`.

## What Needs to Change

### Lexer

- Add `OP_INCREMENT` and `OP_DECREMENT` token types.
- Add lookahead in the `+` and `-` cases to emit `OP_INCREMENT` / `OP_DECREMENT` when a double character is found.

### Parser / AST

- Add an `UpdateExpression` AST node with fields for the operator (`++`/`--`), the operand (must be an identifier or member expression), and whether it's prefix or postfix.
- Parse prefix `++x` / `--x` in `parseUnaryExpression`.
- Parse postfix `x++` / `x--` after parsing a primary expression.

### Interpreter

- Evaluate `UpdateExpression` by reading the current value, incrementing/decrementing by 1, assigning back, and returning the appropriate value (old for postfix, new for prefix).

### Compound Assignment (`+=`, `-=`, `*=`, `/=`, `%=`)

#### Lexer

- Add token types: `OP_PLUS_ASSIGN`, `OP_MINUS_ASSIGN`, `OP_ASTERISK_ASSIGN`, `OP_SLASH_ASSIGN`, `OP_PERCENT_ASSIGN`.
- Add `peekChar()` lookahead in the `+`, `-`, `*`, `/`, `%` cases to check for a following `=`.

#### Parser / AST

- Add a `CompoundAssignmentStatement` AST node (or extend the existing assignment statement) with fields for the operator and the expression.
- Parse `x += expr` as sugar for `x = x + expr`, etc.

#### Interpreter

- Evaluate compound assignment by reading the current value, applying the operator with the right-hand expression, and assigning the result back.

## Workaround

```gsh
# Instead of x++
x = x + 1

# Instead of x--
x = x - 1

# Instead of x += 5
x = x + 5

# Instead of x -= 3
x = x - 3
```
