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
