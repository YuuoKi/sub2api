#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const repoRoot = path.resolve(__dirname, '..', '..');
const projectRoot = path.resolve(repoRoot, '..', '..');
const reviewRoot = path.join(projectRoot, '03_审查包');
const customerRoot = path.join(reviewRoot, '01_客户演示包');
const internalRoot = path.join(reviewRoot, '02_内部审查包');
const providerRoot = path.join(reviewRoot, '03_真实通道接入');
const screenshotRoot = path.join(reviewRoot, '04_截图证据');
const reviewToolsRoot = path.join(reviewRoot, 'tools');

function arg(name, fallback = '') {
  const idx = process.argv.indexOf(name);
  if (idx >= 0 && process.argv[idx + 1]) return process.argv[idx + 1];
  return fallback;
}

const opts = {
  phase381: arg('--phase381', '69f648e20e6f194b08fb120c215e96e88b30e84e'),
  phase382: arg('--phase382', 'PENDING'),
  branch: arg('--branch', 'phase-3.8.2-overnight-readiness'),
  customerZip: arg('--customerZip', ''),
  internalZip: arg('--internalZip', ''),
  ready: process.argv.includes('--ready'),
};

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function write(rel, content) {
  const full = path.join(reviewRoot, rel);
  ensureDir(path.dirname(full));
  fs.writeFileSync(full, content.replace(/\n/g, '\r\n'), 'utf8');
}

function copyFile(src, dst) {
  ensureDir(path.dirname(dst));
  fs.copyFileSync(src, dst);
}

function esc(text) {
  return String(text).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function htmlPage(title, body, extraCss = '') {
  return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${esc(title)}</title>
  <style>
    :root { color-scheme: light; --ink:#172033; --muted:#5b667a; --line:#d9e2ef; --soft:#f5f8fb; --panel:#fff; --brand:#0f766e; --blue:#1559b7; --ok:#087f5b; --warn:#ad6800; --bad:#c92a2a; }
    * { box-sizing: border-box; }
    body { margin:0; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; color:var(--ink); background:var(--soft); }
    header { background:#10233f; color:white; padding:34px 42px 30px; }
    main { max-width:1180px; margin:0 auto; padding:28px 24px 56px; }
    h1 { margin:0 0 10px; font-size:30px; letter-spacing:0; }
    h2 { margin:30px 0 12px; font-size:21px; }
    h3 { margin:0 0 8px; font-size:16px; }
    p, li { line-height:1.68; }
    a { color:var(--blue); font-weight:700; text-decoration:none; }
    a:hover { text-decoration:underline; }
    table { width:100%; border-collapse:collapse; background:white; border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    th, td { padding:11px 12px; border-bottom:1px solid #e8edf4; text-align:left; vertical-align:top; }
    th { background:#eef3f9; color:#344054; }
    code { background:#eef3f9; padding:2px 5px; border-radius:4px; }
    .sub { max-width:900px; color:#cbd8ea; margin:0; }
    .grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(240px,1fr)); gap:14px; }
    .card { background:white; border:1px solid var(--line); border-radius:8px; padding:16px; box-shadow:0 1px 2px rgba(15,23,42,.04); }
    .pill { display:inline-flex; margin:4px 6px 0 0; padding:6px 10px; border-radius:999px; background:#eaf7ef; color:var(--ok); font-weight:750; }
    .pill.warn { background:#fff4df; color:var(--warn); }
    .pill.bad { background:#fff1f1; color:var(--bad); }
    .shotgrid { display:grid; grid-template-columns:repeat(auto-fit,minmax(320px,1fr)); gap:14px; }
    figure { margin:0; background:white; border:1px solid var(--line); border-radius:8px; padding:10px; }
    figcaption { margin-bottom:8px; font-weight:750; }
    img { display:block; width:100%; border:1px solid #e5ebf4; border-radius:6px; }
    .ok { color:var(--ok); font-weight:750; }
    .warn-text { color:var(--warn); font-weight:750; }
    ${extraCss}
  </style>
</head>
<body>
${body}
</body>
</html>`;
}

function customerDemoHtml() {
  const shots = [
    ['API 网关驾驶舱', 'gateway_overview.png', '老板 10 秒内看懂通道、任务和异常。'],
    ['API 通道池', 'api_channel_pool.png', '集中查看 Seedance、Kling 和测试通道状态。'],
    ['健康诊断', 'health_diagnostics.png', '看到哪个账号影响任务，以及下一步动作。'],
    ['发起调用', 'create_call.png', '员工只选模板、写提示词、提交。'],
    ['调用记录', 'call_records.png', '查看成功、处理中、失败和失败原因。'],
    ['调用详情', 'call_detail.png', '管理员审计通道、路由账号、原因和时间线。'],
  ];
  return htmlPage('企业 AI 视频 API 管理中台 - 客户演示', `
  <header>
    <h1>企业 AI 视频 API 管理中台</h1>
    <p class="sub">统一托管企业视频 API Key，自动路由到可用账号，并回收结果与失败原因。</p>
    <div style="margin-top:14px">
      <span class="pill">客户可演示</span>
      <span class="pill warn">当前为安全演示通道</span>
      <span class="pill warn">真实通道需配置测试或正式 API Key 后启用</span>
    </div>
  </header>
  <main>
    <h2>客户可见能力</h2>
    <div class="grid">
      ${['多视频 API 通道统一管理','API Key 脱敏托管','员工通过网关发起调用','系统自动路由可用账号','任务状态回收','失败原因展示','API 健康诊断','管理员审计','下一步可接真实测试通道'].map((item) => `<div class="card"><h3>${item}</h3><p>${abilityText(item)}</p></div>`).join('\n')}
    </div>
    <h2>3 分钟演示路径</h2>
    <table>
      <tr><th>顺序</th><th>讲什么</th><th>客户看到什么</th></tr>
      <tr><td>1</td><td>先看总览</td><td>企业有哪些 API 通道、哪些账号可用、哪里需要处理。</td></tr>
      <tr><td>2</td><td>员工发起一次调用</td><td>选择模板、填写提示词、通过网关提交，不接触 API Key。</td></tr>
      <tr><td>3</td><td>查看任务回收</td><td>看到成功结果、处理中状态或失败原因。</td></tr>
      <tr><td>4</td><td>管理员审计</td><td>看到发起人、API 通道、实际路由账号、路由原因和时间线。</td></tr>
    </table>
    <h2>界面截图</h2>
    <div class="shotgrid">
      ${shots.map(([title, file, desc]) => `<figure><figcaption>${title}</figcaption><a href="screenshots/${file}"><img src="screenshots/${file}" alt="${title}" /></a><p>${desc}</p></figure>`).join('\n')}
    </div>
    <h2>当前边界</h2>
    <div class="card">
      <p>当前演示展示的是产品流程与管理能力。真实 Seedance 2.0 或 Kling 通道需要客户提供测试或正式 API Key 后，在试点中单独启用。</p>
      <p>正式生产前还需要完成部署环境、HTTPS、备份、监控、权限确认、成本结算和故障处理流程。</p>
    </div>
  </main>`);
}

function abilityText(item) {
  const map = {
    '多视频 API 通道统一管理': '把 Seedance 2.0、Kling 和测试通道放在一个管理面板中查看状态。',
    'API Key 脱敏托管': '员工不直接接触 API Key，前端只看到脱敏状态和是否可用。',
    '员工通过网关发起调用': '员工只填写业务提示词，调用从统一入口提交。',
    '系统自动路由可用账号': '系统按可用状态选择账号，减少人工切换和反复试错。',
    '任务状态回收': '任务创建后持续回收处理中、成功、失败等状态。',
    '失败原因展示': '失败时直接显示原因和建议动作，便于业务复用。',
    'API 健康诊断': '把账号异常、限流、鉴权失败等问题转成管理语言。',
    '管理员审计': '管理员可以查看发起人、通道、路由账号、路由原因和时间线。',
    '下一步可接真实测试通道': '试点阶段先接 1 个真实测试通道，验证从提交到回收的闭环。',
  };
  return map[item] || '';
}

function customerPilotHtml() {
  return htmlPage('企业 AI 视频 API 管理中台 - 客户试点方案', `
  <header>
    <h1>企业 AI 视频 API 管理中台</h1>
    <p class="sub">客户试点建议：先接 1 个真实测试通道，验证 API Key 托管、自动路由、任务回收、失败诊断和管理员审计。</p>
  </header>
  <main>
    <h2>试点目标</h2>
    <div class="grid">
      <div class="card"><h3>验证入口</h3><p>员工通过统一网关发起视频 API 调用，不直接接触 API Key。</p></div>
      <div class="card"><h3>验证通道</h3><p>先选择 Seedance 2.0 或 Kling 中的 1 个测试通道接入。</p></div>
      <div class="card"><h3>验证回收</h3><p>成功结果、处理中状态和失败原因都能进入任务记录。</p></div>
      <div class="card"><h3>验证审计</h3><p>管理员能看到发起人、API 通道、路由账号、路由原因和完整提示词。</p></div>
    </div>
    <h2>客户需要准备</h2>
    <table>
      <tr><th>项目</th><th>说明</th></tr>
      <tr><td>测试或正式 API Key</td><td>用于启用 1 个真实测试通道；建议先使用测试额度。</td></tr>
      <tr><td>模型选择</td><td>确认优先试点 Seedance 2.0 或 Kling。</td></tr>
      <tr><td>样例提示词</td><td>准备 3 到 5 个业务场景，覆盖成功和可能失败的情况。</td></tr>
      <tr><td>验收人员</td><td>至少包含业务负责人、普通使用者和管理员。</td></tr>
    </table>
    <h2>建议试点路径</h2>
    <table>
      <tr><th>步骤</th><th>产出</th><th>通过标准</th></tr>
      <tr><td>演示确认</td><td>确认产品价值和界面路径</td><td>客户能复述它解决 API Key 管理、调用入口、失败诊断和审计问题。</td></tr>
      <tr><td>单通道接入</td><td>启用 1 个真实测试通道</td><td>API Key 不出前端，任务可创建。</td></tr>
      <tr><td>任务回收</td><td>回收成功或失败状态</td><td>任务列表和详情能看到状态、结果或失败原因。</td></tr>
      <tr><td>验收复盘</td><td>确认下一步范围</td><td>明确是否继续投入正式上线前能力。</td></tr>
    </table>
    <h2>界面参考</h2>
    <div class="shotgrid">
      ${[
        ['API 网关驾驶舱','gateway_overview.png'],
        ['API 通道池','api_channel_pool.png'],
        ['健康诊断','health_diagnostics.png'],
        ['发起调用','create_call.png'],
        ['调用记录','call_records.png'],
        ['调用详情','call_detail.png'],
      ].map(([title, file]) => `<figure><figcaption>${title}</figcaption><img src="screenshots/${file}" alt="${title}" /></figure>`).join('\n')}
    </div>
  </main>`);
}

const faqItems = [
  ['产品价值', '为什么我们需要 API 管理中台？', '当企业同时使用多个视频 API 账号时，最难的是 Key 管理、员工入口、失败追踪和审计。中台把这些能力集中起来。', '这不是又一个生成页面，而是企业调用视频 API 的管理入口。', '当前先覆盖演示和试点闭环，正式生产能力需后续补齐。'],
  ['产品价值', '老板能从这里看到什么？', '老板可以看到通道是否可用、任务是否成功、失败原因和下一步建议。', '重点看总览、健康诊断和任务记录三屏。', '成本统计和经营报表属于后续增强。'],
  ['和普通视频生成工具的区别', '这和普通视频生成工具有什么区别？', '普通工具面向个人生成；本系统面向企业 API 管理，重点是 Key 托管、自动路由、状态回收和审计。', '员工仍然写提示词，但背后是企业统一管理 API。', '它不是创意剪辑工具，也不做视频后处理。'],
  ['和普通视频生成工具的区别', '为什么不直接让员工用官方平台？', '官方平台适合单人使用；企业需要统一管理账号、权限、失败原因和审计记录。', '员工直接用官方平台时，老板很难知道谁用了哪个账号、为什么失败。', '试点先验证管理闭环，不替代所有官方功能。'],
  ['API Key 安全', 'API Key 会不会泄露？', '设计目标是 Key 不出前端，员工只看到脱敏状态。', '演示时强调员工提交调用，不复制 API Key。', '正式上线前还需完成部署、日志和权限复核。'],
  ['API Key 安全', '谁可以配置 API Key？', '建议只允许管理员配置和轮换，普通员工只发起调用。', '演示时用管理员入口说明配置和脱敏展示。', '复杂权限分层不在当前演示范围。'],
  ['多账号管理', '能不能同时管理多个账号？', '可以统一查看多个 API 通道和账号状态。', '通道池就是给老板和管理员看的账号清单。', '当前不是大规模账号调度系统。'],
  ['多账号管理', '如果某个账号停用了怎么办？', '系统会显示账号已停用，并给出处理建议。', '看通道池和健康诊断里的建议动作。', '是否自动启停账号需后续确认规则。'],
  ['自动路由', '多账号路由是怎么决定的？', '试点阶段按可用状态和处理中数量优先选择可用账号。', '员工不用选账号，系统自动选择。', '复杂权重、成本优先和大规模调度属于后续范围。'],
  ['自动路由', '如果某个账号限流怎么办？', '系统会展示上游限流，并建议降低并发或切换可用账号。', '失败不是黑盒，能看到原因和下一步。', '自动扩容和全量调度不在当前演示范围。'],
  ['失败原因诊断', '如果生成失败，老板能看到什么？', '可以看到失败原因、影响账号和建议动作。', '打开任务详情和健康诊断，看失败原因如何回收。', '失败原因取决于上游返回的可解释程度。'],
  ['失败原因诊断', '失败后员工怎么处理？', '员工可以复制参数重新发起，也可以调整提示词后再提交。', '演示失败任务时说明下一步动作。', '自动修复提示词不在当前范围。'],
  ['员工权限和审计', '员工能不能看到别人任务？', '试点建议普通用户只看自己的任务，管理员看全量审计。', '演示时区分普通员工和管理员视角。', '正式权限策略需按客户组织规则确认。'],
  ['员工权限和审计', '管理员能审计哪些字段？', '可审计发起人、API 通道、实际路由账号、路由原因、提示词和时间线。', '调用详情页就是审计入口。', '更细权限和日志留存周期需正式上线前确认。'],
  ['真实 Seedance/Kling 接入', '能不能接我们的 Seedance / Kling？', '可以作为下一步试点接入 1 个真实测试通道。', '先选 Seedance 2.0 或 Kling 的一个测试通道。', '本演示不直接连接真实外部 API。'],
  ['真实 Seedance/Kling 接入', '接入真实通道需要多久？', '取决于客户提供的 Key、接口文档、样例参数和测试额度。', '先用 1 个通道跑通创建、轮询、结果回收和失败映射。', '不承诺一次性接入所有模型。'],
  ['成本和用量', '能不能统计每个人用了多少？', '可以作为后续试点增强方向，当前重点是调用闭环和审计字段。', '先把谁发起、用了哪个通道、是否成功记录清楚。', '精细计费和成本结算不在当前演示范围。'],
  ['成本和用量', '能不能控制成本？', '可以先通过通道状态、失败原因和账号可用性减少浪费。', '成本控制的第一步是把调用看清楚。', '预算告警和账单结算需后续建设。'],
  ['部署和试点周期', '试点需要我们提供什么？', '提供 1 个测试 API Key、模型选择、样例提示词、验收人员和试点边界。', '客户准备越清楚，单通道验证越快。', '不要在试点阶段直接承诺生产流量。'],
  ['部署和试点周期', '试点验收标准是什么？', '能创建真实任务、回收状态、记录失败原因、Key 不出前端、管理员可审计。', '验收清单已经拆成演示、试点、生产前三级。', '生产前验收另行确认。'],
  ['正式生产限制', '现在是否已经能生产使用？', '当前适合客户演示和收费试点讨论，不建议直接生产。', '说清楚当前是试点前状态，不是生产承诺。', '生产前还缺部署、监控、备份、权限和故障流程。'],
  ['正式生产限制', '上线前还缺什么？', '需要 HTTPS、部署环境、备份、监控、告警、日志留存、权限确认、成本结算和故障处理流程。', '这部分放在正式生产前验收里。', '当前不做生产部署。'],
  ['后续定制范围', '以后能不能接更多模型？', '可以扩展更多供应商，但建议先用一个真实测试通道验证闭环。', '先把单通道跑通，再谈扩展。', '不做多供应商并发真实联调。'],
  ['后续定制范围', '能不能做更多企业定制？', '可以讨论权限、报表、成本、审批等方向。', '当前先以最小试点验证商业价值。', '复杂定制需要单独评估范围和周期。'],
];

function customerFaqMd() {
  const lines = ['# 客户 FAQ 与异议处理', '', '产品名：企业 AI 视频 API 管理中台', '', '一句话：统一托管企业视频 API Key，自动路由到可用账号，并回收结果与失败原因。', ''];
  let current = '';
  for (const [category, q, a, demo, boundary] of faqItems) {
    if (category !== current) {
      current = category;
      lines.push(`## ${category}`, '');
    }
    lines.push(`Q：${q}`);
    lines.push(`A：${a}`);
    lines.push(`演示时怎么说：${demo}`);
    lines.push(`当前边界：${boundary}`);
    lines.push('');
  }
  return lines.join('\n');
}

function customerScriptMd() {
  return `# 客户 3 分钟演示脚本

## 0:00-0:20 开场
这套系统叫企业 AI 视频 API 管理中台。它解决的不是“怎么生成一个视频”，而是企业怎么统一托管视频 API Key、让员工通过网关发起调用、自动路由到可用账号，并回收结果与失败原因。

## 0:20-0:55 看总览
先看 API 网关驾驶舱。老板可以在一屏看到今天的调用、可用通道、异常账号和业务链路。这里的重点是账号能不能用、任务有没有回来、异常该怎么处理。

## 0:55-1:25 看通道池和健康诊断
API 通道池把 Seedance 2.0、Kling 和测试通道统一管理。健康诊断不会展示难懂日志，而是告诉管理员哪个账号影响了任务，以及下一步要配置 Key、处理限流或检查鉴权。

## 1:25-2:00 员工发起调用
员工进入发起调用页，只需要选模板、写提示词、提交。员工不需要选择账号，也不需要理解调度，系统会自动选择可用 API 账号。

## 2:00-2:35 看任务记录
调用任务页可以按全部、成功、处理中、失败筛选。失败时直接显示失败原因，成功时显示结果入口。

## 2:35-3:00 看调用详情和下一步
调用详情顶部先看结果或失败原因，再看 API 通道、实际路由账号、路由原因、完整提示词和时间线。下一步建议是先接 1 个真实测试通道，验证从创建到状态回收的闭环。
`;
}

function checklistRows(items) {
  return items.map((item) => `| ${item[0]} | ${item[1]} | ${item[2]} | ${item[3]} | ${item[4]} | ${item[5]} |`).join('\n');
}

function customerChecklistMd() {
  const demo = [
    ['客户能看懂产品价值','3 分钟演示后请客户复述用途','能说出 Key 托管、自动路由、结果回收、失败诊断','产品方 + 客户业务负责人','已覆盖','是'],
    ['能看到 API 通道池','打开客户演示包截图或现场页面','可识别 Seedance、Kling、测试通道状态','产品方','已覆盖','是'],
    ['能看到健康诊断','展示健康诊断截图','能看到影响账号和建议动作','产品方','已覆盖','是'],
    ['能看到发起调用','展示发起调用截图','员工路径清楚','产品方','已覆盖','是'],
    ['能看到任务记录','展示调用记录截图','状态和失败原因清楚','产品方','已覆盖','是'],
    ['能看到失败原因','展示失败任务或截图','失败原因可被业务理解','产品方','已覆盖','是'],
    ['能看到审计字段','展示调用详情截图','能看到发起人、通道、路由账号、路由原因','产品方','已覆盖','是'],
  ];
  const pilot = [
    ['客户提供 1 个测试 API Key','客户通过约定渠道提供','Key 可用于测试通道','客户','待客户提供','否'],
    ['接入 1 个真实测试通道','选择 Seedance 2.0 或 Kling','创建任务请求可发送','产品方 + 客户','待试点','否'],
    ['真实任务能创建','发起 1 次测试调用','生成上游任务编号或等价状态','产品方','待试点','否'],
    ['真实任务能回收状态','轮询或查询任务状态','状态进入任务记录','产品方','待试点','否'],
    ['失败能记录原因','触发或复现失败场景','失败原因进入任务详情','产品方','待试点','否'],
    ['Key 不出前端','检查页面和截图','前端只展示脱敏状态','产品方 + 客户管理员','待试点','否'],
    ['普通用户权限隔离','普通用户登录验证','普通用户只看到授权范围','产品方 + 客户管理员','待试点','否'],
    ['管理员可审计','管理员查看调用详情','审计字段完整','产品方 + 客户管理员','待试点','否'],
  ];
  const production = [
    ['HTTPS','检查部署访问协议','全站 HTTPS','客户 IT + 产品方','生产前待覆盖','否'],
    ['部署环境','确认服务器和网络','环境稳定可访问','客户 IT','生产前待覆盖','否'],
    ['备份','确认数据备份策略','可恢复关键数据','客户 IT + 产品方','生产前待覆盖','否'],
    ['监控','配置服务监控','异常可发现','产品方','生产前待覆盖','否'],
    ['告警','配置告警接收人','关键故障可通知','客户 IT + 产品方','生产前待覆盖','否'],
    ['日志留存','确认留存周期','满足客户合规要求','客户','生产前待覆盖','否'],
    ['权限确认','确认角色和边界','普通用户和管理员权限明确','客户管理员','生产前待覆盖','否'],
    ['法务和采购确认','客户内部流程确认','合同和采购流程通过','客户','生产前待覆盖','否'],
    ['成本结算方式','确认用量和结算规则','双方认可结算方式','客户 + 商务','生产前待覆盖','否'],
    ['SLA 和故障处理流程','确认响应机制','故障联系人和流程明确','客户 + 产品方','生产前待覆盖','否'],
  ];
  return `# 客户试点验收清单

## A. 演示验收
| 项目 | 验收方法 | 通过标准 | 责任方 | 当前状态 | 是否本轮覆盖 |
|---|---|---|---|---|---|
${checklistRows(demo)}

## B. 试点验收
| 项目 | 验收方法 | 通过标准 | 责任方 | 当前状态 | 是否本轮覆盖 |
|---|---|---|---|---|---|
${checklistRows(pilot)}

## C. 正式生产前验收
| 项目 | 验收方法 | 通过标准 | 责任方 | 当前状态 | 是否本轮覆盖 |
|---|---|---|---|---|---|
${checklistRows(production)}
`;
}

function bossBriefingMd() {
  return `# Boss Briefing - Phase 3.8.2

## 1. 当前项目一句话
企业 AI 视频 API 管理中台：统一托管企业视频 API Key，自动路由到可用账号，并回收结果与失败原因。

## 2. 当前已经完成什么
- 已完成 Phase 3.8 baseline 封存。
- 已完成 Phase 3.8.1 易用性改造并本地提交。
- 已升级客户演示包、试点材料、FAQ、验收清单和审查入口。
- 已准备自动化检查脚本，封包前可检查客户包敏感词、链接、MANIFEST 和 ZIP 内容。

## 3. 为什么它不是普通视频生成工具
普通工具解决“单人生成视频”。本项目解决“企业怎么管理多个视频 API 账号、员工怎么统一发起调用、失败怎么回收、管理员怎么审计”。

## 4. 商业价值
- 降低 API Key 分散带来的安全和管理风险。
- 降低员工直接使用多个平台的培训和试错成本。
- 让老板看到通道状态、失败原因和试点效果。
- 为后续按企业场景做权限、用量、成本和审计打基础。

## 5. 当前可演示能力
API 网关驾驶舱、发起调用、API 通道池、健康诊断、调用任务、调用详情、客户安全演示包和试点验收材料。

## 6. 当前不可承诺能力
不承诺正式生产、不承诺多供应商并发真实联调、不承诺 100 账号调度、不承诺计费、通知、对象存储、视频后处理。

## 7. 收费试点为什么是 CONDITIONALLY_READY
因为产品价值、演示路径和管理闭环已经清楚，但真实客户价值仍需 1 个真实测试通道验证。

## 8. 正式生产为什么还是 NOT_READY
生产前还缺部署环境、HTTPS、备份、监控、告警、日志留存、权限确认、成本结算和故障处理流程。

## 9. 下一个最小商业动作
约客户做 30 分钟演示，确认是否愿意提供 1 个测试 API Key 进入单真实测试通道试点。

## 10. 需要客户提供什么
测试 API Key、优先供应商、样例提示词、验收人员、试点边界。

## 11. 预计 Phase 4 工作范围
只接 1 个真实测试通道，完成任务创建、状态查询、结果回收、失败原因映射和回滚方案。

## 12. 风险清单
- 客户无法提供测试 Key。
- 上游接口文档或字段不稳定。
- 失败原因不够可解释。
- 客户提前要求生产级权限、计费或 SLA。

## 13. 是否建议推进客户试点
建议推进，但只承诺单通道测试试点。

## 14. 是否建议继续投入
建议继续投入 Phase 4，前提是客户明确愿意提供测试通道和验收场景。
`;
}

function providerPrepPlanMd() {
  return `# Phase 4 Single Provider Prep Plan

## 1. Phase 4 目标
只接入 1 个真实测试通道，验证从任务创建、状态轮询、结果回收到失败原因映射的闭环。

## 2. 选择 Seedance 2.0 或 Kling 的决策标准
- 客户是否已有可用测试或正式 API Key。
- 接口文档是否清楚。
- 是否支持任务创建、状态查询、结果 URL 回收。
- 失败原因是否可映射。
- 测试额度是否足够覆盖验收。

## 3. 客户需要提供的信息
供应商选择、API Key、测试额度说明、接口文档、样例提示词、成功样例、失败样例、验收联系人。

## 4. 测试 Key 接入边界
只用于 1 个测试通道，不做生产流量，不做多供应商并发真实联调。

## 5. Secrets 管理要求
Key 只进入后端受控配置或受控管理入口，不进入前端、不进入日志、不进入审查包、不进入截图。

## 6. Key 红线
不允许明文 Key 出现在前端、日志、审查包、截图、聊天记录、错误堆栈或 ZIP。

## 7. Provider adapter 接入点
在现有视频 provider adapter 边界内新增或启用一个真实 provider 实现，保持任务创建、状态查询、失败映射接口稳定。

## 8. 任务创建字段映射
映射提示词、负向提示词、视频比例、时长、分辨率、模型名、回调或查询所需字段。

## 9. 状态轮询字段映射
映射上游任务 ID、运行中、成功、失败、取消、超时等状态。

## 10. 成功结果回收字段
回收视频结果 URL、缩略图或等价结果字段、完成时间、上游任务 ID。

## 11. 失败原因映射
将上游错误映射为鉴权失败、上游限流、额度不足、审核失败、超时、参数错误、未知错误。

## 12. 异常处理方式
- 限流：记录失败原因，建议降低并发或切换账号。
- 鉴权失败：禁用或标记该 Key 需要处理。
- 额度不足：提示补充额度或切换通道。
- 审核失败：返回业务可理解原因。
- 超时：标记超时并允许后续人工复查。

## 13. 单通道联调步骤
1. 客户确认 Seedance 2.0 或 Kling。
2. 建立测试 Key 输入和脱敏展示。
3. 配置 provider adapter。
4. 发起 1 个成功样例任务。
5. 回收状态和结果。
6. 发起或模拟 1 个失败样例。
7. 验证任务列表、详情和健康诊断。
8. 形成试点验收记录。

## 14. 回滚方案
关闭真实测试通道，保留安全演示通道；清理测试 Key；恢复客户演示包为安全演示状态。

## 15. 验收标准
真实任务可创建、状态可回收、失败可记录、Key 不出前端、管理员可审计、客户验收人员认可。

## 16. 停止条件
发现 Key 泄露风险、上游文档无法支持任务闭环、需要改认证权限主链路、客户要求生产流量、或必须接多个真实供应商才能继续。

## 17. 生产前仍需补齐的能力
部署环境、HTTPS、备份、监控、告警、日志留存、权限策略、成本结算、SLA、故障处理流程。

## 明确不做
- 不做多供应商并发真实联调。
- 不做 100 账号调度。
- 不做生产部署。
- 不做计费。
- 不做视频后处理。
`;
}

function providerPolicyMd() {
  return `# Provider Credentials Handling Policy

## Key 输入位置
测试或正式 API Key 只允许通过后端受控配置入口或管理员受控管理入口输入。

## Key 存储方式
Key 必须使用后端受控存储，前端只保存脱敏展示所需状态。示例变量名可以使用 PROVIDER_API_KEY，但不得写入真实值。

## Key 脱敏展示
前端只展示是否已配置、尾号或脱敏片段，不展示完整 Key。

## Key 日志红线
日志中不得输出完整 Key、请求头、鉴权字段或上游返回中的敏感凭证。

## Key 截图红线
截图中不得出现完整 Key、请求头、鉴权字段、配置文件内容或可还原凭证的片段。

## Key 审查包红线
审查包、客户包、内部包、ZIP 和 MANIFEST 不得包含真实 Key 原文。

## Key 回收/轮换流程
试点结束后由客户确认是否回收测试 Key；如继续试点，需记录轮换时间、责任人和授权范围。

## 测试 Key 和生产 Key 区分
优先使用测试 Key；生产 Key 只能在生产验收完成后进入正式环境。

## 客户授权记录要求
接入前需记录客户授权、供应商、Key 类型、用途、有效期、接触人和回收方式。

## 变量名示例
- SEEDANCE_API_KEY
- KLING_API_KEY
- VIDEO_PROVIDER_API_KEY

以上仅为变量名示例，不得填写真实值。
`;
}

function screenshotIndexHtml() {
  const rows = [
    ['api_gateway_dashboard_usability_v1.png','API 网关驾驶舱','三秒价值区、业务链路、角色入口','是','是'],
    ['api_channel_pool_usability_v1.png','API 通道池','通道状态、人话异常、建议动作','是','是'],
    ['api_health_diagnostics_usability_v1.png','API 健康诊断','账号影响、失败原因、下一步动作','是','是'],
    ['api_call_create_usability_v1.png','发起调用','员工主路径、模板、提示词、自动路由说明','是','是'],
    ['api_call_tasks_usability_v1.png','任务列表','快捷筛选、发起人、通道、路由账号、失败原因','是','是'],
    ['api_call_detail_usability_v1.png','调用详情','结果、失败原因、通道、路由账号、时间线','是','是'],
  ];
  return htmlPage('Phase 3.8.2 Screenshot Index', `
  <header><h1>Phase 3.8.2 Screenshot Index</h1><p class="sub">内部截图证据索引；客户包引用的是客户包 screenshots 目录内的安全副本。</p></header>
  <main>
    <table>
      <tr><th>截图</th><th>对应页面</th><th>证明的功能点</th><th>是否客户可见</th><th>是否被客户包引用</th></tr>
      ${rows.map((r) => `<tr><td><a href="${r[0]}">${r[0]}</a></td><td>${r[1]}</td><td>${r[2]}</td><td>${r[3]}</td><td>${r[4]}</td></tr>`).join('\n')}
    </table>
    <h2>客户包安全副本</h2>
    <ul>
      <li><code>01_客户演示包/screenshots/gateway_overview.png</code></li>
      <li><code>01_客户演示包/screenshots/api_channel_pool.png</code></li>
      <li><code>01_客户演示包/screenshots/health_diagnostics.png</code></li>
      <li><code>01_客户演示包/screenshots/create_call.png</code></li>
      <li><code>01_客户演示包/screenshots/call_records.png</code></li>
      <li><code>01_客户演示包/screenshots/call_detail.png</code></li>
    </ul>
  </main>`);
}

function mainReviewHtml() {
  const conclusion = opts.ready ? 'READY' : 'PENDING_VALIDATION';
  const validation = opts.ready ? 'PASS' : 'PENDING';
  return htmlPage('Phase 3.8.2 Overnight Readiness Review', `
  <header>
    <h1>Phase 3.8.2 Overnight Readiness Review</h1>
    <p class="sub">彻夜级复查、客户演示包升级、自动化验收补齐、Phase 4 单真实通道联调前置准备。</p>
    <div style="margin-top:14px">
      <span class="pill ${opts.ready ? '' : 'warn'}">${conclusion}</span>
      <span class="pill warn">未进入 Phase 4</span>
      <span class="pill warn">未接真实 API</span>
      <span class="pill bad">生产仍为 NOT_READY</span>
    </div>
  </header>
  <main>
    <h2>关键结论</h2>
    <table>
      <tr><th>项目</th><th>结果</th></tr>
      <tr><td>最终结论</td><td>${conclusion}</td></tr>
      <tr><td>当前阶段定位</td><td>Phase 3.8.2 Overnight Readiness</td></tr>
      <tr><td>3.8 baseline commit</td><td><code>58f79542ab23e56c19fc5215e4b9c1a74ff033bf</code></td></tr>
      <tr><td>3.8.1 usability commit</td><td><code>${opts.phase381}</code></td></tr>
      <tr><td>3.8.2 readiness commit</td><td><code>${opts.phase382}</code></td></tr>
      <tr><td>当前分支</td><td><code>${opts.branch}</code></td></tr>
      <tr><td>本轮是否进入 Phase 4</td><td>否</td></tr>
      <tr><td>本轮是否接真实 API</td><td>否</td></tr>
      <tr><td>是否修改 Phase 3.8 封存快照</td><td>否</td></tr>
      <tr><td>是否 push</td><td>否</td></tr>
    </table>
    <h2>验收结果</h2>
    <table>
      <tr><th>检查项</th><th>结果</th><th>证据</th></tr>
      <tr><td>3.8.1 易用性复查结论</td><td>${validation}</td><td><a href="PHASE_3_8_2_USABILITY_AUDIT.md">PHASE_3_8_2_USABILITY_AUDIT.md</a></td></tr>
      <tr><td>客户演示包升级结果</td><td>${validation}</td><td><a href="01_客户演示包/CUSTOMER_DEMO_SAFE_REVIEW.html">CUSTOMER_DEMO_SAFE_REVIEW.html</a></td></tr>
      <tr><td>客户 FAQ 结果</td><td>${validation}</td><td><a href="01_客户演示包/CUSTOMER_FAQ_AND_OBJECTION_HANDLING.md">CUSTOMER_FAQ_AND_OBJECTION_HANDLING.md</a></td></tr>
      <tr><td>客户验收清单结果</td><td>${validation}</td><td><a href="01_客户演示包/CUSTOMER_ACCEPTANCE_CHECKLIST.md">CUSTOMER_ACCEPTANCE_CHECKLIST.md</a></td></tr>
      <tr><td>老板汇报包结果</td><td>${validation}</td><td><a href="02_内部审查包/BOSS_BRIEFING_PHASE_3_8_2.md">BOSS_BRIEFING_PHASE_3_8_2.md</a></td></tr>
      <tr><td>Phase 4 前置计划结果</td><td>${validation}</td><td><a href="03_真实通道接入/PHASE_4_SINGLE_PROVIDER_PREP_PLAN.md">PHASE_4_SINGLE_PROVIDER_PREP_PLAN.md</a></td></tr>
      <tr><td>Provider 凭证处理政策结果</td><td>${validation}</td><td><a href="03_真实通道接入/PROVIDER_CREDENTIALS_HANDLING_POLICY.md">PROVIDER_CREDENTIALS_HANDLING_POLICY.md</a></td></tr>
      <tr><td>自动化检查脚本结果</td><td>${validation}</td><td><a href="tools/run_phase_3_8_2_checks.ps1">run_phase_3_8_2_checks.ps1</a></td></tr>
      <tr><td>自动化检查日志</td><td>${validation}</td><td><a href="PHASE_3_8_2_AUTOMATED_CHECK_LOG.md">PHASE_3_8_2_AUTOMATED_CHECK_LOG.md</a></td></tr>
      <tr><td>客户包 scrub</td><td>${validation}</td><td>客户包禁词和疑似 Key 模式检查</td></tr>
      <tr><td>内部敏感信息检查</td><td>${validation}</td><td>ZIP 禁止文件与疑似 Key 检查</td></tr>
      <tr><td>链接检查</td><td>${validation}</td><td>HTML 本地 href/src 检查</td></tr>
      <tr><td>MANIFEST 检查</td><td>${validation}</td><td><a href="MANIFEST.json">MANIFEST.json</a></td></tr>
      <tr><td>前端 build</td><td>${validation}</td><td><code>corepack pnpm --dir frontend run build</code></td></tr>
      <tr><td>demo build</td><td>${validation}</td><td><code>VITE_PRODUCT_MODE=video_gateway_demo corepack pnpm --dir frontend run build</code></td></tr>
      <tr><td>git diff --check</td><td>${validation}</td><td>无空白错误</td></tr>
      <tr><td>ZIP 禁止文件检查</td><td>${validation}</td><td>客户 ZIP 与内部 ZIP 均检查</td></tr>
    </table>
    <h2>截图证据</h2>
    <p><a href="04_截图证据/PHASE_3_8_2_SCREENSHOT_INDEX.html">PHASE_3_8_2_SCREENSHOT_INDEX.html</a></p>
    <h2>客户可外发文件清单</h2>
    <ul>
      <li><a href="01_客户演示包/CUSTOMER_DEMO_SAFE_REVIEW.html">CUSTOMER_DEMO_SAFE_REVIEW.html</a></li>
      <li><a href="01_客户演示包/CUSTOMER_PILOT_PROPOSAL.html">CUSTOMER_PILOT_PROPOSAL.html</a></li>
      <li><a href="01_客户演示包/CUSTOMER_3MIN_DEMO_SCRIPT.md">CUSTOMER_3MIN_DEMO_SCRIPT.md</a></li>
      <li><a href="01_客户演示包/CUSTOMER_FAQ_AND_OBJECTION_HANDLING.md">CUSTOMER_FAQ_AND_OBJECTION_HANDLING.md</a></li>
      <li><a href="01_客户演示包/CUSTOMER_ACCEPTANCE_CHECKLIST.md">CUSTOMER_ACCEPTANCE_CHECKLIST.md</a></li>
    </ul>
    <h2>内部审查文件清单</h2>
    <ul>
      <li><a href="PHASE_3_8_2_WORKSPACE_AUDIT.md">PHASE_3_8_2_WORKSPACE_AUDIT.md</a></li>
      <li><a href="PHASE_3_8_2_USABILITY_AUDIT.md">PHASE_3_8_2_USABILITY_AUDIT.md</a></li>
      <li><a href="PHASE_3_8_1_COMMIT_RECORD.md">PHASE_3_8_1_COMMIT_RECORD.md</a></li>
      <li><a href="02_内部审查包/BOSS_BRIEFING_PHASE_3_8_2.md">BOSS_BRIEFING_PHASE_3_8_2.md</a></li>
      <li><a href="03_真实通道接入/PHASE_4_SINGLE_PROVIDER_PREP_PLAN.md">PHASE_4_SINGLE_PROVIDER_PREP_PLAN.md</a></li>
      <li><a href="03_真实通道接入/PROVIDER_CREDENTIALS_HANDLING_POLICY.md">PROVIDER_CREDENTIALS_HANDLING_POLICY.md</a></li>
    </ul>
    <h2>当前仍不是生产级的范围</h2>
    <p>正式生产仍为 NOT_READY：部署、HTTPS、备份、监控、告警、日志留存、权限确认、成本结算、SLA 和故障处理流程需另行完成。</p>
    <h2>是否建议进入 Phase 4</h2>
    <p>${opts.ready ? '建议进入 Phase 4，但只做单真实测试通道联调。' : '待自动化和 ZIP 检查完成后再给出建议。'}</p>
    <h2>Phase 4 下一步最小任务</h2>
    <p>客户确认 Seedance 2.0 或 Kling 其一，并提供测试 API Key，按单通道计划验证任务创建、状态回收、失败原因映射和审计。</p>
    <h2>阻断项</h2>
    <p>${opts.ready ? '无。' : '等待最终检查。'}</p>
    <h2>ZIP 路径</h2>
    <table>
      <tr><th>客户 ZIP</th><td><code>${esc(opts.customerZip || 'PENDING')}</code></td></tr>
      <tr><th>内部 ZIP</th><td><code>${esc(opts.internalZip || 'PENDING')}</code></td></tr>
    </table>
  </main>`);
}

function startHereHtml() {
  return htmlPage('Phase 3.8.2 Latest Entry', `
  <header><h1>Phase 3.8.2 Overnight Readiness</h1><p class="sub">最新入口。客户外发材料只使用 <code>01_客户演示包/</code>。</p></header>
  <main>
    <div class="grid">
      <div class="card"><h3>最新主审查包</h3><p><a href="PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html">PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html</a></p></div>
      <div class="card"><h3>客户演示包</h3><p><a href="01_客户演示包/CUSTOMER_DEMO_SAFE_REVIEW.html">CUSTOMER_DEMO_SAFE_REVIEW.html</a></p></div>
      <div class="card"><h3>客户试点方案</h3><p><a href="01_客户演示包/CUSTOMER_PILOT_PROPOSAL.html">CUSTOMER_PILOT_PROPOSAL.html</a></p></div>
      <div class="card"><h3>截图索引</h3><p><a href="04_截图证据/PHASE_3_8_2_SCREENSHOT_INDEX.html">PHASE_3_8_2_SCREENSHOT_INDEX.html</a></p></div>
    </div>
    <h2>状态</h2>
    <table>
      <tr><th>项目</th><th>结论</th></tr>
      <tr><td>当前最新阶段</td><td>Phase 3.8.2 Overnight Readiness</td></tr>
      <tr><td>客户试点</td><td>CONDITIONALLY_READY</td></tr>
      <tr><td>正式生产</td><td>NOT_READY</td></tr>
      <tr><td>真实 API 联调</td><td>本轮未执行</td></tr>
      <tr><td>push</td><td>否</td></tr>
    </table>
  </main>`);
}

function startHereMd() {
  return `# Phase 3.8.2 Overnight Readiness

最新主审查包：PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html

客户可外发范围：
- 01_客户演示包/CUSTOMER_DEMO_SAFE_REVIEW.html
- 01_客户演示包/CUSTOMER_PILOT_PROPOSAL.html
- 01_客户演示包/CUSTOMER_3MIN_DEMO_SCRIPT.md
- 01_客户演示包/CUSTOMER_FAQ_AND_OBJECTION_HANDLING.md
- 01_客户演示包/CUSTOMER_ACCEPTANCE_CHECKLIST.md

内部范围：
- PHASE_3_8_2_WORKSPACE_AUDIT.md
- PHASE_3_8_2_USABILITY_AUDIT.md
- PHASE_3_8_1_COMMIT_RECORD.md
- PHASE_3_8_2_AUTOMATED_CHECK_LOG.md
- 02_内部审查包/BOSS_BRIEFING_PHASE_3_8_2.md
- 03_真实通道接入/PHASE_4_SINGLE_PROVIDER_PREP_PLAN.md
- 03_真实通道接入/PROVIDER_CREDENTIALS_HANDLING_POLICY.md

状态：
- 客户试点：CONDITIONALLY_READY
- 正式生产：NOT_READY
- 本轮是否接真实 API：否
- 本轮是否 push：否
`;
}

function latestHtml() {
  return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta http-equiv="refresh" content="0; url=PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Latest Review Package - Phase 3.8.2</title>
</head>
<body>
  <main>
    <h1>Latest Review Package: Phase 3.8.2 Overnight Readiness</h1>
    <p><a href="PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html">Open PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html</a></p>
  </main>
</body>
</html>`;
}

function manifestJson() {
  const validation = opts.ready ? 'PASS' : 'PENDING';
  return JSON.stringify({
    project: '企业 AI 视频 API 管理中台',
    updated_at: new Date().toISOString(),
    phase: 'phase3_8_2_overnight_readiness',
    branch: opts.branch,
    commits: {
      phase3_8_baseline: '58f79542ab23e56c19fc5215e4b9c1a74ff033bf',
      phase3_8_1_usability: opts.phase381,
      phase3_8_2_readiness: opts.phase382,
    },
    commercial_status: {
      customer_demo: opts.ready ? 'READY' : 'PENDING',
      paid_pilot: 'CONDITIONALLY_READY',
      production: 'NOT_READY',
    },
    latest_files: [
      fileItem('00_START_HERE.html','entrypoint','internal_only','latest entry'),
      fileItem('00_START_HERE.md','entrypoint','internal_only','latest entry markdown'),
      fileItem('LATEST_REVIEW_PACKAGE.html','entrypoint','internal_only','latest redirect'),
      fileItem('PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html','internal_review','internal_only','main review'),
      fileItem('PHASE_3_8_2_WORKSPACE_AUDIT.md','internal_review','internal_only','workspace audit'),
      fileItem('PHASE_3_8_2_USABILITY_AUDIT.md','internal_review','internal_only','usability audit'),
      fileItem('PHASE_3_8_2_AUTOMATED_CHECK_LOG.md','internal_review','internal_only','automated check log'),
      fileItem('PHASE_3_8_1_COMMIT_RECORD.md','internal_review','internal_only','3.8.1 commit record'),
      fileItem('01_客户演示包/CUSTOMER_DEMO_SAFE_REVIEW.html','customer_safe','client_shareable','customer demo'),
      fileItem('01_客户演示包/CUSTOMER_PILOT_PROPOSAL.html','customer_safe','client_shareable','pilot proposal'),
      fileItem('01_客户演示包/CUSTOMER_3MIN_DEMO_SCRIPT.md','customer_safe','client_shareable','3 minute script'),
      fileItem('01_客户演示包/CUSTOMER_FAQ_AND_OBJECTION_HANDLING.md','customer_safe','client_shareable','FAQ'),
      fileItem('01_客户演示包/CUSTOMER_ACCEPTANCE_CHECKLIST.md','customer_safe','client_shareable','acceptance checklist'),
      fileItem('02_内部审查包/BOSS_BRIEFING_PHASE_3_8_2.md','internal_review','internal_only','boss briefing'),
      fileItem('03_真实通道接入/PHASE_4_SINGLE_PROVIDER_PREP_PLAN.md','internal_review','internal_only','phase 4 prep'),
      fileItem('03_真实通道接入/PROVIDER_CREDENTIALS_HANDLING_POLICY.md','internal_review','internal_only','credentials policy'),
      fileItem('04_截图证据/PHASE_3_8_2_SCREENSHOT_INDEX.html','internal_review','internal_only','screenshot index'),
      fileItem('tools/run_phase_3_8_2_checks.ps1','tooling','internal_only','one click checks'),
    ],
    screenshots: [
      '03_审查包/04_截图证据/api_gateway_dashboard_usability_v1.png',
      '03_审查包/04_截图证据/api_channel_pool_usability_v1.png',
      '03_审查包/04_截图证据/api_health_diagnostics_usability_v1.png',
      '03_审查包/04_截图证据/api_call_create_usability_v1.png',
      '03_审查包/04_截图证据/api_call_tasks_usability_v1.png',
      '03_审查包/04_截图证据/api_call_detail_usability_v1.png',
      '03_审查包/01_客户演示包/screenshots/gateway_overview.png',
      '03_审查包/01_客户演示包/screenshots/api_channel_pool.png',
      '03_审查包/01_客户演示包/screenshots/health_diagnostics.png',
      '03_审查包/01_客户演示包/screenshots/create_call.png',
      '03_审查包/01_客户演示包/screenshots/call_records.png',
      '03_审查包/01_客户演示包/screenshots/call_detail.png',
    ],
    customer_package: {
      path: '03_审查包/01_客户演示包',
      share_level: 'client_shareable',
      scrub: validation,
      zip_path: opts.customerZip || 'PENDING',
    },
    internal_package: {
      share_level: 'internal_only',
      zip_path: opts.internalZip || 'PENDING',
    },
    validation: {
      workspace_audit: 'PASS',
      usability_audit: 'PASS',
      customer_scrub: validation,
      link_check: validation,
      manifest_check: validation,
      zip_forbidden_files: validation,
      frontend_build: validation,
      demo_build: validation,
      git_diff_check: validation,
      screenshot_check: 'PASS',
      real_external_api_connected: false,
      production: 'NOT_READY',
    },
    next_step: 'Phase 4 只建议接入 1 个真实测试通道，先验证创建、状态回收、失败原因映射和审计。',
  }, null, 2);
}

function fileItem(relPath, category, shareLevel, description) {
  return { name: path.basename(relPath), path: `03_审查包/${relPath}`, category, share_level: shareLevel, is_latest: true, description };
}

function copyScreenshots() {
  const map = [
    ['api_gateway_dashboard_usability_v1.png','gateway_overview.png'],
    ['api_channel_pool_usability_v1.png','api_channel_pool.png'],
    ['api_health_diagnostics_usability_v1.png','health_diagnostics.png'],
    ['api_call_create_usability_v1.png','create_call.png'],
    ['api_call_tasks_usability_v1.png','call_records.png'],
    ['api_call_detail_usability_v1.png','call_detail.png'],
  ];
  const dstDir = path.join(customerRoot, 'screenshots');
  ensureDir(dstDir);
  for (const file of fs.readdirSync(dstDir)) {
    if (file.toLowerCase().endsWith('.png')) fs.unlinkSync(path.join(dstDir, file));
  }
  for (const [src, dst] of map) {
    copyFile(path.join(screenshotRoot, src), path.join(dstDir, dst));
  }
}

function copyTools() {
  ensureDir(reviewToolsRoot);
  for (const file of fs.readdirSync(__dirname)) {
    if (/\.(js|ps1)$/.test(file)) copyFile(path.join(__dirname, file), path.join(reviewToolsRoot, file));
  }
}

function main() {
  [reviewRoot, customerRoot, internalRoot, providerRoot, screenshotRoot, reviewToolsRoot].forEach(ensureDir);
  copyScreenshots();
  copyTools();

  write('01_客户演示包/CUSTOMER_DEMO_SAFE_REVIEW.html', customerDemoHtml());
  write('01_客户演示包/CUSTOMER_PILOT_PROPOSAL.html', customerPilotHtml());
  write('01_客户演示包/CUSTOMER_3MIN_DEMO_SCRIPT.md', customerScriptMd());
  write('01_客户演示包/CUSTOMER_FAQ_AND_OBJECTION_HANDLING.md', customerFaqMd());
  write('01_客户演示包/CUSTOMER_ACCEPTANCE_CHECKLIST.md', customerChecklistMd());
  write('02_内部审查包/BOSS_BRIEFING_PHASE_3_8_2.md', bossBriefingMd());
  write('03_真实通道接入/PHASE_4_SINGLE_PROVIDER_PREP_PLAN.md', providerPrepPlanMd());
  write('03_真实通道接入/PROVIDER_CREDENTIALS_HANDLING_POLICY.md', providerPolicyMd());
  write('04_截图证据/PHASE_3_8_2_SCREENSHOT_INDEX.html', screenshotIndexHtml());
  write('PHASE_3_8_2_OVERNIGHT_READINESS_REVIEW.html', mainReviewHtml());
  write('00_START_HERE.html', startHereHtml());
  write('00_START_HERE.md', startHereMd());
  write('LATEST_REVIEW_PACKAGE.html', latestHtml());
  write('MANIFEST.json', manifestJson());

  console.log(`Generated Phase 3.8.2 package at ${reviewRoot}`);
}

main();
