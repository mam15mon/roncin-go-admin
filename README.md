# Roncin 国际货代协同管理平台 (Roncin Go Admin)

企业级国际货运代理协同与财务结算一体化管理平台。基于 **Go (Kratos v3 + Ent) + React 19 (Ant Design Pro + UmiJS Max) + PostgreSQL** 架构，单体仓库（Monorepo），支持前后端独立开发热重载，以及生产环境由 Go 服务单一二进制直接提供 API 与前端静态资源同域交付。

---

## 🌟 核心业务特性

- **海运出口订单管理**：海运出口 (FCL/LCL) 全生命周期状态流转与单证处理（提单、集装箱、POD 放货凭证）；海运进口、空运、陆运与铁路目前仅作为统一订单聚合的线路类型预留，录单入口尚未开放。
- **通用业务标签**：组织级标签字典可关联企业、订单、费用与账单，订单列表、费用台账与账单均支持标签展示、筛选和批量打标。
- **多币种费用录入工作台**：支持原币/本位币实时折算、汇率基准自动匹配、手动汇率重写与防重校验。
- **财务对账与结算中心**：
  - 双向应收/应付账单管理与批量合并账单 (`BillCreationWorkbench`)。
  - 增值税发票登记、红冲与税率分类。
  - 银行流水核销与智能对账分摊。
  - 业务员/操作提成自动化核算与多维度业绩归属。
- **行业基础主数据**：
  - 集成 ISO 4217 货币代码、标准箱型 (20GP/40HQ 等)、增值税税目。
  - 支持联合国 UN/LOCODE 全球海运港口、IATA 全球民航机场、民政部全国五级行政区划实时同步。
- **安全与权限体系**：基于 RBAC 的 200+ 个细粒度业务权限清单、组织机构隔离、支持企业微信与钉钉扫码登录。

---

## 🛠️ 技术栈

| 领域 | 核心技术选型 |
| :--- | :--- |
| **后端框架** | Go 1.25+, [go-kratos/kratos](https://github.com/go-kratos/kratos) v3, Protocol Buffers / gRPC / HTTP |
| **ORM / 数据库** | [entgo.io/ent](https://entgo.io), PostgreSQL 16 (支持 GIN 模糊全文检索) |
| **依赖注入与安全** | Google Wire, JWT Session, Argon2id 密码哈希, OpenTelemetry 链路追踪 |
| **前端架构** | React 19, TypeScript 7.x, [@umijs/max](https://umijs.org), [Ant Design](https://ant.design) 6.x, Ant Design ProComponents |
| **前端工具链** | Vite / Webpack, Biome (高极速 Lint & Format), Vitest |
| **包管理** | pnpm (Workspace Monorepo) |

---

## 📂 项目目录结构

```text
roncin-go-admin/
├── server/                     # 后端 Go 服务
│   ├── api/                    # Protobuf 定义及生成的 gRPC / HTTP 代码
│   ├── cmd/                    # 入口指令
│   │   ├── server/             # 主服务入口
│   │   ├── bootstrap-admin/    # 超级管理员与权限初始化 CLI
│   │   ├── migrate/            # 数据库版本迁移执行器（迁移后自动同步权限清单）
│   │   ├── sync-unlocode/      # 联合国港口代码同步 CLI
│   │   ├── sync-airports/      # 全球机场代码同步 CLI
│   │   ├── sync-regions/       # 全国行政区划同步 CLI
│   │   ├── export-permission-manifest/  # 导出权限码清单供前端权限键生成
│   │   └── generate-access-rules/       # 由 Proto 契约生成接口权限规则
│   ├── configs/                # 配置文件 (config.yaml, config.production.yaml)
│   ├── internal/               # 领域业务逻辑 (biz), 数据访问 (data), 传输服务 (service)
│   └── migrations/             # 数据库 SQL 版本迁移脚本库
├── web/                        # 前端 React / Ant Design Pro 应用
│   ├── config/                 # Umi 路由与构建配置
│   ├── src/
│   │   ├── components/         # 业务通用模板库 (UI 抽象与 Drawer/Modal 模板)
│   │   ├── pages/              # 业务页面 (海空运订单、财务结算、主数据管理等)
│   │   └── services/           # 自动生成的 OpenAPI 接口请求客户端
├── scripts/                    # 自动化与验收测试脚本
├── docs/                       # 架构设计与本地开发详细文档
├── .env.example                # 环境变量配置模板
└── package.json                # Monorepo 脚本集成入口
```

---

## 🚀 快速上手指南

### 1. 环境准备 (Prerequisites)

- **Node.js** >= `24.14.1` 及 **pnpm** >= `10.0`
- **Go** >= `1.25`
- **PostgreSQL** >= `16`

### 2. 克隆项目与安装依赖

```bash
git clone https://github.com/mam15mon/roncin-go-admin.git
cd roncin-go-admin

# 安装根目录及前端依赖
pnpm install
```

### 3. 配置本地环境变量

复制环境配置文件模板：
```bash
cp .env.example .env.local
```
编辑 `.env.local`，填入本地 PostgreSQL 密码及配置：
```dotenv
POSTGRES_DB=roncin_go_admin
POSTGRES_USER=roncin
POSTGRES_PASSWORD=your_local_password
DATABASE_SOURCE=postgresql://roncin:your_local_password@127.0.0.1:5432/roncin_go_admin?sslmode=disable
```

### 4. 数据库初始化

```bash
# 1. 执行数据库结构迁移与基础规则种子导入
pnpm run migrate:server

# 2. 初始化根组织与超级管理员账号 (自动挂载全量权限)
pnpm run bootstrap:admin
```
> **初始管理员账号**：`admin`  
> **初始管理员密码**：由 `.env.local` 中的 `BOOTSTRAP_ADMIN_PASSWORD` 指定（必须至少 12 位，命令不会使用任何默认密码）。

### 5. 启动本地开发服务

```bash
# 一键并行启动前后端（支持热重载）
pnpm dev
```
或分别在独立终端启动：
```bash
# 启动前端 (http://localhost:8001)
pnpm run dev:web

# 启动后端 API (http://127.0.0.1:8000 & gRPC :9000)
pnpm run dev:server
```

打开浏览器访问 `http://localhost:8001`，使用超管账号登录管理平台。

---

## 🌐 行业公共主数据同步 (可选)

系统初始化时已预置标准箱型、服务类型、货类与国家等基础字典。机场、港口等全球
标准字典通过以下官方数据链路按需同步：

```bash
# 1. 同步民政部全国 12 位五级行政区划 (省市区县街道)
pnpm run sync:regions

# 2. 同步 OurAirports 全球机场数据 (仅导入含合法 IATA 代码的记录，直接联网下载)
pnpm run sync:airports

# 3. 同步 UNECE UN/LOCODE 官方发布包 (仅导入海港；需人工下载 ZIP 后显式指定)
pnpm run sync:unlocode -- -source /path/to/loc251csv.zip -release 2025-1
```

---

## 💻 常用开发指令

### 代码生成与权限同步

```bash
# 当修改 protobuf 或 openapi 定义后，重新生成前端 TypeScript SDK 客户端
pnpm run generate:web-client

# 权限清单 (access.Manifest) 由 pnpm run migrate:server 在迁移后自动同步并补挂至系统管理员；
# 如需在不执行迁移的情况下手工同步，可单独运行：
pnpm run dev:permit
```

### 代码检查与自动化测试

```bash
# 全项目一键质量检查 (前端 Lint/Test + 后端 Test/Vet/Vulncheck)
pnpm run check

# 前端代码静态检查与单元测试 (Biome + Vitest)
pnpm run check:web

# 后端代码检查、单元测试与漏洞扫描
pnpm run check:server

# 运行财务结算端到端验收测试套件
pnpm run acceptance:finance
```

---

## 📦 生产构建与独立部署

项目支持编译为**单二进制独立部署**模式：前端生产构建产物会自动注入 Go 二进制的内嵌资源系统 (`server/internal/webassets`)。

```bash
# 1. 一键全量生产打包
pnpm run build

# 2. 生成的独立执行文件位于 server/bin/
ls -lh server/bin/roncin-server
```

生产运行：
```bash
./server/bin/roncin-server -conf server/configs/config.production.yaml
```
启动后服务将在单一端口上同时提供完整的前端 Web 页面与后端 API 服务。
