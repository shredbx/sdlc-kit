#!/usr/bin/env bash
# Verify that every local CDN asset is actually live on media.wanflo.com, and that nothing
# is duplicated back into static/.
#
#   make -f Makefile.d/workflows/check.mk run \
#        DIR=clients/wanflo/projects/wanflo/apps/web/svelte CMD=tools/cdn-verify.sh
#
# This replaced tools/cdn-stage.sh, which used to COPY static/ into cdn-upload/. That copy
# was the duplication: the same bytes lived in two folders and either could silently drift.
# The generators now write straight into cdn-upload/ (tools/cut-assets.sh, tools/og/shoot.mjs)
# and that folder is the single local home of everything the bucket serves.
#
# WORKFLOW: regenerate -> run this -> sync cdn-upload/ to the bucket root -> run this again.
set -uo pipefail
cd "$(dirname "$0")/.."

BASE=$(sed -n 's/^  base: "\(.*\)"$/\1/p' content/site.yml | head -1)
PATHS=$(sed -n '/^media:/,/^[a-z]/p' content/site.yml | sed -n 's/^  paths: \[\(.*\)\]/\1/p' | tr -d ' ')

echo "base:  ${BASE:-<empty — assets would be served locally>}"
echo "paths: $PATHS"
echo

# A file under static/ that also lives in cdn-upload/ is exactly the duplication this
# structure exists to prevent: two copies, one CDN URL, and no way to tell which is stale.
dupes=0
while IFS= read -r f; do
	rel=${f#cdn-upload/}
	if [ -e "static/$rel" ]; then echo "  DUPLICATE  static/$rel"; dupes=$((dupes + 1)); fi
done < <(find cdn-upload -type f ! -name README.txt)
[ "$dupes" -eq 0 ] && echo "no duplicates: nothing in cdn-upload/ also exists under static/"
echo

if [ -z "$BASE" ]; then
	echo "base is empty — skipping the live check"
	exit 0
fi

total=0
bad=0
while IFS= read -r f; do
	rel=${f#cdn-upload/}
	total=$((total + 1))
	code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 20 "$BASE/$rel")
	if [ "$code" != "200" ]; then
		echo "  $code  $BASE/$rel"
		bad=$((bad + 1))
	fi
done < <(find cdn-upload -type f ! -name README.txt | sort)

echo "$total files checked on $BASE — $bad not answering 200"
[ "$bad" -eq 0 ] || exit 1
