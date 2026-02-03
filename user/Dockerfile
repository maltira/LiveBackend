FROM golang:alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/main ./cmd/main.go

FROM alpine

WORKDIR /app

COPY --from=builder /app/main /app/main

CMD ["/app/main"]