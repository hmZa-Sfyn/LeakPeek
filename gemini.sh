#!/usr/bin/env bash
# gemini_test.sh
# Tests all Google API keys from geminiapi.txt against Gemini 1.5-flash
# One key per line, ignores empty lines and comments (#)

set -u
set -e

KEYS_FILE="geminiapi.txt"
MODEL="gemini-1.5-flash"
ENDPOINT="https://generativelanguage.googleapis.com/v1beta/models/${MODEL}:generateContent"

if [[ ! -f "${KEYS_FILE}" ]]; then
  echo "Error: File not found: ${KEYS_FILE}"
  exit 1
fi

echo "Testing keys from ${KEYS_FILE}..."
echo "Model: ${MODEL}"
echo "───────────────────────────────────────────────"
echo ""

count=0
success_count=0

while IFS= read -r line || [[ -n "${line}" ]]; do
  # Skip empty lines and comments
  line=$(echo "${line}" | xargs)  # trim whitespace
  [[ -z "${line}" ]] && continue
  [[ "${line}" =~ ^# ]] && continue

  ((count++))

  key="${line}"
  display_key="${key:0:6}...${key:(-4)}"

  echo -n "Key ${count} (${display_key}) → "

  response=$(curl -s -m 10 \
    -H "Content-Type: application/json" \
    -d '{"contents":[{"parts":[{"text":"hi"}]}]}' \
    "${ENDPOINT}?key=${key}" 2>/dev/null || true)

  # Check common status patterns
  if echo "${response}" | grep -q '"candidates"'; then
    echo "OK (valid key - got candidates)"
    ((success_count++))
    # Show the actual reply if present
    reply=$(echo "${response}" | grep -o '"text": *"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
    [[ -n "${reply}" ]] && echo "    Reply: ${reply}"
  elif echo "${response}" | grep -qi "invalid" || echo "${response}" | grep -qi "not valid"; then
    echo "FAIL - Invalid / not authorized"
  elif echo "${response}" | grep -qi "quota" || echo "${response}" | grep -qi "limit"; then
    echo "FAIL - Quota exceeded or rate limited"
  elif echo "${response}" | grep -qi "permission" || echo "${response}" | grep -qi "403"; then
    echo "FAIL - Permission denied (likely API not enabled or restricted)"
  elif echo "${response}" | grep -qi "404"; then
    echo "FAIL - Endpoint/model not found"
  else
    # Show beginning of raw error for debugging
    short_err=$(echo "${response}" | head -c 180 | tr -d '\n' | sed 's/"/\"/g')
    [[ -z "${short_err}" ]] && short_err="No response / timeout / curl error"
    echo "FAIL - ${short_err}"
  fi

  # Tiny delay to be polite to the API
  sleep 0.4

done < "${KEYS_FILE}"

echo ""
echo "───────────────────────────────────────────────"
echo "Finished. Tested ${count} keys. Working: ${success_count}"

if (( success_count > 0 )); then
  echo ""
  echo "WARNING ───────────────────────────────────────"
  echo "Any key that worked is almost certainly already leaked and monitored."
  echo "Do NOT use it for anything real — it can get revoked any moment or rack up charges."
  echo "→ Get your own free key here: https://aistudio.google.com/app/apikey"
fi
