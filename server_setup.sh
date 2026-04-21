#!/bin/bash
set -e

NETWORK="realty-net"
APP_IMAGE="realty-parser"
APP_CONTAINER="realty-parser"
LOG_DIR="/tmp/realty-parser-logs"

echo "==> Building parser image: $APP_IMAGE"
docker build -t "$APP_IMAGE" .

echo "==> Stopping existing parser container (if any)"
docker rm -f "$APP_CONTAINER" 2>/dev/null || true

echo "==> Creating log directory: $LOG_DIR"
mkdir -p "$LOG_DIR"

echo "==> Starting parser container: $APP_CONTAINER"
docker run -d \
  --name "$APP_CONTAINER" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -p 9090:9090 \
  -v "$(pwd)/config.yaml:/app/config.yaml:ro" \
  -v "$LOG_DIR:/var/log/realty-parser" \
  "$APP_IMAGE"

echo ""
echo "Useful commands:"
echo "  Logs:    docker logs -f $APP_CONTAINER"
echo "  Metrics: curl http://localhost:9090/metrics"
echo "  Health:  curl http://localhost:9090/healthz"
echo "  Stop:    docker stop $APP_CONTAINER"
