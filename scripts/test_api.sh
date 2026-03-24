#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"
PASS=0
FAIL=0

# ─── Unique per run ──────────────────────────────────────
TS=$(date +%s)
EMAIL="testuser_${TS}@test.com"
EMAIL2="other_${TS}@test.com"

# ─── Track created resources ─────────────────────────────
CREATED_BOT_IDS=()
USER1_TOKEN=""
USER2_TOKEN=""

# ─── Colors ──────────────────────────────────────────────
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ─── Cleanup ─────────────────────────────────────────────
cleanup() {
  echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo -e "  Cleanup"
  echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

  # Hapus semua bot yang dibuat selama test
  for bot_id in "${CREATED_BOT_IDS[@]}"; do
    RESP=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
      "$BASE_URL/bots/$bot_id" \
      -H "Authorization: Bearer $USER1_TOKEN")
    if [ "$RESP" -eq 200 ] || [ "$RESP" -eq 404 ]; then
      echo -e "  ${GREEN}✓${NC} Bot $bot_id deleted"
    else
      echo -e "  ${YELLOW}~${NC} Bot $bot_id → HTTP $RESP (mungkin sudah terhapus)"
    fi
  done

  # Hapus test users langsung via DB
  echo -e "  Cleaning up test users from DB..."
  PGPASSWORD="${DB_PASSWORD:-postgres}" psql \
    -h "${DB_HOST:-localhost}" \
    -p "${DB_PORT:-5432}" \
    -U "${DB_USER:-neknowbot}" \
    -d "${DB_NAME:-neknow_bot_db}" \
    -c "DELETE FROM users WHERE email IN ('$EMAIL', '$EMAIL2');" \
    > /dev/null 2>&1

  if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓${NC} Test users deleted ($EMAIL, $EMAIL2)"
  else
    echo -e "  ${YELLOW}~${NC} DB cleanup skipped (cek DB_* env vars di bawah)"
    echo -e "  Manual cleanup:"
    echo -e "  DELETE FROM users WHERE email LIKE '%_${TS}@test.com';"
  fi

  echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Jalankan cleanup di akhir atau saat interrupt
trap cleanup EXIT

# ─── Helpers ─────────────────────────────────────────────
print_section() {
  echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo -e "  $1"
  echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

assert_status() {
  local label=$1
  local expected=$2
  local actual=$3
  local body=$4

  if [ "$actual" -eq "$expected" ]; then
    echo -e "${GREEN}✓ PASS${NC} [$label] → HTTP $actual"
    PASS=$((PASS + 1))
  else
    echo -e "${RED}✗ FAIL${NC} [$label] → expected HTTP $expected, got HTTP $actual"
    echo -e "  Response: $body"
    FAIL=$((FAIL + 1))
  fi
}

call() {
  local method=$1
  local endpoint=$2
  local data=$3
  local token=$4

  HEADERS=(-H "Content-Type: application/json")
  if [ -n "$token" ]; then
    HEADERS+=(-H "Authorization: Bearer $token")
  fi

  if [ -n "$data" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
      "${HEADERS[@]}" -d "$data")
  else
    RESPONSE=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
      "${HEADERS[@]}")
  fi

  BODY=$(echo "$RESPONSE" | head -n -1)
  STATUS=$(echo "$RESPONSE" | tail -n 1)
}

# ════════════════════════════════════════
print_section "AUTH — Register"
# ════════════════════════════════════════

call POST "/auth/register" "{\"email\":\"$EMAIL\",\"password\":\"password123\"}"
assert_status "Register valid user" 201 "$STATUS" "$BODY"

call POST "/auth/register" "{\"email\":\"$EMAIL\",\"password\":\"password123\"}"
assert_status "Register duplicate email" 409 "$STATUS" "$BODY"

call POST "/auth/register" '{"email":"bukan-email","password":"password123"}'
assert_status "Register invalid email format" 400 "$STATUS" "$BODY"

call POST "/auth/register" "{\"email\":\"another_${TS}@test.com\",\"password\":\"\"}"
assert_status "Register empty password" 400 "$STATUS" "$BODY"

call POST "/auth/register" '{}'
assert_status "Register empty body" 400 "$STATUS" "$BODY"

# ════════════════════════════════════════
print_section "AUTH — Login"
# ════════════════════════════════════════

call POST "/auth/login" "{\"email\":\"$EMAIL\",\"password\":\"password123\"}"
assert_status "Login valid credentials" 200 "$STATUS" "$BODY"

USER1_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$BODY" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$USER1_TOKEN" ]; then
  echo -e "${RED}✗ FATAL${NC} Tidak bisa ambil access_token, stop testing."
  exit 1
fi
echo -e "  ${GREEN}Token diperoleh ✓${NC}"

call POST "/auth/login" "{\"email\":\"$EMAIL\",\"password\":\"wrongpassword\"}"
assert_status "Login wrong password" 401 "$STATUS" "$BODY"

call POST "/auth/login" '{"email":"notexist@test.com","password":"password123"}'
assert_status "Login unregistered email" 401 "$STATUS" "$BODY"

call POST "/auth/login" '{}'
assert_status "Login empty body" 400 "$STATUS" "$BODY"

# ════════════════════════════════════════
print_section "AUTH — Refresh Token"
# ════════════════════════════════════════

call POST "/auth/refresh" "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
assert_status "Refresh with valid token" 200 "$STATUS" "$BODY"

call POST "/auth/refresh" '{"refresh_token":"invalid.token.here"}'
assert_status "Refresh with invalid token" 401 "$STATUS" "$BODY"

call POST "/auth/refresh" '{}'
assert_status "Refresh with empty body" 400 "$STATUS" "$BODY"

# ════════════════════════════════════════
print_section "AUTH — Me"
# ════════════════════════════════════════

call GET "/me" "" "$USER1_TOKEN"
assert_status "Get me with valid token" 200 "$STATUS" "$BODY"

call GET "/me" "" "invalidtoken123"
assert_status "Get me with invalid token" 401 "$STATUS" "$BODY"

call GET "/me" "" ""
assert_status "Get me without token" 401 "$STATUS" "$BODY"

# ════════════════════════════════════════
print_section "BOT — Create"
# ════════════════════════════════════════

call POST "/bots/" "{
  \"name\": \"Test Bot\",
  \"description\": \"Bot untuk testing\",
  \"system_prompt\": \"Kamu adalah asisten yang membantu menjawab pertanyaan dengan baik.\",
  \"model\": \"gpt-4o-mini\",
  \"is_public\": false
}" "$USER1_TOKEN"
assert_status "Create bot valid" 201 "$STATUS" "$BODY"

BOT_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$BOT_ID" ]; then
  CREATED_BOT_IDS+=("$BOT_ID")  # ← track untuk cleanup
  echo -e "  ${GREEN}Bot ID: $BOT_ID ✓${NC}"
fi

call POST "/bots/" "{
  \"name\": \"Test Bot\",
  \"system_prompt\": \"Kamu adalah asisten yang membantu menjawab pertanyaan.\",
  \"model\": \"gpt-99-ultra\",
  \"is_public\": false
}" "$USER1_TOKEN"
assert_status "Create bot invalid model" 400 "$STATUS" "$BODY"

call POST "/bots/" "{
  \"name\": \"Test Bot\",
  \"system_prompt\": \"\",
  \"model\": \"gpt-4o-mini\"
}" "$USER1_TOKEN"
assert_status "Create bot empty system_prompt" 400 "$STATUS" "$BODY"

call POST "/bots/" "{
  \"name\": \"Test Bot\",
  \"system_prompt\": \"Hi\",
  \"model\": \"gpt-4o-mini\"
}" "$USER1_TOKEN"
assert_status "Create bot short system_prompt" 400 "$STATUS" "$BODY"

call POST "/bots/" "{
  \"name\": \"Test Bot\",
  \"system_prompt\": \"Kamu adalah asisten yang membantu menjawab pertanyaan.\",
  \"model\": \"gpt-4o-mini\"
}" ""
assert_status "Create bot without token" 401 "$STATUS" "$BODY"

# ════════════════════════════════════════
print_section "BOT — Get All"
# ════════════════════════════════════════

call GET "/bots/" "" "$USER1_TOKEN"
assert_status "Get all bots" 200 "$STATUS" "$BODY"

call GET "/bots/" "" ""
assert_status "Get all bots without token" 401 "$STATUS" "$BODY"

# ════════════════════════════════════════
print_section "BOT — Get By ID"
# ════════════════════════════════════════

if [ -n "$BOT_ID" ]; then
  call GET "/bots/$BOT_ID" "" "$USER1_TOKEN"
  assert_status "Get bot by valid ID" 200 "$STATUS" "$BODY"
fi

call GET "/bots/00000000-0000-0000-0000-000000000000" "" "$USER1_TOKEN"
assert_status "Get bot by non-existent ID" 404 "$STATUS" "$BODY"

if [ -n "$BOT_ID" ]; then
  call GET "/bots/$BOT_ID" "" ""
  assert_status "Get bot without token" 401 "$STATUS" "$BODY"
fi

# ════════════════════════════════════════
print_section "BOT — Update"
# ════════════════════════════════════════

if [ -n "$BOT_ID" ]; then
  call PUT "/bots/$BOT_ID" '{"name": "Updated Bot Name"}' "$USER1_TOKEN"
  assert_status "Update bot name only" 200 "$STATUS" "$BODY"

  call PUT "/bots/$BOT_ID" '{"model": "gpt-4o"}' "$USER1_TOKEN"
  assert_status "Update bot model valid" 200 "$STATUS" "$BODY"

  call PUT "/bots/$BOT_ID" '{"model": "gpt-99-ultra"}' "$USER1_TOKEN"
  assert_status "Update bot model invalid" 400 "$STATUS" "$BODY"

  call PUT "/bots/$BOT_ID" '{"status": "inactive"}' "$USER1_TOKEN"
  assert_status "Update bot status inactive" 200 "$STATUS" "$BODY"

  call PUT "/bots/$BOT_ID" '{"status": "banned"}' "$USER1_TOKEN"
  assert_status "Update bot status invalid" 400 "$STATUS" "$BODY"

  call PUT "/bots/$BOT_ID" '{"name": "Hacked"}' ""
  assert_status "Update bot without token" 401 "$STATUS" "$BODY"

  call PUT "/bots/00000000-0000-0000-0000-000000000000" '{"name": "Ghost"}' "$USER1_TOKEN"
  assert_status "Update non-existent bot" 404 "$STATUS" "$BODY"
fi

# ════════════════════════════════════════
print_section "BOT — Ownership (Second User)"
# ════════════════════════════════════════

call POST "/auth/register" "{\"email\":\"$EMAIL2\",\"password\":\"password123\"}"
assert_status "Register second user" 201 "$STATUS" "$BODY"

call POST "/auth/login" "{\"email\":\"$EMAIL2\",\"password\":\"password123\"}"
assert_status "Login second user" 200 "$STATUS" "$BODY"
USER2_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$BOT_ID" ] && [ -n "$USER2_TOKEN" ]; then
  call GET "/bots/$BOT_ID" "" "$USER2_TOKEN"
  assert_status "Get private bot owned by other user" 401 "$STATUS" "$BODY"

  call PUT "/bots/$BOT_ID" '{"name": "Stolen"}' "$USER2_TOKEN"
  assert_status "Update bot owned by other user" 401 "$STATUS" "$BODY"

  call DELETE "/bots/$BOT_ID" "" "$USER2_TOKEN"
  assert_status "Delete bot owned by other user" 401 "$STATUS" "$BODY"
fi

# ════════════════════════════════════════
print_section "BOT — Delete"
# ════════════════════════════════════════

if [ -n "$BOT_ID" ]; then
  call DELETE "/bots/00000000-0000-0000-0000-000000000000" "" "$USER1_TOKEN"
  assert_status "Delete non-existent bot" 404 "$STATUS" "$BODY"

  call DELETE "/bots/$BOT_ID" "" ""
  assert_status "Delete bot without token" 401 "$STATUS" "$BODY"

  call DELETE "/bots/$BOT_ID" "" "$USER1_TOKEN"
  assert_status "Delete bot valid" 200 "$STATUS" "$BODY"
  # Hapus dari tracking karena sudah terhapus
  CREATED_BOT_IDS=("${CREATED_BOT_IDS[@]/$BOT_ID}")

  call DELETE "/bots/$BOT_ID" "" "$USER1_TOKEN"
  assert_status "Delete already deleted bot" 404 "$STATUS" "$BODY"
fi

# ════════════════════════════════════════
print_section "Summary"
# ════════════════════════════════════════

TOTAL=$((PASS + FAIL))
echo -e "\n  Total  : $TOTAL"
echo -e "  ${GREEN}Pass   : $PASS${NC}"
echo -e "  ${RED}Fail   : $FAIL${NC}\n"

if [ "$FAIL" -eq 0 ]; then
  echo -e "${GREEN}  ✓ All tests passed!${NC}\n"
else
  echo -e "${RED}  ✗ $FAIL test(s) failed.${NC}\n"
  exit 1
fi
