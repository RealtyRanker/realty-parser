#!/bin/bash
set -e

NETWORK="realty-net"
KAFKA_CONTAINER="realty-kafka"

echo "==> Creating Docker network: $NETWORK"
docker network create "$NETWORK" 2>/dev/null || echo "    (already exists)"

echo "==> Stopping existing Kafka container (if any)"
docker rm -f "$KAFKA_CONTAINER" 2>/dev/null || true

echo "==> Starting Kafka container (KRaft mode): $KAFKA_CONTAINER"
docker run -d \
  --name "$KAFKA_CONTAINER" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -e KAFKA_NODE_ID=1 \
  -e KAFKA_PROCESS_ROLES=broker,controller \
  -e KAFKA_CONTROLLER_QUORUM_VOTERS="1@realty-kafka:9093" \
  -e KAFKA_LISTENERS="PLAINTEXT://:9092,CONTROLLER://:9093" \
  -e KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://realty-kafka:9092" \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP="CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT" \
  -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER \
  -e KAFKA_INTER_BROKER_LISTENER_NAME=PLAINTEXT \
  -e KAFKA_AUTO_CREATE_TOPICS_ENABLE=true \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1 \
  -p 9092:9092 \
  apache/kafka:latest

echo "==> Waiting for Kafka to be ready..."
until docker exec "$KAFKA_CONTAINER" /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list >/dev/null 2>&1; do
  sleep 2
done
echo "    Kafka is ready."

echo "==> Creating topic: realty.flats"
docker exec "$KAFKA_CONTAINER" /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create --if-not-exists \
  --topic realty.flats \
  --partitions 1 \
  --replication-factor 1

echo ""
echo "Useful commands:"
echo "  Topics:   docker exec $KAFKA_CONTAINER /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list"
echo "  Consume:  docker exec $KAFKA_CONTAINER /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic realty.flats --from-beginning"
echo "  Stop:     docker stop $KAFKA_CONTAINER"
