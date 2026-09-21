# 依赖复核

- 全仓排除锁文件、package.json、构建产物与旧 .umi-production 后，五个依赖名在源码/配置/测试/脚本中无引用。
- 通过 pnpm --dir web remove 移除 @ant-design/graphs、@ant-design/x、@ant-design/x-markdown、@ant-design/x-sdk、highlight.js；锁文件同步裁剪，不做版本升级。
- go mod tidy -diff 实际仅涉及 go.sum：删除失效版本/工具依赖校验和，补充当前选中 golang.org/x/image 的 go.mod 校验和。执行 tidy 后 go.mod 无变化。
- Sentry 在 requestErrorConfig.ts 仍有消费且保留上传脚本，初始化未接入口问题延期，不作为无用依赖删除。
