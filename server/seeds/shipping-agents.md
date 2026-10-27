# 船代候选快照来源

- 来源：海管家 `GET https://anoingress.hgj.com/saas-support/data/item/1020`。
- 核对日期：2026-10-27；29 条与订单候选逐条核对，仅代表已观察字典，并非完整行业名录。
- 文件：`shipping-agents.hgj.20261027.json`；仅包含无凭据字典响应。
- SHA-256：`93dca1753a3db49847b6396853b12beac20d5f6772e6b16cd7ebbd86d193069c`。
- 映射：`name=value`、`name_en=enValue`、`sort_order=order`；英文原值为中文时忠实保留。
  `code=HGJ-<id>` 是内部导入标识，不是行业标准编码；本系统独立生成 UUID。
  `source=HGJ_SHIPPING_AGENTS_20261027`。源记录保留在此文件供溯源。
- 使用：正式迁移后执行 `pnpm run sync:shipping-agents`，或通过 `pnpm run sync:all` 同步。
  所有解析与写入均离线完成，无第三方登录、网络请求或定时抓取。
- 幂等规则：按船代类型和固定导入码只补缺，已有 ID、名称、英文、排序、来源和启用状态均保留。
