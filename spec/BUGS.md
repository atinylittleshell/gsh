# Known Bugs in gsh Language Implementation

## Tool Scope Leakage (CRITICAL)

**Description:** When a tool assigns to a variable from the parent scope, it modifies the variable in the enclosing scope instead of creating a local copy.

**Expected Behavior:** When a tool executes, it should create a new scope with local copies of any accessible parent scope variables. Assignments to those variables should modify only the local copies, not the originals in the parent scope. This follows standard lexical scoping semantics.

**Actual Behavior:** Assignments inside tools modify variables in the parent scope directly, without creating local isolation.

**Test Case:**

```gsh
x = "global"

tool changeX() {
    x = "inside tool"
    print("Inside: " + x)
}

print("Before: " + x)  # Prints "global"
changeX()              # Prints "Inside: inside tool"
print("After: " + x)   # Should print "global", but prints "inside tool"
```

**Expected Output:**

```
Before: global
Inside: inside tool
After: global
```

**Actual Output:**

```
Before: global
Inside: inside tool
After: inside tool
```

**Impact:** Tools cannot safely work with outer scope variables. Any modification leaks out, breaking encapsulation and making tools prone to unintended side effects. This violates standard function scoping expectations.

**Related Code:**

- `internal/script/interpreter/expressions.go` - `CallTool` function (line ~487)
- `internal/script/interpreter/statements.go` - Tool environment creation
- `internal/script/interpreter/environment.go` - Variable assignment and lookup

**Fix Notes:** The issue is likely in how the tool environment is created and how variable assignment works. When a tool's environment is created, it should:

1. Create a new local environment for the tool execution
2. When assigning to a variable that exists in parent scope, create a new local binding instead of modifying the parent
3. When reading a variable, follow the scope chain upward if not found locally

---

## Anthropic Model Provider Not Implemented

**Description:** The spec (GSH_SCRIPT_SPEC.md) documents support for Anthropic Claude models via `provider: "anthropic"`, but this provider is not implemented in the interpreter.

**Expected Behavior:** Users should be able to declare models with `provider: "anthropic"` and use Claude models in agents, as documented in the spec.

**Actual Behavior:** Scripts that declare `provider: "anthropic"` fail with error: "unknown model provider: anthropic"

**Test Case:**

```gsh
model claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-3-5-sonnet-20241022",
}
```

**Error:** `Runtime error: unknown model provider: anthropic`

**Impact:** Users cannot use Anthropic models despite the spec promising support. Only OpenAI provider is currently implemented.

**Related Code:**

- `internal/script/interpreter/interpreter.go` - Only `NewOpenAIProvider()` is registered (lines 60, 81)
- `internal/script/interpreter/provider.go` - Provider registry interface
- No `provider_anthropic.go` file exists

**Fix Notes:** Need to implement `NewAnthropicProvider()` following the same pattern as `NewOpenAIProvider()` and register it in the interpreter initialization.

---

## Missing Property Access Returns Error Instead of Null

**Description:** Accessing a property that doesn't exist on an object throws a runtime error instead of returning null.

**Expected Behavior:** When accessing a non-existent property on an object (e.g., `user.email` where `user` is `{name: "Alice"}`), the expression should evaluate to `null`, allowing defensive null checks with `??` operator or `== null` comparisons.

**Actual Behavior:** Accessing a non-existent property throws a runtime error: "property 'X' not found on object"

**Test Case:**

```gsh
user = {name: "Alice"}
if (user.email == null) {
    print("email is null")
}
```

**Error:** `Runtime error: property 'email' not found on object (line 2, column 9)`

**Expected Output:**

```
email is null
```

**Impact:** Cannot safely check for optional fields on objects. This prevents common validation patterns and forces users to assume all properties exist. Workaround: explicitly set all properties to null when creating objects, or restructure code to avoid property access on potentially incomplete objects.

**Related Code:**

- `internal/script/interpreter/expressions.go` - Property access evaluation
- `internal/script/interpreter/value.go` - Object property lookup

**Fix Notes:** Property access should check if the property exists and return `null` instead of throwing an error, similar to JavaScript's behavior.
