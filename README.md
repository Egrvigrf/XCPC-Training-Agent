# XCPC-Training-Agent

一个给队伍用的训练数据管理服务：把集训队队员在 Codeforces / AtCoder 上的训练与比赛记录统一抓取、落库，然后提供一套 API 做查询/管理，并且支持每天自动同步。

---

## 你能用它做什么

- **账号体系 + JWT 鉴权**
  - 支持登录普通用户自助登录、查看信息、改密码
  - 管理员接口有单独的权限保护，只有管理员有资格调度爬虫和创建用户

- **训练数据同步**
  - 支持管理员按时间区间批爬取比赛数据并且导入数据库（请不要频繁调度爬虫，以免给服务器造成过大压力导致被风控）
  - 同步逻辑为“覆盖式同步”，便于修复历史数据

- **每日自动任务**
  - 按 T+1 模式自动同步“昨天”的数据

- **容器化**
  - docker-compose 一键起 MySQL / Redis / app
  - 初始化 SQL 自动执行

---

## 未来希望增加的功能

- 数据计算和排名

- agent 分析训练数据

- 可视化前端

---

## 技术栈 / 结构

技术栈：
- Go + Gin（HTTP API）
- GORM（MySQL）
- Redis（目前用于基础依赖，后续可扩展缓存/锁）
- Python 爬虫（以子进程方式被 Go 调用，结果 JSON 回传）
- gocron 任务调度

目录简略说明：
- `internal/handler/api`：HTTP 层
- `internal/handler/task`：定时任务
- `internal/logic`：业务层（用户逻辑/数据管理）
- `internal/model`：数据库访问
- `internal/crawler`：Python 爬虫调用封装
- `sql/init.sql`：建表初始化  

---

## 快速开始（Docker）

### 启动依赖与服务

```bash
docker compose up -d
```

### 调用示例                  

1. 登录 root
curl -s http://localhost:8080/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"20001","password":"000000"}'

2. 批量创建用户（把 token 填到 Authorization）
curl -s http://localhost:8080/v1/admin/users/create \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{"users":[{"id":"示例学号","name":"示例姓名","password":"默认密码","cf_handle":"示例codeforcesID","ac_handle":"示例atCoderID"}]}'

3. 手动同步区间（只传 student_id）
curl -s http://localhost:8080/v1/admin/op/training/sync \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{"students":[{"student_id":"示例学号"}],"from":"2026-03-01T00:00:00+08:00","to":"2026-03-07T23:59:59+08:00"}'
