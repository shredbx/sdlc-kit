#!/usr/bin/env bash
# SC3 — scaffold build gates (task 2607-076). RED until WP2 lands the app.
# Runs the governed build path (never raw pnpm): sbx build + the app's check script via pnpm.mk.
set -u
APP_DIR="clients/wanflo/projects/wanflo/apps/web/svelte"
fail=0

[ -f "$APP_DIR/package.json" ] || { echo "FAIL SC3: $APP_DIR/package.json missing (scaffold absent)"; exit 1; }
[ -f "$APP_DIR/vercel.json" ]  || { echo "FAIL SC3: vercel.json missing"; fail=1; }
grep -q "adapter-vercel" "$APP_DIR/svelte.config.js" 2>/dev/null || { echo "FAIL SC3: svelte.config.js not on adapter-vercel"; fail=1; }

if sbx build --dir "$APP_DIR"; then
  echo "PASS SC3.build: sbx build green"
else
  echo "FAIL SC3.build: sbx build failed"; fail=1
fi

# svelte-check 0 errors / 0 warnings (via the app's declared check script — pnpm.mk has no
# check/run target yet; gap reported to the owner 2026-07-14)
if pnpm --filter @shredbx/wanflo-web run check; then
  echo "PASS SC3.check: svelte-check clean"
else
  echo "FAIL SC3.check: svelte-check reported problems"; fail=1
fi

exit $fail
