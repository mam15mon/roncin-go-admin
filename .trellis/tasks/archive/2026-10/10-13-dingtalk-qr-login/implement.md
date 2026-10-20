# 实施计划：默认钉钉扫码

## 前提与顺序

- [ ] 用户批准完整方案后先做本子任务的协议/真实应用前置验证；保留代码事实和现场结果，不读取/输出秘密。
- [ ] 取消主要组织、批量成员与管理员补齐完成后，在最终认证契约上完成本页面交付。
- [ ] 激活任务并加载前端栈、Ant Design、React Query 及认证规范。

## 实施

1. [ ] 根据官方脚本协议验证 OAuth2 iframe 与当前回调，确认消息来源/格式及 state。任何改变“打开即扫”或认证边界的限制都先回到规划。
2. [x] Proto 增加配置有效期，service 使用同一次 state 的过期时间并禁用缓存；生成 API/Web。
3. [x] 实现就近的扫码 iframe 组件与配置 Query、消息校验、清理/刷新/到期处理。
4. [x] 默认扫码布局及管理员密码模式切换，复用现有回调和工作台选择，不改页面品牌资源。
5. [ ] 实现后补针对性测试及浏览器体验检查，确认真实认证与组织选择。
6. [ ] 更新认证规范和 verification.md，完成门禁后提交。

## 检查命令

```bash
make -C server api
pnpm run generate:web-client
go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run 'DingTalk|Auth' -count=1
pnpm --dir web exec vitest run src/pages/user/login/index.test.tsx src/pages/user/login/dingtalk-callback.test.tsx src/pages/user/login/components/organization-picker.test.tsx
pnpm --dir web tsc
git diff --check
```

新增扫码组件测试用实际路径加入定向命令；组件重型测试超过 10 项或 20 秒拆分到生命周期/消息类文件。改动文件 Biome 检查。最终 check:fast；若修改生产 CSP/入口/构建配置，则补 build 并验证同域静态页面。

## 验收证据

- [ ] 默认无账号密码表单、二维码可直接扫；次级切换前后可用。
- [ ] 单/多工作台，管理员密码与员工扫码均正确。
- [ ] pending/disabled/not-synced 人员状态没有被改成自动开通。
- [ ] 伪造 origin/source/state、旧 iframe消息和重复回调不触发认证。
- [ ] StrictMode、快速切换和连点刷新不产生冲突 state；失败不隐藏错误。
- [ ] 真实企业扫码、开发/生产回调地址已分别记录验证结果；未完成必须明确列出限制。


## 2026-10-13 实施记录

已实现配置 expires_at（同次 state）、响应 no-store/Pragma；默认真实 OAuth2 iframe 加 iframe=true，次级管理员密码入口和返回扫码。使用固定 Query key、不消费 AbortSignal、禁自动重试及焦点/重连刷新，手动刷新先同步卸载旧实例，再串行 refetch。消息精确核对 origin、非空 iframe window、回调路径、单个 state/authCode；失败、过期、超时和成功立即退役，回调仅导航本站固定路径。单/多工作台和人员状态沿用现有链路。既有测试仅同步夹具及陈旧按钮行为，没有新增或运行。

生成：make -C server api、pnpm run generate:web-client；生成差异为 auth.pb.go、OpenAPI、Web输入及typings。

实现代理非测试检查通过：定向Go build、service vet、web tsc、三个变更TSX的Biome、两产品组件antd lint（零问题）、diff --check。

真实企业二维码显示、本人手机扫码、开发代理/生产同域回调、StrictMode与快速切换浏览器实测尚未执行，不把源码协议或静态检查视为这些验收通过。无实际库、企业接口或环境秘密操作。
