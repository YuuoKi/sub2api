# Codex 任务书 · 调用标准补齐专项（P0-5 API Contract）

> 生成时间：2026-07-05  
> 前置：先读 [CODEX_START_HERE.md](./CODEX_START_HERE.md)。本文件与 [CODEX_TASK_BILLING.md](./CODEX_TASK_BILLING.md) 是姊妹篇：BILLING 管「钱算得准」，本文件管「请求接得住」。官方接口规范已由 Claude 于 2026-07-05 查实写在下方，**不需要再上网查**。  
> 背景：智能画布（QCanvas/TapCanvas）通过我们的 `/v1/video/tasks` 和 LLM 网关调用，未来还有其他系统接入。我们的封装 API 就是对外产品，参数面必须对齐上游官方能力，否则调用方发来的高级参数会被静默丢弃。

---

## 审计结论（2026-07-05，Claude 核实）

### 我们现状（证据在代码）

- 视频创建 DTO：`backend/internal/handler/video_handler.go:37-49` —— 只有单个 `reference_image_url` + 单个 `reference_video_url`，无 role、无音频、无首尾帧、无 generate_audio/watermark/return_last_frame/seed/webhook；`task_type` 存了但 Seedance 适配器**根本不读**（只看 URL 是否存在）
- Seedance 适配器：`backend/internal/service/video_gateway_adapter.go:171-221` —— content 数组只拼 text + 最多各一个 image_url/video_url，**不带 role 字段**；`duration`/`resolution`/`video_url` 字段名标注 UNVERIFIED
- 轮询响应：回显请求参数 + result_url，**不返回**上游确认的分辨率/时长/usage tokens/last_frame
- Gemini 图像：native `/v1beta` 全透传 ✅；但 `/v1/messages`（Claude 格式转 Gemini）**不转换** `imageConfig`/`responseModalities`，图片输入只支持 base64；计费侧 `extractImageSize` 只认 1K/2K/4K（未知默认 2K），imageCount **写死 1**

### 官方标准（已核实）

#### A. Seedance 2.0（火山方舟 Ark，`POST /api/v3/contents/generations/tasks`）

content 数组多模态输入，每个条目 `type` + 可选 `role`：

| type | role 可选值 | 数量上限 |
|------|-------------|----------|
| `text` | — | 1 |
| `image_url` | `first_frame` / `last_frame` / `reference_image` | 参考模式最多 **9 张**；首尾帧模式 1-2 张 |
| `video_url` | `reference_video` | 最多 **3 段**，单段 2-15s，总时长 ≤15s |
| `audio_url` | `reference_audio` | 最多 **3 段**（音频不能单独传，需至少 1 图或 1 视频） |

三种图片模式互斥：文生视频 / 首尾帧（`first_frame`+`last_frame`）/ 多模态全能参考（`reference_*`）。

顶层参数：`model`、`duration`（整数，-1 表示自动）、`ratio`（16:9/4:3/1:1/3:4/9:16/21:9…）、`resolution`（480p/720p/1080p，2.0-fast 无 1080p）、`generate_audio`（bool，默认 true）、`watermark`（bool）、`camera_fixed`（bool）、`return_last_frame`（bool，成功后返回尾帧图用于续拍）。

图片约束：URL 或 Base64 data URL，jpeg/png/webp/bmp/tiff/gif/heic/heif，宽高比 (0.4, 2.5)，边长 (300, 6000)px，单张 ≤30MB。视频 mp4/mov ≤50MB。音频 wav/mp3 ≤15MB。

轮询响应（任务完成）：`status`、`content.video_url`、**`usage.total_tokens`（计费依据，接 BILLING V-2）**、`return_last_frame=true` 时含尾帧图 URL。

**计费联动**：content 含 `video_url` 走"含视频输入"低价（2.0：28 元/M；不含 46 元/M）——所以**必须准确记录任务是否含视频输入**，BILLING V-2 依赖这个标志。

#### B. Gemini 3.1 Flash Image（nanobanana2，`generateContent`）

- `generationConfig.imageConfig`：`aspectRatio`（1:1/3:2/2:3/3:4/4:3/4:5/5:4/9:16/16:9/21:9/1:4/4:1/1:8/8:1/9:21）、`imageSize`（**"512" / "1K" / "2K" / "4K"**，大写 K，小写会被拒）
- `responseModalities: ["TEXT", "IMAGE"]`
- 多图输入：单次最多 **14 张参考图**（inlineData 或 fileData）
- 多轮编辑：依赖 `thoughtSignature` 回传（响应里 thoughts 后第一个 part 和每个 inlineData part 都带签名，客户端必须原样带回，否则报错）
- 一次响应可含多个 inlineData 输出图（按输出 token 计费，上限 32768 tokens）
- 输出 token：512→747、1K→1120、2K→1680、4K→2520（对应 BILLING V-1 的每图价）

---

## 子任务清单（A 系列，按顺序）

### A-1 视频创建契约升级：对齐 Seedance content 数组【最高优先】

1. `apiKeyVideoTaskCreateRequest` / `videoTaskCreateRequest` 新增 `content[]` 字段（type/role/url 结构，与 Ark 语义一致），**保留旧的** `reference_image_url`/`reference_video_url` 单数字段做兼容（内部转成 content 条目）。
2. 校验：三种图片模式互斥；数量上限（9 图/3 视频/3 音频）；audio 不能单独出现；role 枚举校验；所有 URL 过 SSRF/allowlist 检查（复用现有 `video_gateway_adapter.go:172-192` 逻辑）。
3. 新增顶层可选参数：`generate_audio`、`watermark`、`camera_fixed`、`return_last_frame`；`resolution` 加枚举校验（480p/720p/1080p，fast 拒 1080p）；`duration` 上限从 60 改为与 provider 能力一致（Seedance 2.0：4-15，-1 自动）。
4. `task_type` 从"摆设"变为派生校验：根据 content 组合推导模式，与传入 task_type 不符时报 400（错误信息说清楚）。
5. Seedance 适配器把 content 数组 + role **原样映射**到 Ark payload（消除现有 UNVERIFIED 注释——字段名 `duration`/`resolution`/`ratio`/`video_url` 均已按官方文档核实，就是这些名字）。
6. 落库：`video_tasks` 需要能存 content 数组（JSON 列迁移）与 `has_video_input` 布尔（计费用）。
7. 单测：模式互斥、数量上限、audio 约束、兼容旧单数字段、payload 映射快照。

验收：一条「文本 + 2 参考图 + 1 参考视频 + 1 参考音频」的请求能通过校验并生成正确的 Ark payload；旧格式请求行为不变。

### A-2 轮询响应增强：调用方拿得到结果全貌

1. 任务成功后从 Ark 轮询响应提取并落库：`usage.total_tokens`（给 BILLING V-2）、上游确认的实际分辨率/时长、`last_frame_url`（当 `return_last_frame=true`）。
2. `GET /v1/video/tasks/:id` 响应新增：`usage`（tokens）、`actual_resolution`、`actual_duration`、`last_frame_url`、`has_video_input`。
3. 删掉 API-key 响应里 PascalCase 重复字段 `ResultURL`（保留 `result_url`），在审查包里提醒 Claude 前端同步。
4. 单测：字段提取、老任务（无新字段）兼容。

验收：智能画布轮询一次拿到视频 URL + 尾帧 + 实际消耗，可直接续拍下一段。

### A-3 Gemini 图像通道对齐（/v1/messages 路径 + 计费）

1. Claude→Gemini 转换（`gemini_messages_compat_service.go`）：把请求里的 `imageConfig`（aspectRatio/imageSize）和 `responseModalities` 透传到 Gemini `generationConfig`；图片输入支持 URL source（下载转 inlineData 或转 fileData，注意大小限制与 SSRF）。
2. `extractImageSize` 支持 `"512"`（0.5K 档），并与 BILLING V-1 的四档价格打通；未知值仍默认 2K 但记 warning 日志。
3. imageCount 不再写死 1：从响应里数实际 `inlineData` 图片 part 数量（native 与 messages 两条路径都要）。
4. 多轮编辑：确认 native 透传路径不丢 `thoughtSignature`（现有 `ensureGeminiFunctionCallThoughtSignatures` 只处理 function call，检查图像签名是否被 `filterEmptyParts` 或响应转换剥掉；若剥掉则修）。
5. 单测：imageConfig 透传、512 档计费、多图计数、签名保留。

验收：通过 `/v1/messages` 发一个带 `imageConfig:{imageSize:"512"}` 的 NB2 请求，上游收到正确 generationConfig，计费按 $0.045/图、张数按实际输出数。

### A-4 契约文档固化（给智能画布和未来接入方）

1. 在 `docs/api/` 新增 `video-gateway-contract.md` 与 `image-gateway-contract.md`：完整请求/响应 schema、模式矩阵、错误码表、计费说明（引用 BILLING 任务书价表）。
2. 更新现有 QCanvas 契约测试（`video_handler_c1_contract_test.go`、`api_key_video_task_contract_v1.json`）覆盖新字段，**保证旧契约字段不破坏**（id/status/result_url/error_message/provider 不变）。

---

## 与 BILLING 任务书的依赖关系

```text
A-1 (has_video_input 标志) ──> BILLING V-2 (28 vs 46 元/M 选价)
A-2 (usage.total_tokens 落库) ──> BILLING V-2 (真实 token 计费)
A-3 (512 档 + 实际张数) ──> BILLING V-1 (四档每图价)
```

建议执行顺序：**V-1 → A-3 → A-1 → A-2 → V-2 → V-3/V-4**（若 BILLING 已在进行，完成当前子任务后插入 A 系列再回来做 V-2）。

## 硬性约束

- 同 CODEX_START_HERE：分层、depguard、`go test ./...` + `golangci-lint run ./...` 全绿、不改 `frontend/**`、不提交真实 Key
- **向后兼容是红线**：智能画布已在用旧契约，旧字段/旧行为必须原样可用，契约测试先行
- 所有外部 URL 输入必须过现有 SSRF/allowlist 检查
- 每个 A 子任务独立审查包，放 `deliverables/`，命名 `2026-07-0X-A-N-review.md`；「给 Claude 的前端说明」必填（前端要展示 content 数组编辑器、尾帧续拍、图像尺寸选择等）
