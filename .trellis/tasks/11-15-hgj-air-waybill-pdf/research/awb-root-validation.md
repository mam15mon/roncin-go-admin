# 航空制作与PDF：主会话验收

## 当前状态

本组实现、定向验证、两独立全范围复核、生产构建、最终全栈门禁及真实浏览器整轮已通过。产品代码冻结后完成真实验收，没有以测试文件存在或跳过项代替结果。

## 已核对

- 仅新增主单真实公司英文名的只读API投影，无新数据库列、迁移或权限。后端实现代理的真实PG与生成幂等通过，独立后端全范围复核已完成且无确认缺陷；具体命令和范围见`awb-editor-validation.md`、`awb-server-review.md`。
- 制作页按MAWB/HAWB真实身份区分；订单MAWB入口先核验本票归属，再固定真实主单快照，不借后续订单版本。主单修改仍走Preview及共享影响确认，分单只提交本票正文和原基线。
- Root独立读取实际普通PDF的MAWB第1页、HAWB第2页PNG：中文与主体区清晰，0.00、6.7183、纯日期、正文与表格、保存版本和续页页码可见，无观察到裁切。renderer后续正式报告已完成长文和200行极值64页核验，详见`awb-pdf-validation.md`；Root另独立查看双200行最后三页、长party第14页、HBL四版本联系图，末行、续页与主体正常。
- `python3 .trellis/scripts/task.py validate 11-15-hgj-air-waybill-pdf`：18条implement/check上下文通过；两份既有大规范存在注入截断提示，已要求检查代理按相关段落主动补读。
- 两份字形JSON从HBL移入中立目录，Root逐字节比对HEAD原文件均完全相等，没有重制或删改覆盖范围。
- 长货物末页曾有整页图片页眉显示疑点；Root按同一PDF独立重渲染并核对：顶部1198暗像素、标题/副题字框与单独裁剪实图一致，无确认产品缺陷，未凭显示差异改源码。
- `git diff --check`：当前通过。

## 最终验证

- 两独立检查已覆盖所有本组server/web变化。Web补齐主单合法模式及link/embedded模式一致性校验，修正合法DIRECT夹具；五个真实PDF测试局部包React act，混合测试零act噪音；字体README同步中立再生路径。详见`awb-web-review.md`。
- 实际主分PDF普通/边界/200行、HBL四版及SI归档相邻；提字、字符/末行与字框边界、逐页实图。
- 稳定代码的`pnpm run build`已实际exit0，Web构建27.21秒、Go生产入口成功；只有既有chunk体积提示和本机未配置Sentry上传提示。生产路由和共用字体变更是本次build依据。`pnpm run check:fast`最终exit0，总245.3秒（WEB245.3/SERVER147.8）；前端521文件2799项通过，既有1suite/11tests跳过不计通过。日志`/tmp/hgj-awb-check-fast.log`。存在既有TimeoutNaN/getComputedStyle/localhost3000拒连等测试环境stderr，前组通过日志亦有同类输出，未通过关闭规则掩盖。
- Root独立复跑`PYTHONPATH=/tmp/roncin-hbl-pdf-tools python3 /tmp/roncin-awb-pdf-validation/verify.py`实际exit0，64页全文、行尾与字框边界复验通过；未重新生成/改变冻结产品。
- 一次性数据库真实浏览器最终命令`RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-awb-browser-acceptance/run.mjs`实际exit0，Playwright整例1 passed / 1.7分钟；AE/AI双模式适用6份PDF、真实保存/409/在途新输入/15秒与回焦、同Blob三种输出、生产同域深链/脚本/字体MIME/404及HBL相邻全部通过，pageerror为空。
- 最终证据`/tmp/roncin-awb-browser-Oczm1d`，6份16页实际下载PDF完成独立提字及实图核验。Root另查看最终AE HAWB末页（第8其他费/保存版本页脚）与AI HAWB归档末页（AS ARRANGED、未知合计与0.00、汇率6.7183），并独立裁剪复核HAWB页眉实字完整。
- 首轮浏览器整例已通过，但私有PDF后处理把第一列表格续行和其他列串接，误报末标记，导致runner总exit1。改按真实第一列坐标提字后原6份16页独立复验通过，再完整复跑最终runner exit0；没有删掉末行断言或改冻结产品。
- 最终清理与独立核验：本轮随机数据库0、角色0，自己的8010/9010/8002三端口空，未知8001未操作。完整过程见`awb-browser-validation.md`，未触开发业务数据。

## 边界

物理打印、航司或海管家写入、电子签章、真实员工通知不在本组。临时证据和测试数据留在仓库外，不把凭据、真实登录状态或原始业务数据写入Git。
