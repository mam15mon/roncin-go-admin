import { FileDoneOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Drawer,
  Popconfirm,
  Space,
  Typography,
} from 'antd';
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  makeValueEnum,
  orderReleasePodStatusMeta,
  statusTag,
  statusText,
} from '@/constants/statusMeta';
import { OrderBusinessType, SeaDocumentType } from '@/enums.generated';
import {
  orderReleasePodServiceAddReleasePod,
  orderReleasePodServiceListReleasePods,
  orderReleasePodServiceRemoveReleasePod,
  orderReleasePodServiceTransitionReleasePodStatus,
  orderReleasePodServiceUpdateReleasePod,
} from '@/services/roncin/orderReleasePodService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { seaDocumentServiceGetSeaOrderDocuments } from '@/services/roncin/seaDocumentService';
import { toTableRequest, unwrapList } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { notifyReleasePodsChanged } from './release-pod-events';

const { Text } = Typography;

type ReleasePodFormValues = {
  releaseNo?: string;
  podNo?: string;
  documentReference?: string;
  note?: string;
};

export type ReleasePodDocumentOption = {
  label: string;
  value: string;
  shippingDocumentId?: string;
  seaDocumentType?: number;
  seaDocumentId?: string;
};

export function buildSeaReleasePodDocumentOptions(
  documents?: API.SeaOrderDocuments,
): ReleasePodDocumentOption[] {
  const options: ReleasePodDocumentOption[] = [];
  if (documents?.masterBill?.id) {
    options.push({
      label: `MBL: ${documents.masterBill.masterNo || '-'}`,
      value: `mbl:${documents.masterBill.id}`,
      seaDocumentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
      seaDocumentId: documents.masterBill.id,
    });
  }
  for (const houseBill of documents?.houseBills ?? []) {
    if (!houseBill.id) continue;
    options.push({
      label: `HBL: ${houseBill.houseNo || '-'}`,
      value: `hbl:${houseBill.id}`,
      seaDocumentType: SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL,
      seaDocumentId: houseBill.id,
    });
  }
  return options;
}

export function buildLegacyReleasePodDocumentOptions(
  documents: API.OrderShippingDocument[],
): ReleasePodDocumentOption[] {
  return documents
    .filter((document) => document.id)
    .map((document) => ({
      label: `分单: ${document.houseNo || '-'}`,
      value: `legacy:${document.id}`,
      shippingDocumentId: document.id,
    }));
}

export function getReleasePodDocumentValue(record: API.OrderReleasePod) {
  if (record.shippingDocumentId) {
    return `legacy:${record.shippingDocumentId}`;
  }
  if (
    record.seaDocumentType ===
      SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL &&
    record.seaDocumentId
  ) {
    return `mbl:${record.seaDocumentId}`;
  }
  if (
    record.seaDocumentType ===
      SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL &&
    record.seaDocumentId
  ) {
    return `hbl:${record.seaDocumentId}`;
  }
  return undefined;
}

type ReleasePodPanelProps = {
  canManage: boolean;
};

export type ReleasePodPanelRef = {
  open: (order: API.Order) => void;
};

export function getReleasePodTransition(record?: API.OrderReleasePod) {
  const status = record?.status;
  const toStatus = record?.allowedTargetStatuses?.[0];
  if (!status || !toStatus) return undefined;
  return {
    currentText: statusText(orderReleasePodStatusMeta, status, '未知状态'),
    nextText: statusText(orderReleasePodStatusMeta, toStatus, '未知状态'),
    toStatus,
  };
}

const ReleasePodPanel = forwardRef<ReleasePodPanelRef, ReleasePodPanelProps>(
  function ReleasePodPanel({ canManage }, ref) {
    const actionRef = useRef<ActionType | undefined>(undefined);
    const formRef = useRef<ProFormInstance | undefined>(undefined);
    const { message } = App.useApp();
    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();
    const [documentOptions, setDocumentOptions] = useState<
      ReleasePodDocumentOption[]
    >([]);
    const [documentsError, setDocumentsError] = useState('');
    const [editingRecord, setEditingRecord] = useState<API.OrderReleasePod>();
    const activeOrderIdRef = useRef<string | undefined>(undefined);

    useEffect(() => {
      if (!canManage) setModalOpen(false);
    }, [canManage]);

    const ensureCanManage = () => {
      if (canManage) return true;
      message.warning('订单当前不可编辑，请刷新锁定状态后重试');
      return false;
    };

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDocumentOptions([]);
        setDocumentsError('');
        setDrawerOpen(true);
        const orderId = record.id as string;
        activeOrderIdRef.current = orderId;
        const loadDocuments =
          record.businessType === OrderBusinessType.BUSINESS_TYPE_SE
            ? seaDocumentServiceGetSeaOrderDocuments({ orderId }).then(
                (response) =>
                  buildSeaReleasePodDocumentOptions(response.data),
              )
            : orderShippingDocumentServiceListShippingDocuments({
                orderId,
              }).then((response) =>
                buildLegacyReleasePodDocumentOptions(unwrapList(response)),
              );
        loadDocuments
          .then((response) => {
            if (activeOrderIdRef.current === orderId) {
              setDocumentOptions(response);
            }
          })
          .catch((error: Error) => {
            if (activeOrderIdRef.current === orderId) {
              setDocumentOptions([]);
              setDocumentsError(error.message || '关联提单加载失败');
            }
          });
      },
    }));

    const documentMap = Object.fromEntries(
      documentOptions.map((option) => [option.value, option.label]),
    );

    const openCreate = () => {
      setEditingRecord(undefined);
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const openEdit = (record: API.OrderReleasePod) => {
      setEditingRecord(record);
      formRef.current?.setFieldsValue({
        releaseNo: record.releaseNo,
        podNo: record.podNo,
        documentReference: getReleasePodDocumentValue(record),
        note: record.note,
      });
      setModalOpen(true);
    };

    const columns: ProColumns<API.OrderReleasePod>[] = [
      {
        title: '放货编号',
        dataIndex: 'releaseNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.releaseNo || '-'}
          </Text>
        ),
      },
      {
        title: '回单编号',
        dataIndex: 'podNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.podNo || '-'}
          </Text>
        ),
      },
      {
        title: '关联提单',
        dataIndex: 'seaDocumentId',
        ellipsis: true,
        render: (_, record) => {
          const value = getReleasePodDocumentValue(record);
          return (value && documentMap[value]) || '-';
        },
      },
      {
        title: '状态',
        dataIndex: 'status',
        valueType: 'select',
        valueEnum: makeValueEnum(orderReleasePodStatusMeta),
        render: (_, record) =>
          statusTag(orderReleasePodStatusMeta, record.status ?? 0),
      },
      {
        title: '签收时间',
        dataIndex: 'signedAt',
        valueType: 'dateTime',
        width: 170,
        render: (_, record) =>
          record.signedAt ? (
            formatDate(record.signedAt)
          ) : (
            <Text type="secondary">-</Text>
          ),
      },
      {
        title: '签收人',
        dataIndex: 'signedBy',
        copyable: true,
        ellipsis: true,
        render: (_, record) =>
          record.signedBy ? (
            <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
              {record.signedBy}
            </Text>
          ) : (
            <Text type="secondary">-</Text>
          ),
      },
      {
        title: '备注',
        dataIndex: 'note',
        ellipsis: true,
        render: (_, record) => record.note || '-',
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        valueType: 'dateTime',
        width: 170,
        render: (_, record) => formatDate(record.createdAt),
      },
      {
        title: '操作',
        valueType: 'option',
        search: false,
        width: 170,
        fixed: 'right',
        render: (_, record) => {
          const transition = getReleasePodTransition(record);
          if (!canManage) return null;
          return (
            <Space size="small">
              <Button type="link" size="small" onClick={() => openEdit(record)}>
                编辑
              </Button>
              {transition && (
                <Popconfirm
                  title={`确定将放货凭证状态从「${transition.currentText}」流转为「${transition.nextText}」？`}
                  onConfirm={async () => {
                    if (
                      !ensureCanManage() ||
                      !order?.id ||
                      !record.id ||
                      !record.status
                    )
                      return;
                    await orderReleasePodServiceTransitionReleasePodStatus(
                      { orderId: order.id, id: record.id },
                      {
                        orderId: order.id,
                        id: record.id,
                        expectedStatus: record.status,
                        toStatus: transition.toStatus,
                      },
                    );
                    message.success('流转放货凭证状态成功');
                    actionRef.current?.reload();
                    notifyReleasePodsChanged(order.id);
                  }}
                >
                  <Button type="link" size="small">
                    流转
                  </Button>
                </Popconfirm>
              )}
              <Popconfirm
                title="确定移除该放货凭证？"
                onConfirm={async () => {
                  if (!ensureCanManage() || !order?.id || !record.id) return;
                  await orderReleasePodServiceRemoveReleasePod({
                    orderId: order.id,
                    id: record.id,
                  });
                  message.success('移除放货凭证成功');
                  actionRef.current?.reload();
                  notifyReleasePodsChanged(order.id);
                }}
              >
                <Button type="link" danger size="small">
                  删除
                </Button>
              </Popconfirm>
            </Space>
          );
        },
      },
    ];

    return (
      <>
        <Drawer
          title={
            order
              ? `订单放货凭证 (POD) - ${order.orderNo || order.id}`
              : '订单放货凭证 (POD)'
          }
          open={drawerOpen}
          onClose={() => {
            activeOrderIdRef.current = undefined;
            setDrawerOpen(false);
            setOrder(undefined);
            setDocumentOptions([]);
            setDocumentsError('');
          }}
          size={920}
          destroyOnHidden
        >
          {documentsError && (
            <Alert
              type="error"
              showIcon
              title="关联提单加载失败"
              description={documentsError}
              style={{ marginBottom: 16 }}
            />
          )}
          {order?.id && (
            <ProTable<API.OrderReleasePod>
              headerTitle={
                <Space size={8}>
                  <FileDoneOutlined style={{ color: '#52c41a' }} />
                  <span>放货回单记录</span>
                </Space>
              }
              actionRef={actionRef}
              rowKey="id"
              columns={columns}
              bordered
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderReleasePodServiceListReleasePods({
                  orderId: order.id as string,
                });
                return toTableRequest(response);
              }}
              toolBarRender={() => [
                canManage && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreate}
                  >
                    添加放货凭证
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<ReleasePodFormValues>
          title={editingRecord ? '编辑放货凭证' : '添加放货凭证'}
          open={modalOpen}
          formRef={formRef}
          initialValues={editingRecord}
          modalProps={{
            destroyOnHidden: true,
            width: 560,
            onCancel: () => setModalOpen(false),
          }}
          onOpenChange={setModalOpen}
          onFinish={async (values) => {
            if (!ensureCanManage() || !order?.id) return false;
            const input = {
              orderId: order.id,
              releaseNo: values.releaseNo?.trim() || undefined,
              podNo: values.podNo?.trim() || undefined,
              note: values.note?.trim() || undefined,
            };
            const selectedDocument = values.documentReference
              ? documentOptions.find(
                  (option) => option.value === values.documentReference,
                )
              : undefined;
            if (values.documentReference && !selectedDocument) {
              message.error('关联提单已变化，请刷新后重试');
              return false;
            }
            const documentInput = selectedDocument
              ? {
                  shippingDocumentId: selectedDocument.shippingDocumentId,
                  seaDocumentType: selectedDocument.seaDocumentType,
                  seaDocumentId: selectedDocument.seaDocumentId,
                }
              : {};
            if (editingRecord?.id) {
              await orderReleasePodServiceUpdateReleasePod(
                { orderId: order.id, id: editingRecord.id },
                { ...input, ...documentInput, id: editingRecord.id },
              );
              message.success('更新放货凭证成功');
            } else {
              await orderReleasePodServiceAddReleasePod(
                { orderId: order.id },
                { ...input, ...documentInput },
              );
              message.success('添加放货凭证成功');
            }
            setModalOpen(false);
            actionRef.current?.reload();
            notifyReleasePodsChanged(order.id);
            return true;
          }}
        >
          <ProFormText
            name="releaseNo"
            label="放货编号"
            placeholder="请输入放货编号 (可选)"
          />
          <ProFormText
            name="podNo"
            label="回单编号 (POD No)"
            placeholder="请输入回单编号 (可选)"
          />
          <ProFormSearchableSelect
            name="documentReference"
            label="关联提单"
            options={documentOptions}
            placeholder="请选择关联提单 (可选)"
            disabled={Boolean(documentsError)}
          />
          <ProFormTextArea
            name="note"
            label="备注说明"
            placeholder="请输入备注说明 (可选)"
            fieldProps={{ maxLength: 500, showCount: true }}
          />
        </ModalForm>
      </>
    );
  },
);

export default ReleasePodPanel;
