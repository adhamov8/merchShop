FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /avito-shop ./cmd/main.go

FROM alpine:3.17
WORKDIR /app
COPY --from=builder /avito-shop /app/avito-shop

EXPOSE 8080
ENTRYPOINT ["/app/avito-shop"]
