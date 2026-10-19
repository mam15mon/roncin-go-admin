# 钉钉通知免登验收记录

## 交付行为

钉钉内业务链接无有效会话时进入客户端免登；单工作台返回原完整路径，多工作台明确选择后返回。普通浏览器保留扫码/账号登录。SDK 失败、空码、超时和账号状态均明确提示，支持显式重试/账号登录；换码写 Cookie 与多工作台选择期间锁定方式切换。

## 代码与规范证据

- 后端/生成物：`11aa13c8`；前端/SDK依赖：`3fdc5fab`。
- 公开协议：GET `/api/v1/auth/dingtalk/embedded-config`、POST `/api/v1/auth/dingtalk/embedded-login`；独立 state Cookie、no-store、复用人员准入与会话创建。
- JSAPI 固定 SDK3.2.9 旧版 `runtime.permission.requestAuthCode({corpId})`，后端固定企业应用 token + v2 getuserinfo；必填 unionid/userid/name，身份双字段匹配，无 OAuth 回退/自动建人/权限扩大。
- 新执行规范：`.trellis/spec/server/backend/dingtalk-embedded-login.md`；前后端索引、扫码/会话规范与构建测试辅助目录约束同步。
- 原生成器复跑 `make -C server api` → `pnpm run generate:web-client` 退出0，生成物无差异。

## 验收标准映射

| 标准 | 证据 |
| --- | --- |
| 钉钉内无会话自动认证，单工作台完整回跳 | 实际 Chromium 业务路径401→免登→目标query/hash，配置/SDK/换码各1次，无扫码 |
| 多工作台明确选择再回原目标 | Chromium选择第二工作台，switch1次；组件测试含当前组织直入、其他组织轮转及失败留页 |
| PUBLIC、camelCase、state、Cookie、账号与资格边界 | 实际 Kratos HTTP 路由含 Authorization/RequiredFieldsValidator；缺Cookie/错state/扫码Cookie不可替代/缺字段/坏JSON/禁用/未同步/待分配/停用/无资格/身份冲突及单多工作台 |
| 单一授权码协议及固定企业 | data HTTP替身断言token申请、固定getuserinfo路径/JSON code，无OAuth或详情补读，身份必填/错误安全映射/token缓存 |
| SDK失败/超时/StrictMode/迟到/切换 | SDK空结果、原生失败、不返回15秒超时、导入超时后迟到不native；StrictMode单次与卸载/切账号/旧换码/旧轮转结果隔离 |
| 钉钉内无扫码预热、普通浏览器保持 | Go/Vite/main定向测试，Chromium钉钉qr0/iframe0；普通浏览器扫码config1/SDK0 |
| 生成物、review、完整验收、构建 | 定向测试、独立review、全栈门禁通过；生产构建退出0、产物扫描通过，Git交付按下述流程核验 |

## 独立 review 及修复

- `832cca75`：JSON解码/必填校验在Service前拒绝时也强制no-store，HTTP回归覆盖缺state/坏JSON/上游零调用。
- `e77ac978`：换码前同步父子ref门禁，防旧响应Set-Cookie与账号登录并行覆盖；成功后保持锁到组织选择/导航，失败settle恢复，SDK等待仍可切。
- `98bc83d4`：补动态import超时后迟到不调用native的精确回归。
- `035bfb03`：新测试helper移到web/tests；构建进一步发现既有测试辅助也命中页面扫描，`06225ec9` 追加实际命名负glob排除，最终核对产物。
- 审阅未发现自动建人、按通知URL猜组织、协议回退、秘密日志或手改生成物。

## 检查结果

- 后端定向测试：biz `Auth.*DingTalk`、data `DingTalk`、service/server/webassets；五包vet、Proto lint通过。HTTP含正式中间件，数据上游为HTTP替身。
- 前端定向：实施首轮13文件127用例；Cookie修复相关5文件、SDK9用例、helper迁移3文件23用例均通过；独立review另行复跑相关用例。修改文件Biome、tsc、架构检查通过。
- `pnpm run check:fast`：退出0，前后端全门禁通过；前端261文件通过、1既有文件跳过，1434用例通过、12既有跳过；后端全包测试/vet、分层/Proto/漏洞检查通过。耗时86.5秒。
- 全门禁后仅测试helper目录和页面glob排除调整，按影响复跑定向路由、helper测试/tsc及生产构建，全部通过。
- 最终 `SENTRY_AUTH_TOKEN= pnpm run build` 退出0：Web构建21.85秒，Go服务构建通过；315个SourceMap源路径扫描未发现Vitest、Vite服务端运行时或测试helper，辅助模块chunk为0，union/requestAuthCode保留独立按需chunk。未上传SourceMap；仅保留既有大chunk提示。
- 真实 Chromium 协议模拟：7项PASS、全部pageerror0，覆盖完整回跳、多组织、390px失败重试、切账号后迟到SDK、普通浏览器、真实npm模块加载及旧API调用、换码在途两个入口禁用。
- 当前开发服务：live/ready均HTTP200；新公开配置HTTP200/success=true，必需字段存在、独立state Cookie及no-store；只取配置，未取真实客户端码或发送消息。

## 实际环境验证边界

没有调用真实钉钉客户端、没有真实用户免登或主动发送测试通知。客户端测试的HTTP/SDK或native bridge为替身；它们证明协议与程序行为，不代表可信域名、网页应用能力、应用发布及可见范围已经真机验收。真实用户收件与免登需要现有钉钉应用配置及可访问部署地址。

本期没有Schema/权限/配置字段变更，未修改或清空数据库。默认后端测试中的数据库集成用例受环境变量控制，未启用真实隔离库验证，不将Skip表述为数据库通过。

前任务全部8类Markdown通知作为本次依赖保留，消息仍使用固定本站费用业务路径，未修改发送条件或收件规则。

## Git交付流程

本记录与规范作为工作提交完成后，由任务脚本归档并记录开发日志；随后将当前分支快进合并至main，验证Markdown依赖和免登分支均已成为main祖先，使用 `git branch -d` 安全删除两个已合并的本地分支，推送origin/main并核对本地、远程跟踪和远程实际HEAD一致。远程核验结果在最终交付说明中报告。
