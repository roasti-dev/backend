#!/usr/bin/env bash
# Seed the database with test data via the API.
# Usage: ./scripts/seed/seed.sh [--host HOST] [--data DIR] [--images DIR] [--email EMAIL] [--password PASSWORD] [--username USERNAME]
#
# Options:
#   --host       Base URL of the server (default: http://localhost:9090)
#   --data       Directory with JSON seed files (default: scripts/seed/data)
#   --images     Directory with seed images (default: scripts/seed/images)
#   --email      Test user email (default: seed@example.com)
#   --password   Test user password (default: Seed1234!)
#   --username   Test user username (default: seeduser)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

HOST="http://localhost:9090"
DATA_DIR="$SCRIPT_DIR/data"
IMAGES_DIR="$SCRIPT_DIR/images"
EMAIL="seed@example.com"
PASSWORD="Seed1234!"
USERNAME="seeduser"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)     HOST="$2";     shift 2 ;;
    --data)     DATA_DIR="$2"; shift 2 ;;
    --images)   IMAGES_DIR="$2"; shift 2 ;;
    --email)    EMAIL="$2";    shift 2 ;;
    --password) PASSWORD="$2"; shift 2 ;;
    --username) USERNAME="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ── helpers ──────────────────────────────────────────────────────────────────

log() { echo "→ $*"; }

api() {
  local method="$1"; shift
  local path="$1";   shift
  curl -fsSL -X "$method" "$HOST$path" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    "$@"
}

# Upload an image file, return its ID.
upload_image() {
  local file="$1"
  curl -fsSL -X POST "$HOST/api/v1/uploads/images" \
    -H "Authorization: Bearer $TOKEN" \
    -F "image=@$file" \
    | jq -r '.id'
}

# ── auth ──────────────────────────────────────────────────────────────────────

log "Logging in as $EMAIL..."
TOKEN=$(curl -fsSL -X POST "$HOST/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" \
  | jq -r '.access_token' 2>/dev/null || true)

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  log "Login failed, registering user..."
  curl -fsSL -X POST "$HOST/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" \
    > /dev/null

  TOKEN=$(curl -fsSL -X POST "$HOST/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" \
    | jq -r '.access_token')
fi

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "ERROR: Failed to obtain access token" >&2
  exit 1
fi

log "Authenticated."

# ── images ────────────────────────────────────────────────────────────────────

declare -A IMAGE_IDS

if [[ -d "$IMAGES_DIR" ]]; then
  log "Uploading images from $IMAGES_DIR..."
  for img in "$IMAGES_DIR"/*; do
    [[ -f "$img" ]] || continue
    [[ "$img" =~ \.(jpg|jpeg|png|webp)$ ]] || continue
    name=$(basename "$img")
    id=$(upload_image "$img")
    IMAGE_IDS["$name"]="$id"
    log "  $name → $id"
  done
fi

# helper: resolve image_id field — looks up IMAGE_IDS by filename, returns empty string if not found
image_id_for() {
  local filename="$1"
  echo "${IMAGE_IDS[$filename]:-}"
}

# ── beans ─────────────────────────────────────────────────────────────────────

BEANS_FILE="$DATA_DIR/beans.json"
if [[ -f "$BEANS_FILE" ]]; then
  log "Seeding beans..."
  count=$(jq '.beans | length' "$BEANS_FILE")
  for i in $(seq 0 $((count - 1))); do
    item=$(jq ".beans[$i]" "$BEANS_FILE")
    img_file=$(echo "$item" | jq -r '.image // empty')
    if [[ -n "$img_file" ]]; then
      img_id=$(image_id_for "$img_file")
      [[ -n "$img_id" ]] && item=$(echo "$item" | jq --arg id "$img_id" '. + {image_id: $id}')
    fi
    payload=$(echo "$item" | jq 'del(.image)')
    name=$(echo "$item" | jq -r '.name')
    api POST /api/v1/beans -d "$payload" > /dev/null
    log "  bean: $name"
  done
fi

# ── recipes ───────────────────────────────────────────────────────────────────

RECIPES_FILE="$DATA_DIR/recipes.json"
if [[ -f "$RECIPES_FILE" ]]; then
  log "Seeding recipes..."
  count=$(jq '.recipes | length' "$RECIPES_FILE")
  for i in $(seq 0 $((count - 1))); do
    item=$(jq ".recipes[$i]" "$RECIPES_FILE")
    img_file=$(echo "$item" | jq -r '.image // empty')
    if [[ -n "$img_file" ]]; then
      img_id=$(image_id_for "$img_file")
      [[ -n "$img_id" ]] && item=$(echo "$item" | jq --arg id "$img_id" '. + {image_id: $id}')
    fi
    payload=$(echo "$item" | jq 'del(.image)')
    title=$(echo "$item" | jq -r '.title')
    api POST /api/v1/recipes -d "$payload" > /dev/null
    log "  recipe: $title"
  done
fi

# ── posts ─────────────────────────────────────────────────────────────────────

POSTS_FILE="$DATA_DIR/posts.json"
if [[ -f "$POSTS_FILE" ]]; then
  log "Seeding posts..."
  count=$(jq '.posts | length' "$POSTS_FILE")
  for i in $(seq 0 $((count - 1))); do
    item=$(jq ".posts[$i]" "$POSTS_FILE")
    # resolve image references inside blocks
    block_count=$(echo "$item" | jq '.blocks | length')
    for b in $(seq 0 $((block_count - 1))); do
      img_files=$(echo "$item" | jq -r ".blocks[$b].images[]? // empty")
      resolved_ids="[]"
      while IFS= read -r img_file; do
        [[ -z "$img_file" ]] && continue
        img_id=$(image_id_for "$img_file")
        [[ -n "$img_id" ]] && resolved_ids=$(echo "$resolved_ids" | jq --arg id "$img_id" '. + [$id]')
      done <<< "$img_files"
      item=$(echo "$item" | jq --argjson ids "$resolved_ids" --argjson b "$b" \
        '.blocks[$b].images = $ids')
    done
    payload="$item"
    title=$(echo "$item" | jq -r '.title')
    api POST /api/v1/posts -d "$payload" > /dev/null
    log "  post: $title"
  done
fi

log "Done."
