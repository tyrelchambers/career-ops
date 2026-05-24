#!/usr/bin/env node

/**
 * prune-pipeline.mjs — Retroactively prune pipeline.md by location
 *
 * Re-fetches Greenhouse/Ashby/Lever APIs for tracked companies to build a
 * URL → location map, then drops pending entries whose location is blocked
 * by `location_filter` in portals.yml.
 *
 * URLs that can't be resolved (manual adds, LinkedIn, unknown ATS) are
 * kept as-is — we never drop something we can't classify.
 *
 * Usage:
 *   node prune-pipeline.mjs            # apply prune
 *   node prune-pipeline.mjs --dry-run  # preview only
 */

import { readFileSync, writeFileSync, appendFileSync, existsSync } from 'fs';
import yaml from 'js-yaml';
import { buildLocationFilter } from './location-filter.mjs';

const PORTALS_PATH = 'portals.yml';
const PIPELINE_PATH = 'data/pipeline.md';
const SCAN_HISTORY_PATH = 'data/scan-history.tsv';
const FETCH_TIMEOUT_MS = 10_000;
const CONCURRENCY = 10;

function detectApi(company) {
  if (company.api && company.api.includes('greenhouse')) {
    return { type: 'greenhouse', url: company.api };
  }
  const url = company.careers_url || '';
  const ashby = url.match(/jobs\.ashbyhq\.com\/([^/?#]+)/);
  if (ashby) return { type: 'ashby', url: `https://api.ashbyhq.com/posting-api/job-board/${ashby[1]}?includeCompensation=true` };
  const lever = url.match(/jobs\.lever\.co\/([^/?#]+)/);
  if (lever) return { type: 'lever', url: `https://api.lever.co/v0/postings/${lever[1]}` };
  const ghEu = url.match(/job-boards(?:\.eu)?\.greenhouse\.io\/([^/?#]+)/);
  if (ghEu && !company.api) return { type: 'greenhouse', url: `https://boards-api.greenhouse.io/v1/boards/${ghEu[1]}/jobs` };
  return null;
}

const PARSERS = {
  greenhouse: (json) => (json.jobs || []).map(j => ({ url: j.absolute_url || '', location: j.location?.name || '' })),
  ashby: (json) => (json.jobs || []).map(j => ({ url: j.jobUrl || '', location: j.location || '' })),
  lever: (json) => (Array.isArray(json) ? json : []).map(j => ({ url: j.hostedUrl || '', location: j.categories?.location || '' })),
};

async function fetchJson(url) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);
  try {
    const res = await fetch(url, { signal: controller.signal });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return await res.json();
  } finally {
    clearTimeout(timer);
  }
}

async function parallelMap(items, fn, limit) {
  const results = [];
  let i = 0;
  async function worker() {
    while (i < items.length) {
      const idx = i++;
      results[idx] = await fn(items[idx]);
    }
  }
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker));
  return results;
}

async function buildUrlLocationMap(config) {
  const companies = (config.tracked_companies || [])
    .filter(c => c.enabled !== false)
    .map(c => ({ ...c, _api: detectApi(c) }))
    .filter(c => c._api !== null);

  const map = new Map();
  await parallelMap(companies, async (company) => {
    try {
      const json = await fetchJson(company._api.url);
      const jobs = PARSERS[company._api.type](json);
      for (const j of jobs) {
        if (j.url) map.set(j.url, j.location || '');
      }
    } catch {
      // ignore — those URLs simply won't be in the map and will be kept
    }
  }, CONCURRENCY);

  return map;
}

function parsePipelineEntry(line) {
  const m = line.match(/^- \[ \] (\S+)(?:\s*\|\s*([^|]+?)\s*\|\s*(.+))?$/);
  if (!m) return null;
  return { url: m[1], company: (m[2] || '').trim(), title: (m[3] || '').trim() };
}

async function main() {
  const dryRun = process.argv.includes('--dry-run');

  if (!existsSync(PORTALS_PATH) || !existsSync(PIPELINE_PATH)) {
    console.error('Error: portals.yml or data/pipeline.md missing.');
    process.exit(1);
  }

  const config = yaml.load(readFileSync(PORTALS_PATH, 'utf-8'));
  const locationFilter = buildLocationFilter(config);

  console.log('Building URL → location map from tracked APIs…');
  const urlLocations = await buildUrlLocationMap(config);
  console.log(`Resolved locations for ${urlLocations.size} URLs.`);

  const text = readFileSync(PIPELINE_PATH, 'utf-8');
  const lines = text.split('\n');

  const dropped = [];
  const ambiguous = [];
  const kept = [];
  const unresolved = [];

  const out = [];
  for (const line of lines) {
    const entry = parsePipelineEntry(line);
    if (!entry) { out.push(line); continue; }

    const location = urlLocations.get(entry.url);
    if (location === undefined) {
      // URL not in any tracked-API response — keep as-is (manual adds, expired, LinkedIn, etc.)
      unresolved.push(entry);
      out.push(line);
      continue;
    }

    const verdict = locationFilter(location);
    if (verdict === 'block') {
      dropped.push({ ...entry, location });
      continue;
    }
    if (verdict === 'ambiguous') {
      ambiguous.push({ ...entry, location });
    } else {
      kept.push({ ...entry, location });
    }
    out.push(line);
  }

  // Collapse runs of blank lines created by removals
  const cleaned = out.join('\n').replace(/\n{3,}/g, '\n\n');

  console.log(`\n${'━'.repeat(45)}`);
  console.log(`Prune Pipeline — ${new Date().toISOString().slice(0, 10)}`);
  console.log(`${'━'.repeat(45)}`);
  console.log(`Resolved & kept (Canada/global):  ${kept.length}`);
  console.log(`Resolved & ambiguous (kept):      ${ambiguous.length}`);
  console.log(`Resolved & blocked (DROPPED):     ${dropped.length}`);
  console.log(`Unresolved (kept untouched):      ${unresolved.length}`);

  if (dropped.length > 0) {
    console.log('\nDropped entries (sample, up to 30):');
    for (const d of dropped.slice(0, 30)) {
      console.log(`  - ${d.company} | ${d.title} | ${d.location}`);
    }
    if (dropped.length > 30) console.log(`  …and ${dropped.length - 30} more`);
  }

  if (dryRun) {
    console.log('\n(dry run — no files changed)');
    return;
  }

  writeFileSync(PIPELINE_PATH, cleaned, 'utf-8');

  // Log dropped to scan-history.tsv as `skipped_location`
  if (existsSync(SCAN_HISTORY_PATH) && dropped.length > 0) {
    const date = new Date().toISOString().slice(0, 10);
    const lines = dropped.map(d =>
      `${d.url}\t${date}\tprune-pipeline\t${d.title}\t${d.company}\tskipped_location`
    ).join('\n') + '\n';
    appendFileSync(SCAN_HISTORY_PATH, lines, 'utf-8');
  }

  console.log(`\nWrote pruned pipeline to ${PIPELINE_PATH}.`);
  console.log(`Dropped URLs logged to ${SCAN_HISTORY_PATH} as skipped_location.`);
}

main().catch(err => {
  console.error('Fatal:', err.message);
  process.exit(1);
});
