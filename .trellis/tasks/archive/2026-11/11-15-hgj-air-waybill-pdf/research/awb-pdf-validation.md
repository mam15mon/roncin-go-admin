# 航空 PDF 渲染与海运相邻验收

本报告属于本次 renderer 子组，所有样张只使用伪造资料，保存在 `/tmp/roncin-awb-pdf-validation/`，不进入 Git。未取得海管家原始 PDF，本组交付的是已批准的一套航空 A4 分区版式，不声称复刻外站 PDF。

## 实现边界

- `web/src/pages/orders/awb/pdf/`：冻结的 `AwbPdfModel`、模板修订 `2026-11-16.1`、航空分区文档、数值/声明/字符校验、按需预览及风险测试。正文为生成 `API.AirWaybillContent` 的编译期结构类型引用，没有协议运行时依赖。
- `web/src/pages/orders/documents/pdf/`：真实共用的同域字体注册、字符覆盖/字宽 JSON、保守折行与同一 `usePDF` Blob 展示；不包含航空/海运业务模型或法律语义。
- HBL 保留字体包装、业务内容函数及其原 API；预览消费同一 `PdfBlobView`。自签四版、SI 外部归档仍使用各自原文档，未把航空深引 HBL 专属版式。HBL 公司测试只增加按环境变量保存合成样张的入口。
- 编辑器、版本/主体来源、保存/重读/路由/权限和服务端由 core 子组负责；本报告不替代其验证，也不声称已完成真实浏览器生产验收。

## 纸面与存在性

所有 55 个当前 `AirWaybillContent` 字段均在普通 fixture 中有明确输入，并经逐字段核对覆盖收发/账号/通知/代理、路线航班、账务/处理/SCI、币种/声明、唛头/货描、运价、其他费用、PPD/COLL 汇总、目的地换算、声明/签发/备注。金额消费 `formatAmount` 两位，数量/单价/汇率消费 `trimDecimal`；纯日期保持原 `YYYY-MM-DD`。`0.00`、未知 `-`、NVD/NCV/XXX 和 AS ARRANGED 分别表示，不把未知或按约定运费写成零。

主体只消费已冻结模型：本公司 AE HAWB 与外部 MAWB/AI 主分资料分别使用 `SELF_ISSUED`/`EXTERNAL_ARCHIVE`。外部归档明确标记已登记事实，不产生签章、放货、外部发送或业务状态流转。正文保留已存 issuedBy，不由 PDF 根据当前工作区猜测。

页眉只放固定目标标题/归档副题；页脚是 32 字符号码摘要、保存版本与页码，原号码及真实主体全文留在可分页正文。每行运价的七个数字/代码列保留表格，其品名紧随该行全宽输出，避免 10000 字品名挤在窄列产生大量无意义续页。其他费用的所有明细和字段也完整保留。

## 已执行检查

1. `AWB_PDF_VALIDATION_DIR=/tmp/roncin-awb-pdf-validation pnpm --dir web exec vitest run src/pages/orders/awb/pdf src/pages/orders/hbl/pdf --reporter=default --reporter=json --outputFile=/tmp/roncin-awb-pdf-validation/vitest-final.json`
   - 12 个文件、29 个测试全部通过，整体 15.91 秒。
   - 三个航空规模文件分别为货物 3.487 秒、party 4.369 秒、200 行表 15.083 秒；HBL 极值 10.798 秒，均未超过 20 秒。
   - 预览测试验证同 URL、关闭/版本/主体/正文/上层指纹卸载、失败不输出旧 URL、重试、字符/主体阻断。它们 mock renderer 生命周期，真实 PDF 另以本节后述方法验证，不将 mock 视为浏览器下载证据。
   - 本轮混合 node 渲染与 happy-dom 预览时，普通 AWB 和既有 HBL node 渲染测试出现 `act` 警告；此前单独 node 执行未出现。新增规模 helper 已将真实渲染包在 `await act(...)` 内。已向 Root/checker 移交五个普通 node 渲染测试的同类收口，未通过静音或关闭全局规则掩盖；此结果不能称为零噪音门禁，后续收口以 checker 报告为准。
2. `pnpm --dir web exec biome check src/pages/orders/awb/pdf src/pages/orders/documents/pdf src/pages/orders/hbl/pdf/content.ts src/pages/orders/hbl/pdf/font.ts src/pages/orders/hbl/pdf/HblPdfPreview.tsx src/pages/orders/hbl/pdf/HblPdfDocument.company.test.tsx`：通过。
3. `pnpm --dir web tsc`：最终通过；首轮仅 core 在途 `awb/model.ts:67` 两处可选字符串错误，已由其修复，无 PDF 类型错误。
4. `pnpm run check:architecture`：18 个架构规则测试通过，产品扫描通过，无跨模块反向依赖。
5. `antd lint web/src/pages/orders/awb/pdf/AwbPdfPreview.tsx --format json` 和 `antd lint web/src/pages/orders/documents/pdf/PdfBlobView.tsx --format json`：均为零问题。写控件前已查询 Drawer/Button API 与 Button demo。
6. `git diff --check`：通过。没有安装新依赖、提交私有 PDF、提交 Git 或推送。

## 独立提字与实际图片

执行 `PYTHONPATH=/tmp/roncin-hbl-pdf-tools python3 /tmp/roncin-awb-pdf-validation/verify.py`，使用环境已有 PyMuPDF 从最终 PDF 独立提字、按水平坐标重组列文本、核对源字段全文与全部明细品名/费用名称，并检查重复中文源字符数量下界、每页页眉/保存版本页码和字框边界。辅助脚本、逐页 PNG、纯文本和 `results.json` 留在同一私有目录。

| 私有 PDF 文件 | 页数 | 独立提字字符数 | 已核对重点 |
| --- | ---: | ---: | --- |
| `normal-HAWB.pdf` | 2 | 3150 | 本公司主体、全部分区、显式零与未知、纯日期、数量/单价/汇率有效精度 |
| `normal-MAWB.pdf` | 2 | 3153 | 外部归档分区和目标身份，不引入海运法律模板 |
| `long-parties.pdf` | 15 | 35633 | 6 个接近 5000 字的收发/通知/代理/印发正文、长原号全文及末标记 |
| `long-cargo.pdf` | 14 | 39850 | 接近 10000 字的唛头/品名/单条运价品名、接近 5000 字备注全文 |
| `many-rows.pdf` | 19 | 37897 | 运价与其他费各 200 行，全部行名及 `RATE-ROW-200-END`、`OTHER-ROW-200-END`，NVD/NCV/XXX、AS ARRANGED |
| `hbl-draft.pdf` | 1 | 1834 | HBL 草稿及真实英文主体 |
| `hbl-original.pdf` | 1 | 1843 | HBL 正本及真实英文主体 |
| `hbl-surrendered.pdf` | 1 | 1878 | HBL 电放标识及真实英文主体 |
| `hbl-copy.pdf` | 1 | 1865 | HBL 副本及真实英文主体 |
| `hbl-archive.pdf` | 1 | 300 | SI 外部主体/版本/显式零，保持归档语义 |
| `hbl-archive-long.pdf` | 7 | 13098 | SI 全文、末货描和第 110 个箱号 |

以上共 64 页，独立字框检查均无页面边界溢出；航空正文位于左右 30pt/底部预留范围内，HBL 按其原 18pt 页边界检查，不把海空版式混为同一边距。长号码为 renderer 压力输入，不表示业务 API 接受带多次换行的规范 MAWB 号码。

已实际 `view_image` 查看普通主分样张、全部 48 页极值航空的逐页联系图（每格为一页），并查看 HBL 四版及 SI 归档普通/7 页长文。逐页原图为 `<stem>-01.png` 等，联系图为 `long-parties-contact-1..4.png`、`long-cargo-contact-1..4.png`、`many-rows-contact-1..5.png`、`hbl-archive-long-contact-1..2.png` 和 `hbl-four-variants-contact.png`。中文真实字形、表格边界、末行和固定页脚清晰，未仅依赖 React 布局树的 Text 节点。

曾对 `long-cargo-14.png` 的整页查看提出页眉疑点，双方均补做定量核对：最终 PDF 该页标题/副题字框分别在 `(30,23,...,42.2)`、`(30,46.2,...,55.8)`；同 PDF 整页 raster 的顶部 `x=28..300,y=20..60` 有 1198 个暗像素，各航空页相同，没有低墨页。原 PNG 与重新 raster 的 `debug-long-cargo-14.png` SHA256 都为 `28855787cbfdf09ac61dbc6e6045bbd747d3657a3461ed082ae2dc3a3649ca22`，单独裁剪 `debug-long-cargo-14-clip.png` 实图完整显示两行页眉，故未确认产品丢页眉问题，也未凭整页展示差异盲改产品。

## 首轮失败及修复

- 首轮 Biome 报静态数组行的 index key：改为先构造只读行对象的稳定 ID 再渲染，未关闭规则。
- 首个合并极值 fixture 一次生成 40 页，用时 51.5 秒；全树按阅读顺序拼接会穿插左右两列末行，导致 `SHIPPER-END` 判断失败。改为按实际水平坐标收集，并用独立 PyMuPDF 逐列全文核对，未删掉源文字断言。
- 规模用例已按 party、货物、200 行表拆文件。原 20% 窄列的 10000 字运价品名造成单个货物 suite 34 秒超时，改为每行品名全宽；最终三个规模文件均小于 20 秒，不靠延长超时掩盖长尾。
- `act` 混合环境警告如上，最终由 Root/checker 接续收口，不能将它遗漏为“全部无警告通过”。

## 后续边界

Web PDF 产品/测试写权已明确释放给 Root/checker。本组未执行生产 build、全栈 check:fast、真实浏览器多类型保存/409/下载/同域按需字体，它们由 Root/core/browser 统一执行并另记证据；本报告只交付 renderer 和相邻 HBL 的实际生成与独立文件核验，不声称物理打印机已测试。
