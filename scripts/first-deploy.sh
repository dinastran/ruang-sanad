#!/bin/bash

# Laju Go - First Deploy Script
# Sets up the application and systemd service from scratch.
# Run AFTER deploy.sh has uploaded artifacts to the server.

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

echo -e "${BLUE}═══ FIRST DEPLOY ═══${NC}"
echo ""

# Interactive prompts for environment configuration
echo -e "${YELLOW}Application Port (default: 8080):${NC}"
read -r APP_PORT_INPUT
APP_PORT=${APP_PORT_INPUT:-8080}

echo -e "${YELLOW}Application URL (e.g., https://yourdomain.com):${NC}"
read -r APP_URL

# Auto-generate SESSION_SECRET
SESSION_SECRET=$(openssl rand -hex 32)

# Step 1: Create remote directories
echo -e "${YELLOW}[1/4] Creating remote directories...${NC}"
ssh "$SERVER_USER@$SERVER_HOST" "mkdir -p $SERVER_PATH/data $SERVER_PATH/storage $SERVER_PATH/backups"
echo -e "${GREEN}      ✓ Directories created${NC}"

# Step 2: Setup .env file
echo -e "${YELLOW}[2/4] Setting up environment file...${NC}"
# Upload .env.example for reference
scp "$PROJECT_ROOT/.env.example" "$SERVER_USER@$SERVER_HOST:$SERVER_PATH/"

# Create .env from template
ssh "$SERVER_USER@$SERVER_HOST" "
    if [ ! -f $SERVER_PATH/.env ]; then
        cp $SERVER_PATH/.env.example $SERVER_PATH/.env
        sed -i 's/APP_PORT=8080/APP_PORT=$APP_PORT/g' $SERVER_PATH/.env
        sed -i \"s|APP_URL=http://localhost:8080|APP_URL=$APP_URL|g\" $SERVER_PATH/.env
        sed -i 's/APP_ENV=development/APP_ENV=production/g' $SERVER_PATH/.env
        sed -i 's/SESSION_SECRET=your-secret-key-change-this-in-production/SESSION_SECRET=$SESSION_SECRET/g' $SERVER_PATH/.env
        sed -i \"s|DB_PATH=./data/app.db|DB_PATH=$SERVER_PATH/data/app.db|g\" $SERVER_PATH/.env
        echo '      Created .env from .env.example (production-ready)'
    else
        echo '      .env already exists, skipping'
    fi
"
echo -e "${GREEN}      ✓ Environment configured${NC}"

# Step 3: Upload systemd service file
echo -e "${YELLOW}[3/4] Setting up systemd service...${NC}"

# Check if systemd service file exists locally
SERVICE_FILE="$PROJECT_ROOT/systemd/$APP_NAME.service"
if [ ! -f "$SERVICE_FILE" ]; then
    SERVICE_FILE="$PROJECT_ROOT/systemd/laju-go.service"
fi

if [ -f "$SERVICE_FILE" ]; then
    # Upload and configure service file
    scp "$SERVICE_FILE" "$SERVER_USER@$SERVER_HOST:/etc/systemd/system/$SERVICE_NAME.service"
    ssh "$SERVER_USER@$SERVER_HOST" "
        sed -i 's|/opt/APP_NAME|$SERVER_PATH|g' /etc/systemd/system/$SERVICE_NAME.service
        sed -i 's|APP_NAME|$APP_NAME|g' /etc/systemd/system/$SERVICE_NAME.service
        sed -i 's|SyslogIdentifier=laju-go|SyslogIdentifier=$SERVICE_NAME|g' /etc/systemd/system/$SERVICE_NAME.service
    "
else
    # Create service file directly on server
    ssh "$SERVER_USER@$SERVER_HOST" "cat > /etc/systemd/system/$SERVICE_NAME.service << 'SERVICEEOF'
[Unit]
Description=$APP_NAME Application
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$SERVER_PATH
ExecStart=$SERVER_PATH/$APP_NAME
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$SERVER_PATH/data $SERVER_PATH/storage $SERVER_PATH/backups

EnvironmentFile=$SERVER_PATH/.env

[Install]
WantedBy=multi-user.target
SERVICEEOF"
fi

# Enable and start
ssh "$SERVER_USER@$SERVER_HOST" "
    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME
"

sleep 2
echo -e "${GREEN}      ✓ Service created and started${NC}"

# Step 4: Set permissions for data directories
echo -e "${YELLOW}[4/4] Setting up permissions...${NC}"
ssh "$SERVER_USER@$SERVER_HOST" "
    chmod 755 $SERVER_PATH/data
    chmod 770 $SERVER_PATH/storage
    chmod 770 $SERVER_PATH/backups
"
echo -e "${GREEN}      ✓ Permissions set${NC}"

# Verify
echo ""
echo -e "${BLUE}Verifying service...${NC}"
if ssh "$SERVER_USER@$SERVER_HOST" "systemctl is-active $SERVICE_NAME" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Service is running${NC}"
else
    echo -e "${RED}Service failed to start. Check logs:${NC}"
    ssh "$SERVER_USER@$SERVER_HOST" "journalctl -u $SERVICE_NAME -n 30 --no-pager"
    exit 1
fi

echo ""
echo -e "${GREEN}═══ FIRST DEPLOY COMPLETE ═══${NC}"
echo ""
echo -e "${YELLOW}Next steps: Configure OAuth & SMTP for full functionality:${NC}"
echo "  ssh $SERVER_USER@$SERVER_HOST 'nano $SERVER_PATH/.env'"
echo ""
echo "  # Google OAuth (get from console.cloud.google.com)"
echo "  GOOGLE_CLIENT_ID=your-client-id"
echo "  GOOGLE_CLIENT_SECRET=your-secret"
echo "  GOOGLE_REDIRECT_URL=$APP_URL/auth/google/callback"
echo ""
echo "  # SMTP (for password reset)"
echo "  SMTP_HOST=smtp.gmail.com"
echo "  SMTP_USER=your-email@gmail.com"
echo "  SMTP_PASS=your-app-password"
echo "  FROM_EMAIL=noreply@yourdomain.com"
echo ""
