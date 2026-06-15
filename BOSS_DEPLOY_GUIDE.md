# Sub2API 企业 AI 视频 API 调度中台 — 上线手册

**版本**: v1.1  
**更新**: 2026-05-28  
**面向**: 老板 + 运维人员

---

## 一、这是什么？

一个**企业内部 AI 视频 API 网关**。员工通过这个系统调用 AI 视频生成服务（Seedance 2.0），系统负责：
- 统一管理 API Key（员工不需要直接接触 Key）
- 计费追踪（谁用了多少、花了多少）
- 并发控制（防止超用）
- 任务管理（创建、查看、取消视频任务）

**核心价值**：老板掌控所有 API Key，员工通过系统调用，安全 + 可控 + 可追踪。

---

## 二、部署前准备

### 硬件要求
- 一台 Windows 电脑（安装 WSL + Docker）
- 或一台 Linux 服务器
- 最低 4GB RAM，20GB 磁盘

### 软件要求
- Docker + Docker Compose
- 网络能访问 `ark.cn-beijing.volces.com`（火山方舟 API）

### 需要准备的凭证
1. **火山方舟 API Key**（用于 Seedance 2.0 视频生成）
   - 获取方式：https://console.volcengine.com/ark → 创建 API Key
2. **管理员密码**（系统会自动生成，见下方）

---

## 三、一键部署步骤

### 步骤 1：进入部署目录
```bash
cd D:\Codex创业任务\企业 API 管理后台项目\02_source\sub2api\deploy
```

### 步骤 2：确认环境配置文件
```bash
# .env 文件已预生成，包含所有生产密码
# 确认文件存在：
dir .env

# 如需自定义域名（启用 HTTPS），编辑 .env 修改：
# CADDY_DOMAIN=api.yourcompany.com
# TLS_EMAIL=your-email@company.com
```

> **注意**: `.env` 包含所有密码，切勿上传到任何公共平台。

### 步骤 3：启动服务
```bash
docker-compose -f docker-compose.wsl.prod.yml up -d
```

### 步骤 4：等待服务启动（约30秒）
```bash
# 检查服务状态
docker-compose -f docker-compose.wsl.prod.yml ps

# 应该看到以下容器都是 healthy：
# - sub2api-prod
# - sub2api-postgres-prod
# - sub2api-redis-prod
# - sub2api-caddy
```

### 步骤 5：访问系统

| 访问方式 | 地址 |
|---|---|
| 前端页面 | `http://localhost` (Caddy 默认) 或 `http://<服务器IP>:8080` (直连) |
| 管理后台 | `http://localhost/admin` |
| 登录账号 | `admin@sub2api.local` |
| 登录密码 | 见 `deploy/.env` 文件中的 `ADMIN_PASSWORD` |

> 如果配置了真实域名（`CADDY_DOMAIN=api.xxx.com`），Caddy 会自动申请 HTTPS 证书，用 `https://` 访问即可。

---

## 四、配置 Seedance 视频通道

### 步骤 1：登录管理后台
用管理员账号登录后，进入「视频网关」→「Provider 管理」

### 步骤 2：添加 Seedance Provider
- 名称：`Seedance 2.0`（或自定义）
- Provider 类型：`seedance`
- API Key：填入火山方舟的 API Key
- Base URL：`https://ark.cn-beijing.volces.com/api/v3`（默认）
- 默认模型：`doubao-seedance-2-0-260128`（默认）

### 步骤 3：测试连接
点击「测试」按钮，确认 API Key 有效

### 步骤 4：启用
将 Provider 状态改为「启用」

---

## 五、员工使用方式

### 方式 1：网页界面
1. 访问 `http://<服务器IP>` 或 `https://<域名>`
2. 注册账号（或管理员创建）
3. 进入「发起调用」
4. 填写视频描述（prompt）
5. 选择 Seedance 2.0
6. 点击「创建任务」
7. 在「任务列表」查看进度和结果

### 方式 2：API 调用（给技术同事）
```bash
# 1. 获取 API Key（管理后台 → API Key 管理 → 创建）
# 2. 调用
curl -X POST http://<服务器IP>/api/v1/video/tasks \
  -H "Authorization: Bearer <你的API Key>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "seedance",
    "task_type": "text_to_video",
    "prompt": "一只猫在阳光下打盹",
    "duration": 5,
    "aspect_ratio": "16:9"
  }'
```

---

## 六、日常运维

### 查看日志
```bash
docker-compose -f docker-compose.wsl.prod.yml logs -f sub2api
```

### 重启服务
```bash
docker-compose -f docker-compose.wsl.prod.yml restart
```

### 停止服务
```bash
docker-compose -f docker-compose.wsl.prod.yml down
```

### 数据库备份
```bash
# 使用内置备份脚本（默认保留 7 天）
./backup.sh

# 自定义保留天数
./backup.sh 14
```

备份文件存放在 `deploy/backups/` 目录，gzip 压缩。

建议加入定时任务（每天凌晨 3 点）：
```bash
crontab -e
# 添加：
0 3 * * * cd /path/to/deploy && ./backup.sh >> /var/log/sub2api-backup.log 2>&1
```

---

## 七、安全须知

1. **`.env` 文件包含所有密码，务必妥善保管，不要上传到任何公共平台**
2. 管理员密码请在首次登录后修改
3. 生产环境通过 Caddy 反代访问，不要直接暴露 8080 端口到公网
4. 定期更换 API Key
5. 所有密码已替换为随机强密码（非 dev 默认值）

---

## 八、常见问题

**Q: 任务一直显示 "queued" 不动？**
A: 检查 Provider 是否已启用，API Key 是否有效

**Q: 视频生成失败？**
A: 查看任务详情中的错误信息，通常是 API Key 额度不足或 prompt 不合规

**Q: 忘记管理员密码？**
A: 查看 `deploy/.env` 文件中的 `ADMIN_PASSWORD`

**Q: 如何添加更多员工？**
A: 管理后台 → 用户管理 → 创建用户

**Q: 如何启用 HTTPS？**
A: 编辑 `.env`，将 `CADDY_DOMAIN` 改为你的域名，`TLS_EMAIL` 改为你的邮箱，重启服务即可。Caddy 会自动申请 Let's Encrypt 证书。

**Q: Caddy / SSL 相关问题？**
A: 默认 `CADDY_DOMAIN=localhost` 是 HTTP 模式。设置真实域名后自动启用 HTTPS。确保障名 DNS 已指向服务器 IP。

---

## 九、联系方式

有问题找博士。
