# Chapter 02: Hello World

Welcome to your first gsh script! In this chapter, you'll write and run a simple script. That's it. No complex examples, no edge cases—just get something working.

## Your First Script

Create a file called `hello.gsh`:

```gsh
print("Hello, world!")
```

That's a complete, valid gsh script.

## Running Your Script

Run it with:

```bash
gsh hello.gsh
```

Output:

```
Hello, world!
```

Congratulations! You've written and executed your first gsh script.

## Making It Executable (Optional)

On Unix-like systems (macOS, Linux), you can make your script executable by adding a shebang line:

```gsh
#!/usr/bin/env gsh

print("Hello, world!")
```

Make it executable:

```bash
chmod +x hello.gsh
```

Now run it directly:

```bash
./hello.gsh
```

Output:

```
Hello, world!
```

The shebang (`#!/usr/bin/env gsh`) tells your system to use gsh to run the script. This is the same idea as Python or bash scripts.

## What's Next?

You've just run your first gsh script! In Chapter 03, we'll explore **Values and Types**—learning about the different kinds of data gsh can work with and how to use them.

---

**Previous Chapter:** [Chapter 01: Introduction](01-introduction.md)

**Next Chapter:** [Chapter 03: Values and Types](03-values-and-types.md)
