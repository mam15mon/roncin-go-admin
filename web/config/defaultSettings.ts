import type { ProLayoutProps } from '@ant-design/pro-components';
import { layoutDimensions } from './layout';

/**
 * 全局后台布局默认配置
 * 纯白全高侧栏与紧凑顶部栏，共享尺寸由 layout.ts 统一管理。
 */
const Settings: ProLayoutProps & {
  logo?: string;
} = {
  navTheme: 'light',
  colorPrimary: '#1677ff',
  layout: 'mix',
  contentWidth: 'Fluid',
  fixedHeader: true,
  fixSiderbar: true,
  colorWeak: false,
  title: 'Roncin 货代后台',
  logo: '/logo.svg',
  iconfontUrl: '',
  siderWidth: layoutDimensions.siderWidth,
  splitMenus: false,
  token: {
    sider: {
      colorBgCollapsedButton: '#ffffff',
      colorTextCollapsedButton: 'rgba(0, 0, 0, 0.45)',
      colorTextCollapsedButtonHover: '#1677ff',
      // 折叠态子菜单浮层保持纯白，规范见 AGENTS.md 侧边栏交互。
      colorBgMenuItemCollapsedElevated: '#ffffff',
      colorBgMenuItemHover: 'rgba(0, 0, 0, 0.04)',
      colorBgMenuItemSelected: '#e6f4ff',
      colorTextMenu: '#475569',
      colorTextMenuSelected: '#1677ff',
      colorTextMenuItemHover: '#0f172a',
      colorTextMenuTitle: '#0f172a',
      colorMenuBackground: '#ffffff',
    },
    header: {
      colorBgHeader: '#ffffff',
      colorHeaderTitle: 'rgba(0, 0, 0, 0.88)',
      colorTextMenu: 'rgba(0, 0, 0, 0.65)',
      colorBgMenuItemHover: 'rgba(0, 0, 0, 0.04)',
      colorTextMenuSelected: '#1677ff',
      heightLayoutHeader: layoutDimensions.headerHeight,
    },
    pageContainer: {
      paddingBlockPageContainerContent:
        layoutDimensions.pageContainerPaddingBlock,
      paddingInlinePageContainerContent: layoutDimensions.contentPaddingInline,
    },
  },
};

export default Settings;
