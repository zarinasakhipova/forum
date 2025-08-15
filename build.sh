#!/bin/sh
set -e

echo "[1/3] Building Docker image..."
docker build -t forum .

echo "[2/3] Removing old container (if exists)..."
docker rm -f forum 2>/dev/null || true

echo "[3/3] Running new container..."
docker run -d -p 8080:8080 --name forum forum

echo "Done! Forum is running at http://localhost:8080" 