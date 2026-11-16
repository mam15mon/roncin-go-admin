# 跨层边界问题复盘

本组最终字段审计发现两类已有缺口，均属于批准字段合同内。实现负责人先修功能，
再补实际触发与相邻路径验证；详细命令和最终结果由分层验证报告记录。

## 中文长度被持久化拒绝

1. 根因分类：B跨层合同、D覆盖缺口。biz用`utf8.RuneCountInString`允许100/200/1000
   个字符，Ent仍以`MaxLen`计算UTF-8字节；ASCII样例与纯adapter测试无法区分两者。
2. 此前并未做同类完整落库修复；不能把界面maxLength或领域单测通过当数据库边界一致。
3. 防止重现：本期同一Order聚合自由文本和航空主单notes按已有字符合同使用
   `MaxRuneLen`，机器标识/币种/ISO/UN不机械替换；运行正规Ent生成并核对SQL列形，
   中文边界实际落库与+1拒绝必须在正式迁移后的PG执行。
4. 有限扩展：只核对当前四类表单内同一合同的相邻文本，不改其他领域全仓字段。
   当前varchar字符尺寸不变时不生成空迁移，避免把验证口径修复误当Schema重建。
5. 知识已写入`air-order-commercial-fields.md`及跨层思考指南，随本组代码一起提交。

## 航空取消危险品后旧字段保留

1. 根因分类：B跨层合同、C变更传播、D覆盖缺口。共享控件把UN/CLASS/日期/PG设为
   undefined，航空创建与更新复用序列化时均省略；service正确按optional存在性合并，
   因而把未出现字段理解为保持旧值。海运已有显式空更新，但航空没有同步。
2. 之前公共字段显式清空只覆盖新增经营字段，危险品沿旧省略逻辑；创建未选的测试
   无法证明“已有值→取消→更新→重读”。这次不通过服务端从货类猜测清空绕过合同。
3. 防止重现：继续同一commonFields入口，创建未选省略，更新文本/日期显式空串、
   PG显式UNSPECIFIED零；测试真实JSON及AE/AI API落库，分别断言省略保持和显式清空。
4. 有限扩展：复用真实共享危险品控件做相邻取消回归，初始化不误清；不建立危险品
   自动规则或UN到PG数据库，不改变服务端权限/事务/版本。
5. 知识已写入公共字段规范和跨层指南；最终按批准F24/F25回填实际证据。

## 导出箱型标签脱离读快照

1. 根因分类：B跨层合同、D覆盖缺口。服务器虽在RR内读取箱量计划ID/数量，
   前端沿旧列表mapper额外请求当前箱型字典；只测试无箱计划DIRECT结果无法发现
   同一文件的标签不属于该响应快照。
2. 修复放在事实所有者：OrderContainerRequest只读投影code/name，Get/List/Export
   共用批量映射，导出回调内读取；前端消费投影，不复制字典缓存或另加同步平台。
3. 验证补含实际箱计划的205条场景，另一连接更改箱型name时（code既有immutable，不伪造setter），旧快照保持
   原投影、新请求读取新值；普通分页/无箱计划和授权裁列相邻保持。
4. 源/正规生成物/前端映射同组审阅，知识已写入order-cross-module-export.md，
   最终命令结果由server及独立checker报告记录。

## 子资源读取权变化后的旧导出

1. 根因分类：C变更传播、D覆盖缺口。导出会话原按用户/公司/kind重建，
   但只收回container/attachment读取权、保留订单read时模板仍存活；旧闭包可能
   在响应结束后按原授权列下载。
2. 最薄修复在现有会话身份中加入真实两个access位；变动卸载旧模板，使原
   isCurrent立即失效，新会话只构建仍有权列，不增加第二套权限判定。
3. 独立checker补撤回两个权限的在途拒下载，以及仍有权正常列的相邻验证；
   根spec已同步，实际批次与结果见cross-module-web-review。

## 成功封套省略被宽松工具放行

1. 根因分类：B跨层合同、E隐含假设。通用ensureListResponse/toTableRequest
   宽松处理缺success，而本组三条链的真实ok/okList固定输出success=true；
   非optional bool省略等于false，携旧data不能据此视为成功。
2. 最薄修复只约束queryOrderList、fetchOrderListExport与finance fees detail
   的真实成功封套；不全仓机械替换，不把普通领域false当错误或丢弃optional存在性。
3. 回归区分success=true、显式false夹旧data、success省略夹旧data，并同步成功
   mock的真实wire封套；根执行spec已写入，实际复验见独立web报告。
