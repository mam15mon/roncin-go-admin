import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudDownloadOutlined,
  CloudSyncOutlined,
  CopyOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormCheckbox,
  ProFormDigit,
  ProFormRadio,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Badge,
  Button,
  Card,
  Col,
  Form,
  Input,
  Popconfirm,
  Select,
  Space,
  Statistic,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { formatDate } from '@/utils/format';
import { useColumnSettings } from '../column-settings';
import type {
  BaseMasterDataItem,
  MasterDataStatItem,
  MasterDataTemplateProps,
} from './types';

const { Text } = Typography;

export function MasterDataTemplate<
  T extends BaseMasterDataItem = BaseMasterDataItem,
  TFormValues = Record<string, unknown>,
>({
  title,
  columnSettingsKey,
  subtitle,
  icon,
  codeLabel = '代码',
  items,
  loading = false,
  total,
  activeTotal,
  disabledTotal,
  query,
  onQueryChange,
  onRefresh,
  searchPlaceholder = '输入代码或名称搜索...',
  filterOptions = [],
  formFields,
  extraColumns = [],
  onCreate,
  onUpdate,
  onToggleActive,
  onSync,
  onExport,
  customStats,
  extraStats = [],
  showStats = true,
  nameWidth,
  renderCode,
  renderStatus,
  notice,
  canEditRecord,
  showUpdatedAt = true,
  style,
  className,
  request,
  actionRef: externalActionRef,
}: MasterDataTemplateProps<T, TFormValues>) {
  const { message } = App.useApp();
  const isRequestMode = typeof request === 'function';
  const serverMode =
    !isRequestMode && query !== undefined && onQueryChange !== undefined;
  const internalActionRef = useRef<ActionType | undefined>(undefined);
  const actionRef = externalActionRef || internalActionRef;

  // Search & Filter state
  const [search, setSearch] = useState('');
  const [filterValues, setFilterValues] = useState<Record<string, unknown>>({});
  const [activeFilter, setActiveFilter] = useState<'all' | 'true' | 'false'>(
    'all',
  );
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [requestTotal, setRequestTotal] = useState(0);

  useEffect(() => {
    if (!serverMode || search === (query.keyword ?? '')) return;
    const timer = window.setTimeout(() => {
      onQueryChange({ ...query, page: 1, keyword: search || undefined });
    }, 300);
    return () => window.clearTimeout(timer);
  }, [onQueryChange, query, search, serverMode]);

  // Request mode search trigger
  useEffect(() => {
    if (!isRequestMode) return;
    const timer = window.setTimeout(() => {
      actionRef.current?.reload();
    }, 300);
    return () => window.clearTimeout(timer);
  }, [actionRef, isRequestMode, search, activeFilter]);

  // Syncing state
  const [syncing, setSyncing] = useState(false);

  // Dialog state
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<T | null>(null);
  const [form] = Form.useForm();

  // Filter items in clientMode
  const filteredItems = useMemo(() => {
    if (isRequestMode || serverMode || !items) return items ?? [];
    return items.filter((item) => {
      // 1. Keyword search (code, name, nameEn)
      if (search.trim()) {
        const q = search.trim().toLowerCase();
        const matchCode = item.code?.toLowerCase().includes(q);
        const matchName = item.name?.toLowerCase().includes(q);
        const matchNameEn = (item.nameEn || '').toLowerCase().includes(q);
        if (!matchCode && !matchName && !matchNameEn) {
          let matchAny = false;
          for (const key of Object.keys(item)) {
            if (
              typeof item[key] === 'string' &&
              item[key].toLowerCase().includes(q)
            ) {
              matchAny = true;
              break;
            }
          }
          if (!matchAny) return false;
        }
      }

      // 2. Active status filter
      if (activeFilter === 'true' && !item.enabled) return false;
      if (activeFilter === 'false' && item.enabled) return false;

      // 3. Custom dynamic filters
      for (const [key, val] of Object.entries(filterValues)) {
        if (val !== undefined && val !== 'all' && val !== '') {
          if (Array.isArray(item[key])) {
            if (!item[key].includes(val)) return false;
          } else if (item[key] !== val) {
            return false;
          }
        }
      }

      return true;
    });
  }, [items, search, activeFilter, filterValues, serverMode]);

  // Client pagination
  const displayDataSource = useMemo(() => {
    if (serverMode) return filteredItems;
    const start = (page - 1) * pageSize;
    return filteredItems.slice(start, start + pageSize);
  }, [serverMode, filteredItems, page, pageSize]);

  // Totals for statistics
  const totalCount =
    total !== undefined
      ? total
      : isRequestMode
        ? requestTotal
        : filteredItems.length;
  const activeCount =
    activeTotal !== undefined
      ? activeTotal
      : filteredItems.filter((i) => i.enabled).length;
  const disabledCount =
    disabledTotal !== undefined
      ? disabledTotal
      : filteredItems.filter((i) => !i.enabled).length;
  const filteredTotal = isRequestMode
    ? (total ?? requestTotal)
    : serverMode
      ? (total ?? 0)
      : filteredItems.length;
  const currentSearch = search;
  const currentActiveFilter = serverMode
    ? query.enabled === undefined
      ? 'all'
      : String(query.enabled)
    : activeFilter;
  const currentPage = serverMode ? query.page : page;
  const currentPageSize = serverMode ? query.pageSize : pageSize;

  // Copy code to clipboard
  const handleCopyCode = async (code: string) => {
    try {
      await navigator.clipboard.writeText(code);
      message.success(`已复制：${code}`);
    } catch {
      message.error('复制失败，请手动选择复制');
    }
  };

  // Open Create Dialog
  const handleOpenCreate = () => {
    setEditingItem(null);
    form.resetFields();
    const initialVals: Record<string, unknown> = {};
    for (const f of formFields) {
      if (f.initialValue !== undefined) {
        initialVals[f.name] = f.initialValue;
      }
    }
    form.setFieldsValue(initialVals);
    setModalOpen(true);
  };

  // Open Edit Dialog
  const handleOpenEdit = (record: T) => {
    setEditingItem(record);
    form.resetFields();
    form.setFieldsValue({ ...record });
    setModalOpen(true);
  };

  // Handle Form Submit
  const handleFormFinish = async (values: TFormValues) => {
    try {
      if (editingItem && onUpdate) {
        await onUpdate(editingItem.id, values);
        message.success('更新成功');
      } else if (onCreate) {
        await onCreate(values);
        message.success('创建成功');
      }
      setModalOpen(false);
      if (isRequestMode) {
        actionRef.current?.reload();
      }
      if (onRefresh) await onRefresh();
    } catch (err) {
      message.error((err as { message?: string }).message || '操作失败');
    }
  };

  // Handle Sync
  const handleSyncTrigger = async () => {
    if (!onSync) return;
    setSyncing(true);
    try {
      await onSync();
      actionRef.current?.reload();
    } finally {
      setSyncing(false);
    }
  };

  const handleRefresh = async () => {
    if (!onRefresh) return;
    try {
      await onRefresh();
      actionRef.current?.reload();
    } catch (err) {
      message.error((err as { message?: string })?.message || '刷新失败');
    }
  };

  const handleReset = () => {
    setSearch('');
    if (serverMode) {
      onQueryChange({ page: 1, pageSize: query.pageSize });
    } else {
      setActiveFilter('all');
      setFilterValues({});
      setPage(1);
      void onRefresh?.();
    }
  };

  const handleToggleActive = async (record: T) => {
    if (!onToggleActive) return;
    try {
      await onToggleActive(record);
      if (isRequestMode) {
        actionRef.current?.reload();
      }
    } catch (err) {
      message.error((err as { message?: string })?.message || '状态切换失败');
    }
  };

  // Build Columns
  const proColumns: ProColumns<T>[] = useMemo(
    () => [
      {
        title: codeLabel,
        dataIndex: 'code',
        key: 'code',
        width: 140,
        render: (_, record) => {
          const defaultDom = (
            <Space size={4}>
              <Tag
                style={{
                  fontFamily: 'monospace',
                  fontWeight: 600,
                  color: '#1677ff',
                  backgroundColor: '#e6f4ff',
                  borderColor: '#91caff',
                  margin: 0,
                  fontSize: 12,
                  padding: '1px 6px',
                }}
              >
                {record.code}
              </Tag>
              <Tooltip title={`复制${codeLabel}`}>
                <Button
                  type="text"
                  size="small"
                  icon={
                    <CopyOutlined style={{ fontSize: 11, color: '#8c8c8c' }} />
                  }
                  onClick={() => handleCopyCode(record.code)}
                  style={{ width: 20, height: 20, padding: 0 }}
                />
              </Tooltip>
            </Space>
          );
          return renderCode ? renderCode(record, defaultDom) : defaultDom;
        },
      },
      {
        title: '名称 (中/英文)',
        dataIndex: 'name',
        key: 'name',
        width: nameWidth ?? (extraColumns.length > 0 ? 240 : undefined),
        render: (_, record) => {
          const primaryName = record.name || record.nameEn || '-';
          const secondaryName =
            record.name && record.nameEn ? record.nameEn : '';
          return (
            <div>
              <div style={{ fontWeight: 600, fontSize: 13, color: '#262626' }}>
                {primaryName}
              </div>
              {secondaryName && (
                <div
                  style={{
                    fontSize: 11,
                    color: '#8c8c8c',
                    fontFamily: 'sans-serif',
                  }}
                >
                  {secondaryName}
                </div>
              )}
            </div>
          );
        },
      },
      ...(extraColumns as ProColumns<T>[]),
      {
        title: '状态',
        dataIndex: 'enabled',
        key: 'enabled',
        width: 100,
        render: (_, record) => {
          const defaultDom = (
            <Badge
              status={record.enabled ? 'success' : 'default'}
              text={record.enabled ? '启用' : '停用'}
            />
          );
          return renderStatus ? renderStatus(record, defaultDom) : defaultDom;
        },
      },
      ...(showUpdatedAt
        ? [
            {
              title: '更新时间',
              dataIndex: 'updatedAt',
              key: 'updatedAt',
              width: 160,
              render: (_: unknown, record: T) => (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {formatDate(record.updatedAt)}
                </Text>
              ),
            } as ProColumns<T>,
          ]
        : []),
      {
        title: '操作',
        key: 'action',
        width: 130,
        align: 'right',
        fixed: 'right',
        render: (_, record) => {
          // B 型基线行对非系统管理禁用编辑：不渲染编辑与停用/启用入口。
          const canEditRow = !canEditRecord || canEditRecord(record);
          return (
            <Space size={6}>
              {onUpdate && canEditRow && (
                <Button
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  style={{ padding: 0 }}
                  onClick={() => handleOpenEdit(record)}
                >
                  编辑
                </Button>
              )}
              {onToggleActive && canEditRow && (
                <Popconfirm
                  title={`确定要${record.enabled ? '停用' : '启用'}【${record.name || record.code}】吗？`}
                  onConfirm={() => handleToggleActive(record)}
                  okText="确定"
                  cancelText="取消"
                >
                  <Button
                    type="link"
                    size="small"
                    danger={record.enabled}
                    style={{ padding: 0 }}
                  >
                    {record.enabled ? '停用' : '启用'}
                  </Button>
                </Popconfirm>
              )}
            </Space>
          );
        },
      },
    ],
    [
      codeLabel,
      extraColumns,
      onUpdate,
      onToggleActive,
      canEditRecord,
      showUpdatedAt,
      nameWidth,
      renderCode,
      renderStatus,
    ],
  );

  // 统一列设置：表格标识默认由页面标题派生（业务视图级，不含实体 ID）。
  const columnSettings = useColumnSettings<ProColumns<T>>({
    tableKey: columnSettingsKey ?? `master-data:${title}`,
    columns: proColumns,
    structuralKeys: ['action'],
  });

  // Unified Statistics Items (Custom or Base + Extra)
  const allStats: MasterDataStatItem[] = useMemo(() => {
    if (customStats && customStats.length > 0) {
      return customStats;
    }
    const list: MasterDataStatItem[] = [
      {
        label: `全部${title.replace(/管理|维护/g, '')}`,
        value: totalCount,
        color: '#262626',
      },
      {
        label: '启用中',
        value: activeCount,
        color: '#52c41a',
        prefix: <CheckCircleOutlined style={{ fontSize: 14 }} />,
      },
      {
        label: '已停用',
        value: disabledCount,
        color: '#ff4d4f',
        prefix: <CloseCircleOutlined style={{ fontSize: 14 }} />,
      },
    ];
    if (extraStats && extraStats.length > 0) {
      list.push(...extraStats);
    }
    return list;
  }, [customStats, title, totalCount, activeCount, disabledCount, extraStats]);

  return (
    <div style={{ minHeight: '100%' }}>
      {/* 0. 组织治理提示横幅：A 型非系统管理只读提示 / B 型非系统管理基线+本地说明 */}
      {notice && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          title={notice}
        />
      )}
      {/* 1. Stats Grid: 现代 CSS Grid 消除 Row gutter 负外边距外凸，确保与 Tab 卡片和表格卡片 100% 垂直平齐 */}
      {showStats && allStats.length > 0 && (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: `repeat(auto-fit, minmax(160px, 1fr))`,
            gap: 10,
            marginBottom: 12,
            width: '100%',
          }}
        >
          {allStats.map((stat, idx) => (
            <Card
              key={stat.label || idx}
              size="small"
              variant="borderless"
              style={{
                borderRadius: 8,
                backgroundColor: '#ffffff',
                border: '1px solid #f0f0f0',
                boxShadow: '0 1px 2px rgba(0, 0, 0, 0.02)',
                height: '100%',
              }}
              styles={{ body: { padding: '8px 12px' } }}
            >
              <Statistic
                title={
                  <span style={{ fontSize: 12, color: '#8c8c8c' }}>
                    {stat.label}
                  </span>
                }
                value={stat.value}
                styles={{
                  content: {
                    fontSize: 18,
                    fontWeight: 600,
                    color: stat.color || '#262626',
                  },
                }}
                prefix={stat.prefix}
              />
            </Card>
          ))}
        </div>
      )}

      {/* 2. Unified ProTable Card: Integrated Filters, Actions, and High-Density Table */}
      <Card
        variant="borderless"
        className={`roncin-master-data-template ${className || ''}`}
        style={{
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          backgroundColor: '#ffffff',
          boxShadow: '0 1px 2px rgba(0, 0, 0, 0.03)',
          ...style,
        }}
        styles={{ body: { padding: '12px 16px' } }}
      >
        <ProTable<T>
          actionRef={actionRef}
          rowKey="id"
          columns={columnSettings.columns}
          dataSource={isRequestMode ? undefined : displayDataSource}
          request={
            isRequestMode
              ? async (params, sort, filter) => {
                  const res = await request({
                    ...params,
                    keyword: search || undefined,
                    enabled:
                      activeFilter === 'all'
                        ? undefined
                        : activeFilter === 'true',
                    ...sort,
                    ...filter,
                  });
                  if (res.total !== undefined) {
                    setRequestTotal(res.total);
                  }
                  return res;
                }
              : undefined
          }
          loading={loading}
          cardProps={false}
          tableAlertRender={false}
          tableAlertOptionRender={false}
          search={false}
          scroll={{ x: 'max-content' }}
          headerTitle={
            <Space wrap size={8} align="center">
              {icon && (
                <span
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 6,
                    backgroundColor: '#e6f4ff',
                    color: '#1677ff',
                    display: 'inline-flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 15,
                    flexShrink: 0,
                  }}
                >
                  {icon}
                </span>
              )}
              <Tooltip title={subtitle}>
                <Text
                  strong
                  style={{
                    fontSize: 15,
                    color: '#1f1f1f',
                    marginRight: 4,
                    cursor: subtitle ? 'help' : 'default',
                  }}
                >
                  {title.replace(/管理|维护/g, '')}
                </Text>
              </Tooltip>
              <Input
                placeholder={searchPlaceholder}
                prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
                value={currentSearch}
                onChange={(e) => {
                  if (serverMode) {
                    setSearch(e.target.value);
                  } else {
                    setSearch(e.target.value);
                    setPage(1);
                  }
                }}
                style={{ width: 220 }}
                allowClear
              />
              <Select
                value={currentActiveFilter}
                onChange={(val) => {
                  if (serverMode) {
                    onQueryChange({
                      ...query,
                      page: 1,
                      enabled: val === 'all' ? undefined : val === 'true',
                    });
                  } else {
                    setActiveFilter(val as 'all' | 'true' | 'false');
                    setPage(1);
                  }
                }}
                options={[
                  { label: '全部状态', value: 'all' },
                  { label: '仅启用', value: 'true' },
                  { label: '仅停用', value: 'false' },
                ]}
                style={{ width: 100 }}
              />
              {!serverMode &&
                filterOptions.map((opt) => (
                  <Select
                    key={opt.key}
                    placeholder={opt.placeholder || opt.label}
                    value={filterValues[opt.key] ?? 'all'}
                    onChange={(val) => {
                      setFilterValues({ ...filterValues, [opt.key]: val });
                      setPage(1);
                    }}
                    options={opt.options}
                    style={{ width: opt.width || 120 }}
                    allowClear
                  />
                ))}
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
            </Space>
          }
          toolBarRender={() =>
            [
              <Tag
                key="total"
                color="blue"
                style={{
                  borderRadius: 10,
                  margin: 0,
                  padding: '2px 8px',
                  fontSize: 12,
                  fontWeight: 500,
                }}
              >
                共 {filteredTotal} 条
              </Tag>,
              columnSettings.entry,
              onSync && (
                <Button
                  key="sync"
                  icon={<CloudSyncOutlined />}
                  loading={syncing}
                  onClick={handleSyncTrigger}
                >
                  同步官方数据
                </Button>
              ),
              onExport && (
                <Button
                  key="export"
                  icon={<CloudDownloadOutlined />}
                  onClick={onExport}
                >
                  导出数据
                </Button>
              ),
              onCreate && (
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={handleOpenCreate}
                  style={{ fontWeight: 500 }}
                >
                  新增{title.replace(/管理|维护/g, '')}
                </Button>
              ),
            ].filter(Boolean)
          }
          options={{
            reload: () => {
              if (isRequestMode) {
                actionRef.current?.reload();
              } else if (onRefresh) {
                handleRefresh();
              }
            },
            density: true,
            fullScreen: true,
            setting: false,
          }}
          pagination={
            isRequestMode
              ? {
                  defaultPageSize: 10,
                  showSizeChanger: true,
                  pageSizeOptions: ['10', '20', '50', '100'],
                  showTotal: (t) => `共 ${t} 条`,
                }
              : {
                  current: currentPage,
                  pageSize: currentPageSize,
                  total: filteredTotal,
                  showSizeChanger: true,
                  pageSizeOptions: ['10', '20', '50', '100'],
                  showTotal: (t) => `共 ${t} 条`,
                  onChange: (p, ps) => {
                    if (serverMode) {
                      onQueryChange({ ...query, page: p, pageSize: ps });
                    } else {
                      setPage(p);
                      setPageSize(ps);
                    }
                  },
                }
                }
        />
        {columnSettings.modal}
      </Card>

      {/* 3. Dynamic Create / Edit Modal Form */}
      <ModalForm
        title={
          editingItem
            ? `编辑${title.replace(/管理|维护/g, '')} - ${editingItem.code}`
            : `新增${title.replace(/管理|维护/g, '')}`
        }
        open={modalOpen}
        form={form}
        onOpenChange={setModalOpen}
        onFinish={handleFormFinish}
        modalProps={{
          destroyOnHidden: true,
          mask: { closable: false },
          width: 520,
        }}
        layout="vertical"
        grid
      >
        {formFields.map((field) => {
          const disabled = editingItem ? field.disabledOnEdit : false;
          const span = field.span || 24;

          if (field.type === 'select') {
            return (
              <Col span={span} key={field.name}>
                <ProFormSelect
                  name={field.name}
                  label={field.label}
                  options={field.options}
                  placeholder={field.placeholder || `请选择${field.label}`}
                  rules={
                    field.rules ||
                    (field.required
                      ? [{ required: true, message: `请选择${field.label}` }]
                      : undefined)
                  }
                  disabled={disabled}
                  extra={field.extra}
                />
              </Col>
            );
          }

          if (field.type === 'number') {
            return (
              <Col span={span} key={field.name}>
                <ProFormDigit
                  name={field.name}
                  label={field.label}
                  placeholder={field.placeholder || `请输入${field.label}`}
                  rules={
                    field.rules ||
                    (field.required
                      ? [{ required: true, message: `请输入${field.label}` }]
                      : undefined)
                  }
                  disabled={disabled}
                  extra={field.extra}
                />
              </Col>
            );
          }

          if (field.type === 'textarea') {
            return (
              <Col span={span} key={field.name}>
                <ProFormTextArea
                  name={field.name}
                  label={field.label}
                  placeholder={field.placeholder || `请输入${field.label}`}
                  rules={
                    field.rules ||
                    (field.required
                      ? [{ required: true, message: `请输入${field.label}` }]
                      : undefined)
                  }
                  fieldProps={{ rows: 3 }}
                  disabled={disabled}
                  extra={field.extra}
                />
              </Col>
            );
          }

          if (field.type === 'checkboxGroup') {
            return (
              <Col span={span} key={field.name}>
                <ProFormCheckbox.Group
                  name={field.name}
                  label={field.label}
                  options={field.options}
                  rules={
                    field.rules ||
                    (field.required
                      ? [{ required: true, message: `请选择${field.label}` }]
                      : undefined)
                  }
                  disabled={disabled}
                  extra={field.extra}
                />
              </Col>
            );
          }

          if (field.type === 'radio') {
            return (
              <Col span={span} key={field.name}>
                <ProFormRadio.Group
                  name={field.name}
                  label={field.label}
                  options={field.options}
                  rules={
                    field.rules ||
                    (field.required
                      ? [{ required: true, message: `请选择${field.label}` }]
                      : undefined)
                  }
                  disabled={disabled}
                  extra={field.extra}
                />
              </Col>
            );
          }

          // Default: Text input
          return (
            <Col span={span} key={field.name}>
              <ProFormText
                name={field.name}
                label={field.label}
                placeholder={field.placeholder || `请输入${field.label}`}
                rules={
                  field.rules ||
                  (field.required
                    ? [{ required: true, message: `请输入${field.label}` }]
                    : undefined)
                }
                disabled={disabled}
                extra={field.extra}
              />
            </Col>
          );
        })}
      </ModalForm>
    </div>
  );
}
