#!/usr/bin/env python3
"""High-confidence source secret scanner.

The scanner is intentionally conservative:
- scans Git-tracked files by default, with an opt-in pending-change mode;
- skips local ignored files such as .env;
- skips secret-bearing file types instead of reading them;
- prints only file, line number, and rule id, never the matched value.
"""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Iterable, Iterator, NamedTuple


MAX_FILE_BYTES = 1024 * 1024

SKIP_DIRS = {
    ".git",
    ".impeccable",
    ".idea",
    ".vscode",
    "_review",
    "__pycache__",
    "bin",
    "build",
    "coverage",
    "dist",
    "node_modules",
    "output",
    "vendor",
}

SKIP_FILE_NAMES = {
    ".env",
    ".env.local",
    ".env.production",
    ".env.staging",
    ".env.development",
}

SKIP_SUFFIXES = {
    ".7z",
    ".bin",
    ".crt",
    ".db",
    ".der",
    ".dll",
    ".exe",
    ".gif",
    ".gz",
    ".ico",
    ".jar",
    ".jpeg",
    ".jpg",
    ".key",
    ".lock",
    ".log",
    ".mp4",
    ".p12",
    ".pem",
    ".png",
    ".pyc",
    ".sqlite",
    ".webp",
    ".zip",
}

ALLOWLIST_TOKENS = (
    "allowlist secret",
    "dummy",
    "example",
    "fake",
    "fixture",
    "mock",
    "nosec",
    "notsecret",
    "placeholder",
    "redacted",
    "sample",
    "test",
)


class Rule(NamedTuple):
    name: str
    pattern: re.Pattern[str]


RULES = (
    Rule("aws-access-key-id", re.compile(r"\b(AKIA|ASIA)[0-9A-Z]{16}\b")),
    Rule("github-token", re.compile(r"\bgh[pousr]_[A-Za-z0-9_]{36,}\b")),
    Rule("openai-api-key", re.compile(r"\bsk-[A-Za-z0-9_-]{32,}\b")),
    Rule("stripe-live-secret", re.compile(r"\bsk_live_[A-Za-z0-9]{24,}\b")),
    Rule("slack-token", re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{20,}\b")),
    Rule("private-key-block", re.compile(r"-----BEGIN (RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----")),
    Rule(
        "generic-secret-assignment",
        re.compile(
            r"(?i)\b(api[_-]?key|client[_-]?secret|jwt[_-]?secret|password|private[_-]?key|secret|token)\b"
            r"\s*[:=]\s*(['\"])[A-Za-z0-9_./+=-]{32,}\2"
        ),
    ),
)


class Finding(NamedTuple):
    path: Path
    line_number: int
    rule_name: str


def repo_root() -> Path:
    root = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
    return Path(root)


def git_tracked_files(root: Path) -> list[Path]:
    raw = subprocess.check_output(["git", "ls-files", "-z"], cwd=root)
    return [root / name.decode("utf-8", errors="surrogateescape") for name in raw.split(b"\0") if name]


def git_untracked_files(root: Path) -> list[Path]:
    raw = subprocess.check_output(["git", "ls-files", "--others", "--exclude-standard", "-z"], cwd=root)
    return [root / name.decode("utf-8", errors="surrogateescape") for name in raw.split(b"\0") if name]


def git_pending_files(root: Path) -> list[Path]:
    seen: set[Path] = set()
    paths: list[Path] = []
    for path in [*git_tracked_files(root), *git_untracked_files(root)]:
        try:
            resolved = path.resolve()
        except OSError:
            continue
        if resolved in seen:
            continue
        seen.add(resolved)
        paths.append(path)
    return paths


def all_candidate_files(root: Path) -> Iterator[Path]:
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [name for name in dirnames if name not in SKIP_DIRS]
        for filename in filenames:
            yield Path(dirpath) / filename


def should_skip(path: Path, root: Path) -> bool:
    try:
        rel = path.relative_to(root)
    except ValueError:
        return True
    parts = set(rel.parts[:-1])
    if parts & SKIP_DIRS:
        return True
    name = path.name.lower()
    if name in SKIP_FILE_NAMES or name.startswith(".env."):
        return True
    if path.suffix.lower() in SKIP_SUFFIXES:
        return True
    try:
        return path.stat().st_size > MAX_FILE_BYTES
    except OSError:
        return True


def is_allowlisted(line: str) -> bool:
    lower = line.lower()
    return any(token in lower for token in ALLOWLIST_TOKENS)


def is_context_allowlisted(path: Path, rule_name: str) -> bool:
    suffix = path.suffix.lower()
    parts = {part.lower() for part in path.parts}
    name = path.name.lower()
    if rule_name == "generic-secret-assignment":
        if suffix in {".md", ".markdown"} or name.endswith(".example.yaml") or name.endswith(".example.yml"):
            return True
        if "_archive" in parts:
            return True
    if rule_name in {"generic-secret-assignment", "private-key-block"}:
        if "_test." in name or name.startswith("test_") or "__tests__" in parts:
            return True
    return False


def scan_file(path: Path, root: Path) -> Iterator[Finding]:
    if should_skip(path, root):
        return
    try:
        with path.open("r", encoding="utf-8", errors="ignore") as handle:
            for line_number, line in enumerate(handle, start=1):
                if is_allowlisted(line):
                    continue
                for rule in RULES:
                    if rule.pattern.search(line):
                        rel = path.relative_to(root)
                        if is_context_allowlisted(rel, rule.name):
                            continue
                        yield Finding(rel, line_number, rule.name)
                        break
    except OSError:
        return


def scan_paths(paths: Iterable[Path], root: Path) -> list[Finding]:
    findings: list[Finding] = []
    for path in paths:
        findings.extend(scan_file(path, root))
    return findings


def run_self_test() -> int:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        good = root / "good.txt"
        bad = root / "bad.txt"
        generic = root / "generic.txt"
        env = root / ".env"
        good.write_text("OPENAI_API_KEY=placeholder-value\n", encoding="utf-8")
        bad.write_text("OPENAI_API_KEY=sk-" + "a" * 40 + "\n", encoding="utf-8")
        generic.write_text('JWT_SECRET="' + "c" * 40 + '"\n', encoding="utf-8")
        env.write_text("OPENAI_API_KEY=sk-" + "b" * 40 + "\n", encoding="utf-8")
        findings = scan_paths([good, bad, generic, env], root)
    found = {(finding.path.as_posix(), finding.rule_name) for finding in findings}
    if found != {("bad.txt", "openai-api-key"), ("generic.txt", "generic-secret-assignment")}:
        print("secret-scan self-test failed", file=sys.stderr)
        return 1
    print("secret-scan self-test passed")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description="Scan source files for high-confidence secret leaks.")
    parser.add_argument("--all", action="store_true", help="scan all non-ignored-looking files under the repo root")
    parser.add_argument(
        "--include-untracked",
        action="store_true",
        help="scan tracked files plus untracked, non-ignored files intended for pending review",
    )
    parser.add_argument("--self-test", action="store_true", help="run scanner self-test")
    args = parser.parse_args()

    if args.all and args.include_untracked:
        parser.error("--all and --include-untracked cannot be used together")

    if args.self_test:
        return run_self_test()

    root = repo_root()
    if args.all:
        paths = all_candidate_files(root)
        scope = "all candidate"
    elif args.include_untracked:
        paths = git_pending_files(root)
        scope = "tracked-plus-untracked"
    else:
        paths = git_tracked_files(root)
        scope = "tracked-file"
    findings = scan_paths(paths, root)
    if not findings:
        print(f"secret-scan: no high-confidence {scope} findings")
        return 0

    print("secret-scan: potential secrets found; matched values are intentionally hidden")
    for finding in findings:
        print(f"{finding.path.as_posix()}:{finding.line_number}: {finding.rule_name}")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
