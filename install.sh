#!/bin/bash
#
# Build the project
echo "Building Matrix Guard monitor..."
go build -o matrix-guard

# Move the binary to /usr/local/bin
echo "Moving binary to /usr/local/bin/..."
mv matrix-guard /usr/local/bin/

# Create the systemd service file
echo "Configuring Systemd service..."
cat <<EOF > /etc/systemd/system/matrix-guard.service
[Unit]
Description=Matrix Guard server writting in Go
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/matrix-guard -log=/var/log/matrix-guard.log
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

# Reload and start
echo "Starting Matrix Guard server...."
systemctl daemon-reload
systemctl enable --now matrix-guard

echo " Done!! Dashboard available at: http://$(hostname -I | awk '{print $1}'):8080"
echo " Log availale! tail -f /var/log/matrix-guard.log"
