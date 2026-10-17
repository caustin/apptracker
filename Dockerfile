# syntax=docker/dockerfile:1
# Build: docker build -t apptracker .
# Run:   docker run -p 8080:8080 -v apptracker-data:/data \
#          -e APP_ENV=production -e APP_URL=https://your-domain apptracker

FROM node:22-bookworm-slim AS web
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY index.html vite.config.js ./
COPY src ./src
COPY internal/web/dist/.gitkeep ./internal/web/dist/.gitkeep
RUN npm run build

FROM golang:1.26-bookworm AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY docs ./docs
COPY --from=web /src/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/apptracker ./cmd/apptracker \
 && mkdir -p /out/data

# Static, CGO-free binary: distroless with a non-root user and CA certs.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go /out/apptracker /apptracker
# Owned by nonroot so a fresh named volume mounted here is writable.
COPY --from=go --chown=nonroot:nonroot /out/data /data
ENV APP_ENV=production \
    DB_PATH=/data/app.db \
    DATA_DIR=/data \
    PORT=8080
# The database and resume files live on this volume.
VOLUME ["/data"]
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/apptracker"]
