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
  App,
  Alert,
  Badge,
  Button,
  Card,
  Col,
  Form,
  Input,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { formatDate } from '@/utils/format';
import type { BaseMasterDataItem, MasterDataTemplateProps } from './types';

const { Text } = Typography;

export function MasterDataTemplate<T extends BaseMasterDataItem = BaseMasterDataItem>({
  title,
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
  extraStats = [],
  showStats = true,
  notice,
  canEditRecord,
  style,
  className,
}: MasterDataTemplateProps<T>) {
  const { message } = App.useApp();
  const serverMode = query !== undefined && onQueryChange !== undefined;
  const actionRef = useRef<ActionType | undefined>(undefined);

  // Search & Filter state
  const [search, setSearch] = useState('');
  const [filterValues, setFilterValues] = useState<Record<string, any>>({});
  const [activeFilter, setActiveFilter] = useState<'all' | 'true' | 'false'>('all');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  useEffect(() => {
    if (!serverMode || search === (query.keyword ?? '')) return;
    const timer = window.setTimeout(() => {
      onQueryChange({ ...query, page: 1, keyword: search || undefined });
    }, 300);
    return () => window.clearTimeout(timer);
  }, [onQueryChange, query, search, serverMode]);

  // Syncing state
  const [syncing, setSyncing] = useState(false);

  // Dialog state
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<T | null>(null);
  const [form] = Form.useForm();

  // Filter items in clientMode
  const filteredItems = useMemo(() => {
    if (serverMode) return items;
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
            if (typeof item[key] === 'string' && item[key].toLowerCase().includes(q)) {
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

  // Current display data source
  const displayDataSource = useMemo(() => {
    if (serverMode) return items;
    const start = (page - 1) * pageSize;
    return filteredItems.slice(start, start + pageSize);
  }, [serverMode, items, filteredItems, page, pageSize]);

  // Stats calculation
  const filteredTotal = serverMode ? (total ?? 0) : filteredItems.length;
  const activeCount = serverMode
    ? (activeTotal ?? 0)
    : items.filter((item) => item.enabled).length;
  const disabledCount = serverMode
    ? (disabledTotal ?? 0)
    : items.length - activeCount;
  const totalCount = serverMode ? activeCount + disabledCount : items.length;
  const currentSearch = search;
  const currentActiveFilter = serverMode
    ? query.enabled === undefined
      ? 'all'
      : String(query.enabled)
    : activeFilter;
  const currentPage = serverMode ? query.page : page;
  const currentPageSize = serverMode ? query.pageSize : pageSize;

  // Copy code to clipboard
  const handleCopyCode = (code: string) => {
    navigator.clipboard?.writeText(code);
    message.success(`已复制 ${codeLabel}: ${code}`);
  };

  // Open Create Dialog
  const handleOpenCreate = () => {
    setEditingItem(null);
    form.resetFields();
    const initialVals: Record<string, any> = {};
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
  const handleFormFinish = async (values: any) => {
    try {
      if (editingItem) {
        if (onUpdate) {
          const res = await onUpdate(editingItem.id, values);
          if (res === false) return false;
        }
        message.success(`${title}已更新`);
      } else {
        if (onCreate) {
          const res = await onCreate(values);
          if (res === false) return false;
        }
        message.success(`${title}已创建`);
      }
      setModalOpen(false);
      actionRef.current?.reload();
      return true;
    } catch (err: any) {
      message.error(err?.message || '操作失败');
      return false;
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
    } catch (err: any) {
      message.error(err?.message || '刷新失败');
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
      actionRef.current?.reload();
    } catch (err: any) {
      message.error(err?.message || '操作失败');
    }
  };

  // Build Columns
  const proColumns: ProColumns<T>[] = useMemo(() => [
    {
      title: codeLabel,
      dataIndex: 'code',
      key: 'code',
      width: 140,
      render: (_, record) => (
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
              icon={<CopyOutlined style={{ fontSize: 11, color: '#8c8c8c' }} />}
              onClick={() => handleCopyCode(record.code)}
              style={{ width: 20, height: 20, padding: 0 }}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '名称 (中/英文)',
      dataIndex: 'name',
      key: 'name',
      render: (_, record) => {
        const primaryName = record.name || record.nameEn || '-';
        const secondaryName = record.name && record.nameEn ? record.nameEn : '';
        return (
          <div>
            <div style={{ fontWeight: 600, fontSize: 13, color: '#262626' }}>
              {primaryName}
            </div>
            {secondaryName && (
              <div style={{ fontSize: 11, color: '#8c8c8c', fontFamily: 'sans-serif' }}>
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
      width: 90,
      render: (_, record) => (
        <Badge
          status={record.enabled ? 'success' : 'default'}
          text={record.enabled ? '启用' : '停用'}
        />
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      render: (_, record) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {formatDate(record.updatedAt)}
        </Text>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 130,
      align: 'right',
      fixed: 'right',
      render: (_, record) => {
        // B 型基线行对非总部禁用编辑：不渲染编辑与停用/启用入口。
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
  ], [codeLabel, extraColumns, onUpdate, onToggleActive, canEditRecord]);

  return (
    <div style={{ minHeight: '100%' }}>
      {/* 0. 组织治理提示横幅：A 型非总部只读提示 / B 型非总部基线+本地说明 */}
      {notice && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message={notice}
        />
      )}
      {/* 1. Stats Row: 6-column grid per row (xs=12, sm=8, md=4, lg=4, xl=4) */}
      {showStats && (
        <Row gutter={[10, 10]} style={{ marginBottom: 12 }}>
          <Col xs={12} sm={8} md={4} lg={4} xl={4}>
            <Card
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
                title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>全部{title.replace(/管理|维护/g, '')}</span>}
                value={totalCount}
                styles={{ content: { fontSize: 18, fontWeight: 600, color: '#262626' } }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={4} lg={4} xl={4}>
            <Card
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
                title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>启用中</span>}
                value={activeCount}
                styles={{ content: { fontSize: 18, fontWeight: 600, color: '#52c41a' } }}
                prefix={<CheckCircleOutlined style={{ fontSize: 14 }} />}
              />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={4} lg={4} xl={4}>
            <Card
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
                title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>已停用</span>}
                value={disabledCount}
                styles={{ content: { fontSize: 18, fontWeight: 600, color: '#ff4d4f' } }}
                prefix={<CloseCircleOutlined style={{ fontSize: 14 }} />}
              />
            </Card>
          </Col>
          {extraStats.map((stat, idx) => (
            <Col xs={12} sm={8} md={4} lg={4} xl={4} key={stat.label || idx}>
              <Card
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
                  title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>{stat.label}</span>}
                  value={stat.value}
                  styles={{ content: { fontSize: 18, fontWeight: 600, color: stat.color || '#1677ff' } }}
                />
              </Card>
            </Col>
          ))}
        </Row>
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
        styles={{ body: { padding: '10px 14px' } }}
      >
        <ProTable<T>
          actionRef={actionRef}
          rowKey="id"
          columns={proColumns}
          dataSource={displayDataSource}
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
          toolBarRender={() => [
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
          ].filter(Boolean)}
          options={{
            reload: onRefresh ? () => handleRefresh() : false,
            density: true,
            fullScreen: true,
            setting: true,
          }}
          pagination={{
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
          }}
        />
      </Card>

      {/* 3. Dynamic Create / Edit Modal Form */}
      <ModalForm
        title={editingItem ? `编辑${title.replace(/管理|维护/g, '')} - ${editingItem.code}` : `新增${title.replace(/管理|维护/g, '')}`}
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
                  rules={field.rules || (field.required ? [{ required: true, message: `请选择${field.label}` }] : undefined)}
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
                  rules={field.rules || (field.required ? [{ required: true, message: `请输入${field.label}` }] : undefined)}
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
                  rules={field.rules || (field.required ? [{ required: true, message: `请输入${field.label}` }] : undefined)}
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
                  rules={field.rules || (field.required ? [{ required: true, message: `请选择${field.label}` }] : undefined)}
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
                  rules={field.rules || (field.required ? [{ required: true, message: `请选择${field.label}` }] : undefined)}
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
                rules={field.rules || (field.required ? [{ required: true, message: `请输入${field.label}` }] : undefined)}
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
