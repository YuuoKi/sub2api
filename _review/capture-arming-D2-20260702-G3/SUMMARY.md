# Sub2API D2 G3 采集受控 dev 验证补跑审查包

> 执行日期：2026-07-02  
> 执行目录：`D:\sub2api-trunk`  
> 分支：`wujie/video-capture-moat-20260702`  
> 目标：G3 从「已阻塞」补跑到「内部可用」  
> 当前判定：**内部可用**  
> 边界：未 push、未 deploy、未调用真实付费 provider、未打印密码/API Key/JWT 明文、未修改 `config.example.yaml`。

## 结论

G3 受控 dev 闭环已跑通，可从「已阻塞」改判为 **内部可用**。

本轮完成了 dev compose 环境启动、采集双闸开启、1 条 chat completions、1 条 `provider=mock` video task、SQL 脱敏验证、Admin stats/samples 看板验证，以及 `down -v` 清理。

关键验收结果：

- dev 是否起得来：**能起，并在验证窗口内 `/health=true`**；但本机 Docker/WSL 多次出现外部 fast shutdown/restart，见风险。
- `ai_generation_content` 近 15 分钟行数：**2**。
- `suspicious_rows`：**0**。
- Admin 看板：**`is_live=true`**，`captured_today=2`，`sample_count=2`。
- 清理：**已完成**，`deploy/.env` 已删除，`sub2api_g3_*` named volumes 已由 `down -v` 删除，`deploy/data/`、`deploy/postgres_data/`、`deploy/redis_data/` 不存在。

## 变更清单

- `backend/internal/config/config.go`
  - 新增 content capture / retention 的显式 `BindEnv`。
  - 解决 `GATEWAY_CONTENT_CAPTURE_ENABLED=true`、`GATEWAY_CONTENT_RETENTION_ENABLED=true` 只存在于环境变量但没有进入 Viper unmarshal 结果的问题。
- `backend/internal/config/config_test.go`
  - 新增 `TestLoadContentCaptureFlagsFromEnv`。
- `deploy/docker-compose.dev.yml`
  - 显式传入 `GATEWAY_CONTENT_CAPTURE_ENABLED` / `GATEWAY_CONTENT_RETENTION_ENABLED`。
  - 将 dev 数据落点改为 compose named volumes，便于 `down -v` 清理，避免 Windows bind mount 残留。
  - 增加 compose 内部 `anthropic-mock`，仅用于 dev chat 采集路径，不暴露宿主机端口。
- `frontend/package.json`
  - 增加 `pnpm.overrides`：`form-data@^4.0.6`、`js-cookie@^3.0.8`，匹配当前 lockfile，解除 Docker frozen install 阻塞。
- `_review/capture-arming-D2-20260702-G3/*`
  - 保存本轮 G3 验证脚本、SQL/Admin 证据和结果摘要。
- `docs/reviews/LATEST_REVIEW_PACKAGE.html`
  - 更新为本轮 G3 单一审查包。

未纳入本轮的既有脏文件：`.cursor/SETUP.md`、`.impeccable/`、`MORNING_RESULT_2026_06_28.md`。本轮未处理这些文件。

## 实现细节

### 配置映射

`gateway.content_capture.enabled` 依赖 Viper 点号转下划线，对应环境变量：

```text
GATEWAY_CONTENT_CAPTURE_ENABLED=true
GATEWAY_CONTENT_RETENTION_ENABLED=true
```

原问题是 `AutomaticEnv` 配合 `Unmarshal` 对未绑定 key 的 env-only 配置不可靠。本轮增加显式 `BindEnv`，默认仍保持关闭，不改示例配置默认值。

### Dev harness

`deploy/docker-compose.dev.yml` 继续只服务 dev 验证：

- Postgres / Redis / server 使用 `sub2api_g3` project。
- 数据使用 named volumes，验证结束后 `down -v` 删除。
- `anthropic-mock` 在 compose network 内提供 `/v1/messages` mock SSE 响应，用于 chat capture，不触达真实付费 provider。
- mock video 仍走现有 `provider=mock` 路径。

## 验证命令与结果

### Git 基线

```powershell
git status --short
git branch --show-current
git rev-parse --show-toplevel
```

结果：

- 分支：`wujie/video-capture-moat-20260702`
- Git root：`D:/sub2api-trunk`
- 本轮未执行 `git add`、未 push。

### Dev 启动

```powershell
cd D:\sub2api-trunk\deploy
docker compose -p sub2api_g3 -f docker-compose.dev.yml up --build -d
```

验证窗口内：

- `sub2api` health 探针返回 true。
- chat + mock video + SQL + Admin probe 均通过容器内/compose 内路径完成。

补充风险证据：本机 Docker/WSL 在后续 `ps/logs` 复查时多次出现 postgres `received fast shutdown request`，随后服务自动重启；因此审查结论限定为本机受控 dev 验证「内部可用」，并保留运行稳定性风险。

### HTTP 用户路径

证据文件：`_review/capture-arming-D2-20260702-G3/g3_http_result.json`

结果摘要：

```json
{
  "status": "ok",
  "chat_ok": true,
  "video_provider": "mock",
  "video_final_status": "succeeded",
  "result_url_present": true,
  "ResultURL_present": true
}
```

说明：请求阶段未保存 API Key/JWT 明文。`run_g3_verify.ps1` 在 video `succeeded`、`result_url` 与 `ResultURL` 双字段检查之后进入 SQL 阶段；后续早期失败点是 PowerShell 脚本内部无法查找 `docker` CLI，不是 chat/video HTTP 失败。SQL 与 Admin 证据随后由容器内 probe 补齐。

### SQL 采集与脱敏

执行：

```powershell
docker compose -p sub2api_g3 -f docker-compose.dev.yml cp ../_review/capture-arming-D2-20260702-G3/g3_sql_evidence.sql postgres:/tmp/g3_sql_evidence.sql
docker compose -p sub2api_g3 -f docker-compose.dev.yml exec -T postgres psql -U sub2api -d sub2api -v ON_ERROR_STOP=1 -f /tmp/g3_sql_evidence.sql
```

证据文件：

- `_review/capture-arming-D2-20260702-G3/g3_sql_recent_count.txt`：`2`
- `_review/capture-arming-D2-20260702-G3/g3_sql_suspicious_rows.txt`：`0`
- `_review/capture-arming-D2-20260702-G3/g3_sql_recent_rows.txt`

SQL 行预览包含：

- `mock-video-v1`：prompt 中手机号和 token 已脱敏。
- `claude-3-5-haiku-20241022`：chat prompt 中手机号和 token 已脱敏。

### Admin 看板

执行：

```powershell
docker compose -p sub2api_g3 -f docker-compose.dev.yml exec -T sub2api sh -c "chmod +x /tmp/g3_admin_probe.sh && /tmp/g3_admin_probe.sh"
```

证据文件：`_review/capture-arming-D2-20260702-G3/g3_admin_result.json`

结果：

```json
{
  "status": "ok",
  "stats_status": 200,
  "samples_status": 200,
  "is_live": true,
  "samples_live": true,
  "captured_today": 2,
  "sample_count": 2,
  "sample_preview_safe": true
}
```

## 清理结果

执行：

```powershell
cd D:\sub2api-trunk\deploy
docker compose -p sub2api_g3 -f docker-compose.dev.yml down -v
Remove-Item -LiteralPath D:\sub2api-trunk\deploy\.env -Force
```

确认：

- `down -v` 输出显示 `sub2api_g3_postgres_data_dev`、`sub2api_g3_redis_data_dev`、`sub2api_g3_sub2api_data_dev` 已删除。
- `deploy/.env`：不存在。
- `deploy/data/`：不存在。
- `deploy/postgres_data/`：不存在。
- `deploy/redis_data/`：不存在。
- `git status --short --ignored deploy _review/capture-arming-D2-20260702-G3 docs/reviews/LATEST_REVIEW_PACKAGE.html` 未显示 `.env` 或数据目录被 staged。

## 风险

- 本机 Docker/WSL 复查阶段存在外部 fast shutdown/restart 敏感性；G3 闭环已在健康窗口内完成，但不应据此宣称长期 dev 环境稳定。
- `anthropic-mock` 是 dev-only harness；它不暴露宿主机端口，不用于生产。
- `deploy/docker-compose.dev.yml` 改为 named volumes 后，不再生成 `deploy/postgres_data/` 等目录；这有利于清理，但与旧的本地目录观察习惯不同。
- `docs/reviews/LATEST_REVIEW_PACKAGE.html` 当前是 ignored 文件，已更新但不会自然出现在普通 `git status --short` 中。

## 回滚方案

```powershell
git checkout -- backend/internal/config/config.go backend/internal/config/config_test.go deploy/docker-compose.dev.yml frontend/package.json
Remove-Item -Recurse -Force _review/capture-arming-D2-20260702-G3
```

环境清理可重复执行：

```powershell
cd D:\sub2api-trunk\deploy
docker compose -p sub2api_g3 -f docker-compose.dev.yml down -v
```

## 文件索引

- `_review/capture-arming-D2-20260702-G3/SUMMARY.md`
- `_review/capture-arming-D2-20260702-G3/run_g3_verify.ps1`
- `_review/capture-arming-D2-20260702-G3/admin_probe_container.sh`
- `_review/capture-arming-D2-20260702-G3/g3_http_result.json`
- `_review/capture-arming-D2-20260702-G3/g3_admin_result.json`
- `_review/capture-arming-D2-20260702-G3/g3_sql_evidence.sql`
- `_review/capture-arming-D2-20260702-G3/g3_sql_recent_count.txt`
- `_review/capture-arming-D2-20260702-G3/g3_sql_suspicious_rows.txt`
- `_review/capture-arming-D2-20260702-G3/g3_sql_recent_rows.txt`
- `docs/reviews/LATEST_REVIEW_PACKAGE.html`

## 可复制后续提示词

```text
继续 Sub2API D2/G3 采集受控 dev 验证收口。当前 G3 已在 D:\sub2api-trunk 的 wujie/video-capture-moat-20260702 分支补跑到「内部可用」：SQL 近 15 分钟 2 行，suspicious_rows=0，Admin is_live=true，清理已完成。请只读复核 _review/capture-arming-D2-20260702-G3/SUMMARY.md 与 docs/reviews/LATEST_REVIEW_PACKAGE.html，不触达真实 provider，不打印任何密钥/JWT/API Key，不 push。
```
