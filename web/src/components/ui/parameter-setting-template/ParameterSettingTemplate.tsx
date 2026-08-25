import { PageContainer } from '@ant-design/pro-components';
import { history, useLocation } from '@umijs/max';
import { Space, Tabs, Tooltip, Typography } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import type { ParameterSettingTemplateProps } from './types';

const { Text } = Typography;

export const ParameterSettingTemplate: React.FC<ParameterSettingTemplateProps> = ({
  title = '业务参数与规则设置',
  subTitle = '集中维护单据自动编号、费用科目、汇率时间标准与业务履约里程碑规则',
  extra,
  items,
  activeKey: controlledActiveKey,
  defaultActiveKey,
  onChange,
  syncUrlQuery = true,
  queryParamKey = 'tab',
  style,
  className,
  tabType = 'card',
}) => {
  const location = useLocation();

  // 1. 过滤可见的 Tabs
  const visibleItems = useMemo(() => {
    return items.filter((item) => item.visible !== false);
  }, [items]);

  // 2. 本地非受控激活状态
  const [uncontrolledActiveKey, setUncontrolledActiveKey] = useState<string | undefined>(defaultActiveKey);

  // 3. 从 URL search 中解析当前 tab
  const queryActiveTab = useMemo(() => {
    if (!syncUrlQuery) return undefined;
    const params = new URLSearchParams(location.search);
    return params.get(queryParamKey) || undefined;
  }, [location.search, syncUrlQuery, queryParamKey]);

  // 4. 计算当前应该高亮的 tab key
  const currentActiveKey = useMemo(() => {
    if (controlledActiveKey) return controlledActiveKey;
    if (queryActiveTab && visibleItems.some((item) => item.key === queryActiveTab)) {
      return queryActiveTab;
    }
    if (uncontrolledActiveKey && visibleItems.some((item) => item.key === uncontrolledActiveKey)) {
      return uncontrolledActiveKey;
    }
    if (defaultActiveKey && visibleItems.some((item) => item.key === defaultActiveKey)) {
      return defaultActiveKey;
    }
    return visibleItems[0]?.key;
  }, [controlledActiveKey, queryActiveTab, uncontrolledActiveKey, defaultActiveKey, visibleItems]);

  // 5. 处理 Tab 切换
  const handleTabChange = useCallback(
    (nextKey: string) => {
      setUncontrolledActiveKey(nextKey);
      if (syncUrlQuery) {
        const params = new URLSearchParams(location.search);
        params.set(queryParamKey, nextKey);
        history.replace(`${location.pathname}?${params.toString()}`);
      }
      onChange?.(nextKey);
    },
    [syncUrlQuery, location.search, location.pathname, queryParamKey, onChange],
  );

  // 6. 渲染 Tabs Items
  const tabItems = useMemo(() => {
    return visibleItems.map((item) => {
      const labelNode = (
        <span key={`tab-label-${item.key}`} style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          {item.icon}
          <span>{item.label}</span>
          {item.badge}
        </span>
      );

      return {
        key: item.key,
        disabled: item.disabled,
        label: item.tooltip ? (
          <Tooltip title={item.tooltip}>{labelNode}</Tooltip>
        ) : (
          labelNode
        ),
        children: <div style={{ paddingTop: 4 }}>{item.children}</div>,
      };
    });
  }, [visibleItems]);

  return (
    <PageContainer
      header={{
        title: title ? (
          <Space size={8} align="center">
            <span
              style={{
                width: 3,
                height: 18,
                borderRadius: 2,
                backgroundColor: '#1677ff',
                display: 'inline-block',
                flexShrink: 0,
              }}
            />
            <Text strong style={{ fontSize: 18, color: 'rgba(0, 0, 0, 0.88)' }}>
              {title}
            </Text>
          </Space>
        ) : undefined,
        subTitle: subTitle ? (
          <Text type="secondary" style={{ fontSize: 13 }}>
            {subTitle}
          </Text>
        ) : undefined,
        extra,
      }}
      className={`roncin-parameter-setting-template ${className || ''}`}
      style={{
        minHeight: '100vh',
        backgroundColor: '#f5f7fa',
        ...style,
      }}
    >
      <div style={{ marginTop: 4 }}>
        <Tabs
          type={tabType}
          activeKey={currentActiveKey}
          onChange={handleTabChange}
          items={tabItems}
          tabBarStyle={{
            marginBottom: 16,
            backgroundColor: '#ffffff',
            padding: '8px 12px 0',
            borderRadius: '8px 8px 0 0',
            border: '1px solid #f0f0f0',
            borderBottom: 'none',
          }}
        />
      </div>
    </PageContainer>
  );
};
