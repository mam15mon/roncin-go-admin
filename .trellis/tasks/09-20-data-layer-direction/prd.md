# 数据层方向决策与统一迁移

> 状态：**方向已定（2026-09-20）：React Query v5**。complex task：design.md +
> implement.md 完成并确认后再启动执行。

## Goal

消除全项目「手写 useState + useEffect + 竞态令牌」服务端状态层的系统性偏离：
同一套 loading/竞态/重拉防护逻辑在 20+ 文件重复实现约 15 遍，已产生过真实缺陷
（`useMasterDataCrud` 不稳定依赖导致 effect 无限重取，2026-09-19 修复）。

## 方向决策记录（2026-09-20）

**选型：`@tanstack/react-query` v5，源码直接 import，不经 umi 插件注入。**

决策依据（当日实测）：

1. **umi 插件 useRequest 出局**：
   - 底层是 `@ahooksjs/use-request@2.8.15`（2021 年 v2 架构，官方最新
     `@umijs/plugins@4.7.19` 仍绑 `^2.0.0`，无现代化计划）；防抖参数为
     `debounceInterval`、无卸载后 setState 保护（与测试 stderr 零噪音口径冲突）。
   - `import { useRequest } from '@umijs/max'` 在 vitest 下为 undefined
     （探针实测 TypeError）——umi 构建期注入的导出测试环境拿不到，51 个测试
     文件 mock `@umijs/max` 正源于此；走该方案需给每个迁移测试手工补 mock。
   - 该插件唯一增值（service 为 url/对象时路由到 umi request 实例）对本项目
     价值为零：全部请求走 OpenAPI 生成函数，已统一走 request 客户端。
2. **React Query 胜出**：
   - AGENTS.md 本就要求「服务端状态优先 React Query」。
   - `@tanstack/react-query@4.44` 已作为 `@umijs/plugins` 传递依赖存在于
     依赖树，正式启用边际成本小；直装 v5 获得官方 React 19 支持。
   - 源码直接 `import { useQuery } from '@tanstack/react-query'`，生产与
     vitest 同源可用，测试零 mock 改造成本。
   - 缓存/失效/竞态/卸载保护/重试/devtools 全套内置，是手写令牌模板的
     完整替代。

## Requirements

1. 基建：直装 `@tanstack/react-query@^5`；QueryClient 单例 + Provider 挂载；
   全局 retry=false、refetchOnWindowFocus=false；QueryCache/MutationCache
   统一 onError 提示（经 `appFeedback` 桥接，支持 `meta.errorMessage` 定制文案）。
2. 迁移行为等价：不改接口契约、不改用户可见交互；错误提示文案逐处保持。
3. 分阶段迁移（阶梯见 implement.md），每阶段独立提交可回滚；先高痛点抽屉/
   弹窗与订单详情聚合，再逐步铺开。
4. spec 沉淀：唯一服务端状态模式写入 `.trellis/spec/web/frontend/`，
   禁止新增手写 useEffect+useState 竞态模板。
5. 测试保持零噪音口径（act/deprecated = 0），用例数不减少。

## Acceptance Criteria

- [x] 方向决策记录在本 PRD（含理由）。
- [ ] design.md / implement.md 完成并通过确认。
- [ ] 阶段 0-2 迁移完成：指定文件全部改用 React Query，旧竞态令牌删除。
- [ ] 定向测试与全量门禁通过，测试 stderr 零噪音。
- [ ] spec 沉淀完成，新增请求不再出现第二套手写形态。
- [ ] 剩余批次列出后续任务清单。

## Notes

- 审计来源：2026-09-19 两个只读审计代理报告；2026-09-20 选型实测记录见上。
- 用户工作区可能存在 master-data-template 在改内容，迁移文件集须与之无交集。
