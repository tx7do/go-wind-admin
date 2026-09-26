#!/usr/bin/env node
// Trilingual README structural parity check.
//
// README.md / README.en-US.md / README.ja-JP.md are maintained as line-level
// mirrors: every `##` section heading must sit on the same line number in all
// three files, and total line counts must match. An edit applied to only one
// language shifts the headings and fails this check — mirror the edit into the
// other two files to fix.
//
// Usage: node scripts/check-readme-parity.mjs [zhPath enPath jaPath]
//        (paths are optional; overriding them is for testing only)

import { readFileSync } from 'node:fs';

const args = process.argv.slice(2);
if (args.length !== 0 && args.length !== 3) {
  console.error('usage: node scripts/check-readme-parity.mjs [zhPath enPath jaPath]');
  process.exit(2);
}
const paths = args.length === 3 ? args : ['README.md', 'README.en-US.md', 'README.ja-JP.md'];

function scan(path) {
  const lines = readFileSync(path, 'utf8').split(/\r?\n/);
  const headings = [];
  let inFence = false;
  lines.forEach((line, idx) => {
    if (/^\s*```/.test(line)) inFence = !inFence; // fenced blocks may contain '#' comment lines
    if (inFence) return;
    const m = /^## (.+?)\s*$/.exec(line);
    if (m) headings.push({ line: idx + 1, text: m[1] });
  });
  return { path, total: lines.length, headings };
}

const scans = paths.map(scan);
const base = scans[0];
const problems = [];

for (const s of scans.slice(1)) {
  if (s.total !== base.total) {
    problems.push(`total line count differs: ${base.path}=${base.total}, ${s.path}=${s.total}`);
  }
  if (s.headings.length !== base.headings.length) {
    problems.push(`section count differs: ${base.path}=${base.headings.length}, ${s.path}=${s.headings.length}`);
    continue;
  }
  s.headings.forEach((h, i) => {
    const b = base.headings[i];
    if (h.line !== b.line) {
      problems.push(
        `section #${i + 1} drifted: ${base.path} "${b.text}" at line ${b.line} vs ` +
        `${s.path} "${h.text}" at line ${h.line} — an edit was mirrored into one language only`
      );
    }
  });
}

if (problems.length > 0) {
  console.error(`README parity check FAILED (baseline ${base.path}: ${base.headings.length} sections, ${base.total} lines):`);
  for (const p of problems) console.error('  - ' + p);
  console.error('fix: apply the same edit to all three READMEs — they are line-level mirrors.');
  process.exit(1);
}
console.log(`README parity OK: 3 files, ${base.headings.length} sections, ${base.total} lines, all headings line-parallel.`);
