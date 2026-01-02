# Chapter 19: Conversations and the Pipe Operator

Welcome! By now you've learned how to declare agents and models. But here's the real power: the **pipe operator** (`|`) lets you weave together conversations with AI agents in a natural, intuitive way.

Think of the pipe operator as a conversation flow. Data flows left to right, and at each stage something new happens. Want to start a conversation with an agent? Pipe a string to it. Want to add another message? Pipe another string. Want the agent to respond? Pipe to the agent again. That's it.

In this chapter, you'll master:

- How the pipe operator works with strings, agents, and conversations
- Creating and extending conversations naturally
- Multi-turn dialogues with agents
- Agent handoffs and complex workflows

Let's dive in.

---

## The Pipe Operator Semantics

The pipe operator (`|`) works with three key combinations:

1. **String | Agent** → Create a new conversation and execute the agent
2. **Conversation | String** → Add a user message to an existing conversation
3. **Conversation | Agent** → Execute the agent with the conversation context

Let's explore each one.

---

## Starting a Conversation: String | Agent

The simplest pattern is piping a string directly to an agent. This creates a new conversation with your message as the opening, then immediately executes the agent.

```gsh
#!/usr/bin/env gsh

model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

agent HelpfulAssistant {
    model: exampleModel,
    systemPrompt: "You are a helpful assistant. Answer questions clearly and concisely.",
    tools: [],
}

# Start a conversation by piping a string to an agent
result = "What is the capital of France?" | HelpfulAssistant

# result is now a Conversation object containing the exchange
print(result)
```

**Output:**

```
<conversation with 2 messages>
```

The agent received your message, processed it with its system prompt, and returned a conversation object containing both your message and the agent's response. The conversation holds the full history—user messages, agent responses, everything.

---

## Building Multi-Turn Conversations

Now here's where it gets interesting. After getting an agent's response, you can add more messages and continue the conversation.

To add a user message to a conversation, use `Conversation | String`:

```gsh
#!/usr/bin/env gsh

model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

agent Geographer {
    model: exampleModel,
    systemPrompt: "You are a geography expert. Provide accurate information about cities and countries.",
    tools: [],
}

# Start a conversation
conv = "What is the capital of France?" | Geographer

# Add another message to the conversation
conv = conv | "What about Germany?"

# Add yet another message
conv = conv | "Tell me about Berlin's population"

# Execute the agent with the extended conversation
conv = conv | Geographer

print(conv)
```

**Output:**

```
<conversation with 6 messages>
```

Notice that the conversation grows with each addition. We started with 2 messages (user question + agent response). Then we added two more user messages (2 more messages = 4 total). Finally, executing the agent with `conv | Geographer` added the agent's response, bringing us to 6 messages.

The beauty of this pattern is that the agent sees the full context. When you ask about Berlin's population, the agent remembers that you've been talking about European capitals. It can reference that context naturally.

---

## Chaining in One Expression

You can chain multiple operations together in a single expression, making the conversation flow feel natural and readable:

```gsh
#!/usr/bin/env gsh

model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

agent Tutor {
    model: exampleModel,
    systemPrompt: "You are a patient tutor. Explain concepts clearly, starting with basics.",
    tools: [],
}

# Chain the entire conversation in one flowing expression
result = "Explain photosynthesis" | Tutor
       | "How do the light reactions work?" | Tutor
       | "What is ATP and why is it important?" | Tutor

print(result)
```

**Output:**

```
<conversation with 6 messages>
```

This single expression creates a three-turn conversation:

1. User asks about photosynthesis → Agent responds
2. User asks about light reactions → Agent responds (remembering the context)
3. User asks about ATP → Agent responds (still aware of the full conversation)

This reads like a natural dialogue, and that's intentional. The pipe operator is designed to feel like the flow of conversation itself.

---

## Why Conversations Matter: Context and State

You might wonder: why not just call the agent three times? The answer is **context**. Each call to an agent after the first carries the full conversation history.

Let's see this in action:

```gsh
#!/usr/bin/env gsh

model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

agent ContextfulAssistant {
    model: exampleModel,
    systemPrompt: "You are a helpful assistant. Remember context from earlier in the conversation.",
    tools: [],
}

# Build a conversation with context
conv = "I'm learning Python" | ContextfulAssistant
     | "Show me a for loop example" | ContextfulAssistant
     | "Now show me the same thing using list comprehension" | ContextfulAssistant

print(conv)
```

**Output:**

```
<conversation with 6 messages>
```

In the third turn, the agent knows you're learning Python (from turn 1) and that you want list comprehension as an alternative to for loops (from turn 2). It doesn't need to ask "what language?" or "what construct?"—it has the full context.

This is fundamentally different from three independent agent calls, where each call starts fresh with no memory of previous exchanges.

---

## Agent Handoffs: Switching Agents Mid-Conversation

Sometimes you want to hand off a conversation from one agent to another. The pipe operator makes this elegant:

```gsh
#!/usr/bin/env gsh

model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

agent DataScientist {
    model: exampleModel,
    systemPrompt: "You are a data scientist. You analyze data and suggest ML approaches.",
    tools: [],
}

agent PythonDeveloper {
    model: exampleModel,
    systemPrompt: "You are an expert Python developer. You write clean, efficient code.",
    tools: [],
}

# Start with the data scientist
conv = "I have customer purchase data. How should I analyze it?" | DataScientist

# Hand off to the Python developer
conv = conv | "Now write Python code to implement this analysis" | PythonDeveloper

print(conv)
```

**Output:**

```
<conversation with 4 messages>
```

Here's what happened:

1. DataScientist receives the initial question and responds with an analysis approach
2. We add a new user message: "Now write Python code to implement this analysis"
3. PythonDeveloper receives the full conversation (including the data scientist's suggestion) and responds with code

The Python developer agent sees the full context—the original question, the data scientist's suggestion, and the request for code. It can write code that directly implements the suggested approach.

---

## Working with Tools in Conversations

When agents have tools assigned, they can call those tools during conversations. Let's see how this works:

```gsh
#!/usr/bin/env gsh

model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

tool writeToFile(filepath: string, content: string): string {
    # Simulate writing to a file
    return `Successfully wrote ${content.length} bytes to ${filepath}`
}

agent Assistant {
    model: exampleModel,
    systemPrompt: "You are an assistant that can write files. When asked to create content, use the writeToFile tool.",
    tools: [writeToFile],
}

# Have a conversation where the agent uses tools
conv = "Create a greeting file named hello.txt with the content 'Hello, World!'" | Assistant

print(conv)
```

**Output:**

```
<conversation with 2 messages>
```

When the agent decides to call `writeToFile`, it does so automatically. The tool's result gets added to the conversation history, and if needed, the agent makes another call to provide a final response.

From your perspective as a script author, this is seamless. You pipe a request to the agent, the agent executes its tools as needed, and you get back a conversation with the full history including tool calls and results.

---

## Building Complex Workflows

Let's combine everything into a more realistic workflow: a PR review assistant that uses multiple agents and maintains conversation state.

```gsh
#!/usr/bin/env gsh

model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

tool getCodeQualityMetrics(code: string): string {
    # Simulate getting metrics
    return `Quality score: 8/10, complexity: medium`
}

agent CodeReviewer {
    model: exampleModel,
    systemPrompt: "You are a code reviewer. Review code for quality, performance, and best practices.",
    tools: [getCodeQualityMetrics],
}

agent DocumentationWriter {
    model: exampleModel,
    systemPrompt: "You are a technical writer. You write clear, helpful documentation.",
    tools: [],
}

# Start the workflow
prCode = """
def calculate_total(items):
    total = 0
    for item in items:
        total = total + item.price
    return total
"""

# Phase 1: Code review
workflow = `Review this code:\n${prCode}` | CodeReviewer

# Phase 2: Ask for improvement suggestions
workflow = workflow | "What are the main issues?" | CodeReviewer

# Phase 3: Hand off to documentation writer
workflow = workflow | "Now write documentation for this function" | DocumentationWriter

print(workflow)
```

**Output:**

```
<conversation with 6 messages>
```

This workflow demonstrates:

- Starting a conversation with code context
- Asking follow-up questions that leverage previous context
- Handing off between agents with full conversation history
- Agents calling tools as needed

Each agent understands the full context because conversations preserve history.

---

## Common Patterns and Best Practices

### Pattern 1: Question and Clarification

```gsh
# Ask a complex question, then refine based on the response
result = "Explain machine learning" | Tutor
       | "Can you focus on supervised learning?" | Tutor
       | "Now explain how decision trees work" | Tutor
```

### Pattern 2: Task Decomposition

```gsh
# Break work into phases with different agents
conv = "Analyze this dataset" | Analyst
     | "Now create a visualization strategy" | Visualizer
     | "Finally, write the plotting code" | Developer
```

### Pattern 3: Error Recovery

```gsh
# If an agent response isn't quite right, ask for revision
conv = "Write a function to sort an array" | Coder
     | "That's close, but I need it to handle null values" | Coder
     | "Perfect! Now add error handling" | Coder
```

### Pattern 4: Side-by-Side Comparison

```gsh
# Get different perspectives on the same problem
conv1 = "How would you solve this problem?" | Expert1
conv2 = "How would you solve this problem?" | Expert2

# Each is a separate conversation with full context for that agent
```

---

## Key Takeaways

- **Pipe operator creates natural conversation flow**: `String | Agent` starts, `Conversation | String` adds context, `Conversation | Agent` continues
- **Conversations maintain full history**: Each agent sees all previous messages in the conversation
- **Context is preserved across turns**: Agents can reference earlier parts of the conversation
- **Agent handoffs are elegant**: Seamlessly switch between agents while preserving context
- **Tools integrate naturally**: Agents call tools during conversations, and results stay in history
- **Chaining is readable**: Multi-turn conversations can be written as single, flowing expressions

---

## What's Next

You now understand how to build sophisticated conversations with agents. In the next chapter, we'll explore **Debugging and Troubleshooting**—how to diagnose and fix problems in your gsh scripts, debug agents, and troubleshoot MCP integration.

---

**Previous Chapter:** [Chapter 18: Agent Declarations](18-agent-declarations.md)

**Next Chapter:** [Chapter 20: Debugging and Troubleshooting](20-debugging-and-troubleshooting.md)
