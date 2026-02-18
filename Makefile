.PHONY: all build dev backend frontend clean start stop restart status

all: build

build: backend frontend

backend:
	cd backend && go build -o ../bin/rp .

frontend:
	cd frontend && npm install && npm run build

dev:
	make dev-backend & make dev-frontend

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm install && npm run dev

# Process management
start:
	@echo "Starting services..."
	@make start-backend
	@make start-frontend
	@echo "Services started. Run 'make status' to check."

start-backend:
	@if lsof -ti:8080 >/dev/null 2>&1; then \
		echo "Backend already running on port 8080"; \
	else \
		echo "Starting backend..."; \
		cd backend && nohup go run . > backend.log 2>&1 & \
		sleep 2 && echo "Backend started"; \
	fi

start-frontend:
	@if lsof -ti:5173 >/dev/null 2>&1; then \
		echo "Frontend already running on port 5173"; \
	else \
		echo "Starting frontend..."; \
		cd frontend && nohup npm run dev > frontend.log 2>&1 & \
		sleep 3 && echo "Frontend started"; \
	fi

stop:
	@echo "Stopping services..."
	@make stop-backend
	@make stop-frontend
	@echo "Services stopped."

stop-backend:
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || echo "Backend not running"

stop-frontend:
	@lsof -ti:5173 | xargs kill -9 2>/dev/null || echo "Frontend not running"

restart:
	@make stop
	@sleep 1
	@make start

status:
	@echo "Service Status:"
	@echo "-------------"
	@if lsof -ti:8080 >/dev/null 2>&1; then \
		echo "Backend:  RUNNING (port 8080)"; \
	else \
		echo "Backend:  STOPPED"; \
	fi
	@if lsof -ti:5173 >/dev/null 2>&1; then \
		echo "Frontend: RUNNING (port 5173)"; \
	else \
		echo "Frontend: STOPPED"; \
	fi

clean:
	rm -rf bin/
	rm -rf frontend/dist
	rm -rf frontend/node_modules

.DEFAULT_GOAL := dev
