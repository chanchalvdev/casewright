BIN     := siemagent
BINARY  := ./bin/$(BIN)
WEB_DIR := web

.PHONY: build run serve test vet clean setup dev docker-up docker-down seed build-all

# ── Build ──────────────────────────────────────────────────────────────────────

build:
	go build -o $(BINARY) ./cmd/siemagent

build-all: build-frontend embed-binary

build-frontend:
	cd $(WEB_DIR) && npm run build

embed-binary:
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/siemagent
	@echo "Binary built with embedded frontend: $(BINARY)"

# ── Run ───────────────────────────────────────────────────────────────────────

run: build
	$(BINARY) sample.log

serve: build
	$(BINARY) --serve --port $${CONDUCTOR_PORT:-8080}

# ── Dev (live reload) ─────────────────────────────────────────────────────────

dev:
	@echo "Starting Go server (air) + Vite dev server in parallel..."
	@trap 'kill 0' SIGINT SIGTERM; \
	  (cd $(WEB_DIR) && npm run dev) & \
	  (air -c .air.toml 2>/dev/null || go run ./cmd/siemagent --serve) & \
	  wait

# ── Test ──────────────────────────────────────────────────────────────────────

test:
	go test ./... -v -race

test-integration:
	go test -tags integration ./... -v

vet:
	go vet ./...

# ── Setup ─────────────────────────────────────────────────────────────────────

setup:
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; fi
	go mod download
	@if [ -d $(WEB_DIR) ]; then cd $(WEB_DIR) && npm install; fi
	@mkdir -p bin

# ── Docker ────────────────────────────────────────────────────────────────────

docker-up:
	docker compose up -d
	@echo "Services starting... run 'docker compose ps' to check status."

docker-down:
	docker compose down

pull-models:
	bash scripts/pull-models.sh

# ── Seed ──────────────────────────────────────────────────────────────────────

PORT ?= $${CONDUCTOR_PORT:-8080}

seed:
	@echo "Seeding POST /classify with 10 sample syslog lines..."
	@for line in \
	  '<34>1 2024-01-15T10:30:00Z webserver01 sshd 1234 - Failed password for root from 192.168.1.100 port 22' \
	  '<34>1 2024-01-15T10:30:01Z webserver01 sshd 1234 - Failed password for root from 192.168.1.100 port 22' \
	  '<34>1 2024-01-15T10:30:02Z webserver01 sshd 1234 - Failed password for admin from 10.0.0.5 port 22' \
	  '<34>1 2024-01-15T10:31:00Z webserver01 nginx 5678 - GET /etc/passwd HTTP/1.1 200 0 "-" "-"' \
	  '<34>1 2024-01-15T10:31:05Z db01 postgres 9012 - FATAL: role "root" does not exist' \
	  '<34>1 2024-01-15T10:32:00Z appserver01 sudo 3456 - user: tty=pts/0 ; USER=root ; COMMAND=/bin/bash' \
	  '<34>1 2024-01-15T10:33:00Z firewall01 kernel 0 - DROP TCP 203.0.113.1:4444->10.0.1.5:80' \
	  '<34>1 2024-01-15T10:34:00Z workstation01 powershell 7890 - Invoke-Expression (New-Object Net.WebClient).DownloadString' \
	  '<34>1 2024-01-15T10:35:00Z webserver01 apache2 2345 - POST /login.php HTTP/1.1 200 - "sqlmap/1.6"' \
	  '<34>1 2024-01-15T10:36:00Z cron01 CRON 1111 - CMD (/usr/local/bin/backup.sh)' \
	  ; do \
	  curl -sf -X POST http://localhost:$(PORT)/classify \
	    -H "Content-Type: application/json" \
	    -d "{\"log\":\"$$line\",\"format\":\"syslog\"}" | python3 -m json.tool --no-ensure-ascii 2>/dev/null || true; \
	  echo "---"; \
	done

# ── Clean ─────────────────────────────────────────────────────────────────────

clean:
	rm -rf bin/
	@if [ -d $(WEB_DIR)/dist ]; then rm -rf $(WEB_DIR)/dist; fi
