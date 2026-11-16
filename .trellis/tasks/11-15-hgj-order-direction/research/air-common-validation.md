# 空运公共经营字段本组验证

## 已实现边界

- 只补当前AE表单公共经营字段UI及创建/详情/更新双向映射；不改航空正文、单证模式和进口入口。
- AirCommercialFields放在订单模板内，只消费本票字段及既有候选/金额控件；不复制公共字段到主单或单证JSON。
- 金额原始字符串含显式零和4位精度保留，时间回显Dayjs、提交ISO时刻；更新清空采用既有服务端optional出现性协议的显式空串/枚举UNSPECIFIED。

## 实现代理已运行结果

- `pnpm --dir web exec vitest run src/pages/orders/order-kinds/air-export/form-adapter.test.ts src/pages/orders/templates/air-create-layout.test.tsx`：2文件17测试通过。
- `pnpm --dir web tsc`：通过。
- 修改文件Biome、Ant Design CLI lint及`git diff --check`：通过。

独立检查已完成并修复货值/保费两个金额控件遗漏`requireAmountWhenCurrency`的问题。真实Form用例验证只选币种失败、零/4位小数成功及金额币种同时清空成功；修复后重跑上述定向17例、五文件Biome、Antd lint、tsc及diff检查全部通过。尚未用真实浏览器/API保存完成整票验收；不据这组通过宣称航空直单或进口已实现。

## 后续回归

- 制作结构化航空正文时不能把本组本票货值/保费或业务付款方式当AWB声明/印字费用。
- SI/AI后续模板与adapter需复用真实公共字段映射并继续保留显式清空意图，不通过undefined掩盖旧值。
