# 四类横切与完整CSV：前端独立复核

## 复核范围与结论

2026-11-16，全scope复核本组所有非生成Web产品和测试：公共订单模板/工具栏、
列表筛选与完整导出、四类注册定义与共享导航、工作台/通知白名单、财务费用详情、
航空危险品显式清空。读取任务prd/design/implement/check及完整hook保存文件，
补读注入截断的状态管理尾段；结合根AGENTS、server/web索引Quality Check、
当前公司/类型安全/目录边界、航空正文/进口和本组完整导出合同。

原实现代理已冻结全部非生成Web后接手，没有覆盖其他产品写入。确认并修复
下述三点，未留下已确认未修复的前端问题。未提交或推送，未修改服务端、生成物、
规范主文档和最终验收E2E；实际浏览器CSV和四类财务由主会话后续执行。

## 已修复

### 箱计划名称脱离完整导出快照

- 原`mapOrderListItems`对海运箱计划另调用`getMasterDataOptions`，名称可能来自
  导出之后发生变化的字典；此前205条DIRECT夹具没有箱计划，未覆盖该路径。
- Root协调server在`OrderContainerRequest`正规增加只读
  `containerSpecCode/containerSpecName`投影，Get/List/Export统一由真实箱型读取，
  Export处于同一RR快照，停用历史箱型仍有真实名称。
- 本复核修改`web/src/pages/orders/list-query.ts`：列表与导出共用同次响应的
  `code.trim() → name.trim()`，删除独立字典请求。有计划但引用ID缺失或两投影
  皆空时明确失败，不猜ID、不显示“箱型缺失”掩盖合同，不作历史回退；零计划正常空。
- 更新`list-query.test.ts`旧字典夹具为真实wire投影，并验证代码优先、无代码
  取名称、停用历史投影、无额外请求、缺失/空白投影及缺真实引用拒绝；新增
  `list-export.test.ts`的SE/SI箱计划CSV验证，完整结果只调用一次专用接口。

### 只收回子资源权限时旧导出仍可能下载

- 原会话key只有用户/公司/kind。保持订单read但撤回箱号或附件read时，
  旧mutation闭包仍能取得原采纳对象，并按旧授权列下载已在途响应。
- 本复核修改`web/src/pages/orders/list.tsx`：沿同一真实access判断取得
  `container.read/attachment.read`，两个授权位进入现有模板会话key，同时直接
  用于CSV列选择。授权变化卸载旧模板，原`isCurrent`同步失效，新会话重新采纳；
  没有新增权限真相或组织状态机。
- `list-export-page.test.tsx`补两个独立权限撤回场景：旧响应即使成功不下载，
  新会话可导出但相应列缺席；另有权限不变的正常授权列与仅read相邻成功用例。

### 成功封套省略时错误接纳旧资料

- Root最后复读指出，三个本组读取入口仍只拒绝`success === false`，从而把
  省略成功标志但夹带旧data的封套当成功。真实Proto `success`是非optional bool，
  省略语义为false；真实服务`ok/okList`成功总会输出true。
- 本复核只对`queryOrderList`、`fetchOrderListExport`和财务fees/detail要求
  `success === true`。没有修改全仓utils解包、领域布尔值或optional零语义。
- 三条入口分别新增省略/false夹带旧data拒绝回归，既有成功API夹具显式true；
  非法行/缺生成时刻等原断言保持在真实成功封套下执行，不被成功标志校验遮盖。
- 严格封套后的首轮复验因旧成功mock尚未全部补true而失败（4文件29项失败、
  6个错误），没有计为通过；按实际服务协议补齐这些夹具后，最终6文件81项
  全通过且无未处理错误。没有为旧mock增加产品兼容或吞失败。

## 完整链路复核

- 列表继续官方request/actionRef，WeakMap仅关联本次数组与filters/sort/total/
  generation，只有官方onLoad采纳。筛选/排序/分页、新request和卸载均撤销旧资格；
  空成功可采纳、失败不冒空成功。实际ProTable用例保留，不以手动触发onLoad替代。
- Export只用生成客户端一次请求无page筛选，校验完整total、生成时刻、每行ID/
  订单号/公司/type。快照与CSV文件是新请求时的数据，不冒称锁定此前页面行值。
  BOM/CRLF/中文引号多行/公式防护沿公共CSV，金额2位无千分位、数量精度与本地时间正确。
- 导出列权限沿后端auth/me派生access；未授权箱号/附件列不出现，optional未知
  不补零。普通列表仍保持分页协议，不用巨pageSize或循环列表拼全量。
- 新`features/orders/navigation/index.ts`确有四定义、工作台及通知三类消费者，
  公开最小映射与列表/详情/费用path，未知/未开放type无链接；名称沿statusMeta。
  system通知测试页canSelectOrder=false与示例首页规则未放开，无发送动作。
- 财务费用详情当前公司与canReadFinanceFees门控，API封套及orderId/company严格
  等值；query key为公司/订单，成功必须显式true；外层OrganizationWorkspace按用户/工作台隔离QueryClient。
  费用更正区只在成功校验目标详情后挂载，没有改变费用/核销业务规则或引入新权限。
- 航空危险品创建未填省略，更新UN/CLASS/截申报显式空串、PG显式UNSPECIFIED0，
  AE/AI共用adapter，null清空类型明确；后端optional语义不变，不猜货类自动清空。

## 本人实际执行

所有Vitest均使用`pnpm --dir web exec vitest run <相对web路径>`，没有运行全量
check:fast或build，不以已有前置门禁替本组。下列批次有重合，不累加为唯一样本数。

| 检查 | 实际结果 |
| --- | --- |
| `list-query.test.ts`、`list-export.test.ts`、`list-export-page.test.tsx`、`OrderListTemplate.export.test.tsx` | 4文件58项PASS，21.27秒，含真实官方采纳和箱型投影 |
| 最后页面授权正例/权限撤回6例、财务detail7例、DG clear、四类navigation、MyTodosDrawer、notification-test-action | 6文件74项PASS，22.95秒；成功封套收紧之前的增量版本 |
| 最终严格成功封套后的订单list/query/export/page、真实ProTable、财务detail相邻组合 | 6文件81项PASS，23.57秒，无未处理错误；日志`/tmp/check-cross-web-wire-fixed.log` |
| `pnpm --dir web tsc` | 最终exit0，全源及测试通过，日志`/tmp/check-cross-web-tsc-wire-final.log` |
| 全部本组非生成TS/TSX Biome（排除他人负责E2E） | 29文件PASS，无遗留fix；文件清单`/tmp/check-cross-web-biome-files.txt` |
| `pnpm run check:architecture` | 18规则测试PASS，577产品文件/2365本地依赖AST扫描PASS，无边界违规 |
| antd lint Toolbar、订单list、财务fees/detail | 三文件0 issues、partial=false、无跳过 |
| `git diff --check` | PASS |

修正后的最终定向批次没有act、antd运行时或deprecated警告；未静音console、跳过
类型错误、增加超时或修改依赖。读取Vitest当前结果缓存核对最终各文件：页面导出
6例18.032秒、既有list相邻16.401秒、真实ProTable4.330秒、财务detail2.515秒，
两个纯映射文件均不足0.04秒，均未超过20秒；没有为长尾放宽限制。

## 沿用实现证据及未执行边界

原实现报告`cross-module-web-validation.md`的既有列表/模板、工作台系统身份、
注册适配器和DG真实组件相邻批次已阅读，未无因重复所有交叉组合。

server的真实PG完整205行/RR快照及箱型并发改名、源/生成幂等由相应报告证明；
本人只消费正规生成类型并检查前端路径，没有重生成或冒称执行PG。
真实四类费用财务链、CSV浏览器下载、生产build/同域和最终check:fast仍由主会话
统合；本报告不把mock、静态扫描或准备脚本当成它们已通过。

全部非生成Web产品与测试现已冻结交回Root，无未修复确认问题。

## 最终门禁后的测试长尾拆分

主会话报告最终`check:fast`已退出0：527文件2855项通过、总246.2秒；产品与
生成物随后保持冻结。该全量运行缓存记录新页面导出文件6例耗时22688.56毫秒，
超过仓库20秒单文件约束，因此主会话仅重新分配测试调度拆分，不授权产品修改。

本次等价拆为：

- `list-export-page.test.tsx`：基础成功/授权列2例、缺身份或read资格1例，共3例。
- `list-export-permissions.test.tsx`：在途切公司1例、箱号/附件授权撤回各1例，共3例。
- 装配与mock移至`list-export-page.fixtures.tsx`，命名匹配现有
  `adaptRoutes.tsx`的生产页面扫描排除规则；产品不导入该文件，没有新扫描豁免。

所有6例原业务断言保留，未删除、增加超时、降低并发或静音。首轮共享fixture
直接导出`vi.hoisted`绑定被Vitest拒绝，在执行用例前失败；已按仓库既有fixture
方式改为普通声明与独立export，未为测试装配错误修改产品。

实际运行：

```bash
pnpm --dir web exec vitest run src/pages/orders/list-export-page.test.tsx src/pages/orders/list-export-permissions.test.tsx --reporter=json --outputFile=/tmp/check-cross-web-export-split-results.json
pnpm --dir web tsc
pnpm --dir web exec biome check src/pages/orders/list-export-page.test.tsx src/pages/orders/list-export-permissions.test.tsx src/pages/orders/list-export-page.fixtures.tsx
git diff --check
```

最终全部exit0，2文件6项通过。当前Vitest结果缓存：基础文件5403.22毫秒，
身份/授权变化文件13140.26毫秒，均小于20秒；JSON结果分别3项通过、0失败。
最终tsc和3文件Biome通过，无act/antd/deprecated或未处理错误输出。

本轮只改上述3个测试文件及本报告；产品、生成物、scripts、spec、E2E均未改，
因此未无因重跑已通过全栈门禁、PG或生产构建，不把局部拆分结果冒称新的全量门禁。
测试写权现已冻结交回主会话。
