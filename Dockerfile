# syntax=docker/dockerfile:1

# Build stages run on the builder's native platform and cross-compile,
# so multi-arch images don't need emulation.
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS api
ARG TARGETOS TARGETARCH VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /src/web/dist web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags="-s -w -X main.version=$VERSION" -o /out/kipitfit ./cmd/kipitfit
# distroless has no shell to create the data dir with the right owner.
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=api /out/kipitfit /kipitfit
COPY --from=api --chown=65532:65532 /out/data /data
ENV KIPITFIT_DATA_DIR=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/kipitfit"]
