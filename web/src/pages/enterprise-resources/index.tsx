import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  LinkOutlined,
  PlusOutlined,
  TagsOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { history, useAccess, useLocation } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Checkbox,
  ColorPicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Tabs,
  Tag,
  Upload,
} from 'antd';
import type { UploadFile } from 'antd';
import type { RcFile } from 'antd/es/upload';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  enterpriseResourceServiceBatchCreateAssociations,
  enterpriseResourceServiceBatchAssignAddressTypes,
  enterpriseResourceServiceBatchRemoveAddressTypes,
  enterpriseResourceServiceBatchAssignAssignees,
  enterpriseResourceServiceBatchRemoveAssignees,
  enterpriseResourceServiceBatchDeleteAssociations,
  enterpriseResourceServiceCreateEnterpriseResource,
  enterpriseResourceServiceCreateEnterpriseTagGroup,
  enterpriseResourceServiceDeleteEnterpriseResource,
  enterpriseResourceServiceDeleteEnterpriseTagGroup,
  enterpriseResourceServiceGetEnterpriseResourceCapabilities,
  enterpriseResourceServiceListEnterpriseResources,
  enterpriseResourceServiceListEnterpriseResourceRegionOptions,
  enterpriseResourceServiceListEnterpriseTagGroups,
  enterpriseResourceServicePrepareEnterpriseResourceImageUpload,
  enterpriseResourceServiceSearchEnterpriseResourceAssigneeOptions,
  enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions,
  enterpriseResourceServiceGetEnterpriseResourceImageAccess,
  enterpriseResourceServicePreviewEnterpriseResourceImport,
  enterpriseResourceServiceCommitEnterpriseResourceImport,
  enterpriseResourceServiceUpdateEnterpriseResource,
  enterpriseResourceServiceUpdateEnterpriseTagGroup,
} from '@/services/roncin/enterpriseResourceService';
import { toTableRequest, unwrapList } from '@/utils/api';

const resourceTabs = [
  { key: 'addresses', label: '地址管理', type: 1 },
  { key: 'remarks', label: '备注管理', type: 2 },
  { key: 'images', label: '图片管理', type: 3 },
  { key: 'tags', label: '标签管理', type: 4 },
  { key: 'consignees', label: '收货人管理', type: 6 },
  { key: 'shippers', label: '发货人管理', type: 5 },
  { key: 'notify-parties', label: '通知人管理', type: 7 },
] as const;

const remarkTypes = [
  '订舱备注', '配舱备注', '运输委托备注', '订单备注', '提单备注', '客户备注', '供应商备注',
  '国外代理备注', '报价备注', '舱单备注', '装箱单备注', '操作备注', '提成备注', '仓储备注',
].map((label, index) => ({ label, value: index + 1 }));
const addressTypes = [
  { label: '拆/装箱地址', value: 1 },
  { label: '提货地址', value: 2 },
  { label: '送货地址', value: 3 },
];
const partyTypes = new Set([5, 6, 7]);
const importHeaders = ['简称', '企业名称', '企业代码', '地址', '国家代码', '联系人', '电话', '邮箱', '税号', 'AEO代码'];
const importConflictFieldLabels: Record<string, string> = { business_code: '企业代码', company_name: '企业名称' };

async function imageChecksum(file: Blob): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', await file.arrayBuffer());
  return btoa(String.fromCharCode(...new Uint8Array(digest)));
}

function formatStorageSize(value?: string): string {
  const bytes = Number(value ?? 0);
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KiB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MiB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GiB`;
}

function parseCSV(content: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let field = '';
  let quoted = false;
  for (let index = 0; index < content.length; index += 1) {
    const character = content[index];
    if (character === '"') {
      if (quoted && content[index + 1] === '"') { field += '"'; index += 1; }
      else quoted = !quoted;
    } else if (character === ',' && !quoted) { row.push(field.trim()); field = ''; }
    else if ((character === '\n' || character === '\r') && !quoted) {
      if (character === '\r' && content[index + 1] === '\n') index += 1;
      row.push(field.trim());
      if (row.some(Boolean)) rows.push(row);
      row = [];
      field = '';
    } else field += character;
  }
  if (quoted) throw new Error('CSV 文件存在未闭合的双引号');
  row.push(field.trim());
  if (row.some(Boolean)) rows.push(row);
  return rows;
}

function parseImportFile(content: string, resourceType: number): API.EnterpriseResourceInput[] {
  const rows = parseCSV(content.replace(/^\uFEFF/, ''));
  if (!rows.length || importHeaders.some((header, index) => rows[0][index] !== header)) throw new Error(`CSV 表头必须为：${importHeaders.join(',')}`);
  return rows.slice(1).map((values) => {
    const [shortName, companyName, businessCode, address, countryCode = 'CN', contactName, contactPhone, email, taxIdentifier, aeoCode] = values;
    return { resourceType, shortName: shortName ?? '', enabled: true, sortOrder: 0, party: { companyName, businessCode, address, countryCode: countryCode || 'CN', contactName, contactPhone, email, taxIdentifier, aeoCode } };
  });
}

type EditorValues = {
  shortName: string;
  enabled: boolean;
  sortOrder?: number;
  partnerIds?: string[];
  contactName?: string;
  contactPhone?: string;
  countryCode?: string;
  provinceCode?: string;
  cityCode?: string;
  districtCode?: string;
  addressDetail?: string;
  addressRemark?: string;
  addressTypes?: number[];
  remarkType?: number;
  content?: string;
  companyName?: string;
  businessCode?: string;
  partyAddress?: string;
  email?: string;
  taxIdentifier?: string;
  aeoCode?: string;
  customDisplay?: boolean;
  displayContent?: string;
  partyRemark?: string;
  groupId?: string;
  assigneeIds?: string[];
};

const EnterpriseResourcesPage: React.FC = () => {
  const access = useAccess();
  const location = useLocation();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType>(undefined);
  const [form] = Form.useForm<EditorValues>();
  const queryTab = new URLSearchParams(location.search).get('tab');
  const [capabilities, setCapabilities] = useState<API.GetEnterpriseResourceCapabilitiesResponse>();
  const availableTabs = useMemo(
    () => capabilities?.imageEnabled ? resourceTabs : resourceTabs.filter((item) => item.type !== 3),
    [capabilities?.imageEnabled],
  );
  const active =
    availableTabs.find((item) => item.key === queryTab) ?? availableTabs[0];
  const activeTab = active.key;
  const [editorOpen, setEditorOpen] = useState(false);
  const [editing, setEditing] = useState<API.EnterpriseResource>();
  const [saving, setSaving] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [associationOpen, setAssociationOpen] = useState(false);
  const [associationMode, setAssociationMode] = useState<'link' | 'unlink'>('link');
  const [associationPartners, setAssociationPartners] = useState<string[]>([]);
  const [addressTypeOpen, setAddressTypeOpen] = useState(false);
  const [addressTypeMode, setAddressTypeMode] = useState<'assign' | 'remove'>('assign');
  const [batchAddressTypes, setBatchAddressTypes] = useState<number[]>([]);
  const [assigneeOpen, setAssigneeOpen] = useState(false);
  const [assigneeMode, setAssigneeMode] = useState<'assign' | 'remove'>('assign');
  const [batchAssignees, setBatchAssignees] = useState<string[]>([]);
  const [partnerOptions, setPartnerOptions] = useState<{ label: string; value: string }[]>([]);
  const [assigneeOptions, setAssigneeOptions] = useState<{ label: string; value: string }[]>([]);
  const [provinceOptions, setProvinceOptions] = useState<{ label: string; value: string }[]>([]);
  const [cityOptions, setCityOptions] = useState<{ label: string; value: string }[]>([]);
  const [districtOptions, setDistrictOptions] = useState<{ label: string; value: string }[]>([]);
  const [tagGroups, setTagGroups] = useState<API.EnterpriseTagGroup[]>([]);
  const [groupOpen, setGroupOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<API.EnterpriseTagGroup>();
  const [groupForm] = Form.useForm();
  const [imageFiles, setImageFiles] = useState<UploadFile[]>([]);
  const [importOpen, setImportOpen] = useState(false);
  const [importFiles, setImportFiles] = useState<UploadFile[]>([]);
  const [importRows, setImportRows] = useState<API.EnterpriseResourceInput[]>([]);
  const [importPreview, setImportPreview] = useState<API.PreviewEnterpriseResourceImportResponse>();
  const [importLoading, setImportLoading] = useState(false);
  const countryCode = Form.useWatch('countryCode', form);
  const provinceCode = Form.useWatch('provinceCode', form);
  const cityCode = Form.useWatch('cityCode', form);

  useEffect(() => {
    void enterpriseResourceServiceGetEnterpriseResourceCapabilities().then(setCapabilities);
  }, []);

  const loadTagGroups = useCallback(async () => {
    const response = await enterpriseResourceServiceListEnterpriseTagGroups();
    setTagGroups(unwrapList(response));
  }, []);

  useEffect(() => {
    if (active.type === 4) void loadTagGroups();
    setSelectedKeys([]);
  }, [active.type, loadTagGroups]);

  const searchPartners = useCallback(async (keyword = '') => {
    const response = await enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions({ page: 1, pageSize: 50, keyword });
    setPartnerOptions(unwrapList(response).flatMap((item) => item.id ? [{ value: item.id, label: `${item.code ?? ''} ${item.name ?? ''}`.trim() }] : []));
  }, []);

  const searchAssignees = useCallback(async (keyword = '') => {
    const response = await enterpriseResourceServiceSearchEnterpriseResourceAssigneeOptions({ page: 1, pageSize: 50, keyword });
    setAssigneeOptions(unwrapList(response).flatMap((item) => item.id ? [{ value: item.id, label: item.displayName ?? item.username ?? item.id }] : []));
  }, []);

  const loadRegions = useCallback(async (level: number, parentCode?: string) => {
    const response = await enterpriseResourceServiceListEnterpriseResourceRegionOptions({ level, parentCode, page: 1, pageSize: 200 });
    return unwrapList(response).flatMap((item) => item.code ? [{ value: item.code, label: item.name ?? item.code }] : []);
  }, []);

  useEffect(() => {
    if (editorOpen && active.type === 1 && countryCode === 'CN') void loadRegions(1).then(setProvinceOptions);
  }, [active.type, countryCode, editorOpen, loadRegions]);
  useEffect(() => {
    if (editorOpen && provinceCode) void loadRegions(2, provinceCode).then(setCityOptions);
  }, [editorOpen, loadRegions, provinceCode]);
  useEffect(() => {
    if (editorOpen && cityCode) void loadRegions(3, cityCode).then(setDistrictOptions);
  }, [cityCode, editorOpen, loadRegions]);

  const openEditor = (record?: API.EnterpriseResource) => {
    setEditing(record);
    setImageFiles([]);
    if (!record) {
      form.resetFields();
      form.setFieldsValue({ enabled: true, sortOrder: 0, countryCode: 'CN', partnerIds: [] });
    } else {
      form.setFieldsValue({
        shortName: record.shortName ?? '', enabled: record.enabled ?? true, sortOrder: record.sortOrder ?? 0,
        partnerIds: record.partnerIds ?? [], contactName: record.address?.contactName ?? record.party?.contactName,
        contactPhone: record.address?.contactPhone ?? record.party?.contactPhone, countryCode: record.address?.countryCode ?? record.party?.countryCode ?? 'CN',
        provinceCode: record.address?.provinceCode, cityCode: record.address?.cityCode, districtCode: record.address?.districtCode,
        addressDetail: record.address?.addressDetail, addressRemark: record.address?.remark, addressTypes: record.addressTypes, assigneeIds: record.assigneeIds,
        remarkType: record.remark?.remarkType, content: record.remark?.content, companyName: record.party?.companyName,
        businessCode: record.party?.businessCode, partyAddress: record.party?.address, email: record.party?.email,
        taxIdentifier: record.party?.taxIdentifier, aeoCode: record.party?.aeoCode, customDisplay: record.party?.customDisplay,
        displayContent: record.party?.displayContent, partyRemark: record.party?.remark, groupId: record.tag?.groupId,
      });
      void searchPartners();
    }
    setEditorOpen(true);
  };

  const toInput = (values: EditorValues): API.EnterpriseResourceInput => {
    const input: API.EnterpriseResourceInput = {
      resourceType: active.type, shortName: values.shortName.trim(), enabled: values.enabled,
      sortOrder: values.sortOrder ?? 0, partnerAssociations: { partnerIds: values.partnerIds ?? [] },
    };
    if (active.type === 1) {
      input.address = { contactName: values.contactName, contactPhone: values.contactPhone, countryCode: values.countryCode, provinceCode: values.provinceCode, cityCode: values.cityCode, districtCode: values.districtCode, addressDetail: values.addressDetail, remark: values.addressRemark };
      input.addressTypes = values.addressTypes ?? [];
      input.assigneeIds = values.assigneeIds ?? [];
    } else if (active.type === 2) {
      input.remark = { remarkType: values.remarkType, content: values.content };
    } else if (partyTypes.has(active.type)) {
      input.party = { companyName: values.companyName, businessCode: values.businessCode, address: values.partyAddress, countryCode: values.countryCode, contactName: values.contactName, contactPhone: values.contactPhone, email: values.email, taxIdentifier: values.taxIdentifier, aeoCode: values.aeoCode, customDisplay: values.customDisplay, displayContent: values.displayContent, remark: values.partyRemark };
    } else if (active.type === 4) {
      input.tag = { groupId: values.groupId };
    } else if (active.type === 3 && imageFiles[0]) {
      const file = imageFiles[0].originFileObj;
      input.image = { fileName: file?.name, mimeType: file?.type, fileSize: String(file?.size ?? 0), objectKey: imageFiles[0].response?.objectKey, checksum: imageFiles[0].response?.checksum };
    }
    return input;
  };

  const saveResource = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const input = toInput(values);
      if (editing?.id) await enterpriseResourceServiceUpdateEnterpriseResource({ id: editing.id }, { id: editing.id, resource: input });
      else await enterpriseResourceServiceCreateEnterpriseResource({ resource: input });
      message.success(editing ? '资源已更新' : '资源已创建');
      setEditorOpen(false);
      actionRef.current?.reload();
    } finally { setSaving(false); }
  };

  const columns = useMemo<ProColumns<API.EnterpriseResource>[]>(() => {
    const detailColumns: ProColumns<API.EnterpriseResource>[] = [];
    if (active.type === 1) detailColumns.push({ title: '详细地址', dataIndex: ['address', 'addressDetail'], ellipsis: true }, { title: '联系人', dataIndex: ['address', 'contactName'] });
    if (active.type === 2) detailColumns.push({ title: '备注类型', render: (_, record) => remarkTypes.find((item) => item.value === record.remark?.remarkType)?.label ?? '-' }, { title: '备注内容', dataIndex: ['remark', 'content'], ellipsis: true });
    if (partyTypes.has(active.type)) detailColumns.push({ title: '企业名称', dataIndex: ['party', 'companyName'], ellipsis: true }, { title: '企业代码', dataIndex: ['party', 'businessCode'] }, { title: '国家', dataIndex: ['party', 'countryCode'], width: 72 });
    if (active.type === 4) detailColumns.push({ title: '标签组', render: (_, record) => tagGroups.find((group) => group.id === record.tag?.groupId)?.name ?? '-' });
    if (active.type === 3) detailColumns.push({ title: '文件', render: (_, record) => <Button type="link" onClick={async () => { if (!record.id) return; const response = await enterpriseResourceServiceGetEnterpriseResourceImageAccess({ id: record.id }); if (response.url) window.open(response.url, '_blank', 'noopener,noreferrer'); }}>{record.image?.fileName ?? '预览'}</Button> }, { title: '大小', render: (_, record) => record.image?.fileSize ? `${(Number(record.image.fileSize) / 1024 / 1024).toFixed(2)} MiB` : '-' });
    return [
      { title: '关键词', dataIndex: 'keyword', hideInTable: true },
      { title: '关联状态', dataIndex: 'linked', hideInTable: true, valueType: 'select', fieldProps: { options: [{ label: '已关联', value: 'true' }, { label: '独立资源', value: 'false' }] } },
      { title: '关联企业', dataIndex: 'partnerId', hideInTable: true, valueType: 'select', fieldProps: { showSearch: { filterOption: false, onSearch: (value: string) => void searchPartners(value) }, onFocus: () => void searchPartners(), options: partnerOptions } },
      ...(active.type === 1 ? [{ title: '地址类型', dataIndex: 'addressType', hideInTable: true, valueType: 'select' as const, fieldProps: { options: addressTypes } }, { title: '关联人员', dataIndex: 'assigneeId', hideInTable: true, valueType: 'select' as const, fieldProps: { showSearch: { filterOption: false, onSearch: (value: string) => void searchAssignees(value) }, onFocus: () => void searchAssignees(), options: assigneeOptions } }] : []),
      { title: '简称/名称', dataIndex: 'shortName', ellipsis: true, hideInSearch: true, sorter: true }, ...detailColumns,
      { title: '关联企业', dataIndex: 'partnerIds', render: (_, record) => <Tag color={record.partnerIds?.length ? 'blue' : 'default'}>{record.partnerIds?.length ? `${record.partnerIds.length} 家` : '独立资源'}</Tag> },
      { title: '状态', dataIndex: 'enabled', width: 80, valueType: 'select', fieldProps: { options: [{ label: '启用', value: 'true' }, { label: '停用', value: 'false' }] }, render: (_, record) => <Tag color={record.enabled ? 'success' : 'default'}>{record.enabled ? '启用' : '停用'}</Tag> },
      { title: '更新时间', dataIndex: 'updatedAt', valueType: 'dateTime', width: 170, sorter: true },
      { title: '操作', valueType: 'option', fixed: 'right', width: 120, render: (_, record) => [
        access.canUpdateEnterpriseResources && active.type !== 3 && <Button key="edit" type="link" size="small" icon={<EditOutlined />} onClick={() => openEditor(record)}>编辑</Button>,
        access.canDeleteEnterpriseResources && <Popconfirm key="delete" title="确认删除此资源？" onConfirm={async () => { if (!record.id) return; await enterpriseResourceServiceDeleteEnterpriseResource({ id: record.id }); message.success('资源已删除'); actionRef.current?.reload(); }}><Button type="link" danger size="small" icon={<DeleteOutlined />}>删除</Button></Popconfirm>,
      ] },
    ];
  }, [access, active.type, assigneeOptions, message, partnerOptions, searchAssignees, searchPartners, tagGroups]);

  const submitAssociation = async () => {
    if (!selectedKeys.length || !associationPartners.length) return;
    const body = { resourceIds: selectedKeys.map(String), partnerIds: associationPartners };
    if (associationMode === 'link') await enterpriseResourceServiceBatchCreateAssociations(body);
    else await enterpriseResourceServiceBatchDeleteAssociations(body);
    message.success(associationMode === 'link' ? '已关联企业' : '已解除关联');
    setAssociationOpen(false); setSelectedKeys([]); actionRef.current?.reload();
  };

  const submitAddressTypes = async () => {
    if (!selectedKeys.length || !batchAddressTypes.length) return;
    const body = { resourceIds: selectedKeys.map(String), addressTypes: batchAddressTypes };
    if (addressTypeMode === 'assign') await enterpriseResourceServiceBatchAssignAddressTypes(body);
    else await enterpriseResourceServiceBatchRemoveAddressTypes(body);
    message.success(addressTypeMode === 'assign' ? '已分配地址类型' : '已移除地址类型');
    setAddressTypeOpen(false); setSelectedKeys([]); actionRef.current?.reload();
  };

  const submitAssignees = async () => {
    if (!selectedKeys.length || !batchAssignees.length) return;
    const body = { resourceIds: selectedKeys.map(String), assigneeIds: batchAssignees };
    if (assigneeMode === 'assign') await enterpriseResourceServiceBatchAssignAssignees(body);
    else await enterpriseResourceServiceBatchRemoveAssignees(body);
    message.success(assigneeMode === 'assign' ? '已关联人员' : '已移除关联人员');
    setAssigneeOpen(false); setSelectedKeys([]); actionRef.current?.reload();
  };

  const saveGroup = async () => {
    const values = await groupForm.validateFields();
    const color = typeof values.color === 'string' ? values.color : values.color?.toHexString();
    if (editingGroup?.id) await enterpriseResourceServiceUpdateEnterpriseTagGroup({ id: editingGroup.id }, { id: editingGroup.id, group: { name: values.name, color, sortOrder: values.sortOrder ?? 0 } });
    else await enterpriseResourceServiceCreateEnterpriseTagGroup({ group: { name: values.name, color, sortOrder: values.sortOrder ?? 0 } });
    message.success('标签组已保存'); setGroupOpen(false); await loadTagGroups();
  };

  const previewImport = async () => {
    setImportLoading(true);
    try { setImportPreview(await enterpriseResourceServicePreviewEnterpriseResourceImport({ resourceType: active.type, rows: importRows })); }
    finally { setImportLoading(false); }
  };

  const commitImport = async () => {
    setImportLoading(true);
    try {
      const response = await enterpriseResourceServiceCommitEnterpriseResourceImport({ resourceType: active.type, rows: importRows, overwriteConflicts: (importPreview?.conflictCount ?? 0) > 0 });
      message.success(`已新增 ${response.createdCount ?? 0} 条，更新 ${response.updatedCount ?? 0} 条资源`);
      setImportOpen(false); setImportPreview(undefined); setImportFiles([]); setImportRows([]); actionRef.current?.reload();
    } finally { setImportLoading(false); }
  };

  const downloadImportTemplate = () => {
    const content = `\uFEFF${importHeaders.join(',')}\r\n示例主体,上海示例公司,SAMPLE001,上海市浦东新区,CN,张三,13800000000,example@example.com,,`;
    const url = URL.createObjectURL(new Blob([content], { type: 'text/csv;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `${active.label.replace('管理', '')}导入模板.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return (
    <PageContainer title="配置管理" subTitle="组织级企业资源备忘录">
      <Tabs activeKey={activeTab} items={availableTabs.map(({ key, label }) => ({ key, label }))} onChange={(tab) => history.replace(`/enterprise-resources/config?tab=${tab}`)} />
      {active.type === 3 && <Space style={{ marginBottom: 16 }}><span>组织图片存储空间</span><Tag color="blue">已用 {formatStorageSize(capabilities?.imageUsedStorageBytes)}</Tag>{Number(capabilities?.imageStorageQuotaBytes) > 0 ? <><Tag>总额 {formatStorageSize(capabilities?.imageStorageQuotaBytes)}</Tag><Tag color="cyan">{Math.min(100, Number(capabilities?.imageUsedStorageBytes) / Number(capabilities?.imageStorageQuotaBytes) * 100).toFixed(2)}%</Tag></> : <Tag>不限额</Tag>}</Space>}
      <ProTable<API.EnterpriseResource>
        actionRef={actionRef} rowKey="id" columns={columns} search={{ labelWidth: 'auto' }}
        rowSelection={{ selectedRowKeys: selectedKeys, preserveSelectedRowKeys: true, onChange: setSelectedKeys }}
        request={async (params, sort) => {
          const sortEntry = Object.entries(sort ?? {})[0];
          const sortBy = sortEntry?.[0] === 'shortName' ? 'short_name' : sortEntry?.[0] === 'updatedAt' ? 'updated_at' : undefined;
          const response = await enterpriseResourceServiceListEnterpriseResources({ resourceType: active.type, page: params.current, pageSize: params.pageSize, keyword: params.keyword as string | undefined, linked: params.linked === 'true' ? true : params.linked === 'false' ? false : undefined, enabled: params.enabled === 'true' ? true : params.enabled === 'false' ? false : undefined, partnerId: params.partnerId as string | undefined, addressType: params.addressType as number | undefined, assigneeId: params.assigneeId as string | undefined, sortBy, sortOrder: sortEntry?.[1] === 'descend' ? 'desc' : sortEntry ? 'asc' : undefined });
          return toTableRequest(response);
        }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }} scroll={{ x: 1100 }}
        toolBarRender={() => [
          active.type === 4 && access.canCreateEnterpriseResources && <Button key="groups" icon={<TagsOutlined />} onClick={() => { setEditingGroup(undefined); groupForm.resetFields(); setGroupOpen(true); }}>标签组</Button>,
          partyTypes.has(active.type) && access.canCreateEnterpriseResources && <Button key="import" icon={<UploadOutlined />} onClick={() => { setImportOpen(true); setImportPreview(undefined); setImportFiles([]); setImportRows([]); }}>批量导入</Button>,
          active.type !== 2 && active.type !== 3 && active.type !== 4 && access.canUpdateEnterpriseResources && <Button key="link" disabled={!selectedKeys.length} icon={<LinkOutlined />} onClick={() => { setAssociationMode('link'); setAssociationPartners([]); void searchPartners(); setAssociationOpen(true); }}>批量关联</Button>,
          active.type !== 2 && active.type !== 3 && active.type !== 4 && access.canUpdateEnterpriseResources && <Button key="unlink" disabled={!selectedKeys.length} onClick={() => { setAssociationMode('unlink'); setAssociationPartners([]); void searchPartners(); setAssociationOpen(true); }}>解除关联</Button>,
          active.type === 1 && access.canUpdateEnterpriseResources && <Button key="address-type" disabled={!selectedKeys.length} onClick={() => { setAddressTypeMode('assign'); setBatchAddressTypes([]); setAddressTypeOpen(true); }}>批量设置地址类型</Button>,
          active.type === 1 && access.canUpdateEnterpriseResources && <Button key="assignee" disabled={!selectedKeys.length} onClick={() => { setAssigneeMode('assign'); setBatchAssignees([]); void searchAssignees(); setAssigneeOpen(true); }}>批量关联人员</Button>,
          access.canCreateEnterpriseResources && <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => openEditor()}>新建{active.label.replace('管理', '')}</Button>,
        ]}
      />

      <Modal title={editing ? `编辑${active.label}` : `新建${active.label}`} open={editorOpen} width={760} confirmLoading={saving} onOk={() => void saveResource()} onCancel={() => setEditorOpen(false)} destroyOnHidden>
        <Form form={form} layout="vertical">
          <Space align="start" size={16} style={{ width: '100%' }}>
            <Form.Item name="shortName" label="简称/名称" rules={[{ required: true }]}><Input style={{ width: 320 }} maxLength={200} /></Form.Item>
            <Form.Item name="sortOrder" label="排序"><InputNumber min={0} /></Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item>
          </Space>
          <Form.Item name="partnerIds" label="关联企业"><Select mode="multiple" showSearch={{ filterOption: false, onSearch: (value) => void searchPartners(value) }} onFocus={() => void searchPartners()} options={partnerOptions} placeholder="可不选择，保存为独立资源" /></Form.Item>
          {active.type === 1 && <>
            <Space size={16}><Form.Item name="contactName" label="联系人"><Input /></Form.Item><Form.Item name="contactPhone" label="联系电话"><Input /></Form.Item><Form.Item name="countryCode" label="国家/地区" rules={[{ required: true }]}><Input maxLength={2} /></Form.Item></Space>
            {countryCode === 'CN' && <Space size={16}><Form.Item name="provinceCode" label="省"><Select style={{ width: 180 }} allowClear options={provinceOptions} onChange={() => form.setFieldsValue({ cityCode: undefined, districtCode: undefined })} /></Form.Item><Form.Item name="cityCode" label="市"><Select style={{ width: 180 }} allowClear disabled={!provinceCode} options={cityOptions} onChange={() => form.setFieldValue('districtCode', undefined)} /></Form.Item><Form.Item name="districtCode" label="区县"><Select style={{ width: 180 }} allowClear disabled={!cityCode} options={districtOptions} /></Form.Item></Space>}
            <Form.Item name="addressDetail" label="详细地址" rules={[{ required: true }]}><Input.TextArea rows={3} maxLength={1000} /></Form.Item>
            <Form.Item name="addressTypes" label="地址类型"><Checkbox.Group options={addressTypes} /></Form.Item>
            <Form.Item name="assigneeIds" label="关联人员"><Select mode="multiple" showSearch={{ filterOption: false, onSearch: (value) => void searchAssignees(value) }} onFocus={() => void searchAssignees()} options={assigneeOptions} placeholder="搜索当前组织人员" /></Form.Item>
            <Form.Item name="addressRemark" label="备注"><Input.TextArea rows={2} maxLength={500} /></Form.Item>
          </>}
          {active.type === 2 && <><Form.Item name="remarkType" label="备注类型" rules={[{ required: true }]}><Select options={remarkTypes} /></Form.Item><Form.Item name="content" label="备注内容" rules={[{ required: true }]}><Input.TextArea rows={8} maxLength={4000} showCount /></Form.Item></>}
          {partyTypes.has(active.type) && <>
            <Space size={16}><Form.Item name="companyName" label="企业/单证主体名称" rules={[{ required: true }]}><Input style={{ width: 320 }} /></Form.Item><Form.Item name="businessCode" label="企业代码"><Input /></Form.Item><Form.Item name="countryCode" label="国家代码" rules={[{ required: true }]}><Input maxLength={2} /></Form.Item></Space>
            <Form.Item name="partyAddress" label="企业地址"><Input.TextArea rows={3} /></Form.Item>
            <Space size={16}><Form.Item name="contactName" label="联系人"><Input /></Form.Item><Form.Item name="contactPhone" label="电话"><Input /></Form.Item><Form.Item name="email" label="邮箱"><Input /></Form.Item></Space>
            <Space size={16}><Form.Item name="taxIdentifier" label="税号"><Input /></Form.Item><Form.Item name="aeoCode" label="AEO 代码"><Input /></Form.Item><Form.Item name="customDisplay" label="自定义页面展示" valuePropName="checked"><Switch /></Form.Item></Space>
            <Form.Item noStyle shouldUpdate={(previous, current) => previous.customDisplay !== current.customDisplay}>{({ getFieldValue }) => getFieldValue('customDisplay') && <Form.Item name="displayContent" label="页面展示内容"><Input.TextArea rows={5} maxLength={4000} /></Form.Item>}</Form.Item>
            <Form.Item name="partyRemark" label="内部备注"><Input.TextArea rows={2} /></Form.Item>
          </>}
          {active.type === 4 && <Form.Item name="groupId" label="标签组" rules={[{ required: true }]}><Select options={tagGroups.flatMap((group) => group.id ? [{ value: group.id, label: group.name }] : [])} /></Form.Item>}
          {active.type === 3 && <Form.Item label="图片" required><Upload.Dragger accept="image/jpeg,image/png,image/bmp,image/gif" maxCount={1} fileList={imageFiles} beforeUpload={(file) => { if (file.size > Number(capabilities?.imageMaxFileSize)) { message.error(`图片不能超过 ${Number(capabilities?.imageMaxFileSize) / 1024 / 1024} MiB`); return Upload.LIST_IGNORE; } return true; }} customRequest={async ({ file, onError, onSuccess }) => { try { const source = file as RcFile; const checksum = await imageChecksum(source); const prepared = await enterpriseResourceServicePrepareEnterpriseResourceImageUpload({ fileName: source.name, mimeType: source.type, fileSize: String(source.size), checksum }); if (!prepared.uploadUrl || !prepared.objectKey) throw new Error('未获得上传凭证'); const response = await fetch(prepared.uploadUrl, { method: 'PUT', headers: prepared.headers, body: source }); if (!response.ok) throw new Error(`对象存储上传失败：${response.status}`); const uploadResponse = { objectKey: prepared.objectKey, checksum }; setImageFiles([{ uid: source.uid, name: source.name, status: 'done', originFileObj: source, response: uploadResponse }]); onSuccess?.(uploadResponse); } catch (error) { onError?.(error as Error); message.error(error instanceof Error ? error.message : '图片上传失败'); } }} onChange={({ fileList }) => setImageFiles(fileList)} onRemove={() => { setImageFiles([]); return true; }}><p><UploadOutlined /> 点击或拖拽图片上传</p><p>支持 JPG、PNG、BMP、GIF，最大 {Number(capabilities?.imageMaxFileSize) / 1024 / 1024} MiB</p></Upload.Dragger></Form.Item>}
        </Form>
      </Modal>

      <Modal title={associationMode === 'link' ? '批量关联企业' : '批量解除企业关联'} open={associationOpen} onOk={() => void submitAssociation()} onCancel={() => setAssociationOpen(false)}><Select style={{ width: '100%' }} mode="multiple" showSearch={{ filterOption: false, onSearch: (value) => void searchPartners(value) }} options={partnerOptions} value={associationPartners} onChange={setAssociationPartners} placeholder="搜索并选择企业" /></Modal>
      <Modal title="批量维护地址类型" open={addressTypeOpen} onOk={() => void submitAddressTypes()} onCancel={() => setAddressTypeOpen(false)}>
        <Space orientation="vertical" style={{ width: '100%' }}><Select value={addressTypeMode} onChange={setAddressTypeMode} options={[{ label: '分配所选类型', value: 'assign' }, { label: '移除所选类型', value: 'remove' }]} /><Checkbox.Group options={addressTypes} value={batchAddressTypes} onChange={(values) => setBatchAddressTypes(values as number[])} /></Space>
      </Modal>
      <Modal title="批量维护关联人员" open={assigneeOpen} onOk={() => void submitAssignees()} onCancel={() => setAssigneeOpen(false)}>
        <Space orientation="vertical" style={{ width: '100%' }}><Select value={assigneeMode} onChange={setAssigneeMode} options={[{ label: '关联所选人员', value: 'assign' }, { label: '移除所选人员', value: 'remove' }]} /><Select style={{ width: '100%' }} mode="multiple" showSearch={{ filterOption: false, onSearch: (value) => void searchAssignees(value) }} options={assigneeOptions} value={batchAssignees} onChange={setBatchAssignees} placeholder="搜索并选择当前组织人员" /></Space>
      </Modal>

      <Modal title={editingGroup ? '编辑标签组' : '新建标签组'} open={groupOpen} onOk={() => void saveGroup()} onCancel={() => setGroupOpen(false)} destroyOnHidden>
        <Form form={groupForm} layout="vertical" initialValues={{ sortOrder: 0 }}><Form.Item name="name" label="组名" rules={[{ required: true }]}><Input maxLength={100} /></Form.Item><Form.Item name="color" label="颜色"><ColorPicker showText allowClear /></Form.Item><Form.Item name="sortOrder" label="排序"><InputNumber min={0} /></Form.Item></Form>
        {!!tagGroups.length && !editingGroup && <Space orientation="vertical" style={{ width: '100%' }}>{tagGroups.map((group) => <Space key={group.id} style={{ justifyContent: 'space-between', width: '100%' }}><Tag color={group.color}>{group.name}</Tag><Space><Button type="link" size="small" onClick={() => { setEditingGroup(group); groupForm.setFieldsValue(group); }}>编辑</Button><Popconfirm title="仅空标签组可删除，确认删除？" onConfirm={async () => { if (!group.id) return; await enterpriseResourceServiceDeleteEnterpriseTagGroup({ id: group.id }); await loadTagGroups(); }}><Button type="link" danger size="small">删除</Button></Popconfirm></Space></Space>)}</Space>}
      </Modal>
      <Modal title={`批量导入${active.label.replace('管理', '')}`} open={importOpen} okText={!importPreview ? '校验数据' : (importPreview.conflictCount ?? 0) > 0 ? `确认覆盖 ${importPreview.conflictCount} 条` : '确认导入'} confirmLoading={importLoading} okButtonProps={{ disabled: !importRows.length || (importPreview?.invalidCount ?? 0) > 0 || importPreview?.overwriteAllowed === false }} onOk={() => importPreview ? void commitImport() : void previewImport()} onCancel={() => setImportOpen(false)}>
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Space>
            <Upload accept=".csv,text/csv" maxCount={1} fileList={importFiles} beforeUpload={async (file) => {
              try {
                const rows = parseImportFile(await file.text(), active.type);
                if (!rows.length) throw new Error('CSV 文件没有可导入的数据行');
                setImportRows(rows); setImportPreview(undefined);
              } catch (error) {
                setImportRows([]); setImportPreview(undefined); setImportFiles([]);
                message.error(error instanceof Error ? error.message : 'CSV 文件解析失败');
                return Upload.LIST_IGNORE;
              }
              return false;
            }} onChange={({ fileList }) => setImportFiles(fileList.slice(-1))} onRemove={() => { setImportFiles([]); setImportRows([]); setImportPreview(undefined); return true; }}>
              <Button icon={<UploadOutlined />}>选择 CSV 文件</Button>
            </Upload>
            <Button icon={<DownloadOutlined />} onClick={downloadImportTemplate}>下载模板</Button>
          </Space>
          {!!importRows.length && !importPreview && <Alert type="info" showIcon title={`已读取 ${importRows.length} 条数据，请先校验`} />}
          {importPreview && <Alert type={(importPreview.invalidCount ?? 0) > 0 || importPreview.overwriteAllowed === false ? 'error' : (importPreview.conflictCount ?? 0) > 0 ? 'warning' : 'success'} showIcon title={`有效 ${importPreview.validCount ?? 0} 条，无效 ${importPreview.invalidCount ?? 0} 条，冲突 ${importPreview.conflictCount ?? 0} 条`} description={<Space orientation="vertical" size={4}>
            {importPreview.overwriteAllowed === false && <span>存在一行匹配多个资源的歧义，请先修正企业名称或企业代码后重新上传。</span>}
            {importPreview.rows?.filter((row) => row.errors?.length).map((row) => <span key={`error-${row.rowNumber}`}>第 {row.rowNumber} 行：{row.errors?.join('；')}</span>)}
            {importPreview.rows?.filter((row) => row.conflicts?.length).map((row) => <span key={`conflict-${row.rowNumber}`}>第 {row.rowNumber} 行与“{row.conflicts?.map((item) => item.existingShortName).join('、')}”冲突（{row.conflicts?.flatMap((item) => item.matchedFields ?? []).map((field) => importConflictFieldLabels[field] ?? field).join('、')}）{importPreview.overwriteAllowed === false ? '' : '，确认后将覆盖该资源'}</span>)}
          </Space>} />}
        </Space>
      </Modal>
    </PageContainer>
  );
};

export default EnterpriseResourcesPage;
