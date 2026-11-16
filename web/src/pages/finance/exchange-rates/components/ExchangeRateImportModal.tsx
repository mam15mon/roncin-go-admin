import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownloadOutlined,
  FileExcelOutlined,
  InboxOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useMutation } from '@tanstack/react-query';
import type { TableColumnsType } from 'antd';
import { Alert, App, Button, Modal, Space, Tag, Upload } from 'antd';
import type { UploadFile } from 'antd/es/upload/interface';
import React, { useEffect, useState } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { isRequestTimeoutError } from '@/requestErrorConfig';
import {
  exchangeRateServiceConfirmExchangeRateImport,
  exchangeRateServicePreviewExchangeRateImport,
} from '@/services/roncin/exchangeRateService';
import { isWriteResponse } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatDate } from '@/utils/format';
import { longRequestOptions } from '@/utils/requestTimeout';
import { generateUUID } from '@/utils/uuid';
import { downloadExchangeRateImportTemplate } from './exchangeRateTemplate';

type Props = {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
};

export function ExchangeRateImportModal({ open, onClose, onSuccess }: Props) {
  const { markDirty, resetDirty } = useUnsavedChangesGuard({ enabled: open });
  const { message } = App.useApp();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const previewImport = useMutation({
    meta: { silent: true },
    mutationFn: async (file: File) => {
      const base64 = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result as string;
          const commaIndex = result.indexOf(',');
          resolve(commaIndex !== -1 ? result.slice(commaIndex + 1) : result);
        };
        reader.onerror = reject;
        reader.readAsDataURL(file);
      });
      const response = await exchangeRateServicePreviewExchangeRateImport(
        { fileName: file.name, fileContent: base64 },
        longRequestOptions,
      );
      if (
        !response?.data ||
        response.success === false ||
        !response.previewToken
      )
        throw new Error('上传预检失败，请重试');
      return { ...response, confirmationKey: generateUUID() };
    },
  });
  const previewing = previewImport.isPending;
  const batch = previewImport.data?.data;
  const confirmation = useMutation({
    meta: { silent: true },
    mutationFn: (intent: API.ConfirmExchangeRateImportRequest) =>
      exchangeRateServiceConfirmExchangeRateImport(intent, longRequestOptions),
  });
  const confirming = confirmation.isPending;
  const resetPreview = previewImport.reset;
  const resetConfirmation = confirmation.reset;
  useEffect(() => {
    if (!open) {
      resetPreview();
      resetConfirmation();
    }
  }, [open, resetPreview, resetConfirmation]);
  const [downloadingTemplate, setDownloadingTemplate] = useState(false);

  const reset = () => {
    resetDirty();
    setFileList([]);
    previewImport.reset();
    confirmation.reset();
  };

  const handleClose = () => {
    if (confirming) return;
    reset();
    onClose();
  };

  const handleDownloadTemplate = async () => {
    setDownloadingTemplate(true);
    try {
      await downloadExchangeRateImportTemplate(message);
    } finally {
      setDownloadingTemplate(false);
    }
  };

  const processFile = async (file: File) => {
    if (!file.name.endsWith('.xlsx')) {
      message.error('只允许上传 .xlsx 格式的 Excel 文件');
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      message.error('文件大小不能超过 5MB');
      return;
    }

    previewImport.mutate(file, {
      onSuccess: (response) => {
        if (response.data?.canConfirm) {
          markDirty();
          message.success(
            `预检成功：共 ${response.data.totalCount} 条汇率准备就绪`,
          );
        } else {
          message.warning(
            `预检发现问题：存在 ${response.data?.invalidCount || 0} 条无效数据`,
          );
        }
      },
      onError: (error) => {
        message.error(
          isRequestTimeoutError(error)
            ? '上传预检超时，请确认操作结果后重试'
            : getErrorMessage(error, '上传预检失败'),
        );
      },
    });
  };

  const handleConfirm = () => {
    const snapshot = previewImport.data;
    if (!snapshot?.previewToken) {
      message.error('缺少预检令牌，请重新上传文件');
      return;
    }
    confirmation.mutate(
      {
        previewToken: snapshot.previewToken,
        idempotencyKey: snapshot.confirmationKey,
      },
      {
        onSuccess: (response) => {
          if (!isWriteResponse(response, true)) return;
          message.success('汇率批量导入成功');
          onSuccess();
          reset();
          onClose();
        },
        onError: (error) => {
          message.error(
            isRequestTimeoutError(error)
              ? '确认导入超时，请确认操作结果后重试'
              : getErrorMessage(error, '确认导入失败'),
          );
        },
      },
    );
  };

  const previewColumns: TableColumnsType<API.ExchangeRateImportRow> = [
    {
      title: '行号',
      dataIndex: 'rowNumber',
      width: 65,
      align: 'center',
      render: (val) => `#${val}`,
    },
    {
      title: '原币',
      dataIndex: 'fromCurrency',
      width: 75,
      align: 'center',
    },
    {
      title: '本币',
      dataIndex: 'toCurrency',
      width: 75,
      align: 'center',
    },
    {
      title: '应收汇率（卖出价）',
      dataIndex: 'arRate',
      width: 140,
      align: 'right',
      render: (val) => (
        <span style={{ fontWeight: 600, color: '#1677ff' }}>{val || '-'}</span>
      ),
    },
    {
      title: '应付汇率（买入价）',
      dataIndex: 'apRate',
      width: 140,
      align: 'right',
      render: (val) => (
        <span style={{ fontWeight: 600, color: '#52c41a' }}>{val || '-'}</span>
      ),
    },
    {
      title: '基准汇率',
      dataIndex: 'rate',
      width: 105,
      align: 'right',
      render: (val) => <span style={{ color: '#8c8c8c' }}>{val || '-'}</span>,
    },
    {
      title: '生效周',
      dataIndex: 'effectiveFrom',
      width: 190,
      render: (_, r) =>
        `${formatDate(r.effectiveFrom)} ~ ${formatDate(r.effectiveTo)}`,
    },
    {
      title: '预检状态',
      dataIndex: 'status',
      width: 90,
      align: 'center',
      render: (val) =>
        val === 'VALID' ? (
          <Tag color="green">合格</Tag>
        ) : (
          <Tag color="red">异常</Tag>
        ),
    },
    {
      title: '校验说明 / 错误原因',
      dataIndex: 'errors',
      render: (errors: string[] | undefined, r) => {
        if (r.status === 'VALID' || !errors || errors.length === 0) {
          return <span style={{ color: '#52c41a' }}>校验通过</span>;
        }
        return (
          <div style={{ color: '#ff4d4f' }}>
            {errors.map((err) => (
              <div key={err}>• {err}</div>
            ))}
          </div>
        );
      },
    },
  ];

  const columnSettings = useColumnSettings<
    TableColumnsType<API.ExchangeRateImportRow>[number]
  >({
    tableKey: 'finance:exchange-import-preview',
    columns: previewColumns,
  });

  return (
    <Modal
      title={
        <Space>
          <FileExcelOutlined style={{ color: '#52c41a' }} />
          <span>汇率 Excel 批量导入</span>
        </Space>
      }
      open={open}
      width={1060}
      destroyOnHidden
      closable={!confirming}
      mask={{ closable: !confirming }}
      keyboard={!confirming}
      onCancel={handleClose}
      footer={
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <div>
            <Button
              icon={<DownloadOutlined />}
              loading={downloadingTemplate}
              onClick={handleDownloadTemplate}
            >
              下载标准导入模板
            </Button>
          </div>
          <Space>
            {batch && (
              <Button
                icon={<ReloadOutlined />}
                onClick={reset}
                disabled={confirming}
              >
                重新上传
              </Button>
            )}
            <Button onClick={handleClose} disabled={confirming}>
              取消
            </Button>
            <Button
              type="primary"
              disabled={!batch?.canConfirm}
              loading={confirming}
              onClick={handleConfirm}
            >
              确认导入 ({batch?.validCount || 0} 条)
            </Button>
          </Space>
        </div>
      }
    >
      {!batch ? (
        <div style={{ padding: '24px 0' }}>
          <Upload.Dragger
            accept=".xlsx"
            maxCount={1}
            fileList={fileList}
            onChange={({ fileList: nextFileList }) => setFileList(nextFileList)}
            showUploadList={false}
            beforeUpload={(file) => {
              void processFile(file);
              return false;
            }}
            disabled={previewing}
          >
            <p className="ant-upload-drag-icon">
              <InboxOutlined style={{ fontSize: 48, color: '#1677ff' }} />
            </p>
            <p
              className="ant-upload-text"
              style={{ fontSize: 16, fontWeight: 500 }}
            >
              {previewing
                ? '正在解析并严格预检 Excel 文件，请稍候...'
                : '点击或将 Excel 文件拖拽至此处'}
            </p>
            <p className="ant-upload-hint" style={{ color: '#8c8c8c' }}>
              仅支持 .xlsx 格式文件，单文件最大 5MB，最多 500
              条数据。请使用本页下载的标准模板（列：原币 / 本币 / 应收汇率 /
              应付汇率 / 基准汇率 / 生效开始时间）。
              任选目标自然周内时间，系统自动按自然周（周一至周日）生效并严格校验币种与点差。
            </p>
          </Upload.Dragger>
        </div>
      ) : (
        <div>
          {batch.canConfirm ? (
            <Alert
              type="success"
              showIcon
              icon={<CheckCircleOutlined />}
              title="预检通过，数据格式完全合规"
              description={`文件「${batch.fileName}」包含 ${batch.totalCount} 条汇率记录，无冲突与异常，点击下方「确认导入」即可写入系统。`}
              style={{ marginBottom: 16 }}
            />
          ) : (
            <Alert
              type="error"
              showIcon
              icon={<CloseCircleOutlined />}
              title="预检未通过，已拒绝整批导入"
              description={`文件「${batch.fileName}」共 ${batch.totalCount} 行，其中包含 ${batch.invalidCount} 行错误。根据严格防冲突策略，请根据下方错误明细修改 Excel 后重新上传。`}
              style={{ marginBottom: 16 }}
            />
          )}

          <div style={{ marginBottom: 12, display: 'flex', gap: 16 }}>
            <Tag color="blue">总行数: {batch.totalCount}</Tag>
            <Tag color="green">合规行数: {batch.validCount}</Tag>
            <Tag color={batch.invalidCount ? 'red' : 'default'}>
              异常行数: {batch.invalidCount || 0}
            </Tag>
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginBottom: 8,
            }}
          >
            {columnSettings.entry}
          </div>
          <Table<API.ExchangeRateImportRow>
            rowKey={(r) => `${r.rowNumber}-${r.fromCurrency}-${r.toCurrency}`}
            size="small"
            bordered
            pagination={{ pageSize: 10, showSizeChanger: false }}
            dataSource={batch.rows || []}
            columns={columnSettings.columns}
          />
        </div>
      )}
    </Modal>
  );
}

export default ExchangeRateImportModal;
