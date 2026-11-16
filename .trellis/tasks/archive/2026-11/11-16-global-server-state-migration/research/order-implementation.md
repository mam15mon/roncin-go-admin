# 订单与单证组实施记录

## 范围和结果

仅在任务 worktree 修改本组入口，未提交 Git，未改源 main 工作区。原始盘点中的 8 个订单查询入口和 4 个补充命令入口均已处理。

| 入口 | 结果 |
| --- | --- |
| SameBatchOrdersSection | 按订单/品类的 useQuery，去重后派生展示，计数只在成功查询上报；乱序与失败不冒充空列表。 |
| UnlockRequestHistoryDrawer | useQuery 管理历史和加载/失败；保留原 5 秒审批进度周期，前台且存在活动状态才轮询，隐藏停止、恢复立即检查，关闭停止。 |
| SeaOrderChangeHistoryDrawer/Section | 预览、分页与事件详情分别 Query；同订单分页保留内容，换订单不借旧列表或详情。 |
| SeaExportDetailFeatures | 动作资格按订单、表单身份和已采纳版本查询，失败关闭操作资格；保留写后续体卸载保护。 |
| SeaDocumentSection | 无 documents prop 时查询单证，首次成功采纳编辑基线；背景成功不覆盖正文或版本，正式动作后显式接受；放货关联独立 Query。模式预览使用 mutation，与提交意图身份绑定。 |
| SeaExternalConfirmationFields | 附件候选 Query，失败有就地展示，skipErrorHandler + silent 避免重复通知。 |
| SeaDocumentHistoryActions | 版本与事件各自 useInfiniteQuery，按服务端原始返回条数决定下一页，再过滤本单证事件；重开历史重置第一页。修订/作废预览 mutation 绑定编辑修订并保留预览载荷。 |
| release-pod-panel | 已选订单和品类驱动关联提单 Query，删除打开抽屉后的手工 then/request identity 链；列表仍走官方 request。 |
| SeaTransportExecutionUpdateModal | 共享航程预览 mutation，预览绑定编辑修订，输入变化立即使旧预览无效；地点候选 key 补组织/业务类型。 |
| SeaBillContentFormFields | 显式客户/代理带入使用当前客户端 fetchQuery，保留用户主动动作，补资源/目标文本/修订保护，旧请求不覆盖新输入。 |
| air-master | onFinish 阶段预览使用 mutation，原有编辑会话、版本、输入修订和写入保护继续保留。 |
| AirTransportMaintenance | 同上，独立运输命令仍保留完整正文与打开基线。 |

## 测试基础设施

renderWithClient 使用 RTL wrapper 包住 QueryClientProvider，rerender 保留同一客户端；可选 RenderOptions 的 wrapper 组合保留。新增 queryClientTestUtils.test.tsx 验证同一客户端、外层 Context 和独立新实例，已由主会话单独提交。

本组原先直接 RTL render 的单证/动作资格测试改成真实独立 QueryClient。只调整异步查询必要等待、请求新增 signal/options 断言，保留真实控件、事件和原业务断言。

新增针对性回归覆盖同批订单换实体和迟到、undefined/success:false/非法列表、合法空 repeated，单证背景重查不覆盖正文/提交版本，审批前台/隐藏/恢复/关闭的真实 Query 轮询。

## 验证

- 本组 29 个修改文件 `pnpm --dir web exec biome check --write` 通过。
- `pnpm --dir web tsc` 通过（记录 /tmp/order-documents-final-tsc.log）。
- `git diff --check` 通过。
- 前置定向批次：16 文件 55 测试、8 文件 33 测试、新增 3 文件 13 测试、审批 2 文件 4 测试、相邻 3 文件 8 测试全部通过。
- 最终去重定向批次：29 个文件、104 个测试全部通过，用时 19.73 秒；路径清单位于 /tmp/order-documents-test-paths.json，输出 /tmp/order-documents-final-tests.log。该次默认 reporter 仅记录 SWR 的 Node TimeoutNaNWarning，未出现网络错误或失败。

## 已定位的环境噪音

不声称零噪音。未静音 console、process warning，也未加入全局计时器、CSS 或测量补丁。

1. air-master.edit.test.tsx 原有真实 focus 事件触发 SWR 2.5.1 的 Node TimeoutNaNWarning。依赖 `swr/dist/config-context-s960zqob.js:514` 将 `setTimeout.bind(undefined, revalidateAllKeys...)` 直接注册为 focus/visibilitychange listener，事件被当成 delay；Node 警告，浏览器按 0 处理。临时尝试事后局部 fake timer 无效（库初始化时已绑定原函数），已完全撤回，保留原真实回焦事件与全部断言。
2. 海运原有 noStyle/自动高度 TextArea 在 happy-dom 的 getComputedStyle 中 borderTopWidth/borderBottomWidth 返回 initial；`@rc-component/input/es/calculateNodeHeight.js` parseFloat(initial) 导致 React NaN height 告警。诊断确认普通文本域计算为 1px，未为此修改产品边框或测试度量，临时日志/CSSVar/主题动画试验均已撤回。
3. 退关单证测试曾未 mock 放货记录而触发 localhost:3000 网络连接；已补真实合法空响应 mock，消除该项。单证元数据隐藏字段改用 Antd Input，避免异步首次回填从 uncontrolled 到 controlled 的原生 input 警告；保存测试的真实绑定模拟同样使用 Antd Input。

## 交付路径

本组精确修改文件列表见 /tmp/order-documents-owned.json（29 项，包括实现及相关测试），另加本研究记录。主会话按列表审阅提交；其他代理和主会话正在修改的费用、公共 options、财务文件均不属于本组提交范围。
