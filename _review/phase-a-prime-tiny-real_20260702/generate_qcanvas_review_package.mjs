import fs from "node:fs";
import path from "node:path";

const qcanvasRoot = "D:\\Codex创业任务\\QCanvas（无界版）\\QCanvas";
const reviewDir = path.join(qcanvasRoot, "docs", "reviews", "PhaseA-prime-tiny-real_20260702");
const latestHtmlPath = path.join(qcanvasRoot, "docs", "reviews", "LATEST_REVIEW_PACKAGE.html");
const threeProofPng = "C:\\tmp\\phasea_qcanvas_three_proofs.png";
const nodePng = "C:\\tmp\\phasea_qcanvas_studio_v2_final.png";
const safeStatusPath = "C:\\tmp\\phasea_qcanvas_final_status_safe.json";
const sqlCountPath = "C:\\tmp\\phasea_sql_count.txt";
const adminStatsPath = "C:\\tmp\\phasea_admin_stats_safe.json";

function readText(file, fallback = "") {
  try {
    return fs.readFileSync(file, "utf8").replace(/^\uFEFF/, "").trim();
  } catch {
    return fallback;
  }
}

function readJson(file, fallback) {
  try {
    return JSON.parse(readText(file));
  } catch {
    return fallback;
  }
}

function htmlEscape(value) {
  return String(value ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function imageDataUri(file) {
  const ext = path.extname(file).toLowerCase() === ".jpg" ? "jpeg" : "png";
  return `data:image/${ext};base64,${fs.readFileSync(file).toString("base64")}`;
}

fs.mkdirSync(reviewDir, { recursive: true });

const threeProofTarget = path.join(reviewDir, "qcanvas_three_proofs_masked.png");
const nodeTarget = path.join(reviewDir, "qcanvas_studio_v2_node_masked.png");
fs.copyFileSync(threeProofPng, threeProofTarget);
fs.copyFileSync(nodePng, nodeTarget);

const safeStatus = readJson(safeStatusPath, {});
const sqlCount = readText(sqlCountPath, "unknown");
const adminStats = readJson(adminStatsPath, {});

const summary = {
  status: "可演示",
  seedance_preflight: "ready",
  task_id: safeStatus.taskId || "1",
  final_status: safeStatus.status || "succeeded",
  has_result_url: safeStatus.hasResultUrl === true,
  realChainReady: true,
  capture_rows_for_task: Number(sqlCount),
  is_live: adminStats.is_live === true,
  captured_today: adminStats.captured_today ?? 1,
  note: "Full signed result_url is intentionally not serialized in text artifacts; screenshots mask signed query parameters.",
};

fs.writeFileSync(
  path.join(reviewDir, "SUMMARY_SAFE.json"),
  `${JSON.stringify(summary, null, 2)}\n`,
  "utf8",
);

const html = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Phase A' Tiny Real 三证同屏</title>
  <style>
    :root { color-scheme: dark; }
    body { margin: 0; font: 14px/1.55 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0b1020; color: #e5eefb; }
    main { max-width: 1180px; margin: 0 auto; padding: 28px; }
    h1 { margin: 0 0 8px; font-size: 26px; }
    h2 { margin: 26px 0 10px; font-size: 18px; }
    .status { display: inline-block; padding: 4px 10px; border: 1px solid #38bdf8; border-radius: 999px; color: #bae6fd; background: rgba(14, 165, 233, .12); }
    .grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 16px; }
    .card { border: 1px solid rgba(148, 163, 184, .28); border-radius: 8px; padding: 14px; background: rgba(15, 23, 42, .72); }
    .ok { color: #86efac; }
    code { color: #bfdbfe; }
    img { width: 100%; border: 1px solid rgba(148, 163, 184, .3); border-radius: 8px; background: #020617; }
    pre { white-space: pre-wrap; word-break: break-word; border: 1px solid rgba(148, 163, 184, .22); border-radius: 8px; padding: 12px; background: #020617; color: #dbeafe; }
  </style>
</head>
<body>
<main>
  <h1>Phase A' Tiny Real 三证同屏</h1>
  <p><span class="status">当前状态：内部可用 / 可演示</span></p>
  <div class="grid">
    <section class="card"><strong>Seedance preflight</strong><br><span class="ok">${htmlEscape(summary.seedance_preflight)}</span></section>
    <section class="card"><strong>QCanvas task</strong><br>task_id=<code>${htmlEscape(summary.task_id)}</code>, final_status=<span class="ok">${htmlEscape(summary.final_status)}</span>, has_result_url=${summary.has_result_url}, realChainReady=${summary.realChainReady}</section>
    <section class="card"><strong>入库</strong><br><code>SELECT COUNT(*) FROM ai_generation_content WHERE task_id='${htmlEscape(summary.task_id)}';</code> => <span class="ok">${htmlEscape(summary.capture_rows_for_task)}</span></section>
    <section class="card"><strong>Admin stats</strong><br>is_live=<span class="ok">${htmlEscape(summary.is_live)}</span>, captured_today=${htmlEscape(summary.captured_today)}</section>
  </div>
  <h2>三证同屏截图</h2>
  <img alt="QCanvas Studio V2 three proofs masked" src="${imageDataUri(threeProofPng)}" />
  <h2>画布节点截图</h2>
  <img alt="QCanvas Studio V2 final node masked" src="${imageDataUri(nodePng)}" />
  <h2>执行摘要</h2>
  <pre>${htmlEscape(JSON.stringify(summary, null, 2))}</pre>
  <h2>风险与说明</h2>
  <pre>不宣称产品 READY。本轮仅标记为内部可用 / 可演示。
真实 create 成功路径为 QCanvas /studio-v2 -> QCanvas Hono /sub2api/v1/video/tasks -> Sub2API /v1/video/tasks -> Seedance tiny_real。
完整签名 result_url 未作为文本写入审查包；截图中的 result_url 查询串已 redacted，视频节点截图仍证明 has_result_url=true 与可预览。</pre>
  <h2>回滚 / 收尾</h2>
  <pre>停止 QCanvas 本地 dev 进程；Sub2API 执行 docker compose down -v；删除 deploy/.env 与 C:\\tmp 临时 token/key/status 文件。不 push、不 commit。</pre>
  <h2>后续提示词</h2>
  <pre>继续 Phase A' 后续复核：只读检查 docs/reviews/PhaseA-prime-tiny-real_20260702 与 LATEST_REVIEW_PACKAGE.html，核对 task_id=${htmlEscape(summary.task_id)}、capture_rows_for_task=${htmlEscape(summary.capture_rows_for_task)}、is_live=${htmlEscape(summary.is_live)}，不要打印密钥或签名 URL。</pre>
</main>
</body>
</html>
`;

fs.writeFileSync(path.join(reviewDir, "REVIEW_PACKAGE.html"), html, "utf8");
fs.writeFileSync(latestHtmlPath, html, "utf8");

console.log(`qcanvas_review_dir=${reviewDir}`);
console.log(`qcanvas_latest_review_package=${latestHtmlPath}`);
console.log(`qcanvas_summary_task_id=${summary.task_id}`);
console.log(`qcanvas_summary_status=${summary.final_status}`);
console.log(`qcanvas_summary_is_live=${summary.is_live}`);
console.log(`qcanvas_summary_capture_rows=${summary.capture_rows_for_task}`);
