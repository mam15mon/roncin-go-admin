import { Checkbox, Modal, Select, Space } from 'antd';
import type { Dispatch, SetStateAction } from 'react';
import { addressTypes } from '../resourceConstants';

type BatchAddressTypeModalProps = {
  open: boolean;
  setOpen: Dispatch<SetStateAction<boolean>>;
  mode: 'assign' | 'remove';
  setMode: Dispatch<SetStateAction<'assign' | 'remove'>>;
  batchAddressTypes: number[];
  setBatchAddressTypes: Dispatch<SetStateAction<number[]>>;
  onOk: () => void;
};

/** 批量维护地址类型弹窗。 */
export default function BatchAddressTypeModal({
  open,
  setOpen,
  mode,
  setMode,
  batchAddressTypes,
  setBatchAddressTypes,
  onOk,
}: BatchAddressTypeModalProps) {
  return (
    <Modal
      title="批量维护地址类型"
      open={open}
      onOk={onOk}
      onCancel={() => setOpen(false)}
    >
      <Space orientation="vertical" style={{ width: '100%' }}>
        <Select
          value={mode}
          onChange={setMode}
          options={[
            { label: '分配所选类型', value: 'assign' },
            { label: '移除所选类型', value: 'remove' },
          ]}
        />
        <Checkbox.Group
          options={addressTypes}
          value={batchAddressTypes}
          onChange={(values) => setBatchAddressTypes(values as number[])}
        />
      </Space>
    </Modal>
  );
}
