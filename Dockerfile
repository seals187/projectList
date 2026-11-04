# syntax=docker/dockerfile:1

# ---- 1) Frontend (SvelteKit, adapter-static) --------------------------------
FROM node:26-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- 2) Go-Build (CGO-frei, Frontend eingebettet) ---------------------------
FROM golang:1.27-alpine AS build
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
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o /out/projectlist ./cmd/projectlist \
 && mkdir -p /out/data

# ---- 3) Laufzeit-Image (distroless, ohne Shell, nonroot) --------------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=compile /out/projectlist /projectlist
COPY --from=compile --chown=65532:65532 /out/data /data
ENV DB_PATH=/data/projectlist.db \
    LISTEN_ADDR=:8080
EXPOSE 8080
VOLUME ["/data"]
USER nonroot:nonroot
ENTRYPOINT ["/projectlist"]
CMD ["serve"]
