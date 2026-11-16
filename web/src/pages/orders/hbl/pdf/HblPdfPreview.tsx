import { ReloadOutlined } from '@ant-design/icons';
import { Alert, Button, Drawer } from 'antd';
import { useState } from 'react';
import { PdfBlobView } from '../../documents/pdf/PdfBlobView';
import { hblPdfFileName, validateHblPdfCharacters } from './content';
import { registerHblPdfFont } from './font';
import { HblArchivePdfDocument } from './HblArchivePdfDocument';
import { HblPdfDocument } from './HblPdfDocument';
import {
  HBL_PDF_TEMPLATE_REVISION,
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
  const [fileName] = useState(() => {
    const name = hblPdfFileName(model.houseNo, variant, new Date());
    return model.outputKind === 'external-archive'
      ? name.replace(`-${variant}-`, '-archive-')
      : name;
  });
  return (
    <PdfBlobView
      document={
        model.outputKind === 'external-archive' ? (
          <HblArchivePdfDocument model={model} />
        ) : (
          <HblPdfDocument model={model} variant={variant} />
        )
      }
      fileName={fileName}
      title={`HBL ${model.houseNo} PDF 预览`}
    />
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
  const label =
    model.outputKind === 'external-archive'
      ? '外部提单登记资料（归档）'
      : HBL_PDF_VARIANTS.find((item) => item.value === variant)?.label;
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
            key={`${HBL_PDF_TEMPLATE_REVISION}:${JSON.stringify(model)}:${variant}:${attempt}`}
            model={model}
            variant={variant}
          />
        )
      )}
    </Drawer>
  );
}
