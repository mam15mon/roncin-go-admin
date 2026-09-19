import {
  ExclamationCircleOutlined,
  MenuFoldOutlined,
  OrderedListOutlined,
} from '@ant-design/icons';
import { Badge, Button, Space, Tooltip, Typography } from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import {
  scrollToSectionWithStickyOffset,
  waitForLayoutStable,
} from './formErrorUtils';
import type { FormAnchorNavProps } from './types';

const { Text } = Typography;

export const FormAnchorNav: React.FC<FormAnchorNavProps> = ({
  items,
  sectionErrors = {},
  activeKey: controlledActiveKey,
  onSelect,
  onErrorClick,
  style,
  className,
  targetOffset = 146,
  defaultCollapsed = false,
  onCollapsedChange,
}) => {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  const updateCollapsed = (next: boolean) => {
    setCollapsed(next);
    onCollapsedChange?.(next);
  };
  const [internalActiveKey, setInternalActiveKey] = useState<string>(
    items[0]?.key || '',
  );
  const activeKey = controlledActiveKey ?? internalActiveKey;

  // 统计总错误数
  const totalErrorCount = useMemo(() => {
    return Object.values(sectionErrors).reduce((sum, count) => sum + count, 0);
  }, [sectionErrors]);

  // 监听页面滚动，自动高亮当前所在的分节
  useEffect(() => {
    const handleScroll = () => {
      const scrollY = window.pageYOffset || document.documentElement.scrollTop;
      const threshold = scrollY + targetOffset + 120;

      let currentKey = items[0]?.key || '';
      for (const item of items) {
        const el =
          document.getElementById(`section-${item.key}`) ||
          document.querySelector(`[data-section-key="${item.key}"]`);
        if (el) {
          const top = el.getBoundingClientRect().top + scrollY;
          if (top <= threshold) {
            currentKey = item.key;
          }
        }
      }
      setInternalActiveKey(currentKey);
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [items, targetOffset]);

  // 点击跳转到指定 Section
  const handleItemClick = (key: string) => {
    const errCount = sectionErrors[key] || 0;
    if (errCount > 0 && onErrorClick) {
      onErrorClick(key);
      return;
    }

    onSelect?.(key);
    setInternalActiveKey(key);

    const targetEl =
      document.getElementById(`section-${key}`) ||
      document.querySelector(`[data-section-key="${key}"]`);
    if (!targetEl) return;
    // onSelect 可能触发折叠分节展开/数据加载：等布局稳定后再按实测吸顶
    // 高度落位，避免平滑滚动与布局变化竞态导致落点漂移或被按钮栏遮挡。
    void waitForLayoutStable(targetEl).then(() => {
      if (!targetEl.isConnected) return;
      scrollToSectionWithStickyOffset(targetEl, targetOffset);
    });
  };

  if (!items || items.length === 0) {
    return null;
  }

  // 折叠状态（迷你浮标）
  if (collapsed) {
    return (
      <div
        className={`roncin-form-anchor-nav-collapsed ${className || ''}`}
        style={{
          position: 'fixed',
          right: 16,
          top: 156,
          zIndex: 88,
          ...style,
        }}
      >
        <Tooltip
          title={
            totalErrorCount > 0
              ? `表单有 ${totalErrorCount} 处校验错误，点击展开导航`
              : '展开表单分节大纲'
          }
          placement="left"
        >
          <Badge count={totalErrorCount} offset={[-4, 4]} size="small">
            <Button
              shape="circle"
              icon={
                totalErrorCount > 0 ? (
                  <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
                ) : (
                  <OrderedListOutlined />
                )
              }
              onClick={() => updateCollapsed(false)}
              style={{
                boxShadow: '0 3px 12px rgba(0, 0, 0, 0.12)',
                backgroundColor: '#ffffff',
                borderColor: totalErrorCount > 0 ? '#ff4d4f' : '#d9d9d9',
              }}
            />
          </Badge>
        </Tooltip>
      </div>
    );
  }

  return (
    <div
      className={`roncin-form-anchor-nav ${className || ''}`}
      style={{
        position: 'fixed',
        right: 16,
        top: 156,
        width: 140,
        maxHeight: 'calc(100vh - 200px)',
        backgroundColor: 'rgba(255, 255, 255, 0.94)',
        backdropFilter: 'blur(8px)',
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        boxShadow: '0 4px 16px rgba(0, 0, 0, 0.08)',
        zIndex: 88,
        display: 'flex',
        flexDirection: 'column',
        padding: '8px 6px',
        userSelect: 'none',
        ...style,
      }}
    >
      {/* 头部标题与收起按钮 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '2px 6px 6px',
          borderBottom: '1px solid #f5f5f5',
          marginBottom: 4,
        }}
      >
        <Space size={4}>
          <Text strong style={{ fontSize: 12, color: 'rgba(0, 0, 0, 0.75)' }}>
            表单导航
          </Text>
          {totalErrorCount > 0 && (
            <Badge
              count={totalErrorCount}
              size="small"
              style={{ backgroundColor: '#ff4d4f' }}
            />
          )}
        </Space>
        <Button
          type="text"
          size="small"
          icon={
            <MenuFoldOutlined
              style={{ fontSize: 12, color: 'rgba(0, 0, 0, 0.45)' }}
            />
          }
          onClick={() => updateCollapsed(true)}
          style={{ width: 22, height: 22, padding: 0 }}
        />
      </div>

      {/* 楼层列表 */}
      <nav
        aria-label="表单分节大纲"
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
          overflowY: 'auto',
        }}
      >
        {items.map((item) => {
          const isCurrent = activeKey === item.key;
          const errCount = sectionErrors[item.key] || 0;

          return (
            <button
              type="button"
              key={item.key}
              onClick={() => handleItemClick(item.key)}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '5px 8px',
                borderRadius: 4,
                cursor: 'pointer',
                fontSize: 12,
                backgroundColor: isCurrent
                  ? 'rgba(22, 119, 255, 0.08)'
                  : 'transparent',
                color: isCurrent
                  ? '#1677ff'
                  : errCount > 0
                    ? '#ff4d4f'
                    : 'rgba(0, 0, 0, 0.75)',
                fontWeight: isCurrent ? 600 : 400,
                transition: 'all 0.2s ease',
                border: 'none',
                width: '100%',
                textAlign: 'left',
                outline: 'none',
              }}
            >
              <span
                style={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  maxWidth: 90,
                  display: 'inline-flex',
                  alignItems: 'center',
                }}
              >
                {item.icon && (
                  <span
                    style={{
                      marginRight: 4,
                      display: 'inline-flex',
                      alignItems: 'center',
                    }}
                  >
                    {item.icon}
                  </span>
                )}
                {item.title}
              </span>
              {errCount > 0 ? (
                <Badge
                  count={errCount}
                  size="small"
                  style={{ backgroundColor: '#ff4d4f' }}
                />
              ) : isCurrent ? (
                <span
                  style={{
                    width: 4,
                    height: 4,
                    borderRadius: '50%',
                    backgroundColor: '#1677ff',
                  }}
                />
              ) : null}
            </button>
          );
        })}
      </nav>
    </div>
  );
};
