/**
 * location-filter.mjs — Shared location filter logic
 *
 * Reads `location_filter` from portals.yml and returns a classifier:
 *   (location: string) => 'pass' | 'block' | 'ambiguous'
 *
 * Used by scan.mjs (filter incoming offers) and prune-pipeline.mjs
 * (retroactively prune pipeline.md).
 */

// Bare-remote markers: location strings that mean "remote, country unspecified".
// These stay ambiguous (kept) even in allowlist mode so genuinely-Canada-eligible
// "Remote" postings aren't lost to overzealous filtering.
const BARE_REMOTE_PATTERNS = [
  /^remote$/i,
  /^hybrid$/i,
  /^remote\s*[/-]\s*hybrid$/i,
  /^anywhere$/i,
  /^global$/i,
  /^various$/i,
];

export function buildLocationFilter(config) {
  const filter = config?.location_filter;
  if (!filter || filter.enabled === false) {
    return () => 'pass';
  }

  const allowed = (filter.allowed_substrings || []).map(s => s.toLowerCase());
  const blocked = (filter.blocked_substrings || []).map(s => s.toLowerCase());
  const ambiguousAction = filter.ambiguous_action || 'flag';
  // mode: "allowlist" → unrecognized non-empty locations are blocked (safer for
  //   hard work-authorization constraints; bare "Remote" still ambiguous).
  // mode: "denylist" (default) → unrecognized locations follow ambiguous_action.
  const mode = filter.mode || 'denylist';

  const resolveAmbiguous = () => {
    if (ambiguousAction === 'allow') return 'pass';
    if (ambiguousAction === 'block') return 'block';
    return 'ambiguous';
  };

  return (location) => {
    const lower = (location || '').toLowerCase().trim();
    if (!lower) return ambiguousAction === 'block' ? 'block' : 'ambiguous';

    if (allowed.some(s => lower.includes(s))) return 'pass';
    if (blocked.some(s => lower.includes(s))) return 'block';

    if (mode === 'allowlist' && !BARE_REMOTE_PATTERNS.some(p => p.test(lower))) {
      return 'block';
    }

    return resolveAmbiguous();
  };
}
