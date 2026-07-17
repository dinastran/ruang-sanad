#!/bin/bash

# Laju Go - Update Deploy Script
# Restarts the service after deploy.sh has uploaded updated artifacts.
# No git pull or source code sync — only binary + dist are updated.

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

source "$PROJECT_ROOT/.deploy"

# Set defaults
APP_NAME=${APP_NAME:-laju-go}
SERVICE_NAME=${SERVICE_NAME:-$APP_NAME}

echo -e "${BLUE}═══ UPDATE DEPLOY ═══${NC}"
echo ""

# Step 1: Stop service
echo -e "${YELLOW}[1/3] Stopping service...${NC}"
ssh "$SERVER_USER@$SERVER_HOST" "systemctl stop $SERVICE_NAME" || true
echo -e "${GREEN}      ✓ Service stopped${NC}"

# Step 2: Restart service (artifacts already uploaded by deploy.sh)
echo -e "${YELLOW}[2/3] Restarting service...${NC}"
ssh "$SERVER_USER@$SERVER_HOST" "
    systemctl daemon-reload
    systemctl start $SERVICE_NAME
"
sleep 2
echo -e "${GREEN}      ✓ Service restarted${NC}"

# Step 3: Verify
echo -e "${YELLOW}[3/3] Verifying service...${NC}"
if ssh "$SERVER_USER@$SERVER_HOST" "systemctl is-active $SERVICE_NAME" > /dev/null 2>&1; then
    echo -e "${GREEN}      ✓ Service is running${NC}"
else
    echo -e "${RED}Service failed to start. Check logs:${NC}"
    ssh "$SERVER_USER@$SERVER_HOST" "journalctl -u $SERVICE_NAME -n 30 --no-pager"
    exit 1
fi

# Show recent logs
echo ""
echo -e "${BLUE}Recent logs:${NC}"
ssh "$SERVER_USER@$SERVER_HOST" "journalctl -u $SERVICE_NAME -n 5 --no-pager"

echo ""
echo -e "${GREEN}═══ UPDATE DEPLOY COMPLETE ═══${NC}"
