/** 全站共享布局尺寸；组件数值与 CSS 变量均由此配置派生。 */
export const layoutDimensions = {
  siderWidth: 208,
  collapsedSiderWidth: 48,
  headerHeight: 48,
  tagsHeight: 36,
  headerPaddingInline: 14,
  mobileHeaderPaddingInline: 8,
  contentPaddingInline: 12,
  contentPaddingBlock: 10,
  pageContainerPaddingBlock: 8,
  pageSectionGap: 12,
  collapsedMenuItemSize: 36,
  siderTriggerHeight: 40,
  siderMenuBottomGap: 10,
  formNavigatorWidth: 140,
  formNavigatorRight: 16,
  formNavigatorContentGap: 24,
  formNavigatorBottomReserve: 64,
  formScrollOffset: 146,
  formNavigatorTopOffset: 156,
  formScrollGap: 12,
  tableErrorOffset: 100,
  tableErrorScrollGap: 16,
  stickyFooterPaddingBlock: 10,
  stickyFooterPaddingInline: 24,
  stickyFooterMarginTop: 16,
} as const;

/** 保持联动关系，避免单独维护第二份尺寸。 */
export const layoutOffsets = {
  topStackHeight: layoutDimensions.headerHeight + layoutDimensions.tagsHeight,
  formNavigatorContentReserve:
    layoutDimensions.formNavigatorWidth +
    layoutDimensions.formNavigatorContentGap,
  siderMenuPaddingBottom:
    layoutDimensions.siderTriggerHeight + layoutDimensions.siderMenuBottomGap,
} as const;

/** 公共布局层级分别命名，避免数值相同的层级相互绑定。 */
export const layoutLayers = {
  header: 100,
  sider: 101,
  siderTrigger: 102,
  tags: 19,
  pageHeader: 18,
  footer: 15,
  formNavigator: 88,
} as const;

function toCssVariables(
  values: Readonly<Record<string, number>>,
  prefix: string,
  unit: string,
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(values).map(([key, value]) => [
      `${prefix}${key.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`)}`,
      `${value}${unit}`,
    ]),
  );
}

/** 在主入口安装到 html，使页面与挂到 body 的浮层共享同一份配置。 */
export const layoutCssVariables = {
  ...toCssVariables(layoutDimensions, '--roncin-', 'px'),
  ...toCssVariables(layoutOffsets, '--roncin-', 'px'),
  ...toCssVariables(layoutLayers, '--roncin-layer-', ''),
};
