# Sub2API W5 后续建议

日期：2026-07-03 Asia/Shanghai

## 建议顺序

1. 授权修复 payment webhook P1  
   重新定义非 wxpay provider lookup error 场景下的 ack / retry / 落库语义，避免真实支付回调被 2xx 吞掉。

2. 收紧 weekly report 窗口  
   在保持 admin-only 的前提下增加最大跨度、分页或异步导出边界，减少误用导致的慢查询风险。

3. 决策真实 provider 后续试跑条件  
   若要继续 tiny real 以外的供应商调用，必须单独授权预算、次数、模型、停止条件、脱敏审查包和回滚方案。

4. 推进 QCanvas B2 adoption 回流  
   让 QCanvas 侧把 B1 generation-content 账本能力接回真实团队生产路径，但不得自动解冻 provider。

## 推荐下一条提示词

请在 D:\sub2api-trunk 只读复核 `_review/MEGA_LOOP_FINAL_AUDIT_20260703/`，确认 open P0=0、W5 门禁结果、Phase A' 三证与 B1 账本未回归；然后给出是否授权修复 payment webhook P1 的判断。不得读取 key/.env/token/cookie，不 push，不触发真实 provider。
