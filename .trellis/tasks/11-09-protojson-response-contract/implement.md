# 实施计划

## 启动条件
规划文档和研究供用户审阅，用户在后续消息明确批准后才执行 task.py start。当前不得修改产品代码。

## 执行顺序
1. 使用 trellis-implement 子代理，加载 implement.jsonl 中规范及研究，所有产品修改均遵循先实现后针对风险补测试。
2. 提取主数据公共字段映射，迁移船代/国家/船司/港口/航司/机场调用点；保留专用字段规则。
3. 增补服务端真实 codec 契约测试、前端映射及主数据页面回归。测试覆盖省略零值、显式零值、非零、optional presence 及缺少真实必填字段。
4. 增加最薄复用门禁及负例测试，接入现有前端检查流程。
5. 主会话更新 AGENTS.md、type-safety.md 和 capability-navigation.md，明确适配边界与默认值语义。
6. trellis-check 子代理验证规范、测试有效性与未扩大范围，主会话审阅后提交中文 Conventional Commit。

## 验证
- 受影响主数据 mapper 与页面定向 Vitest（路径相对 web，按实体拆分避免单个重型 suite 长尾）。
- 修改文件 Biome、前端 tsc、静态门禁正反例、git diff --check。
- go -C server test ./internal/server -run TestProtoJSONCodec 验证零值与 presence 编码，不改实际编码策略。
- 最终 pnpm run check:fast（包含前后端检查，因补 codec 契约测试与跨实体公共映射需要覆盖）；不因本任务固定构建。

## 交付与回退
每组验证后提交，只暂存任务及本次产品文件，保留既有未跟踪海运布局任务目录。回退可按本任务提交处理，不触及数据库。研究中其他领域问题记录后续范围，不混入当前实现。
