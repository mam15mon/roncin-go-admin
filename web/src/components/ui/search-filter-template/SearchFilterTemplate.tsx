import {
  DownOutlined,
  ReloadOutlined,
  SearchOutlined,
  UpOutlined,
} from '@ant-design/icons';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Row,
  Space,
} from 'antd';
import React, { useMemo, useState } from 'react';
import { SearchableSelect } from '../searchable-select';
import type {
  SearchFilterFieldItem,
  SearchFilterTemplateProps,
} from './types';

const { RangePicker } = DatePicker;

/**
 * 统一搜索区域模板 SearchFilterTemplate
 * 遵循 Roncin 纯白高密度企业级视觉规范，提供三种模式：
 * 1. 'grid'：多字段栅格配置表单（支持折叠/展开、智能模糊下拉）
 * 2. 'bar'：紧凑单行快捷筛选栏（关键字输入 + 快捷下拉 + 按钮组）
 * 3. 'custom'：自由 JSX 渲染插槽
 */
export function SearchFilterTemplate<TValues extends Record<string, any> = Record<string, any>>({
  layout = 'grid',
  collapsible = true,
  defaultCollapsed = true,
  defaultVisibleCount = 3,
  items = [],
  keywordPlaceholder = '输入关键字搜索...',
  keywordName = 'keyword',
  quickFilters = [],
  onSearch,
  onReset,
  loading = false,
  style,
  className,
  extraRight,
  children,
  form: externalForm,
}: SearchFilterTemplateProps<TValues>) {
  const [internalForm] = Form.useForm();
  const form = externalForm || internalForm;
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  const toggleCollapse = () => setCollapsed((prev) => !prev);

  // 顶层计算可见字段（确保 Hooks 顶层无条件执行）
  const visibleItems = useMemo(() => {
    if (!collapsible || !collapsed) {
      return items;
    }
    return items.slice(0, defaultVisibleCount);
  }, [items, collapsible, collapsed, defaultVisibleCount]);

  const hasHiddenItems = collapsible && items.length > defaultVisibleCount;

  // 提交处理
  const handleFinish = (values: any) => {
    onSearch?.(values);
  };

  // 重置处理
  const handleReset = () => {
    form.resetFields();
    onReset?.();
  };

  // 渲染单一表单字段
  const renderFieldInput = (item: SearchFilterFieldItem) => {
    const {
      type = 'input',
      placeholder,
      options,
      allowClear = true,
      fieldProps,
    } = item;

    switch (type) {
      case 'select':
      case 'searchable-select':
        return (
          <SearchableSelect
            allowClear={allowClear}
            options={options}
            placeholder={placeholder as string}
            style={{ width: '100%' }}
            {...fieldProps}
          />
        );

      case 'date':
        return (
          <DatePicker
            allowClear={allowClear}
            placeholder={placeholder as string}
            style={{ width: '100%' }}
            {...fieldProps}
          />
        );

      case 'date-range':
        return (
          <RangePicker
            allowClear={allowClear}
            placeholder={placeholder as [string, string]}
            style={{ width: '100%' }}
            {...fieldProps}
          />
        );

      case 'digit':
        return (
          <InputNumber
            placeholder={placeholder as string}
            style={{ width: '100%' }}
            {...fieldProps}
          />
        );

      case 'custom':
        return item.render ? item.render(form) : null;

      default:
        return (
          <Input
            allowClear={allowClear}
            placeholder={placeholder as string}
            {...fieldProps}
          />
        );
    }
  };

  // 1. 快捷单行搜索栏模式 ('bar')
  if (layout === 'bar') {
    return (
      <Card
        bordered={false}
        className={className}
        style={{
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          backgroundColor: '#ffffff',
          marginBottom: 12,
          ...style,
        }}
        styles={{ body: { padding: '12px 16px' } }}
      >
        <Form form={form} layout="inline" onFinish={handleFinish}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              width: '100%',
              flexWrap: 'wrap',
              gap: 12,
            }}
          >
            <Space size={12} wrap style={{ flex: 1 }}>
              {/* 关键字搜索输入框 */}
              <Form.Item name={keywordName} noStyle>
                <Input
                  allowClear
                  prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder={keywordPlaceholder}
                  style={{ width: 260 }}
                  onPressEnter={() => form.submit()}
                />
              </Form.Item>

              {/* 快捷下拉筛选 */}
              {quickFilters.map((qf) => (
                <Form.Item key={qf.name} name={qf.name} noStyle initialValue={qf.initialValue}>
                  <SearchableSelect
                    allowClear
                    placeholder={qf.placeholder || '全部'}
                    options={qf.options}
                    style={{ width: qf.width || 140 }}
                    showSearch={qf.showSearch ?? true}
                    onChange={() => form.submit()}
                  />
                </Form.Item>
              ))}

              {/* 搜索与重置按钮 */}
              <Button
                type="primary"
                icon={<SearchOutlined />}
                loading={loading}
                onClick={() => form.submit()}
              >
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
            </Space>

            {/* 右侧扩展操作插槽 */}
            {extraRight && <Space size={8}>{extraRight}</Space>}
          </div>
        </Form>
      </Card>
    );
  }

  // 2. 自定义插槽模式 ('custom')
  if (layout === 'custom') {
    return (
      <Card
        bordered={false}
        className={className}
        style={{
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          backgroundColor: '#ffffff',
          marginBottom: 12,
          ...style,
        }}
        styles={{ body: { padding: '14px 16px 8px' } }}
      >
        <Form form={form} layout="vertical" onFinish={handleFinish}>
          {typeof children === 'function'
            ? children({ form, collapsed, toggleCollapse })
            : children}
        </Form>
      </Card>
    );
  }

  // 3. 多字段配置化网格表单模式 ('grid')
  return (
    <Card
      bordered={false}
      className={className}
      style={{
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        backgroundColor: '#ffffff',
        marginBottom: 12,
        ...style,
      }}
      styles={{ body: { padding: '14px 16px 8px' } }}
    >
      <Form form={form} layout="vertical" onFinish={handleFinish}>
        <Row gutter={[16, 0]}>
          {visibleItems.map((item) => (
            <Col key={item.name} span={item.span || 6}>
              <Form.Item
                name={item.name}
                label={item.label}
                initialValue={item.initialValue}
                style={{ marginBottom: 12 }}
              >
                {renderFieldInput(item)}
              </Form.Item>
            </Col>
          ))}

          {/* 操作按钮区 */}
          <Col
            span={24}
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 8,
            }}
          >
            <div>{extraRight}</div>
            <Space size={8}>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                loading={loading}
                htmlType="submit"
              >
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
              {hasHiddenItems && (
                <Button
                  type="link"
                  onClick={toggleCollapse}
                  style={{ padding: '0 4px', fontSize: 13 }}
                >
                  {collapsed ? (
                    <>
                      展开 <DownOutlined />
                    </>
                  ) : (
                    <>
                      收起 <UpOutlined />
                    </>
                  )}
                </Button>
              )}
            </Space>
          </Col>
        </Row>
      </Form>
    </Card>
  );
}

export default SearchFilterTemplate;
