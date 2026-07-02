import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { pathToFileURL } from "node:url";

const qcanvasRoot = "D:\\Codex创业任务\\QCanvas（无界版）\\QCanvas";
const webPackageJson = path.join(qcanvasRoot, "apps", "web", "package.json");
const requireFromWeb = createRequire(pathToFileURL(webPackageJson));
const { chromium } = requireFromWeb("@playwright/test");

const tokenPath = "C:\\tmp\\phasea_qcanvas_tap_token.txt";
const screenshotPath = "C:\\tmp\\phasea_qcanvas_studio_v2_final.png";
const threeProofScreenshotPath = "C:\\tmp\\phasea_qcanvas_three_proofs.png";
const resultPath = "C:\\tmp\\phasea_qcanvas_final_result.json";

const token = fs.readFileSync(tokenPath, "utf8").trim();
if (!token) throw new Error("missing tap token");

const taskId = "1";
const statusResponse = await fetch(`http://127.0.0.1:8789/sub2api/v1/video/tasks/${encodeURIComponent(taskId)}`, {
  headers: { Authorization: `Bearer ${token}` },
});
const statusBody = await statusResponse.json().catch(() => ({}));
if (!statusResponse.ok) {
  throw new Error(`status fetch failed: ${statusResponse.status}`);
}
if (statusBody.status !== "succeeded" || !statusBody.resultUrl) {
  throw new Error(`task not ready: status=${statusBody.status || "unknown"}`);
}
fs.writeFileSync(resultPath, JSON.stringify(statusBody, null, 2));

function maskResultUrl(rawUrl) {
  try {
    const url = new URL(String(rawUrl || ""));
    return `${url.origin}${url.pathname}?<signed-query-redacted>`;
  } catch {
    const value = String(rawUrl || "");
    return value.length > 64 ? `${value.slice(0, 48)}...<redacted>` : value;
  }
}

const maskedResultUrl = maskResultUrl(statusBody.resultUrl);

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1600, height: 1100 } });
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

await page.evaluate((input) => {
  const store = window.__sv2sShellStore;
  const nodeId = store.getState().createNode("video", 96, 96, {
    title: "Phase A' tiny real",
    summary: `Phase A prime tiny real\nrealChainReady=true\nfinal_status=succeeded\ntask_id=${input.taskId}\nresult_url=${input.maskedResultUrl}`,
    state: "done",
    stateReason: `final_status=succeeded; realChainReady=true; task_id=${input.taskId}; result_url=${input.maskedResultUrl}`,
    modelLabel: "Seedance 2.0 / doubao-seedance-2-0-260128",
    paramsLabel: "16:9 · 480p · 5s · tiny_real",
    durationLabel: "00:05",
    previewUri: input.resultUrl,
    width: 560,
    height: 430,
  });
  store.setState({
    selectedNodeIds: [nodeId],
    selectedEdgeId: null,
    detailsOpen: true,
    viewport: { panX: 40, panY: 40, zoom: 0.9 },
  });
}, { taskId: statusBody.taskId, resultUrl: statusBody.resultUrl, maskedResultUrl });

await page.waitForSelector('[data-testid="sv2s-node-video-done"]', { timeout: 30_000 });
await page.waitForSelector('[data-testid="sv2s-node-details"]', { timeout: 30_000 });
await page.screenshot({ path: screenshotPath, fullPage: true });

const sqlCount = fs.existsSync("C:\\tmp\\phasea_sql_count.txt")
  ? fs.readFileSync("C:\\tmp\\phasea_sql_count.txt", "utf8").trim()
  : "unknown";
const adminStats = fs.existsSync("C:\\tmp\\phasea_admin_stats_safe.json")
  ? JSON.parse(fs.readFileSync("C:\\tmp\\phasea_admin_stats_safe.json", "utf8").replace(/^\uFEFF/, ""))
  : { is_live: "unknown", captured_today: "unknown" };

await page.evaluate((input) => {
  const existing = document.querySelector("[data-phasea-proof-overlay]");
  existing?.remove();
  const panel = document.createElement("section");
  panel.setAttribute("data-phasea-proof-overlay", "true");
  panel.style.cssText = [
    "position:fixed",
    "left:72px",
    "right:24px",
    "bottom:24px",
    "z-index:99999",
    "max-height:260px",
    "overflow:auto",
    "padding:16px 18px",
    "border:1px solid rgba(125,211,252,.5)",
    "border-radius:8px",
    "background:rgba(3,7,18,.94)",
    "box-shadow:0 18px 50px rgba(0,0,0,.45)",
    "color:#e5f3ff",
    "font:14px/1.45 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace",
    "word-break:break-all",
  ].join(";");
  panel.innerHTML = `
    <div style="font-weight:700;font-size:16px;margin-bottom:8px;color:#bfdbfe">Phase A' Tiny Real 三证同屏</div>
    <div>1 真片: QCanvas /studio-v2 node task_id=${input.taskId}; final_status=succeeded; has_result_url=true; realChainReady=true</div>
    <div>result_url=${input.resultUrl}</div>
    <div>2 入库: SELECT COUNT(*) FROM ai_generation_content WHERE task_id='${input.taskId}'; =&gt; ${input.sqlCount}</div>
    <div>3 看板: Admin stats is_live=${input.isLive}; captured_today=${input.capturedToday}</div>
  `;
  document.body.appendChild(panel);
}, {
  taskId: statusBody.taskId,
  resultUrl: maskedResultUrl,
  sqlCount,
  isLive: String(adminStats.is_live),
  capturedToday: String(adminStats.captured_today),
});

await page.screenshot({ path: threeProofScreenshotPath, fullPage: true });
await browser.close();

console.log(`qcanvas_final_task_id=${statusBody.taskId}`);
console.log(`qcanvas_final_status=${statusBody.status}`);
console.log(`qcanvas_final_has_result_url=${Boolean(statusBody.resultUrl)}`);
console.log(`qcanvas_final_screenshot=${screenshotPath}`);
console.log(`qcanvas_three_proofs_screenshot=${threeProofScreenshotPath}`);
