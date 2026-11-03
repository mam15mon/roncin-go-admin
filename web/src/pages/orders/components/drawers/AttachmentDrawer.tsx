import { UploadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProFormSelect } from '@ant-design/pro-components';
import type { UploadFile } from 'antd';
import {
  Alert,
  App,
  Button,
  Drawer,
  Form,
  Modal,
  Progress,
  Space,
  Tag,
  Upload,
} from 'antd';
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import { useInitialState } from '@/app/AppProvider';
import {
  AsyncPopconfirm,
  GuardedModalForm,
  ResponsiveProTable,
  useColumnSettings,
} from '@/components/ui';
import type { SubEntityDrawerRef } from '@/components/ui/sub-entity-drawer';
import {
  orderAttachmentServiceGetAttachmentAccess,
  orderAttachmentServiceListAttachments,
  orderAttachmentServicePrepareAttachmentUpload,
  orderAttachmentServiceRegisterAttachment,
  orderAttachmentServiceRemoveAttachmentReference,
} from '@/services/roncin/orderAttachmentService';
import { isWriteResponse, toTableRequest } from '@/utils/api';
import { formatDate } from '@/utils/format';
import {
  attachmentChecksum,
  attachmentPreviewType,
  attachmentSize,
  MAX_ATTACHMENT_SIZE,
} from './attachmentFiles';

export type AttachmentDrawerRef = SubEntityDrawerRef<API.Order>;
type AttachmentDrawerProps = { canRegister: boolean };
type UploadValues = { docType: string; files: UploadFile[] };
type Preview = { url: string; name: string; type: 'image' | 'pdf' };

const AttachmentDrawer = forwardRef<AttachmentDrawerRef, AttachmentDrawerProps>(
  function AttachmentDrawer({ canRegister }, ref) {
    const { initialState } = useInitialState();
    const workspace = `${initialState?.currentUser?.id}:${initialState?.currentUser?.currentOrganization?.id}`;
    const [selection, setSelection] = useState<{
      order: API.Order;
      workspace: string;
      generation: number;
    }>();
    const generation = useRef(0);
    const actionRef = useRef<ActionType | undefined>(undefined);
    useImperativeHandle(
      ref,
      () => ({
        open: (order) =>
          setSelection({ order, workspace, generation: ++generation.current }),
        close: () => setSelection(undefined),
        reload: () => {
          actionRef.current?.reload();
        },
      }),
      [workspace],
    );
    return selection?.workspace === workspace ? (
      <AttachmentDrawerContent
        key={`${workspace}:${selection.generation}`}
        order={selection.order}
        canRegister={canRegister}
        actionRef={actionRef}
        onClose={() => setSelection(undefined)}
      />
    ) : null;
  },
);

function AttachmentDrawerContent({
  order,
  canRegister,
  actionRef,
  onClose,
}: AttachmentDrawerProps & {
  order: API.Order;
  actionRef: React.RefObject<ActionType | undefined>;
  onClose: () => void;
}) {
  const { message } = App.useApp();
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadStage, setUploadStage] = useState('');
  const [uploadError, setUploadError] = useState('');
  const [preview, setPreview] = useState<Preview>();
  const [accessing, setAccessing] = useState<string>();
  const alive = useRef(true);
  const requests = useRef(new Set<AbortController>());
  const previewURL = useRef<string | undefined>(undefined);
  const uploaded = useRef<
    | {
        file: File;
        docType: string;
        uploadToken: string;
        expiresAt: number;
      }
    | undefined
  >(undefined);

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
      for (const controller of requests.current) controller.abort();
      requests.current.clear();
      if (previewURL.current) URL.revokeObjectURL(previewURL.current);
    };
  }, []);

  const closePreview = () => {
    if (previewURL.current) URL.revokeObjectURL(previewURL.current);
    previewURL.current = undefined;
    setPreview(undefined);
  };

  const accessFile = async (record: API.OrderAttachment, download: boolean) => {
    if (!record.id || !order.id) return;
    const controller = new AbortController();
    requests.current.add(controller);
    setAccessing(`${record.id}:${download}`);
    try {
      const response = await orderAttachmentServiceGetAttachmentAccess(
        { orderId: order.id, id: record.id, download },
        { signal: controller.signal },
      );
      if (!alive.current || !response?.data?.url) return;
      if (download) {
        const link = document.createElement('a');
        link.href = response.data.url;
        link.download = record.fileName ?? '附件';
        link.rel = 'noopener noreferrer';
        link.referrerPolicy = 'no-referrer';
        document.body.appendChild(link);
        link.click();
        link.remove();
      } else {
        const type = attachmentPreviewType(record.mimeType);
        if (!type) return;
        const file = await fetch(response.data.url, {
          credentials: 'omit',
          signal: controller.signal,
          referrerPolicy: 'no-referrer',
        });
        if (!file.ok) throw new Error('附件读取失败');
        const bytes = await file.arrayBuffer();
        if (!alive.current) return;
        closePreview();
        // 使用已登记的受控 MIME，不能信任下载响应提供的任意活动内容类型。
        const url = URL.createObjectURL(
          new Blob([bytes], { type: record.mimeType }),
        );
        previewURL.current = url;
        setPreview({ url, name: record.fileName ?? '附件预览', type });
      }
    } catch {
      if (alive.current && !controller.signal.aborted)
        message.error('附件读取失败，请重试。');
    } finally {
      requests.current.delete(controller);
      if (alive.current) setAccessing(undefined);
    }
  };

  const columns: ProColumns<API.OrderAttachment>[] = [
    {
      title: '文档类型',
      dataIndex: 'docType',
      width: 130,
      render: (type) => <Tag color="geekblue">{type}</Tag>,
    },
    { title: '文件名', dataIndex: 'fileName', ellipsis: true },
    {
      title: '文件大小',
      dataIndex: 'fileSize',
      width: 110,
      render: (_, record) => attachmentSize(record.fileSize),
    },
    {
      title: '上传时间',
      dataIndex: 'createdAt',
      width: 165,
      render: (_, record) => formatDate(record.createdAt, 'minute'),
    },
    {
      title: '操作',
      key: 'action',
      valueType: 'option',
      width: canRegister ? 220 : 140,
      fixed: 'right',
      align: 'right',
      render: (_, record) => (
        <Space size={0}>
          {attachmentPreviewType(record.mimeType) && (
            <Button
              type="link"
              size="small"
              disabled={Boolean(accessing)}
              loading={accessing === `${record.id}:false`}
              onClick={() => void accessFile(record, false)}
            >
              预览
            </Button>
          )}
          <Button
            type="link"
            size="small"
            disabled={Boolean(accessing)}
            loading={accessing === `${record.id}:true`}
            onClick={() => void accessFile(record, true)}
          >
            下载
          </Button>
          {canRegister && (
            <AsyncPopconfirm
              title="解除附件引用"
              description="其他订单仍引用时文件会保留；最后一条引用解除后文件会删除。确定继续？"
              okText="确定解除"
              cancelText="取消"
              onConfirm={async () => {
                if (!order.id || !record.id) return false;
                const response =
                  await orderAttachmentServiceRemoveAttachmentReference({
                    orderId: order.id,
                    id: record.id,
                  });
                if (!isWriteResponse(response)) return false;
                if (alive.current) {
                  message.success('已解除附件引用');
                  actionRef.current?.reload();
                }
                return true;
              }}
            >
              <Button type="link" danger size="small">
                解除引用
              </Button>
            </AsyncPopconfirm>
          )}
        </Space>
      ),
    },
  ];
  const columnSettings = useColumnSettings({
    tableKey: 'orders:sub-entity:附件',
    columns,
    structuralKeys: ['action'],
  });

  const submitUpload = async (values: UploadValues): Promise<boolean> => {
    const selected = values.files?.[0];
    const file =
      selected?.originFileObj ??
      (selected instanceof File ? selected : undefined);
    if (!file || !order.id || !canRegister) return false;
    const controller = new AbortController();
    requests.current.add(controller);
    const docType = values.docType.trim();
    setUploadError('');
    try {
      let prepared = uploaded.current;
      if (
        !prepared ||
        prepared.file !== file ||
        prepared.docType !== docType ||
        prepared.expiresAt <= Date.now()
      ) {
        setUploadStage('正在校验文件');
        const checksum = await attachmentChecksum(file);
        if (!alive.current || controller.signal.aborted) return false;
        const response = await orderAttachmentServicePrepareAttachmentUpload(
          { orderId: order.id },
          {
            orderId: order.id,
            docType,
            fileName: file.name,
            mimeType: file.type || 'application/octet-stream',
            fileSize: String(file.size),
            checksum,
          },
          { signal: controller.signal },
        );
        if (!alive.current) return false;
        if (!response?.data?.uploadUrl || !response.data.uploadToken) {
          setUploadError('无法准备上传，文件和填写内容已保留，请重试。');
          return false;
        }
        setUploadStage('正在上传文件');
        const result = await fetch(response.data.uploadUrl, {
          method: 'PUT',
          body: file,
          headers: response.data.headers,
          credentials: 'omit',
          signal: controller.signal,
          referrerPolicy: 'no-referrer',
        });
        if (!result.ok) throw new Error('文件上传失败');
        if (!alive.current) return false;
        prepared = {
          file,
          docType,
          uploadToken: response.data.uploadToken,
          expiresAt: Date.parse(response.data.expiresAt ?? ''),
        };
        uploaded.current = prepared;
      }
      setUploadStage('正在保存附件');
      const response = await orderAttachmentServiceRegisterAttachment(
        { orderId: order.id },
        { orderId: order.id, uploadToken: prepared.uploadToken },
        { signal: controller.signal },
      );
      if (!alive.current) return false;
      if (!isWriteResponse(response, true)) {
        setUploadError('保存附件失败，文件和填写内容已保留，请重试。');
        return false;
      }
      uploaded.current = undefined;
      message.success('附件上传成功');
      actionRef.current?.reload();
      return true;
    } catch {
      if (alive.current && !controller.signal.aborted)
        setUploadError('上传失败，文件和填写内容已保留，请重试。');
      return false;
    } finally {
      requests.current.delete(controller);
      if (alive.current) setUploadStage('');
    }
  };

  return (
    <>
      <Drawer
        title={`订单附件 - ${order.orderNo || order.id}`}
        open
        onClose={onClose}
        size={1000}
        destroyOnHidden
      >
        <ResponsiveProTable<API.OrderAttachment>
          actionRef={actionRef}
          rowKey="id"
          columns={columnSettings.columns}
          bordered
          search={false}
          pagination={false}
          options={{ reload: true, density: true, setting: false }}
          request={async () => {
            const response = await orderAttachmentServiceListAttachments({
              orderId: order.id as string,
            });
            return response
              ? toTableRequest(response)
              : { data: [], success: false };
          }}
          toolBarRender={() => [
            columnSettings.entry,
            canRegister && (
              <Button
                key="upload"
                type="primary"
                icon={<UploadOutlined />}
                onClick={() => {
                  uploaded.current = undefined;
                  setUploadError('');
                  setUploadOpen(true);
                }}
              >
                上传附件
              </Button>
            ),
          ]}
        />
      </Drawer>
      <GuardedModalForm<UploadValues>
        guardIdentity={order.id}
        title="上传订单附件"
        initialValues={{ docType: '其他' }}
        open={uploadOpen}
        onOpenChange={setUploadOpen}
        modalProps={{
          destroyOnHidden: true,
          mask: { closable: false },
          closable: !uploadStage,
          keyboard: !uploadStage,
        }}
        submitter={{
          searchConfig: { submitText: '上传' },
          resetButtonProps: { disabled: Boolean(uploadStage) },
        }}
        onFinish={submitUpload}
      >
        <ProFormSelect
          name="docType"
          label="文档类型"
          allowClear={false}
          options={[
            '提单',
            '运单',
            '订舱确认',
            '发票',
            '装箱单',
            '报关单',
            '费用凭证',
            '其他',
          ].map((type) => ({ label: type, value: type }))}
          rules={[{ required: true, message: '请选择文档类型' }]}
          disabled={Boolean(uploadStage)}
        />
        <Form.Item
          name="files"
          label="附件文件"
          valuePropName="fileList"
          getValueFromEvent={(event) => event.fileList}
          rules={[{ required: true, message: '请选择附件文件' }]}
        >
          <Upload.Dragger
            name="attachment"
            maxCount={1}
            multiple={false}
            disabled={Boolean(uploadStage)}
            beforeUpload={(file) => {
              if (file.size <= 0 || file.size > MAX_ATTACHMENT_SIZE) {
                message.error(
                  file.size <= 0 ? '不能上传空文件' : '单个附件不能超过100 MB',
                );
                return Upload.LIST_IGNORE;
              }
              return false;
            }}
          >
            <p>
              <UploadOutlined />
            </p>
            <p>点击或拖拽文件到此处</p>
            <p>单个附件最大100 MB</p>
          </Upload.Dragger>
        </Form.Item>
        {uploadStage && (
          <>
            <Progress percent={undefined} status="active" showInfo={false} />
            <p role="status">{uploadStage}</p>
          </>
        )}
        {uploadError && <Alert type="error" showIcon title={uploadError} />}
      </GuardedModalForm>
      <Modal
        open={Boolean(preview)}
        title={preview?.name}
        onCancel={closePreview}
        footer={null}
        width="85vw"
        destroyOnHidden
      >
        {preview?.type === 'image' && (
          <img
            src={preview.url}
            alt={preview.name}
            style={{
              display: 'block',
              maxWidth: '100%',
              maxHeight: '75vh',
              margin: '0 auto',
            }}
          />
        )}
        {preview?.type === 'pdf' && (
          <iframe
            src={preview.url}
            title={preview.name}
            sandbox=""
            style={{ width: '100%', height: '75vh', border: 0 }}
          />
        )}
      </Modal>
    </>
  );
}

export default AttachmentDrawer;
