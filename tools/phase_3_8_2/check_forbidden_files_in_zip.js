#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const zlib = require('zlib');

const args = process.argv.slice(2);
const customerMode = args.includes('--customer');
const zipPaths = args.filter((arg) => !arg.startsWith('--'));

if (!zipPaths.length) {
  console.log('FAIL zip path required');
  process.exit(1);
}

const genericPathRules = [
  ['.env', /(^|\/)\.env($|[./])/i],
  ['.git', /(^|\/)\.git($|\/)/i],
  ['node_modules', /(^|\/)node_modules($|\/)/i],
  ['dist cache', /(^|\/)(dist|build|cache|\.cache)($|\/)/i],
  ['temporary file', /(~$|\.tmp$|\.temp$|\.bak$|\.log$)/i],
];

const customerPathRules = [
  ['internal review material', /(^|\/)(02_内部审查包|03_真实通道接入|04_截图证据|05_Phase_3_8_封存快照|90_历史归档)(\/|$)/i],
  ['phase review file', /(^|\/)(PHASE_|BOSS_|PROVIDER_|WORKSPACE_AUDIT|AUTOMATED_CHECK|MANIFEST|LATEST_REVIEW_PACKAGE|00_START_HERE)/i],
  ['tooling file', /(^|\/)tools(\/|$)/i],
];

const secretRules = [
  ['Authorization header', /\bAuthorization\s*:/i],
  ['Bearer token', /\bBearer\s+[A-Za-z0-9._-]{20,}/i],
  ['JWT token', /\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b/],
  ['OpenAI-style key', /\bsk-[A-Za-z0-9_-]{20,}\b/],
  ['AWS-style key', /\bAKIA[0-9A-Z]{16}\b/],
];

const customerTextRules = [
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
  ['local_qa', /local_qa/i],
  ['video_qa_admin', /video_qa_admin/i],
];

function readZipEntries(zipPath) {
  const buf = fs.readFileSync(zipPath);
  let eocd = -1;
  for (let i = buf.length - 22; i >= Math.max(0, buf.length - 66000); i--) {
    if (buf.readUInt32LE(i) === 0x06054b50) { eocd = i; break; }
  }
  if (eocd < 0) throw new Error('EOCD not found');
  const entryCount = buf.readUInt16LE(eocd + 10);
  const cdOffset = buf.readUInt32LE(eocd + 16);
  const entries = [];
  let p = cdOffset;
  for (let i = 0; i < entryCount; i++) {
    if (buf.readUInt32LE(p) !== 0x02014b50) throw new Error(`central directory corrupt at ${p}`);
    const method = buf.readUInt16LE(p + 10);
    const compressedSize = buf.readUInt32LE(p + 20);
    const nameLen = buf.readUInt16LE(p + 28);
    const extraLen = buf.readUInt16LE(p + 30);
    const commentLen = buf.readUInt16LE(p + 32);
    const localOffset = buf.readUInt32LE(p + 42);
    const name = buf.slice(p + 46, p + 46 + nameLen).toString('utf8').replace(/\\/g, '/');
    entries.push({ name, method, compressedSize, localOffset });
    p += 46 + nameLen + extraLen + commentLen;
  }
  return { buf, entries };
}

function entryData(zip, entry) {
  const p = entry.localOffset;
  if (zip.buf.readUInt32LE(p) !== 0x04034b50) return null;
  const nameLen = zip.buf.readUInt16LE(p + 26);
  const extraLen = zip.buf.readUInt16LE(p + 28);
  const start = p + 30 + nameLen + extraLen;
  const compressed = zip.buf.slice(start, start + entry.compressedSize);
  if (entry.method === 0) return compressed;
  if (entry.method === 8) return zlib.inflateRawSync(compressed);
  return null;
}

function isTextName(name) {
  return /\.(html?|md|json|txt|csv|tsv|js|mjs|cjs|ps1|css)$/i.test(name);
}

const findings = [];
for (const zipPath of zipPaths) {
  if (!fs.existsSync(zipPath)) {
    findings.push(`${zipPath}: zip missing`);
    continue;
  }
  const zip = readZipEntries(zipPath);
  for (const entry of zip.entries) {
    const name = entry.name;
    if (name.endsWith('/')) continue;
    for (const [label, rule] of genericPathRules) {
      if (rule.test(name)) findings.push(`${zipPath}: ${name}: forbidden path ${label}`);
    }
    if (customerMode) {
      for (const [label, rule] of customerPathRules) {
        if (rule.test(name)) findings.push(`${zipPath}: ${name}: customer zip contains ${label}`);
      }
    }
    if (!isTextName(name)) continue;
    const data = entryData(zip, entry);
    if (!data || data.length > 2_000_000) continue;
    const text = data.toString('utf8');
    for (const [label, rule] of secretRules) {
      if (rule.test(text)) findings.push(`${zipPath}: ${name}: sensitive token pattern ${label}`);
    }
    if (customerMode) {
      for (const [label, rule] of customerTextRules) {
        if (rule.test(text)) findings.push(`${zipPath}: ${name}: customer forbidden text ${label}`);
      }
    }
  }
}

if (findings.length) {
  console.log('FAIL zip forbidden file check');
  for (const finding of findings) console.log(finding);
  process.exit(1);
}

console.log(`PASS zip forbidden file check: ${zipPaths.join(', ')}`);
