#!/bin/bash

# Wait for Docker to be ready
echo "Waiting for Docker to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if docker info &> /dev/null; then
        echo "Docker is ready!"
        break
    fi
    
    echo "Waiting for Docker to start... ($((ATTEMPT+1))/$MAX_ATTEMPTS)"
    sleep 2
    ((ATTEMPT++))
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo "Docker did not start within the expected time."
        echo "Please ensure Docker Desktop is running and try again."
        exit 1
    fi
done

# Build and run the Docker container
echo "Building Docker image..."
docker build -t realy-parser-image .

echo "Running Docker container..."
docker run -d -p 8086:8086 --rm realy-parser-image

echo "Docker container is running!"
echo "Your application should be available at: http://localhost:8086"
echo ""
echo "To stop the container, run: docker stop $(docker ps -q --filter ancestor=realy-parser-image)"
echo "To view running containers: docker ps"
echo "To view logs: docker logs $(docker ps -q --filter ancestor=realy-parser-image)"