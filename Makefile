BIN_DIR := /usr/bin
DATA_DIR := /var/lib/scopewarden

.PHONY: all daemon cmd clean

all: daemon cli 

daemon:
	@mkdir -p $(BIN_DIR)
	@mkdir -p $(DATA_DIR)
	go build -o $(BIN_DIR)/scopewarden-daemon ./daemon
	@echo "scopewarden-daemon built at $(BIN_DIR)/scopewarden-daemon"

install-daemon: 
	# useradd --system --no-create-home --user-group scopewarden
	mkdir -p /etc/scopewarden
	touch /etc/scopewarden/scopewarden.env
	touch /etc/scopewarden/scopewarden.yaml
	echo "SCOPEWARDEN_CONFIG=/etc/scopewarden/scopewarden.yaml" > /etc/scopewarden/scopewarden.env

	chown root:scopewarden /etc/scopewarden/scopewarden.env
	chmod 640 /etc/scopewarden/scopewarden.env

	chown scopewarden:root $(DATA_DIR)
	chmod 775 $(DATA_DIR)

	cp etc/scopewarden-daemon.service /etc/systemd/system/ 
	systemctl daemon-reload
	systemctl enable scopewarden-daemon
	systemctl start scopewarden-daemon

uninstall-daemon:
	systemctl disable scopewarden-daemon || true
	rm -rf $(BIN_DIR)/scopewarden-daemon
	rm -rf $(DATA_DIR)
	rm -rf /etc/scopewarden
	systemctl daemon-reload
	@echo "scopewarden-daemon uninstalled"

cli:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/scopewarden ./cmd
	@echo "scopewarden CLI built at $(BIN_DIR)/scopewarden"

clean:
	rm -rf $(BIN_DIR)/scopewarden-daemon
	rm -rf $(BIN_DIR)/scopewarden
	@echo "Cleaned binaries"
