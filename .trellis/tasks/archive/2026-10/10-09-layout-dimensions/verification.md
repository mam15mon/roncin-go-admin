# 全站布局尺寸统一验收

## 实现结果
- 唯一尺寸来源 `web/config/layout.ts`，导出 `layoutDimensions`、`layoutOffsets`、`layoutLayers`、`layoutCssVariables`。共35个变量，尺寸/偏移为px、层级为无单位整数。
- `main.tsx` 首次渲染前安装变量到html，页面和body浮层共享；ProLayout展开侧栏、折叠占位、header/page-container token使用同一数值配置。
- 全局壳、公共模板、订单和客商导航避让、页面根级区块间距已按research明确消费端同步。默认视觉参数未改动，经验offset和不同语义的同值间距独立命名。
- 保留主题内部密度、弹窗尺寸既有唯一入口、media条件、组织图270/340等局部估算；没有猜测性高度分解或CSS默认值双读。
- 前端规范与能力导航已记录唯一入口、CSS桥接、派生关系、显式参数覆盖及浏览器验收要求。

## 实现后最小验证
- 原有受影响定向测试：15文件/86用例通过。
- 新增 `formScrollLayout.test.ts`：4个轻量用例，以替代180/20/130/24检验滚动坐标、实测页头优先、显式完整offset和独立表格阈值；不以CSS字符串检查代替浏览器布局。
- 修改文件Biome、LESS编译、tsc、`git diff --check`通过。

## 浏览器默认与替代尺寸验收
使用隔离Vite开发服务8002与Chromium，不依赖数据库写入。真实工作台壳通过浏览器route fixture提供只读用户数据；公共模板由实际组件直接挂载在同一布局主区，以独立React root包裹既有Provider，不复制模板实现。

每组通过8个壳场景：1600px展开/展开滚动/折叠滚动，767/768/769/390px滚动，恢复桌面。检查外层和内部等宽、对应侧栏占位、header高度、页签top与height、右边界、透明度、body变量继承、header/页签多点hit-test及横向溢出。

每组同时通过公共模板验收：真实OrderFormTemplate的PageHeaderShell sticky偏移、FormAnchorNav宽度/实测定位、正文预留、折叠归零、点击分节滚动落位、SectionCard外间距、StickyFooterBar留白；真实ParameterSettingTemplate业务Tabs紧接顶部栈。浏览器pageerror为0。

| 参数/实测结果 | 默认组 | 替代组 |
| --- | --- | --- |
| 展开侧栏/header偏移 | 208px | 240px |
| 折叠侧栏/header/页签偏移 | 48px | 64px |
| header高度/页签top | 48px | 56px |
| 页签高度 | 36px | 40px |
| 页头与参数Tabs sticky top | 84px | 96px |
| 正文纵向/横向留白 | 10px / 12px | 14px / 18px |
| 表单导航宽度 | 140px | 180px |
| 正文导航预留 | 164px | 212px |
| 页面区块间距 | 12px | 16px |
| footer纵向/横向留白 | 10px / 24px | 12px / 28px |
| 导航滚动落位 | 实测页头底边+12px | 实测页头底边+20px |

替代组仅临时修改唯一配置；浏览器实际组件参数与CSS解析同步，无任何组件/CSS消费端再修改。执行在try/finally保护下完成，随后恢复默认配置并确认内容与替换前副本完全一致。临时替代值未提交。

## 完整验收
- `pnpm run check:web` 的权限键、Proto、架构、重复扫描、lint和tsc均通过。
- 首次全量Vitest有1条海运DIRECT/HOUSE页签用例等待失败：184套件通过、1套件失败、1跳过，1092用例通过、1失败、12跳过。未改该业务或测试；单独复跑该套件5/5通过，随后仅复跑全量测试步骤185套件通过、1跳过，1093用例通过、12跳过（44.12秒）。最终门禁所有步骤通过，保留首次时序失败事实。
- `pnpm --dir web build`通过（36.53秒）；本次主入口安装变量，因此额外检查生产构建。没有调用Sentry上传、部署或推送。
- 既有7条Biome info、localhost:3000网络错误日志及超过500kB chunk提示未导致最终失败，未扩大修改测试环境或打包边界。
- 最终默认配置恢复确认与`git diff --check`通过。

## 复核产物（不提交临时文件）
- 浏览器脚本：`/tmp/roncin-layout-browser.cjs`；默认/替代日志与结果：`/tmp/roncin-layout-default.log`、`/tmp/roncin-layout-default-results.json`、`/tmp/roncin-layout-alternative.log`、`/tmp/roncin-layout-alternative-results.json`。
- 默认header完整11场景回归：`/tmp/roncin-layout-default-header.log`。
- 门禁：`/tmp/roncin-layout-check-web.log`；失败套件复跑：`/tmp/roncin-layout-recheck-visibility.log`；全量测试复跑：`/tmp/roncin-layout-check-web-tests-recheck.log`；构建：`/tmp/roncin-layout-build.log`。
