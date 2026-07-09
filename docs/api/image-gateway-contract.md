# Image Gateway Contract

> Status: internal / ready for integrators  
> Audience: QCanvas / TapCanvas / future internal clients  
> Last updated: 2026-07-10  
> Related: `docs/superpowers/codex-handoff/CODEX_TASK_API_CONTRACT.md`, `docs/superpowers/codex-handoff/CODEX_TASK_BILLING.md`

This document freezes Sub2API image generation/edit contracts for:

- **Nano Banana 2 (NB2)** = `gemini-3.1-flash-image-preview`
- **GPT Image 2** = `gpt-image-2`

## Official alignment (this round)

| Surface | Preferred path | Official contract notes |
|---------|----------------|-------------------------|
| NB2 | **`POST /v1/messages`** (Claude-compat → Gemini) and native **`POST /v1beta/models/{model}:generateContent`** | Pass through `imageConfig` / `responseModalities`; official `imageSize` values `"512"` / `"1K"` / `"2K"` / `"4K"`; billing normalizes `"512"` → `0.5K`; retain `thoughtSignature` across image-edit rounds (clean only on account switch / binding loss). |
| GPT Image 2 | **`POST /v1/images/generations`** and **`POST /v1/images/edits`** | Model id `gpt-image-2` is in OpenAI default model constants. OpenAI bills **token-based** (not a flat per-image list price). Packaged pricing includes a `gpt-image-2` entry; if that key is missing at runtime, pricing lookup falls back **`gpt-image-2` → `gpt-image-1.5` → `gpt-image-1`**. |
| Jimeng / 即梦 | **Not in Sub2API this round** | Out of scope for the dual-surface alignment pass; do not assume a Sub2API route. |

Kling / 可灵 video is live-but-gated on the video gateway (JWT AK+SK + tiny_real/production); see [video-gateway-contract.md](./video-gateway-contract.md) §4.2 and [qcanvas-integration-guide.md](./qcanvas-integration-guide.md). Real smoke remains `blocked: awaiting AK/SK`. LLM productization is out of scope here.

## 1. Endpoints

| Scenario | Method | Path | Auth | Notes |
|----------|--------|------|------|-------|
| Claude messages compat (NB2 preferred) | POST | `/v1/messages` | `Authorization: Bearer <api-key>` | Claude-shaped body; Gemini image gen/edit via platform routing. |
| Gemini native (NB2) | POST | `/v1beta/models/{model}:generateContent` | `Authorization: Bearer <api-key>` | Native Gemini SDK/CLI passthrough. |
| OpenAI Images (GPT Image 2) | POST | `/v1/images/generations` | `Authorization: Bearer <api-key>` | OpenAI platform group. |
| OpenAI Images Edit (GPT Image 2) | POST | `/v1/images/edits` | `Authorization: Bearer <api-key>` | OpenAI platform group. |

`/v1/messages` routes by the API key's group platform. Gemini image capability requires the group to allow image generation and the model/request to be recognized as image intent.

## 2. `/v1/messages` request schema (NB2)

Compatible with Claude messages, with image generation config accepted at the top level or under `generationConfig` and mapped into Gemini `generationConfig`:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `model` | string | yes | Prefer `gemini-3.1-flash-image-preview`. |
| `max_tokens` | number | yes | Claude-compat field. |
| `messages` | array | yes | Claude messages array. |
| `imageConfig` | object | no | Top-level compat; mapped to Gemini `generationConfig.imageConfig`. |
| `generationConfig.imageConfig` | object | no | Native shape; passed through to Gemini. |
| `responseModalities` | array | no | Top-level compat; mapped to Gemini `generationConfig.responseModalities`. Image gen should send `["TEXT", "IMAGE"]`. |
| `generationConfig.responseModalities` | array | no | Native shape; passed through to Gemini. |
| `stream` | bool | no | Streaming and non-streaming both count images from response `inlineData`. |

### imageConfig

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `aspectRatio` | string | no | Supported: `1:1`, `3:2`, `2:3`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9`, `1:4`, `4:1`, `1:8`, `8:1`, `9:21`, … |
| `imageSize` | string | no | Official Gemini values: `"512"` / `"1K"` / `"2K"` / `"4K"`. Callers should send these as-is. |

Billing normalization:

- `"512"`, `"512px"`, `"0.5K"` → `0.5K`
- `"1K"` / `"1024px"` → `1K`
- `"2K"` / `"2048px"` → `2K`
- `"4K"` / `"4096px"` → `4K`
- Unknown values default to `2K` billing with a warning log

### Image input

Claude content blocks support base64:

```json
{
  "type": "image",
  "source": {
    "type": "base64",
    "media_type": "image/png",
    "data": "<base64>"
  }
}
```

URL sources are also supported and converted to Gemini `fileData`:

```json
{
  "type": "image",
  "source": {
    "type": "url",
    "url": "https://example.invalid/reference.png",
    "media_type": "image/png"
  }
}
```

URL sources must pass SSRF/allowlist checks. Gemini allows up to 14 reference images per request; integrators should enforce limits client-side.

Media URL allowlist prefers `SUB2API_MEDIA_URL_ALLOWLIST`; if unset, falls back to legacy `SUB2API_VIDEO_URL_ALLOWLIST`. Image and video share this allowlist.

## 3. `/v1/messages` example

```json
{
  "model": "gemini-3.1-flash-image-preview",
  "max_tokens": 8192,
  "imageConfig": {
    "aspectRatio": "1:1",
    "imageSize": "512"
  },
  "responseModalities": ["TEXT", "IMAGE"],
  "messages": [
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "Generate a clean product concept image."},
        {
          "type": "image",
          "source": {
            "type": "url",
            "url": "https://example.invalid/reference.png",
            "media_type": "image/png"
          }
        }
      ]
    }
  ]
}
```

Key structure forwarded to Gemini:

```json
{
  "generationConfig": {
    "imageConfig": {
      "aspectRatio": "1:1",
      "imageSize": "512"
    },
    "responseModalities": ["TEXT", "IMAGE"]
  },
  "contents": [
    {
      "role": "user",
      "parts": [
        {"text": "Generate a clean product concept image."},
        {
          "fileData": {
            "fileUri": "https://example.invalid/reference.png",
            "mimeType": "image/png"
          }
        }
      ]
    }
  ]
}
```

## 4. Gemini native path (`/v1beta`)

`POST /v1beta/models/{model}:generateContent`

Integrators may send a native Gemini body. Sub2API keeps `thoughtSignature` for multi-turn image edit continuity, and only replaces/cleans signatures when sticky-session account switch or binding loss requires a retry recovery.

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {"text": "Generate an image."}
      ]
    }
  ],
  "generationConfig": {
    "responseModalities": ["TEXT", "IMAGE"],
    "imageConfig": {
      "aspectRatio": "16:9",
      "imageSize": "1K"
    }
  }
}
```

Multi-turn edit notes:

- Response `thoughtSignature` is part of Gemini session continuity.
- Clients must echo related signed parts on the next turn.
- Empty-part filtering does **not** drop image/thought-signature parts.

## 5. GPT Image 2 (`/v1/images/*`)

Use OpenAI Images endpoints with model `gpt-image-2` (also listed in Sub2API OpenAI default models).

- Generations: `POST /v1/images/generations`
- Edits: `POST /v1/images/edits`

Pricing notes:

- Official OpenAI GPT Image 2 billing is **token-based** (text input / image input / image output tokens).
- Packaged LiteLLM-style table includes `gpt-image-2` token rates for lookup.
- If `gpt-image-2` is absent from the loaded price table, `GetModelPricing` falls back in order: **`gpt-image-2` → `gpt-image-1.5` → `gpt-image-1`** (never to a text chat model).
- Channel/group image overrides still win when configured.

## 6. Response and image counting (NB2)

Gemini image output is counted from `inlineData` image parts. Sub2API does not hard-code 1 image; it counts actual returned image parts:

- non-streaming: count `inlineData` image parts in the final JSON body
- streaming: aggregate `inlineData` image parts across the stream
- native and `/v1/messages` compat share the same counting semantics

Integrators should parse `content` / `parts` / `inlineData` per upstream contract and must not assume a single image per response.

## 7. Error codes

| HTTP | type / reason | When |
|------|---------------|------|
| 400 | `invalid_request_error` or `VALIDATION_ERROR` | Invalid JSON, field type mismatch, upstream rejection |
| 400 | URL validation error | URL source failed SSRF/allowlist checks |
| 403 | `permission_error` | Group disallows image generation or model not allowed |
| 404 | `not_found_error` | Non-OpenAI platform calling `/v1/images/*`, or unsupported endpoint |
| 429 | `rate_limit_error` | Image concurrency slot exhausted or upstream rate limit |
| 502/503 | upstream error | Upstream unavailable, timeout, or response parse failure |

## 8. NB2 billing

Model: `gemini-3.1-flash-image-preview`

| imageSize | Normalized tier | Standard price / image |
|-----------|-----------------|------------------------|
| `512` | `0.5K` | `$0.045` |
| `1K` | `1K` | `$0.067` |
| `2K` | `2K` | `$0.101` |
| `4K` | `4K` | `$0.151` |

Other rules:

- Text/image input: `$0.50 / 1M tokens`
- Text output: `$3.00 / 1M tokens`
- Image output token rate: `$60.00 / 1M image tokens` (code path bills per-image table above)
- `usage_logs.media_type` = `image` for image jobs
- `usage_logs.image_count` = actual output image count
- `total_cost` is model raw cost; `actual_cost` applies user/group rate (`rate=1` → equal)

## 9. Integrator checklist

- Size pickers should expose only `512` / `1K` / `2K` / `4K` to avoid unknown → `2K` billing fallback.
- Multi-image outputs must be displayed/downloaded by actual `inlineData` count.
- Multi-turn edits must preserve and return `thoughtSignature`, or upstream may reject follow-ups.
- URL image inputs go through SSRF/allowlist; confirm product CDN hosts are allowlisted before production.
- Prefer NB2 via `/v1/messages` or `/v1beta`; prefer GPT Image 2 via `/v1/images/*`.
- Do not expect Jimeng/即梦 on Sub2API in this round.
