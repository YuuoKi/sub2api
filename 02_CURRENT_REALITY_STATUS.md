# Sub2API Current Reality Status

更新时间：2026-06-16 Asia/Shanghai

## 结论

当前状态：**真实链路 · 单条已验证 · 局部 READY**。

- 真实 Seedance 2.0 链路（形态 B：直连 adapter 结构体）已用**一条真实计费调用**端到端验证通过。
- 这是**局部 READY**，**不是**产品 READY、**不是**团队 READY、**不是** commercial READY。
- 形态 A（经 worker / DB / 计费 / Hono 的产品路径）尚未真实端到端验证。

> 本轮只验证了"adapter 结构体直连真实 Ark"的最小链路（形态 B），证明 provider 适配器在真实网络下可用、四字段名坐实、产物可播、密钥不泄漏。产品化路径（worker / DB / 计费 / Hono 预算门）的真实验证留待后续。

## 已验证事实（形态 B · 单条真实冒烟 · 物证在本会话）

- **链路连通**：形态 B 全链路通——`seedanceVideoAdapter` → 本机代理（Clash 7890）→ `ark.cn-beijing.volces.com/api/v3`（cn-beijing 火山仅经家服务器规则代理可达，fake-ip）。
- **HTTP 全绿**：1 次 create + 最多 30 次 poll，**全部 HTTP 200**；最终任务状态 `succeeded`。
- **四字段名坐实**（首条真实响应逐一核对回填）：create 响应任务 id 在顶层 `id`；poll 响应为 `id` / `status` / 视频地址在嵌套 `content.video_url`。
- **产物可播**：720p、16:9、5s、24fps、带音频，可正常播放。
- **真实 key 未泄漏**：开 socket 前脱敏自检（裸 / `Bearer ` / JSON 三形态）通过；审计日志、API 响应、DB、错误消息均无明文 key。
- **成本可控**：本条实际花费为个位数人民币量级，落在 ~¥200 硬预算封顶内（精确金额见「待复核」）。
- **退弹复位**：冒烟后所有 env 已复位，普通 `go test ./...` 恢复无真实调用能力；一次性脱敏审计日志按约定清理。

## 局部 / 待复核

- **形态 A 未验证**：经 worker / DB / 每日限额 / 团队额度扣费 / Hono 预算门的**产品路径**尚未真实端到端跑通；本轮形态 B 直连 adapter——不写 DB、不计费、不走 Hono。
- **账单精确金额待复核**：需以真实火山方舟账单核对单条实际计费并回填精确数字（当前仅"个位数 ￥ 量级"的现场估计）。
- **生成时长待复核**：单样本观察到端到端生成约 ~170s，**单点数据**，需更多样本确认是否稳定 / 受参数影响。
- **请求侧 resolution / aspect_ratio 未测**：本条未对请求侧分辨率 / 宽高比做参数矩阵验证。
- **响应宽高比字段名不一致**：真实响应宽高比字段为 `ratio`，而 adapter 误假定为 `aspect_ratio`——字段对齐待修。

## Follow-up（按优先级）

- **VA2**：worker 轮询无上限——产品 worker 缺 per-task poll 上限（形态 B 手动循环有硬上限 30，产品路径没有）。补 per-task 轮询 / 超时上限。
- **VA1**：Hono 预算门 + 计费——补 Hono 侧预算硬上限与真实计费扣减（现 `cost_estimate` 仅记录、从不校验）。
- **字段对齐**：响应 `ratio` vs adapter `aspect_ratio`；按真实响应回填字段映射。
- **分钟级 / 参数矩阵**：补 duration / resolution / aspect_ratio 等请求参数的真实验证。

## Deploy / 改动归档状态（本轮候选，未部署、未 commit、未 push）

- 形态 B 真实冒烟 harness（`backend/internal/service/video_gateway_realsmoke_test.go`）：build tag `realsmoke` + run-flag + 真实 key 三层关断，默认 inert（不开 socket、不读 key）；归档候选。
- 安全加固（`video_gateway_redact.go` / `video_gateway_security_test.go`）：补 VIDEO-LOCAL 不透明 token 脱敏 pass（闭合对抗发现 `redact-gap-opaque-token`）+ 对应测试；归档候选。
- Day0 / WSL 启动脚本与 compose：本地 / LAN 候选，隔离，不自动执行。**注意 `deploy/docker-compose.wsl.yml` 含硬编码本地 dev 口令（DB / Redis / Admin，非 provider key），prod 变体 `docker-compose.wsl.prod.yml` 已改 `${ENV}` 插值——是否入 git 由老板 commit 前裁量。**
- 备份脚本（`deploy/backup.sh`）：附条件提交候选。

## 当前状态标签

- Real provider 链路（形态 B · 直连 adapter）：**LOCALLY_VERIFIED（单条真实冒烟通过）**。
- Real provider 产品路径（形态 A · worker / DB / 计费 / Hono）：**NOT_VERIFIED**。
- Mock-only backend：READY（不变）。
- Production：NOT_READY。
- Commercial：NOT_READY。

## 风险边界（硬约束）

- 真实付费调用、填入真实 key、打开真实闸门、`git commit` / `git push`——**每一项都须老板显式授权**，不得自动执行。
- 真实 key 绝不入文件 / 入 git / 贴给 Claude；只经 env 注入，进程读取后从不打印。
