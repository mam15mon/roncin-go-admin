import { DownloadOutlined, PrinterOutlined } from '@ant-design/icons';
import { type DocumentProps, usePDF } from '@react-pdf/renderer';
import { Alert, Button, Space, Spin } from 'antd';
import type { ReactElement } from 'react';

/** 预览、下载与打印只消费此实例的同一个 Blob；卸载时由 usePDF 回收 URL。 */
export function PdfBlobView({
  document,
  fileName,
  title,
}: {
  document: ReactElement<DocumentProps>;
  fileName: string;
  title: string;
}) {
  const [instance] = usePDF({ document });
  const ready = Boolean(instance.url && !instance.loading && !instance.error);
  return (
    <>
      <Space wrap style={{ marginBottom: 12 }}>
        <Button
          type="primary"
          icon={<DownloadOutlined />}
          disabled={!ready}
          href={ready ? (instance.url ?? undefined) : undefined}
          download={fileName}
        >
          下载 PDF
        </Button>
        <Button
          icon={<PrinterOutlined />}
          disabled={!ready}
          href={ready ? (instance.url ?? undefined) : undefined}
          target="_blank"
          rel="noopener noreferrer"
        >
          打开打印
        </Button>
      </Space>
      {instance.loading && (
        <Spin description="正在生成单证 PDF" fullscreen={false}>
          <div style={{ height: 180 }} />
        </Spin>
      )}
      {instance.error && (
        <Alert type="error" showIcon title="PDF 生成失败，请重试。" />
      )}
      {ready && (
        <iframe
          src={instance.url ?? undefined}
          title={title}
          style={{
            width: '100%',
            height: 'calc(100vh - 190px)',
            border: '1px solid #f0f0f0',
          }}
        />
      )}
    </>
  );
}
