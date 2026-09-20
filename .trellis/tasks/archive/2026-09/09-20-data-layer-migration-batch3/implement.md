# 执行计划：数据层迁移第三批

- [x] 1. 批次 A：`useMasterDataCrud` 迁移 + 5 面板/MasterDataTemplate/
      hook 测试适配与 Provider 补齐（API 兼容零改动；reload 签名收敛
      Promise<void>；update 变量类型精确化；hook 测试 1→5 用例含
      重渲染稳定性回归断言）。
- [x] 2. 批次 B：`useCreditLimitIntervention`（silent）、fees 标签筛选
      （防抖关键词进 queryKey + 渲染侧保活合并）迁移；
      `ProFormSearchableSelect` 判定豁免（pro 原生 request 管理，
      spec 已记录）。
- [x] 3. spec 增补事件驱动型请求处置约定（豁免清单 + 命令式搜索收敛
      模式 + 小型策略查询约定）。
- [x] 4. 收口：全量 vitest 873/12 双零；tsc 通过；biome 通过；
      分批提交（A=08407069、B+spec=4a7df6af）。
- [x] 5. `pnpm run check:fast` 终验通过；归档 + 日志。

## 备注

- design.md 中「原 reloadStats 失败静默」表述与代码事实不符（实际有
  message.error 文案），实现按事实等价保留动态文案，design 不再修正
  （任务即归档）。
