FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build inference server
RUN CGO_ENABLED=0 GOOS=linux go build -o /inference-server ./cmd/inference-server

# Build management server
RUN CGO_ENABLED=0 GOOS=linux go build -o /management-server ./cmd/management-server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata

# Copy both binaries
COPY --from=builder /inference-server /inference-server
COPY --from=builder /management-server /management-server
COPY migrations /migrations

EXPOSE 8080 8081

# Default: run inference server (can be overridden by command in docker-compose)
CMD ["/inference-server"]
