import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownloadOutlined,
  FileExcelOutlined,
  InboxOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Modal,
  Space,
  Table,
  Tag,
  Upload,
} from 'antd';
import type { UploadFile } from 'antd/es/upload/interface';
import React, { useState } from 'react';
import { isRequestTimeoutError } from '@/requestErrorConfig';
import {
  exchangeRateServiceConfirmExchangeRateImport,
  exchangeRateServiceDownloadExchangeRateImportTemplate,
  exchangeRateServicePreviewExchangeRateImport,
} from '@/services/roncin/exchangeRateService';
import { formatDate } from '@/utils/format';
import { longRequestOptions } from '@/utils/requestTimeout';

const rateTypeLabels: Record<string, string> = {
  BASE_CURRENCY: '折本币',
  INVOICE: '开票汇率',
  SETTLEMENT: '结算汇率',
  WRITE_OFF: '核销汇率',
  BILL: '账单汇率',
};

type Props = {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
};

export function ExchangeRateImportModal({ open, onClose, onSuccess }: Props) {
  const { message } = App.useApp();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [previewing, setPreviewing] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [downloadingTemplate, setDownloadingTemplate] = useState(false);
  const [previewToken, setPreviewToken] = useState<string>();
  const [batch, setBatch] = useState<API.ExchangeRateImportBatch>();

  const reset = () => {
    setFileList([]);
    setPreviewing(false);
    setConfirming(false);
    setPreviewToken(undefined);
    setBatch(undefined);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const handleDownloadTemplate = async () => {
    setDownloadingTemplate(true);
    try {
      const res = await exchangeRateServiceDownloadExchangeRateImportTemplate();
      const base64Data = res.content;
      if (!base64Data) {
        message.error('下载模板失败：文件内容为空');
        return;
      }
      const byteCharacters = atob(base64Data);
      const byteNumbers = new Array(byteCharacters.length);
      for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i);
      }
      const byteArray = new Uint8Array(byteNumbers);
      const blob = new Blob([byteArray], {
        type:
          res.contentType ||
          'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = res.fileName || '汇率导入模板.xlsx';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      message.success('导入模板下载成功');
    } catch (e: any) {
      message.error(e.message || '下载导入模板失败');
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

    setPreviewing(true);
    try {
      const base64 = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result as string;
          // 去除 data:...;base64, 前缀
          const commaIndex = result.indexOf(',');
          resolve(commaIndex !== -1 ? result.slice(commaIndex + 1) : result);
        };
        reader.onerror = (error) => reject(error);
        reader.readAsDataURL(file);
      });

      const res = await exchangeRateServicePreviewExchangeRateImport(
        {
          fileName: file.name,
          fileContent: base64,
        },
        longRequestOptions,
      );

      setPreviewToken(res.previewToken);
      setBatch(res.data);
      if (res.data?.canConfirm) {
        message.success(`预检成功：共 ${res.data.totalCount} 条汇率准备就绪`);
      } else {
        message.warning(`预检发现问题：存在 ${res.data?.invalidCount || 0} 条无效数据`);
      }
    } catch (e: any) {
      message.error(
        isRequestTimeoutError(e)
          ? '上传预检超时，请确认操作结果后重试'
          : e.message || '上传预检失败',
      );
      setBatch(undefined);
      setPreviewToken(undefined);
    } finally {
      setPreviewing(false);
    }
  };

  const handleConfirm = async () => {
    if (!previewToken) {
      message.error('缺少预检令牌，请重新上传文件');
      return;
    }
    setConfirming(true);
    try {
      await exchangeRateServiceConfirmExchangeRateImport(
        {
          previewToken,
          idempotencyKey: globalThis.crypto.randomUUID(),
        },
        longRequestOptions,
      );
      message.success('汇率批量导入成功');
      onSuccess();
      handleClose();
    } catch (e: any) {
      message.error(
        isRequestTimeoutError(e)
          ? '确认导入超时，请确认操作结果后重试'
          : e.message || '确认导入失败',
      );
    } finally {
      setConfirming(false);
    }
  };

  return (
    <Modal
      title={
        <Space>
          <FileExcelOutlined style={{ color: '#52c41a' }} />
          <span>汇率 Excel 批量导入</span>
        </Space>
      }
      open={open}
      width={980}
      destroyOnHidden
      onCancel={handleClose}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
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
              <Button icon={<ReloadOutlined />} onClick={reset} disabled={confirming}>
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
            <p className="ant-upload-text" style={{ fontSize: 16, fontWeight: 500 }}>
              {previewing ? '正在解析并严格预检 Excel 文件，请稍候...' : '点击或将 Excel 文件拖拽至此处'}
            </p>
            <p className="ant-upload-hint" style={{ color: '#8c8c8c' }}>
              仅支持 .xlsx 格式文件，单文件最大 5MB，最多 500 条数据。
              系统将自动进行两阶段严格校验（重叠检测、币种检查、精度及时间格式）。
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

          <Table<API.ExchangeRateImportRow>
            rowKey={(r) => `${r.rowNumber}-${r.rateType}-${r.fromCurrency}`}
            size="small"
            bordered
            pagination={{ pageSize: 10, showSizeChanger: false }}
            dataSource={batch.rows || []}
            columns={[
              {
                title: '行号',
                dataIndex: 'rowNumber',
                width: 65,
                align: 'center',
                render: (val) => `#${val}`,
              },
              {
                title: '汇率类型',
                dataIndex: 'rateType',
                width: 100,
                render: (val) => rateTypeLabels[val] || val,
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
                title: '应收汇率',
                dataIndex: 'receivableRate',
                width: 100,
                align: 'right',
              },
              {
                title: '应付汇率',
                dataIndex: 'payableRate',
                width: 100,
                align: 'right',
              },
              {
                title: '生效开始时间',
                dataIndex: 'effectiveFrom',
                width: 160,
                render: (val) => formatDate(val),
              },
              {
                title: '生效结束时间',
                dataIndex: 'effectiveTo',
                width: 160,
                render: (val) =>
                  val ? (
                    formatDate(val)
                  ) : (
                    <Tag color="cyan">长期有效</Tag>
                  ),
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
            ]}
          />
        </div>
      )}
    </Modal>
  );
}

export default ExchangeRateImportModal;
