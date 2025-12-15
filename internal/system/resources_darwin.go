//go:build darwin

package system

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func getResources() *Resources {
	res := &Resources{
		Timestamp: time.Now(),
	}

	// RAM using vm_stat
	// Pages free:                               3664.
	// Pages active:                           126956.
	// Pages inactive:                         122676.
	// Pages speculative:                        6462.
	// Pages throttled:                             0.
	// Pages wired down:                        67761.
	// Pages purgeable:                          2688.
	// "Translation faults":                 59648939.
	// Pages copy-on-write:                    541434.
	// Pages zero filled:                    41132649.
	// Pages reactivated:                        2346.
	// Pages purged:                           622268.
	// File-backed pages:                       43560.
	// Anonymous pages:                        212534.
	// Pages stored in compressor:             164280.
	// Pages occupied by compressor:            55944.
	// Decompressions:                         372370.
	// Compressions:                           522501.
	// Pageins:                               1941655.
	// Pageouts:                                 5093.
	// Swapins:                                     0.
	// Swapouts:                                    0.

	// Total RAM via sysctl hw.memsize
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64); err == nil {
			res.RAMTotal = val
		}
	}

	// Used RAM approximation: (active + wired + compressed) * 4096 (or 16384 on Apple Silicon?)
	// Actually vm_stat pages are usually 4096 bytes.
	// A simpler heuristic: Total - Free.
	// But on macOS "Free" is often low due to caching.
	// "App Memory" + "Wired Memory" + "Compressed" is what Activity Monitor shows as "Memory Used".

	// Getting vm_stat
	out, err = exec.Command("vm_stat").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		var pagesActive, pagesWired, pagesCompressed uint64
		pageSize := uint64(4096) // Default
		// On Apple Silicon page size is 16k? No, vm_stat usually reports in 4096 byte pages, the header says:
		// "Mach Virtual Memory Statistics: (page size of 4096 bytes)"
		// We should parse the first line if possible.

		if len(lines) > 0 && strings.Contains(lines[0], "page size of") {
			parts := strings.Fields(lines[0])
			for i, p := range parts {
				if p == "bytes)" && i > 0 {
					if val, err := strconv.ParseUint(parts[i-1], 10, 64); err == nil {
						pageSize = val
					}
				}
			}
		}

		for _, line := range lines {
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			valStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))
			val, _ := strconv.ParseUint(valStr, 10, 64)

			switch key {
			case "Pages active":
				pagesActive = val
			case "Pages wired down":
				pagesWired = val
			case "Pages occupied by compressor":
				pagesCompressed = val
			}
		}

		// Memory Used = App Memory + Wired + Compressed
		// App Memory = Anonymous + Purgeable?
		// Alternative: Used = Total - (Free + Inactive) (Inactive is file cache usually)

		if res.RAMTotal > 0 {
			// This is a rough approximation
			res.RAMUsed = (pagesActive + pagesWired + pagesCompressed) * pageSize
		}
	}

	// CPU Load
	// sysctl -n vm.loadavg
	// returns "{ 2.06 1.89 1.84 }"
	// This is load average, not percentage.
	// To get percentage we can use `top -l 1 -n 0 | grep "CPU usage"`
	// Output: "CPU usage: 5.88% user, 10.58% sys, 83.52% idle"

	out, err = exec.Command("top", "-l", "1", "-n", "0").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "CPU usage:") {
				// CPU usage: 3.52% user, 5.88% sys, 90.58% idle
				parts := strings.Fields(line)
				if len(parts) >= 7 {
					// We can extract idle and subtract from 100
					// idle is usually the last one? "90.58% idle"
					for i, p := range parts {
						if p == "idle" && i > 0 {
							idleStr := strings.TrimSuffix(parts[i-1], "%")
							idleVal, err := strconv.ParseFloat(idleStr, 64)
							if err == nil {
								res.CPUPercent = 100.0 - idleVal
							}
						}
					}
				}
				break
			}
		}
	}

	return res
}
