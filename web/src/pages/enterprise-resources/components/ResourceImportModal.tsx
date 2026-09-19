import { DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd';
import { Alert, Button, Modal, Space, Upload } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import type { Dispatch, SetStateAction } from 'react';
import {
  importConflictFieldLabels,
  parseImportFile,
  type ResourceTab,
} from '../resourceConstants';

type ResourceImportModalProps = {
  active: ResourceTab;
  open: boolean;
  setOpen: Dispatch<SetStateAction<boolean>>;
  files: UploadFile[];
  setFiles: Dispatch<SetStateAction<UploadFile[]>>;
  rows: API.EnterpriseResourceInput[];
  setRows: Dispatch<SetStateAction<API.EnterpriseResourceInput[]>>;
  preview?: API.PreviewEnterpriseResourceImportResponse;
  setPreview: Dispatch<
    SetStateAction<API.PreviewEnterpriseResourceImportResponse | undefined>
  >;
  loading: boolean;
  onPreview: () => void;
  onCommit: () => void;
  onDownloadTemplate: () => void;
  message: MessageInstance;
};

/** 企业资源 CSV 批量导入弹窗：文件选择、模板下载与校验结果预览。 */
export default function ResourceImportModal({
  active,
  open,
  setOpen,
  files,
  setFiles,
  rows,
  setRows,
  preview,
  setPreview,
  loading,
  onPreview,
  onCommit,
  onDownloadTemplate,
  message,
}: ResourceImportModalProps) {
  return (
    <Modal
      title={`批量导入${active.label.replace('管理', '')}`}
      open={open}
      okText={
        !preview
          ? '校验数据'
          : (preview.conflictCount ?? 0) > 0
            ? `确认覆盖 ${preview.conflictCount} 条`
            : '确认导入'
      }
      confirmLoading={loading}
      okButtonProps={{
        disabled:
          !rows.length ||
          (preview?.invalidCount ?? 0) > 0 ||
          ((preview?.conflictCount ?? 0) > 0 &&
            preview?.overwriteAllowed !== true),
      }}
      onOk={() => (preview ? onCommit() : onPreview())}
      onCancel={() => setOpen(false)}
    >
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <Space>
          <Upload
            accept=".csv,text/csv"
            maxCount={1}
            fileList={files}
            beforeUpload={async (file) => {
              try {
                const parsedRows = parseImportFile(
                  await file.text(),
                  active.type,
                );
                if (!parsedRows.length)
                  throw new Error('CSV 文件没有可导入的数据行');
                setRows(parsedRows);
                setPreview(undefined);
              } catch (error) {
                setRows([]);
                setPreview(undefined);
                setFiles([]);
                message.error(
                  error instanceof Error ? error.message : 'CSV 文件解析失败',
                );
                return Upload.LIST_IGNORE;
              }
              return false;
            }}
            onChange={({ fileList }) => setFiles(fileList.slice(-1))}
            onRemove={() => {
              setFiles([]);
              setRows([]);
              setPreview(undefined);
              return true;
            }}
          >
            <Button icon={<UploadOutlined />}>选择 CSV 文件</Button>
          </Upload>
          <Button icon={<DownloadOutlined />} onClick={onDownloadTemplate}>
            下载模板
          </Button>
        </Space>
        {!!rows.length && !preview && (
          <Alert
            type="info"
            showIcon
            title={`已读取 ${rows.length} 条数据，请先校验`}
          />
        )}
        {preview && (
          <Alert
            type={
              (preview.invalidCount ?? 0) > 0 ||
              ((preview.conflictCount ?? 0) > 0 &&
                preview.overwriteAllowed !== true)
                ? 'error'
                : (preview.conflictCount ?? 0) > 0
                  ? 'warning'
                  : 'success'
            }
            showIcon
            title={`有效 ${preview.validCount ?? 0} 条，无效 ${preview.invalidCount ?? 0} 条，冲突 ${preview.conflictCount ?? 0} 条`}
            description={
              <Space orientation="vertical" size={4}>
                {(preview.conflictCount ?? 0) > 0 &&
                  preview.overwriteAllowed !== true && (
                    <span>
                      存在一行匹配多个资源的歧义，请先修正企业名称或企业代码后重新上传。
                    </span>
                  )}
                {preview.rows
                  ?.filter((row) => row.errors?.length)
                  .map((row) => (
                    <span key={`error-${row.rowNumber}`}>
                      第 {row.rowNumber} 行：{row.errors?.join('；')}
                    </span>
                  ))}
                {preview.rows
                  ?.filter((row) => row.conflicts?.length)
                  .map((row) => (
                    <span key={`conflict-${row.rowNumber}`}>
                      第 {row.rowNumber} 行与“
                      {row.conflicts
                        ?.map((item) => item.existingShortName)
                        .join('、')}
                      ”冲突（
                      {row.conflicts
                        ?.flatMap((item) => item.matchedFields ?? [])
                        .map(
                          (field) => importConflictFieldLabels[field] ?? field,
                        )
                        .join('、')}
                      ）
                      {(preview.conflictCount ?? 0) > 0 &&
                      preview.overwriteAllowed !== true
                        ? ''
                        : '，确认后将覆盖该资源'}
                    </span>
                  ))}
              </Space>
            }
          />
        )}
      </Space>
    </Modal>
  );
}
