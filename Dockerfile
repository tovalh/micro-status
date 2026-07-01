# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26 AS build
WORKDIR /src

# Cache de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Código fuente
COPY . .

# Binario estático para Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o /health_status ./cmd

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12
WORKDIR /app

COPY --from=build /health_status /app/health_status
COPY config.yaml /app/config.yaml

ENTRYPOINT ["/app/health_status"]