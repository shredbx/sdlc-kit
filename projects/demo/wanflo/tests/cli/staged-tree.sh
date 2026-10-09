#!/usr/bin/env bash
# SC11 — mirror staged-tree assertions (task 2607-076). RED until WP6 (post-merge, from MAIN).
# Runs the sync dry-run and asserts must-contain / must-NOT-contain rules (fx-staged-tree-rules).
set -u
OUT=$(sbx external sync wanflo --dry-run 2>&1) || { echo "FAIL SC11: sync --dry-run errored"; printf '%s\n' "$OUT" | head -5; exit 1; }
fail=0
must_have=("apps/web/svelte/package.json" "apps/web/svelte/vercel.json" "apps/web/svelte/src")
must_not=("apps/web/svelte/tests" ".svelte-kit" "CLAUDE.md" ".env")

for p in "${must_have[@]}"; do
  printf '%s' "$OUT" | grep -q "$p" && echo "PASS SC11.have: $p staged" || { echo "FAIL SC11.have: $p NOT in staged tree"; fail=1; }
done
for p in "${must_not[@]}"; do
  printf '%s' "$OUT" | grep -q "$p" && { echo "FAIL SC11.block: $p leaked into staged tree"; fail=1; } || echo "PASS SC11.block: $p excluded"
done
printf '%s' "$OUT" | grep -q "op://" && { echo "FAIL SC11.scrub: op:// ref visible in dry-run"; fail=1; } || echo "PASS SC11.scrub: no op:// refs"

exit $fail
