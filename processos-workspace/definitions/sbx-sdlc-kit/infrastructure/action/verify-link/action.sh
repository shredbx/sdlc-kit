#!/usr/bin/env bash
# The link check: the API is healthy on its published port, and the web page, fetched from its
# published port, says so. A web that is up but cannot reach the API fails here, which is the
# difference between "running" and "linked".
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

dir="$(bundle_dir)"
api_port="$(published_port "$dir" bos-api)"
web_port="$(published_port "$dir" bos-web)"

# poll <what> <seconds> <command...>: run the command once a second until it succeeds.
poll() {
  local what="$1" seconds="$2" i
  shift 2
  for ((i = 0; i < seconds; i++)); do
    if "$@"; then return 0; fi
    sleep 1
  done
  echo "$what: not ready after ${seconds}s" >&2
  return 1
}

api_healthy() {
  curl -fsS --max-time 3 "http://127.0.0.1:$api_port/health" 2>/dev/null | jq -e '.status == "healthy"' >/dev/null 2>&1
}

poll "the API on localhost:$api_port (/health)" 90 api_healthy
api="$(curl -fsS --max-time 3 "http://127.0.0.1:$api_port/health" | jq -r '"\(.service) \(.version), database \(.database)"')"
echo "API healthy on :$api_port ($api)"

page=""
web_says_healthy() {
  page="$(curl -fsS --max-time 5 "http://127.0.0.1:$web_port/" 2>/dev/null)" && grep -q 'API: healthy' <<<"$page"
}

if ! poll "the web page on localhost:$web_port" 60 web_says_healthy; then
  said="$(grep -oE 'API: [a-z]+' <<<"$page" | head -1 || true)"
  echo "the API is healthy, but the web page does not say so (${said:-no API status on the page}): the two halves are not linked" >&2
  exit 1
fi
echo "web page on :$web_port says 'API: healthy': the halves are linked"
