#!/bin/bash
set -e

NETWORK="realty-net"
PG_IMAGE="realty-postgres"
PG_CONTAINER="realty-postgres"

PG_USER="realty_parser"
PG_PASSWORD="password"
PG_DB="realty_parser"

echo "==> Creating Docker network: $NETWORK"
docker network create "$NETWORK" 2>/dev/null || echo "    (already exists)"

echo "==> Building Postgres image: $PG_IMAGE"
docker build -f Dockerfile.postgres -t "$PG_IMAGE" .

echo "==> Stopping existing Postgres container (if any)"
docker rm -f "$PG_CONTAINER" 2>/dev/null || true

echo "==> Starting Postgres container: $PG_CONTAINER"
docker run -d \
  --name "$PG_CONTAINER" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -e POSTGRES_USER="$PG_USER" \
  -e POSTGRES_PASSWORD="$PG_PASSWORD" \
  -e POSTGRES_DB="$PG_DB" \
  -p 5432:5432 \
  "$PG_IMAGE"

echo "==> Waiting for Postgres to be ready..."
until docker exec "$PG_CONTAINER" pg_isready -U "$PG_USER" -q; do
  sleep 1
done
echo "    Postgres is ready."

echo ""
echo "Useful commands:"
echo "  Postgres CLI: docker exec -it $PG_CONTAINER psql -U $PG_USER -d $PG_DB"
echo "  Stop:         docker stop $PG_CONTAINER"
