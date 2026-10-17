# 体检证据

- 基线 main 851a3403，干净；用户收到体检结论后明确要求创建任务、修全部 bug、回归 main、删分支。
- 基线 check:fast 通过：235 前端文件/1303 用例通过，1 文件/12 用例跳过；本机日志 /tmp/roncin-system-audit-check.log。后端常规测试/静态/漏洞通过，不能替代真实库。
- 实际 request 模块模拟 409，成功分支仍执行且不进入 catch。真实 ResetPasswordModal 也关闭、刷新并提示密码已重置；API 全部浏览器拦截，无开发库写入。
- 实际 MyCommissionDrawer + 共享客户端：A 缓存、卸载重挂、延迟 B 时仍显示 A，B 落定替换。证明缓存身份缺口。
- 生产仅 main.tsx 直接导入 utils/queryClient，业务使用 useQueryClient，可在工作区隔离客户端。
- 六台账为费用/账单/发票/流水/核销/对冲；默认 CSV 原始字段缺金额币种状态时间语义，提成专用导出已有转义/公式保护思想可复用。
- 本期不实现附件存储、PDF 发送确认、其他运输业务或备份平台。
