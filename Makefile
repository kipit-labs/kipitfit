BIN := bin/kipitfit
VERSION ?= $(shell cat VERSION)
export CGO_ENABLED := 0
# Only goes up: raise it when coverage does.
COVERAGE_FLOOR := 87
PLAYWRIGHT_VERSION = $(shell node -p "require('./web/node_modules/@playwright/test/package.json').version")
DEV_ENV := KIPITFIT_ADDR=:8080 KIPITFIT_DATA_DIR=./data KIPITFIT_LOG_LEVEL=debug KIPITFIT_APP_URL=http://localhost:5173

.PHONY: all build ui ui-stub dev-api dev-web test cover e2e lint docker clean

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

# Every Go test with the race detector and coverage of internal/; fails under COVERAGE_FLOOR.
cover: ui-stub
	CGO_ENABLED=1 go test -race -coverpkg=./internal/... -coverprofile=cover.out ./...
	@go tool cover -func=cover.out | awk -v floor=$(COVERAGE_FLOOR) '/^total:/ { sub("%", "", $$3); print "coverage " $$3 "% (floor " floor "%)"; if ($$3 + 0 < floor) exit 1 }'

# UI tests in real browsers (WebKit at phone size, Chromium desktop), in Playwright's image: no browser to install.
e2e: build
	docker run --rm --network host --ipc host -u $$(id -u):$$(id -g) -e HOME=/tmp -e CI \
		-v $(CURDIR):$(CURDIR) -w $(CURDIR)/web mcr.microsoft.com/playwright:v$(PLAYWRIGHT_VERSION)-noble \
		node_modules/.bin/playwright test $(ARGS)

lint: ui-stub
	@unformatted=$$(gofmt -l cmd internal web); if [ -n "$$unformatted" ]; then echo "Run gofmt on:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	cd web && pnpm lint && pnpm exec tsc -b

docker:
	docker build --build-arg VERSION=$(VERSION) -t kipitfit -t ghcr.io/kipit-labs/kipitfit:dev .

clean:
	rm -rf bin web/dist
