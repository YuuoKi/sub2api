# Sub2API · 当前唯一目标 = M1 / C1 端到端

更新时间：2026-06-18 Asia/Shanghai

## 唯一焦点

**C1：打穿第一条端到端真实链路。**

```
QCanvas 界面点一下
   → Sub2API（从 harness 验证升级为真实可调本地服务进程）
   → provider 层（C1 阶段走 mock/stub，真实 Seedance 闸门保持关闭）
   → 结果回 QCanvas 候选区
```

通了 = 证明整个架构真能跑（不是 mock 演示），之后 Kling / 图片 / 文本沿同一条路复制。

## C1 的两段

1. **Phase B（已在本轮夜跑推进）**：修 Seedance 适配器契约缺陷（B1 比例字段 / B2 轮询窗口 / B3 v2v 草案），用「不真实付费调用」的契约测试证明 payload 正确。
2. **Phase C（本轮夜跑骨架）**：Sub2API 本地起真实服务进程（provider=mock），QCanvas 解 mock 接本地 Sub2API，端到端走 创建→轮询→结果→候选 一遍。

## 验收

一个人在浏览器里真生成出一个视频（含竖屏），结果回到画布。
（C1 骨架阶段先用 mock provider 证明骨架通；真实 provider 门 = M1 收尾，需老板授权。）

## 不做（在 C1 端到端真通之前）

不碰 Kling、不碰图片/文本、不美化 UI、不抠 Seedance 边角字段。所有火力收到一条线上。

> 历史目标文档 `docs/goals/03_CURRENT_GOAL.md` 已被本文件取代为根级单一锚点。
