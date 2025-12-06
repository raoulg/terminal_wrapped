#!/bin/bash
set -e

SERVER_IP="145.38.190.202"
USER="rgrouls" # Update this if known, otherwise user must edit
REMOTE_DIR="~/proxy"

echo "🚀 Deploying proxy to $SERVER_IP..."

# Copy files
echo "📦 Copying files..."
ssh $USER@$SERVER_IP "mkdir -p $REMOTE_DIR"
scp proxy/main.go proxy/Dockerfile proxy/docker-compose.yml proxy/.env proxy/.env.example $USER@$SERVER_IP:$REMOTE_DIR/

# Rebuild and restart
echo "🔄 Rebuilding and restarting..."
ssh $USER@$SERVER_IP "cd $REMOTE_DIR && docker compose up --build -d"

echo "✅ Deployment complete!"
echo "🧪 Testing health..."
curl http://$SERVER_IP:8080/health
