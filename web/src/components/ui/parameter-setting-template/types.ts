import type { ActionType, ProColumns } from '@ant-design/pro-components';
import type { ModalProps } from 'antd';
import type { TableRowSelection } from 'antd/es/table/interface';
import type { ReactNode } from 'react';

export interface ParameterSettingTabItem {
  /** 唯一标识 */
  key: string;
  /** 选项卡标签名 */
  label: ReactNode;
  /** 图标 */
  icon?: ReactNode;
  /** 权限或条件可见性，默认 true */
  visible?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 内容面板 */
  children: ReactNode;
  /** 徽标或附加标记 */
  badge?: ReactNode;
  /** 提示文案 */
  tooltip?: string;
}

export interface ParameterSettingTemplateProps {
  /** 页面/容器主标题 */
  title?: ReactNode;
  /** 页面/容器副标题 */
  subTitle?: ReactNode;
  /** 顶部右侧自定义操作区 */
  extra?: ReactNode;
  /** Tab 选项卡列表 */
  items: ParameterSettingTabItem[];
  /** 当前激活的 Tab key（受控） */
  activeKey?: string;
  /** 默认激活的 Tab key */
  defaultActiveKey?: string;
  /** 切换 Tab 回调 */
  onChange?: (activeKey: string) => void;
  /** 是否将当前 activeTab 同步到 URL Query 参数中（key 默认为 'tab'） */
  syncUrlQuery?: boolean;
  /** URL Query 参数名，默认 'tab' */
  queryParamKey?: string;
  /** 容器外层自定义样式 */
  style?: React.CSSProperties;
  /** 容器外层类名 */
  className?: string;
  /** Tab 栏类型，默认 'line' */
  tabType?: 'line' | 'card';
}

export interface SettingTableTemplateProps<
  TRecord extends Record<string, any> = Record<string, any>,
  TFormValues extends Record<string, any> = Record<string, any>,
> {
  /** 实体业务名称，如 '计费单位'、'异常情况'、'货物或应税劳务' */
  entityName: string;
  /** 行主键，默认 'id' */
  rowKey?: string;
  /** ProTable 列定义 */
  columns: ProColumns<TRecord>[];
  /** 渲染弹窗表单项 */
  renderFormItems: (editingRecord?: TRecord) => ReactNode;
  /** 异步获取数据 */
  query: (params?: any) => Promise<{
    data?: TRecord[];
    success?: boolean;
    total?: number;
  }>;
  /** 异步创建数据 */
  createItem?: (values: TFormValues) => Promise<any>;
  /** 异步更新数据 */
  updateItem?: (record: TRecord, values: TFormValues) => Promise<any>;
  /** 是否具备创建权限，默认 true */
  canCreate?: boolean;
  /** 是否具备编辑权限，默认 true */
  canUpdate?: boolean;
  /** 初始表单值生成函数或对象 */
  initialValues?: (editingRecord?: TRecord) => Partial<TFormValues>;
  /** 提交前数据转换钩子 */
  beforeSubmit?: (values: TFormValues, editingRecord?: TRecord) => any;
  /** 弹窗宽度，默认 520 */
  modalWidth?: number;
  /** 是否开启 ModalForm Grid 栅格布局，默认 false */
  grid?: boolean;
  /** 弹窗表单布局方式，默认 'horizontal' */
  layout?: 'horizontal' | 'vertical';
  /** 标签宽度（如 145 或 '145px'），在 horizontal 下有效，默认自动计算（两列时 145px，单列时 140px） */
  labelWidth?: number | string;
  /** 自定义 labelCol 配置 */
  labelCol?: any;
  /** 自定义 wrapperCol 配置 */
  wrapperCol?: any;
  /** ModalForm 的 rowProps（如栅格间距） */
  rowProps?: any;
  /** 自定义 Modal 属性 */
  modalProps?: Partial<ModalProps>;
  /** 是否开启搜索表单，默认 false */
  search?: boolean;
  /** 是否开启分页，默认 false */
  pagination?: boolean | { pageSize?: number };
  /** 表格多选行配置 */
  rowSelection?: TableRowSelection<TRecord> | false;
  /** 横向/纵向滚动条配置 */
  scroll?: { x?: number | string; y?: number | string };
  /** 顶部工具栏额外按钮 */
  extraToolBarButtons?: ReactNode[];
  /** 卡片外层自定义样式 */
  cardStyle?: React.CSSProperties;
  /** 自定义 ActionRef 暴露 */
  actionRef?: React.MutableRefObject<ActionType | undefined>;
}
