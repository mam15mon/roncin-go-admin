# 企业微信下线验收记录

## 实施与审查

用户批准方案并确认开发数据可删除。已在 `codex/retire-wecom` 完整下线企业微信前后端能力，保留用户和业务数据。前后端 Trellis implement 子代理分工实施，Trellis check 子代理全范围审查，无未修复发现。

生成通过 `make -C server api config`、`go -C server generate ./...`、`pnpm run generate:web-client`、`pnpm run generate:permission-keys`、`node scripts/generate-error-catalog.mjs`。再次生成 875 个服务端生成文件，指纹一致。历史 SQL、历史前态/下线回归夹具、Proto reserved 名称和其生成描述符保留企业微信引用；当前运行实现、配置、OpenAPI、前端客户端和权限能力无残留。

## 检查结果

| 验证 | 结果 |
| --- | --- |
| 前端定向 Vitest | 13 文件、98 用例全部通过，7.43 秒 |
| 前端 TypeScript / 修改文件 Biome | 通过；17 手写 TS/TSX 文件 Biome 无问题 |
| 登录/用户弹窗 Ant Design CLI | 无问题 |
| 后端定向六包测试、vet、Proto lint | 通过 |
| 独立 PostgreSQL 定向集成 | 真实执行通过，无 SKIP，约 32.7 秒 |
| 正式迁移升级与冷启动 | 通过；七张表保留字段/关联全行快照一致，检索来源重建，旧权限删除，有效权限保留 |
| 相邻流程集成 | 本地登录、钉钉身份、旧会话、组织切换、目录同步/开通并发、生命周期、系统公司授权边界通过 |
| HTTP 注册路由测试 | 三条企业微信路由 404，相邻本地/钉钉路由命中原处理器 |
| 全栈 `pnpm run check:fast` | 通过；WEB 58.7 秒，SERVER 127.8 秒，总 127.8 秒 |
| 全量前端测试 | 180 文件 / 1071 用例通过，1 文件 / 12 用例按现有条件跳过；不把 SKIP 计为通过 |
| `pnpm run build` | 前后端生产构建通过；前端约 20.92 秒，保留现有大 chunk 提示，未配置 Sentry Token 时跳过上传 |
| `git diff --check` | 通过 |

全栈门禁未注入集成库，因此不能把其中环境跳过的数据库用例视为实际数据库 PASS；上述独立 PostgreSQL 定向验证单独提供数据库证据。

## 开发库迁移

已执行 `pnpm run migrate:dev`，权限同步移除 1 项，用户检索键回填完成。

| 数据 | 迁移前 | 迁移后 |
| --- | ---: | ---: |
| 用户 | 34 | 34 |
| 组织 | 5 | 5 |
| 成员关系 | 17 | 17 |
| 角色 | 5 | 5 |
| 角色分配 | 11 | 11 |
| 企业微信绑定行 | 0 | 字段已删除 |
| 企业微信专属字段 | 2 | 0 |
| 企业微信唯一索引 | 1 | 0 |
| 企业微信授权权限 | 1 | 0 |
| 空用户检索键 | — | 0 |

没有重建数据库，没有删除用户或业务记录，没有自动生成密码或转换账号。

## 实际页面与服务验证

在与用户地址同一开发服务的 `http://127.0.0.1:8001` 使用无头 Chromium 实际登录，选择系统管理工作台并打开 `/admin?tab=users`。用户原 IP 地址无头访问遇到 ERR_EMPTY_RESPONSE，改用本机地址完成同一服务验证。

- 本地登录与组织选择通过；用户列表无「企业微信」列，钉钉通讯录面板及分配开通操作正常呈现。
- 当前用户列表 API 无 wecomUserid/wecomName，auth/me 无 system.user.authorize_wecom。
- 登录及用户页面不请求企业微信接口。
- 运行服务 GET 旧企业微信配置返回 404；POST 旧登录/授权返回 405。运行服务在根路径注册既有静态处理器，未命中的 POST 被该层拒绝；纯业务 Router 测试的三个端点返回 404。两者均证明旧业务处理器不存在，不为下线新增兼容路由。
- 页面截图：`/tmp/retire-wecom/admin-users.png`；运行验证结果：`/tmp/retire-wecom/runtime-validation.json`。

本地日志：`/tmp/retire-wecom-frontend-vitest-final.log`、`/tmp/retire-wecom-frontend-tsc.log`、`/tmp/retire-wecom-check-fast.log`、`/tmp/retire-wecom-build.log`。日志和截图不提交 Git。

## 交付边界

本次已更新当前开发环境，未推送远程、未合并 main、未部署生产。其他钉钉任务记录保持原状态。
