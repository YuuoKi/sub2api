# 审查包：DBug-8 — MLA-P2-013 secret-scan hook fail-closed

> 执行者：Codex
> 完成时间：2026-07-08 02:12 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- `hooks.json` 中 `git push` 前 secret-scan 钩子改为 `failClosed: true`。
- `secret-scan.ps1` 在缺少 Python 或 `tools/secret_scan.py` 时 `Write-Error` 并 **exit 2**，不再静默 exit 0 放行。
- `.cursor/SETUP.md` 补充 fail-closed 行为说明。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `.cursor/hooks.json` | `failClosed: true` |
| `.cursor/hooks/secret-scan.ps1` | 缺 Python/scanner → exit 2 |
| `.cursor/SETUP.md` | 文档同步 fail-closed 语义 |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 无 Python 时 exit 2 | pass | 见下方命令（Agent 沙箱无 Python） |
| `failClosed: true` | pass | `hooks.json` |
| 有 Python 时 scan 逻辑未改 | pass | 仍调用 `tools/secret_scan.py` |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk
# 模拟 hook stdin（空）+ 无 Python 环境
'' | powershell -NoProfile -ExecutionPolicy Bypass -File .cursor/hooks/secret-scan.ps1
# Write-Error: secret-scan requires Python ...
# exit=2

# 本机有 Python 时（开发者环境）
make secret-scan
# 或
python tools/secret_scan.py --include-untracked
```

---

## 5. 给 Claude 的前端接口说明（如有）

无。

---

## 6. 风险与遗留

- Agent 沙箱环境未安装 Python，无法在本会话内复现「有 Python 时 exit 0」；逻辑路径与改前一致，仅移除 fail-open 分支。
- **MLA Dbug 全 Phase（DBug-0～8）已完成。**

---

## 7. 阻塞项（若 status=blocked）

无。
