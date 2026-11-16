# 空运订单公共经营字段

## 1. 适用范围

AE/AI新建和详情的公共经营字段UI、默认值、载荷和回显；入口为订单域内的`templates/components/air/AirCommercialFields.tsx`与`order-kinds/air-export/form-adapter.ts`。AI定义组合共有映射并显式传入AI/IMPORT，不逐页复制公共字段；进口边界见[空运进口闭环](../../server/backend/air-import-contract.md)。航空运输、MAWB/HAWB正文与重量确认仍由各自契约管理，不以公共货值或付款方式代替单证声明/印字。

## 2. 签名与字段

- `getAirCommercialFields(props: TemplateProps)`：返回字段片段（`shipmentMode/trade/payment/contract/agents/times/cargoValue/insurance/factory`），写法同海运 `getSeaBaseInfoFields`，由模板按海运同序摆放：
  - 业务与客户：客户、单号片段复用海运，后接 `shipmentMode`；
  - 订舱与运输首行：航空单证模式、主单关联方式（仅新建）、订舱号、`agents`；
  - 货物与商业：品类 + `trade/payment/contract` 同行，件重尺 + `times` + `cargoValue/insurance/factory` 同一 6 列网格。
- `AirCargoDescriptionFields`：委托描述/特殊要求对半。
- `AirNotesFields`：订舱/配舱/操作/普通四类备注，可折叠，位于订舱与运输卡「备注」小节；主单备注写共享主单，随新建主单字段出现，不并入此处。
- `airExportFormAdapter.buildCreatePayload/buildDetailInitialValues/buildUpdatePayload` 是公共字段往返入口。
- 服务端继续用已有`CreateOrderRequest/UpdateOrderRequest`及Order本票字段；本组无新增Proto或Schema。

## 3. 输入与存在性合同

- 货值与保费保持独立的金额/币种对；复用CurrencyAmountInput验证和原始字符串，`'0'`不丢失，不将formatAmount展示写回请求。
- DatePicker回显用Dayjs，提交用ISO时刻；字段清空发送既有optional出现性协议的显式空字符串，不省略成undefined而保留旧值。
- 更新可清文本、代理ID、金额/币种以空字符串清除，枚举以生成UNSPECIFIED零值清除；新建未填可选项继续省略，不伪造值。
- 航空取消危险品后，更新载荷必须显式清空UN、CLASS和截申报时间，并携带
  `dangerousPackingGroup=UNSPECIFIED(0)`清除旧PG；创建未选仍省略。
  控件只写`undefined`再让JSON省略，会被后端optional合并解释为保持旧值，
  不能以危险品区块隐藏证明数据已清除。后端不从货类猜测清空，沿明确写入意图处理。
- 业务付款方式提供可见选择，不能与海运货描组件的隐藏付款字段重复注册；本票四类备注只在运输卡「备注」小节出现一次，货物卡不再放备注。
- 候选复用订单域现有供应商/国外代理能力和权限，不添加新主数据副本；工厂是本票文本，不新建工厂平台。
- 只读状态传递给所有新控件；普通业务字段不写入共享AirMaster或空运正文。
- 本期公共文本上限按Unicode字符：客户/内部业务号及合约100、工厂200、各备注1000。
  biz的`utf8.RuneCountInString`、Ent的`MaxRuneLen`及控件业务限制必须一致，不能
  在Ent使用按UTF-8字节的`MaxLen`拒绝合法中文边界。该调整只改变输入验证，
  PostgreSQL varchar字符尺寸不变时不创建空迁移；必须用真实PG验证中文边界和超限。

## 4. 校验与错误矩阵

| 场景 | 结果 |
| --- | --- |
| 金额为字符串0且币种存在 | 正常保留显式零 |
| 只填金额或只填币种 | 金额控件成对校验，服务端保留最终校验 |
| 清空已有合约/代理/金额/日期 | JSON载荷保留空字符串清除意图 |
| 创建未填可选字段 | 请求可省略，不从主单/创建时间回退 |
| 日期跨本地时区午夜 | 同一时刻ISO往返，不截字符串伪装转换 |
| 修改本票货值/保费/代理 | 不自动创建经营费用或覆盖已存单证印字 |

## 5. 正常、基础与错误例

- 正常：保费`123.4567`和币种分别保存，详情仍可编辑原始4位小数；展示两位只作用于渲染。
- 基础：用户清空已存工厂，更新发送`factoryName:''`，刷新后仍为空。
- 错误：以`value?.trim() || undefined`序列化更新清空，导致optional字段缺席，服务端沿用旧值。

## 6. 必需验证

- `order-kinds/air-export/form-adapter.test.ts`：原始金额/显式零、optional省略、ISO与Dayjs、创建与更新差异、JSON序列化后清空仍存在。
- `templates/air-create-layout.test.tsx`：真实挂载新字段、业务付款只注册一次、编辑/清空可取得值且不写入airMaster。
- 前端变更运行定向Vitest、修改文件Biome与必要tsc；最终仍需整票API及浏览器保存验收，不把adapter测试当全业务通过。
- 公共文本边界须同时核对biz与生成Ent验证：100中文编号/合约、200中文工厂、1000中文备注可落库，超限1字符拒绝，清空与原值保持不变。
- AE/AI危险品选择/取消后验证真实JSON及API重读：未设置optional保持旧值、显式空串/零清空是两条不同路径，首次初始化不得误清。

## 7. 错误与正确

```ts
// 错误：更新空字段在JSON中消失，旧值清不掉。
const update = { contractNo: values.contractNo?.trim() || undefined };
// 正确：使用领域adapter按更新合同保留显式清空意图。
const update = airExportFormAdapter.buildUpdatePayload(id, version, values);
```
