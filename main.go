package main

import (
	"flag"
	"fmt"
	"log"
	"net" // Added for IP detection
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var currentStatus string

func getDetailedMemory() (float64, string) {
	var total float64
	var maxMem float64
	var hogName string
	rootPath := "/sys/fs/cgroup/system.slice"

	filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() == "memory.current" {
			if strings.Contains(path, "matrix") {
				data, _ := os.ReadFile(path)
				var bytes int64
				fmt.Sscanf(string(data), "%d", &bytes)
				mbs := float64(bytes) / 1024 / 1024
				total += mbs

				if mbs > maxMem {
					maxMem = mbs
					parts := strings.Split(path, "/")
					if len(parts) >= 2 {
						hogName = parts[len(parts)-2]
					}
				}
			}
		}
		return nil
	})
	return total, fmt.Sprintf("%s (%.2fMB)", hogName, maxMem)
}

func getSwapUsage() string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "N/A"
	}
	lines := strings.Split(string(data), "\n")
	var total, free int64
	for _, line := range lines {
		if strings.HasPrefix(line, "SwapTotal:") {
			fmt.Sscanf(line, "SwapTotal: %d", &total)
		}
		if strings.HasPrefix(line, "SwapFree:") {
			fmt.Sscanf(line, "SwapFree: %d", &free)
		}
	}
	used := (total - free) / 1024
	return fmt.Sprintf("%dMB", used)
}

func getLoadAvg() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "err"
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		return strings.Join(fields[:3], " ")
	}
	return "N/A"
}

func getIOPressure() string {
	data, err := os.ReadFile("/proc/pressure/io")
	if err != nil {
		return "no-psi"
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return "clear"
}

func main() {
	logPath := flag.String("log", "/var/log/matrix-guard.log", "Path to the log file")
	flag.Parse()

	if _, err := os.Stat(*logPath); os.IsNotExist(err) {
		file, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Printf("Critical: Could not create log file at %s. Error: %v\n", *logPath, err)
			return
		}
		file.Close()
	}

	f, err := os.OpenFile(*logPath, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer f.Close()
	log.SetOutput(f)

	// Dynamic IP detection for the dashboard link
	displayIP := "localhost"
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				displayIP = ipnet.IP.String()
				break
			}
		}
	}

	fmt.Println("🛡️ Matrix Guard is ACTIVE.")
	fmt.Printf("📊 Dashboard: http://%s:8080\n", displayIP)
	fmt.Printf("📜 Logging to: %s\n", *logPath)

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<html><head><title>Matrix Guard</title><meta http-equiv='refresh' content='30'></head>")
			fmt.Fprintf(w, "<body style='background:#1a1b26;color:#c0caf5;font-family:monospace;padding:20px;'>")
			fmt.Fprintf(w, "<h1>Matrix Guard Status</h1><hr>")
			fmt.Fprintf(w, "<pre style='font-size:1.2em;'>%s</pre>", currentStatus)
			fmt.Fprintf(w, "</body></html>")
		})
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		mem, hog := getDetailedMemory()
		swap := getSwapUsage()
		load := getLoadAvg()
		io := getIOPressure()

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		currentStatus = fmt.Sprintf("Time: %s\nTotal RAM: %.2f MB\nTop Hog:  %s\nSwap:     %s\nCPU Load: %s\nDisk I/O: %s", 
			timestamp, mem, hog, swap, load, io)

		log.Println(strings.ReplaceAll(currentStatus, "\n", " | "))

		if strings.Contains(hog, "worker") {
			var val float64
			fmt.Sscanf(strings.Split(hog, "(")[1], "%f", &val)
			
			if val > 1150 {
				serviceName := strings.Split(hog, " ")[0]
				fmt.Printf("🚑 ALERT: Restarting %s (Used: %.2fMB)\n", serviceName, val)
				exec.Command("systemctl", "restart", serviceName).Run()
				log.Printf("SELF-HEAL: Restarted %s at %.2fMB", serviceName, val)
			}
		}
	}
}
