# 航空运单真实浏览器与PDF验收

## 最终结果

2026-11-16：正式SQL迁移后的真实PostgreSQL隔离库、真实后端和浏览器整例通过；AE/AI × DIRECT/HOUSE的六份适用MAWB/HAWB实际下载PDF共16页，独立PyMuPDF提字、末行和逐页图片核对通过。生产Go同域深链/资源/实际AWB和HBL预览通过，本次临时数据库、角色、进程及端口全部清理并独立核验。

实际执行：

```bash
RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-awb-browser-acceptance/run.mjs
```

最终退出码0，内部执行`pnpm --dir web exec playwright test tests/e2e/air-waybill.e2e.ts`，结果`1 passed (1.7m)`；后处理输出`6份适用主分输出，合计16页`。最终证据绝对目录为`/tmp/roncin-awb-browser-Oczm1d`，目录0700、原PDF/截图/日志0600。产品和生成物在最终轮前冻结，新生产构建由主会话完成，本组未修改产品代码。

## 真实场景结果

| 场景 | 实际结果 |
| --- | --- |
| 双业务双模式 | API创建AE/AI DIRECT和HOUSE共四票，DIRECT无HAWB实体，详情更多操作无HAWB制作入口 |
| 六份适用正文 | AE DIRECT MAWB、AE HOUSE MAWB/HAWB、AI DIRECT MAWB、AI HOUSE MAWB/HAWB，全部真实UI改正文备注并保存、刷新重读；保存版本分别推进，主分货描独立未互相覆盖 |
| MAWB共享保存 | 按正式Preview→“确认主单正文共享修改”→“确认影响并保存”→Update；填写明确修改原因，服务端复核真实成员与指纹，成功返回新版本 |
| HAWB保存 | 按Order/Link/House原版本提交真实本票正文；保存成功后只采纳确切保存响应 |
| 真主体 | AE HAWB抬头为实际经营公司英文名`ISOLATED REAL AIR COMPANY`；MAWB及AI主分按已保存`EXTERNAL ISSUER ...`文字归档，PDF中未借当前公司名称冒充外部主体 |
| 数值/声明 | wire重读零金额`0`、空账号、费率`6.7183`；PDF保留0.00/123.45、12.345KG、15.5KG、1.25数量及6.7183费率/换算汇率；NVD/NCV/XXX与AMOUNT0/AMOUNT123.45分别验证；PPD/COLL明确，AI AS ARRANGED及未确定总额保留未知语义 |
| 普通与长样张 | 普通五份各2页；AE HOUSE HAWB长发货人25行、收货人20行、唛头20行、货描45行、8运价+8其他费用共6页。全部长文末行和第8行运价/其他费用末行完整 |
| 同Blob输出 | 六份均实际打开iframe、下载文件；iframe src、下载href、打开打印href相等，未用独立重算副本 |
| 在途新输入 | 延迟本次真实HAWB PUT但不伪造响应，pending期间继续输入，真实200后新输入保留且服务端仍为已提交原值；页面提示继续编辑、禁止输出未保存稿 |
| 真实409 | 延迟本次真实PUT，独立管理员API会话推进Order.version后放行请求，服务端真实409；编辑原文保留，服务端未写入失败正文，无假预览 |
| 15秒/回焦 | 独立API更改后分别等待真实15秒轮询和回焦检查，持续提示“这个航空运单已经发生变动，请重新刷新。”；编辑原文及原基线保留 |
| 不推进业务事实 | 每份预览/下载后重读，flow、Order.version、主分签发状态未改变，经营费用仍空 |
| 生产同域 | 8010 AWB深链返回SPA200/html，主JS200/javascript、PDF字体200且font/octet-stream MIME、未知assets404、health/ready200 |
| 按需加载 | 直接进入生产普通订单首屏，网络无PDF渲染器/预览模块/字体请求；打开制作并请求输出后才生成PDF |
| 生产实际预览 | 8010真实AWB HAWB实际生成/下载，原SE自签HBL输出版本控件可见并实际生成预览 |
| 页面异常 | 最终整例收集pageerror为空 |

制作路由为`/orders/{air-export|air-import}/:id/awb/{MAWB|HAWB}`；主单独立路由为`/orders/{kind}/masters/:masterId/awb`。当前实际制作输出使用新构建Go服务和同域字体，不依赖外站。

## PDF文件、页数与SHA-256

实际PDF位于最终目录的`playwright/air-waybill.e2e.ts-AE与AI适用主分运单制作、保存快照、真实PDF与生产按需加载/`子目录。

| 文件 | 页数 | SHA-256 |
| --- | --- | --- |
| `ae-direct-mawb.pdf` | 2 | `0fdd5b01a595b742feb34d2ac265c6df1b31d5c896b12cbc31b23130ec4d843a` |
| `ae-house-mawb.pdf` | 2 | `585e05c2debb1442e444f429ae52eec3ffd592925e5959f2a8270c7e493411a2` |
| `ae-house-hawb.pdf` | 6 | `0a574006d34fd6f2ed99b1d72c0805196102af14001de7114324879190b75aa1` |
| `ai-direct-mawb.pdf` | 2 | `42de0456ef944d9c5786ff35dc2d3106611858e0b0a5a2607ca8a60986d18671` |
| `ai-house-mawb.pdf` | 2 | `0df21ff6e6688a4d096ed7a957169a49a211ad782e77fe5efee6f20a8e19177d` |
| `ai-house-hawb.pdf` | 2 | `9900b29c0dc00e8667f190113fd91647fa36f5ece9360e07f262c3b68a7f735c` |

同目录另有`production-ae-house-hawb.pdf`（生产实际下载）、预览与并发输入保护截图及`awb-pdf-manifest.json`。独立提字汇总为最终根目录`awb-pdf-validation.json`；每份PDF的提字和逐页图片放在其同名stem目录。

实际实图核对：首轮同源码六份16页均逐页查看高清PNG；最终轮16页联系表`/tmp/roncin-awb-browser-Oczm1d/all-16-pages-contact-sheet.png`也已实际查看，版式和内容一致。中文及中英文对照清晰，长行正常换行，连续页无重叠或裁切，页脚保留目标单号、保存版本及页码。长样末货描在第5页、末费用在第6页；AI归档AS ARRANGED、未知总额横杠与真实外部主体可见。逐页文件绝对示例：

- `/tmp/roncin-awb-browser-Oczm1d/ae-house-hawb/page-6.png`
- `/tmp/roncin-awb-browser-Oczm1d/ai-house-hawb/page-2.png`

## 资源隔离与清理

编排`/tmp/roncin-awb-browser-acceptance/run.mjs`复用已验收的最薄资源管理：随机AWB前缀库/角色、正式有序SQL迁移、随机本地管理员、真实Go验收配置/Vite、单文件Playwright、finally/信号进程组清理。只管理本次追踪资源，不修改开发库或账号，不停止8001未知进程；通知和对象存储关闭。

最终随机资源`roncin_acc_awb_browser_1794797952916_3ebaa406`：harness校验owner后DROP库与角色，三个资源追踪集合归零。另独立执行SQL确认指定`pg_database`计数0、`pg_roles`计数0，`ss`确认8010/9010/8002均无监听。

## 检查与首轮后处理修正

- 最终`pnpm --dir web tsc`：通过。
- `pnpm --dir web exec biome check tests/e2e/air-waybill.e2e.ts`：通过。
- `git diff --check`（本组两文件）：通过。
- `/tmp`编排`node --check`：通过。

首轮Playwright已`1 passed (1.9m)`，但后处理默认按行提字把表格第一列名称的续行与其他列插入交错，误报费用末行缺失。经实际坐标与高清图确认字形完整后，私有校验脚本补充按真实第一列裁剪提字，未改PDF、未降低末行断言；首轮独立校验六份16页通过，随后完整重跑取得最终总退出码0。首轮同样完整清理全部资源，未把阶段结果冒充整轮通过。

## 验收边界

本组实际验证了打印入口与预览/下载共享Blob，没有连接物理打印机或实施电子签章/外发。API辅助创建本地伪造夹具，不写海管家、航司或真实客户资料。代表多页样张不代替renderer的200行极值专项；HBL四法律版本/SI归档相邻由renderer与主会话另有专项，本browser只实际预览一次原SE自签HBL。完整经营费用、工作台、通知、报表和CSV仍属父最终验收。
