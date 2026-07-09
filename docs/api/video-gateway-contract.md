# Video Gateway Contract

> Status: internal / ready for integrators  
> Audience: QCanvas / TapCanvas / future internal clients  
> Last updated: 2026-07-10  
> Related: `docs/superpowers/codex-handoff/CODEX_TASK_API_CONTRACT.md`, `docs/superpowers/codex-handoff/CODEX_TASK_BILLING.md`

This document freezes Sub2API video gateway contracts. HTTP responses wrap a single envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

Error response:

```json
{
  "code": 400,
  "message": "content item type must be text, image_url, video_url, or audio_url",
  "reason": "VIDEO_INVALID_CONTENT"
}
```

## 1. 绔偣

| 鍦烘櫙 | Method | Path | Auth | 璇存槑 |
|------|--------|------|------|------|
| API Key 鍒涘缓瑙嗛浠诲姟 | POST | `/v1/video/tasks` | `Authorization: Bearer <api-key>` | QCanvas 浼樺厛浣跨敤銆?|
| API Key 鏌ヨ浠诲姟 | GET | `/v1/video/tasks/{id}` | `Authorization: Bearer <api-key>` | 杞浠诲姟鐘舵€佷笌缁撴灉銆?|
| API Key 鍙栨秷浠诲姟 | POST | `/v1/video/tasks/{id}/cancel` | `Authorization: Bearer <api-key>` | 浠呴潪缁堟€佷换鍔″彲鍙栨秷銆?|
| 鐢ㄦ埛鍒涘缓瑙嗛浠诲姟 | POST | `/api/v1/video/tasks` | 鐧诲綍鎬?JWT | 绠＄悊鍙?鍐呴儴鐢ㄦ埛璺緞銆?|
| 鐢ㄦ埛鏌ヨ浠诲姟 | GET | `/api/v1/video/tasks/{id}` | 鐧诲綍鎬?JWT | 杩斿洖鍚屼竴浠诲姟瀵硅薄銆?|

## 2. 鍒涘缓璇锋眰 Schema

`POST /v1/video/tasks`

| 瀛楁 | 绫诲瀷 | 蹇呭～ | 璇存槑 |
|------|------|------|------|
| `provider` | string | no | API Key path accepts `mock` / `seedance` / `kling`. Real providers require credentials + smoke/production gates (see §4.1 / §4.2). |
| `task_type` | string | 鏄?| `text_to_video` / `image_to_video` / `reference_to_video`銆傛樉寮?`content[]` 浼氭帹瀵兼ā寮忥紝涓嶅尮閰嶆椂鎶?400銆?|
| `model` | string | 鍚?| provider 妯″瀷锛屽 `doubao-seedance-2-0-260128`銆?|
| `prompt` | string | 鏄?| 鏂囨湰鎻愮ず璇嶏紝鏈€闀?8000銆傛湇鍔＄浼氭妸瀹冧綔涓?`content[0]` 鐨?`text` 鏉＄洰淇濆瓨鍜岃浆鍙戯紝闄ら潪璇锋眰宸叉樉寮忎紶 text 鏉＄洰銆?|
| `negative_prompt` | string | 鍚?| 鍙嶅悜鎻愮ず璇嶏紝鏈€闀?4000銆?|
| `reference_image_url` | string | 鍚?| 鏃у绾﹀吋瀹瑰瓧娈点€傜湡瀹?http(s) URL 浼氳杞负 `content[]` 鐨?`image_url`銆?|
| `reference_video_url` | string | 鍚?| 鏃у绾﹀吋瀹瑰瓧娈点€傜湡瀹?http(s) URL 浼氳杞负 `content[]` 鐨?`video_url`銆?|
| `content` | array | 鍚?| Seedance/Ark 璇箟鐨勫妯℃€?content 鏁扮粍锛岃涓嬭〃銆?|
| `aspect_ratio` | string | 鍚?| Ark `ratio`锛屽 `16:9`銆乣9:16`銆乣1:1`銆?|
| `duration` | int | 鍚?| Seedance 2.0 涓?`-1` 鑷姩鎴?`4..15` 绉掋€傞潪 Seedance 鍏煎鏃?`1..60`銆?|
| `resolution` | string | 鍚?| `480p` / `720p` / `1080p`锛汼eedance 2.0 fast 涓嶆敮鎸?`1080p`銆?|
| `generate_audio` | bool | 鍚?| Ark 椤跺眰鍙傛暟锛岄粯璁ょ敱 provider 鍐冲畾銆?|
| `watermark` | bool | 鍚?| Ark 椤跺眰鍙傛暟銆?|
| `camera_fixed` | bool | 鍚?| Ark 椤跺眰鍙傛暟銆?|
| `return_last_frame` | bool | 鍚?| 鎴愬姛鍚庤姹?provider 杩斿洖灏惧抚鍥撅紝渚涚画鎷嶄娇鐢ㄣ€?|

### content[] Item

| 瀛楁 | 绫诲瀷 | 蹇呭～ | 璇存槑 |
|------|------|------|------|
| `type` | string | 鏄?| `text` / `image_url` / `video_url` / `audio_url`銆?|
| `role` | string | 鏉′欢蹇呭～ | 鍥剧墖銆佽棰戙€侀煶棰戞潯鐩缓璁樉寮忎紶銆?|
| `url` | string | 鏉′欢蹇呭～ | media URL锛屽繀椤婚€氳繃 SSRF/allowlist 鏍￠獙銆?|
| `text` | string | 鏉′欢蹇呭～ | `type=text` 鏃朵娇鐢ㄣ€?|

| type | role 鍙€夊€?| 闄愬埗 |
|------|-------------|------|
| `text` | 涓嶄紶 | 鏈€澶?1 鏉°€?|
| `image_url` | `first_frame` / `last_frame` / `reference_image` | 鍙傝€冨浘鏈€澶?9 寮狅紱棣栧熬甯фā寮忔渶澶?1 寮犻甯?+ 1 寮犲熬甯с€?|
| `video_url` | `reference_video` | 鏈€澶?3 娈点€傚惈姝ゅ瓧娈垫椂 `has_video_input=true`锛岃璐归€夊惈瑙嗛杈撳叆浠枫€?|
| `audio_url` | `reference_audio` | 鏈€澶?3 娈碉紱涓嶈兘鍗曠嫭浣跨敤锛屽繀椤诲悓鏃舵湁鑷冲皯 1 涓浘鐗囨垨瑙嗛鍙傝€冦€?|

## 3. 妯″紡鐭╅樀

| 妯″紡 | task_type | content 缁勫悎 | 浜掓枼瑙勫垯 |
|------|-----------|--------------|----------|
| 鏂囩敓瑙嗛 | `text_to_video` | 浠?`text`锛屾垨鏃犲獟浣?content | 涓嶈兘娣峰叆 image/video/audio銆?|
| 棣栧熬甯?鍥剧敓瑙嗛 | `image_to_video` | `first_frame`锛屽彲閫?`last_frame` | 涓嶈兘鍜?`reference_image` / `reference_video` / `reference_audio` 娣风敤銆?|
| 鍏ㄨ兘鍙傝€?| `reference_to_video` | `reference_image` / `reference_video` / `reference_audio` 浠绘剰缁勫悎锛岄煶棰戜笉鑳藉崟鐙嚭鐜?| 涓嶈兘鍜?`first_frame` / `last_frame` 娣风敤銆?|

鏃у瓧娈靛吋瀹癸細

- `reference_image_url` 鍦?`image_to_video` 涓嬭浆涓?`role=first_frame`銆?- `reference_image_url` 鍦?`reference_to_video` 涓嬭浆涓?`role=reference_image`銆?- `reference_video_url` 杞负 `role=reference_video`锛屽苟璁剧疆 `has_video_input=true`銆?- 鏃у瓧娈典粛浼氬湪鍝嶅簲涓師鏍峰洖鏄撅紝淇濊瘉 QCanvas 鏃у绾︿笉鐮村潖銆?
## 4. 鍒涘缓璇锋眰绀轰緥

```json
{
  "provider": "mock",
  "task_type": "reference_to_video",
  "model": "mock-video-v1",
  "prompt": "QCanvas contract prompt",
  "reference_image_url": "https://example.invalid/ref.png",
  "content": [
    {"type": "text", "text": "QCanvas contract prompt"},
    {"type": "image_url", "role": "reference_image", "url": "https://example.invalid/ref-a.png"},
    {"type": "video_url", "role": "reference_video", "url": "https://example.invalid/ref.mp4"},
    {"type": "audio_url", "role": "reference_audio", "url": "https://example.invalid/ref.mp3"}
  ],
  "aspect_ratio": "16:9",
  "duration": 5,
  "resolution": "720p",
  "generate_audio": false,
  "watermark": false,
  "camera_fixed": true,
  "return_last_frame": true
}
```

## 4.1 Seedance 试跑 vs 正式

当前后端区分两层 gate：

- 试跑账号：`provider_account.metadata.single_smoke_authorized=true` 或 `real_smoke_authorized=true`。必须满足 `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`、脱敏事件日志、媒体 URL allowlist、显式 Seedance model，并且 duration 只能 1..5 秒。
- 正式账号：`provider_account.metadata.production_authorized=true`。仍必须满足全局真实调用 env gate、脱敏事件日志、显式 Seedance model、媒体 URL allowlist；但跳过 1..5 秒 smoke 上限，按 Seedance 契约使用 `-1` 或 `4..15` 秒。
- API-key `/v1/video/tasks`：`provider:"seedance"` 且不带 `trial_mode` 时走正式路径，要求 `provider_account.metadata.production_authorized=true`；`trial_mode:"tiny_real"` 仍走每日 1 次试跑 gate。
- 媒体 URL allowlist 优先读取 `SUB2API_MEDIA_URL_ALLOWLIST`，未配置时 fallback 到旧 `SUB2API_VIDEO_URL_ALLOWLIST`。
- `duration=-1`（自动时长）会**显式**写入 Ark create payload 的 `duration` 字段，不再省略；`duration=0` / 未设置才省略该字段。

## 4.2 Kling / 可灵（真实适配已接入，fail-closed gate）

`provider:"kling"` 已从 disabled/skeleton 升级为**可真实调用**路径，鉴权与 gate 与 Seedance 对称，但凭证形态与时长枚举不同。

### 鉴权（JWT AK+SK）

- Admin 配置 **Access Key + Secret Key**（`auth_mode=kling_aksk`），服务端打包为版本化 blob 写入既有 `encrypted_api_key` 字段。
- 出站请求：`Authorization: Bearer <JWT>`，JWT 由 `klingMintJWT(AK, SK)` 现场签发（HS256，`iss=AK`，`exp≈now+1800`，`nbf≈now-5`），进程内缓存至 `exp-60s`，**不落库**。
- 响应与审计日志对 AK / SK / 派生 JWT 做脱敏；上游回显凭证时 fail-closed 中止。

### 模式与端点

DB `task_type` 仍限制在 `text_to_video` / `image_to_video` / `reference_to_video`。扩展端点通过 **model 别名** 或 `PricingSource` hint 选择，不扩展 DB enum：

| 模式 | 选择方式 | 上游 path |
|------|----------|-----------|
| 文生视频 t2v | `task_type=text_to_video` + 非 omni 模型 | `/v1/videos/text2video` |
| 图生视频 i2v | `task_type=image_to_video` + 非 omni 模型 | `/v1/videos/image2video` |
| 多图参考 multi | `task_type=reference_to_video` + 非 omni 模型 | `/v1/videos/multi-image2video` |
| 全能 omni | 上游模型 `kling-v3-omni` / `kling-video-o1`（见下表） | `/v1/videos/omni-video` |
| 视频延长 extend | model=`kling-video-extend` 或 `PricingSource=kling_mode:video_extend` | `/v1/videos/video-extend` |
| 数字人/口型 avatar | model=`kling-avatar` / `kling-lip-sync` 或 `PricingSource=kling_mode:avatar` | `/v1/videos/avatar` |

### Duration

- Kling **仅允许** `duration=5` 或 `10`（字符串枚举下发上游）。
- 试跑（无 `production_authorized`）：仅允许 `5`。
- 正式（`production_authorized=true`）：允许 `5` 或 `10`。

### Smoke / production gates

与 Seedance 同族 fail-closed 条件（任一不满足则不发起真实 HTTP）：

- `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`
- 账号 metadata：`single_smoke_authorized` / `real_smoke_authorized`（试跑）或 `production_authorized`（正式）
- `SUB2API_VIDEO_REDACTED_EVENT_LOG` 已配置
- 媒体 URL allowlist（`SUB2API_MEDIA_URL_ALLOWLIST` 或 fallback `SUB2API_VIDEO_URL_ALLOWLIST`）
- 显式、allowlist 内的 Kling model
- duration 规则见上

API-key `/v1/video/tasks`：

- `provider:"kling"` + `trial_mode:"tiny_real"` → 试跑路径（每日限额与 Seedance tiny trial 对称）
- `provider:"kling"` 且不带 `trial_mode` → 正式路径，要求 `production_authorized=true`

**真实付费冒烟状态：** `blocked: awaiting AK/SK`（官方 Access Key / Secret Key 尚未到位；代码与契约已就绪，配置密钥并打开上述 gate 后即可冒烟）。

### Model ID map（fail-closed）

| 请求 `model`（catalog / 别名） | 上游 `model_name` | 备注 |
|-------------------------------|-------------------|------|
| `kling-v1` | `kling-v1` | 基础 |
| `kling-2.6-pro` / `kling-v2-6` / `kling-3.0` | `kling-v2-6` | `kling-2.6-pro` 暗示 `mode=pro` |
| `kling-3.0-omni` | `kling-v3-omni` | 走 omni-video |
| `kling-o1` | `kling-video-o1` | 走 omni-video |
| `kling-video-extend` | `kling-v1`（路由别名） | 选 video-extend 端点 |
| `kling-avatar` / `kling-lip-sync` | `kling-v1`（路由别名） | 选 avatar 端点 |
| 其他 | — | `KLING_MODEL_NOT_ALLOWED` |

Kling tiny_real 请求示例：

```json
{
  "provider": "kling",
  "trial_mode": "tiny_real",
  "task_type": "text_to_video",
  "model": "kling-v1",
  "prompt": "A short cinematic shot of rain on a city street at night.",
  "duration": 5,
  "aspect_ratio": "16:9",
  "resolution": "720p"
}
```

智能画布通过 `/v1/video/tasks` 调正式 Seedance 的当前请求示例：

```json
{
  "provider": "seedance",
  "task_type": "reference_to_video",
  "model": "doubao-seedance-2-0-260128",
  "prompt": "Turn the selected canvas scene into a 10 second cinematic shot.",
  "content": [
    {"type": "text", "text": "Turn the selected canvas scene into a 10 second cinematic shot."},
    {"type": "image_url", "role": "reference_image", "url": "https://assets.example.com/canvas/ref-a.png"},
    {"type": "video_url", "role": "reference_video", "url": "https://assets.example.com/canvas/ref.mp4"}
  ],
  "aspect_ratio": "9:16",
  "duration": 10,
  "resolution": "720p",
  "generate_audio": false,
  "watermark": false,
  "return_last_frame": true
}
```

## 5. 浠诲姟鍝嶅簲 Schema

`data` 鍐呬换鍔″璞★細

| 瀛楁 | 绫诲瀷 | 璇存槑 |
|------|------|------|
| `id` | number | Sub2API 浠诲姟 ID銆俀Canvas 搴斾紭鍏堢敤瀹冧綔涓?taskId銆?|
| `provider` | string | `mock` / `seedance` / `kling`. |
| `model` | string | 瀹為檯璁板綍妯″瀷銆?|
| `task_type` | string | 浠诲姟绫诲瀷銆?|
| `prompt` | string | 鏂囨湰鎻愮ず璇嶃€?|
| `negative_prompt` | string | 鍙嶅悜鎻愮ず璇嶃€?|
| `reference_image_url` | string | 鏃у瓧娈靛洖鏄俱€?|
| `reference_video_url` | string | 鏃у瓧娈靛洖鏄俱€?|
| `content` | array | 褰掍竴鍖栧悗鐨?content 鏁扮粍銆?|
| `has_video_input` | bool | 鏄惁鍚?`video_url` 杈撳叆锛涜棰戣璐归€変环渚濊禆姝ゅ瓧娈点€?|
| `aspect_ratio` | string | 璇锋眰姣斾緥銆?|
| `duration` | number | 璇锋眰鏃堕暱銆?|
| `resolution` | string | 璇锋眰鍒嗚鲸鐜囥€?|
| `generate_audio` | bool | 鍙€夊洖鏄俱€?|
| `watermark` | bool | 鍙€夊洖鏄俱€?|
| `camera_fixed` | bool | 鍙€夊洖鏄俱€?|
| `return_last_frame` | bool | 鍙€夊洖鏄俱€?|
| `status` | string | `queued` / `submitted` / `running` / `succeeded` / `failed` / `cancelled`銆?|
| `upstream_task_id` | string | provider 浠诲姟 ID銆?|
| `result_url` | string | 鎴愬姛鍚庣殑瑙嗛 URL銆侫PI-key 鍝嶅簲涓嶅啀杈撳嚭 PascalCase `ResultURL`銆?|
| `usage.total_tokens` | number | provider 鎴愬姛鍝嶅簲涓殑鐪熷疄 token锛岀敤浜?BILLING V-2銆傛棫浠诲姟鏃犲€兼椂涓?0銆?|
| `actual_resolution` | string | provider 纭鐨勫疄闄呭垎杈ㄧ巼銆?|
| `actual_duration` | number/null | provider 纭鐨勫疄闄呮椂闀裤€?|
| `last_frame_url` | string | `return_last_frame=true` 鏃剁殑灏惧抚 URL銆?|
| `error_message` | string | 澶辫触鍘熷洜銆?|
| `cost_estimate` | number | 浠诲姟鎴愭湰銆係eedance 鎴愬姛浠诲姟鎸夌湡瀹?usage 璁¤垂銆?|
| `mock_only` | bool | API-key mock-only 杈圭晫鏍囧織銆?|
| `provider_boundary` | string | mock-only 杈圭晫璇存槑銆?|
| `real_provider_dispatch_count` | number | mock-only 涓嬪繀椤讳负 0銆?|

鎴愬姛鍝嶅簲绀轰緥锛?
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 12345,
    "provider": "mock",
    "model": "mock-video-v1",
    "task_type": "reference_to_video",
    "reference_image_url": "https://example.invalid/ref.png",
    "content": [
      {"type": "text", "text": "QCanvas contract prompt"},
      {"type": "image_url", "role": "reference_image", "url": "https://example.invalid/ref-a.png"},
      {"type": "video_url", "role": "reference_video", "url": "https://example.invalid/ref.mp4"},
      {"type": "audio_url", "role": "reference_audio", "url": "https://example.invalid/ref.mp3"}
    ],
    "has_video_input": true,
    "aspect_ratio": "16:9",
    "duration": 5,
    "resolution": "720p",
    "generate_audio": false,
    "watermark": false,
    "camera_fixed": true,
    "return_last_frame": true,
    "usage": {"total_tokens": 321},
    "actual_resolution": "720p",
    "actual_duration": 5,
    "last_frame_url": "/api/v1/video/mock-assets/12345-last-frame.png",
    "status": "succeeded",
    "result_url": "/api/v1/video/mock-assets/12345.svg",
    "error_message": "",
    "mock_only": true,
    "provider_boundary": "api-key-video-mock-only",
    "real_provider_dispatch_count": 0
  }
}
```

## 6. 閿欒鐮佽〃

| HTTP | reason | 瑙﹀彂鏉′欢 |
|------|--------|----------|
| 400 | `VALIDATION_ERROR` | JSON 缁戝畾澶辫触銆佺己灏戝繀濉瓧娈点€佹灇涓句笉鍚堟硶銆?|
| 400 | `VIDEO_INVALID_PROVIDER` | provider 涓嶅湪鍏佽鑼冨洿銆?|
| 400 | `VIDEO_INVALID_TASK_TYPE` | task_type 涓嶅湪鍏佽鑼冨洿銆?|
| 400 | `VIDEO_INVALID_CONTENT` | content 绫诲瀷/role/鏁伴噺/妯″紡/鏃堕暱/鍒嗚鲸鐜囨牎楠屽け璐ャ€?|
| 400 | `VIDEO_UNSAFE_REFERENCE_URL` | media URL 鏈€氳繃 SSRF/allowlist 鏍￠獙銆?|
| 403 | `VIDEO_API_KEY_PROVIDER_NOT_ALLOWED` | API-key mock-only 杈圭晫鎷掔粷鐪熷疄 provider銆?|
| 404 | `VIDEO_TASK_NOT_FOUND` | 浠诲姟涓嶅瓨鍦ㄦ垨璋冪敤鏂规棤鏉冭闂€?|
| 409 | `VIDEO_TASK_NOT_CANCELABLE` | 缁堟€佷换鍔′笉鍙彇娑堛€?|
| 503 | `VIDEO_MOCK_PROVIDER_UNAVAILABLE` | mock provider 涓嶅彲鐢ㄣ€?|

## 7. 璁¤垂璇存槑

- Seedance 鎴愬姛浠诲姟鎸?provider poll 鍝嶅簲 `usage.total_tokens` 璁¤垂锛歚tokens / 1,000,000 * 鍏?M tokens`銆?- `has_video_input=true` 鏃讹紝Seedance 2.0 浣跨敤鍚棰戣緭鍏ヤ环锛沗false` 鏃朵娇鐢ㄤ笉鍚棰戣緭鍏ヤ环銆?- 浠诲姟澶辫触涓嶈璐癸紝璐圭敤涓?0銆?- Seedance 浠锋牸甯佺涓?CNY锛屽綊鎬诲睍绀烘椂鎸夌郴缁熻缃?`usd_cny_rate` 鎶樼畻涓?USD銆?- 褰撳墠浠诲姟涔︽牳瀹炰环锛歋eedance 2.0 涓嶅惈瑙嗛杈撳叆 46 鍏?M tokens锛屽惈瑙嗛杈撳叆 28 鍏?M tokens锛汼eedance 2.0 fast 涓嶅惈瑙嗛杈撳叆 37 鍏?M tokens锛屽惈瑙嗛杈撳叆 22 鍏?M tokens銆?
## 8. 鎺ュ叆鏂规敞鎰?
- QCanvas 缁х画璇?`id`銆乣status`銆乣result_url`銆乣error_message`銆乣provider`锛涜繖浜涙棫濂戠害瀛楁淇濇寔涓嶅彉銆?- 涓嶈鍐嶈鍙?`ResultURL`锛孉PI-key 鍝嶅簲宸茬Щ闄よ PascalCase 閲嶅瀛楁銆?- 闇€瑕佺画鎷嶆椂锛屽垱寤鸿姹備紶 `return_last_frame=true`锛岃疆璇㈡垚鍔熷悗璇诲彇 `last_frame_url`銆?- 鏋勫缓 content 缂栬緫鍣ㄦ椂锛屽厛鎸夋ā寮忕害鏉?UI锛岄伩鍏嶆妸棣栧熬甯фā寮忓拰 reference 妯″紡娣风敤銆?
