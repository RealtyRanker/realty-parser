FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /realty-parser ./cmd/parser

# ---

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN mkdir -p /var/log/realty-parser

WORKDIR /app
COPY --from=builder /realty-parser .
COPY config.yaml .

EXPOSE 9090

CMD ["./realty-parser", "-config", "config.yaml"]
