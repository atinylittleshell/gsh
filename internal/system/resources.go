package system

import "time"

type Resources struct {
	CPUPercent float64 // 0-100
	RAMUsed    uint64  // bytes
	RAMTotal   uint64  // bytes
	VRAMUsed   uint64  // bytes (0 if unknown)
	VRAMTotal  uint64  // bytes (0 if unknown)
	Timestamp  time.Time
}

// GetResources returns the current system resource usage.
// It may return partial data if some metrics are unavailable.
func GetResources() *Resources {
	return getResources()
}
