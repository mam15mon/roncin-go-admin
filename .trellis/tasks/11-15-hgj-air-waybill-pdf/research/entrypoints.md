# 航空制作与PDF前置定位

这是AI验收期间的只读定位，不代表PDF组已开始实施或输出通过。父design已批准一套A4航空
版式加续页、真实主体/进口归档语义、主分独立保存版本及同Blob生命周期。

## 真实航空源

- `api/order/v1/air_export.proto`、`biz/air_waybill_content.go` 已完整定义正文，
  主单 `content.waybillContent`、分单 `house.content` 各自独立，哈希/版本和历史已建立。
  `issuedBy/agentText/shipperSignature/carrierSignature` 等为保存印字，不是电子签章。
- `templates/components/air/AirWaybillFields.tsx` 已有真实结构化表单入口，
  包含声明令牌、原始decimal、路线印字、运价行、其他费用及派生PPD/COLL汇总。
  制作页组合它，不复制字段校验或把印字费用当经营费用。
- 主单保存沿Preview→Update及完整成员锁；分单沿本票版本/Link版本和明确正文保存。
  制作页目标应明确Master/House ID、真实业务类型、当前公司及打开版本，
  不能仅按号码查找或用最新重读版本自动替换提交基线。

## 可复用工程入口及边界

- `hbl/HblEditor.tsx`、`hbl/use-hbl-data.ts`：保存快照、变动提示、轮询和未保存输入保护。
- `hbl/pdf/HblPdfPreview.tsx`：按需renderer、同一个usePDF URL供iframe/下载/打印、
  重试及随快照/模板重建。新航空输出不复制HBL正本/电放法律语义。
- `hbl/pdf/font.ts`、`content.ts`、`fontCoverage.json`、`fontGlyphWidths.json`：
  实际同域中文字体注册、字符覆盖校验和按真实字宽拆行。字号与容器高度需看最终PDF，
  布局树有文字不代表实际绘制可见，SI组已真实踩坑并在规范记录。
- HBL/AWB均属于同一orders页面模块，真实共用的字体/字形工具或Blob展示可就近移到
  中立的订单单证目录，保持最小公开面。遵当前目录规范，不只为同模块两个页面建立
  跨领域features平台；空运不得深引 `hbl/...` 的专属模型/版式/法律语义。
  如移动公共实现，HBL全入口及4版本与外部归档一起改、验收架构与相邻输出。

## 必须保留的数值及主体语义

运输/海关/保险声明为AMOUNT或NVD/NCV/XXX的明确类型，AS ARRANGED不算0。
运价行及费用最大200行，长party/唛头/货描/条款/备注按既有航空长度合同，输出不能裁切。
金额展示两位、数量/比例/汇率保精度，纯日期不做时区转换，保存载荷仍原始字符串。
已保存 `issuedBy` 及真实航司/公司事实决定主体；不能以当前工作区冒充外部签发方。
进口归档不虚称本系统办理签发、放货或外部发送；不新增电子签章/航司模板市场。

## 验收定位

实际PDF用环境已有 `/tmp/roncin-hbl-pdf-tools` PyMuPDF及逐页实图；普通与极值、多行200行、
中文、声明零/文本、PPD/COLL/AS ARRANGED、末行和边界均核对。至少AE/AI、DIRECT/HOUSE
的适用主分输出及页面保存失败/409/后续版本/同Blob有真实证据。
新增打印入口和共用字体移动后跑生产build及同域深链/JS/字体/未知asset404，
只跑前端布局测试或看到PDF文件头不算打印通过。
