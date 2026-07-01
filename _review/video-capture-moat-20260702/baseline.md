# Sub2API Video Capture Moat Baseline

## Scope

- Date: 2026-07-02
- Repo: `D:\sub2api-trunk`
- Goal: execute stages 0-2 of the video capture moat task only.
- Hard stop: do not enter stage 3 without separate human authorization.
- Red lines: no push, merge, rebase, reset, clean, delete, real provider calls, secret reads, or `AUTH_CONTRACT_SPLIT` changes.

## Git Baseline

- `git rev-parse --show-toplevel`: `D:/sub2api-trunk`
- Linked worktree git dir: `D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.git/worktrees/objective-goldstine-b9bf89`
- Common git dir: `D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.git`
- Source branch before this task: `wujie/night-moat-20260702`
- Source branch last commits:
  - `e7b392ce docs(review): mark night moat closeout complete`
  - `0a4543f1 docs(review): finalize night moat summary`
  - `55040eef docs(review): add optional dependency audit report`
- Task branch: `wujie/video-capture-moat-20260702`
- Stage 0 carry-forward commit: `d11a13c0 chore: carry forward local hook/setup tweaks`

## Dirty Worktree Handling

Initial dirty state matched the task brief:

```text
 M .cursor/SETUP.md
 M .cursor/hooks/project-quality-gate.ps1
?? .impeccable/hook.cache.json
?? MORNING_RESULT_2026_06_28.md
```

The two `.cursor` modifications were inspected. They only document and tune local Cursor hook behavior:

- `.cursor/SETUP.md`: explains July 2026 hook trigger behavior and local state files.
- `.cursor/hooks/project-quality-gate.ps1`: narrows reminders to real source file extensions and emits `{}` when no backend/frontend source diff requires a reminder.

No secrets, provider calls, destructive actions, or project red-line changes were found in those diffs. Per the task brief, they were carried onto the task branch in a separate commit.

Current remaining untracked files intentionally remain untouched:

```text
?? .impeccable/hook.cache.json
?? MORNING_RESULT_2026_06_28.md
```

## Truth Sources Checked

- `00_START_HERE.md`: present. Current status is internal usable / demoable / real supplier authorization pending recheck.
- `docs/reviews/LATEST_REVIEW_PACKAGE.html`: referenced by `00_START_HERE.md`, but not present in the visible file list at this point.
- `docs/goals/20260702_NIGHT_CODEX_长任务_Sub2API_video护城河闭环.md`: not present in the visible file list; the pasted task brief is the controlling task source for this run.
- `_review/night-moat-20260702/SUMMARY.md`: present and checked; confirms prior local-only closeout and that remaining untracked files were intentionally left uncommitted.

## Stage Boundary

Stage 0 is complete. Proceeding to stage 1 baseline gates. Stage 3 remains blocked pending explicit human authorization.
