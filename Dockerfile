FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /realty-parser ./cmd/parser
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /yandex-import ./cmd/yandex-import
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /domclick-import ./cmd/domclick-import

# ---

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN mkdir -p /var/log/realty-parser

WORKDIR /app
COPY --from=builder /realty-parser .
COPY --from=builder /yandex-import /usr/local/bin/yandex-import
COPY --from=builder /domclick-import /usr/local/bin/domclick-import
COPY config.yaml .

EXPOSE 9095

CMD ["./realty-parser", "-config", "config.yaml"]
