#!/bin/bash
set -e

# Configuration
SERVER_URL="http://localhost:8080"
EMAIL="testuser@example.com"
USERNAME="testuser"
PASSWORD="securepassword123"

echo "🧪 Testing Magic Link Flow..."

# 1. Register User (ignore error if exists)
echo "1. Registering user..."
curl -s -X POST "$SERVER_URL/api/v1/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"$USERNAME\", \"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}" \
  > /dev/null || true

# 2. Request Magic Link
echo "2. Requesting magic link..."
RESPONSE=$(curl -s -X POST "$SERVER_URL/api/v1/magic-link" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\"}")

echo "   Response: $RESPONSE"

# Extract token (simple grep/sed as jq might not be installed)
TOKEN=$(echo $RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "[ERROR] Failed to get magic link token"
  exit 1
fi

echo "   Received Token: $TOKEN"

# 3. Verify Magic Link
echo "3. Verifying magic link..."
VERIFY_RESPONSE=$(curl -s -X GET "$SERVER_URL/api/v1/magic-link/$TOKEN")

echo "   Response: $VERIFY_RESPONSE"

# Check for session token
SESSION_TOKEN=$(echo $VERIFY_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$SESSION_TOKEN" ]; then
  echo "[ERROR] Verification failed or no session token returned"
  exit 1
else
  echo "[OK] Magic Link Verified! Session Token: $SESSION_TOKEN"
fi
