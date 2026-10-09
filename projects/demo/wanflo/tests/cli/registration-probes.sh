#!/usr/bin/env bash
# SC1 + SC2 — registration truth + the D-9 anti-accident deploy gate (task 2607-076).
# RED until WP1 registration lands (SC1) and the branch is merged to main (SC2 full wording —
# the deploy pipeline enforces clean-tree + main-branch guards before project resolution).
# Usage: bash registration-probes.sh            (SC1 only, any branch)
#        bash registration-probes.sh --with-gate (SC1 + SC2 refusal wording; run from MAIN)
set -u
fail=0
say() { printf '%s\n' "$*"; }

# --- SC1: registration visible to all sbx tooling -------------------------------------------
if sbx application list 2>/dev/null | grep -q "wanflo"; then
  say "PASS SC1.app: application list has wanflo row"
else
  say "FAIL SC1.app: no wanflo row in sbx application list"; fail=1
fi

if sbx external list 2>/dev/null | grep -q "wanflo \["; then
  say "PASS SC1.ext: external registry has wanflo"
else
  say "FAIL SC1.ext: no wanflo in sbx external list"; fail=1
fi

if sbx external validate 2>&1 | grep -qi "wanflo.*\(error\|invalid\)"; then
  say "FAIL SC1.val: sbx external validate reports wanflo errors"; fail=1
else
  say "PASS SC1.val: external validate has no wanflo errors"
fi

if sbx ports list --consumer wanflo-web 2>/dev/null | grep -q "wanflo-web"; then
  say "PASS SC1.port: wanflo-web port allocated"
else
  say "FAIL SC1.port: no wanflo-web allocation"; fail=1
fi

# --- SC2: gate refusals (full wording needs MAIN — deploy guards fire first elsewhere) -------
if [ "${1:-}" = "--with-gate" ]; then
  out_prod=$(sbx deploy wanflo --env production --dry-run 2>&1)
  if printf '%s' "$out_prod" | grep -q 'server "vercel"'; then
    say "PASS SC2.prod: refused citing unregistered server \"vercel\""
  else
    say "FAIL SC2.prod: expected refusal citing server \"vercel\", got: ${out_prod%%$'\n'*}"; fail=1
  fi

  out_test=$(sbx deploy wanflo --env test --dry-run 2>&1)
  if printf '%s' "$out_test" | grep -q "not declared"; then
    say "PASS SC2.env: undeclared env refused listing declared rows"
  else
    say "FAIL SC2.env: expected 'not declared' refusal, got: ${out_test%%$'\n'*}"; fail=1
  fi

  out_flag=$(sbx deploy wanflo --env production --server contabo-andrei --dry-run 2>&1)
  if printf '%s' "$out_flag" | grep -q "conflicts with environments"; then
    say "PASS SC2.flag: --server conflict refused (row is source of truth)"
  else
    say "FAIL SC2.flag: expected flag-conflict refusal, got: ${out_flag%%$'\n'*}"; fail=1
  fi
fi

exit $fail
