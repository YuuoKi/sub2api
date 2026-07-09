# QCanvas ↔ Sub2API Integration Guide

> Status: internal / current entry  
> Audience: QCanvas / TapCanvas integrators and agents  
> Last updated: 2026-07-10  
> Related contracts: [video-gateway-contract.md](./video-gateway-contract.md), [image-gateway-contract.md](./image-gateway-contract.md)

## 1. Product model (locked)

One middle layer (**Sub2API**), two product APIs:

| Product surface | Shape | Primary paths |
|-----------------|-------|---------------|
| **Video API** | Async create → poll | `POST/GET /v1/video/tasks` |
| **Image API** | Sync request/response | NB2: `POST /v1/messages` or `/v1beta/...:generateContent`; GPT Image 2: `POST /v1/images/generations` / `/edits` |

LLM productization is out of scope for this guide.

## 2. Call chain (hard boundary)

```text
QCanvas / TapCanvas browser
  → QCanvas hono-api (user session)
    → Authorization: Bearer <employee API key>
      → Sub2API /v1/* product surface
```

Rules:

- Browser must **not** call Sub2API admin JWT APIs (`/api/v1/admin/*`).
- Browser must **not** hold Sub2API admin credentials.
- Provider credentials stay server-side (Sub2API and/or hono env).
- QCanvas web talks to hono; hono talks to Sub2API with `SUB2API_BASE_URL` + `SUB2API_API_KEY`.

## 3. Allowed vs forbidden routes

| Allowed for canvas product path | Forbidden for canvas web |
|---------------------------------|--------------------------|
| `/v1/video/tasks` (+ cancel, providers) | `/api/v1/admin/*` |
| `/v1/generation-content/:task_id/adoption` | Browser → Sub2API with admin JWT |
| `/v1/messages`, `/v1beta/...` (NB2) | Direct provider keys in browser |
| `/v1/images/generations`, `/v1/images/edits` (GPT Image 2) | |

## 4. Video: Seedance 2.0 modes

Three mutually exclusive modes (official Ark / Sub2API):

| Mode | `task_type` | content roles |
|------|-------------|---------------|
| Text-to-video | `text_to_video` | `text` only |
| First/last frame | `image_to_video` | `first_frame` (+ optional `last_frame`) |
| Omnimodal reference | `reference_to_video` | `reference_image` / `reference_video` / `reference_audio` |

**Mutex:** never mix `first_frame`/`last_frame` with any `reference_*` in one request.

Studio V2 policy (2026-07-09):

- If first/last frame slots are set → frame mode only (global reference images are dropped from `content[]`).
- If only global reference images → `reference_to_video`.
- Hono Zod rejects mixed frame+reference payloads with 400.

Stable response fields for QCanvas: `id`, `status`, `result_url`, `error_message`, `provider`. Prefer `last_frame_url` when `return_last_frame=true`.

### Duration

- Seedance production: `-1` (auto) or `4..15` seconds.
- Sub2API **explicitly sends** `duration: -1` to Ark (does not omit it).
- Tiny smoke trial still caps duration to 1..5 seconds.

### Kling / 可灵

Kling is **callable** on the API-key video gateway via the same fail-closed gates as Seedance (`tiny_real` trial or `production_authorized` production). It is no longer disabled/skeleton.

#### Credential config (admin)

1. Open Sub2API admin → **Video providers** (or Key Vault).
2. Select / create a `provider=kling` account.
3. Enter **Access Key** + **Secret Key** (dual-key UI; Secret Key never echoed).
4. Save → server packs `auth_mode=kling_aksk` blob and mints outbound JWT at call time.
5. Set metadata gates as needed: `single_smoke_authorized` / `real_smoke_authorized` for tiny trial, or `production_authorized` for production.
6. Ensure env: `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`, redacted event log path, media URL allowlist.

#### Model map (catalog → Sub2API / upstream)

| QCanvas / request model | Sub2API allowlist input | Upstream `model_name` |
|-------------------------|-------------------------|------------------------|
| `kling-v1` | `kling-v1` | `kling-v1` |
| `kling-2.6-pro` / `kling-v2-6` / `kling-3.0` | same aliases | `kling-v2-6` |
| `kling-3.0-omni` | `kling-3.0-omni` | `kling-v3-omni` (omni endpoint) |
| `kling-o1` | `kling-o1` | `kling-video-o1` (omni endpoint) |
| `kling-video-extend` | routing alias | extend endpoint (`model_name` base `kling-v1`) |
| `kling-avatar` / `kling-lip-sync` | routing alias | avatar endpoint |

Duration: **5 or 10** only; tiny smoke without production auth → **5** only.

Studio ARM (when Kling catalog id selected + armed): dispatch `provider=kling` + `trialMode=tiny_real` + `allowRealCalls=true`.

**Real smoke status:** `blocked: awaiting AK/SK` — official Kling Access Key / Secret Key are not available yet. Do not treat unit/fixture green as paid upstream proof.

## 5. Image: NB2 + GPT Image 2

| Model | Preferred Sub2API path | Notes |
|-------|------------------------|-------|
| Nano Banana 2 (`gemini-3.1-flash-image-preview`) | `/v1/messages` or `/v1beta` | Official `imageSize`: `"512"` / `"1K"` / `"2K"` / `"4K"`; billing maps `"512"` → `0.5K` |
| GPT Image 2 (`gpt-image-2`) | `/v1/images/generations` / `/edits` | OpenAI platform group; packaged pricing includes `gpt-image-2` with fallback to `1.5` → `1` |
| Jimeng / 即梦 | **Not in Sub2API** | Out of scope this round |

QCanvas may still vendor-direct some image/LLM calls today; forcing canvas image traffic through Sub2API is a later workstream. This guide freezes the **middle-layer** contracts so callers can integrate correctly.

## 6. Hono env checklist (no secrets)

| Env | Purpose |
|-----|---------|
| `SUB2API_BASE_URL` | Sub2API origin |
| `SUB2API_API_KEY` | Employee API key (server-side only) |
| `SUB2API_VIDEO_MOCK_GATEWAY_ENABLED` | Explicit truthy required to open real fetch path |
| `SUB2API_VIDEO_REAL_SMOKE_ENABLED` / `SUB2API_REAL_HUMAN_AUTHORIZED` / `SUB2API_REAL_*` | Fail-closed real Seedance **and Kling** gates |

Unset gateway config → dry-run / blocked, not silent production dispatch.

## 7. Next round (explicit backlog)

1. Obtain official Kling AK/SK → configure admin dual-key → run paid tiny_real smoke  
2. Studio omnimodal reference video/audio UI  
3. Force QCanvas image path through Sub2API Image API  
4. Jimeng into Sub2API (product decision)  
5. LLM product surface (Kimi / Doubao preferred)

## 8. Verification pointers

- Video contract tests: `backend/internal/service/video_gateway_content_contract_test.go`
- QCanvas mutex: `studioV2RealTaskAdapter.ts` + `sub2api.schemas.ts`
- Image contract: [image-gateway-contract.md](./image-gateway-contract.md)
