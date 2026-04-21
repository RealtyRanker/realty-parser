#!/bin/bash
set -e

NETWORK="realty-net"
CONTAINER="grafana"

echo "==> Creating Docker network: $NETWORK"
docker network create "$NETWORK" 2>/dev/null || echo "    (already exists)"

echo "==> Stopping existing Grafana container (if any)"
docker rm -f "$CONTAINER" 2>/dev/null || true

echo "==> Starting Grafana container"
docker run -d \
  --name "$CONTAINER" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -p 3004:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  -e GF_USERS_ALLOW_SIGN_UP=false \
  grafana/grafana:latest

echo ""
echo "Grafana is running at: http://localhost:3004"
echo "  Login:    admin / admin"
echo ""
echo "Useful commands:"
echo "  Stop: docker stop $CONTAINER"
