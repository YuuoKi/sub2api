# Optional Stage C dependency audit (report only)

## Scope

- Command: `pnpm audit --json`
- Working directory: `D:\sub2api-trunk\frontend`
- Raw output: `_review/night-moat-20260702/frontend-pnpm-audit-raw.json`
- Exit code: 1 (vulnerabilities reported)
- No upgrade, install, fix, or approve-builds command was executed.

## Summary

| Severity | Count |
| --- | ---: |
| critical | 1 |
| high | 14 |
| moderate | 33 |
| low | 3 |
| total advisories | 51 |

## High And Critical Highlights

| Package | Severity | Patched Versions | Note |
| --- | --- | --- | --- |
| `vitest` | critical | `>=3.2.6` | Vitest UI server arbitrary file read/execute advisory. Likely requires major test-stack planning. |
| `vite` | high | `>=6.4.3` | Windows alternate path `server.fs.deny` bypass. Likely major Vite upgrade path. |
| `rollup` | high | `>=4.59.0` | Arbitrary file write via path traversal. |
| `ws` | high | `>=8.21.0` | Memory exhaustion DoS. |
| `lodash` | high | `>=4.17.24` | Template import key code injection advisory. |
| `minimatch` | high | `>=3.1.3`, `>=3.1.4`, `>=9.0.6`, `>=9.0.7` | Multiple ReDoS advisories across dependency ranges. |
| `picomatch` | high | `>=2.3.2`, `>=4.0.4` | ReDoS via extglob quantifiers. |
| `flatted` | high | `>=3.4.0`, `>=3.4.2` | DoS / prototype pollution advisories. |

## Recommendation

Treat this as a separate dependency-hardening task. Prioritize dev-server-exposure packages (`vite`, `vitest`, `ws`) and lockfile transitive chains, but do not combine dependency upgrades with the moat skeleton branch unless explicitly authorized.
