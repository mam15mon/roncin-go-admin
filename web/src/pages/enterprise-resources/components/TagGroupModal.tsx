import type { FormInstance } from 'antd';
import {
  Button,
  ColorPicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Tag,
} from 'antd';
import type { Dispatch, SetStateAction } from 'react';
import { enterpriseResourceServiceDeleteEnterpriseTagGroup } from '@/services/roncin/enterpriseResourceService';

type TagGroupModalProps = {
  open: boolean;
  setOpen: Dispatch<SetStateAction<boolean>>;
  editingGroup?: API.EnterpriseTagGroup;
  setEditingGroup: Dispatch<SetStateAction<API.EnterpriseTagGroup | undefined>>;
  groupForm: FormInstance;
  tagGroups: API.EnterpriseTagGroup[];
  loadTagGroups: () => Promise<void>;
  onSave: () => void;
};

/** 标签组维护弹窗：表单与既有标签组列表。 */
export default function TagGroupModal({
  open,
  setOpen,
  editingGroup,
  setEditingGroup,
  groupForm,
  tagGroups,
  loadTagGroups,
  onSave,
}: TagGroupModalProps) {
  return (
    <Modal
      title={editingGroup ? '编辑标签组' : '新建标签组'}
      open={open}
      onOk={onSave}
      onCancel={() => setOpen(false)}
      destroyOnHidden
    >
      <Form form={groupForm} layout="vertical" initialValues={{ sortOrder: 0 }}>
        <Form.Item name="name" label="组名" rules={[{ required: true }]}>
          <Input maxLength={100} />
        </Form.Item>
        <Form.Item name="color" label="颜色">
          <ColorPicker showText allowClear />
        </Form.Item>
        <Form.Item name="sortOrder" label="排序">
          <InputNumber min={0} />
        </Form.Item>
      </Form>
      {!!tagGroups.length && !editingGroup && (
        <Space orientation="vertical" style={{ width: '100%' }}>
          {tagGroups.map((group) => (
            <Space
              key={group.id}
              style={{ justifyContent: 'space-between', width: '100%' }}
            >
              <Tag color={group.color}>{group.name}</Tag>
              <Space>
                <Button
                  type="link"
                  size="small"
                  onClick={() => {
                    setEditingGroup(group);
                    groupForm.setFieldsValue(group);
                  }}
                >
                  编辑
                </Button>
                <Popconfirm
                  title="仅空标签组可删除，确认删除？"
                  onConfirm={async () => {
                    if (!group.id) return;
                    await enterpriseResourceServiceDeleteEnterpriseTagGroup({
                      id: group.id,
                    });
                    await loadTagGroups();
                  }}
                >
                  <Button type="link" danger size="small">
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            </Space>
          ))}
        </Space>
      )}
    </Modal>
  );
}
