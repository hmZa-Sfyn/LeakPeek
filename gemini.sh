#!/usr/bin/env bash
# =============================================================================
# gemini_test.sh   —   Multi-service Google API Key Tester (2026 edition)
# Tests leaked AIzaSy... keys against Gemini + other common Google services
# Reads keys from geminiapi.txt (one per line, skips # comments & empty lines)
#
# Services tested (stops on first success per key):
#   1. Gemini 1.5 Flash
#   2. Google Maps Static
#   3. YouTube Data v3
#   4. Google Drive v3
#   5. Custom Search JSON
# =============================================================================

set -u

KEYS_FILE="geminiapi.txt"
DELAY_BETWEEN_KEYS=1.2     # seconds — be nice to Google
DELAY_BETWEEN_TESTS=0.5

# Colors (works in most terminals)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ────────────────────────────────────────────────────────────────
# Helper: short key display
short_key() {
    local k="$1"
    if [ ${#k} -gt 12 ]; then
        echo "${k:0:6}...${k: -4}"
    else
        echo "$k"
    fi
}

# ────────────────────────────────────────────────────────────────
echo -e "${YELLOW}Google API Key Multi-Tester${NC}"
echo "Reading from: $KEYS_FILE"
echo ""

if [ ! -f "$KEYS_FILE" ]; then
    echo -e "${RED}Error: File not found → $KEYS_FILE${NC}"
    exit 1
fi

# Load keys (skip comments & empty lines)
mapfile -t KEYS < <(grep -vE '^\s*(#|$)' "$KEYS_FILE" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

if [ ${#KEYS[@]} -eq 0 ]; then
    echo -e "${RED}No keys found in the file.${NC}"
    exit 1
fi

unique_count=$(printf '%s\n' "${KEYS[@]}" | sort -u | wc -l)
echo "Found ${#KEYS[@]} lines → ${YELLOW}$unique_count unique keys${NC}"
echo "Services order: Gemini → Maps → YouTube → Drive → CustomSearch"
echo "───────────────────────────────────────────────────────────────"
echo ""

declare -A RESULTS

for ((i=0; i<${#KEYS[@]}; i++)); do
    key="${KEYS[i]}"
    sk=$(short_key "$key")
    printf "Key %2d/%d  %-14s " "$((i+1))" "${#KEYS[@]}" "$sk"

    found=false

    # 1. Gemini
    curl -s -m 9 -H 'Content-Type: application/json' \
        -d '{"contents":[{"parts":[{"text":"hi"}]}]}' \
        "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=$key" > /tmp/gemini_resp 2>/dev/null

    if grep -q '"candidates"' /tmp/gemini_resp; then
        echo -e "${GREEN}OK Gemini${NC}"
        RESULTS["$key"]="Gemini"
        found=true
    fi

    if ! $found; then
        # 2. Maps Static
        http_code=$(curl -s -m 6 -o /dev/null -w "%{http_code}" \
            "https://maps.googleapis.com/maps/api/staticmap?center=0,0&zoom=1&size=1x1&key=$key")

        if [ "$http_code" = "200" ]; then
            echo -e "${GREEN}OK Maps${NC}"
            RESULTS["$key"]="Maps"
            found=true
        fi
    fi

    if ! $found; then
        # 3. YouTube
        yt_resp=$(curl -s -m 6 \
            "https://www.googleapis.com/youtube/v3/search?part=snippet&maxResults=1&q=test&key=$key")

        if echo "$yt_resp" | grep -q '"items"'; then
            echo -e "${GREEN}OK YouTube${NC}"
            RESULTS["$key"]="YouTube"
            found=true
        fi
    fi

    if ! $found; then
        # 4. Drive
        drive_code=$(curl -s -m 6 -o /dev/null -w "%{http_code}" \
            "https://www.googleapis.com/drive/v3/about?fields=user&key=$key")

        if [ "$drive_code" = "200" ]; then
            echo -e "${GREEN}OK Drive${NC}"
            RESULTS["$key"]="Drive"
            found=true
        fi
    fi

    if ! $found; then
        # 5. Custom Search (dummy CX — will usually 400/403 unless configured)
        cs_resp=$(curl -s -m 6 \
            "https://www.googleapis.com/customsearch/v1?cx=0123456789:qiwtest123&q=test&key=$key")

        if echo "$cs_resp" | grep -q '"searchInformation"'; then
            echo -e "${GREEN}OK CustomSearch${NC}"
            RESULTS["$key"]="CustomSearch"
            found=true
        fi
    fi

    if ! $found; then
        echo -e "${RED}ALL FAILED${NC}"
        RESULTS["$key"]="None"
    fi

    sleep "$DELAY_BETWEEN_KEYS"
done

echo ""
echo "───────────────────────────────────────────────────────────────"
echo -e "${YELLOW}SUMMARY${NC}"

working=0
for key in "${!RESULTS[@]}"; do
    sk=$(short_key "$key")
    res="${RESULTS[$key]}"
    if [ "$res" != "None" ]; then
        echo -e "  $sk → ${GREEN}$res${NC}"
        ((working++))
    else
        echo -e "  $sk → ${RED}None${NC}"
    fi
done

if [ $working -eq 0 ]; then
    echo -e "  ${RED}No working keys found on any service.${NC}"
else
    echo ""
    echo -e "${YELLOW}WARNING — IMPORTANT (March 2026)${NC}"
    echo "  Any key that responded is almost certainly leaked and monitored."
    echo "  Google aggressively revokes abused/leaked Gemini keys."
    echo "  Using them risks instant quota burn, billing spikes, or permanent ban."
    echo "  → Generate your own fresh key here:"
    echo "    https://aistudio.google.com/app/apikey"
fi

echo ""
echo "Done."
rm -f /tmp/gemini_resp 2>/dev/null