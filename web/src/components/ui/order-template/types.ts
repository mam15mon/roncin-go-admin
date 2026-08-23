import type { ProFormInstance } from '@ant-design/pro-components';
import type { ReactNode } from 'react';

/** 订单表单模板的区块定义：每块渲染为一张卡片。 */
export interface OrderFormTemplateSection {
  key: string;
  title: string;
  content: ReactNode;
}

export interface OrderFormTemplateProps<T> {
  /** 主数据加载态；为 true 时渲染加载占位。 */
  loading?: boolean;
  /** 加载占位提示文案。 */
  loadingTip?: string;
  /** 外部持有的表单实例引用，用于跨组件读写表单值。 */
  formRef?: React.MutableRefObject<ProFormInstance | undefined>;
  /** 表单区块列表，按顺序渲染。 */
  sections: OrderFormTemplateSection[];
  /** 表单初始值。 */
  initialValues?: Partial<T>;
  /** 提交处理；返回 false 时表单停留在当前页。 */
  onFinish: (values: T) => Promise<boolean>;
  /** 提交按钮文案。 */
  submitText?: string;
  /** 重置按钮文案。 */
  resetText?: string;
}
