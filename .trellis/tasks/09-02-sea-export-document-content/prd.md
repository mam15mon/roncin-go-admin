# 海运出口主分单内容

## Goal

实现 DIRECT/HOUSE、完整 MBL/HBL 内容、多分单与签发主体页面

## Requirements

- 依赖 `09-02-sea-export-mbl-foundation` 已完成、检查并提交；只扩展阶段 1 的共享 MBL 和成员关系。
- 实现 DIRECT（无 HBL）和 HOUSE（一张或多张真实 HBL），不创建虚拟/隐藏 HBL。
- MBL 与每张 HBL 分别维护完整提单内容；多张 HBL 内容和签发主体互相独立。
- HBL 格式开放，保留原始值；签发主体无默认选择，必须明确选择本公司、委托单位或其他真实主体。
- 提单信息在新建与编辑页默认展开；以“主单 MBL / 各 HBL”紧凑切换，可显式复制上一张 HBL 内容，复制后不联动。
- 本阶段不实施箱货定量分配、外部状态、不可变版本、Switch 或费用分摊。

## Acceptance Criteria

- [ ] DIRECT 可保存完整 MBL 内容且不存在 HBL 记录。
- [ ] HOUSE 支持一票多 HBL，每张内容和签发主体独立。
- [ ] HBL 原始号码无损往返，唯一性按签发主体和规范化键判断。
- [ ] 提单信息默认展开，复制后修改任一 HBL 不影响其他主分单。
- [ ] 阶段 1 共享 MBL、订单和财务回归测试通过。

## Notes

- 后续依赖：`09-02-sea-export-cargo-allocation`。
