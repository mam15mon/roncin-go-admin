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
import {
  ModalForm,
  ProFormCheckbox,
  ProFormDigit,
  ProFormRadio,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  App,
  Badge,
  Button,
  Card,
  Col,
  Form,
  Input,
  Pagination,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import React, { useMemo, useState } from 'react';
import type { BaseMasterDataItem, MasterDataTemplateProps } from './types';

const { Text } = Typography;

export function MasterDataTemplate<T extends BaseMasterDataItem = BaseMasterDataItem>({
  title,
  subtitle,
  icon,
  codeLabel = '代码',
  items,
  loading = false,
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
}: MasterDataTemplateProps<T>) {
  const { message } = App.useApp();

  // Search & Filter state
  const [search, setSearch] = useState('');
  const [filterValues, setFilterValues] = useState<Record<string, any>>({});
  const [activeFilter, setActiveFilter] = useState<'all' | 'true' | 'false'>('all');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // Syncing state
  const [syncing, setSyncing] = useState(false);

  // Dialog state
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<T | null>(null);
  const [form] = Form.useForm();

  // Filter items
  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      // 1. Keyword search (code, name, nameEn)
      if (search.trim()) {
        const query = search.trim().toLowerCase();
        const matchCode = item.code.toLowerCase().includes(query);
        const matchName = item.name.toLowerCase().includes(query);
        const matchNameEn = (item.nameEn || '').toLowerCase().includes(query);
        if (!matchCode && !matchName && !matchNameEn) {
          // Check other string properties
          let matchAny = false;
          for (const key of Object.keys(item)) {
            if (typeof item[key] === 'string' && item[key].toLowerCase().includes(query)) {
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
  }, [items, search, activeFilter, filterValues]);

  // Paged items
  const pagedItems = useMemo(() => {
    const start = (page - 1) * pageSize;
    return filteredItems.slice(start, start + pageSize);
  }, [filteredItems, page, pageSize]);

  // Stats calculation
  const totalCount = items.length;
  const activeCount = useMemo(() => items.filter((i) => i.enabled).length, [items]);
  const disabledCount = totalCount - activeCount;

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
    } finally {
      setSyncing(false);
    }
  };

  // Build Columns
  const columns: ColumnsType<T> = [
    {
      title: codeLabel,
      dataIndex: 'code',
      key: 'code',
      width: 140,
      render: (code: string) => (
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
            {code}
          </Tag>
          <Tooltip title={`复制${codeLabel}`}>
            <Button
              type="text"
              size="small"
              icon={<CopyOutlined style={{ fontSize: 11, color: '#8c8c8c' }} />}
              onClick={() => handleCopyCode(code)}
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
      render: (name: string, record: T) => (
        <div>
          <div style={{ fontWeight: 600, fontSize: 13, color: '#262626' }}>
            {name}
          </div>
          {record.nameEn && (
            <div style={{ fontSize: 11, color: '#8c8c8c', fontFamily: 'sans-serif' }}>
              {record.nameEn}
            </div>
          )}
        </div>
      ),
    },
    // Injected Custom Columns
    ...(extraColumns as any[]),
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 90,
      render: (enabled: boolean) => (
        <Badge
          status={enabled ? 'success' : 'default'}
          text={enabled ? '启用' : '停用'}
        />
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      render: (t: string) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t || '-'}
        </Text>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 130,
      align: 'right',
      render: (_: any, record: T) => (
        <Space size={6}>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            style={{ padding: 0 }}
            onClick={() => handleOpenEdit(record)}
          >
            编辑
          </Button>
          {onToggleActive && (
            <Popconfirm
              title={`确定要${record.enabled ? '停用' : '启用'}【${record.name}】吗？`}
              onConfirm={() => onToggleActive(record)}
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
      ),
    },
  ];

  return (
    <div style={{ minHeight: '100%' }}>
      {/* 1. Page Header */}
      <Card
        size="small"
        bordered={false}
        style={{
          borderRadius: 8,
          marginBottom: 12,
          backgroundColor: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
        }}
        styles={{ body: { padding: '14px 20px' } }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12 }}>
          <Space size={10} align="center">
            {icon && (
              <div
                style={{
                  width: 38,
                  height: 38,
                  borderRadius: 8,
                  backgroundColor: '#e6f4ff',
                  color: '#1677ff',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 20,
                }}
              >
                {icon}
              </div>
            )}
            <div>
              <div style={{ fontSize: 16, fontWeight: 600, color: 'rgba(0, 0, 0, 0.88)' }}>
                {title}
              </div>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {subtitle}
              </Text>
            </div>
          </Space>

          {/* Header Action Buttons */}
          <Space size={8}>
            {onSync && (
              <Button
                icon={<CloudSyncOutlined />}
                loading={syncing}
                onClick={handleSyncTrigger}
              >
                同步官方数据
              </Button>
            )}
            {onExport && (
              <Button
                icon={<CloudDownloadOutlined />}
                onClick={onExport}
              >
                导出数据
              </Button>
            )}
            {onCreate && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleOpenCreate}
                style={{ fontWeight: 500 }}
              >
                新增{title.replace(/管理|维护/g, '')}
              </Button>
            )}
          </Space>
        </div>
      </Card>

      {/* 2. Top Stats Bar */}
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col xs={12} sm={6} md={6}>
          <Card
            size="small"
            bordered={false}
            style={{ borderRadius: 6, backgroundColor: '#ffffff', boxShadow: '0 1px 2px rgba(0,0,0,0.02)' }}
            styles={{ body: { padding: '10px 16px' } }}
          >
            <Statistic
              title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>全部{title.replace(/管理|维护/g, '')}</span>}
              value={totalCount}
              valueStyle={{ fontSize: 20, fontWeight: 600, color: '#262626' }}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6} md={6}>
          <Card
            size="small"
            bordered={false}
            style={{ borderRadius: 6, backgroundColor: '#ffffff', boxShadow: '0 1px 2px rgba(0,0,0,0.02)' }}
            styles={{ body: { padding: '10px 16px' } }}
          >
            <Statistic
              title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>启用中</span>}
              value={activeCount}
              valueStyle={{ fontSize: 20, fontWeight: 600, color: '#52c41a' }}
              prefix={<CheckCircleOutlined style={{ fontSize: 16 }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6} md={6}>
          <Card
            size="small"
            bordered={false}
            style={{ borderRadius: 6, backgroundColor: '#ffffff', boxShadow: '0 1px 2px rgba(0,0,0,0.02)' }}
            styles={{ body: { padding: '10px 16px' } }}
          >
            <Statistic
              title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>已停用</span>}
              value={disabledCount}
              valueStyle={{ fontSize: 20, fontWeight: 600, color: '#ff4d4f' }}
              prefix={<CloseCircleOutlined style={{ fontSize: 16 }} />}
            />
          </Card>
        </Col>
        {extraStats.map((stat, idx) => (
          <Col xs={12} sm={6} md={6} key={stat.label || idx}>
            <Card
              size="small"
              bordered={false}
              style={{ borderRadius: 6, backgroundColor: '#ffffff', boxShadow: '0 1px 2px rgba(0,0,0,0.02)' }}
              styles={{ body: { padding: '10px 16px' } }}
            >
              <Statistic
                title={<span style={{ fontSize: 12, color: '#8c8c8c' }}>{stat.label}</span>}
                value={stat.value}
                valueStyle={{ fontSize: 20, fontWeight: 600, color: stat.color || '#1677ff' }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {/* 3. Search & Filters Bar */}
      <Card
        size="small"
        bordered={false}
        style={{
          borderRadius: 8,
          marginBottom: 12,
          backgroundColor: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
        }}
        styles={{ body: { padding: '12px 16px' } }}
      >
        <Row gutter={[10, 10]} justify="space-between" align="middle">
          <Col xs={24} lg={18}>
            <Space wrap size={8}>
              {/* Keyword input */}
              <Input
                placeholder={searchPlaceholder}
                prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(1);
                }}
                style={{ width: 220 }}
                allowClear
              />

              {/* Status Select */}
              <Select
                value={activeFilter}
                onChange={(val) => {
                  setActiveFilter(val);
                  setPage(1);
                }}
                options={[
                  { label: '全部状态', value: 'all' },
                  { label: '仅启用', value: 'true' },
                  { label: '仅停用', value: 'false' },
                ]}
                style={{ width: 105 }}
              />

              {/* Dynamic Filter Dropdowns */}
              {filterOptions.map((opt) => (
                <Select
                  key={opt.key}
                  placeholder={opt.placeholder || opt.label}
                  value={filterValues[opt.key] ?? 'all'}
                  onChange={(val) => {
                    setFilterValues({ ...filterValues, [opt.key]: val });
                    setPage(1);
                  }}
                  options={opt.options}
                  style={{ width: opt.width || 130 }}
                  allowClear
                />
              ))}

              {/* Reset button */}
              <Button
                icon={<ReloadOutlined />}
                onClick={() => {
                  setSearch('');
                  setActiveFilter('all');
                  setFilterValues({});
                  setPage(1);
                  if (onRefresh) onRefresh();
                }}
              >
                重置
              </Button>
            </Space>
          </Col>

          <Col xs={24} lg={6} style={{ textAlign: 'right' }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              当前筛选显示 <Text strong style={{ color: '#1677ff' }}>{filteredItems.length}</Text> / {totalCount} 条
            </Text>
          </Col>
        </Row>
      </Card>

      {/* 4. Table Container */}
      <Card
        size="small"
        bordered={false}
        style={{
          borderRadius: 8,
          backgroundColor: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
        }}
        styles={{ body: { padding: 0 } }}
      >
        <Spin spinning={loading}>
          <Table
            columns={columns}
            dataSource={pagedItems}
            rowKey="id"
            pagination={false}
            size="middle"
          />
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '12px 16px',
              borderTop: '1px solid #f0f0f0',
            }}
          >
            <Text type="secondary" style={{ fontSize: 12 }}>
              第 {page} 页 / 共 {Math.ceil(filteredItems.length / pageSize) || 1} 页
            </Text>
            <Pagination
              current={page}
              pageSize={pageSize}
              total={filteredItems.length}
              size="small"
              showSizeChanger
              pageSizeOptions={['10', '20', '50', '100']}
              onChange={(p, ps) => {
                setPage(p);
                setPageSize(ps);
              }}
            />
          </div>
        </Spin>
      </Card>

      {/* 5. Dynamic Create / Edit Modal Form */}
      <ModalForm
        title={editingItem ? `编辑${title.replace(/管理|维护/g, '')} - ${editingItem.code}` : `新增${title.replace(/管理|维护/g, '')}`}
        open={modalOpen}
        form={form}
        onOpenChange={setModalOpen}
        onFinish={handleFormFinish}
        modalProps={{
          destroyOnClose: true,
          maskClosable: false,
          width: 520,
        }}
        layout="horizontal"
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
