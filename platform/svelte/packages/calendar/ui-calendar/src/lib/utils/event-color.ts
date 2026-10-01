// Resolve the visual style for one calendar event. design-5 colours events by event
// TYPE (a muted fill + a readable foreground), with STATUS shown via strikethrough /
// opacity rather than colour. Pure + side-effect-free so it unit-tests in isolation.
//
// Precedence, highest first:
//   1. item.color            — a legacy per-event SOLID fill (white text) a host set
//                              explicitly on the item; kept working for back-compat.
//   2. typeColors[eventType] — the muted per-type style (the design-5 model). The HOST
//                              owns the event_type dictionary, so the host supplies the
//                              map; the package ships NO palette of its own.
//   3. undefined             — the caller leaves EC's status-class default fill in place
//                              (--ec-event-bg-color), preserving the prior look for any
//                              consumer that passes no typeColors.
import type { CalendarItem, EventTypeStyle } from '../types.js';

export function resolveEventStyle(
	item: CalendarItem,
	typeColors?: Record<string, EventTypeStyle>
): EventTypeStyle | undefined {
	if (item.color) return { fill: item.color, text: '#ffffff' };
	return typeColors?.[item.eventType];
}
