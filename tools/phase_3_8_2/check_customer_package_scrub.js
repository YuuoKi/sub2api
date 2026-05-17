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
const customerRoot = path.resolve(argValue('--customer', path.join(reviewRoot, '01_客户演示包')));

const binaryExts = new Set(['.png', '.jpg', '.jpeg', '.gif', '.webp', '.ico', '.zip', '.pdf']);
const forbidden = [
  ['Sub2API', /\bSub2API\b/i],
  ['Wei-Shaw', /Wei-Shaw/i],
  ['GitHub', /GitHub/i],
  ['LGPL', /LGPL/i],
  ['Windows path', /[A-Za-z]:[\\/][^\s"'<>]+/],
  ['commit', /\bcommit\b/i],
  ['git', /\bgit\b/i],
  ['QA', /\bQA\b/i],
  ['password', /\bpassword\b/i],
  ['mock', /\bmock\b/i],
  ['localhost', /\blocalhost\b/i],
  ['127.0.0.1', /127\.0\.0\.1/],
  ['secret', /\bsecret\b/i],
  ['encrypted_api_key', /encrypted_api_key/i],
  ['plain_api_key', /plain_api_key/i],
  ['local_qa', /local_qa/i],
  ['video_qa_admin', /video_qa_admin/i],
  ['Authorization header', /\bAuthorization\s*:/i],
  ['Bearer token', /\bBearer\s+[A-Za-z0-9._-]{20,}/i],
  ['JWT token', /\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b/],
  ['OpenAI-style key', /\bsk-[A-Za-z0-9_-]{20,}\b/],
  ['AWS-style key', /\bAKIA[0-9A-Z]{16}\b/],
];

function walk(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return walk(full);
    return [full];
  });
}

const findings = [];
for (const file of walk(customerRoot)) {
  const rel = path.relative(customerRoot, file).replace(/\\/g, '/');
  const ext = path.extname(file).toLowerCase();
  for (const [label, pattern] of forbidden) {
    if (pattern.test(rel)) {
      findings.push({ file: rel, line: 0, term: label, text: '(filename)' });
    }
  }
  if (binaryExts.has(ext)) continue;
  const content = fs.readFileSync(file, 'utf8');
  const lines = content.split(/\r?\n/);
  lines.forEach((line, index) => {
    for (const [label, pattern] of forbidden) {
      pattern.lastIndex = 0;
      if (pattern.test(line)) {
        findings.push({ file: rel, line: index + 1, term: label, text: line.trim().slice(0, 180) });
      }
    }
  });
}

if (!fs.existsSync(customerRoot)) {
  console.log(`FAIL customer package not found: ${customerRoot}`);
  process.exit(1);
}

if (findings.length) {
  console.log('FAIL customer package scrub');
  for (const item of findings) {
    console.log(`${item.file}:${item.line}: ${item.term}: ${item.text}`);
  }
  process.exit(1);
}

console.log(`PASS customer package scrub: ${customerRoot}`);
