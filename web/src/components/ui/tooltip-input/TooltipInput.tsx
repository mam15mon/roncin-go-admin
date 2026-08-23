import { Input, type InputRef, Tooltip } from 'antd';
import React, {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import type { TooltipInputProps } from './types';

/**
 * TooltipInput 全局通用带气泡提示的输入框组件
 *
 * 解决表单输入框中文字过长被遮挡、无法看清全貌的问题：
 * 1. 鼠标 hover 时自动展示当前输入框完整内容的 Tooltip；
 * 2. 完美支持 antd Form.Item 的表单状态收集与受控/非受控模式；
 * 3. 支持自定义 tooltipTitle、tooltipPlacement 和溢出检测。
 */
export const TooltipInput = forwardRef<InputRef, TooltipInputProps>(
  (
    {
      showTooltip = true,
      tooltipTitle,
      tooltipPlacement = 'top',
      autoDetectOverflow = false,
      value,
      defaultValue,
      onChange,
      ...restProps
    },
    ref,
  ) => {
    const inputRef = useRef<InputRef>(null);
    useImperativeHandle(ref, () => inputRef.current as InputRef);

    const [innerVal, setInnerVal] = useState<string>(
      () =>
        value !== undefined && value !== null
          ? String(value)
          : defaultValue !== undefined && defaultValue !== null
            ? String(defaultValue)
            : '',
    );
    const [isOverflow, setIsOverflow] = useState(false);

    const resolvedVal = value !== undefined ? value : innerVal;
    const stringVal = resolvedVal !== undefined && resolvedVal !== null ? String(resolvedVal) : '';

    const handleChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        setInnerVal(e.target.value);
        onChange?.(e);
      },
      [onChange],
    );

    const handleMouseEnter = useCallback(() => {
      if (!showTooltip || !stringVal) {
        setIsOverflow(false);
        return;
      }
      if (autoDetectOverflow) {
        const inputEl = inputRef.current?.input;
        if (inputEl) {
          setIsOverflow(inputEl.scrollWidth > inputEl.clientWidth);
        } else {
          setIsOverflow(true);
        }
      } else {
        setIsOverflow(true);
      }
    }, [showTooltip, stringVal, autoDetectOverflow]);

    const displayTitle = tooltipTitle ?? (stringVal || undefined);
    const shouldOpen = showTooltip && Boolean(displayTitle) && isOverflow;

    return (
      <Tooltip
        title={shouldOpen ? displayTitle : ''}
        placement={tooltipPlacement}
        destroyOnHidden
      >
        <span
          style={{ display: 'inline-block', width: '100%' }}
          onMouseEnter={handleMouseEnter}
        >
          <Input
            ref={inputRef}
            value={value}
            defaultValue={defaultValue}
            onChange={handleChange}
            {...restProps}
          />
        </span>
      </Tooltip>
    );
  },
);

TooltipInput.displayName = 'TooltipInput';
