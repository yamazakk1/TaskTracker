FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/tasktracker ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/tasktracker .
# ⚠️ ВАЖНО: Копируем config.yml
COPY --from=builder /app/config.yml .

EXPOSE 8080

CMD ["./tasktracker"]