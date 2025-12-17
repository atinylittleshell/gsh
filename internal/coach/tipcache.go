package coach

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"sync"
	"time"
)

// TipCache manages a cache of generated tips
type TipCache struct {
	tips       []*GeneratedTip
	maxSize    int
	ttl        time.Duration
	mu         sync.RWMutex
	shownToday map[string]bool
	lastReset  time.Time
}

// NewTipCache creates a new tip cache
func NewTipCache(maxSize int, ttl time.Duration) *TipCache {
	return &TipCache{
		tips:       make([]*GeneratedTip, 0, maxSize),
		maxSize:    maxSize,
		ttl:        ttl,
		shownToday: make(map[string]bool),
		lastReset:  time.Now(),
	}
}

// Add adds a tip to the cache
func (c *TipCache) Add(tip *GeneratedTip) {
	if tip == nil || tip.ID == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove expired tips first
	c.pruneExpiredLocked()

	// Check for duplicates
	for _, existing := range c.tips {
		if existing.ID == tip.ID || c.isSimilarTip(existing, tip) {
			return
		}
	}

	// Remove oldest if at capacity
	if len(c.tips) >= c.maxSize {
		c.tips = c.tips[1:]
	}

	c.tips = append(c.tips, tip)
}

// GetRandom returns a random tip that hasn't been shown today
func (c *TipCache) GetRandom() *GeneratedTip {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checkDayReset()
	c.pruneExpiredLocked()

	// Filter out tips shown today
	var available []*GeneratedTip
	for _, tip := range c.tips {
		if !c.shownToday[tip.ID] {
			available = append(available, tip)
		}
	}

	// If all tips shown, reset
	if len(available) == 0 {
		c.shownToday = make(map[string]bool)
		available = c.tips
	}

	if len(available) == 0 {
		return nil
	}

	// Weighted random by priority
	return c.weightedRandomTip(available)
}

// GetByType returns a tip of a specific type
func (c *TipCache) GetByType(tipType TipType) *GeneratedTip {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checkDayReset()

	for _, tip := range c.tips {
		if tip.Type == tipType && !c.shownToday[tip.ID] {
			return tip
		}
	}
	return nil
}

// GetHighPriority returns the highest priority tip not shown today
func (c *TipCache) GetHighPriority() *GeneratedTip {
	// Acquire full lock for checkDayReset() which mutates shownToday and lastReset
	c.mu.Lock()
	c.checkDayReset()

	// Capture the current state under the write lock to avoid race conditions
	tipsCopy := make([]*GeneratedTip, len(c.tips))
	copy(tipsCopy, c.tips)
	shownTodayCopy := make(map[string]bool)
	for id, shown := range c.shownToday {
		shownTodayCopy[id] = shown
	}

	// Release the write lock since we have the state we need
	c.mu.Unlock()

	// Find the best tip from the captured state (no lock needed)
	var best *GeneratedTip
	for _, tip := range tipsCopy {
		if !shownTodayCopy[tip.ID] {
			if best == nil || tip.Priority > best.Priority {
				best = tip
			}
		}
	}
	return best
}

// MarkShown marks a tip as shown today
func (c *TipCache) MarkShown(tipID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shownToday[tipID] = true
}

// GetRecentIDs returns IDs of recent tips
func (c *TipCache) GetRecentIDs(limit int) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, limit)
	for i := len(c.tips) - 1; i >= 0 && len(ids) < limit; i-- {
		ids = append(ids, c.tips[i].ID)
	}
	return ids
}

// Size returns the number of cached tips
func (c *TipCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.tips)
}

// Clear removes all tips from cache
func (c *TipCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tips = make([]*GeneratedTip, 0, c.maxSize)
	c.shownToday = make(map[string]bool)
}

// GetAll returns all cached tips
func (c *TipCache) GetAll() []*GeneratedTip {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*GeneratedTip, len(c.tips))
	copy(result, c.tips)
	return result
}

// pruneExpiredLocked removes expired tips (must be called with lock held)
func (c *TipCache) pruneExpiredLocked() {
	now := time.Now()
	valid := make([]*GeneratedTip, 0, len(c.tips))
	for _, tip := range c.tips {
		if tip.ExpiresAt.After(now) {
			valid = append(valid, tip)
		}
	}
	c.tips = valid
}

// checkDayReset resets shown tracking at midnight
func (c *TipCache) checkDayReset() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastResetDay := time.Date(c.lastReset.Year(), c.lastReset.Month(), c.lastReset.Day(), 0, 0, 0, 0, c.lastReset.Location())

	if today.After(lastResetDay) {
		c.shownToday = make(map[string]bool)
		c.lastReset = now
	}
}

// isSimilarTip checks if two tips are too similar
func (c *TipCache) isSimilarTip(a, b *GeneratedTip) bool {
	// Same command suggestion
	if a.Command != "" && a.Command == b.Command {
		return true
	}

	// Same type and very similar title
	if a.Type == b.Type && a.Title == b.Title {
		return true
	}

	return false
}

// weightedRandomTip selects a tip weighted by priority
func (c *TipCache) weightedRandomTip(tips []*GeneratedTip) *GeneratedTip {
	if len(tips) == 0 {
		return nil
	}

	totalWeight := 0
	for _, tip := range tips {
		weight := tip.Priority
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	r := rand.Intn(totalWeight)
	cumulative := 0
	for _, tip := range tips {
		weight := tip.Priority
		if weight <= 0 {
			weight = 1
		}
		cumulative += weight
		if r < cumulative {
			return tip
		}
	}

	return tips[len(tips)-1]
}

// GenerateTipID generates a unique ID for a tip based on content
func GenerateTipID() string {
	timestamp := time.Now().UnixNano()
	random := rand.Int63()
	data := []byte{
		byte(timestamp >> 56), byte(timestamp >> 48), byte(timestamp >> 40), byte(timestamp >> 32),
		byte(timestamp >> 24), byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp),
		byte(random >> 56), byte(random >> 48), byte(random >> 40), byte(random >> 32),
		byte(random >> 24), byte(random >> 16), byte(random >> 8), byte(random),
	}
	hash := sha256.Sum256(data)
	return "tip_" + hex.EncodeToString(hash[:8])
}

// HashTipContent creates a hash from tip content for deduplication
func HashTipContent(title, command string) string {
	data := title + ":" + command
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}
