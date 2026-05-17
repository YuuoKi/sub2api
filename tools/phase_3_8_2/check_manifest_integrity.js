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
const manifestPath = path.join(reviewRoot, 'MANIFEST.json');
const requiredValidation = [
  'workspace_audit',
  'usability_audit',
  'customer_scrub',
  'link_check',
  'manifest_check',
  'zip_forbidden_files',
  'frontend_build',
  'demo_build',
  'git_diff_check',
  'screenshot_check',
  'real_external_api_connected',
  'production',
];

function resolveReviewPath(raw) {
  const normalized = String(raw || '').replace(/\\/g, '/');
  if (normalized.startsWith('03_审查包/')) {
    return path.join(projectRoot, normalized.replace(/\//g, path.sep));
  }
  return path.join(reviewRoot, normalized.replace(/\//g, path.sep));
}

const findings = [];
let manifest;
try {
  manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
} catch (err) {
  console.log(`FAIL manifest JSON parse: ${err.message}`);
  process.exit(1);
}

if (!Array.isArray(manifest.latest_files) || !manifest.latest_files.length) {
  findings.push('latest_files missing or empty');
} else {
  for (const item of manifest.latest_files) {
    if (!item.name || !item.path) findings.push(`latest_files item incomplete: ${JSON.stringify(item)}`);
    else if (!fs.existsSync(resolveReviewPath(item.path))) findings.push(`latest_files missing path: ${item.path}`);
  }
}

if (!Array.isArray(manifest.screenshots) || !manifest.screenshots.length) {
  findings.push('screenshots missing or empty');
} else {
  for (const shot of manifest.screenshots) {
    if (!fs.existsSync(resolveReviewPath(shot))) findings.push(`screenshot missing: ${shot}`);
  }
}

if (!manifest.customer_package || !manifest.customer_package.path) {
  findings.push('customer_package.path missing');
} else if (!fs.existsSync(resolveReviewPath(manifest.customer_package.path))) {
  findings.push(`customer_package.path not found: ${manifest.customer_package.path}`);
}

if (!manifest.validation || typeof manifest.validation !== 'object') {
  findings.push('validation missing');
} else {
  for (const key of requiredValidation) {
    if (!(key in manifest.validation)) findings.push(`validation.${key} missing`);
  }
}

if (findings.length) {
  console.log('FAIL manifest integrity');
  for (const finding of findings) console.log(finding);
  process.exit(1);
}

console.log(`PASS manifest integrity: ${manifestPath}`);
