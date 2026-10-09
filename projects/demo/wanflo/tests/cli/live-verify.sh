#!/usr/bin/env bash
# SC12 — live wanflo.com verification (task 2607-076). RED until Andrei's Vercel hookup + DNS.
# Owner-assisted final gate; screenshots are captured separately (Playwright MCP, dual theme).
set -u
BASE="${1:-https://wanflo.com}"
fail=0
for path in "/" "/ru" "/sitemap.xml" "/robots.txt"; do
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$BASE$path")
  if [ "$code" = "200" ]; then echo "PASS SC12: $path -> 200"; else echo "FAIL SC12: $path -> $code"; fail=1; fi
done
curl -s --max-time 15 "$BASE/sitemap.xml" | grep -q "hreflang" && echo "PASS SC12: sitemap carries hreflang alternates" || { echo "FAIL SC12: no hreflang in sitemap"; fail=1; }
exit $fail
