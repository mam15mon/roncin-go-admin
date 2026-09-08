import type { ProFormInstance } from '@ant-design/pro-components';
import type { ReactNode } from 'react';

/** 订单表单模板的区块定义：每块渲染为一张卡片。 */
export interface OrderFormTemplateSection {
  key: string;
  title: string;
  extra?: ReactNode;
  content: ReactNode;
}

export interface OrderFormTemplateProps<T> {
  /** 主数据加载态；为 true 时渲染加载占位。 */
  loading?: boolean;
  /** 加载占位提示文案。 */
  loadingTip?: string;
  /** 是否为只读/详情查看模式 */
  readonly?: boolean;
  /** 顶部吸顶导航或自定义头部 */
  header?: ReactNode;
  /** 外部持有的表单实例引用，用于跨组件读写表单值。 */
  formRef?: React.MutableRefObject<ProFormInstance | undefined>;
  /** 前置自定义区块列表（如：订单状态流程） */
  prependSections?: OrderFormTemplateSection[];
  /** 核心业务区块列表，按顺序渲染。 */
  sections: OrderFormTemplateSection[];
  /** 后置自定义区块列表（如：操作记录日志） */
  appendSections?: OrderFormTemplateSection[];
  /** 表单初始值。 */
  initialValues?: Partial<T>;
  /** 提交处理；返回 false 时表单停留在当前页（只读模式下可选）。 */
  onFinish?: (values: T) => Promise<boolean>;
  /** 提交按钮文案。 */
  submitText?: string;
  /** 重置按钮文案。 */
  resetText?: string;
  /** 底部额外操作栏插槽 */
  footer?: ReactNode;
  /** 自定义页签 Key，不传时自动根据当前路由解析 */
  tabKey?: string;
  /** 外部受控脏检查状态；若不传则由组件内部自动追踪 */
  dirty?: boolean;
  /** 脏检查状态发生变化时的回调 */
  onDirtyChange?: (dirty: boolean) => void;
  /** 是否启用页签关闭拦截提示，默认为 true */
  enableCloseGuard?: boolean;
  /** 自定义关闭提示文案，默认："修改的信息尚未保存，您确定要离开吗？" */
  closeGuardMessage?: string;
  /** 表单值变动回调 */
  onValuesChange?: (changedValues: any, allValues: T) => void;
  /** 表单重置回调 */
  onReset?: () => void;
}
