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
  Tag,
} from 'antd';
import React, { useMemo, useRef, useState } from 'react';
import { formatDate } from '@/utils/format';
import { standardDateRangePresets } from '../date-presets';
import { RemoteSearchSelect, SearchableSelect } from '../searchable-select';
import './SearchFilterTemplate.less';
import type { SearchFilterFieldItem, SearchFilterTemplateProps } from './types';

const { RangePicker } = DatePicker;

/** 判断筛选值是否为空：undefined / null / 空字符串（含 trim 后空串）/ 空数组不产生 chip */
const isEmptyFilterValue = (value: unknown): boolean =>
  value === undefined ||
  value === null ||
  (typeof value === 'string' && value.trim() === '') ||
  (Array.isArray(value) && value.length === 0);

/** 从候选项中提取展示 label（option 可为单对象或数组，兼容多选） */
const extractOptionLabel = (option: unknown): string | undefined => {
  if (Array.isArray(option)) {
    const labels = option
      .map((opt) => extractOptionLabel(opt))
      .filter((label): label is string => label !== undefined);
    return labels.length > 0 ? labels.join('、') : undefined;
  }
  if (option && typeof option === 'object' && 'label' in option) {
    const { label } = option as { label?: unknown };
    if (label !== undefined && label !== null && label !== '') {
      return String(label);
    }
  }
  return undefined;
};

/** 从静态候选项中按 value 匹配展示 label */
const findOptionLabel = (
  options: SearchFilterFieldItem['options'],
  value: unknown,
): string | undefined => {
  if (!options) {
    return undefined;
  }
  const matched = options.find((opt) => opt.value === value);
  return matched ? extractOptionLabel(matched) : undefined;
};

/** 按字段类型解析 chip 展示值，空值返回 null（不产生 chip） */
const resolveChipText = (
  item: SearchFilterFieldItem,
  value: unknown,
  labelMap: Record<string, string>,
): string | null => {
  if (isEmptyFilterValue(value)) {
    return null;
  }
  switch (item.type) {
    case 'select':
    case 'searchable-select':
      // 静态候选项直接解析 label；远程字段用选择时捕获的 label，均未命中回退原值
      return (
        findOptionLabel(item.options, value) ??
        labelMap[item.name] ??
        String(value)
      );
    case 'date':
      // DatePicker 表单值为 Dayjs 对象，formatDate 内部经 dayjs 包装可解析
      return formatDate(value as string, 'date');
    case 'date-range': {
      if (!Array.isArray(value)) {
        return null;
      }
      const [start, end] = value as [unknown, unknown];
      if (isEmptyFilterValue(start) && isEmptyFilterValue(end)) {
        return null;
      }
      return `${formatDate(start as string, 'date')} ~ ${formatDate(end as string, 'date')}`;
    }
    case 'custom':
      // 自定义插槽无法推断展示值，不产生 chip
      return null;
    default:
      // input / digit 展示原始值
      return String(value).trim() || null;
  }
};

/**
 * 统一搜索区域模板 SearchFilterTemplate
 * 遵循 Roncin 纯白高密度企业级视觉规范，提供三种模式：
 * 1. 'grid'：多字段栅格配置表单（支持折叠/展开、智能模糊下拉）
 * 2. 'bar'：紧凑单行快捷筛选栏（关键字输入 + 快捷下拉 + 按钮组）
 * 3. 'custom'：自由 JSX 渲染插槽
 */
export function SearchFilterTemplate<
  TValues extends Record<string, unknown> = Record<string, unknown>,
>({
  layout = 'grid',
  formLayout = 'horizontal',
  labelWidth = 80,
  colSpan = 4,
  collapsible = true,
  defaultCollapsed = true,
  defaultVisibleCount = 5,
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
  // 已提交筛选条件：chips 回显是它的镜像，仅随提交/重置/删除 chip 变更
  const [committed, setCommitted] = useState<TValues>({} as TValues);
  // 与 committed 同批固化的下拉 label 快照：chips 只反映已提交口径，改选未重查不影响回显
  const [labelSnapshot, setLabelSnapshot] = useState<Record<string, string>>(
    {},
  );
  // 选择动作发生时捕获的下拉候选项 label（远程字段无法从配置解析），提交时固化进快照
  const labelMapRef = useRef<Record<string, string>>({});

  const toggleCollapse = () => setCollapsed((prev) => !prev);

  // 顶层计算可见字段（确保 Hooks 顶层无条件执行）
  const visibleItems = useMemo(() => {
    if (!collapsible || !collapsed) {
      return items;
    }
    return items.slice(0, defaultVisibleCount);
  }, [items, collapsible, collapsed, defaultVisibleCount]);

  const hasHiddenItems = collapsible && items.length > defaultVisibleCount;

  // 计算栅格使用量与操作按钮跨度
  const usedSpan = useMemo(() => {
    return (
      visibleItems.reduce((acc, it) => acc + (it.span || colSpan || 4), 0) % 24
    );
  }, [visibleItems, colSpan]);

  const actionSpan = useMemo(() => {
    if (usedSpan === 0) return 24;
    const remaining = 24 - usedSpan;
    // 若剩余栅格不足以容纳操作按钮（有 extraRight 时至少需要 8 栅格，仅查询重置时至少需要 4 栅格），换行独立占满一行（span=24）
    const minRequiredSpan = extraRight ? 8 : 4;
    if (remaining < minRequiredSpan) {
      return 24;
    }
    return remaining;
  }, [usedSpan, extraRight]);

  // 提交处理：固化已提交条件与当时的 label 快照供 chips 回显
  const handleFinish = (values: TValues) => {
    setCommitted(values);
    setLabelSnapshot({ ...labelMapRef.current });
    onSearch?.(values);
  };

  // 重置处理：清空表单、chips 与捕获的 label
  const handleReset = () => {
    form.resetFields();
    labelMapRef.current = {};
    setCommitted({} as TValues);
    setLabelSnapshot({});
    onReset?.();
  };

  // 删除单个 chip：清空对应表单字段，并以剔除该字段后的完整条件触发重新搜索
  const deleteChip = (name: string) => {
    const next = { ...committed } as TValues;
    delete next[name];
    setCommitted(next);
    form.setFieldValue(name, undefined);
    delete labelMapRef.current[name];
    setLabelSnapshot((prev) => {
      const nextLabels = { ...prev };
      delete nextLabels[name];
      return nextLabels;
    });
    onSearch?.(next);
  };

  // 已提交条件的 chips 数据：bar 模式统计关键字 + 快捷筛选，grid 模式统计全部 items；
  // custom 插槽无法推断字段配置，不产生 chip
  const chips = useMemo(() => {
    if (layout === 'custom') {
      return [];
    }
    const chipItems: SearchFilterFieldItem[] =
      layout === 'bar'
        ? [
            // bar 模式关键字输入框无 label 配置，chip 统一展示中文标签
            { name: keywordName, label: '关键字', type: 'input' },
            ...quickFilters.map((qf) => ({
              name: qf.name,
              label: qf.placeholder,
              type: 'select' as const,
              options: qf.options,
            })),
          ]
        : items;
    const result: { name: string; label: React.ReactNode; text: string }[] = [];
    for (const item of chipItems) {
      const text = resolveChipText(
        item,
        (committed as Record<string, unknown>)[item.name],
        labelSnapshot,
      );
      if (text !== null) {
        result.push({ name: item.name, label: item.label ?? item.name, text });
      }
    }
    return result;
  }, [layout, items, quickFilters, keywordName, committed, labelSnapshot]);

  // chips 回显行：grid 渲染在字段栅格上方，bar 渲染在表单行下方
  const chipsRow =
    chips.length > 0 ? (
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 8,
          marginTop: layout === 'bar' ? 10 : undefined,
          marginBottom: layout === 'grid' ? 10 : undefined,
        }}
      >
        {chips.map((chip) => (
          <Tag
            key={chip.name}
            closable
            onClose={() => deleteChip(chip.name)}
            style={{ fontSize: 13, marginInlineEnd: 0 }}
          >
            {chip.label}: {chip.text}
          </Tag>
        ))}
        <Button
          type="link"
          onClick={handleReset}
          style={{ padding: '0 4px', fontSize: 13 }}
        >
          清除全部
        </Button>
      </div>
    ) : null;

  // 捕获所选候选项 label 供 chips 回显（清除选择时同步移除）
  const captureSelectLabel = (name: string, option: unknown) => {
    const label = extractOptionLabel(option);
    if (label === undefined) {
      delete labelMapRef.current[name];
    } else {
      labelMapRef.current[name] = label;
    }
  };

  // 渲染单一表单字段
  const renderFieldInput = (item: SearchFilterFieldItem) => {
    const {
      type = 'input',
      placeholder,
      options,
      request,
      allowClear = true,
      fieldProps,
    } = item;

    switch (type) {
      case 'select':
      case 'searchable-select': {
        // 合并拦截 onChange：捕获所选 label，同时保留消费方传入的 onChange
        const mergedOnChange = (value: unknown, option: unknown) => {
          captureSelectLabel(item.name, option);
          fieldProps?.onChange?.(value, option);
        };
        // 配置了 request 的字段走远程搜索下拉：候选项按关键字服务端过滤
        if (request) {
          return (
            <RemoteSearchSelect
              allowClear={allowClear}
              placeholder={placeholder as string}
              style={{ width: '100%' }}
              request={(keyWords) => request({ keyWords })}
              {...fieldProps}
              onChange={mergedOnChange}
            />
          );
        }
        return (
          <SearchableSelect
            allowClear={allowClear}
            options={options}
            placeholder={placeholder as string}
            style={{ width: '100%' }}
            {...fieldProps}
            onChange={mergedOnChange}
          />
        );
      }

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
            presets={standardDateRangePresets}
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
            style={{ width: '100%' }}
            {...fieldProps}
          />
        );
    }
  };

  // 1. 快捷单行搜索栏模式 ('bar')
  if (layout === 'bar') {
    return (
      <Card
        variant="borderless"
        className={className}
        style={{
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          backgroundColor: '#ffffff',
          marginBottom: 'var(--roncin-page-section-gap)',
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
                <Form.Item
                  key={qf.name}
                  name={qf.name}
                  noStyle
                  initialValue={qf.initialValue}
                >
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

          {/* 已选条件 chips 回显行 */}
          {chipsRow}
        </Form>
      </Card>
    );
  }

  // 2. 自定义插槽模式 ('custom')
  if (layout === 'custom') {
    return (
      <Card
        variant="borderless"
        className={className}
        style={{
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          backgroundColor: '#ffffff',
          marginBottom: 'var(--roncin-page-section-gap)',
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
      variant="borderless"
      className={className}
      style={{
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        backgroundColor: '#ffffff',
        marginBottom: 'var(--roncin-page-section-gap)',
        ...style,
      }}
      styles={{ body: { padding: '14px 16px 6px' } }}
    >
      <Form
        form={form}
        layout={formLayout}
        onFinish={handleFinish}
        className={
          formLayout === 'horizontal'
            ? 'roncin-search-filter-grid-horizontal'
            : undefined
        }
      >
        {/* 已选条件 chips 回显行 */}
        {chipsRow}
        <Row gutter={[16, 0]}>
          {visibleItems.map((item) => (
            <Col key={item.name} span={item.span || colSpan || 4}>
              <Form.Item
                name={item.name}
                label={item.label}
                labelCol={
                  formLayout === 'horizontal'
                    ? {
                        flex:
                          typeof labelWidth === 'number'
                            ? `0 0 ${labelWidth}px`
                            : labelWidth,
                      }
                    : undefined
                }
                wrapperCol={
                  formLayout === 'horizontal'
                    ? { flex: '1 1 0%', style: { minWidth: 0 } }
                    : undefined
                }
                initialValue={item.initialValue}
                style={{ marginBottom: 10 }}
              >
                {renderFieldInput(item)}
              </Form.Item>
            </Col>
          ))}

          {/* 操作按钮区：紧随行内（自适应填补行内剩余栅格）或占满末行居右 */}
          <Col
            span={actionSpan}
            style={{
              display: 'flex',
              justifyContent:
                actionSpan === 24 && extraRight ? 'space-between' : 'flex-end',
              alignItems: 'center',
              marginBottom: 10,
              minHeight: 32,
            }}
          >
            {actionSpan === 24 && extraRight ? <div>{extraRight}</div> : null}
            <Space size={8}>
              {actionSpan !== 24 && extraRight ? extraRight : null}
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
