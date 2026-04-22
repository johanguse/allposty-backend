FROM golang:1.22-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-w -s" -o bin/api ./cmd/api
RUN go build -ldflags="-w -s" -o bin/worker ./cmd/worker

# ---

FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/api ./api
COPY --from=builder /app/bin/worker ./worker

EXPOSE 8080

CMD ["./api"]
