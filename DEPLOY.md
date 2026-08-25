# 部署 new-api 到 Render / Koyeb(免费海外 Web Service)

目标:把 new-api 跑到**海外免费实例**上,以便连通 `api.openai.com` / `api.anthropic.com` / `api.x.ai`(大陆服务器无法访问这三个域名)。

## 0. 为什么需要这一步

- 大陆网络(含腾讯云国内地域)无法直连 OpenAI / Anthropic / xAI。
- Render / Koyeb 的免费实例运行在海外,可作为 API 中转做**短期模型渠道连通性测试**。

## 1. 已经为你准备好的内容

| 需求 | 状态 |
|---|---|
| 读取 `PORT` 环境变量 | ✅ 已支持([main.go](../main.go) 读取 `os.Getenv("PORT")`,空时回退 3000) |
| `/api/status` 健康检查 | ✅ 已支持(返回 200 + `{"success":true,"data":{"version":...}}`) |
| SQLite 免外部数据库 | ✅ 默认(未设置 `SQL_DSN` 即用 SQLite) |
| 密钥不入库 | ✅ `.gitignore` / `.dockerignore` 已排除 `.env`、`*.db`、`backups/`、`logs/`、`.secrets/` |
| Dockerfile | ✅ 仓库根目录现成(多阶段构建:bun 前端 + Go 后端 + Debian 运行时) |
| 支付 | ❌ 不配置,无任何 Stripe 密钥 |

## 2. 前置条件(需要你自己完成,我无法代做)

1. **一个 GitHub 仓库**:把本项目推到你自己的 GitHub(当前 `origin` 指向 Gitee 上游,不要推那里)。
   ```bash
   git remote rename origin upstream            # 先改名保留上游
   # 在 GitHub 新建空仓库后:
   git remote add origin https://github.com/<你>/new-api.git
   git add . && git commit -m "prepare for paas deploy" && git push -u origin main
   ```
   > 确认提交里**没有** `one-api.db` / `.env` / `backups/`(已被 .gitignore 排除)。

2. **Render 账号**(github 登录)或 **Koyeb 账号**(github 登录)。

## 3. 部署方式 A:Render(推荐,最简单)

### 方式 A1:Blueprint 一键部署
1. Render 控制台 → **New +** → **Blueprint** → 连接你的 GitHub 仓库。
2. Render 自动读取根目录的 [render.yaml](./render.yaml),创建免费 Web Service。
3. `SESSION_SECRET` / `CRYPTO_SECRET` 会在首次部署时由 Render 自动生成(不入库)。

### 方式 A2:手动创建 Web Service
1. **New +** → **Web Service** → 选择仓库。
2. Runtime 选 **Docker**,Instance 选 **Free**。
3. 健康检查路径填 `/api/status`。
4. 添加环境变量:`SESSION_SECRET`、`CRYPTO_SECRET`(各自填一段随机字符串)。
5. **不要**设置 `SQL_DSN` / `REDIS_CONN_STRING`(保持 SQLite)。
6. Deploy。

## 4. 部署方式 B:Koyeb

```bash
# 安装 CLI 后:
koyeb deploy new-api \
  --git github.com/<你>/new-api \
  --builder dockerfile \
  --port 8000 \
  --health-check-type http --health-check-port 8000 --health-check-path /api/status \
  --env PORT=8000 --env SESSION_SECRET=<随机串> --env CRYPTO_SECRET=<随机串> \
  --region fra
```
> Koyeb 免费实例默认端口 8000,`--env PORT=8000` 让应用监听 8000。也可在 Koyeb 控制台用 Docker 方式部署,同样设置上述环境变量和健康检查路径。

## 5. 部署后的渠道配置与连通性测试

免费实例是**全新空数据库**,需在部署后的管理后台重新配置渠道:

1. 浏览器打开部署地址,用初始管理员账号登录(首次启动会在日志里打印初始账号密码;new-api 默认 `root` / `123456`,登录后立即改密)。
2. 后台 → 渠道 → 新增渠道,分别填入真实 Key:
   - **GPT**:类型 OpenAI,BaseURL `https://api.openai.com`,模型如 `gpt-4o` 等。
   - **Claude**:类型 Anthropic,BaseURL `https://api.anthropic.com`,模型如 `claude-sonnet-5` 等。
   - **Grok**:类型 OpenAI(xAI),BaseURL `https://api.x.ai`,模型 `grok-4` / `grok-4-0709` / `grok-3-beta`。
3. 每个渠道点「测试」,验证从海外实例到 `api.openai.com` / `api.anthropic.com` / `api.x.ai` 的连通性。
4. 真实 Key 保存在本地 `.secrets/channel-keys.txt`(已 gitignore),可从这里取用,勿提交。

## 6. 免费实例的重要风险(务必知悉)

- **休眠冷启动**:免费实例无流量一段时间会休眠,下次请求需几十秒冷启动。
- **数据丢失**:免费实例磁盘是**临时的**,SQLite 数据库(`one-api.db`)在实例重启/重新部署/休眠恢复后**可能被清空**,已配置的渠道、用户、令牌全部丢失。
- **仅适合短期测试**:不要把生产 Key 长期、唯一地存在免费实例上;测试完请迁移到付费实例或香港服务器并启用持久化存储(PostgreSQL + 持久卷)。
- **临时域名**:免费实例域名是平台分配的,重启通常不变但非固定;正式使用请绑定自己的域名。

## 7. 生产化(测试通过后再做)

- 数据库换成 PostgreSQL,Redis 做缓存(见 [docker-compose.yml](./docker-compose.yml))。
- 挂载持久化磁盘 / 使用 Render Persistent Disk 或 Koyeb Volume。
- 绑定正式域名 + HTTPS(见 [ops/Caddyfile.example](./ops/Caddyfile.example))。
