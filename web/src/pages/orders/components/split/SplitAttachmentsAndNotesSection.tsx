import type { FormInstance } from 'antd';
import { Checkbox, Form, Input, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { SectionCard } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import type { ResultConfig } from '../../splitUtils';
import SeaExternalConfirmationFields from '../../templates/components/sea/SeaExternalConfirmationFields';

const { TextArea } = Input;
const { Text } = Typography;

interface SplitAttachmentsAndNotesSectionProps {
  splitContext: API.SeaOrderSplitContextData | null;
  results: ResultConfig[];
  attAssignments: Record<string, string[]>;
  setAttAssignments: (
    value:
      | Record<string, string[]>
      | ((prev: Record<string, string[]>) => Record<string, string[]>),
  ) => void;
  note: string;
  setNote: (value: string | ((prev: string) => string)) => void;
  orderId: string;
  confirmationForm: FormInstance;
}

/** 拆票页区块 5/6/7：附件共享引用继承、拆票说明与承运方外部确认。 */
export default function SplitAttachmentsAndNotesSection({
  splitContext,
  results,
  attAssignments,
  setAttAssignments,
  note,
  setNote,
  orderId,
  confirmationForm,
}: SplitAttachmentsAndNotesSectionProps) {
  // 附件继承列
  const attColumns: ColumnsType<API.SeaOrderSplitAttachmentItem> = [
    {
      title: '单证附件',
      dataIndex: 'fileName',
      render: (val, r) => (
        <span>
          <Tag color="geekblue">{r.docType}</Tag>
          {val}
        </span>
      ),
    },
    {
      title: '文件大小',
      dataIndex: 'fileSize',
      width: 120,
      render: (s) => `${s} 字节`,
    },
    {
      title: '共享引用至结果票',
      key: 'shareTargets',
      render: (_, att) => {
        if (!att.id) return null;
        const currentKeys = attAssignments[att.id] || [];
        return (
          <Checkbox.Group
            value={currentKeys}
            onChange={(checkedValues) => {
              const nextValues = checkedValues as string[];
              if (!nextValues.includes('res-origin')) {
                nextValues.push('res-origin');
              }
              setAttAssignments({
                ...attAssignments,
                [att.id as string]: nextValues,
              });
            }}
          >
            {results.map((r) => (
              <Checkbox
                key={r.key}
                value={r.key}
                disabled={r.role === 'ORIGINAL'}
              >
                {r.title}
              </Checkbox>
            ))}
          </Checkbox.Group>
        );
      },
    },
  ];

  const columnSettings =
    useColumnSettings<ColumnsType<API.SeaOrderSplitAttachmentItem>[number]>({
      tableKey: 'orders:split-attachments',
      columns: attColumns,
    });

  return (
    <>
      <SectionCard
        title={
          <Space>
            <Text strong>单证附件共享引用</Text>
            <Text type="secondary" style={{ fontSize: 12 }}>
              （勾选后新票将建立对该物理资产的关联引用，解除任一单票引用不影响底层文件）
            </Text>
          </Space>
        }
      >
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
          {columnSettings.entry}
        </div>
        <Table<API.SeaOrderSplitAttachmentItem>
          columns={columnSettings.columns}
          dataSource={splitContext?.attachments || []}
          rowKey="id"
          pagination={false}
          size="middle"
        />
        {columnSettings.modal}
      </SectionCard>

      <SectionCard title="拆票说明（可选）">
        <TextArea
          rows={2}
          maxLength={500}
          showCount
          placeholder="可填写本次拆票说明（将永久记录在不可变拆票事件历史中）"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      </SectionCard>

      {results.some((r) => r.targetType !== 'CURRENT') && (
        <SectionCard
          title={
            <Space>
              <Text strong>承运方外部确认</Text>
              <Text type="danger" style={{ fontSize: 12 }}>
                （检测到结果票换入其他母单，将产生内嵌改配，必须记录外部确认）
              </Text>
            </Space>
          }
        >
          <Form form={confirmationForm} layout="vertical">
            <SeaExternalConfirmationFields orderId={orderId} />
          </Form>
        </SectionCard>
      )}
    </>
  );
}
