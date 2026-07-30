# HC-ATOM Unified Chain Goal

## Goal

Deliver a disabled-by-default, fake-contract-verified chain:

`QCanvas -> Sub2API -> HC-ATOM -> normalized result -> QCanvas`

for text, synchronous image, asynchronous image, standard video, and Seedance V3 video.

## Required outcomes

- One Sub2API HC model catalog maps each enabled model to an explicit protocol.
- HC media credentials remain encrypted and separate from HC video credentials.
- QCanvas receives only authorized, priced, adapter-ready models.
- No silent fallback to another provider.
- No provider key appears in logs, DTOs, traces, or frontend state.
- Existing HC async-image and Seedance V3 implementations are reused.

## Boundaries

- No real or paid provider request in implementation or verification.
- Deployment feature flags remain off.
- No unrelated cleanup, reset, rebase, or broad refactor.

## Acceptance

- Table-driven fake transport tests cover all enabled catalog entries.
- Contract tests cover chat-completions, messages, sync image, async image, video v1, and video v3.
- API key group isolation is enforced for media and video.
- `docs/reviews/LATEST_REVIEW_PACKAGE.html` records commands, results, risks, rollback, and current status.
