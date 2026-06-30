-- Value-Output-C dashboard verification — synthetic capture rows (throwaway).
-- These exercise the READ path (GetCaptureStats / GetRecent): distinct employees/teams/models,
-- today vs week buckets, 7-day series, byte sums, JOIN attribution, rune-safe truncation, redaction passthrough.
-- user_id/group_id resolved by name so we don't depend on serial ids. api_key_id NULL (read path doesn't need it).

-- r1: alice / product-team / claude-3-5-sonnet / today
INSERT INTO ai_generation_content (request_id, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version, created_at)
SELECT 'syn-r1', u.id, g.id, 'claude-3-5-sonnet-20241022',
  '帮我审查这段 Go 并发代码，重点看 goroutine 泄漏，联系方式 [PHONE]',
  'MOCK_REPLY 这里有个 use-after-return 风险，建议用 context 取消。',
  420, 880, false, 1, NOW()
FROM users u, groups g WHERE u.username='alice' AND g.name='product-team';

-- r2: bob / product-team / claude-opus-4-8 / today
INSERT INTO ai_generation_content (request_id, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version, created_at)
SELECT 'syn-r2', u.id, g.id, 'claude-opus-4-8',
  '把这份产品需求文档整理成季度路线图，工单号 [已脱敏]',
  'MOCK_REPLY 已按优先级分为 P0/P1/P2 三档，详见下表。',
  610, 1500, false, 1, NOW()
FROM users u, groups g WHERE u.username='bob' AND g.name='product-team';

-- r3: carol / research-team / gemini-2.5-pro / yesterday
INSERT INTO ai_generation_content (request_id, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version, created_at)
SELECT 'syn-r3', u.id, g.id, 'gemini-2.5-pro',
  '总结这篇关于扩散模型的论文的三个核心贡献',
  'MOCK_REPLY 1) 新的噪声调度 2) 更快的采样 3) 更好的可控性。',
  350, 760, false, 1, NOW() - INTERVAL '1 day'
FROM users u, groups g WHERE u.username='carol' AND g.name='research-team';

-- r4: alice / product-team / claude-opus-4-8 / 2 days ago — LONG (triggers preview truncation, >120/>80 runes)
INSERT INTO ai_generation_content (request_id, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version, created_at)
SELECT 'syn-r4', u.id, g.id, 'claude-opus-4-8',
  '请详细审查我们整个支付网关模块的并发安全性，逐一检查每个 handler 的锁粒度、context 传递、超时控制、重试幂等、以及数据库事务边界，并给出一份可执行的重构清单，覆盖所有边界情况和回滚策略，越详细越好谢谢',
  'MOCK_REPLY 经过逐一排查，我发现以下几个关键问题需要优先处理，第一是订单创建路径存在竞态条件，第二是退款回调缺少幂等保护，第三是事务边界过宽导致锁争用，建议分三个阶段重构。',
  2600, 1800, false, 1, NOW() - INTERVAL '2 days'
FROM users u, groups g WHERE u.username='alice' AND g.name='product-team';

-- r5: bob / research-team / claude-3-5-sonnet / 3 days ago — upstream response_truncated=true
INSERT INTO ai_generation_content (request_id, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version, created_at)
SELECT 'syn-r5', u.id, g.id, 'claude-3-5-sonnet-20241022',
  '生成一个大型数据集的统计报告模板',
  'MOCK_REPLY 报告模板如下（内容超过采集上限，已截断）……',
  300, 65536, true, 1, NOW() - INTERVAL '3 days'
FROM users u, groups g WHERE u.username='bob' AND g.name='research-team';

-- r6: carol / product-team / gemini-2.5-pro / 5 days ago
INSERT INTO ai_generation_content (request_id, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version, created_at)
SELECT 'syn-r6', u.id, g.id, 'gemini-2.5-pro',
  '写一个 SQL 查询，统计每个团队最近 7 天的活跃用户数',
  'MOCK_REPLY SELECT team, COUNT(DISTINCT user_id) ... GROUP BY team;',
  280, 540, false, 1, NOW() - INTERVAL '5 days'
FROM users u, groups g WHERE u.username='carol' AND g.name='product-team';

-- r7: alice / research-team / claude-3-5-sonnet / 6 days ago
INSERT INTO ai_generation_content (request_id, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version, created_at)
SELECT 'syn-r7', u.id, g.id, 'claude-3-5-sonnet-20241022',
  '帮我把这段英文技术文档翻译成中文',
  'MOCK_REPLY 译文：本系统采用旁路采集，对主链路零影响。',
  330, 700, false, 1, NOW() - INTERVAL '6 days'
FROM users u, groups g WHERE u.username='alice' AND g.name='research-team';
