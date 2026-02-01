.PHONY: infra-up infra-down infra-reset infra-logs \
       run-auction run-bid run-bff run-bot run-all \
       run-frontend test

COMPOSE_FILE := infra/compose/docker-compose.yml
ENV_FILE := .env

# ──────────────────────────────────────────────
# Infrastructure
# ──────────────────────────────────────────────

infra-up:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d

infra-down:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down

infra-reset:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down -v

infra-logs:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) logs -f

# ──────────────────────────────────────────────
# Backend services
# ──────────────────────────────────────────────

run-auction:
	go run ./services/auction-service/cmd/auction-service/...

run-bid:
	go run ./services/bid-service/cmd/bid-service/...

run-bff:
	go run ./services/bff/cmd/bff/...

run-bot:
	go run ./services/bot-service/cmd/bot-service/...

run-all:
	@echo "Starting all services..."
	$(MAKE) run-auction & \
	$(MAKE) run-bid & \
	$(MAKE) run-bff & \
	$(MAKE) run-bot & \
	wait

# ──────────────────────────────────────────────
# Frontend
# ──────────────────────────────────────────────

run-frontend:
	cd frontend && npm run dev

# ──────────────────────────────────────────────
# Testing
# ──────────────────────────────────────────────

test:
	go test ./services/auction-service/... ./services/bid-service/... ./services/bff/... ./services/bot-service/... ./shared/...
