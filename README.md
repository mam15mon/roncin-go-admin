# Roncin 国际货代协同管理平台 (Roncin Go Admin)

企业级国际货运代理协同与财务结算一体化管理平台。基于 **Go (Kratos v3 + Ent) + React 19 (Vite + React Router + Ant Design Pro) + PostgreSQL** 架构，单体仓库（Monorepo），支持前后端独立开发热重载，以及生产环境由 Go 服务单一二进制直接提供 API 与前端静态资源同域交付。

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
- **安全与权限体系**：基于后端统一 Manifest 的细粒度 RBAC 权限，支持本地账密、钉钉扫码与钉钉客户端免登。
- **多组织工作区**：顶栏切换当前工作台，页签与表单草稿按用户与工作台隔离；进入公司须有有效成员资格，角色不再配置数据访问范围。公司经营数据限定当前公司，系统工作台负责公共资料与授权治理，经营业务须切换公司后按具体权限办理。
- **订单附件登记**：当前可登记外部对象存储引用及文件元数据、解除订单引用；订单附件入口尚不提供二进制文件上传与下载。

业务入口已开放不代表全部真实场景完成验收。海运出口的当前验收口径、既有脚本限制和运行结果记录要求见[海运出口订单闭环验收](docs/plans/海运出口订单闭环验收.md)。

---

## 🛠️ 技术栈

| 领域 | 核心技术选型 |
| :--- | :--- |
| **后端框架** | Go 1.26.9+, [go-kratos/kratos](https://github.com/go-kratos/kratos) v3, Protocol Buffers / gRPC / HTTP |
| **ORM / 数据库** | [entgo.io/ent](https://entgo.io), PostgreSQL 16 (支持 GIN 模糊全文检索) |
| **依赖注入与安全** | Google Wire, JWT Session, Argon2id 密码哈希, OpenTelemetry 链路追踪 |
| **前端架构** | React 19, TypeScript 7.x, React Router v8, React Query v5, [Ant Design](https://ant.design) 6.x, Ant Design ProComponents |
| **前端工具链** | Vite 7, Biome (Lint & Format), Vitest；构建与测试配置入口为 `web/vite.config.ts` |
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
│   ├── config/                 # 集中式路由、代理、主题与 OpenAPI 配置
│   ├── vite.config.ts          # Vite 构建与 Vitest 测试入口
│   ├── src/
│   │   ├── components/         # 业务通用模板库 (UI 抽象与 Drawer/Modal 模板)
│   │   ├── pages/              # 业务页面 (海运出口、财务结算、主数据管理等)
│   │   ├── features/           # 跨页面共享的领域能力，经 index.ts 公开
│   │   └── services/           # 自动生成的 OpenAPI 接口请求客户端
├── scripts/                    # 自动化与验收测试脚本
├── docs/                       # 架构设计与本地开发详细文档
├── .env.example                # 环境变量配置模板
└── package.json                # Monorepo 脚本集成入口
```

---

## 🚀 快速上手指南

### 1. 环境准备 (Prerequisites)

- **Node.js** >= `24.14.1`，**pnpm** 版本按根 `package.json` 的 `packageManager`（当前为 `10.33.4`）
- **Go** >= `1.26.9`（以 `server/go.mod` 为准）
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

# 2. 创建唯一初始管理员及独立公司，补齐系统管理与启用公司的管理员授权
pnpm run bootstrap:admin
```
> **初始管理员账号**：`admin`  
> **初始管理员密码**：由 `.env.local` 中的 `BOOTSTRAP_ADMIN_PASSWORD` 指定（必须至少 12 位，命令不会使用任何默认密码）。

`bootstrap:admin` 首次运行要求账号表为空，创建唯一标记的初始管理员；以后重复运行
定位该管理员，补齐系统管理及所有实际启用公司的成员关系和 `administrator` 绑定，
不修改密码/姓名、不替换普通同名账号、不删除已有其他角色。重复补齐不要求密码或姓名；
提供的用户名必须匹配原管理员，系统工作台编码须与原配置一致。
账号表非空但没有或有多个初始管理员、初始账号/系统资格停用或身份配置冲突时明确拒绝。
停用公司、公司管理员角色或原公司成员关系保持停用，逐公司报告原因，需人工恢复后重跑。
可处理项和审计在同一事务提交，数据库错误整次回滚；成功摘要区分新增、已满足、跳过。

后台新建公司要求初始管理员完整有效的系统成员、管理员角色及绑定，缺失时先执行 bootstrap；
公司与其管理员成员/绑定在同一事务建立。系统工作台必须唯一；角色已失效的 `self` 范围
不会自动升级，公司跳过、系统资格则整次拒绝。后来手工授予
系统管理员角色的其他账号不会自动加入。部门/团队复用所属公司工作台，不重复加入。
默认分公司种子仅包含北京（`BJ`）和成都（`CD`），初始化不会补建上海或重庆公司；
自建启用公司也在补齐范围。仅补默认公司可用 `pnpm run bootstrap:admin --seed-branches`。

公共港口、机场、船公司等行业资料通过 `pnpm run sync:all` 导入。系统管理员在
“系统管理”工作台维护公共资料及公司授权；初始管理员通过真实公司成员关系进入公司，使用该公司权限。

开发演示数据在完成 `pnpm run bootstrap:admin` 后执行 `pnpm run sync:dev`，仅用于开发数据库。它写入北京公司的测试员工、
客户、订单和财务数据；不应在生产库执行。系统费用模板用于新公司创建时初始化，
后续修改模板不自动覆盖已有公司的费用配置。

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

系统初始化时已预置标准箱型、服务类型、货类与国家等基础字典。行业资料可用
`pnpm run sync:all` 一键同步，也可按需执行以下命令。港口默认读取仓库归档包，机场
默认从 OurAirports 下载；具体来源与参数以对应 `server/cmd/sync-*` 入口为准。

```bash
# 1. 同步民政部全国 12 位五级行政区划 (省市区县街道)
pnpm run sync:regions

# 2. 同步 OurAirports 全球机场数据 (仅导入含合法 IATA 代码的记录，直接联网下载)
pnpm run sync:airports

# 3. 同步 UNECE UN/LOCODE 海港数据（默认读取 server/seeds/loc251csv.zip）
pnpm run sync:unlocode

# 使用其他发布包时，显式指定路径与版本
pnpm run sync:unlocode -- -source /path/to/loc252csv.zip -release 2025-2
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
# 全栈验收优先使用并行门禁（前端与后端并行）
pnpm run check:fast

# 前端代码静态检查与单元测试 (Biome + Vitest)
pnpm run check:web

# 后端代码检查、单元测试与漏洞扫描
pnpm run check:server

# 开发过程中按改动范围选择定向测试，完整命令与阶段要求见 AGENTS.md
```

既有业务验收脚本与单元测试门禁分开使用。运行前核对当前契约、目标数据库和数据写入范围；
集成测试的跳过结果不计为真实数据库验证通过。海运出口脚本的已知限制见
[验收说明](docs/plans/海运出口订单闭环验收.md#既有自动化脚本与限制)。

### 继续开发的规范入口

- [协作约束](AGENTS.md)：变更边界、生成流程、验证和提交要求。
- [领域地图](.trellis/spec/domain/architecture-map.md)：核心业务链路与代码定位。
- [服务端规范](.trellis/spec/server/backend/index.md)：分层、事务、权限和领域契约。
- [前端规范](.trellis/spec/web/frontend/index.md)：页面模板、请求状态和能力复用。

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
