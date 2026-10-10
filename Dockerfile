# syntax=docker/dockerfile:1

# The frontend is plain files, so it is built once on the build machine whatever the target.
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Go cross-compiles, so the target architecture needs no emulation.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-build
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/server ./cmd/server

# Distroless static: CA certificates and tzdata (the TIMEZONE setting needs it), no shell, non-root.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=backend-build /out/server ./server
COPY --from=frontend-build /app/frontend/dist ./frontend/dist
EXPOSE 8080
# The image has no shell or curl, so the binary checks itself.
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 CMD ["/app/server", "healthcheck"]
ENTRYPOINT ["/app/server"]
