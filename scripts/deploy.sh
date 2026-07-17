#!/bin/bash

# Laju Go - One-Click Deploy Script
# Builds locally, uploads only required artifacts, deploys to server.
# No build tools needed on the server — no Go, no Node, no npm.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Laju Go - One-Click Deploy        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Load configuration
if [ ! -f "$PROJECT_ROOT/.deploy" ]; then
    echo -e "${RED}Error: .deploy file not found!${NC}"
    echo -e "${YELLOW}Please copy .deploy.example to .deploy and configure it first.${NC}"
    echo ""
    echo "  cp .deploy.example .deploy"
    echo ""
    exit 1
fi

source "$PROJECT_ROOT/.deploy"

# Set defaults
APP_NAME=${APP_NAME:-laju-go}
SERVICE_NAME=${SERVICE_NAME:-$APP_NAME}

# Validate required variables
if [ -z "$SERVER_USER" ] || [ -z "$SERVER_HOST" ] || [ -z "$SERVER_PATH" ]; then
    echo -e "${RED}Error: Missing required variables in .deploy file${NC}"
    echo "Please ensure SERVER_USER, SERVER_HOST, and SERVER_PATH are set."
    exit 1
fi

echo -e "${GREEN}App:      ${YELLOW}$APP_NAME${NC}"
echo -e "${GREEN}Server:   ${YELLOW}$SERVER_USER@$SERVER_HOST${NC}"
echo -e "${GREEN}Path:     ${YELLOW}$SERVER_PATH${NC}"
echo ""

# Test SSH connection
echo -e "${BLUE}Testing SSH connection...${NC}"
if ! ssh -o ConnectTimeout=10 -o BatchMode=yes "$SERVER_USER@$SERVER_HOST" "echo OK" > /dev/null 2>&1; then
    echo -e "${RED}Error: Cannot connect to server via SSH${NC}"
    echo "Please check your SSH credentials and server accessibility."
    exit 1
fi
echo -e "${GREEN}✓ SSH connection successful${NC}"
echo ""

# Build assets locally
echo -e "${BLUE}Building assets locally...${NC}"

# Build frontend
echo -e "${YELLOW}Building frontend...${NC}"
npm run build
echo -e "${GREEN}✓ Frontend built${NC}"

# Build Go binary for Linux (pure Go SQLite = no CGO needed)
echo -e "${YELLOW}Building Go binary (linux/amd64)...${NC}"
GOOS=linux GOARCH=amd64 go build -o "$APP_NAME" .
echo -e "${GREEN}✓ Binary built: $APP_NAME${NC}"

echo ""

# Detect first vs update deploy by checking if service exists
echo -e "${BLUE}Checking deployment status...${NC}"
IS_FIRST=false
if ssh "$SERVER_USER@$SERVER_HOST" "systemctl is-active $SERVICE_NAME" > /dev/null 2>&1; then
    echo -e "${GREEN}→ Existing deployment detected${NC}"
else
    echo -e "${YELLOW}→ No existing deployment found${NC}"
    IS_FIRST=true
fi
echo ""

# Upload artifacts — only what's needed at runtime
echo -e "${BLUE}Uploading artifacts...${NC}"

# Create remote directory if needed
ssh "$SERVER_USER@$SERVER_HOST" "mkdir -p $SERVER_PATH"

# Upload binary, frontend assets, and migrations
scp "$APP_NAME" "$SERVER_USER@$SERVER_HOST:$SERVER_PATH/"
scp -r dist "$SERVER_USER@$SERVER_HOST:$SERVER_PATH/dist"
scp -r migrations "$SERVER_USER@$SERVER_HOST:$SERVER_PATH/migrations"
ssh "$SERVER_USER@$SERVER_HOST" "chmod +x $SERVER_PATH/$APP_NAME"
echo -e "${GREEN}✓ Binary + assets uploaded${NC}"

echo ""

# Run first-deploy or update-deploy
if [ "$IS_FIRST" = true ]; then
    "$SCRIPT_DIR/first-deploy.sh"
else
    "$SCRIPT_DIR/update-deploy.sh"
fi

# Final status
echo ""
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        Deployment Status             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

if ssh "$SERVER_USER@$SERVER_HOST" "systemctl is-active $SERVICE_NAME" &>/dev/null; then
    echo -e "${GREEN}✓ Service $SERVICE_NAME is running${NC}"
else
    echo -e "${RED}✗ Service $SERVICE_NAME is not running${NC}"
fi
if ssh "$SERVER_USER@$SERVER_HOST" "systemctl is-enabled $SERVICE_NAME" &>/dev/null; then
    echo -e "${GREEN}✓ Service enabled (auto-start on boot)${NC}"
else
    echo -e "${YELLOW}! Service not enabled${NC}"
fi

echo ""
echo -e "${CYAN}Useful commands:${NC}"
echo "  View logs:     ssh $SERVER_USER@$SERVER_HOST 'journalctl -u $SERVICE_NAME -f'"
echo "  Check status:  ssh $SERVER_USER@$SERVER_HOST 'systemctl status $SERVICE_NAME'"
echo "  Restart:       ssh $SERVER_USER@$SERVER_HOST 'systemctl restart $SERVICE_NAME'"
echo ""
echo -e "${GREEN}Deployment complete!${NC}"
