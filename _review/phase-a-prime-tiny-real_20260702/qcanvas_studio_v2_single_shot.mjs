import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { pathToFileURL } from "node:url";

const qcanvasRoot = "D:\\Codex创业任务\\QCanvas（无界版）\\QCanvas";
const webPackageJson = path.join(qcanvasRoot, "apps", "web", "package.json");
const requireFromWeb = createRequire(pathToFileURL(webPackageJson));
const { chromium } = requireFromWeb("@playwright/test");

const tokenPath = "C:\\tmp\\phasea_qcanvas_tap_token.txt";
const outPath = "C:\\tmp\\phasea_qcanvas_create_result.json";
const screenshotPath = "C:\\tmp\\phasea_qcanvas_studio_v2_pending.png";

const token = fs.readFileSync(tokenPath, "utf8").trim();
if (!token) throw new Error("missing tap token");

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
await page.addInitScript((tapToken) => {
  window.localStorage.setItem("tap_token", tapToken);
  window.localStorage.setItem("studioV2RealChainReady", "true");
  window.localStorage.setItem("studioV2RealSeedanceArmed", "true");
  window.document.cookie = `tap_token=${encodeURIComponent(tapToken)}; path=/; SameSite=Lax`;
}, token);

await page.goto("http://127.0.0.1:5173/studio-v2?studioV2RealChainReady=true&studioV2RealSeedanceArmed=true&sv2sEmpty=1", {
  waitUntil: "networkidle",
  timeout: 60_000,
});

await page.waitForFunction(() => Boolean(window.__sv2sShellStore?.getState), null, { timeout: 60_000 });

const result = await page.evaluate(async () => {
  const store = window.__sv2sShellStore;
  const prompt = "Phase A prime tiny real";
  const nodeId = store.getState().createNode("video", 120, 120, {
    title: "Phase A' tiny real",
    summary: prompt,
    state: "loading",
    stateReason: "realChainReady=true; seedance tiny_real create request in flight",
    modelLabel: "Seedance 2.0",
    paramsLabel: "16:9 / 480p / 5s",
    durationLabel: "00:05",
  });
  const tapToken = window.localStorage.getItem("tap_token") || "";
  const response = await fetch("http://127.0.0.1:8789/sub2api/v1/video/tasks", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${tapToken}`,
    },
    body: JSON.stringify({
      provider: "seedance",
      trialMode: "tiny_real",
      prompt,
      model: "seedance-2",
      params: {
        aspect: "16:9",
        quality: "480p",
        duration: 5,
        audio: false,
        count: 1,
      },
      metadata: {
        taskKind: "text_to_video",
        requestedRunMode: "real",
        phase: "phase-a-prime-tiny-real",
      },
      nodeId,
      nodeLabel: "Phase A' tiny real",
    }),
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    store.setState((prev) => ({
      nodes: {
        ...prev.nodes,
        [nodeId]: {
          ...prev.nodes[nodeId],
          state: "failed",
          stateReason: `create failed: HTTP ${response.status}`,
          blockReasons: [body?.code || body?.error || `HTTP ${response.status}`],
        },
      },
    }));
    return { ok: false, statusCode: response.status, nodeId, body };
  }
  store.setState((prev) => ({
    nodes: {
      ...prev.nodes,
      [nodeId]: {
        ...prev.nodes[nodeId],
        state: "loading",
        stateReason: `realChainReady=true; provider=seedance; task_id=${body.taskId}; final_status=${body.status}`,
        blockReasons: [],
      },
    },
  }));
  return { ok: true, statusCode: response.status, nodeId, body };
});

fs.writeFileSync(outPath, JSON.stringify(result, null, 2));
await page.screenshot({ path: screenshotPath, fullPage: true });
await browser.close();

const body = result.body || {};
console.log(`qcanvas_create_ok=${Boolean(result.ok)}`);
console.log(`qcanvas_node_id=${result.nodeId || ""}`);
console.log(`qcanvas_task_id=${body.taskId || ""}`);
console.log(`qcanvas_initial_status=${body.status || ""}`);
console.log(`qcanvas_initial_has_result_url=${Boolean(body.resultUrl)}`);
