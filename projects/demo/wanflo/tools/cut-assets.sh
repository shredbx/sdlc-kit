#!/usr/bin/env bash
# One-off asset preparation. NOT part of the build.
#
# WHERE THE OUTPUT GOES, and why there are two places:
#   static/     brand marks, favicon, QR, fonts — served from the site's own origin because
#               they are in the first paint (see the media block in content/site.yml)
#   cdn-upload/ portfolio shots — served from media.wanflo.com. This folder IS the local
#               copy; nothing is duplicated under static/. Sync it to the bucket after a run.
#
# Kept in the repo so every crop is reproducible and auditable instead of re-guessed.
#
#   make -f Makefile.d/tools/swift.mk run \
#        DIR=clients/wanflo/projects/wanflo/apps/web/svelte CMD=tools/cut-assets.sh
#   (needs the owner's source files in ~/Downloads)
#
# Sources (owner-supplied, outside the repo):
#   ~/Downloads/wanflo/wanflo logo sheet.png   transparent brand sheet, 1536x1024
#   ~/Downloads/wanflo/WhatsApp QR code...png  WhatsApp business QR screenshot
#   ~/Downloads/screens/*.webp                 portfolio screenshots
#
# FORMAT — corrected 2026-08-24. An earlier pass asserted this machine had no WebP encoder
# and wrote every portfolio shot out as JPEG. That was wrong: cwebp 1.6.0 is installed
# (/usr/local/bin/cwebp), and the owner's sources were WebP to begin with. Portfolio shots
# are WebP now, and the MOBILE ones are the owner's own files copied BYTE-FOR-BYTE — they
# need no crop and no resize, so re-encoding them would only throw quality away.
# PNG stays for the brand marks (alpha) and for the favicon / apple-touch-icon, where broad
# client support matters more than a few KB.
set -euo pipefail
cd "$(dirname "$0")/.."

IMG="swift tools/img.swift"
# Screenshots are UI, not photographs: they are full of small text and hard edges, so the
# quality floor is higher than a photo needs and -sharp_yuv keeps coloured text from
# bleeding. -m 6 is the slowest/best compression search; this script runs by hand.
WEBP="cwebp -q 88 -sharp_yuv -m 6 -quiet"
SHEET="$HOME/Downloads/wanflo/wanflo logo sheet.png"
# The finished brand board, "SMART SYSTEMS" variant — the tagline the site actually uses.
# Its top panel is a FLAT navy (sampled srgb(5..7,19..21,45..48) everywhere) carrying the
# vertical lockup at full resolution, with no glow to trim and therefore no alpha fringing.
BOARD="$HOME/Downloads/wanflo/Wanflologo_Smart.png"
QRSRC="$HOME/Downloads/wanflo/WhatsApp QR code from Wan Flo Smart Systems (+66 98 375 7671).png"
SCR="$HOME/Downloads/screens"
mkdir -p static/brand cdn-upload/work/web cdn-upload/work/mobile

# ── BRAND ────────────────────────────────────────────────────────────────────
# The sheet is a light/dark PAIR once its transparency is honoured:
#   left  half = white/silver ink  -> for DARK backgrounds  (suffix -dark)
#   right half = navy ink          -> for LIGHT backgrounds (suffix -light)
echo "== brand =="
# Every piece is TRIMMED to its own artwork. Untrimmed, a CSS height would size the
# transparent padding instead of the logo, and each asset would render at a different
# visual size. Alpha floor 30 keeps the soft glow out of the bounding box.
cut() { $IMG crop "$SHEET" /tmp/wf-raw.png $2 $3 $4 $5 && $IMG trim /tmp/wf-raw.png "$1" - 30; }
cut static/brand/logo-mark-dark.png       100 745 320 220
cut static/brand/logo-mark-light.png      455 750 320 220
cut static/brand/logo-wordmark-dark.png    80 315 660 105
cut static/brand/logo-wordmark-light.png  830 315 680 105
# Full horizontal lockup INCLUDING the tagline — footer only, where it is big enough to read.
cut static/brand/logo-lockup-dark.png      60 555 720 165
cut static/brand/logo-lockup-light.png    820 555 700 165

# ── ICONS ────────────────────────────────────────────────────────────────────
# The sheet's two finished ROUNDED app-icon tiles. Cut for reference only — they are
# deliberately NOT committed under static/: both iOS (apple-touch-icon) and Android
# (maskable) want a FULL-BLEED square, which is what the pad step below produces. Shipping
# the rounded tiles as well put 112 KB of unreferenced PNG in the deploy mirror.
echo "== icons =="
$IMG crop "$SHEET" /tmp/wf-tile-dark.png  1150 700 280 280
$IMG trim /tmp/wf-tile-dark.png /tmp/wf-app-icon-dark.png - 200
$IMG crop "$SHEET" /tmp/wf-tile-light.png  850 700 270 280
$IMG trim /tmp/wf-tile-light.png /tmp/wf-app-icon-light.png - 200
# Favicon / touch icon: FULL-BLEED navy, never the rounded tile. iOS ignores alpha in an
# apple-touch-icon and fills transparent corners with black; Android maskable icons crop.
# Navy #061735 is sampled from the sheet's own dark tile.
for s in 32 48 180 192 512; do
  $IMG pad static/brand/logo-mark-dark.png "/tmp/wf-icon-$s.png" "$s" $((s * 22 / 100)) '#061735'
done
cp /tmp/wf-icon-32.png  static/favicon.png
cp /tmp/wf-icon-180.png static/apple-touch-icon.png
cp /tmp/wf-icon-192.png static/brand/icon-192.png
cp /tmp/wf-icon-512.png static/brand/icon-512.png

# ── OG LOCKUP ────────────────────────────────────────────────────────────────
# The share-card lockup, cut from the finished board rather than rebuilt from the
# transparent sheet. WHY: the sheet's artwork carries a soft glow, and trimming it on an
# alpha threshold kept semi-transparent pixels that composite to a dirty DARK HALO around
# the mark — clearly visible at 1:1 on the first OG images and the reason they were rejected.
# The board's top panel has the finished artwork already sitting on flat brand navy, so a
# plain rectangular crop is pixel-perfect and needs no keying at all.
#
# Ink bbox measured at 525x321+363+229. The crop takes a 120px margin on every side and
# then FEATHERS its alpha to nothing across that margin. The margin is not decoration: the
# board's navy carries a gentle vignette (sampled srgb 5..7,19..21,45..48 corner to centre),
# so a hard-edged crop laid on any single flat navy shows a faint rectangle — visible at 1:1
# on the second attempt at these images. Faded out, the crop has no edge to see on any
# background near this navy.
#
# The feather is fully opaque well before the ink starts: solid from 45px in, blurred over
# ~40px, and the artwork does not begin until 120px in.
echo "== og lockup =="
$IMG crop "$BOARD" /tmp/wf-og-lockup-raw.png 243 109 765 561
magick -size 765x561 xc:black -fill white -draw "rectangle 45,45 720,516" -blur 0x40 /tmp/wf-og-mask.png
magick /tmp/wf-og-lockup-raw.png /tmp/wf-og-mask.png -alpha off -compose CopyOpacity -composite \
       -strip -define png:compression-level=9 static/brand/og-lockup.png
magick identify -format "static/brand/og-lockup.png %wx%h (feathered)\n" static/brand/og-lockup.png

# ── WHATSAPP QR ──────────────────────────────────────────────────────────────
# Just the code plus its quiet zone. The surrounding card repeats the phone number and the
# account name, which the page already renders as real text.
echo "== qr =="
$IMG crop "$QRSRC" static/brand/qr-whatsapp.png 280 677 460 460

# ── PORTFOLIO — WEBSITES ─────────────────────────────────────────────────────
# The sources are 2560px browser captures (up to 859 KB each). The card never gives them
# more than ~600px, so they are resized once, straight from the WebP original — cwebp does
# the crop and the resize in a single decode, with no intermediate file and no second
# encode. -resize <w> 0 keeps the aspect ratio.
echo "== web shots =="
# Bestie ships with browser chrome that shows localhost:4003 AND the developer's private
# bookmarks bar. Cropped to the page content — see the review note for why this matters.
# cwebp applies -crop BEFORE -resize, which is the order this needs.
$WEBP -crop 60 162 1376 1284 -resize 1200 0 "$SCR/1. Bestie Real Estate Thailand.webp" -o cdn-upload/work/web/bestie-real-estate.webp
$WEBP -resize 1200 0 "$SCR/2.KiteCableThailand.webp"       -o cdn-upload/work/web/kitecable.webp
$WEBP -resize 1200 0 "$SCR/3. Best Stays Thailand.webp"    -o cdn-upload/work/web/bestays.webp
$WEBP -resize 1200 0 "$SCR/4. shredbx.com.webp"            -o cdn-upload/work/web/shredbx.webp
$WEBP -resize 1200 0 "$SCR/5. Design Studio Showcase.webp" -o cdn-upload/work/web/design-studio.webp

# ── PORTFOLIO — MOBILE ───────────────────────────────────────────────────────
# EVERY mobile project carries TWO shots (owner directive 2026-08-24): one that reads at
# thumbnail size and one that proves the app actually works.
#
# THESE ARE COPIES, NOT CONVERSIONS. Every source below is already WebP and already at or
# below the size the card renders it at, so there is nothing to crop and nothing to resize
# — `cp` is the correct operation and any re-encode would be pure loss. The card caps the
# rendered width at 100% of its own box, so a 1170px original just renders sharper on a
# retina screen; it is never upscaled and never blown out of the layout.
#
# A second source folder exists because the owner supplied the second shots later:
#   $SCR      first pass
#   $SCR/2    second pass — the missing partners, plus new 7Tree and Intuition captures
SCR2="$SCR/2"
echo "== mobile shots (verbatim copies of the owner's WebP originals) =="

# Intuition Challenge Game — REPLACED wholesale (owner directive). The previous pair was
# two BUILDS of the game; this pair is two SCREENS of the current build: onboarding first,
# a live round second.
cp "$SCR2/m1. Intuition Challenge Game - 1.webp" cdn-upload/work/mobile/intuition-welcome.webp
cp "$SCR2/m1. Intuition Challenge Game - 2.webp" cdn-upload/work/mobile/intuition-round.webp

# Bein — owner-supplied UNWATERMARKED pair (2026-08-24). The previous catalogue capture
# carried a tiled "shredbx" watermark baked into the source; bein-02 replaces it.
cp "$SCR2/bein-01.webp" cdn-upload/work/mobile/bein-item.webp
cp "$SCR2/bein-02.webp" cdn-upload/work/mobile/bein-catalogue.webp

# Velomesto — identity screen, then the working map.
cp "$SCR/m4. Velomesto.webp"   cdn-upload/work/mobile/velomesto-splash.webp
cp "$SCR/m4. Velomesto-2.webp" cdn-upload/work/mobile/velomesto-map.webp

# Le Paris Des Galleries — gallery detail leads (it is the only one of the pair with a
# photograph in it), the filtered listing follows as the proof.
cp "$SCR2/m5. Le Paris Des Galleries-2.webp" cdn-upload/work/mobile/le-paris-detail.webp
cp "$SCR/m5. Le Paris Des Galleries.webp"    cdn-upload/work/mobile/le-paris-list.webp

# iSmartHouse — identity screen, then room controls.
cp "$SCR/m6. iSmartHouse.webp"   cdn-upload/work/mobile/ismarthouse-splash.webp
cp "$SCR/m6. iSmartHouse-2.webp" cdn-upload/work/mobile/ismarthouse-control.webp

# 7Tree — the tree canvas IS the product, so it leads; the person profile proves the depth.
cp "$SCR2/7tree-1.webp" cdn-upload/work/mobile/7tree-canvas.webp
cp "$SCR2/7tree-2.webp" cdn-upload/work/mobile/7tree-person.webp

echo "== done =="
