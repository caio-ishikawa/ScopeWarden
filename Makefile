BIN_DIR := /usr/bin
DATA_DIR := /var/lib/scopewarden
USER_PATH := $(shell echo $$PATH)
USER := $(shell whoami)
GROUP := $(shell id -gn)

.PHONY: all daemon cmd clean

all: daemon cli 

daemon:
	sudo mkdir -p $(BIN_DIR)
	sudo mkdir -p $(DATA_DIR)
	sudo go build -o $(BIN_DIR)/scopewarden-daemon ./daemon
	echo "scopewarden-daemon built at $(BIN_DIR)/scopewarden-daemon"

install-daemon: 
	sudo mkdir -p /etc/scopewarden
	sudo chown $(USER):$(GROUP) /etc/scopewarden
	sudo chmod 775 /etc/scopewarden

	touch /etc/scopewarden/scopewarden.env
	echo "PATH=$(USER_PATH)" > /etc/scopewarden/scopewarden.env
	echo "SCOPEWARDEN_CONFIG=/etc/scopewarden/scopewarden.yaml" >> /etc/scopewarden/scopewarden.env

	sudo chown $(USER):$(GROUP) /etc/scopewarden/scopewarden.env
	sudo chmod 640 /etc/scopewarden/scopewarden.env

	sudo chown $(USER):$(GROUP) $(DATA_DIR)
	sudo chmod 775 $(DATA_DIR)

	sudo touch /etc/scopewarden/scopewarden.yaml
	sudo chown $(USER):$(GROUP) /etc/scopewarden/scopewarden.yaml
	sudo chmod 775 /etc/scopewarden/scopewarden.yaml

	sudo sh -c 'echo "[Unit]" > /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "Description=ScopeWarden Daemon & API" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "After=network.target" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "[Service]" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "Type=simple" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "WorkingDirectory=/var/lib/scopewarden" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "ReadWritePaths=/var/lib/scopewarden" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "EnvironmentFile=/etc/scopewarden/scopewarden.env" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "ExecStart=/usr/bin/scopewarden-daemon" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "Restart=always" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "RestartSec=10" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "StandardOutput=journal" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "StandardError=journal" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "User=$(USER)" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "Group=$(GROUP)" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "[Install]" >> /etc/systemd/system/scopewarden-daemon.service'
	sudo sh -c 'echo "WantedBy=multi-user.target" >> /etc/systemd/system/scopewarden-daemon.service'

	sudo systemctl daemon-reload
	sudo systemctl enable scopewarden-daemon
	sudo systemctl start scopewarden-daemon

uninstall-daemon:
	sudo systemctl disable scopewarden-daemon || true
	sudo systemctl stop scopewarden-daemon || true
	sudo rm -rf $(BIN_DIR)/scopewarden-daemon
	sudo rm -rf $(DATA_DIR)
	sudo rm -rf /etc/scopewarden
	sudo systemctl daemon-reload
	sudo rm /etc/systemd/system/scopewarden-daemon.service
	@echo "scopewarden-daemon uninstalled"

cli:
	sudo @mkdir -p $(BIN_DIR)
	sudo go build -o $(BIN_DIR)/scopewarden ./cmd
	@echo "scopewarden CLI built at $(BIN_DIR)/scopewarden"

uninstall-cli:
	sudo rm -rf $(BIN_DIR)/scopewarden
	@echo "scopewarden CLI deleted from $(BIN_DIR)/scopewarden"

