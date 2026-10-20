# 实施记录：未保存输入保护

- 工作区：`/tmp/roncin-org-switch-20261020`。
- 分工：共享守卫、共享表单模板、管理领域真实编辑表单；未修改导航、其他领域产品文件；没有提交 Git。

## 公共契约与实现

- `components/layout/tabCloseGuard.ts`：每个 tabKey 保存多个 guard；注销只移除自己；任一实时 guard dirty 或当前组织持久草稿均阻止页签关闭；组织切换仍只检查挂载的真实 guard，不扫描其他组织草稿。一次组织离开确认覆盖当次所有 dirty guard，异步结束/失败撤销临时放行；新注册 guard 不借旧确认放行。
- `components/layout/useUnsavedChangesGuard.ts`：`useUnsavedChangesGuard({ enabled?, identity?, tabKey?, message? })` 返回稳定的 `markDirty/resetDirty`；同步 ref 标记，开闭/记录身份变化清理；默认复用 `history.location.pathname` 与 `resolveTabKey`，无需引入 router 上下文。
- `components/ui/GuardedModalForm.tsx`：经 `@/components/ui` 公开。完整透传官方 `ModalFormProps<T,U>`、formRef、业务 onValuesChange 与提交 Promise，仅用户 onValuesChange 登记保护；关闭、身份变动或 `onFinish === true` 清理。官方本地类型不支持 defaultOpen；受控 open 和官方 trigger 内部开闭均有真实测试。
- `components/layout/useModalOwnerSignal.ts`：返回 getter，事件打开 App.modal 时才获取当前 AbortSignal；所属组件 layout cleanup abort，StrictMode 重建使用新信号。
- `utils/confirmWithReason.tsx`：真实原因输入登记 guard；支持 owner signal 销毁旧组织弹窗；失败保留脏状态。`makeVersionActions(app, getOwnerSignal?)` 在事件打开时透传信号。实际本地 Ant Design destroy 不结算确认 thenable，因此以 owner abort Promise.race 显式结束等待并注销守卫，不能仅在 await confirm 的 finally 清理。

## 覆盖

- 四个共享 CRUD 模板：MasterDataTemplate、SettingTableTemplate、SubEntityCardGrid、SubEntityDrawerTemplate；另发现真实 QuickAddFee/Partner 消费的 QuickCreateModal，已在原 Form.onValuesChange 接入，提交返回实体后清理，失败保留。
- 管理独立表单：NumberRuleEditModal、OrgCreateModal、OrgEditModal、DingTalkAssignmentModal、ResetPasswordModal、UserFormModal、UserMembershipModal、RoleFormModal。
- RoleFormModal 的业务权限 setter 额外标脏，初始化权限 prop 不走 setter，搜索/导航/展开不标脏；角色字段由包装登记。两个 guard 同 tab 独立，关闭/成功时均清理。
- 管理旧测试只修公共出口 mock 与缺失的 history.location mock；没有在产品加测试环境回退。
- 原因输入调用方的 owner signal 与其他领域 Form/state 编辑由领域代理接入，本记录不代替其覆盖验收。

## 已完成验证

1. 守卫与共享组件、真实角色和联系人及原因框定向：11 个文件、48 项通过。命令：
   `pnpm --dir web exec vitest run src/components/layout/tabCloseGuard.test.ts src/components/layout/useUnsavedChangesGuard.test.tsx src/components/ui/GuardedModalForm.test.tsx src/components/ui/sub-entity-card-grid/SubEntityCardGrid.unsaved.test.tsx src/pages/admin/components/roles/RoleFormModal.unsaved.test.tsx src/pages/admin/components/roles/RoleFormModal.test.tsx src/components/ui/parameter-setting-template/SettingTableTemplate.test.tsx src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.test.tsx src/components/ui/master-data-template/MasterDataTemplate.test.tsx src/utils/confirmWithReason.unsaved.test.tsx src/utils/versionActions.test.ts`。
2. 管理既有测试：UserFormModal、roles、roles.grantable、users 共 4 个文件、33 项通过。
3. 最后 `pnpm --dir web exec tsc --noEmit --pretty false`：全工作树通过。
4. 本分组 31 个修改/新增文件 Biome check 通过，`git diff --check` 通过。
5. `pnpm run check:architecture`：18 项测试和 459 个产品文件边界扫描通过。
6. 本地 Ant Design CLI 对 GuardedModalForm 进行 lint：无问题。

## 尚需主会话完成

独立 review、最终完整前端门禁、其他领域与导航合并状态验证、规范和任务归档、Git 提交及合并 main。分组未运行完整前端测试，不以定向结果冒充最终验收。

## 补充覆盖：业务标签弹窗

主会话全局核对补充发现 `components/business-tag/BusinessTagModal.tsx`，本分组追加所有权并完成：

- 外层 `selectedTagIds` 是实际关联/移除提交载荷，用户选择明确标记独立守卫；仅搜索候选、打开快捷表单和加载标签组不标脏。
- 快捷标签 Form 的名称、标签组用户 `onValuesChange` 使用另一个守卫；只在真实可见时启用，失败保持输入与 dirty。
- 快捷创建成功仅清理快捷 Form guard，自动选中的新标签登记为尚待关联的外层 dirty；外层失败继续保护，沿用原 `onSubmit !== false` 成功契约，成功关闭才清理两者。没有更改既有业务请求成功判定。
- 新增 `BusinessTagModal.guard.test.tsx`：真实候选搜索无提示、待关联选择在组织切换取消/失败后保持、仅填写名称、仅选择组、快捷创建失败、成功创建后仍待关联、外层失败/成功生命周期共 4 项通过（单文件 2.79 秒，无长尾）。
- 命令：`pnpm --dir web exec vitest run src/components/business-tag/BusinessTagModal.guard.test.tsx`；两个修改/新增文件 Biome check 通过。按主会话分工未重跑完整门禁。

## 补充覆盖：企业资源批量状态编辑

最终复扫补齐 `BatchAddressTypeModal`、`BatchAssigneeModal`、`BatchAssociationModal` 三个真实批量编辑入口，各在 `open` 期间使用独立 `useUnsavedChangesGuard`：用户改分配/移除模式、勾地址类型、选择关联人员或企业均标脏；默认 prop 初始化、搜索候选不标脏。父层 `onOk: () => void` 原业务契约保持，组件不猜测提交成功；父层失败保持 open 即保持脏状态，成功或取消关闭后自动注销。

新增 `BatchEditModals.guard.test.tsx` 的三个真实渲染用例覆盖默认地址初始化、模式切换、地址勾选/失败保留、人员及企业搜索无误提示、待提交选择、组织切换取消与失败、父层成功关闭清理。定向 `pnpm --dir web exec vitest run src/pages/enterprise-resources/components/BatchEditModals.guard.test.tsx` 结果 1 文件 3 项通过（1.10 秒）；4 个新增/修改文件 Biome check 与定向 diff --check 通过。未运行完整门禁。
