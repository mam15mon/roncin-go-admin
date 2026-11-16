# 实施验收

## 交付范围与证据

| 验收 | 落地内容 | 主要验证 |
| --- | --- | --- |
| AC1 | 海运基线保留；航空六列紧凑新建/详情，AI承运文案 | air-create-layout、air-document-mode、form-adapter；四类页面浏览器 |
| AC2 | 订单内完整运输摘要、原地预览与共同运输更新；最新只读展示不替换原编辑基线 | AirTransportMaintenance、air-transport-summary；原地弹窗浏览器 |
| AC3 | 委托/完整实际货量/参考/确认集中核对，未保存货量提示，历史折叠 | AirCargoWorkspace、AirWeightPanel；真实PG确认/过期/来源专项 |
| AC4 | MAWB/HAWB页签、DIRECT仅主单、低频折叠与错误定位、顶部单证菜单 | air-document-workspace、air-document-mode、OrderDetailHeader、SeaExportDetailFeatures |
| AC5 | SI/AI进口服务前移；独立保存保持订单输入和精确版本 | 两类detail-*-import-service-save、浏览器位置断言 |
| AC6 | 四类列表保持官方请求；航空本票双来源重量摘要、进口ETA与服务 | list-query.air-weight、air-list-columns、list.columns、公共模板测试；批量PG投影 |
| AC7 | 类型、Biome、架构/生成门禁、定向及全量测试、浏览器布局 | 下方门禁结果与browser-acceptance/browser-extra记录 |

## 已执行的局部验证

- 主会话整合定向：15文件105项通过，覆盖列表映射/导出、重量与运输、模板、页头和两类进口独立保存。
- 模板专项：7文件28项通过；只读回归补充3文件8项，独立复核另5文件21项通过。
- 列表专项：原7文件65项通过，旧列断言修正并拆分长尾后4文件17项通过。
- 海运类型扩展：12项通过，测试接入新documentMenuItems，保留权限/锁态断言。
- `go -C server test ./internal/data ./internal/service ./internal/biz`通过；真实 PostgreSQL AE/AI 专项按正式迁移构建隔离Schema，两个子用例零跳过，详情见list-implementation.md。
- Proto、Go绑定、OpenAPI、前端客户端按正式生成命令更新；无数据库Schema变更，无迁移需求。

## 最终门禁

- 首次 `pnpm run check:fast`：后端门禁、Go测试、vet与漏洞扫描通过；前端发现两处旧入口/列断言，及检查运行期间新增只读测试与旧转换缓存混用。共531文件通过、4文件失败、1文件跳过。四处均已修正并定向通过。
- 最终 `pnpm run check:web` 退出0：537文件通过、1文件跳过；2878测试通过、11测试跳过。包含生成契约校验、架构/重复代码约束、Biome与TypeScript。最后的单证区纯样式全宽修改另以浏览器真实边界框2项聚焦及TypeScript/Biome验证，不为CSS重复整套门禁。
- 保留既有未使用导入等非阻断Biome警告，不扩大范围清理无关代码；无规则关闭或新豁免。

- 浏览器：海运出口8项、空运进出口8项、SI及航空DIRECT/切换补充8项，共24个场景通过；全宽修正另2项聚焦通过，未知请求/写入/页面异常为空。SI既有useForm警告单列在browser-extra.md。

## 复核修正

- 父表单禁用不能阻止查看货物、单证、历史和重新核对；读动作独立解除禁用继承，写入口继续守原权限与锁态。
- MAWB只读正文不注册第二套输入；共同运输刷新只更新只读摘要，整票稿件版本保持原值。
- 页签竖向布局显式stretch，避免内容只占半幅；低频正文懒挂载，避免隐藏文本域测量与长尾。
- 列偏好测试按功能拆分，避免单文件超过20秒。

## 验证边界

浏览器用真实页面与明确的合成接口测试布局、导航、页签及弹窗，未写开发业务库；不将可见性验收视为真实用户权限、真实数据库写入或事务证明。写入/失败/迟到响应/输入保护由组件和服务端测试覆盖。没有生产部署或构建配置变更，本次未运行生产构建。
