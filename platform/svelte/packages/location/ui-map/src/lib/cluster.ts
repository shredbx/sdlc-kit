// Pixel-collision declustering (task 2607-133) — PURE + unit-tested. Given the SCREEN
// positions of the source markers at the current zoom, merge any that overlap into a single
// cluster so no two markers ever cover each other. This is the safety net that makes the board
// read as cleanly as Google Maps at every zoom: as you zoom in, points move apart in pixel
// space, so clusters naturally dissolve back into individual markers with no extra bookkeeping.
//
// It is deliberately screen-space, not geographic: two markers 3 km apart overlap at country
// zoom but not at street zoom, and only the projection knows which. The adapter projects each
// source point to pixels, calls this, and renders the result.
//
// Collision is judged on each marker's RECTANGLE when its size is known (a price pill is far
// wider than tall, so a center-distance circle either merges pills that don't touch or lets
// wide pills overlap — the exact bug the first pass shipped) and falls back to a
// center-distance circle when it isn't.
//
// The algorithm is a deterministic single-pass greedy merge (stable input order → stable
// output), which is more than adequate for a boutique catalog's handful of areas. If the point
// count ever reaches the thousands, swap this one function for supercluster's KD-tree behind the
// same signature — nothing else changes.

/** A source marker already projected to screen pixels at the current zoom. */
export interface PixelPoint {
	key: string;
	x: number;
	y: number;
	/** How many listings this source marker already stands for (an area group's size, ≥1). */
	count: number;
	/** Rendered glyph size in px. When BOTH points in a pair carry w+h, collision is judged
	 *  rectangle-vs-rectangle (plus the gap); otherwise by the radius circle. */
	w?: number;
	h?: number;
}

/** One rendered item: a lone point (memberKeys = [its key]) or a merged cluster (many keys). */
export interface ClusterCell {
	/** Stable render key — the member key when alone, else a `grp:`-prefixed join of members. */
	key: string;
	memberKeys: string[];
	/** Total listings represented (sum of member counts). */
	count: number;
	/** Pixel centroid of the members (mean) — the anchor the adapter unprojects back to lng/lat. */
	x: number;
	y: number;
}

/** Minimum clear air (px) kept between two marker rectangles before they merge. */
export const COLLIDE_GAP = 8;

/** Do two projected markers collide? Rectangle test when both sizes are known (anchored at the
 *  bottom-center tip, so a marker's box spans x±w/2 horizontally and y-h..y vertically),
 *  else the legacy center-distance circle. Exported for the unit tests. */
export function markersCollide(a: PixelPoint, b: PixelPoint, radius: number, gap = COLLIDE_GAP): boolean {
	if (a.w != null && a.h != null && b.w != null && b.h != null) {
		const overlapX = Math.abs(a.x - b.x) < (a.w + b.w) / 2 + gap;
		// Bottom-anchored boxes: vertical span is [y-h, y].
		const topA = a.y - a.h;
		const topB = b.y - b.h;
		const overlapY = topA < b.y + gap && topB < a.y + gap;
		return overlapX && overlapY;
	}
	const dx = a.x - b.x;
	const dy = a.y - b.y;
	return dx * dx + dy * dy <= radius * radius;
}

/**
 * Merge overlapping projected points into cells.
 *
 * @param points source markers in screen pixels (stable order in → stable order out)
 * @param radius fallback minimum center-to-center pixel gap for points without w/h.
 *               ~52px suits a ~28px-tall pill with breathing room; the caller owns the value.
 */
export function declusterByPixel(points: readonly PixelPoint[], radius: number): ClusterCell[] {
	const claimed = new Array<boolean>(points.length).fill(false);
	const cells: ClusterCell[] = [];

	for (let i = 0; i < points.length; i++) {
		if (claimed[i]) continue;
		const seed = points[i];
		claimed[i] = true;
		const members: PixelPoint[] = [seed];

		// Absorb every not-yet-claimed point colliding with the SEED. Seed-anchored (not
		// running-centroid) keeps the pass deterministic and O(n²) — fine for small n.
		for (let j = i + 1; j < points.length; j++) {
			if (claimed[j]) continue;
			if (markersCollide(seed, points[j], radius)) {
				claimed[j] = true;
				members.push(points[j]);
			}
		}

		if (members.length === 1) {
			cells.push({ key: seed.key, memberKeys: [seed.key], count: seed.count, x: seed.x, y: seed.y });
			continue;
		}

		let sx = 0;
		let sy = 0;
		let total = 0;
		for (const m of members) {
			sx += m.x;
			sy += m.y;
			total += m.count;
		}
		const n = members.length;
		// Sorted member keys → the cluster's identity is independent of input order, so a cluster
		// that survives a zoom nudge keeps its key and the marker is mutated, not recreated.
		const memberKeys = members.map((m) => m.key).sort();
		cells.push({
			key: `grp:${memberKeys.join(',')}`,
			memberKeys,
			count: total,
			x: sx / n,
			y: sy / n
		});
	}

	return cells;
}
