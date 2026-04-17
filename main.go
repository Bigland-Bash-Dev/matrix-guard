package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

//go:embed templates/*
var resources embed.FS

var (
	startTime    = time.Now()
	restartCount = 0 // The new counter
	currentMem   float64
	currentHog   string
	currentSwap  string
	currentCPU   float64
)

type PageData struct {
	Time         string
	GuardUptime  string
	Restarts     int // New field for the UI
	Mem          float64
	Hog          string
	Swap         string
	CPUPercent   float64
	Percent      float64
	BarColor     string
}

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
	if hogName == "" {
		return 0, "No Matrix Workers Detected"
	}
	return total, fmt.Sprintf("%s (%.1fMB)", hogName, maxMem)
}

func getSwapUsage() string {
	data, _ := os.ReadFile("/proc/meminfo")
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
	return fmt.Sprintf("%dMB", (total-free)/1024)
}

func main() {
	logPath := flag.String("log", "/var/log/matrix-guard.log", "Path to the log file")
	flag.Parse()

	f, err := os.OpenFile(*logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer f.Close()
	log.SetOutput(f)

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

	go func() {
		tmpl, _ := template.ParseFS(resources, "templates/index.html")
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			percent := (currentMem / 8192) * 100
			barColor := "#9ece6a"
			if percent > 80 {
				barColor = "#f7768e"
			}

			uptime := time.Since(startTime).Round(time.Second).String()

			data := PageData{
				Time:         time.Now().Format("15:04:05"),
				GuardUptime:  uptime,
				Restarts:     restartCount, // Passing the counter to HTML
				Mem:          currentMem,
				Hog:          currentHog,
				Swap:         currentSwap,
				CPUPercent:   currentCPU,
				Percent:      percent,
				BarColor:     barColor,
			}
			tmpl.Execute(w, data)
		})
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		currentMem, currentHog = getDetailedMemory()
		currentSwap = getSwapUsage()
		cpuUsage, _ := cpu.Percent(0, false)
		if len(cpuUsage) > 0 {
			currentCPU = cpuUsage[0]
		}

		// Self-Healing Logic
		if strings.Contains(currentHog, "worker") {
			var val float64
			parts := strings.Split(currentHog, "(")
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%f", &val)
				if val > 1150 {
					serviceName := strings.TrimSpace(parts[0])
					log.Printf("🚑 ALERT: Restarting %s (Used: %.2fMB)", serviceName, val)

					// Perform the restart
					err := exec.Command("systemctl", "restart", serviceName).Run()
					if err == nil {
						restartCount++ // Increment only on success
					} else {
						log.Printf("❌ ERROR: Failed to restart %s: %v", serviceName, err)
					}
				}
			}
		}
	}
}
