# Codex 交接区（无界 AI 生产控制台）

本目录用于 **Codex 执行后端任务**，执行完成后把审查包交给 **Claude（前端）** 续做。

## 你怎么用（老板）

1. 在 Codex 里打开并让它读完：**[CODEX_START_HERE.md](./CODEX_START_HERE.md)**
2. Codex 做完后，会在 **`deliverables/`** 下生成 `YYYY-MM-DD-<任务ID>-review.md`
3. 你跟我说一声「Codex 做完了」，并告诉我审查包文件名
4. 我（Claude）只读 `deliverables/` 里的审查包，继续做前端，不再重复后端

## 目录说明

| 路径 | 用途 |
|------|------|
| [CODEX_START_HERE.md](./CODEX_START_HERE.md) | **Codex 唯一入口**，从这里开始 |
| [../plans/2026-07-04-console-v2-roadmap.md](../plans/2026-07-04-console-v2-roadmap.md) | 完整路线图（背景与优先级） |
| [DELIVERABLE_TEMPLATE.md](./DELIVERABLE_TEMPLATE.md) | 审查包 Markdown 模板 |
| `deliverables/` | **Codex 交付目录**（审查包放这里） |

## 分工边界（硬规则）

| 执行者 | 负责 |
|--------|------|
| **Codex** | P0-1 配置核对与冒烟、P0-2、P0-3 后端、P1-1 后端、P1-3 后端、P2 后端项 |
| **Claude** | P0-3 前端、P0-4、P1 前端项、P2-2/P2-5 — **等读完 deliverables 后再做** |
| **老板** | 提供 Seedance 等真实密钥；拍板计费单价与预算数字 |
