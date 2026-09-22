import type { ProFormInstance } from '@ant-design/pro-components';
import type { ReactNode } from 'react';

/** 订单表单模板的区块定义：每块渲染为一张卡片。 */
export interface OrderFormTemplateSection {
  key: string;
  title: string;
  extra?: ReactNode;
  content: ReactNode;
  /** 根据当前表单值同步控制分节与导航入口。 */
  visible?: (values: Record<string, unknown>) => boolean;
}

/** 模板对外暴露的显式动作；草稿与脏状态生命周期由模板独占管理。 */
export interface OrderFormTemplateActions<T> {
  /** 清当前草稿、重置 Form store、可选回填指定值并清除脏状态。 */
  resetTo: (values?: Partial<T>) => void;
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
  /** 提交按钮配置；设为 false 时彻底隐藏底部提交栏（如已在顶部页头提供操作按钮时） */
  submitter?: false;
  /** 底部额外操作栏插槽 */
  footer?: ReactNode;
  /** 稳定菜单页签 Key；只有显式提供时才注册页签关闭守卫。 */
  tabKey?: string;
  /** 规范化的业务路径，作为草稿键的一部分；缺少任一草稿身份输入时不读写持久草稿。 */
  draftPathname?: string;
  /** 用户与组织组成的草稿命名空间；缺失时不持久化草稿。 */
  draftScope?: string;
  /** 接收模板显式动作（如 resetTo）的外部引用。 */
  actionsRef?: React.MutableRefObject<OrderFormTemplateActions<T> | undefined>;
  /** 是否启用页签关闭拦截提示，默认为 true */
  enableCloseGuard?: boolean;
  /** 自定义关闭提示文案，默认："修改的信息尚未保存，您确定要离开吗？" */
  closeGuardMessage?: string;
  /** 表单值变动回调 */
  onValuesChange?: (changedValues: Partial<T>, allValues: T) => void;
  /** 表单重置回调 */
  onReset?: () => void;
  /**
   * 校验失败定位前置回调：在滚动定位首个错误前调用，供页面把落在
   * 隐藏区域（页签、折叠备注）的首个错误字段变为可见；等待返回的
   * Promise 完成后再执行滚动与聚焦。通用模板不感知具体业务区块。
   */
  onRevealError?: (errorInfo: {
    errorFields: { name: (string | number)[]; errors?: string[] }[];
  }) => void | Promise<void>;
  /** 是否显示右侧楼层导航与错误定位微标，默认为 true */
  showAnchorNav?: boolean;
}
