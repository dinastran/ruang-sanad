#!/bin/bash

# Laju Go - One-Click Deploy Script
# Builds locally, uploads only required artifacts, deploys to server.
# No build tools needed on the server — no Go, no Node, no npm.
#
# Uploads use rsync, so only new/changed files are sent. The binary is uploaded
# under a temporary name while the old one keeps serving, then swapped in right
# before the restart (downtime is just the restart). If the new binary fails to
# start, the previous one is restored automatically.
#
# Usage:
#   ./scripts/deploy.sh            # deploy
#   ./scripts/deploy.sh --dry-run  # build + show what would change, touch nothing on the server

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

DRY_RUN=0
for arg in "$@"; do
    case "$arg" in
        --dry-run|-n) DRY_RUN=1 ;;
        *) echo -e "${RED}Unknown option: $arg${NC}"; exit 1 ;;
    esac
done

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
# Old hashed Vite chunks are kept this many days so open tabs can still navigate
# after a deploy. Chunks referenced by the current manifest are never removed.
ASSET_RETENTION_DAYS=${ASSET_RETENTION_DAYS:-7}

# Validate required variables
if [ -z "$SERVER_USER" ] || [ -z "$SERVER_HOST" ] || [ -z "$SERVER_PATH" ]; then
    echo -e "${RED}Error: Missing required variables in .deploy file${NC}"
    echo "Please ensure SERVER_USER, SERVER_HOST, and SERVER_PATH are set."
    exit 1
fi

REMOTE="$SERVER_USER@$SERVER_HOST"

echo -e "${GREEN}App:      ${YELLOW}$APP_NAME${NC}"
echo -e "${GREEN}Server:   ${YELLOW}$REMOTE${NC}"
echo -e "${GREEN}Path:     ${YELLOW}$SERVER_PATH${NC}"
if [ "$DRY_RUN" = 1 ]; then
    echo -e "${GREEN}Mode:     ${YELLOW}DRY RUN (server tidak diubah)${NC}"
fi
echo ""

# rsync without owner/group so files stay owned by the remote user (like scp did).
RSYNC_OPTS=(-rlptz --partial)
if [ "$DRY_RUN" = 1 ]; then
    RSYNC_OPTS+=(--dry-run --itemize-changes)
fi

# Removes files in <dist>/assets older than N days that no manifest references.
# Args: <dist dir> <days> <dry-run 0|1> <manifest>...
# Shared by the local and remote cleanup (remote runs it via `bash -s`).
PRUNE_SCRIPT='
dir="$1"; days="$2"; dry="$3"; shift 3
[ -d "$dir/assets" ] || exit 0
keep=$(mktemp)
for m in "$@"; do
    [ -f "$m" ] && grep -oE "\"assets/[^\"]+\"" "$m" | tr -d "\"" >> "$keep"
done
if [ ! -s "$keep" ]; then
    echo "  ! manifest tidak ditemukan, cleanup dilewati"
    rm -f "$keep"
    exit 0
fi
count=0
while IFS= read -r f; do
    rel="${f#"$dir"/}"
    grep -qxF "$rel" "$keep" && continue
    [ "$dry" = 1 ] || rm -f "$f"
    count=$((count + 1))
done < <(find "$dir/assets" -type f -mtime +"$days")
rm -f "$keep"
if [ "$dry" = 1 ]; then
    echo "  $count chunk lama (> $days hari) akan dihapus"
else
    echo "  $count chunk lama (> $days hari) dihapus"
fi
'

# Test SSH connection
echo -e "${BLUE}Testing SSH connection...${NC}"
if ! ssh -o ConnectTimeout=10 -o BatchMode=yes "$REMOTE" "echo OK" > /dev/null 2>&1; then
    echo -e "${RED}Error: Cannot connect to server via SSH${NC}"
    echo "Please check your SSH credentials and server accessibility."
    exit 1
fi
if ! ssh "$REMOTE" "command -v rsync" > /dev/null 2>&1; then
    echo -e "${RED}Error: rsync is not installed on the server${NC}"
    echo "Install it first:  ssh $REMOTE 'sudo apt-get install -y rsync'"
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

# Build Go binary for Linux (pure Go SQLite via modernc = no CGO, fully static)
echo -e "${YELLOW}Building Go binary (linux/amd64)...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$APP_NAME" ./cmd/laju-go
echo -e "${GREEN}✓ Binary built: $APP_NAME${NC}"

echo ""

# Detect first vs update deploy by checking if service exists
echo -e "${BLUE}Checking deployment status...${NC}"
IS_FIRST=false
if ssh "$REMOTE" "systemctl is-active $SERVICE_NAME" > /dev/null 2>&1; then
    echo -e "${GREEN}→ Existing deployment detected${NC}"
else
    echo -e "${YELLOW}→ No existing deployment found${NC}"
    IS_FIRST=true
fi
echo ""

# Clean up old local chunks so dist/ doesn't grow forever
echo -e "${BLUE}Cleaning old local chunks...${NC}"
bash -c "$PRUNE_SCRIPT" _ "$PROJECT_ROOT/dist" "$ASSET_RETENTION_DAYS" "$DRY_RUN" \
    "$PROJECT_ROOT/dist/.vite/manifest.json"
echo ""

# Upload artifacts — only what changed. The service keeps running meanwhile.
echo -e "${BLUE}Uploading artifacts (only changed files)...${NC}"

if [ "$DRY_RUN" = 0 ]; then
    # Create remote directory if needed (uses sudo because /opt is root-owned, then
    # hands ownership to the deploy user so uploads work without sudo).
    ssh "$REMOTE" "sudo mkdir -p $SERVER_PATH && sudo chown $SERVER_USER:$SERVER_USER $SERVER_PATH"
    # Remember the manifest the running binary uses, so its chunks survive cleanup.
    ssh "$REMOTE" "[ -f $SERVER_PATH/dist/.vite/manifest.json ] && cp $SERVER_PATH/dist/.vite/manifest.json $SERVER_PATH/dist/.vite/manifest.prev.json || true"
fi

# Binary goes to a temporary name; it is swapped in just before the restart.
echo -e "${YELLOW}Binary...${NC}"
rsync "${RSYNC_OPTS[@]}" --stats "$APP_NAME" "$REMOTE:$SERVER_PATH/$APP_NAME.new" | grep -E '^(Total transferred file size|Total bytes sent)' || true

# dist/: no --delete, old hashed chunks are kept (cleaned up below by age).
echo -e "${YELLOW}dist/...${NC}"
DIST_ITEMS=$(rsync "${RSYNC_OPTS[@]}" --itemize-changes "$PROJECT_ROOT/dist/" "$REMOTE:$SERVER_PATH/dist/")
# "<f+++" = new file, "<f.s" = size changed; "<f..t" = same content, only the timestamp is synced.
DIST_SENT=$(grep -cE '^[<>]f(\+|.s)' <<< "$DIST_ITEMS" || true)
echo "  $DIST_SENT file dikirim (baru/berubah)"

# public/ and migrations/ are mirrored exactly (previously rm -rf + scp).
echo -e "${YELLOW}public/ + migrations/...${NC}"
for d in public migrations; do
    rsync "${RSYNC_OPTS[@]}" --delete --itemize-changes "$PROJECT_ROOT/$d/" "$REMOTE:$SERVER_PATH/$d/" \
        | grep -E '^([<>]f(\+|.s)|\*deleting)' | sed "s|^|  $d: |" || true
done

echo -e "${YELLOW}Cleaning old server chunks...${NC}"
ssh "$REMOTE" "bash -s -- '$SERVER_PATH/dist' '$ASSET_RETENTION_DAYS' '$DRY_RUN' '$SERVER_PATH/dist/.vite/manifest.json' '$SERVER_PATH/dist/.vite/manifest.prev.json'" <<< "$PRUNE_SCRIPT"

if [ "$DRY_RUN" = 1 ]; then
    echo ""
    echo -e "${GREEN}Dry run selesai — tidak ada yang diubah di server.${NC}"
    exit 0
fi

# Swap binaries. Renaming over a running executable is safe on Linux: the running
# process keeps the old inode until it is restarted.
ssh "$REMOTE" "
    cd $SERVER_PATH
    chmod +x $APP_NAME.new
    [ -f $APP_NAME ] && mv -f $APP_NAME $APP_NAME.prev
    mv -f $APP_NAME.new $APP_NAME
"
echo -e "${GREEN}✓ Binary + assets uploaded${NC}"

echo ""

# Run first-deploy or update-deploy
if [ "$IS_FIRST" = true ]; then
    "$SCRIPT_DIR/first-deploy.sh"
else
    if ! "$SCRIPT_DIR/update-deploy.sh"; then
        echo ""
        echo -e "${RED}New version failed to start — rolling back to the previous binary...${NC}"
        ssh "$REMOTE" "
            cd $SERVER_PATH
            [ -f $APP_NAME.prev ] && mv -f $APP_NAME.prev $APP_NAME
            sudo systemctl restart $SERVICE_NAME
        "
        sleep 2
        if ssh "$REMOTE" "systemctl is-active $SERVICE_NAME" > /dev/null 2>&1; then
            echo -e "${YELLOW}✓ Rolled back — previous version is running again${NC}"
        else
            echo -e "${RED}✗ Rollback failed — check: ssh $REMOTE 'sudo journalctl -u $SERVICE_NAME -n 50'${NC}"
        fi
        exit 1
    fi
fi

# Final status
echo ""
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        Deployment Status             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

if ssh "$REMOTE" "systemctl is-active $SERVICE_NAME" &>/dev/null; then
    echo -e "${GREEN}✓ Service $SERVICE_NAME is running${NC}"
else
    echo -e "${RED}✗ Service $SERVICE_NAME is not running${NC}"
fi
if ssh "$REMOTE" "systemctl is-enabled $SERVICE_NAME" &>/dev/null; then
    echo -e "${GREEN}✓ Service enabled (auto-start on boot)${NC}"
else
    echo -e "${YELLOW}! Service not enabled${NC}"
fi

echo ""
echo -e "${CYAN}Useful commands:${NC}"
echo "  View logs:     ssh $REMOTE 'sudo journalctl -u $SERVICE_NAME -f'"
echo "  Check status:  ssh $REMOTE 'systemctl status $SERVICE_NAME'"
echo "  Restart:       ssh $REMOTE 'sudo systemctl restart $SERVICE_NAME'"
echo ""
echo -e "${GREEN}Deployment complete!${NC}"
