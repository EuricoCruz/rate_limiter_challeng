#!/bin/bash
set -e

echo "🚀 Starting Redis for integration tests..."
docker-compose -f docker-compose.test.yml up -d

echo "⏳ Waiting for Redis to be healthy..."
# Wait up to 30 seconds for Redis to be healthy
counter=0
max_attempts=30
until docker-compose -f docker-compose.test.yml ps | grep -q "healthy"; do
    sleep 1
    counter=$((counter + 1))
    if [ $counter -eq $max_attempts ]; then
        echo "❌ Redis failed to become healthy within 30 seconds"
        echo "Current status:"
        docker-compose -f docker-compose.test.yml ps
        exit 1
    fi
done
echo "✅ Redis is healthy and ready for tests"

echo "🧪 Running integration tests..."
go test -tags=integration ./tests/integration/... -v

echo "🧹 Cleaning up..."
docker-compose -f docker-compose.test.yml down -v

echo "✅ Integration tests completed!"
