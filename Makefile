.PHONY: dev build clean install

dev:
	@bash scripts/start.sh

build:
	@echo "Building frontend..."
	cd web && npm run build
	@echo "Building Go backend..."
	go build -o photo-server ./cmd/server/main.go

install:
	@echo "Installing frontend dependencies..."
	cd web && npm install
	@echo "Installing Python dependencies..."
	cd ml-service && source venv/bin/activate && pip install -r requirements.txt

clean:
	rm -f photo-server server main
	cd web && rm -rf dist node_modules .turbo
	find ml-service -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".turbo" -exec rm -rf {} + 2>/dev/null || true
