#!/bin/bash
sudo tee /etc/systemd/system/setup-ubuntu-user-before-ssh.service > /dev/null <<'EOF'
[Unit]
Description=Create ubuntu user, set password, expire it — before SSH (first boot only)
Before=ssh.service sshd.service
ConditionPathExists=!/var/lib/setup-ubuntu-user.done

[Service]
Type=oneshot
ExecStart=/bin/bash -c '\
  if ! id ubuntu >/dev/null 2>&1; then \
    echo "Creating user ubuntu..."; \
    useradd -m -s /bin/bash ubuntu; \
  fi; \
  if grep -iq '^password:' /var/lib/cloud/instance/user-data.txt* 2>/dev/null; then \
      echo "Yes – password was set via cloud-init!"; \
  else \
    echo "Setting default password for ubuntu..."; \
    echo "ubuntu:password" | chpasswd; \
    passwd --expire ubuntu && echo "Password expired for ubuntu — must change on first login"; \
  fi; \
'
ExecStartPost=/bin/touch /var/lib/setup-ubuntu-user.done
RemainAfterExit=yes
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
sudo systemctl daemon-reexec
sudo systemctl enable setup-ubuntu-user-before-ssh.service