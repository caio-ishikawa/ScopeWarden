BIN_DIR := /usr/bin

.PHONY: all daemon cmd clean

all: daemon cli 

daemon:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/scopewarden-daemon ./daemon
	@echo "scopewarden-daemon built at $(BIN_DIR)/scopewarden-daemon"

cli:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/scopewarden ./cmd
	@echo "scopewarden CLI built at $(BIN_DIR)/scopewarden"

clean:
	rm -rf $(BIN_DIR)/scopewarden-daemon
	rm -rf $(BIN_DIR)/scopewarden
	@echo "Cleaned binaries"
