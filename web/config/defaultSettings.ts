import type { ProLayoutProps } from '@ant-design/pro-components';

/**
 * 全局后台布局默认配置
 * 统一对齐企业级后台设计：左侧全高深色侧栏（216px）、顶部56px紧凑浅色顶栏
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
  siderWidth: 180,
  splitMenus: false,
  token: {
    sider: {
      colorBgCollapsedButton: '#001529',
      colorTextCollapsedButton: 'rgba(255, 255, 255, 0.45)',
      colorTextCollapsedButtonHover: '#1677ff',
      // 折叠态子菜单浮层保持纯白，规范见 AGENTS.md 侧边栏交互。
      colorBgMenuItemCollapsedElevated: '#ffffff',
      colorBgMenuItemHover: 'rgba(255, 255, 255, 0.08)',
      colorBgMenuItemSelected: '#1677ff',
      colorTextMenu: 'rgba(255, 255, 255, 0.65)',
      colorTextMenuSelected: '#ffffff',
      colorTextMenuItemHover: '#ffffff',
      colorTextMenuTitle: 'rgba(255, 255, 255, 0.88)',
      colorMenuBackground: '#001529',
    },
    header: {
      colorBgHeader: '#ffffff',
      colorHeaderTitle: 'rgba(0, 0, 0, 0.88)',
      colorTextMenu: 'rgba(0, 0, 0, 0.65)',
      colorBgMenuItemHover: 'rgba(0, 0, 0, 0.04)',
      colorTextMenuSelected: '#1677ff',
      heightLayoutHeader: 48,
    },
    pageContainer: {
      paddingBlockPageContainerContent: 8,
      paddingInlinePageContainerContent: 12,
    },
  },
};

export default Settings;
