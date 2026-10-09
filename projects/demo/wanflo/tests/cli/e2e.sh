#!/usr/bin/env bash
# SC8/SC9/SC10 + data specs — runs the app's OWN @playwright/test (never pnpm dlx: a second
# downloaded runner conflicts with the local one — proven 2026-07-14). WEBKIT=1 adds the
# webkit project (workspace #882 gate convention). Pass extra playwright args through.
set -u
exec pnpm --filter @shredbx/wanflo-web run test:e2e "$@"
