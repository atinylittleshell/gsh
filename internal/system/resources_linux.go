//go:build linux

package system

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

var lastCPUSampleTime time.Time
var lastTotalTicks uint64
var lastIdleTicks uint64

func getResources() *Resources {
	res := &Resources{
		Timestamp: time.Now(),
	}

	// RAM
	if memInfo, err := os.Open("/proc/meminfo"); err == nil {
		defer func() {
			_ = memInfo.Close()
		}()
		scanner := bufio.NewScanner(memInfo)
		var memTotal, memAvailable uint64
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			key := strings.TrimSuffix(parts[0], ":")
			val, _ := strconv.ParseUint(parts[1], 10, 64)
			// /proc/meminfo values are in kB
			val *= 1024

			switch key {
			case "MemTotal":
				memTotal = val
			case "MemAvailable":
				memAvailable = val
			}
		}
		if memTotal > 0 {
			res.RAMTotal = memTotal
			res.RAMUsed = memTotal - memAvailable
		}
	}

	// CPU
	if stat, err := os.Open("/proc/stat"); err == nil {
		defer func() {
			_ = stat.Close()
		}()
		scanner := bufio.NewScanner(stat)
		if scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) > 4 && parts[0] == "cpu" {
				user, _ := strconv.ParseUint(parts[1], 10, 64)
				nice, _ := strconv.ParseUint(parts[2], 10, 64)
				system, _ := strconv.ParseUint(parts[3], 10, 64)
				idle, _ := strconv.ParseUint(parts[4], 10, 64)
				iowait, _ := strconv.ParseUint(parts[5], 10, 64)
				irq, _ := strconv.ParseUint(parts[6], 10, 64)
				softirq, _ := strconv.ParseUint(parts[7], 10, 64)
				steal, _ := strconv.ParseUint(parts[8], 10, 64)

				total := user + nice + system + idle + iowait + irq + softirq + steal

				if !lastCPUSampleTime.IsZero() {
					deltaTotal := total - lastTotalTicks
					deltaIdle := idle - lastIdleTicks

					if deltaTotal > 0 {
						res.CPUPercent = 100.0 * float64(deltaTotal-deltaIdle) / float64(deltaTotal)
					}
				}

				lastTotalTicks = total
				lastIdleTicks = idle
				lastCPUSampleTime = time.Now()
			}
		}
	}

	return res
}
