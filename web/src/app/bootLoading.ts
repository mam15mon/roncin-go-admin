/** index.html 中首屏启动画面的节点 id，位于 #root 之外，不受 React 渲染影响。 */
export const BOOT_LOADING_ID = 'boot-loading';

/** 移除首屏启动画面；重复调用安全。须在首个真实页面提交后、绘制前调用。 */
export function dismissBootLoading(): void {
  document.getElementById(BOOT_LOADING_ID)?.remove();
}
