#!/bin/bash

MYNODE=$1

if [ -z "$MYNODE" ]; then
  echo "❌ Please provide a node name. Usage: ./run.sh <mynode>"
  exit 1
fi

set -a
source ./${MYNODE}/.env
set +a

# Step 1: Build the Docker image
echo "🔨 Building Docker image..."
docker build -t test .

# Step 2: Auto-setup (generates .env)
echo "⚙️ Running auto-setup for node: $MYNODE"
docker run --rm \
  -i \
  -v "$(pwd)/$MYNODE:/app/$MYNODE" \
  test \
  mrmintchain auto-setup --mynode $MYNODE

# Step 3: Start the node with Docker Compose
echo "🚀 Starting the node using docker-compose..."
MYNODE=$MYNODE docker compose --env-file ./$MYNODE/.env up -d

# For new container
# MYNODE=$MYNODE docker compose -p validator2 --env-file ./$MYNODE/.env up -d
