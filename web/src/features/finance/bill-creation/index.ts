/**
 * 财务建账能力公开入口：账单创建工作台及其递归私有组件收敛于本目录。
 * 内部模块之间使用相对路径互相引用；外部消费方只允许从本入口导入。
 */

export type { BillCreationWorkbenchProps } from './BillCreationWorkbench';
export { default as BillCreationWorkbench } from './BillCreationWorkbench';
/** 账单条款/信用预警同时服务建账工作台与账单页草稿编辑弹窗，随本能力公开。 */
export { default as BillTermsCreditWarnings } from './BillTermsCreditWarnings';
export type { BillCreationMode } from './billWorkbenchHelpers';
