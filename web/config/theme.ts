import type { ThemeConfig } from 'antd';

// antd 主题 token：自 web/config/config.ts 的 antd.configProvider.theme 原样平移。
// 全站高密度企业级视觉规范的唯一 token 真相源，修改需同步 AGENTS.md UI 规范。
export const themeConfig: ThemeConfig = {
  token: {
    fontFamily:
      'AlibabaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
    borderRadius: 6,
    borderRadiusSM: 4,
    borderRadiusLG: 8,
    colorPrimary: '#1677ff',
    colorBgLayout: '#f5f7fa',
    colorBgContainer: '#ffffff',
    colorBorder: '#e2e8f0',
    colorBorderSecondary: '#f1f5f9',
    colorText: '#0f172a',
    colorTextSecondary: '#475569',
    colorTextTertiary: '#94a3b8',
    colorTextQuaternary: '#cbd5e1',
    controlHeight: 32,
    controlHeightSM: 24,
    fontSize: 13,
    boxShadow:
      '0 1px 3px 0 rgba(0, 0, 0, 0.04), 0 1px 2px -1px rgba(0, 0, 0, 0.02)',
    boxShadowSecondary:
      '0 4px 6px -1px rgba(0, 0, 0, 0.06), 0 2px 4px -2px rgba(0, 0, 0, 0.04)',
    boxShadowTertiary:
      '0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -4px rgba(0, 0, 0, 0.04)',
  },
  components: {
    Card: {
      paddingLG: 14,
      padding: 12,
      paddingSM: 8,
      headerHeight: 40,
      headerFontSize: 13,
      colorBorderSecondary: '#f1f5f9',
    },
    Table: {
      headerBg: '#f8fafc',
      headerColor: '#475569',
      headerSortActiveBg: '#f1f5f9',
      headerSortHoverBg: '#f1f5f9',
      rowHoverBg: '#f8fafc',
      rowSelectedBg: '#eff6ff',
      rowSelectedHoverBg: '#dbeafe',
      cellPaddingBlock: 8,
      cellPaddingInline: 12,
      cellPaddingBlockSM: 6,
      cellPaddingInlineSM: 8,
      fontSize: 13,
      borderColor: '#f1f5f9',
      headerSplitColor: 'transparent',
    },
    Button: {
      controlHeight: 32,
      controlHeightSM: 24,
      paddingInline: 12,
      paddingInlineSM: 8,
      borderRadius: 6,
      defaultBorderColor: '#e2e8f0',
      defaultColor: '#334155',
      defaultBg: '#ffffff',
      defaultHoverBorderColor: '#cbd5e1',
      defaultHoverColor: '#0f172a',
      defaultHoverBg: '#f8fafc',
    },
    Input: {
      colorBorder: '#e2e8f0',
      hoverBorderColor: '#cbd5e1',
      activeBorderColor: '#1677ff',
      activeShadow: '0 0 0 3px rgba(22, 119, 255, 0.12)',
    },
    Select: {
      colorBorder: '#e2e8f0',
      hoverBorderColor: '#cbd5e1',
    },
    DatePicker: {
      colorBorder: '#e2e8f0',
      hoverBorderColor: '#cbd5e1',
    },
    Tabs: {
      itemColor: '#64748b',
      itemHoverColor: '#0f172a',
      itemSelectedColor: '#1677ff',
      titleFontSize: 13,
      horizontalItemPadding: '8px 12px',
    },
    Tag: {
      defaultBg: '#f1f5f9',
      defaultColor: '#475569',
    },
    Form: {
      itemMarginBottom: 12,
      verticalLabelPadding: '0 0 4px',
    },
    Modal: {
      headerBg: '#ffffff',
      contentBg: '#ffffff',
      borderRadiusLG: 10,
    },
    Drawer: {
      paddingLG: 16,
    },
  },
};
