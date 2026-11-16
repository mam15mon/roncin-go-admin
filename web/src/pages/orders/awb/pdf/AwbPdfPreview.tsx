import { ReloadOutlined } from '@ant-design/icons';
import { Alert, Button, Drawer } from 'antd';
import { useState } from 'react';
import { registerOrderPdfFont } from '../../documents/pdf/font';
import { PdfBlobView } from '../../documents/pdf/PdfBlobView';
import { AwbPdfDocument } from './AwbPdfDocument';
import {
  awbNumber,
  awbNumberSummary,
  awbPdfFileName,
  awbPdfFingerprint,
  validateAwbPdf,
} from './content';
import type { AwbPdfModel } from './types';

export type AwbPdfPreviewProps = {
  open: boolean;
  onClose: () => void;
  model: AwbPdfModel;
  fingerprint?: string;
};

registerOrderPdfFont();

function GeneratedPdf({ model }: { model: AwbPdfModel }) {
  const [fileName] = useState(() => awbPdfFileName(model, new Date()));
  return (
    <PdfBlobView
      document={<AwbPdfDocument model={model} />}
      fileName={fileName}
      title={`${model.target} ${awbNumber(model)} PDF 预览`}
    />
  );
}

/** 由制作页按需加载；身份、已保存快照或重试变化均卸载旧 Blob 实例。 */
export default function AwbPdfPreview({
  open,
  onClose,
  model,
  fingerprint = '',
}: AwbPdfPreviewProps) {
  const [attempt, setAttempt] = useState(0);
  const error = validateAwbPdf(model);
  return (
    <Drawer
      title={`${model.target} 预览 · ${model.outputKind === 'EXTERNAL_ARCHIVE' ? '登记资料归档' : '航空分运单'} · ${awbNumberSummary(model)}`}
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
            key={`${awbPdfFingerprint(model)}:${fingerprint}:${attempt}`}
            model={model}
          />
        )
      )}
    </Drawer>
  );
}
