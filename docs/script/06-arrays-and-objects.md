# Chapter 06: Arrays and Objects

So far, we've worked with individual values: strings, numbers, and booleans. But real programs need to work with collections of data. You might need to store a list of users, a configuration with multiple settings, or unique tags from a dataset. That's where arrays and objects come in.

In this chapter, you'll learn how to structure and manipulate data using gsh's collection types. We'll start with the basics—creating and accessing arrays and objects—then explore more specialized collections like Sets and Maps when you need them.

---

## Arrays: Ordered Collections

An **array** is an ordered list of values. You create arrays using square brackets:

```gsh
fruits = ["apple", "banana", "orange"]
numbers = [1, 2, 3, 4, 5]
mixed = ["hello", 42, true, null]
```

Output when printed:

```
["apple", "banana", "orange"]
[1, 2, 3, 4, 5]
["hello", 42, true, null]
```

### Accessing Elements

Access individual elements by their **index** (position). Indices start at 0:

```gsh
fruits = ["apple", "banana", "orange"]
print(fruits[0])    # First element
print(fruits[1])    # Second element
print(fruits[2])    # Third element
```

Output:

```
apple
banana
orange
```

Accessing an out-of-bounds index throws an error. Always check the length first:

```gsh
fruits = ["apple", "banana"]
index = 10
if (index < fruits.length) {
    print(fruits[index])
} else {
    print("Index out of bounds")
}
```

Output:

```
Index out of bounds
```

### Array Length

Use the `.length` property to find how many elements an array contains:

```gsh
fruits = ["apple", "banana", "orange"]
print(fruits.length)

empty = []
print(empty.length)
```

Output:

```
3
0
```

### Modifying Arrays

Arrays are **mutable**—you can change their contents. Update an element by assigning to an index:

```gsh
fruits = ["apple", "banana", "orange"]
fruits[1] = "grape"
print(fruits)
```

Output:

```
["apple", "grape", "orange"]
```

### Common Array Methods

#### push() - Add to the end

```gsh
items = [1, 2, 3]
items.push(4)
items.push(5, 6)  # Add multiple elements
print(items)
```

Output:

```
[1, 2, 3, 4, 5, 6]
```

#### pop() - Remove from the end

```gsh
items = [1, 2, 3]
removed = items.pop()
print(`Removed: ${removed}`)
print(items)
```

Output:

```
Removed: 3
[1, 2]
```

#### slice() - Extract a portion

```gsh
items = [1, 2, 3, 4, 5]
subset = items.slice(1, 4)  # Elements at indices 1, 2, 3
print(subset)
```

Output:

```
[2, 3, 4]
```

Negative indices work from the end:

```gsh
items = [1, 2, 3, 4, 5]
last_two = items.slice(-2)  # Last 2 elements
print(last_two)
```

Output:

```
[4, 5]
```

#### join() - Combine into a string

```gsh
words = ["hello", "world", "from", "gsh"]
sentence = words.join(" ")
print(sentence)
```

Output:

```
hello world from gsh
```

#### reverse() - Flip the order

```gsh
numbers = [1, 2, 3, 4]
numbers.reverse()
print(numbers)
```

Output:

```
[4, 3, 2, 1]
```

---

## Objects: Key-Value Pairs

An **object** stores data as key-value pairs, like a dictionary or hash map. Create objects using curly braces:

```gsh
user = {name: "Alice", age: 30, email: "alice@example.com"}
config = {host: "localhost", port: 8080, debug: true}
print(user)
print(config)
```

Output when printed:

```
{name: "Alice", age: 30, email: "alice@example.com"}
{host: "localhost", port: 8080, debug: true}
```

### Accessing Properties

Access properties using **dot notation**:

```gsh
user = {name: "Alice", email: "alice@example.com"}
print(user.name)
print(user.email)
```

Output:

```
Alice
alice@example.com
```

Or use **bracket notation** with string keys:

```gsh
user = {name: "Alice", email: "alice@example.com"}
print(user["name"])
print(user["email"])
```

Output:

```
Alice
alice@example.com
```

Bracket notation is useful when your key is dynamic:

```gsh
config = {host: "localhost", port: 8080}
key = "host"
print(config[key])
```

Output:

```
localhost
```

### Modifying Objects

Update properties by assignment:

```gsh
user = {name: "Alice", age: 30}
user.age = 31
user["email"] = "alice@example.com"
print(user)
```

Output:

```
{name: "Alice", age: 31, email: "alice@example.com"}
```

Add new properties the same way:

```gsh
user = {name: "Alice"}
user.city = "New York"
user.country = "USA"
print(user)
```

Output:

```
{name: "Alice", city: "New York", country: "USA"}
```

### Nested Structures

Objects and arrays can contain each other, creating complex nested structures:

```gsh
user = {name: "Alice", address: {street: "123 Main St", city: "New York", country: "USA"}, tags: ["developer", "python", "go"]}
print(user.address.city)
print(user.tags[0])
```

Output:

```
New York
developer
```

Real-world example—parsing API data:

```gsh
response = {status: "success", data: [{id: 1, name: "Product A", price: 29.99}, {id: 2, name: "Product B", price: 49.99}]}
first_product = response.data[0]
print(`${first_product.name} costs $${first_product.price}`)
```

Output:

```
Product A costs $29.99
```

---

## When to Use Each Type

| Type       | Use When                                     | Example                                              |
| ---------- | -------------------------------------------- | ---------------------------------------------------- |
| **Array**  | You have an ordered list of similar items    | `users = ["alice", "bob", "charlie"]`                |
| **Object** | You need named properties for structure      | `user = {name: "alice", email: "alice@example.com"}` |
| **Set**    | You need unique values with no duplicates    | Deduplicate tags or IDs                              |
| **Map**    | You need a key-value store with dynamic keys | Cache results, count occurrences                     |

---

## Sets: Unique Values

A **Set** automatically removes duplicates. Create a Set from an array:

```gsh
tags = Set(["javascript", "python", "javascript", "go", "python"])
print(tags)
```

Output:

```
Set({"javascript", "python", "go"})
```

Use Sets when you need to track unique items without caring about order:

```gsh
seen_ids = Set()
ids = [1, 2, 2, 3, 3, 3, 4]

for (id of ids) {
    seen_ids.add(id)
}

print(seen_ids)
```

Output:

```
Set({1, 2, 3, 4})
```

### Common Set Methods

#### add() - Add an element

```gsh
colors = Set(["red", "blue"])
colors.add("green")
print(colors)
```

Output:

```
Set({"red", "blue", "green"})
```

#### has() - Check if element exists

```gsh
colors = Set(["red", "blue", "green"])
print(colors.has("red"))
print(colors.has("yellow"))
```

Output:

```
true
false
```

#### size - Number of unique elements

```gsh
colors = Set(["red", "blue", "green"])
print(colors.size)
```

Output:

```
3
```

---

## Maps: Flexible Key-Value Storage

A **Map** is like an object but with more flexibility. You can use any string as a key:

```gsh
user_ages = Map()
user_ages.set("alice", 30)
user_ages.set("bob", 25)
user_ages.set("charlie", 35)

print(user_ages.get("bob"))
```

Output:

```
25
```

Create a Map from an array of [key, value] pairs:

```gsh
scores = Map([["alice", 95], ["bob", 87], ["charlie", 92]])
print(scores.get("alice"))
```

Output:

```
95
```

### Common Map Methods

#### set() - Add or update a key

```gsh
cache = Map()
cache.set("user:1", {id: 1, name: "Alice"})
cache.set("user:2", {id: 2, name: "Bob"})
print(cache.size)
```

Output:

```
2
```

Methods chain together:

```gsh
cache = Map()
cache.set("a", 1).set("b", 2).set("c", 3)
print(cache.size)
```

Output:

```
3
```

#### get() - Retrieve a value

```gsh
config = Map([["host", "localhost"], ["port", 8080]])
print(config.get("host"))
print(config.get("missing"))  # Returns null if not found
```

Output:

```
localhost
null
```

#### has() - Check if key exists

```gsh
config = Map([["host", "localhost"], ["port", 8080]])
print(config.has("host"))
print(config.has("timeout"))
```

Output:

```
true
false
```

#### delete() - Remove a key

```gsh
config = Map([["host", "localhost"], ["port", 8080]])
config.delete("port")
print(config.size)
```

Output:

```
1
```

#### size - Number of entries

```gsh
config = Map([["host", "localhost"], ["port", 8080]])
print(config.size)
```

Output:

```
2
```

#### keys() - Get all keys

```gsh
config = Map([["host", "localhost"], ["port", 8080]])
keys = config.keys()
print(keys)
```

Output:

```
["host", "port"]
```

#### values() - Get all values

```gsh
config = Map([["host", "localhost"], ["port", 8080]])
values = config.values()
print(values)
```

Output:

```
["localhost", 8080]
```

---

## Practical Examples

### Example 1: Processing a List

```gsh
prices = [19.99, 29.99, 39.99, 49.99]
total = 0

for (price of prices) {
    total = total + price
}

average = total / prices.length
print(`Total: $${total}, Average: $${average}`)
```

Output:

```
Total: $139.96, Average: $34.99
```

### Example 2: Deduplicating Tags

```gsh
post1_tags = ["javascript", "web", "tutorial"]
post2_tags = ["javascript", "programming", "tips"]
post3_tags = ["web", "design"]

all_tags = Set()
for (tag of post1_tags) { all_tags.add(tag) }
for (tag of post2_tags) { all_tags.add(tag) }
for (tag of post3_tags) { all_tags.add(tag) }

print(all_tags)
```

Output:

```
Set({"javascript", "web", "tutorial", "programming", "tips", "design"})
```

### Example 3: Counting Occurrences

```gsh
words = ["apple", "banana", "apple", "cherry", "banana", "apple"]
word_counts = Map()

for (word of words) {
    count = 0
    existing = word_counts.get(word)
    if (existing != null) {
        count = existing
    }
    word_counts.set(word, count + 1)
}

print(word_counts.get("apple"))
print(word_counts.get("banana"))
print(word_counts.get("cherry"))
```

Output:

```
3
2
1
```

---

## Key Takeaways

- **Arrays** store ordered collections of values, accessed by index
- **Objects** store named key-value pairs for structured data
- **Sets** automatically maintain unique values
- **Maps** provide flexible key-value storage with helpful methods
- Arrays and objects can be nested for complex data structures
- Use `.length` / `.size` to measure collection size
- Use `.push()` and `.pop()` to modify arrays
- Use `.set()` and `.get()` to work with Maps

---

## What's Next

Now that you can work with structured data, you're ready to make decisions based on that data. In the next chapter, [**Conditionals**](07-string-manipulation.md), you'll learn how to write `if` statements to control what your script does based on conditions.

But first, you might want to practice with strings, which appear everywhere in gsh scripts. The chapter after next covers [**String Manipulation**](07-string-manipulation.md) in depth—string methods, templates, and multi-line strings.
