import { Modal, Select, Space } from 'antd';
import type { Dispatch, SetStateAction } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';

type BatchAssigneeModalProps = {
  open: boolean;
  setOpen: Dispatch<SetStateAction<boolean>>;
  mode: 'assign' | 'remove';
  setMode: Dispatch<SetStateAction<'assign' | 'remove'>>;
  batchAssignees: string[];
  setBatchAssignees: Dispatch<SetStateAction<string[]>>;
  assigneeOptions: { label: string; value: string }[];
  searchAssignees: (keyword?: string) => Promise<void>;
  onOk: () => void;
};

/** 批量维护关联人员弹窗。 */
export default function BatchAssigneeModal({
  open,
  setOpen,
  mode,
  setMode,
  batchAssignees,
  setBatchAssignees,
  assigneeOptions,
  searchAssignees,
  onOk,
}: BatchAssigneeModalProps) {
  const { markDirty } = useUnsavedChangesGuard({ enabled: open });
  return (
    <Modal
      title="批量维护关联人员"
      open={open}
      onOk={onOk}
      onCancel={() => setOpen(false)}
    >
      <Space orientation="vertical" style={{ width: '100%' }}>
        <Select
          value={mode}
          onChange={(value) => {
            setMode(value);
            markDirty();
          }}
          options={[
            { label: '关联所选人员', value: 'assign' },
            { label: '移除所选人员', value: 'remove' },
          ]}
        />
        <Select
          style={{ width: '100%' }}
          mode="multiple"
          showSearch={{
            filterOption: false,
            onSearch: (value) => void searchAssignees(value),
          }}
          options={assigneeOptions}
          value={batchAssignees}
          onChange={(values) => {
            setBatchAssignees(values);
            markDirty();
          }}
          placeholder="搜索并选择当前组织人员"
        />
      </Space>
    </Modal>
  );
}
