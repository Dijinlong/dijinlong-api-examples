#!/usr/bin/env bash
#
# dijinlong-api-examples / curl
# -----------------------------
# Four things you will actually need, in the order you need them:
#
#   1. list models
#   2. a non-streaming chat completion
#   3. a streaming chat completion
#   4. a failed request, handled properly
#
# Usage:
#     export API_KEY="sk-..."
#     bash curl/examples.sh
#
# Optional:
#     export BASE_URL="https://api.dijinlong.com/v1"
#     export MODEL="deepseek-v4-flash"

set -euo pipefail

: "${API_KEY:?set API_KEY first (export API_KEY=sk-...)}"
BASE="${BASE_URL:-https://api.dijinlong.com/v1}"
MODEL="${MODEL:-deepseek-v4-flash}"

hr() { printf '\n=== %s ===\n' "$1"; }

# ---------------------------------------------------------------------------
hr "1. list models"
curl -sS "$BASE/models" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Accept: application/json" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('\n'.join(' - '+m['id'] for m in d.get('data',[])[:20]))" 2>/dev/null \
  || echo "(could not parse the model list)"

# ---------------------------------------------------------------------------
hr "2. chat completion (non-streaming)"
curl -sS "$BASE/chat/completions" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
        \"model\": \"$MODEL\",
        \"messages\": [{\"role\": \"user\", \"content\": \"Say hello in exactly three words.\"}]
      }" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['choices'][0]['message']['content'])" 2>/dev/null \
  || echo "(could not parse the response)"

# ---------------------------------------------------------------------------
hr "3. chat completion (streaming)"
curl -sS -N "$BASE/chat/completions" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "{
        \"model\": \"$MODEL\",
        \"stream\": true,
        \"messages\": [{\"role\": \"user\", \"content\": \"Count from one to five.\"}]
      }" \
  | while IFS= read -r line; do
      case "$line" in
        data:\ *)
          payload="${line#data: }"
          [ "$payload" = "[DONE]" ] && break
          printf '%s' "$payload" \
            | python3 -c "import sys,json;d=json.load(sys.stdin);c=(d.get('choices') or [{}])[0].get('delta',{}).get('content');sys.stdout.write(c or '')" 2>/dev/null || true
          ;;
      esac
    done
printf '\n'

# ---------------------------------------------------------------------------
hr "4. handling an error properly"
# Deliberately ask for a model that does not exist.
resp=$(curl -sS -w '\n%{http_code}' "$BASE/chat/completions" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "definitely-not-a-real-model", "messages": [{"role":"user","content":"hi"}]}')

code=$(printf '%s' "$resp" | tail -n1)
body=$(printf '%s' "$resp" | sed '$d')

echo "HTTP status: $code"
echo "Body: $body"
echo
if [ "$code" = "200" ]; then
  echo "NOTE: the endpoint answered 200 for a bogus model. Check whether it silently"
  echo "      substitutes a default model -- that is a different bug worth reporting."
else
  echo "Good: the error surfaced as a non-200 status rather than a 200 with an error inside."
  echo "Check that the error body is JSON with a human-readable message, not an HTML page."
fi
