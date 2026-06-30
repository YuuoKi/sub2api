# Sub2API · 当前真实状态（真相矩阵）

更新时间：2026-06-18 Asia/Shanghai
证据规则：状态以最新 repo / 审查包 / 日志 / 测试 / 真实链路为准；长期记忆与 Agent 自述只作背景。

## 一句话

真实 Seedance 2.0 链路在 **单仓 harness 层已真打通**（真出片、真计费、token 计费已校准）= **局部 READY**；
端到端（QCanvas→Sub2API→Seedance→回画布）= **mock，从未真通**。局部 READY ≠ 产品 READY。

## 真实链路状态矩阵

| 环节 | 状态 | 证据 |
|---|---|---|
| Sub2API → Seedance（harness） | 真通 · 有缺陷 | 真实冒烟 redacted 事件日志 `backend/data/seedance-smoke.shot3-916-720p5s.redacted-events.log`（720p/16:9/5s/24fps、`status:succeeded`、`content.video_url` 有值、`usage.total_tokens:108900`）；harness `video_gateway_realsmoke_test.go`（形态 B）/ `video_gateway_forma_realsmoke_test.go`（形态 A），均 `//go:build realsmoke` 三层关断 |
| QCanvas → Sub2API | mock · 端到端未通 | QCanvas `apps/hono-api` 走 `SUB2API_BASE_URL`/`SUB2API_API_KEY`，未配置→落本地 dry-run；真实服务进程未起 |
| Sub2API → Kling | 未接 | `klingVideoAdapter` 骨架，create/poll 均返回 `KLING_REAL_CALL_DISABLED` |
| Sub2API → 图片 / 文本 | 未接 | 无 adapter |

## token 计费真相（首枪坐实）

- 火山方舟 Seedance **按 token 计费、非按秒**：一条 5s 片 = `108900` tokens（real-shot `usage`）。
- 计费精确金额仍「待复核」（需以真实账单核对回填）。

## 接口契约缺陷（Phase B 修复对象）

### B1【必修】请求侧比例字段名错 —— 竖屏做不出的根因
- **现状**：`backend/internal/service/video_gateway_adapter.go:198` 请求体写 `payload["aspect_ratio"] = task.AspectRatio`；`BuildCreatePayload`（~:462）同样输出 `aspect_ratio`。
- **真相**：Ark 官方字段是 `ratio`（取值 `16:9` / `9:16` / `1:1`）；真实响应回显的也是 `ratio`（不是 `aspect_ratio`）。**响应侧解析已在 commit `7b78f9ca` 改对（解析 `ratio`），但请求侧仍发错字段** —— 这正是「ratio 字段对齐」与「缺陷仍在」两说并存的真相：**响应已修、请求未修**。
- **根因链**：请求发 `aspect_ratio` → Ark 不识别 → 落默认 16:9 横屏 → 竖屏（9:16）做不出。
- **修法（Phase B，不真实调用）**：请求体改 `ratio`；建立 竖屏→9:16 / 横屏→16:9 / 方形→1:1 映射 + 直通合法 `W:H`；`resolution` 字段不动。契约测试断言 payload 字段名与取值正确。
- **待授权**：「发 `ratio:9:16` 是否真产出竖屏视频」需一次真实付费调用确认 → 列入待授权清单。

### B2【必修】轮询窗口不足
- **现状**：`backend/internal/service/video_gateway_worker.go` `videoDefaultMaxPollAttempts=72` × `5s` = **6 分钟**轮询窗；非分辨率相关；外层 wall-clock `task_timeout_minutes` 默认 15 分钟。
- **真相**：1080p/5s 实测生成约 19 分钟，远超 6 分钟窗 → 1080p 必超时。
- **修法（Phase B）**：`max_poll_attempts` / 轮询总时长随分辨率上调（480p 短、1080p 长），给可配置上限；外层 wall-clock 一并对齐为外层兜底。单测断言不同分辨率的窗口数。

### B3【降级·草案+可实现构造】换皮 video-to-video 未接
- **现状**：`task.ReferenceVideoURL` 已在数据模型 + handler 存在，但 `seedanceVideoAdapter.CreateTask` **忽略它**，`content` 数组只挂 text + image_url。
- **修法（Phase B）**：产接入方案草案（请求体结构 + 字段）；镜像 image_url 模式构造 `content` 视频项 + SSRF 校验 + 单测断言 payload；**Ark 视频参考的确切 content 字段名未坐实**，标注 UNVERIFIED，真实确认列入待授权。

## 状态标签

- Real provider 链路（harness · 形态 B 直连 / 形态 A 全链路）：**LOCALLY_VERIFIED（单条真实冒烟通过）**
- Real provider 端到端产品路径（QCanvas→Sub2API→Seedance→回画布）：**NOT_VERIFIED（mock）**
- Mock-only backend：READY
- Production / Commercial：NOT_READY

## 风险边界（硬约束）

真实付费调用、填真实 key、开真实闸门、`git commit`（公司代码）/`git push`——每一项都须老板显式授权，不得无人值守自动执行。真实 key 绝不入文件/入 git/贴给 Claude，只经 env 注入。
