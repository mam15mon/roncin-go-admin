# 浏览器验收记录

日期：2026-11-16（Asia/Shanghai）

## 环境与范围

- 使用 `web/playwright.config.ts` 的 Chromium、Vite `http://127.0.0.1:8001`，所有业务请求由测试路由合成响应，未写开发库。
- 产物目录：`/tmp/order-pages-browser/sea-layout`、`/tmp/order-pages-browser/air-layout`、`/tmp/order-pages-browser/air-modal`、`/tmp/order-pages-browser/air-final`。
- 原有海运稳定性夹具未修改；空运新增夹具为 `web/tests/e2e/order-pages-usability-air.fixture.ts`，仅声明AE/AI所需读请求并拦截写入。

## 通过项

- `RONCIN_ACCEPTANCE_ARTIFACT_DIR=/tmp/order-pages-browser/sea-layout pnpm --dir web exec playwright test tests/e2e/sea-order-layout.e2e.ts`：8/8通过，390、1024、1440、1920下新建和详情均无横向溢出，海运字段、操作入口可见，合成夹具写请求为空。
- 空运页面初次全量运行：8/8通过，覆盖AE/AI新建与详情、390/1440两档，业务与客户、订舱/承运与运输、货物与商业、单证信息可见；详情验证 MAWB/HAWB、实际货物与计费重量；AI详情验证进口服务；无横向溢出，写请求为空。
- AE详情390单项复验增加“修改共同运输资料”点击，运输编辑弹窗中的“本次仅保存共同运输资料”可见，Escape可关闭，写请求为空。
- 空运页面截图和请求记录均在上述临时产物目录；本次只读展示，不提交保存/确认操作。

## 限制与未验收项

- 中途有一次进程退出143；主会话随后完整重跑8/8通过（44.9秒），再启用未知GET拒绝与断言后8/8通过（55.6秒），原环境限制已解除。严格版产物在 `/tmp/order-pages-browser/air-strict-final`。进一步明确禁止写入、捕获页面异常后，最终8/8通过（52.4秒），产物在 `/tmp/order-pages-browser/air-final-verified`，写请求/未声明请求/页面异常均为空。
- 本次未点击“预览影响/确认提交”、重量确认、货物新增/编辑、HAWB保存、进口服务保存；因此不把合成页面可见性当成写入、版本、权限或事务验收。
- 新建页面只验证结构可见与无溢出，没有提交订单。生产后端接口、真实公司权限、真实PostgreSQL数据未在本次浏览器夹具中验证。
- 390宽度通过的是页面容器无横向滚动，不代表每个低频单证字段都在首屏可见；具体折叠区需操作员试用。
