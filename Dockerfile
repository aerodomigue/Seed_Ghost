# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o seedghost ./cmd/seedghost/

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /app/seedghost .
COPY --from=backend /app/internal/client/profiles ./profiles
RUN mkdir -p /app/data

ENV SEEDGHOST_LISTEN_ADDR=:8333
ENV SEEDGHOST_DB_PATH=/app/data/seedghost.db
ENV SEEDGHOST_PROFILES_DIR=/app/profiles
ENV SEEDGHOST_DATA_DIR=/app/data

EXPOSE 8333

VOLUME ["/app/data"]

ENTRYPOINT ["./seedghost"]
