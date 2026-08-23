import { CopyOutlined } from '@ant-design/icons';
import { App, Tooltip } from 'antd';
import React, {
  useCallback,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { EllipsisTooltipProps } from './types';

/**
 * 递归提取 React 节点中的纯文本内容
 */
function extractText(node: React.ReactNode): string {
  if (typeof node === 'string') return node;
  if (typeof node === 'number') return String(node);
  if (!node) return '';
  if (Array.isArray(node)) return node.map(extractText).join('');
  if (React.isValidElement(node)) {
    return extractText((node.props as { children?: React.ReactNode }).children);
  }
  return '';
}

/**
 * EllipsisTooltip 全局通用文本省略与气泡提示组件
 *
 * 特性：
 * 1. 自动检测文本是否实际发生截断（scrollWidth > clientWidth / scrollHeight > clientHeight），仅在溢出时弹出气泡；
 * 2. 支持单行及多行省略（通过 maxLines 控制）；
 * 3. 支持复制完整文本功能（copyable）；
 * 4. 采用稳定的 Tooltip 结构包装，无闪烁；全面兼容 antd Tooltip 的配置。
 */
export const EllipsisTooltip: React.FC<EllipsisTooltipProps> = ({
  children,
  title,
  maxWidth = '100%',
  width,
  maxLines = 1,
  autoDetect = true,
  alwaysShowTooltip = false,
  copyable = false,
  copySuccessText = '已复制到剪贴板',
  tooltipProps,
  className,
  style,
}) => {
  const containerRef = useRef<HTMLSpanElement>(null);
  const [isOverflow, setIsOverflow] = useState(false);
  const app = App.useApp();
  const message = app?.message;

  const plainText = useMemo(() => extractText(children), [children]);
  const resolvedTitle = title ?? plainText;

  // 鼠标移入时检测是否发生溢出截断
  const checkOverflow = useCallback(() => {
    if (alwaysShowTooltip) {
      setIsOverflow(true);
      return;
    }
    if (!autoDetect) {
      setIsOverflow(true);
      return;
    }
    const el = containerRef.current;
    if (!el) return;

    if (maxLines > 1) {
      const overflowY = el.scrollHeight > el.clientHeight;
      setIsOverflow(overflowY);
    } else {
      const overflowX = el.scrollWidth > el.clientWidth;
      setIsOverflow(overflowX);
    }
  }, [alwaysShowTooltip, autoDetect, maxLines]);

  const handleCopy = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      if (!plainText) return;
      try {
        if (navigator.clipboard?.writeText) {
          await navigator.clipboard.writeText(plainText);
        } else {
          // 降级使用 textarea 复制
          const textarea = document.createElement('textarea');
          textarea.value = plainText;
          textarea.style.position = 'fixed';
          textarea.style.opacity = '0';
          document.body.appendChild(textarea);
          textarea.select();
          document.execCommand('copy');
          document.body.removeChild(textarea);
        }
        message?.success?.(copySuccessText);
      } catch {
        message?.error?.('复制失败，请手动选择复制');
      }
    },
    [plainText, copySuccessText, message],
  );

  const containerStyle: React.CSSProperties = useMemo(() => {
    if (maxLines > 1) {
      return {
        display: '-webkit-box',
        WebkitLineClamp: maxLines,
        WebkitBoxOrient: 'vertical',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        wordBreak: 'break-all',
        maxWidth,
        width,
        ...style,
      };
    }

    return {
      display: 'inline-block',
      verticalAlign: 'bottom',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
      wordBreak: 'break-all',
      maxWidth,
      width,
      ...style,
    };
  }, [maxLines, maxWidth, width, style]);

  if (!children && !title) {
    return null;
  }

  const shouldShowTooltip = Boolean(resolvedTitle) && (alwaysShowTooltip || isOverflow);

  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        maxWidth: '100%',
      }}
    >
      <Tooltip
        title={shouldShowTooltip ? resolvedTitle : ''}
        {...tooltipProps}
      >
        <span
          ref={containerRef}
          className={className}
          style={containerStyle}
          onMouseEnter={checkOverflow}
        >
          {children}
        </span>
      </Tooltip>
      {copyable && plainText && (
        <CopyOutlined
          title="点击复制"
          style={{
            marginLeft: 4,
            fontSize: 12,
            color: '#8c8c8c',
            cursor: 'pointer',
            flexShrink: 0,
          }}
          className="roncin-ellipsis-copy-icon"
          onClick={handleCopy}
        />
      )}
    </span>
  );
};
