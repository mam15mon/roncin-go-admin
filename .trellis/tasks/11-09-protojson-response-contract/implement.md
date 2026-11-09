# 实施计划

## 启动条件
规划已由用户后续消息明确批准，task.py start 已执行，当前分支 codex/protojson-response-contract。用户追加交付要求：独立 review 完成后合并 main、安全删除任务分支并 push。

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

## 合并与推送
用户已授权合并、删除分支和 push。独立 review 与完整门禁通过后提交，归档本任务并记录日志；重新 fetch 核对 main 没有遗漏远端提交，合并至 main（可快进），确认任务提交已包含再用 git branch -d 删除任务分支，最后普通 push origin main，不强推。保留其他任务的未提交目录。

## 实施与验收结果（2026-11-09）
- 公共 mapPersistedMasterDataFields 已实现，国家/船代及四类专用目录统一组合，保留实体必填差异；航司 cargoOnly 省略按 false 映射。
- 主数据 map* DTO 复用门禁已接入 check:web，独立 review 收窄普通 helper 与内层函数作用域，7 项正反例通过。
- 服务端真实 codec 覆盖零值省略与 optional presence；前端映射与页面回归覆盖缺 key。
- 实现代理定向 Vitest 4 文件 40 项、tsc、修改文件 Biome、Go codec 均通过。
- 独立审查同样通过上述检查，并补国家/港口页面 wire 回归，最终治理文件 6 项通过；无剩余阻塞。记录见 research/independent-review.md。
- pnpm run check:fast 全栈通过：前端 393 个文件通过、1 文件跳过，2146 项通过、11 项跳过；后端测试、vet、分层、Proto lint 与漏洞门禁均通过。完整门禁启动后的 reviewer 补充由最终门禁 7 项和治理用例 6 项验证，无关闭规则或测试。
- AGENTS.md、类型安全规范和能力导航同步到实际签名及静态门禁范围；optional 业务存在性与纯展示空文本投影明确区分。
- 订单改配相关发现留在研究作为延后项，本次未修改订单业务。
