# syntax=docker/dockerfile:1

# Mehrere Architekturen (docker buildx --platform linux/amd64,linux/arm64):
# Frontend und Go laufen nativ auf der Build-Plattform ($BUILDPLATFORM), das
# Go-Binary wird für die Zielplattform cross-kompiliert (CGO-frei) – keine Emulation.

# ---- 1) Frontend (SvelteKit, adapter-static) --------------------------------
FROM --platform=$BUILDPLATFORM node:26-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- 2) Go-Build (CGO-frei, Frontend eingebettet) ---------------------------
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-trimpath
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /web/build ./web/build

# Tests: docker build --target test .
FROM build AS test
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go vet ./... && go test ./...

FROM build AS compile
ARG VERSION=dev
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w -X main.version=${VERSION}" -o /out/projectlist ./cmd/projectlist \
 && mkdir -p /out/data

# ---- 3) Laufzeit-Image (distroless, ohne Shell, nonroot) --------------------
FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
LABEL org.opencontainers.image.title="ProjectList" \
      org.opencontainers.image.description="Schlanke, selbst gehostete Projektverwaltung: Kunden, Projekte, Checklisten" \
      org.opencontainers.image.source="https://github.com/seals187/projectList" \
      org.opencontainers.image.licenses="AGPL-3.0-only" \
      org.opencontainers.image.version="$VERSION"
COPY --from=compile /out/projectlist /projectlist
COPY --from=compile --chown=65532:65532 /out/data /data
ENV DB_PATH=/data/projectlist.db \
    LISTEN_ADDR=:8080
EXPOSE 8080
VOLUME ["/data"]
USER nonroot:nonroot
ENTRYPOINT ["/projectlist"]
CMD ["serve"]
