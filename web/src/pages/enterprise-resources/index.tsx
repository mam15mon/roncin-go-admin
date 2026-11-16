import {
  DeleteOutlined,
  EditOutlined,
  LinkOutlined,
  PlusOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { paginationSizeChanger } from '@root/config/pagination';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { UploadFile } from 'antd';
import { Alert, App, Button, Form, Tabs, Tag } from 'antd';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useSearchParams } from 'react-router';
import { useAccess } from '@/app/access';
import {
  AsyncPopconfirm,
  ResponsiveProTable as ProTable,
} from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { defaultSelectFilterOption } from '@/components/ui/searchable-select/utils';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import { history } from '@/router/history';
import {
  enterpriseResourceServiceBatchAssignAddressTypes,
  enterpriseResourceServiceBatchAssignAssignees,
  enterpriseResourceServiceBatchCreateAssociations,
  enterpriseResourceServiceBatchDeleteAssociations,
  enterpriseResourceServiceBatchRemoveAddressTypes,
  enterpriseResourceServiceBatchRemoveAssignees,
  enterpriseResourceServiceCommitEnterpriseResourceImport,
  enterpriseResourceServiceCreateEnterpriseResource,
  enterpriseResourceServiceDeleteEnterpriseResource,
  enterpriseResourceServiceGetEnterpriseResourceCapabilities,
  enterpriseResourceServiceGetEnterpriseResourceImageAccess,
  enterpriseResourceServiceListEnterpriseResources,
  enterpriseResourceServicePreviewEnterpriseResourceImport,
  enterpriseResourceServiceSearchEnterpriseResourceAssigneeOptions,
  enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions,
  enterpriseResourceServiceUpdateEnterpriseResource,
} from '@/services/roncin/enterpriseResourceService';
import {
  ensureListResponse,
  isWriteResponse,
  toTableRequest,
  unwrapList,
} from '@/utils/api';
import { longRequestOptions } from '@/utils/requestTimeout';
import BatchAddressTypeModal from './components/BatchAddressTypeModal';
import BatchAssigneeModal from './components/BatchAssigneeModal';
import BatchAssociationModal from './components/BatchAssociationModal';
import ImageStorageCard from './components/ImageStorageCard';
import ResourceEditorModal from './components/ResourceEditorModal';
import ResourceImportModal from './components/ResourceImportModal';
import {
  addressTypes,
  type EditorValues,
  importHeaders,
  partyTypes,
  type ResourceTab,
  remarkTypes,
  resourceTabs,
} from './resourceConstants';
import { useRegionOptions } from './useRegionOptions';

type CandidateOption = { label: string; value: string };

/** 只保留已选项的历史名称，不把旧关键词的全部候选混进当前结果。 */
function withSelectedCandidates(
  current: CandidateOption[],
  history: CandidateOption[],
  selected: string[],
): CandidateOption[] {
  const ids = new Set(selected);
  return [
    ...new Map(
      [...history.filter((option) => ids.has(option.value)), ...current].map(
        (option) => [option.value, option],
      ),
    ).values(),
  ];
}

type ResourceTypePageProps = {
  active: ResourceTab;
  availableTabs: readonly ResourceTab[];
  capabilities?: API.GetEnterpriseResourceCapabilitiesResponse;
};

const ResourceTypePage: React.FC<ResourceTypePageProps> = ({
  active,
  availableTabs,
  capabilities,
}) => {
  const access = useAccess();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType>(undefined);
  const [form] = Form.useForm<EditorValues>();
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
  const [partnerKeyword, setPartnerKeyword] = useState<string>();
  const [assigneeKeyword, setAssigneeKeyword] = useState<string>();
  const [partnerFilterId, setPartnerFilterId] = useState<string>();
  const [assigneeFilterId, setAssigneeFilterId] = useState<string>();
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
    setSelectedKeys([]);
  }, [active.type]);

  const searchPartners = useCallback(
    (keyword = '') => setPartnerKeyword(keyword.trim()),
    [],
  );
  const searchAssignees = useCallback(
    (keyword = '') => setAssigneeKeyword(keyword.trim()),
    [],
  );
  const partnerQuery = useQuery({
    queryKey: [
      'enterprise-resources',
      'partner-options',
      { keyword: partnerKeyword },
    ],
    enabled: partnerKeyword !== undefined,
    meta: { errorMessage: '企业候选加载失败' },
    queryFn: async ({ signal }) => {
      const response =
        await enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions(
          { page: 1, pageSize: 50, keyword: partnerKeyword || undefined },
          { signal, skipErrorHandler: true },
        );
      return unwrapList(
        ensureListResponse(response, '企业候选加载失败'),
      ).flatMap((item) =>
        item.id
          ? [
              {
                value: item.id,
                label: `${item.code ?? ''} ${item.name ?? ''}`.trim(),
              },
            ]
          : [],
      );
    },
  });
  const assigneeQuery = useQuery({
    queryKey: [
      'enterprise-resources',
      'assignee-options',
      { keyword: assigneeKeyword },
    ],
    enabled: assigneeKeyword !== undefined,
    meta: { errorMessage: '人员候选加载失败' },
    queryFn: async ({ signal }) => {
      const response =
        await enterpriseResourceServiceSearchEnterpriseResourceAssigneeOptions(
          { page: 1, pageSize: 50, keyword: assigneeKeyword || undefined },
          { signal, skipErrorHandler: true },
        );
      return unwrapList(
        ensureListResponse(response, '人员候选加载失败'),
      ).flatMap((item) =>
        item.id
          ? [
              {
                value: item.id,
                label: item.displayName ?? item.username ?? item.id,
              },
            ]
          : [],
      );
    },
  });
  const editingPartners =
    Form.useWatch('partnerIds', { form, preserve: true }) ?? [];
  const editingAssignees =
    Form.useWatch('assigneeIds', { form, preserve: true }) ?? [];
  const partnerOptions = withSelectedCandidates(
    partnerQuery.data ?? [],
    queryClient
      .getQueriesData<CandidateOption[]>({
        queryKey: ['enterprise-resources', 'partner-options'],
      })
      .flatMap(([, options]) => options ?? []),
    [
      ...editingPartners,
      ...associationPartners,
      ...(partnerFilterId ? [partnerFilterId] : []),
    ],
  );
  const assigneeOptions = withSelectedCandidates(
    assigneeQuery.data ?? [],
    queryClient
      .getQueriesData<CandidateOption[]>({
        queryKey: ['enterprise-resources', 'assignee-options'],
      })
      .flatMap(([, options]) => options ?? []),
    [
      ...editingAssignees,
      ...batchAssignees,
      ...(assigneeFilterId ? [assigneeFilterId] : []),
    ],
  );

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
        enabled: Boolean(record.enabled),
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

  const saveResource = async (
    values: EditorValues,
  ): Promise<false | undefined> => {
    setSaving(true);
    try {
      const input = toInput(values);
      if (editing?.id) {
        const writeResponse1 =
          await enterpriseResourceServiceUpdateEnterpriseResource(
            { id: editing.id },
            { id: editing.id, resource: input },
          );
        if (!isWriteResponse(writeResponse1, true)) return false;
      } else {
        const writeResponse2 =
          await enterpriseResourceServiceCreateEnterpriseResource({
            resource: input,
          });
        if (!isWriteResponse(writeResponse2, true)) return false;
      }
      message.success(editing ? '资源已更新' : '资源已创建');
      setEditorOpen(false);
      actionRef.current?.reload();
    } catch (error) {
      message.error(getRequestErrorMessage(error, '资源保存失败，请稍后重试'));
      return false;
    } finally {
      setSaving(false);
    }

    return undefined;
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
          key: 'remarkType',
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
    if (active.type === 3)
      detailColumns.push(
        {
          title: '文件',
          key: 'fileName',
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
          key: 'fileSize',
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
          filterOption: defaultSelectFilterOption,
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
          onChange: (value?: string) => setPartnerFilterId(value),
        },
      },
      ...(active.type === 1
        ? [
            {
              title: '地址类型',
              dataIndex: 'addressType',
              hideInTable: true,
              valueType: 'select' as const,
              fieldProps: {
                options: addressTypes,
                filterOption: defaultSelectFilterOption,
              },
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
                onChange: (value?: string) => setAssigneeFilterId(value),
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
          filterOption: defaultSelectFilterOption,
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
            <AsyncPopconfirm
              key="delete"
              title="确认删除此资源？"
              onConfirm={async (): Promise<false | undefined> => {
                if (!record.id) return false;
                const writeResponse3 =
                  await enterpriseResourceServiceDeleteEnterpriseResource({
                    id: record.id,
                  });
                if (!isWriteResponse(writeResponse3)) return false;
                message.success('资源已删除');
                actionRef.current?.reload();

                return undefined;
              }}
            >
              <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                删除
              </Button>
            </AsyncPopconfirm>
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
  ]);

  const columnSettings = useColumnSettings<ProColumns<API.EnterpriseResource>>({
    // 各页签列集合不同，偏好按页签隔离，避免跨页签套用错位。
    tableKey: `enterprise-resources:list:${activeTab}`,
    columns,
  });

  const submitAssociation = async (): Promise<false | undefined> => {
    if (!selectedKeys.length || !associationPartners.length) return;
    const body = {
      resourceIds: selectedKeys.map(String),
      partnerIds: associationPartners,
    };
    const response =
      associationMode === 'link'
        ? await enterpriseResourceServiceBatchCreateAssociations(body)
        : await enterpriseResourceServiceBatchDeleteAssociations(body);
    if (!isWriteResponse(response)) return false;
    message.success(associationMode === 'link' ? '已关联企业' : '已解除关联');
    setAssociationOpen(false);
    setSelectedKeys([]);
    actionRef.current?.reload();

    return undefined;
  };

  const submitAddressTypes = async (): Promise<false | undefined> => {
    if (!selectedKeys.length || !batchAddressTypes.length) return;
    const body = {
      resourceIds: selectedKeys.map(String),
      addressTypes: batchAddressTypes,
    };
    const response =
      addressTypeMode === 'assign'
        ? await enterpriseResourceServiceBatchAssignAddressTypes(body)
        : await enterpriseResourceServiceBatchRemoveAddressTypes(body);
    if (!isWriteResponse(response)) return false;
    message.success(
      addressTypeMode === 'assign' ? '已分配地址类型' : '已移除地址类型',
    );
    setAddressTypeOpen(false);
    setSelectedKeys([]);
    actionRef.current?.reload();

    return undefined;
  };

  const submitAssignees = async (): Promise<false | undefined> => {
    if (!selectedKeys.length || !batchAssignees.length) return;
    const body = {
      resourceIds: selectedKeys.map(String),
      assigneeIds: batchAssignees,
    };
    const response =
      assigneeMode === 'assign'
        ? await enterpriseResourceServiceBatchAssignAssignees(body)
        : await enterpriseResourceServiceBatchRemoveAssignees(body);
    if (!isWriteResponse(response)) return false;
    message.success(
      assigneeMode === 'assign' ? '已关联人员' : '已移除关联人员',
    );
    setAssigneeOpen(false);
    setSelectedKeys([]);
    actionRef.current?.reload();

    return undefined;
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

  const commitImport = async (): Promise<false | undefined> => {
    const overwriteConflicts = (importPreview?.conflictCount ?? 0) > 0;
    if (
      !access.canCreateEnterpriseResources ||
      (overwriteConflicts &&
        (!access.canUpdateEnterpriseResources ||
          importPreview?.overwriteAllowed !== true))
    ) {
      message.warning('覆盖已有资源需要编辑资源权限，且每行只能匹配一个资源');
      return false;
    }
    setImportLoading(true);
    try {
      const response =
        await enterpriseResourceServiceCommitEnterpriseResourceImport(
          {
            resourceType: active.type,
            rows: importRows,
            overwriteConflicts,
          },
          longRequestOptions,
        );
      if (!isWriteResponse(response)) return false;
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

    return undefined;
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
        tableAlertRender={false}
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
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: paginationSizeChanger,
        }}
        options={{ reload: true, density: true, setting: false }}
        toolBarRender={() => [
          columnSettings.entry,
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
        onSave={saveResource}
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

      <ResourceImportModal
        active={active}
        open={importOpen}
        setOpen={setImportOpen}
        files={importFiles}
        setFiles={setImportFiles}
        rows={importRows}
        setRows={setImportRows}
        preview={importPreview}
        canOverwrite={access.canUpdateEnterpriseResources}
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

const EnterpriseResourcesPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const capabilitiesQuery = useQuery({
    queryKey: ['enterprise-resources', 'capabilities', {}],
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response =
        await enterpriseResourceServiceGetEnterpriseResourceCapabilities({
          signal,
          skipErrorHandler: true,
        });
      if (!response || response.success === false)
        throw new Error('企业资源能力加载失败');
      return response;
    },
  });
  const capabilities = capabilitiesQuery.data;
  const availableTabs = useMemo(
    () =>
      capabilities?.imageEnabled
        ? resourceTabs
        : resourceTabs.filter((item) => item.type !== 3),
    [capabilities?.imageEnabled],
  );
  const active =
    availableTabs.find((item) => item.key === searchParams.get('tab')) ??
    availableTabs[0];

  // 类型是请求、选择和所有编辑会话的边界，切换后卸载旧表格及其在途请求。
  return (
    <>
      {capabilitiesQuery.isError && (
        <Alert
          type="error"
          showIcon
          title="企业资源能力加载失败"
          action={
            <Button onClick={() => void capabilitiesQuery.refetch()}>
              重试
            </Button>
          }
        />
      )}
      <ResourceTypePage
        key={active.type}
        active={active}
        availableTabs={availableTabs}
        capabilities={capabilities}
      />
    </>
  );
};

export default EnterpriseResourcesPage;
