# 港口名称与同步现状

## 已有调查产物

绝对目录：`/home/vince/.codex/visualizations/2026/10/27/01a17be3-5817-7cd1-967e-a8bb0aa74d6b/port-research/`，包括原始港口候选样本 JSON、142 条 CSV、50 个同码中文候选 CSV 和调查报告。均为字典数据，不含凭据。第三方临时登录资料已删除。

## 当前实现定位

- UNLOCODE 解析：`server/cmd/sync-unlocode/main.go`，只取 Function[0] == 1 的海港，排除别名并按变更优先级选行。
- 同步记录与落库：`server/internal/data/industry_reference_sync.go` 的 PortSyncRecord / ApplyPorts；统一 Data.WithTx，已有记录只更新基础英文数据，不设置中文。
- 中文检索：`server/internal/data/ent/schema/port.go` 的 name_zh、name_en 使用 searchKeywordsHook；`server/internal/data/search_keywords_backfill.go` 有既有中文空检索键回填。
- 开发数据：`server/cmd/sync-dev/main.go:329` 仅为 9 个示例港配置中文，已有记录会跳过。

## 数据源初步线索（尚未确认完整覆盖或可导入性）

- UNECE UNLOCODE：现有 ZIP 是代码和海港身份底表，不提供本项目需要的中文列。https://unece.org/trade/uncefact/unlocode
- 中国电子口岸提供港口代码参数查询，需调查当前数据结构、国标映射和实际覆盖，不能假定与 UNLOCODE 一致。https://www.chinaport.gov.cn/pages/parameter/parameter-list.html
- 深圳单一窗口官方手册说明港口参数可包含中文、英文及国标代码映射；这是进一步寻找完整名称来源的线索，不是当前已获得的数据文件。https://sz.singlewindow.cn/dyckfile/news/file/customerService/201807131630129897096.pdf
- 其他商业字典或海管家样本可用于名称参考，既有代码体系不一致；没有证据表明能独立覆盖全部 17,524 条。

## 规划状态

用户已把范围从样本改为全量，并接受缺少通用中文名时采用翻译/音译及来源标记。PRD、设计和实施计划已收敛，用户已以“开始吧”批准实施，现已生成全量中文数据并完成同步与验收，详见 verification.md。

## 全量规模与实现约束补充

现有 ZIP 包含 232 个国家/地区海港、17,073 个不同原始名称。中国 765、日本 1,562、美国 1,762 条；名称编制需按国家语义处理，不以一套英语逐字母音译规则全量替代。PortSyncRecord 增加中文后直接复用 Port searchKeywordsHook 更新检索键。portSyncConflicts 拒绝其他来源占用的标准码，继续保留该显式错误。
