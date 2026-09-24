import {
  DeleteOutlined,
  EditOutlined,
  LinkOutlined,
  PlusOutlined,
  TagsOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { history } from '@/router/history';
import { useAccess } from '@/app/access';
import { useSearchParams } from 'react-router';
import type { UploadFile } from 'antd';
import { App, Button, Form, Popconfirm, Tabs, Tag } from 'antd';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useColumnSettings } from '@/components/ui/column-settings';
import {
  enterpriseResourceServiceBatchAssignAddressTypes,
  enterpriseResourceServiceBatchAssignAssignees,
  enterpriseResourceServiceBatchCreateAssociations,
  enterpriseResourceServiceBatchDeleteAssociations,
  enterpriseResourceServiceBatchRemoveAddressTypes,
  enterpriseResourceServiceBatchRemoveAssignees,
  enterpriseResourceServiceCommitEnterpriseResourceImport,
  enterpriseResourceServiceCreateEnterpriseResource,
  enterpriseResourceServiceCreateEnterpriseTagGroup,
  enterpriseResourceServiceDeleteEnterpriseResource,
  enterpriseResourceServiceGetEnterpriseResourceCapabilities,
  enterpriseResourceServiceGetEnterpriseResourceImageAccess,
  enterpriseResourceServiceListEnterpriseResources,
  enterpriseResourceServiceListEnterpriseTagGroups,
  enterpriseResourceServicePreviewEnterpriseResourceImport,
  enterpriseResourceServiceSearchEnterpriseResourceAssigneeOptions,
  enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions,
  enterpriseResourceServiceUpdateEnterpriseResource,
  enterpriseResourceServiceUpdateEnterpriseTagGroup,
} from '@/services/roncin/enterpriseResourceService';
import { toTableRequest, unwrapList } from '@/utils/api';
import { longRequestOptions } from '@/utils/requestTimeout';
import BatchAddressTypeModal from './components/BatchAddressTypeModal';
import BatchAssigneeModal from './components/BatchAssigneeModal';
import BatchAssociationModal from './components/BatchAssociationModal';
import ImageStorageCard from './components/ImageStorageCard';
import ResourceEditorModal from './components/ResourceEditorModal';
import ResourceImportModal from './components/ResourceImportModal';
import TagGroupModal from './components/TagGroupModal';
import {
  addressTypes,
  type EditorValues,
  importHeaders,
  partyTypes,
  remarkTypes,
  resourceTabs,
} from './resourceConstants';
import { useRegionOptions } from './useRegionOptions';

const EnterpriseResourcesPage: React.FC = () => {
  const access = useAccess();
  const [searchParams] = useSearchParams();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType>(undefined);
  const [form] = Form.useForm<EditorValues>();
  const queryTab = searchParams.get('tab');
  const [capabilities, setCapabilities] =
    useState<API.GetEnterpriseResourceCapabilitiesResponse>();
  const availableTabs = useMemo(
    () =>
      capabilities?.imageEnabled
        ? resourceTabs
        : resourceTabs.filter((item) => item.type !== 3),
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
  const [associationMode, setAssociationMode] = useState<'link' | 'unlink'>(
    'link',
  );
  const [associationPartners, setAssociationPartners] = useState<string[]>([]);
  const [addressTypeOpen, setAddressTypeOpen] = useState(false);
  const [addressTypeMode, setAddressTypeMode] = useState<'assign' | 'remove'>(
    'assign',
  );
  const [batchAddressTypes, setBatchAddressTypes] = useState<number[]>([]);
  const [assigneeOpen, setAssigneeOpen] = useState(false);
  const [assigneeMode, setAssigneeMode] = useState<'assign' | 'remove'>(
    'assign',
  );
  const [batchAssignees, setBatchAssignees] = useState<string[]>([]);
  const [partnerOptions, setPartnerOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [assigneeOptions, setAssigneeOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [tagGroups, setTagGroups] = useState<API.EnterpriseTagGroup[]>([]);
  const [groupOpen, setGroupOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<API.EnterpriseTagGroup>();
  const [groupForm] = Form.useForm();
  const [imageFiles, setImageFiles] = useState<UploadFile[]>([]);
  const [importOpen, setImportOpen] = useState(false);
  const [importFiles, setImportFiles] = useState<UploadFile[]>([]);
  const [importRows, setImportRows] = useState<API.EnterpriseResourceInput[]>(
    [],
  );
  const [importPreview, setImportPreview] =
    useState<API.PreviewEnterpriseResourceImportResponse>();
  const [importLoading, setImportLoading] = useState(false);
  const countryCode = Form.useWatch('countryCode', form);
  const provinceCode = Form.useWatch('provinceCode', form);
  const cityCode = Form.useWatch('cityCode', form);
  const { provinceOptions, cityOptions, districtOptions } = useRegionOptions({
    editorOpen,
    isAddressTab: active.type === 1,
    countryCode,
    provinceCode,
    cityCode,
  });

  useEffect(() => {
    void enterpriseResourceServiceGetEnterpriseResourceCapabilities().then(
      setCapabilities,
    );
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
    const response =
      await enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions({
        page: 1,
        pageSize: 50,
        keyword,
      });
    setPartnerOptions(
      unwrapList(response).flatMap((item) =>
        item.id
          ? [
              {
                value: item.id,
                label: `${item.code ?? ''} ${item.name ?? ''}`.trim(),
              },
            ]
          : [],
      ),
    );
  }, []);

  const searchAssignees = useCallback(async (keyword = '') => {
    const response =
      await enterpriseResourceServiceSearchEnterpriseResourceAssigneeOptions({
        page: 1,
        pageSize: 50,
        keyword,
      });
    setAssigneeOptions(
      unwrapList(response).flatMap((item) =>
        item.id
          ? [
              {
                value: item.id,
                label: item.displayName ?? item.username ?? item.id,
              },
            ]
          : [],
      ),
    );
  }, []);

  const openEditor = (record?: API.EnterpriseResource) => {
    setEditing(record);
    setImageFiles([]);
    if (!record) {
      form.resetFields();
      form.setFieldsValue({
        enabled: true,
        sortOrder: 0,
        countryCode: 'CN',
        partnerIds: [],
      });
    } else {
      form.setFieldsValue({
        shortName: record.shortName ?? '',
        enabled: record.enabled ?? true,
        sortOrder: record.sortOrder ?? 0,
        partnerIds: record.partnerIds ?? [],
        contactName: record.address?.contactName ?? record.party?.contactName,
        contactPhone:
          record.address?.contactPhone ?? record.party?.contactPhone,
        countryCode:
          record.address?.countryCode ?? record.party?.countryCode ?? 'CN',
        provinceCode: record.address?.provinceCode,
        cityCode: record.address?.cityCode,
        districtCode: record.address?.districtCode,
        addressDetail: record.address?.addressDetail,
        addressRemark: record.address?.remark,
        addressTypes: record.addressTypes,
        assigneeIds: record.assigneeIds,
        remarkType: record.remark?.remarkType,
        content: record.remark?.content,
        companyName: record.party?.companyName,
        businessCode: record.party?.businessCode,
        partyAddress: record.party?.address,
        email: record.party?.email,
        taxIdentifier: record.party?.taxIdentifier,
        aeoCode: record.party?.aeoCode,
        customDisplay: record.party?.customDisplay,
        displayContent: record.party?.displayContent,
        partyRemark: record.party?.remark,
        groupId: record.tag?.groupId,
      });
      void searchPartners();
    }
    setEditorOpen(true);
  };

  const toInput = (values: EditorValues): API.EnterpriseResourceInput => {
    const input: API.EnterpriseResourceInput = {
      resourceType: active.type,
      shortName: values.shortName.trim(),
      enabled: values.enabled,
      sortOrder: values.sortOrder ?? 0,
      partnerAssociations: { partnerIds: values.partnerIds ?? [] },
    };
    if (active.type === 1) {
      input.address = {
        contactName: values.contactName,
        contactPhone: values.contactPhone,
        countryCode: values.countryCode,
        provinceCode: values.provinceCode,
        cityCode: values.cityCode,
        districtCode: values.districtCode,
        addressDetail: values.addressDetail,
        remark: values.addressRemark,
      };
      input.addressTypes = values.addressTypes ?? [];
      input.assigneeIds = values.assigneeIds ?? [];
    } else if (active.type === 2) {
      input.remark = { remarkType: values.remarkType, content: values.content };
    } else if (partyTypes.has(active.type)) {
      input.party = {
        companyName: values.companyName,
        businessCode: values.businessCode,
        address: values.partyAddress,
        countryCode: values.countryCode,
        contactName: values.contactName,
        contactPhone: values.contactPhone,
        email: values.email,
        taxIdentifier: values.taxIdentifier,
        aeoCode: values.aeoCode,
        customDisplay: values.customDisplay,
        displayContent: values.displayContent,
        remark: values.partyRemark,
      };
    } else if (active.type === 4) {
      input.tag = { groupId: values.groupId };
    } else if (active.type === 3 && imageFiles[0]) {
      const file = imageFiles[0].originFileObj;
      input.image = {
        fileName: file?.name,
        mimeType: file?.type,
        fileSize: String(file?.size ?? 0),
        objectKey: imageFiles[0].response?.objectKey,
        checksum: imageFiles[0].response?.checksum,
      };
    }
    return input;
  };

  const saveResource = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const input = toInput(values);
      if (editing?.id)
        await enterpriseResourceServiceUpdateEnterpriseResource(
          { id: editing.id },
          { id: editing.id, resource: input },
        );
      else
        await enterpriseResourceServiceCreateEnterpriseResource({
          resource: input,
        });
      message.success(editing ? '资源已更新' : '资源已创建');
      setEditorOpen(false);
      actionRef.current?.reload();
    } finally {
      setSaving(false);
    }
  };

  const columns = useMemo<ProColumns<API.EnterpriseResource>[]>(() => {
    const detailColumns: ProColumns<API.EnterpriseResource>[] = [];
    if (active.type === 1)
      detailColumns.push(
        {
          title: '详细地址',
          dataIndex: ['address', 'addressDetail'],
          ellipsis: true,
        },
        { title: '联系人', dataIndex: ['address', 'contactName'] },
      );
    if (active.type === 2)
      detailColumns.push(
        {
          title: '备注类型',
          render: (_, record) =>
            remarkTypes.find((item) => item.value === record.remark?.remarkType)
              ?.label ?? '-',
        },
        { title: '备注内容', dataIndex: ['remark', 'content'], ellipsis: true },
      );
    if (partyTypes.has(active.type))
      detailColumns.push(
        {
          title: '企业名称',
          dataIndex: ['party', 'companyName'],
          ellipsis: true,
        },
        { title: '企业代码', dataIndex: ['party', 'businessCode'] },
        { title: '国家', dataIndex: ['party', 'countryCode'], width: 72 },
      );
    if (active.type === 4)
      detailColumns.push({
        title: '标签组',
        render: (_, record) =>
          tagGroups.find((group) => group.id === record.tag?.groupId)?.name ??
          '-',
      });
    if (active.type === 3)
      detailColumns.push(
        {
          title: '文件',
          render: (_, record) => (
            <Button
              type="link"
              onClick={async () => {
                if (!record.id) return;
                const response =
                  await enterpriseResourceServiceGetEnterpriseResourceImageAccess(
                    { id: record.id },
                  );
                if (response.url)
                  window.open(response.url, '_blank', 'noopener,noreferrer');
              }}
            >
              {record.image?.fileName ?? '预览'}
            </Button>
          ),
        },
        {
          title: '大小',
          render: (_, record) =>
            record.image?.fileSize
              ? `${(Number(record.image.fileSize) / 1024 / 1024).toFixed(2)} MiB`
              : '-',
        },
      );
    return [
      { title: '关键词', dataIndex: 'keyword', hideInTable: true },
      {
        title: '关联状态',
        dataIndex: 'linked',
        hideInTable: true,
        valueType: 'select',
        fieldProps: {
          options: [
            { label: '已关联', value: 'true' },
            { label: '独立资源', value: 'false' },
          ],
        },
      },
      {
        title: '关联企业',
        dataIndex: 'partnerId',
        hideInTable: true,
        valueType: 'select',
        fieldProps: {
          showSearch: {
            filterOption: false,
            onSearch: (value: string) => void searchPartners(value),
          },
          onFocus: () => void searchPartners(),
          options: partnerOptions,
        },
      },
      ...(active.type === 1
        ? [
            {
              title: '地址类型',
              dataIndex: 'addressType',
              hideInTable: true,
              valueType: 'select' as const,
              fieldProps: { options: addressTypes },
            },
            {
              title: '关联人员',
              dataIndex: 'assigneeId',
              hideInTable: true,
              valueType: 'select' as const,
              fieldProps: {
                showSearch: {
                  filterOption: false,
                  onSearch: (value: string) => void searchAssignees(value),
                },
                onFocus: () => void searchAssignees(),
                options: assigneeOptions,
              },
            },
          ]
        : []),
      {
        title: '简称/名称',
        dataIndex: 'shortName',
        ellipsis: true,
        hideInSearch: true,
        sorter: true,
      },
      ...detailColumns,
      {
        title: '关联企业',
        dataIndex: 'partnerIds',
        render: (_, record) => (
          <Tag color={record.partnerIds?.length ? 'blue' : 'default'}>
            {record.partnerIds?.length
              ? `${record.partnerIds.length} 家`
              : '独立资源'}
          </Tag>
        ),
      },
      {
        title: '状态',
        dataIndex: 'enabled',
        width: 80,
        valueType: 'select',
        fieldProps: {
          options: [
            { label: '启用', value: 'true' },
            { label: '停用', value: 'false' },
          ],
        },
        render: (_, record) => (
          <Tag color={record.enabled ? 'success' : 'default'}>
            {record.enabled ? '启用' : '停用'}
          </Tag>
        ),
      },
      {
        title: '更新时间',
        dataIndex: 'updatedAt',
        valueType: 'dateTime',
        width: 170,
        sorter: true,
      },
      {
        title: '操作',
        valueType: 'option',
        fixed: 'right',
        width: 120,
        render: (_, record) => [
          access.canUpdateEnterpriseResources && active.type !== 3 && (
            <Button
              key="edit"
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => openEditor(record)}
            >
              编辑
            </Button>
          ),
          access.canDeleteEnterpriseResources && (
            <Popconfirm
              key="delete"
              title="确认删除此资源？"
              onConfirm={async () => {
                if (!record.id) return;
                await enterpriseResourceServiceDeleteEnterpriseResource({
                  id: record.id,
                });
                message.success('资源已删除');
                actionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                删除
              </Button>
            </Popconfirm>
          ),
        ],
      },
    ];
  }, [
    access,
    active.type,
    assigneeOptions,
    message,
    partnerOptions,
    searchAssignees,
    searchPartners,
    tagGroups,
  ]);

  const columnSettings = useColumnSettings<ProColumns<API.EnterpriseResource>>({
    tableKey: 'enterprise-resources:list',
    columns,
  });

  const submitAssociation = async () => {
    if (!selectedKeys.length || !associationPartners.length) return;
    const body = {
      resourceIds: selectedKeys.map(String),
      partnerIds: associationPartners,
    };
    if (associationMode === 'link')
      await enterpriseResourceServiceBatchCreateAssociations(body);
    else await enterpriseResourceServiceBatchDeleteAssociations(body);
    message.success(associationMode === 'link' ? '已关联企业' : '已解除关联');
    setAssociationOpen(false);
    setSelectedKeys([]);
    actionRef.current?.reload();
  };

  const submitAddressTypes = async () => {
    if (!selectedKeys.length || !batchAddressTypes.length) return;
    const body = {
      resourceIds: selectedKeys.map(String),
      addressTypes: batchAddressTypes,
    };
    if (addressTypeMode === 'assign')
      await enterpriseResourceServiceBatchAssignAddressTypes(body);
    else await enterpriseResourceServiceBatchRemoveAddressTypes(body);
    message.success(
      addressTypeMode === 'assign' ? '已分配地址类型' : '已移除地址类型',
    );
    setAddressTypeOpen(false);
    setSelectedKeys([]);
    actionRef.current?.reload();
  };

  const submitAssignees = async () => {
    if (!selectedKeys.length || !batchAssignees.length) return;
    const body = {
      resourceIds: selectedKeys.map(String),
      assigneeIds: batchAssignees,
    };
    if (assigneeMode === 'assign')
      await enterpriseResourceServiceBatchAssignAssignees(body);
    else await enterpriseResourceServiceBatchRemoveAssignees(body);
    message.success(
      assigneeMode === 'assign' ? '已关联人员' : '已移除关联人员',
    );
    setAssigneeOpen(false);
    setSelectedKeys([]);
    actionRef.current?.reload();
  };

  const saveGroup = async () => {
    const values = await groupForm.validateFields();
    const color =
      typeof values.color === 'string'
        ? values.color
        : values.color?.toHexString();
    if (editingGroup?.id)
      await enterpriseResourceServiceUpdateEnterpriseTagGroup(
        { id: editingGroup.id },
        {
          id: editingGroup.id,
          group: { name: values.name, color, sortOrder: values.sortOrder ?? 0 },
        },
      );
    else
      await enterpriseResourceServiceCreateEnterpriseTagGroup({
        group: { name: values.name, color, sortOrder: values.sortOrder ?? 0 },
      });
    message.success('标签组已保存');
    setGroupOpen(false);
    await loadTagGroups();
  };

  const previewImport = async () => {
    setImportLoading(true);
    try {
      setImportPreview(
        await enterpriseResourceServicePreviewEnterpriseResourceImport(
          { resourceType: active.type, rows: importRows },
          longRequestOptions,
        ),
      );
    } finally {
      setImportLoading(false);
    }
  };

  const commitImport = async () => {
    setImportLoading(true);
    try {
      const response =
        await enterpriseResourceServiceCommitEnterpriseResourceImport(
          {
            resourceType: active.type,
            rows: importRows,
            overwriteConflicts: (importPreview?.conflictCount ?? 0) > 0,
          },
          longRequestOptions,
        );
      message.success(
        `已新增 ${response.createdCount ?? 0} 条，更新 ${response.updatedCount ?? 0} 条资源`,
      );
      setImportOpen(false);
      setImportPreview(undefined);
      setImportFiles([]);
      setImportRows([]);
      actionRef.current?.reload();
    } finally {
      setImportLoading(false);
    }
  };

  const downloadImportTemplate = () => {
    const content = `\uFEFF${importHeaders.join(',')}\r\n示例主体,上海示例公司,SAMPLE001,上海市浦东新区,CN,张三,13800000000,example@example.com,,`;
    const url = URL.createObjectURL(
      new Blob([content], { type: 'text/csv;charset=utf-8' }),
    );
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `${active.label.replace('管理', '')}导入模板.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return (
    <PageContainer title="配置管理" subTitle="组织级企业资源备忘录">
      <Tabs
        activeKey={activeTab}
        items={availableTabs.map(({ key, label }) => ({ key, label }))}
        onChange={(tab) =>
          history.replace(`/enterprise-resources/config?tab=${tab}`)
        }
        tabBarStyle={{
          marginBottom: 16,
          backgroundColor: '#ffffff',
          padding: '0 16px',
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.03)',
        }}
      />
      {active.type === 3 && <ImageStorageCard capabilities={capabilities} />}
      <ProTable<API.EnterpriseResource>
        actionRef={actionRef}
        rowKey="id"
        columns={columnSettings.columns}
        search={{ labelWidth: 'auto' }}
        cardProps={{ style: { borderRadius: 8, border: '1px solid #f0f0f0' } }}
        rowSelection={{
          selectedRowKeys: selectedKeys,
          preserveSelectedRowKeys: true,
          onChange: setSelectedKeys,
        }}
        request={async (params, sort) => {
          const sortEntry = Object.entries(sort ?? {})[0];
          const sortBy =
            sortEntry?.[0] === 'shortName'
              ? 'short_name'
              : sortEntry?.[0] === 'updatedAt'
                ? 'updated_at'
                : undefined;
          const response =
            await enterpriseResourceServiceListEnterpriseResources({
              resourceType: active.type,
              page: params.current,
              pageSize: params.pageSize,
              keyword: params.keyword as string | undefined,
              linked:
                params.linked === 'true'
                  ? true
                  : params.linked === 'false'
                    ? false
                    : undefined,
              enabled:
                params.enabled === 'true'
                  ? true
                  : params.enabled === 'false'
                    ? false
                    : undefined,
              partnerId: params.partnerId as string | undefined,
              addressType: params.addressType as number | undefined,
              assigneeId: params.assigneeId as string | undefined,
              sortBy,
              sortOrder:
                sortEntry?.[1] === 'descend'
                  ? 'desc'
                  : sortEntry
                    ? 'asc'
                    : undefined,
            });
          return toTableRequest(response);
        }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1100 }}
        options={{ reload: true, density: true, setting: false }}
        toolBarRender={() => [
          columnSettings.entry,
          active.type === 4 && access.canCreateEnterpriseResources && (
            <Button
              key="groups"
              icon={<TagsOutlined />}
              onClick={() => {
                setEditingGroup(undefined);
                groupForm.resetFields();
                setGroupOpen(true);
              }}
            >
              标签组
            </Button>
          ),
          partyTypes.has(active.type) &&
            access.canCreateEnterpriseResources && (
              <Button
                key="import"
                icon={<UploadOutlined />}
                onClick={() => {
                  setImportOpen(true);
                  setImportPreview(undefined);
                  setImportFiles([]);
                  setImportRows([]);
                }}
              >
                批量导入
              </Button>
            ),
          active.type !== 2 &&
            active.type !== 3 &&
            active.type !== 4 &&
            access.canUpdateEnterpriseResources && (
              <Button
                key="link"
                disabled={!selectedKeys.length}
                icon={<LinkOutlined />}
                onClick={() => {
                  setAssociationMode('link');
                  setAssociationPartners([]);
                  void searchPartners();
                  setAssociationOpen(true);
                }}
              >
                批量关联
              </Button>
            ),
          active.type !== 2 &&
            active.type !== 3 &&
            active.type !== 4 &&
            access.canUpdateEnterpriseResources && (
              <Button
                key="unlink"
                disabled={!selectedKeys.length}
                onClick={() => {
                  setAssociationMode('unlink');
                  setAssociationPartners([]);
                  void searchPartners();
                  setAssociationOpen(true);
                }}
              >
                解除关联
              </Button>
            ),
          active.type === 1 && access.canUpdateEnterpriseResources && (
            <Button
              key="address-type"
              disabled={!selectedKeys.length}
              onClick={() => {
                setAddressTypeMode('assign');
                setBatchAddressTypes([]);
                setAddressTypeOpen(true);
              }}
            >
              批量设置地址类型
            </Button>
          ),
          active.type === 1 && access.canUpdateEnterpriseResources && (
            <Button
              key="assignee"
              disabled={!selectedKeys.length}
              onClick={() => {
                setAssigneeMode('assign');
                setBatchAssignees([]);
                void searchAssignees();
                setAssigneeOpen(true);
              }}
            >
              批量关联人员
            </Button>
          ),
          access.canCreateEnterpriseResources && (
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openEditor()}
            >
              新建{active.label.replace('管理', '')}
            </Button>
          ),
        ]}
      />

      <ResourceEditorModal
        active={active}
        open={editorOpen}
        setOpen={setEditorOpen}
        editing={editing}
        saving={saving}
        form={form}
        onSave={() => void saveResource()}
        imageFiles={imageFiles}
        setImageFiles={setImageFiles}
        capabilities={capabilities}
        partnerOptions={partnerOptions}
        searchPartners={searchPartners}
        assigneeOptions={assigneeOptions}
        searchAssignees={searchAssignees}
        provinceOptions={provinceOptions}
        cityOptions={cityOptions}
        districtOptions={districtOptions}
        countryCode={countryCode}
        provinceCode={provinceCode}
        cityCode={cityCode}
        tagGroups={tagGroups}
        message={message}
      />

      <BatchAssociationModal
        open={associationOpen}
        setOpen={setAssociationOpen}
        mode={associationMode}
        partners={associationPartners}
        setPartners={setAssociationPartners}
        partnerOptions={partnerOptions}
        searchPartners={searchPartners}
        onOk={() => void submitAssociation()}
      />
      <BatchAddressTypeModal
        open={addressTypeOpen}
        setOpen={setAddressTypeOpen}
        mode={addressTypeMode}
        setMode={setAddressTypeMode}
        batchAddressTypes={batchAddressTypes}
        setBatchAddressTypes={setBatchAddressTypes}
        onOk={() => void submitAddressTypes()}
      />
      <BatchAssigneeModal
        open={assigneeOpen}
        setOpen={setAssigneeOpen}
        mode={assigneeMode}
        setMode={setAssigneeMode}
        batchAssignees={batchAssignees}
        setBatchAssignees={setBatchAssignees}
        assigneeOptions={assigneeOptions}
        searchAssignees={searchAssignees}
        onOk={() => void submitAssignees()}
      />

      <TagGroupModal
        open={groupOpen}
        setOpen={setGroupOpen}
        editingGroup={editingGroup}
        setEditingGroup={setEditingGroup}
        groupForm={groupForm}
        tagGroups={tagGroups}
        loadTagGroups={loadTagGroups}
        onSave={() => void saveGroup()}
      />
      <ResourceImportModal
        active={active}
        open={importOpen}
        setOpen={setImportOpen}
        files={importFiles}
        setFiles={setImportFiles}
        rows={importRows}
        setRows={setImportRows}
        preview={importPreview}
        setPreview={setImportPreview}
        loading={importLoading}
        onPreview={() => void previewImport()}
        onCommit={() => void commitImport()}
        onDownloadTemplate={downloadImportTemplate}
        message={message}
      />
    </PageContainer>
  );
};

export default EnterpriseResourcesPage;
