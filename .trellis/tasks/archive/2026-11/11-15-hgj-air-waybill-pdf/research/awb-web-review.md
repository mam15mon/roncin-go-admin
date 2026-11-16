# 航空制作前端独立全范围复核

日期：2026-11-16。复核本组全部非生成Web变更：航空制作路由/入口、制作页与模型、保存和探测、航空PDF版式/预览、中立字体/Blob工具、HBL共用迁移及相邻测试。服务端与生成物另有独立复核，本报告不替代最终check:fast、生产构建和真实浏览器。

## 已修复

### 制作模型漏验主单模式

`awb/model.ts` 原来只检查本票 `AirOrderInfo.documentMode` 是否为DIRECT/HOUSE，未核对真实主单模式和内嵌主单模式；原模型用例甚至把本票改为DIRECT而主单仍为HOUSE，并断言MAWB可用。这会让契约损坏的模式组合进入制作模型。

现读取边界要求主单模式合法，本票关联模式、内嵌主单模式与真实主单一致。缺失/非法/不一致明确失败，不以旧mock补默认。已修合法DIRECT夹具，并补充缺失/非法主单模式、本票模式冲突、内嵌模式冲突及合法HOUSE相邻回归；合法DIRECT无HAWB仍可制作MAWB。Root已确认这一局部合同补强。

### 实际PDF测试的异步React生命周期

`awb/pdf/AwbPdfDocument.test.tsx` 与HBL的 `HblPdfDocument.test.tsx`、`HblPdfDocument.company.test.tsx`、`HblPdfDocument.overflow.test.tsx`、`HblArchivePdfDocument.test.tsx` 中，实际 `renderToBuffer` 全过程现包在 `await act(...)`。混合happy-dom预览与node实际PDF时不再出现未包act的stderr；没有静音console、关闭全局flag或更改产品代码。

### 字体再生文档漏同步

`web/public/fonts/roncin-hbl/README.md` 仍让字体覆盖表写回已经迁走的HBL私有目录。已更新为 `orders/documents/pdf/fontCoverage.json` 及同目录字宽表，并同步HBL/AWB共用、四种海运输出/归档与航空实际渲染验证范围；字体文件、字体内部名称、授权和SHA均未改。

## 全范围实码核对

- 集中routes提供AE/AI实际读取权限、标题及切公司列表落点；DIRECT无分单菜单，手动HAWB入口同样拒绝。普通订单入口不静态导入PDF renderer，页面成功读取严格固定当前公司/类型/ID/有效版本。
- 订单MAWB首次验证本票归属后固定真实主单身份；LoadedAwb与AwbEditor均规范化主单独立保存快照，保存/探测不会因多出Order字段误判，也不借更晚Order版本。HAWB始终沿真实本票/主单/Link/House原基线。
- MAWB保持打开时运输副本，只改正文，走Preview及完整共享影响确认；HAWB最薄更新不复制经营或运输字段。两个保存分支都复验确切响应身份、版本和正文；后续版本、409与在途新输入不覆盖本地草稿。
- 前台15秒、回焦、hidden/blur/离页和卸载保护符合现行规范；公司英文名纳入探测及输出指纹，即使Master版本不变也使旧稿失效。新实例owner与刷新session隔离A→B→A迟到响应。覆盖dirty前确认，失败保留；刷新途中继续输入不会被旧响应覆盖。
- AE HAWB取真实公司英文名；MAWB与AI主分取各自保存issuedBy，缺主体允许维护但阻输出。保存、签发事实和输出分开，不引入状态流转/费用/外部发送。只读及锁定票按读取权限输出保存快照，更新同时要求真实公司办理权。
- 输出生成前重读快照，未保存内容不伪装新版本。预览/下载/打开打印共用usePDF URL；改变单证、主体、模板、上层指纹或重试均重建，关闭卸载由usePDF回收。
- 版式覆盖保存内容分区，长正文按字宽拆短行分页；运价数值列与该行全宽品名保持对应，两个200行表无前N行裁切。金额/数量汇率/纯日期与声明tokens沿领域格式化，零与未知分开。固定页眉页脚有界，原号/主体全文保留在可分页正文。
- 中立PDF目录仅持共用字体、字形与断行、Blob生命周期，无海空业务模型或法律语义。HBL字体/内容包装API和四种自签/外部归档继续使用原各自文档。新目录的两份字形JSON与HEAD旧目录版本逐字节一致。

## 本人实际验证

| 检查 | 结果 |
| --- | --- |
| 核心模型/保存/输出/刷新/真实ReactQuery hook定向：`pnpm --dir web exec vitest run src/pages/orders/awb/model.test.ts src/pages/orders/awb/AwbEditor.save.test.tsx src/pages/orders/awb/AwbEditor.output.test.tsx src/pages/orders/awb/AwbEditor.refresh.test.tsx src/pages/orders/awb/use-awb-data.test.tsx` | 5文件28测试PASS，14.96秒；模型新增模式回归已覆盖，组件每文件不超过6个重用例 |
| 混合实际PDF与预览：`pnpm --dir web exec vitest run src/pages/orders/awb/pdf/AwbPdfPreview.test.tsx src/pages/orders/awb/pdf/AwbPdfDocument.test.tsx src/pages/orders/hbl/pdf/HblPdfPreview.test.tsx src/pages/orders/hbl/pdf/HblPdfDocument.test.tsx src/pages/orders/hbl/pdf/HblPdfDocument.company.test.tsx src/pages/orders/hbl/pdf/HblPdfDocument.overflow.test.tsx src/pages/orders/hbl/pdf/HblArchivePdfDocument.test.tsx` | 7文件19测试PASS，11.72秒；无act/stderr提示 |
| `pnpm --dir web tsc` | 最终PASS；首次新模式条件的一处可空master类型提示已改可选链后复验 |
| 对本组航空制作、PDF、中立目录、HBL目录、入口及routes执行Biome | 44文件PASS，无修复残留 |
| `pnpm run check:architecture` | 18项测试及575产品文件/2342本地依赖边扫描PASS |
| antd lint：AwbEditor、awb页面、AwbPdfPreview、PdfBlobView | 四文件均0 issues，partial=false |
| 字形覆盖/字宽JSON移动内容比较 | 两文件均与HEAD原文件逐字节一致 |
| `git diff --check` | PASS |

核心测试仍有既有Node `TimeoutNaNWarning`，没有失败断言，也没有为隐藏提示更改通用计时器。PDF混跑日志没有该提示或act警告。

## 沿用证据与未执行

renderer已完成普通主分2+2页、长party15页、货物14页、各200行19页及HBL四版/归档1和7页的实际PDF与独立提字/逐页实图，字框overflow为0；详见 `awb-pdf-validation.md`。本人实码核对其版式及验证入口，未无因重跑全部规模suite。

本次自修只改变模式边界校验、测试生命周期和字体再生文档，不改PDF产品版式或生成物，无需重做全部已过极值样张。生产build、全栈check:fast、真实API浏览器保存/下载/同域字体由Root后续统合。没有把组件mock预览或布局树当作真实浏览器证据，没有验证物理打印机。

## 交接

全部本组非生成Web产品/测试已冻结并释放给Root。无未修复确认问题；未提交或推送，未改根spec/任务主文档，未重生成客户端或修改服务端。
