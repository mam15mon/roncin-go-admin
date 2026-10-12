# 真实浏览器验收

环境：隔离 PostgreSQL + 本地 API 18010 + Vite 18001，未操作日常开发库，钉钉发送禁用。

## 审核失败与重试

脚本 `/tmp/roncin-fixed-approval-browser.cjs`：选择隔离环境待审订单，填写1000个中文字符，拦截审批请求返回409。验证错误可见、弹窗不关闭、输入原样保留、不出现成功通知、服务端仍PENDING；移除拦截后重试，幂等键保持一致，HTTP200，实际状态APPROVED。

结果：PASS。截图 `/tmp/roncin-fixed-approval-failure.png`、`/tmp/roncin-fixed-approval-success.png`。

## 发放失败与重试

脚本 `/tmp/roncin-fixed-payment-browser.cjs`：选择隔离环境可发放提成，拦截 `/paid` 返回409。验证弹窗和已勾选确认保留，服务端仍CONFIRMED；解除拦截重试，幂等键一致，服务端实际PAID，金额280.00000000。

结果：PASS。截图 `/tmp/roncin-fixed-payment-failure.png`。测试只登记隔离账务事实，不涉及任何外部资金转移。

注：发放脚本首轮误写 `/pay` 匹配，未拦截实际 `/paid` 而超时；修正脚本后用另一隔离样本重新完整通过，未据首轮失败判定产品缺陷。

## 待审补录撤回重提与真实父页面刷新

脚本 `/tmp/roncin-fixed-supplement-browser.cjs`：创建隔离环境补录申请，验证待审时不出现另建按钮且本人可撤回重提。首次发现撤回成功后的订单查询刷新会卸载子组件并丢失预填弹窗；修复费用页仅首次加载显示占位后重新执行完整浏览器验收。

最终结果：PASS。撤回 HTTP200 后预填弹窗出现、原原因完整；服务端 hasPendingSupplement=false；关闭草稿后新建补录按钮恢复。截图 `/tmp/roncin-fixed-supplement-resubmit.png`。
