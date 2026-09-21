# 简化海运订单提单模式与分单展示

## 目标与授权
用户已确认上一轮方案并回复「好开始吧」，授权本轮建立任务并实施已审阅范围。降低 SE 订单提单配置理解成本，DIRECT 不再展示无用 HBL 区域。

## 已确认事实
- sea-template.tsx 的 SeaCreateHouseBillFields 在 DIRECT 下返回占位文字，外层 houseBillContent 分节仍显示。
- SeaCreateDocumentModeField 已在切换 DIRECT 时清空 seaHouseBill。
- 签发主体目前有本公司、委托单位、其他主体三个合法值。

## 范围与需求
1. 模式标签改为「提单模式」，选项为「有货代分单（HOUSE）」「仅船公司主单（DIRECT）」，保留模式选择入口。
2. DIRECT 隐藏整个 HBL 分节、页签、导航入口和占位提示，保留 MBL；HOUSE 正常展示 HBL。
3. 签发主体改为「分单由谁签发」，选项「我们公司签发」「委托单位签发」「指定其他合作方签发」，增加简短说明「请选择 HBL 上显示的签发主体」。仅其他合作方显示合作伙伴选择。
4. 保持既有默认值、签发身份解析、权限与模式变更流程，不猜测默认签发主体。

## 验收
- 初始 DIRECT 及 HOUSE 切到 DIRECT 均无 HBL 区块、空状态或导航项，MBL 与模式选择仍在。
- DIRECT 无分单必填校验，提交不携带 HBL；切回 HOUSE 正常展示并校验分单。
- 新建、草稿/详情相关文案一致；详情既有模式切换权限和预览流程不变。
- 定向测试、修改文件静态检查和风险匹配前端门禁通过。

## 边界与验证说明
预计修改 sea-template.tsx、SeaCreateDocumentModeField.tsx、HouseBillIdentityFields.tsx、SeaDocumentSection.tsx，以及实际消费分节可见性的订单表单组合层与相关测试。先利用已有分节可见性机制，不新增通用抽象。不改后端、契约、生成物、数据库或签发默认策略。用户已有 web/src/components/ui/form-navigator/FormAnchorNav.tsx 改动不得覆盖或提交。本任务无未决产品问题。

## 实施与验收记录
- 模板增加最薄 visible(values) 条件，同一过滤结果驱动卡片与导航；仅订阅可见性变化，普通输入不触发整体重渲染。
- 实际新增公共模板与海运可见性两组测试，覆盖动态分节、初始 DIRECT、双向切换、校验解除、程序回填、resetTo 及加载后草稿恢复。
- 独立审查完成，新增测试引用类型与异步断言已修正。
- pnpm run check:web 退出 0：163 文件通过、1 文件跳过；921 用例通过、12 用例跳过。包含类型、静态、架构及生成物一致性检查。Biome 存量警告保留。
- git diff --check 通过。本次为本地提交任务，无 PR；归档使用脚本提供的 --skip-branch-validation。
