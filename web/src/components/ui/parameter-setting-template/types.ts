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
  /** Tab 栏类型，默认 'card' */
  tabType?: 'line' | 'card';
}
