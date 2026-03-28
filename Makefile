.PHONY: build build-web build-go build-pi clean test dev

# Default: build everything
build: build-web build-go

# Build the React web UI
build-web:
	cd web && npm install && npm run build
	rm -rf cmd/mpr-api/dist
	cp -r web/dist cmd/mpr-api/dist

# Build the Go binary (requires web UI to be built first)
build-go:
	go build -o mpr-api ./cmd/mpr-api/

# Cross-compile for Raspberry Pi 3B (armv7)
build-pi: build-web
	rm -rf cmd/mpr-api/dist
	cp -r web/dist cmd/mpr-api/dist
	GOOS=linux GOARCH=arm GOARM=7 go build -o mpr-api-linux-armv7 ./cmd/mpr-api/

# Deploy to Pi (set PI_HOST env var, e.g., PI_HOST=mpr)
deploy: build-pi
	scp mpr-api-linux-armv7 $(PI_HOST):/tmp/mpr-api
	scp mpr-api.service $(PI_HOST):/tmp/mpr-api.service
	ssh $(PI_HOST) 'sudo systemctl stop mpr-api 2>/dev/null; sudo mv /tmp/mpr-api /usr/local/bin/mpr-api && sudo chmod +x /usr/local/bin/mpr-api && sudo mv /tmp/mpr-api.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable mpr-api && sudo systemctl restart mpr-api'

# Run tests
test:
	go test ./... -v

# Dev mode: run the Go API with a placeholder web UI
dev:
	mkdir -p cmd/mpr-api/dist
	echo '<!doctype html><html><body><h1>mpr-api dev</h1></body></html>' > cmd/mpr-api/dist/index.html
	go run ./cmd/mpr-api/

# Clean build artifacts
clean:
	rm -f mpr-api mpr-api-linux-armv7
	rm -rf cmd/mpr-api/dist
	rm -rf web/dist web/node_modules
