# 技术设计

## 行为缺口和最薄边界

当前所有消息都发sampleMarkdown，字段逐段加粗、无清晰按钮；改为紧凑Markdown内容+可选单动作，动作存在发sampleActionCard，否则sampleMarkdown。沿用当前企业token/robotCode/userIds/batchSend，不接入历史StandardCard、现行模板实例或DWS个人态。用户已给出ActionCard方向并授权完整实施review合并push，无其他未决产品选择。

## 领域和发送

将DingTalkMarkdownMessage/SendMarkdown/PrepareMarkdown统一改为中性DingTalkMessage/SendMessage/PrepareMessage（最终命名实施可按惯例微调），内容为Title、Text、ActionTitle、ActionPath及准备后的ActionURL。biz只生成固定标题、转义后的Markdown和受控根相对路径；data复用配置回调的scheme/host解析本站URL。准备后路径清除、Send再次准备保持幂等，不追加正文链接。只有合法成对动作才能进入sampleActionCard，正文与动作参数一次json.Marshal为msgParam字符串。保留processQueryKey受理判断，令牌缓存和错误语义不动。

动作严格限定既有路由：/welcome、/orders、/finance/exchange-rates、/orders/sea-export/{UUID}/fees、/commission-adjustments/{UUID}/my-supplement-source。不把RequestID当OrderID、不猜订单品类、不新增数据库lookup；订单分配/补录采用准确标注的列表入口。费用入口继续匹配ResourceID构造的完整Parameter后缀，缺合法入口就无动作，不纠错。动态名称/摘要不能形成结构或链接。

结果状态只根据现有生产端生成的固定结果后缀判定，不能contains匹配任意姓名。合法费用更正已处理可用信息态；不扩展Schema或修改入队业务。验证码仍直接即时发送，不走通知队列，保持PENDING/READY、冷却、绑定和失败规则。

## Admin契约与前端

Preview响应保留notificationType/title/markdown，移除仅费用专用orderUrl，增加actionTitle/actionUrl字段；源Proto修改后生成Go/OpenAPI/Web客户端，源和必要生成物同提交，不保留旧字段双读。Service仅DTO映射。

测试业务模板清除真实资源路径，使用固定/welcome及「打开系统首页」演示ActionCard，正文注明示例不指向真实订单；验证码清除动作。仍保留现有orderId显式拒绝/权限逻辑，不借本期视觉任务删除或开放跨公司能力。Catalog仍由后端有效权限及绑定真相决定。

前端沿用React Query、同一query key隔离迟到预览和无自动重试mutation，PageContainer/PageHeaderShell/SectionCard保持流式满屏。内层卡片以窄消息宽度左对齐模拟阅读密度，不约束页面容器；安全react-markdown禁HTML/图片、正文任意链接，唯一动作来自服务端允许路径和http(s)URL。按钮在预览区明确为示例首页；展示以钉钉为准，不伪装像素一致。源码保留正文及按钮目标核对，不引入渲染平台。

## 文件和职责

后端实施拥有biz通知/密码renderer/测试用例、data通知适配、admin.proto/service及全部生成物和相关stub测试。前端实施拥有通知测试页/私有样式和测试，不修改generated services/types。主会话拥有任务/专项规范、完整门禁、浏览器、归档、合并与push；独立check最后只review稳定版本。

## 风险与交付

涉及消息副作用与验证码相邻接口，需定向Go/HTTP/前端后执行check:fast；无依赖/构建配置变化默认不固定构建，若实施引入变化再按风险构建。开发后端只能health/匿名鉴权冒烟，不自动发真实通知。发布整组代码，沿用站点配置，无Schema或新钉钉模板依赖。回滚整组提交，不建自动旧格式分支。
