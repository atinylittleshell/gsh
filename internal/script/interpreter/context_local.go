package interpreter

import (
	"context"
	"sync"
)

// goroutineContexts stores execution contexts per goroutine so concurrent REPL
// work like prediction and foreground command execution do not overwrite each
// other's cancellation state.
type goroutineContexts struct {
	mu       sync.RWMutex
	contexts map[int64]context.Context
}

func newGoroutineContexts() *goroutineContexts {
	return &goroutineContexts{
		contexts: make(map[int64]context.Context),
	}
}

func (g *goroutineContexts) set(ctx context.Context) {
	gid := getGoroutineID()

	g.mu.Lock()
	defer g.mu.Unlock()

	g.contexts[gid] = ctx
}

func (g *goroutineContexts) clear() {
	gid := getGoroutineID()

	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.contexts, gid)
}

func (g *goroutineContexts) get() context.Context {
	gid := getGoroutineID()

	g.mu.RLock()
	defer g.mu.RUnlock()

	ctx := g.contexts[gid]
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
