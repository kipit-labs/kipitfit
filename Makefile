BIN := bin/kipitfit
VERSION ?= $(shell cat VERSION)
export CGO_ENABLED := 0
DEV_ENV := KIPITFIT_ADDR=:8080 KIPITFIT_DATA_DIR=./data KIPITFIT_LOG_LEVEL=debug KIPITFIT_APP_URL=http://localhost:5173

.PHONY: all build ui ui-stub dev-api dev-web test lint docker clean

all: build

ui:
	cd web && pnpm install --frozen-lockfile && pnpm build

# go:embed needs web/dist to exist; stub it when the UI hasn't been built.
ui-stub:
	@test -f web/dist/index.html || (mkdir -p web/dist && \
		echo '<!doctype html><p>UI not built. Run <code>make ui</code> or use the Vite dev server.</p>' > web/dist/index.html)

build: ui
	go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(BIN) ./cmd/kipitfit

# Go API on :8080; run `make dev-web` alongside for the Vite dev server on :5173 (proxies /api).
dev-api: ui-stub
	$(DEV_ENV) go run ./cmd/kipitfit

dev-web:
	cd web && pnpm dev

test: ui-stub
	go test ./...
	cd web && pnpm test

lint: ui-stub
	@unformatted=$$(gofmt -l cmd internal web); if [ -n "$$unformatted" ]; then echo "Run gofmt on:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	cd web && pnpm lint && pnpm exec tsc -b

docker:
	docker build --build-arg VERSION=$(VERSION) -t kipitfit -t ghcr.io/kipit-labs/kipitfit:dev .

clean:
	rm -rf bin web/dist
