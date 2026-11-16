# 航空单证内部布局浏览器对照验收

日期：2026-11-16（北京时间）。仅使用真实前端组件与浏览器合成API，不写开发库或外站。

## 执行

```bash
RONCIN_ACCEPTANCE_ARTIFACT_DIR=/tmp/air-document-alignment-final pnpm --dir web exec playwright test tests/e2e/air-document-alignment.e2e.ts
pnpm --dir web tsc
pnpm --dir web exec biome check tests/e2e/air-document-alignment.e2e.ts
git diff --check
```

Playwright 3项全部通过，耗时29.9秒。使用Vite `http://127.0.0.1:8001`、Chromium、Asia/Shanghai时区。测试复用原海运稳定性与航空白名单夹具，不修改两份既有夹具；必要的航空编辑能力与客户抬头只在新增测试中精确覆写对应GET请求。所有未声明请求由白名单夹具显式404，所有写请求均被拦截且验收断言为空。

## 实际布局结果

海运使用现有 `SeaBillContentFormFields`（海运出口新建页）；航空使用新增订单常用正文组件（AE详情可编辑HAWB及只读MAWB）。两者在同一个浏览器、相同viewport逐一测量真实文本域边界。

| viewport | 海运抬头宽×高 | 航空HAWB抬头宽×高 | 航空MAWB抬头宽×高 | 海运/航空货描高度 |
| --- | --- | --- | --- | --- |
| 1440×1000 | 543×95px | 543×95px | 543×95px | 均116px |
| 390×1000 | 268×95px | 268×95px | 268×95px | 均116px |

- 1440：发货人/收货人、通知人/代理、唛头/品名均为左右两列，同行Y坐标相等、左右宽度相等；海空左右列X坐标同为245/804px。
- 390：上述三组均退为单列，相同X坐标且下方控件在上方控件之后，文本域无横向挤压。
- 航空常用抬头为4行（95px），货描为5行（116px），不再被全局240px高度影响。
- HAWB真实文本域可编辑；切到MAWB后同样文本域为原生readonly，实测宽高相同，白底文本框外观一致。

## 交互结果

1. 页面未做任何人工编辑时，点击“从委托客户带入”，合成英文抬头进入发货人框。
2. 紧接着点击“返回列表”，真实未保存离开确认弹窗出现，URL仍停留原订单；取消“继续编辑”后保留带入内容。这条断言排除了先手工输入标脏的干扰，直接验证程序带入经父模板applyValues标脏。
3. 通知人/第二通知人切换，修改第二通知人后切回再切入，两份独立正文均保留。
4. “更多运单资料”可展开，并能看到原有“路线与航班印字”内部分组。

## 截图及度量产物

根目录：`/tmp/air-document-alignment-final/`。

- `air-document-alignment.e2e.ts-海空单证内部边界对照-1440px/sea-document.png`
- `air-document-alignment.e2e.ts-海空单证内部边界对照-1440px/air-house-document.png`
- `air-document-alignment.e2e.ts-海空单证内部边界对照-1440px/air-master-document.png`
- 390截图位于同根目录下的 `air-document-alignment.e2e.ts-海空单证内部边界对照-390px/`。
- 两个尺寸目录均保存 `geometry.json`，包含三组字段及只读主单的完整边界坐标。
- 带入离开保护截图：`air-document-alignment.e2e.ts-航空通知人切换、低频展开及显式带入触发离开保护/import-leave-guard.png`。

## 实际发现与修复后复验

首轮真实浏览器发现航空所有文本域均为240px，而海运为95/116px，单元测试的rows=4/5无法发现全局CSS影响。已反馈实现代理；实现代理对航空常用文本域设置局部`height:auto`保留原生rows，无需修改全站样式。以上最终3项是在修正后完整重跑通过的结果。

## 边界

本轮浏览器覆盖AE共享航空组件；AI同组件的身份/导入竞态由定向组件测试与主会话门禁另行验证。本轮不生成PDF、不保存订单或单证，不把合成浏览器交互当作真实后端权限、事务或存储正确性证明。更多低频字段的全量编辑协议沿用已有组件测试，不在本轮增加重复浏览器大表单用例。
