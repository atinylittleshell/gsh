//go:build windows

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

	// Use PowerShell to get memory info
	// Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory
	cmd := exec.Command("powershell", "-Command", "Get-CimInstance Win32_OperatingSystem | Select-Object -Property TotalVisibleMemorySize, FreePhysicalMemory")
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		// Format is typically:
		// TotalVisibleMemorySize FreePhysicalMemory
		// ---------------------- ------------------
		//               33554432           16777216

		if len(lines) >= 3 {
			fields := strings.Fields(lines[2]) // The values line
			if len(fields) >= 2 {
				totalKB, _ := strconv.ParseUint(fields[0], 10, 64)
				freeKB, _ := strconv.ParseUint(fields[1], 10, 64)

				res.RAMTotal = totalKB * 1024
				res.RAMUsed = (totalKB - freeKB) * 1024
			}
		}
	}

	// Use PowerShell to get CPU LoadPercentage
	// Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average | Select-Object Average
	cmd = exec.Command("powershell", "-Command", "Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average | Select-Object -ExpandProperty Average")
	out, err = cmd.Output()
	if err == nil {
		valStr := strings.TrimSpace(string(out))
		if val, err := strconv.ParseFloat(valStr, 64); err == nil {
			res.CPUPercent = val
		}
	}

	return res
}
