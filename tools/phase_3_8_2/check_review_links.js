#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const repoRoot = path.resolve(__dirname, '..', '..');
const projectRoot = path.resolve(repoRoot, '..', '..');
const defaultReviewRoot = path.join(projectRoot, '03_审查包');

function argValue(name, fallback) {
  const idx = process.argv.indexOf(name);
  if (idx >= 0 && process.argv[idx + 1]) return process.argv[idx + 1];
  return fallback;
}

const reviewRoot = path.resolve(argValue('--review', defaultReviewRoot));
const files = [
  '00_START_HERE.html',
  'LATEST_REVIEW_PACKAGE.html',
  'PHASE_3_8_1_USABILITY_REVIEW.html',
  'PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html',
  '01_客户演示包/CUSTOMER_DEMO_SAFE_REVIEW.html',
  '01_客户演示包/CUSTOMER_PILOT_PROPOSAL.html',
];

const skip = /^(https?:|mailto:|tel:|javascript:|data:|#)/i;
const findings = [];

for (const relFile of files) {
  const file = path.join(reviewRoot, relFile);
  if (!fs.existsSync(file)) {
    findings.push(`${relFile}: source file missing`);
    continue;
  }
  const html = fs.readFileSync(file, 'utf8');
  const base = path.dirname(file);
  const re = /\b(?:href|src)\s*=\s*["']([^"']+)["']/gi;
  let match;
  while ((match = re.exec(html))) {
    const raw = match[1].trim();
    if (!raw || skip.test(raw)) continue;
    const clean = raw.split('#')[0].split('?')[0];
    if (!clean) continue;
    let decoded = clean;
    try { decoded = decodeURIComponent(clean); } catch (_) {}
    const target = path.resolve(base, decoded.replace(/\//g, path.sep));
    if (!target.startsWith(reviewRoot)) {
      findings.push(`${relFile}: ${raw} resolves outside review root`);
      continue;
    }
    if (!fs.existsSync(target)) findings.push(`${relFile}: missing ${raw}`);
  }
}

if (findings.length) {
  console.log('FAIL review link check');
  for (const finding of findings) console.log(finding);
  process.exit(1);
}

console.log(`PASS review link check: ${reviewRoot}`);
