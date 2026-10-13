# 钉钉默认扫码技术设计

## 技术依据与选型

父任务 research/current-state.md 记录已读取的钉钉托管 0.21.0 脚本。它以 OAuth2 iframe=true 展示嵌入授权页，通过 message 回传 redirectUrl/authCode/state。脚本没有公开清理监听入口，因此本任务采用同一已核验协议的最小 React iframe 组件，自行管理可清理的 message 监听，不反复调用不可销毁的全局初始化函数。

复用服务端 authorize_url 及其中 client_id、scope、corpId、redirect_uri、state；URLSearchParams 增加 iframe=true，避免手工拼接或二次编码。具体是否需要额外参数才能保持二维码视图，以官方端点与真实应用测试核实，不依赖未经验证的第三方文章。方案不添加 npm 包，不存储二维码图片或自行生成含授权 URL 的通用码。

## 服务端配置

GetDingTalkLoginConfig 保留 enabled/authorize_url，并在 Proto 补充 expires_at，来自 StartDingTalkLogin 同一个 state 的到期时间。OAuth state、Cookie、扫码 iframe 必须来自同一次配置响应。公开参数可供页面展示，但 clientSecret、token、原始身份响应仅服务端保留。

本任务不修改换码与企业校验，不改人员同步/待分配状态机。配置响应及 Cookie 应禁止缓存；定向校验真实回调同源。应用参数缺失明确提示，不隐藏配置问题。

## 页面与状态

登录页初始 mode 为钉钉；主体为二维码区域、扫码说明、刷新操作，次级“管理员账号登录”切换密码。保留既有品牌与动画资源，不重做无关布局；密码表单与扫码组件互斥挂载，小屏能访问两个入口。

服务端配置使用 React Query 管理，禁用窗口焦点/重连自动刷新，重试显式触发。一次有效扫码实例只用一份配置；刷新先停用旧实例、移除其监听，再串行获取新配置，防止旧请求覆盖 state Cookie。快速连点/切换期间禁止并行初始化；StrictMode 的挂载/卸载也必须验证不会产生两份互相冲突的 state。

useEffect 只承担 iframe/事件/到期定时器的生命周期，不手写 useEffect 请求镜像状态。配置失败有可见错误和重试，不回退成密码表单；管理员入口始终可用。到期显示“二维码已过期，请刷新”，不自动轮询刷新。恢复前台时复核 expires_at，避免后台计时器延迟造成误显示。

## 消息与回调

- 只接受 origin 精确等于 https://login.dingtalk.com 且 source 等于当前 iframe.contentWindow 的消息；不用包含域名的正则判断。
- 校验消息结构、成功标记与 redirectUrl；解析后必须为配置的回调 origin/path，state 与本实例一致，authCode 非空。
- 每实例只处理一次成功回调，当前实例已过期/停用或组件已卸载时忽略迟到消息。
- 保存已校验的站内回跳地址，导航到本系统既有 dingtalk-callback 路由；不盲目打开消息提供的任意外链。后端再次校验 Cookie state 与企业身份。
- 用户换密码或刷新扫码后移除旧监听、定时器和 iframe。过期或失败显式提示，不输出授权码、state、Cookie 或 token 到日志。

保留 dingtalk-callback 的一次换码、状态提示、单工作台直进和多工作台选择；其候选消费使用取消主要组织后的 currentOrganization 规则。

## 验收与限制

测试覆盖配置/SDK协议失败、非匹配消息、错误 state、旧 iframe消息、重复回调、过期刷新和模式切换。真实扫码需使用当前企业应用与实际回调域名，验证开发代理和生产同域地址、iframe加载及成功/失败返回。官方文档动态正文不可抓取和未完成的实测必须记录，不能通过更换为跳转按钮降低 MVP。

回退本子任务前后端配置字段一起处理；不改变数据库。生产入口/CSP若确需调整，只允许所需钉钉来源并补构建验收，不泛化放开全部 iframe 域名。
