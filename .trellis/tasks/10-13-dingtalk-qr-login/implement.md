# 实施计划：默认钉钉扫码

## 前提与顺序

- [ ] 用户批准完整方案后先做本子任务的协议/真实应用前置验证；保留代码事实和现场结果，不读取/输出秘密。
- [ ] 取消主要组织、批量成员与管理员补齐完成后，在最终认证契约上完成本页面交付。
- [ ] 激活任务并加载前端栈、Ant Design、React Query 及认证规范。

## 实施

1. [ ] 根据官方脚本协议验证 OAuth2 iframe 与当前回调，确认消息来源/格式及 state。任何改变“打开即扫”或认证边界的限制都先回到规划。
2. [ ] Proto 增加配置有效期，service 使用同一次 state 的过期时间并禁用缓存；生成 API/Web。
3. [ ] 实现就近的扫码 iframe 组件与配置 Query、消息校验、清理/刷新/到期处理。
4. [ ] 默认扫码布局及管理员密码模式切换，复用现有回调和工作台选择，不改页面品牌资源。
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
