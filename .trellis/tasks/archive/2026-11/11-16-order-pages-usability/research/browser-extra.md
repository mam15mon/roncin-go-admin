# 订单页面补充浏览器验收

日期：2026-11-16（Asia/Shanghai）。本报告只记录本组补充验收，不替代主会话门禁。

## 环境与隔离

- 使用现有Vite开发服务`http://127.0.0.1:8001`及Playwright Chromium，`workers=1`。
- 产品页面与组件是真实代码；业务API均由浏览器路由合成响应，不写开发库、不发外站请求。
- 新增`web/tests/e2e/order-pages-usability-extra.e2e.ts`及`order-pages-usability-extra.fixture.ts`。
- 扩展夹具在基础海运/航空夹具前增加逐路径GET白名单，未知读取返回404并记录；写入全部返回405并记录。唯一显式允许的POST是既有只读`/api/v1/air-weights/preview`，不是业务写入。
- 海运进口合成订单、当前用户权限与审核状态均使用SI身份，不借SE权限模拟进口。

## 已通过范围

执行：

```bash
RONCIN_ACCEPTANCE_ARTIFACT_DIR=/tmp/order-pages-browser/extra-final pnpm --dir web exec playwright test tests/e2e/order-pages-usability-extra.e2e.ts --workers=1
```

8/8通过，约1.4分钟：

- SI新建及详情，390/1440各两项：业务、承运、货物区可见，无横向页面溢出；详情服务区位于业务与客户前，新建无已保存服务区。
- AE/AI的DIRECT详情，390/1440各两项：只有MAWB页签，展示真实合成主单正文，无虚构HAWB；AI同时核验进口服务位置。
- 同项继续进入新建：HOUSE页签可切换，分单号跨主分单页签保留；切DIRECT移除HAWB，切回HOUSE仍保留原输入。
- 所有用例未知请求、业务写入、页面异常、未分类控制台错误均为空。
- SI1440完整截图已目视检查状态→进口服务→五卡顺序及各区边界；截图位于上述产物目录。

## 单证全宽修正

浏览器复核发现：SectionCard内容被`Row`包装，`Tabs`缺少明确宽度时会按内容收缩；竖向`Space`默认排列又使内部字段缩宽。按本期全宽要求，最薄修正`AirDocumentSection`：Tabs与DIRECT提示占100%宽，三个竖向Space使用stretch。无公共样式修改。

聚焦验收进一步检查1440下的真实边界框：页签内容宽度须大于单证卡宽度90%，常用长文本字段宽度须大于页签内容宽度40%。实际测量另见最终补充结果。

## 首轮失败与限制

- 首轮扩展2项通过、6项失败，失败原因均明确留存：误把POST重量预览归为写入，以及SI详情既有未连接`useForm`警告。修正只读RPC白名单，并将这一条准确匹配的既有警告单列`baselineWarnings`附件，未忽略其他控制台错误。
- 既有SI控制台告警全文为`Warning: Instance created by useForm is not connected to any Form element. Forget to pass form prop?`（原报文中的标识带反引号）；基础海运稳定性夹具也已单列该类警告。本组不宣称浏览器零告警。
- 本组未提交创建、保存、运输更新、重量确认或服务办理，不把合成页面验收当作真实后端事务、权限或并发验收。
- 本组未运行全量门禁、未提交Git；最终门禁与提交由主会话统合。

## 最终宽度聚焦结果

执行：

```bash
RONCIN_ACCEPTANCE_ARTIFACT_DIR=/tmp/order-pages-browser/extra-width-final pnpm --dir web exec playwright test tests/e2e/order-pages-usability-extra.e2e.ts --workers=1 --grep '航空.*1440' --reporter=json
```

2/2通过（28秒），页签内容大于单证卡宽度90%的断言通过。真实测量：

- 航空AE直单与新建页签切换 1440px：页签内容 1102px，发货人文本域 543px；长文本占页签宽度 49.27%。
- 航空AI直单与新建页签切换 1440px：页签内容 1102px，发货人文本域 543px；长文本占页签宽度 49.27%。

页签切换/直单验证与宽度断言均通过，表单字段铺满卡片可用宽度。JSON结果在`/tmp/order-pages-browser/extra-width-final-report.json`，截图在同名产物目录。随后只为附件补充cardWidth数值元数据（不改断言和产品），未重复运行。Biome三文件及git diff --check通过。
