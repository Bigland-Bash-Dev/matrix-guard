# 🛡️ Matrix-Guard

A lightweight, high-performance system health engine and self-healing daemon written in Go. Specifically designed to protect Matrix Synapse homeservers from OOM (Out of Memory) crashes and I/O thrashing.

## 🚀 Overview

Matrix-Guard monitors your Synapse instance by depth-charging into Linux cgroups to track individual worker memory usage. It features a real-time web dashboard and an automated "Guardian" loop that identifies "Memory Hogs" and restarts them before they hit hard system limits.

### Key Features
* **Real-time Dashboard:** Built-in Go web server providing live stats on RAM, Swap, CPU Load, and Disk I/O.
* **Cgroup Awareness:** Tracks memory usage directly from `/sys/fs/cgroup/` for precise worker monitoring.
* **Self-Healing:** Automatically restarts greedy Matrix workers when they exceed a configurable threshold.
* **Pressure Stall Information (PSI):** Monitors Disk I/O pressure to detect when swap is causing system lag.
* **Portable & Fast:** Compiles to a single static binary with zero external dependencies.

## 📊 Dashboard

Once running, the dashboard is available at:
`http://<your-server-ip>:8080`

The dashboard auto-refreshes every 30 seconds, providing a "Tokyo Night" themed overview of your server's vitals.

## 🛠️ Installation

### 1. Requirements
* Go 1.20+ (to build)
* A Matrix Synapse instance running via Systemd (Debian, Arch, etc.)
* Root privileges (to monitor cgroups and restart services)

### 2. Automatic Install
Clone the repo and run the installer:
```bash
git clone https://github.com/Bigland-Bash-Dev/matrix-guard.git
cd matrix-guard
chmod +x install.sh
sudo ./install.sh

3. Manual Build

    go build -o matrix-guard
    sudo ./matrix-guard -log=/var/log/matrix-guard.log
    
 ⚙️ Configuration

Matrix-Guard supports the following flags:

    -log: Set a custom path for the log file (Default: /var/log/matrix-guard.log)
    
🛡️ Self-Healing Logic

The daemon currently checks for workers nearing a 1.2 GB limit. If a worker is detected using over 1150 MB, Matrix-Guard will:

Log the event to /var/log/matrix-guard.log.

Issue a systemctl restart for that specific worker.

Flush the "Hog" stats to free up physical RAM for the rest of the stack.

📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

Developed by Bigland-Bash-Dev
