# Phase A' Tiny Real 阻塞记录

日期：2026-07-02
状态：已阻塞

## 结论

本轮已在 WSL 内恢复 Docker 并重建 Sub2API，本轮唯一一次 Sub2API 直连 Seedance tiny real 已发出，但任务最终失败。

未进入 QCanvas `/studio-v2` 第二次真实出片，避免 retry 风暴与预算扩大。

## 执行边界

- WSL distro：`Ubuntu-24.04`
- Docker：WSL 内 `docker` / `docker compose`
- Compose project：`sub2api_phasea_prime`
- Sub2API root：`D:/sub2api-trunk`
- QCanvas：未执行真实调用
- 真实 Seedance create 次数：1

## 前置验证

- `deploy/.env` 写入后校验：20 个键完整、无 CRLF、无 BOM、`VIDEO_GATEWAY_ENCRYPTION_KEY` 长度 64
- `/health` 连续 10 次返回 `{"status":"ok"}`
- Seedance provider preflight：`seedance_preflight=ready`

## 真实任务证据

Sub2API 直连 tiny real：

- `create_status=201`
- `task_id=1`
- `final_status=failed`
- `has_result_url=False`
- `error_message=video task exceeded max poll attempts (48)`

SQL 证据：

```sql
SELECT id, status, COALESCE(result_url,'') <> '' AS has_result_url,
       LEFT(COALESCE(error_message,''),120) AS error_preview
FROM video_tasks
WHERE id=1;
```

结果：

```text
id=1
status=failed
has_result_url=false
error_preview=video task exceeded max poll attempts (48)
```

入库证据：

```sql
SELECT COUNT(*) AS capture_rows_for_task_1
FROM ai_generation_content
WHERE task_id='1';
```

结果：

```text
capture_rows_for_task_1=0
```

Admin stats：

```text
login_http=200
stats_http=200
is_live=False
captured_today=0
```

## 未完成项

- QCanvas `/studio-v2` 未发起真实 tiny real
- 未产生 QCanvas 节点 `result_url`
- 未产生 QCanvas task 对应 `ai_generation_content` 行
- 未形成三证同屏截图

## 风险与说明

- 本轮真实 provider create 已发生一次，按边界不重发。
- 任务失败原因来自 Sub2API 最终任务态：`video task exceeded max poll attempts (48)`。
- 复核后确认，480p 默认 worker 轮询 cap 是 48 次；按默认 5s 间隔约 4 分钟。这个窗口短于 Seedance/字节视频常见 10 分钟级生成耗时。
- 收尾阶段已按原计划执行 `docker compose ... down -v` 并删除 `deploy/.env`，因此本地 DB、`task_id=1`、临时 API key 文件已经不可继续轮询；当前只能重新建环境并重新发起一次真实任务。
- 后续若继续真实复测，需要用临时 compose override 把容器环境中的 `VIDEO_GATEWAY_MAX_POLL_ATTEMPTS` / `VIDEO_GATEWAY_POLL_INTERVAL_SECONDS` 抬高，再执行一次 create；这会消耗新的真实调用预算。
- 当前不能声明“内部可用 / 可演示”，只能标记为“已阻塞 / 待复核”。
- 本记录不包含 API key、JWT、Bearer token、cookie 或数据库密码。

## 回滚 / 清理

按用户要求执行：

```bash
cd /mnt/d/sub2api-trunk/deploy
docker compose -p sub2api_phasea_prime -f docker-compose.dev.yml down -v
rm -f .env
```

额外删除 WSL 临时 key 文件：

```bash
rm -f /tmp/phasea_sub2api_api_key /tmp/phasea_direct_task.json
```
