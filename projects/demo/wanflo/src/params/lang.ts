import type { ParamMatcher } from '@sveltejs/kit';

// Only 'ru' and 'th' are real prefixes — English lives at the bare root, so the [[lang]]
// segment is absent for it. Anything else must NOT match, otherwise /nonsense would render
// an empty locale instead of 404ing (SC2e).
export const match: ParamMatcher = (param) => param === 'ru' || param === 'th';
