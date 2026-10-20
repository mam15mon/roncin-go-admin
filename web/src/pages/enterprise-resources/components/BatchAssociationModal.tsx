import { Modal, Select } from 'antd';
import type { Dispatch, SetStateAction } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';

type BatchAssociationModalProps = {
  open: boolean;
  setOpen: Dispatch<SetStateAction<boolean>>;
  mode: 'link' | 'unlink';
  partners: string[];
  setPartners: Dispatch<SetStateAction<string[]>>;
  partnerOptions: { label: string; value: string }[];
  searchPartners: (keyword?: string) => Promise<void>;
  onOk: () => void;
};

/** 批量关联 / 解除关联企业弹窗。 */
export default function BatchAssociationModal({
  open,
  setOpen,
  mode,
  partners,
  setPartners,
  partnerOptions,
  searchPartners,
  onOk,
}: BatchAssociationModalProps) {
  const { markDirty } = useUnsavedChangesGuard({ enabled: open });
  return (
    <Modal
      title={mode === 'link' ? '批量关联企业' : '批量解除企业关联'}
      open={open}
      onOk={onOk}
      onCancel={() => setOpen(false)}
    >
      <Select
        style={{ width: '100%' }}
        mode="multiple"
        showSearch={{
          filterOption: false,
          onSearch: (value) => void searchPartners(value),
        }}
        options={partnerOptions}
        value={partners}
        onChange={(values) => {
          setPartners(values);
          markDirty();
        }}
        placeholder="搜索并选择企业"
      />
    </Modal>
  );
}
