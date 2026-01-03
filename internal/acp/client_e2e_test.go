//go:build e2e
// +build e2e

package acp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestRovoDevE2E tests the ACP client with the real Rovo Dev agent.
// Run with: go test -v -tags=e2e ./internal/acp/... -run TestRovoDevE2E
//
// Prerequisites:
// - acli must be installed and in PATH
// - User must be authenticated with Atlassian (run `acli auth login` first)
func TestRovoDevE2E(t *testing.T) {
	// Skip if acli is not available
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	// Create client configuration
	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	// Connect to the agent
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("Connecting to Rovo Dev...")
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully!")

	// Log capabilities
	caps := client.AgentCapabilities()
	t.Logf("Agent capabilities: loadSession=%v, protocolVersion=%d",
		caps.LoadSession, client.ProtocolVersion())

	// Create a new session
	t.Log("Creating new session...")
	session, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	t.Logf("Session created: %s", session.SessionID())

	// Send a simple prompt
	t.Log("Sending prompt...")
	var chunks []string
	var toolCalls []string

	result, err := session.SendPrompt(ctx, "What is 2 + 2? Reply with just the number.", func(update *SessionUpdateParams) {
		switch update.Update.SessionUpdate {
		case SessionUpdateAgentMessageChunk:
			if content := update.Update.GetMessageContent(); content != nil {
				chunks = append(chunks, content.Text)
				t.Logf("Chunk: %q", content.Text)
			}
		case SessionUpdateToolCall:
			toolCalls = append(toolCalls, update.Update.Name)
			t.Logf("Tool call: %s", update.Update.Name)
		case SessionUpdateToolCallUpdate:
			t.Logf("Tool update: %s status=%s", update.Update.ToolCallID, update.Update.Status)
		}
	})

	if err != nil {
		t.Fatalf("Failed to send prompt: %v", err)
	}

	t.Logf("Prompt completed with stop reason: %s", result.StopReason)

	// Verify we got a response
	fullResponse := strings.Join(chunks, "")
	t.Logf("Full response: %q", fullResponse)

	if fullResponse == "" {
		t.Error("Expected non-empty response")
	}

	// Check message history
	messages := session.GetMessages()
	t.Logf("Message count: %d", len(messages))

	if len(messages) < 2 {
		t.Errorf("Expected at least 2 messages (user + assistant), got %d", len(messages))
	}

	// Verify first message is user
	if len(messages) > 0 && messages[0].Role != "user" {
		t.Errorf("Expected first message to be user, got %s", messages[0].Role)
	}

	// Verify last message
	lastMsg := session.GetLastMessage()
	if lastMsg == nil {
		t.Fatal("Expected last message, got nil")
	}
	t.Logf("Last message role: %s", lastMsg.Role)
	t.Logf("Last message content: %q", lastMsg.Content)

	if lastMsg.Role != "assistant" {
		t.Errorf("Expected last message role 'assistant', got %s", lastMsg.Role)
	}
}

// TestRovoDevConversationE2E tests multi-turn conversation with Rovo Dev.
func TestRovoDevConversationE2E(t *testing.T) {
	// Skip if acli is not available
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Connecting to Rovo Dev...")
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	session, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	t.Logf("Session ID: %s", session.SessionID())

	// First turn
	t.Log("Turn 1: Asking about a number...")
	_, err = session.SendPrompt(ctx, "Remember this number: 42. Just say 'OK' to acknowledge.", func(update *SessionUpdateParams) {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			if content := update.Update.GetMessageContent(); content != nil {
				t.Logf("T1 chunk: %q", content.Text)
			}
		}
	})
	if err != nil {
		t.Fatalf("Turn 1 failed: %v", err)
	}

	// Second turn - test context retention
	t.Log("Turn 2: Asking to recall the number...")
	var turn2Response strings.Builder
	_, err = session.SendPrompt(ctx, "What number did I ask you to remember? Reply with just the number.", func(update *SessionUpdateParams) {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			if content := update.Update.GetMessageContent(); content != nil {
				turn2Response.WriteString(content.Text)
				t.Logf("T2 chunk: %q", content.Text)
			}
		}
	})
	if err != nil {
		t.Fatalf("Turn 2 failed: %v", err)
	}

	response := turn2Response.String()
	t.Logf("Turn 2 full response: %q", response)

	// Check if 42 is mentioned in the response
	if !strings.Contains(response, "42") {
		t.Errorf("Expected response to contain '42', got: %s", response)
	}

	// Verify message history has all turns
	messages := session.GetMessages()
	t.Logf("Total messages: %d", len(messages))

	// Should have: user1, assistant1, user2, assistant2 = 4 messages
	if len(messages) < 4 {
		t.Errorf("Expected at least 4 messages for 2 turns, got %d", len(messages))
	}

	for i, msg := range messages {
		t.Logf("Message %d: role=%s content=%q", i, msg.Role, truncate(msg.Content, 50))
	}
}

// TestRovoDevStreamingE2E verifies that streaming updates are received correctly.
func TestRovoDevStreamingE2E(t *testing.T) {
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	session, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Ask for a longer response to ensure we get multiple chunks
	t.Log("Sending prompt expecting multiple chunks...")
	var chunkCount int
	var updateTypes = make(map[string]int)

	_, err = session.SendPrompt(ctx, "Count from 1 to 5, one number per line.", func(update *SessionUpdateParams) {
		updateTypes[update.Update.SessionUpdate]++
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			chunkCount++
		}
	})
	if err != nil {
		t.Fatalf("Failed to send prompt: %v", err)
	}

	t.Logf("Received %d chunks", chunkCount)
	t.Logf("Update types: %v", updateTypes)

	if chunkCount == 0 {
		t.Error("Expected to receive streaming chunks")
	}

	// Verify we received agent_message_chunk updates
	if updateTypes[SessionUpdateAgentMessageChunk] == 0 {
		t.Error("Expected agent_message_chunk updates")
	}
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// TestRovoDevToolCallsE2E tests that tool calls are properly tracked.
func TestRovoDevToolCallsE2E(t *testing.T) {
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	session, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Ask something that should trigger tool usage (file reading)
	t.Log("Sending prompt that should trigger tool calls...")
	var toolCallIDs []string
	var toolCallNames []string
	var toolUpdates []string
	var chunks []string

	_, err = session.SendPrompt(ctx, "Read the contents of the go.mod file in this directory and tell me the module name. Be concise.", func(update *SessionUpdateParams) {
		switch update.Update.SessionUpdate {
		case SessionUpdateAgentMessageChunk:
			if content := update.Update.GetMessageContent(); content != nil {
				chunks = append(chunks, content.Text)
			}
		case SessionUpdateToolCall:
			toolCallIDs = append(toolCallIDs, update.Update.ToolCallID)
			toolCallNames = append(toolCallNames, update.Update.GetToolName())
			t.Logf("Tool call started: name=%q id=%s title=%q kind=%q",
				update.Update.GetToolName(), update.Update.ToolCallID, update.Update.Title, update.Update.Kind)
		case SessionUpdateToolCallUpdate:
			toolUpdates = append(toolUpdates, fmt.Sprintf("%s:%s", update.Update.ToolCallID, update.Update.Status))
			t.Logf("Tool update: id=%s status=%s name=%q",
				update.Update.ToolCallID, update.Update.Status, update.Update.GetToolName())
		}
	})

	if err != nil {
		t.Fatalf("Failed to send prompt: %v", err)
	}

	t.Logf("Tool call IDs: %v", toolCallIDs)
	t.Logf("Tool call names: %v", toolCallNames)
	t.Logf("Tool updates received: %d", len(toolUpdates))
	t.Logf("Response: %s", strings.Join(chunks, ""))

	// We expect at least one tool call event (either tool_call or tool_call_update)
	totalToolEvents := len(toolCallIDs) + len(toolUpdates)
	if totalToolEvents == 0 {
		t.Log("Warning: No tool events detected (agent may have answered from context)")
	} else {
		t.Logf("Successfully tracked %d tool event(s)", totalToolEvents)
	}

	// Response should mention the module name
	response := strings.Join(chunks, "")
	if !strings.Contains(response, "gsh") && !strings.Contains(response, "github.com") {
		t.Logf("Response may not contain expected module info: %s", response)
	}
}

// TestRovoDevEmptyPromptE2E tests behavior with edge case prompts.
func TestRovoDevEmptyPromptE2E(t *testing.T) {
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	session, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test with a minimal prompt
	t.Log("Sending minimal prompt...")
	var gotResponse bool

	_, err = session.SendPrompt(ctx, "Hi", func(update *SessionUpdateParams) {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			gotResponse = true
		}
	})

	if err != nil {
		t.Fatalf("Failed with minimal prompt: %v", err)
	}

	if !gotResponse {
		t.Error("Expected response to minimal prompt")
	}

	lastMsg := session.GetLastMessage()
	if lastMsg == nil || lastMsg.Content == "" {
		t.Error("Expected non-empty response")
	}
	t.Logf("Response to 'Hi': %s", truncate(lastMsg.Content, 100))
}

// TestRovoDevMultipleSessionsE2E tests creating multiple independent sessions.
func TestRovoDevMultipleSessionsE2E(t *testing.T) {
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create first session
	session1, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}
	t.Logf("Session 1 ID: %s", session1.SessionID())

	// Create second session
	session2, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}
	t.Logf("Session 2 ID: %s", session2.SessionID())

	// Verify they have different IDs
	if session1.SessionID() == session2.SessionID() {
		t.Error("Expected different session IDs")
	}

	// Send different info to each session
	t.Log("Sending to session 1: remember 'apple'")
	_, err = session1.SendPrompt(ctx, "Remember the word 'apple'. Just say OK.", func(update *SessionUpdateParams) {})
	if err != nil {
		t.Fatalf("Session 1 prompt failed: %v", err)
	}

	t.Log("Sending to session 2: remember 'banana'")
	_, err = session2.SendPrompt(ctx, "Remember the word 'banana'. Just say OK.", func(update *SessionUpdateParams) {})
	if err != nil {
		t.Fatalf("Session 2 prompt failed: %v", err)
	}

	// Verify each session remembers its own word
	var response1 strings.Builder
	_, err = session1.SendPrompt(ctx, "What word did I ask you to remember? Reply with just the word.", func(update *SessionUpdateParams) {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			if content := update.Update.GetMessageContent(); content != nil {
				response1.WriteString(content.Text)
			}
		}
	})
	if err != nil {
		t.Fatalf("Session 1 recall failed: %v", err)
	}

	var response2 strings.Builder
	_, err = session2.SendPrompt(ctx, "What word did I ask you to remember? Reply with just the word.", func(update *SessionUpdateParams) {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			if content := update.Update.GetMessageContent(); content != nil {
				response2.WriteString(content.Text)
			}
		}
	})
	if err != nil {
		t.Fatalf("Session 2 recall failed: %v", err)
	}

	t.Logf("Session 1 recalled: %s", response1.String())
	t.Logf("Session 2 recalled: %s", response2.String())

	// Check isolation
	r1 := strings.ToLower(response1.String())
	r2 := strings.ToLower(response2.String())

	if !strings.Contains(r1, "apple") {
		t.Errorf("Session 1 should remember 'apple', got: %s", r1)
	}
	if !strings.Contains(r2, "banana") {
		t.Errorf("Session 2 should remember 'banana', got: %s", r2)
	}
	if strings.Contains(r1, "banana") {
		t.Error("Session 1 should not know about 'banana'")
	}
	if strings.Contains(r2, "apple") {
		t.Error("Session 2 should not know about 'apple'")
	}
}

// TestRovoDevLongResponseE2E tests handling of longer responses with many chunks.
func TestRovoDevLongResponseE2E(t *testing.T) {
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	session, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Ask for a longer response
	t.Log("Requesting longer response...")
	var chunkCount int
	var totalLength int
	var chunks []string

	_, err = session.SendPrompt(ctx, "Write a short poem (4-8 lines) about coding. Include line breaks.", func(update *SessionUpdateParams) {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			if content := update.Update.GetMessageContent(); content != nil {
				chunkCount++
				totalLength += len(content.Text)
				chunks = append(chunks, content.Text)
			}
		}
	})

	if err != nil {
		t.Fatalf("Failed to get long response: %v", err)
	}

	fullResponse := strings.Join(chunks, "")
	t.Logf("Received %d chunks, total %d characters", chunkCount, totalLength)
	t.Logf("Response:\n%s", fullResponse)

	// Verify we got a reasonable response
	if chunkCount < 2 {
		t.Logf("Warning: Expected multiple chunks for longer response, got %d", chunkCount)
	}

	if totalLength < 50 {
		t.Errorf("Expected longer response, got %d characters", totalLength)
	}

	// Verify the response is stored correctly in message history
	lastMsg := session.GetLastMessage()
	if lastMsg == nil {
		t.Fatal("Expected last message")
	}
	if lastMsg.Content != fullResponse {
		t.Errorf("Message history content doesn't match streamed content.\nStreamed: %q\nStored: %q", fullResponse, lastMsg.Content)
	}
}

// TestRovoDevClientReconnectE2E tests that we can create a new client after closing one.
func TestRovoDevClientReconnectE2E(t *testing.T) {
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// First connection
	t.Log("Creating first client...")
	client1 := NewClient(config)
	if err := client1.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}

	session1, err := client1.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}

	_, err = session1.SendPrompt(ctx, "Say 'first'", func(update *SessionUpdateParams) {})
	if err != nil {
		t.Fatalf("First prompt failed: %v", err)
	}
	t.Log("First client worked")

	// Close first client
	t.Log("Closing first client...")
	if err := client1.Close(); err != nil {
		t.Logf("Warning: error closing client 1: %v", err)
	}

	// Verify it's closed
	if client1.IsConnected() {
		t.Error("Client 1 should report as disconnected after Close()")
	}

	// Create second client
	t.Log("Creating second client...")
	client2 := NewClient(config)
	defer client2.Close()

	if err := client2.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}

	session2, err := client2.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	var response strings.Builder
	_, err = session2.SendPrompt(ctx, "Say 'second'", func(update *SessionUpdateParams) {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			if content := update.Update.GetMessageContent(); content != nil {
				response.WriteString(content.Text)
			}
		}
	})
	if err != nil {
		t.Fatalf("Second prompt failed: %v", err)
	}

	t.Logf("Second client response: %s", response.String())
	if !strings.Contains(strings.ToLower(response.String()), "second") {
		t.Errorf("Expected 'second' in response, got: %s", response.String())
	}
}

// TestRovoDevUpdateTypesE2E logs all update types received for debugging.
func TestRovoDevUpdateTypesE2E(t *testing.T) {
	if _, err := exec.LookPath("acli"); err != nil {
		t.Skip("acli not found in PATH, skipping E2E test")
	}

	config := ClientConfig{
		Command:     "acli",
		Args:        []string{"rovodev", "acp"},
		Cwd:         os.Getenv("PWD"),
		InitTimeout: 60 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	session, err := client.NewSession(ctx, os.Getenv("PWD"), nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	t.Log("Capturing all update types...")
	updateTypes := make(map[string]int)
	var allUpdates []string

	_, err = session.SendPrompt(ctx, "What files are in this directory? List just the first 3.", func(update *SessionUpdateParams) {
		updateType := update.Update.SessionUpdate
		updateTypes[updateType]++

		// Log detailed info for each update type
		switch updateType {
		case SessionUpdateAgentMessageChunk:
			if content := update.Update.GetMessageContent(); content != nil {
				allUpdates = append(allUpdates, fmt.Sprintf("chunk: %q", truncate(content.Text, 30)))
			}
		case SessionUpdateToolCall:
			allUpdates = append(allUpdates, fmt.Sprintf("tool_call: %s (id=%s, kind=%s)",
				update.Update.GetToolName(), update.Update.ToolCallID, update.Update.Kind))
		case SessionUpdateToolCallUpdate:
			allUpdates = append(allUpdates, fmt.Sprintf("tool_update: id=%s status=%s name=%s",
				update.Update.ToolCallID, update.Update.Status, update.Update.GetToolName()))
		default:
			allUpdates = append(allUpdates, fmt.Sprintf("other: %s", updateType))
		}
	})

	if err != nil {
		t.Fatalf("Failed to send prompt: %v", err)
	}

	t.Log("Update type summary:")
	for typ, count := range updateTypes {
		t.Logf("  %s: %d", typ, count)
	}

	t.Log("All updates received:")
	for i, u := range allUpdates {
		t.Logf("  %d: %s", i+1, u)
	}
}
