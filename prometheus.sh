#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NETWORK="realty-net"
CONTAINER="prometheus"

echo "==> Creating Docker network: $NETWORK"
docker network create "$NETWORK" 2>/dev/null || echo "    (already exists)"

echo "==> Stopping existing Prometheus container (if any)"
docker rm -f "$CONTAINER" 2>/dev/null || true

echo "==> Starting Prometheus container"
docker run -d \
  --name "$CONTAINER" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -p 9090:9090 \
  -v "$SCRIPT_DIR/prometheus_config.yaml:/etc/prometheus/prometheus.yml:ro" \
  prom/prometheus:latest \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --web.enable-lifecycle

echo ""
echo "Prometheus is running at: http://localhost:9090"
echo ""
echo "Useful commands:"
echo "  Targets: http://localhost:9090/targets"
echo "  Reload:  curl -X POST http://localhost:9090/-/reload"
echo "  Stop:    docker stop $CONTAINER"
