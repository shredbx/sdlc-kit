This folder IS the local copy of everything https://media.wanflo.com serves.

There is deliberately no second copy under static/ — the generators write straight in here:
  tools/cut-assets.sh    portfolio screenshots -> work/
  tools/og/shoot.mjs     share images          -> og/
  icons/tech/            brand + framework marks, added by hand

SYNC: upload the CONTENTS of this folder to the ROOT of the R2 bucket `wanflo-homepage`.

  cdn-upload/work/...   ->  https://media.wanflo.com/work/...
  cdn-upload/og/...     ->  https://media.wanflo.com/og/...
  cdn-upload/icons/...  ->  https://media.wanflo.com/icons/...

Do NOT upload the folder itself as a directory called "cdn-upload", and do not upload this
README. Afterwards run tools/cdn-verify.sh — it checks every file answers 200 and that
nothing has been duplicated back into static/.

CACHE: the bucket serves these with cache-control max-age=14400 (4 hours) and filenames are
stable, so REPLACING an image means purging that path in Cloudflare — or renaming the file
and updating content/work.yml.
