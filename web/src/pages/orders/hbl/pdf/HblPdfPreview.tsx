import {
  DownloadOutlined,
  PrinterOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { usePDF } from '@react-pdf/renderer';
import { Alert, Button, Drawer, Space, Spin } from 'antd';
import { useState } from 'react';
import { hblPdfFileName, validateHblPdfCharacters } from './content';
import { registerHblPdfFont } from './font';
import { HblPdfDocument } from './HblPdfDocument';
import {
  HBL_PDF_VARIANTS,
  type HblPdfModel,
  type HblPdfVariant,
} from './types';

export type HblPdfPreviewProps = {
  open: boolean;
  onClose: () => void;
  model: HblPdfModel;
  variant: HblPdfVariant;
};

registerHblPdfFont();

function GeneratedPdf({
  model,
  variant,
}: Pick<HblPdfPreviewProps, 'model' | 'variant'>) {
  const [fileName] = useState(() =>
    hblPdfFileName(model.houseNo, variant, new Date()),
  );
  const [instance] = usePDF({
    document: <HblPdfDocument model={model} variant={variant} />,
  });
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
        <Spin description="正在生成提单 PDF" fullscreen={false}>
          <div style={{ height: 180 }} />
        </Spin>
      )}
      {instance.error && (
        <Alert type="error" showIcon title="PDF 生成失败，请重试。" />
      )}
      {ready && (
        <iframe
          src={instance.url ?? undefined}
          title={`HBL ${model.houseNo} PDF 预览`}
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

/** 每次打开及切换数据/版本重挂实例，旧 URL 由 usePDF 生命周期回收。 */
export default function HblPdfPreview({
  open,
  onClose,
  model,
  variant,
}: HblPdfPreviewProps) {
  const [attempt, setAttempt] = useState(0);
  const error = validateHblPdfCharacters(model);
  const label = HBL_PDF_VARIANTS.find((item) => item.value === variant)?.label;
  return (
    <Drawer
      title={`HBL 预览 · ${label} · ${model.houseNo}`}
      open={open}
      onClose={onClose}
      size="90vw"
      destroyOnHidden
      extra={
        <Button
          icon={<ReloadOutlined />}
          onClick={() => setAttempt((value) => value + 1)}
        >
          重新生成
        </Button>
      }
    >
      {error ? (
        <Alert type="error" showIcon title={error} />
      ) : (
        open && (
          <GeneratedPdf
            key={`${JSON.stringify(model)}:${variant}:${attempt}`}
            model={model}
            variant={variant}
          />
        )
      )}
    </Drawer>
  );
}
