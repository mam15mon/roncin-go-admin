import clsx from 'clsx';
import type React from 'react';
import type { QuarterRingProps } from './types';

export const QuarterRing: React.FC<QuarterRingProps> = ({
  size,
  duration = '1s',
  strokeWidth = '2.5px',
  className,
  style,
  ...props
}) => {
  const durationValue =
    typeof duration === 'number' ? `${duration}s` : duration;
  const strokeValue =
    typeof strokeWidth === 'number' ? `${strokeWidth}px` : strokeWidth;

  const sizeStyle: React.CSSProperties = size
    ? {
        width: typeof size === 'number' ? `${size}px` : size,
        height: typeof size === 'number' ? `${size}px` : size,
      }
    : {};

  return (
    <>
      <style>{`
        @keyframes loading-ui-quarter-ring-rotation {
          0% {
            transform: rotate(0deg);
          }
          100% {
            transform: rotate(360deg);
          }
        }
      `}</style>
      <span
        role="status"
        aria-label="loading"
        className={clsx(
          'inline-block rounded-full',
          !size && 'w-[1em] h-[1em]',
          className,
        )}
        style={{
          boxSizing: 'border-box',
          borderStyle: 'solid',
          borderColor: 'transparent',
          borderTopColor: 'currentColor',
          borderWidth: strokeValue,
          animation: `loading-ui-quarter-ring-rotation ${durationValue} linear infinite`,
          ...sizeStyle,
          ...style,
        }}
        {...props}
      >
        <span className="sr-only">Loading</span>
      </span>
    </>
  );
};

export default QuarterRing;
