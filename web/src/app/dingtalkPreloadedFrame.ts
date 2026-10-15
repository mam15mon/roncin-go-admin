import { type FrameConfig, PRELOADED_FRAME_ID } from './dingtalkLoginProtocol';

export function discardPreloadedDingTalkFrame(): void {
  document.getElementById(PRELOADED_FRAME_ID)?.remove();
}

function normalizedURL(value: string): string {
  const url = new URL(value);
  url.searchParams.sort();
  return url.href;
}

export function claimPreloadedDingTalkFrame(
  config: FrameConfig,
): HTMLIFrameElement | undefined {
  const frame = document.getElementById(PRELOADED_FRAME_ID);
  if (!frame) return undefined;
  if (
    !(frame instanceof HTMLIFrameElement) ||
    normalizedURL(frame.src) !== normalizedURL(config.src) ||
    frame.dataset.expiresAt !== String(config.expiresAt) ||
    config.expiresAt <= Date.now()
  ) {
    frame.remove();
    return undefined;
  }
  return frame;
}

// 外部 iframe 永远留在 HTML 原位置；改几何样式不会重建浏览上下文。
export function alignPreloadedDingTalkFrame(
  frame: HTMLIFrameElement,
  slot: HTMLElement,
): () => void {
  let lastGeometry = '';
  const align = () => {
    const rect = slot.getBoundingClientRect();
    const geometry = [
      rect.left,
      rect.top,
      rect.width,
      rect.height,
      window.scrollX,
      window.scrollY,
    ].join(':');
    if (geometry === lastGeometry) return;
    lastGeometry = geometry;
    Object.assign(frame.style, {
      position: 'absolute',
      left: `${rect.left + window.scrollX}px`,
      top: `${rect.top + window.scrollY}px`,
      width: `${rect.width}px`,
      height: `${rect.height}px`,
      visibility: rect.width > 0 && rect.height > 0 ? 'visible' : 'hidden',
      border: '0',
      borderRadius: '8px',
      zIndex: '1',
    });
  };
  // 整个登录卡片可能随字体、提示和窗口变化平移，用帧同步避免只观察尺寸漏掉位置变化。
  let animation = 0;
  const track = () => {
    align();
    animation = window.requestAnimationFrame(track);
  };
  track();
  return () => {
    window.cancelAnimationFrame(animation);
    frame.style.visibility = 'hidden';
  };
}
