import {
  Button,
  Form,
  Input,
  Modal,
  message,
  Segmented,
  Select,
  Space,
  Tag,
} from 'antd';
import { useEffect, useMemo, useState } from 'react';
import {
  enterpriseResourceServiceCreateEnterpriseResource,
  enterpriseResourceServiceListEnterpriseTagGroups,
} from '@/services/roncin/enterpriseResourceService';
import { orderTagServiceListOrderTagOptions } from '@/services/roncin/orderTagService';

export type BusinessTagModalMode = 'assign' | 'remove';

interface BusinessTagModalProps {
  open: boolean;
  /** 选中待操作的业务对象数量 */
  targetCount: number;
  /** 业务对象已有标签（含停用），移除模式展示用 */
  existingTags?: API.BusinessTagSummary[];
  /** 是否显示快捷新建入口（需同时具备企业资源创建权限） */
  canQuickCreate: boolean;
  /** 自定义标签候选加载器（费用/账单页面传入对应领域接口） */
  loadOptions?: (params: {
    keyword?: string;
    page: number;
    pageSize: number;
  }) => Promise<
    | API.ListOrderTagOptionsResponse
    | API.ListFinanceFeeTagOptionsResponse
    | API.ListFinanceBillTagOptionsResponse
  >;
  /** 提交选中标签 ID，由调用方执行对应领域的批量接口 */
  onSubmit: (mode: BusinessTagModalMode, tagIds: string[]) => Promise<void>;
  onCancel: () => void;
}

function tagColorStyle(
  tag: API.BusinessTagSummary,
): React.CSSProperties | undefined {
  return tag.groupColor
    ? { color: tag.groupColor, borderColor: tag.groupColor }
    : undefined;
}

/** 业务标签选择弹窗：订单、费用和账单页面共用。 */
export function BusinessTagModal({
  open,
  targetCount,
  existingTags,
  canQuickCreate,
  loadOptions,
  onSubmit,
  onCancel,
}: BusinessTagModalProps) {
  const [mode, setMode] = useState<BusinessTagModalMode>('assign');
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([]);
  const [tagOptions, setTagOptions] = useState<API.BusinessTagSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [quickCreateOpen, setQuickCreateOpen] = useState(false);
  const [groupOptions, setGroupOptions] = useState<
    { label: string; value: string; color?: string }[]
  >([]);
  const [quickForm] = Form.useForm<{ groupId: string; name: string }>();

  const fetchOptions = loadOptions ?? orderTagServiceListOrderTagOptions;
  const loadTagOptions = async (keyword?: string) => {
    setLoading(true);
    try {
      const response = await fetchOptions({
        page: 1,
        pageSize: 50,
        keyword: keyword?.trim() || undefined,
      });
      setTagOptions(response.tags ?? []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!open) return;
    setMode('assign');
    setSelectedTagIds([]);
    setQuickCreateOpen(false);
    void loadTagOptions();
  }, [open]);

  const mergedOptions = useMemo(() => {
    if (mode !== 'remove') return tagOptions;
    const byId = new Map(tagOptions.map((tag) => [tag.id ?? '', tag]));
    for (const tag of existingTags ?? []) {
      if (tag.id && !byId.has(tag.id)) byId.set(tag.id, tag);
    }
    return [...byId.values()];
  }, [mode, tagOptions, existingTags]);

  const submit = async () => {
    if (!selectedTagIds.length) {
      message.warning('请先选择标签');
      return;
    }
    setSubmitting(true);
    try {
      await onSubmit(mode, selectedTagIds);
      onCancel();
    } finally {
      setSubmitting(false);
    }
  };

  const submitQuickCreate = async () => {
    const values = await quickForm.validateFields();
    const response = await enterpriseResourceServiceCreateEnterpriseResource({
      resource: {
        resourceType: 4,
        shortName: values.name.trim(),
        enabled: true,
        sortOrder: 0,
        tag: { groupId: values.groupId },
      },
    });
    message.success('标签已创建');
    setQuickCreateOpen(false);
    quickForm.resetFields();
    const createdID = response.data?.id;
    if (createdID) {
      const group = groupOptions.find(
        (option) => option.value === values.groupId,
      );
      const createdTag: API.BusinessTagSummary = {
        id: createdID,
        name: response.data?.shortName ?? values.name.trim(),
        groupId: values.groupId,
        groupName: group?.label,
        groupColor: group?.color,
        enabled: response.data?.enabled ?? true,
      };
      setTagOptions((current) => [
        createdTag,
        ...current.filter((tag) => tag.id !== createdID),
      ]);
      setSelectedTagIds([createdID]);
    }
  };

  return (
    <Modal
      title="标签管理"
      open={open}
      onOk={() => void submit()}
      confirmLoading={submitting}
      okText={mode === 'assign' ? '添加标签' : '移除标签'}
      okButtonProps={{ disabled: !selectedTagIds.length }}
      onCancel={onCancel}
      destroyOnHidden
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Space>
          <Segmented
            value={mode}
            onChange={(value) => {
              setMode(value as BusinessTagModalMode);
              setSelectedTagIds([]);
            }}
            options={[
              { label: '添加标签', value: 'assign' },
              { label: '移除标签', value: 'remove' },
            ]}
          />
          <span>已选 {targetCount} 个对象</span>
        </Space>
        <Select
          mode="multiple"
          style={{ width: '100%' }}
          placeholder={
            mode === 'assign'
              ? '搜索并选择要添加的标签'
              : '选择要移除的标签（含已停用）'
          }
          value={selectedTagIds}
          onChange={setSelectedTagIds}
          loading={loading}
          showSearch
          filterOption={false}
          onSearch={(keyword) => void loadTagOptions(keyword)}
          options={mergedOptions.map((tag) => ({
            value: tag.id ?? '',
            label: (
              <Space size={4}>
                {tag.groupColor ? (
                  <span
                    style={{
                      display: 'inline-block',
                      width: 8,
                      height: 8,
                      borderRadius: 4,
                      backgroundColor: tag.groupColor,
                    }}
                  />
                ) : null}
                <span>{tag.name}</span>
                {tag.groupName ? (
                  <span style={{ color: '#999' }}>{tag.groupName}</span>
                ) : null}
                {!tag.enabled ? (
                  <Tag style={{ marginRight: 0 }}>已停用</Tag>
                ) : null}
              </Space>
            ),
          }))}
        />
        {selectedTagIds.length > 0 && (
          <Space size={4} wrap>
            {selectedTagIds.map((id) => {
              const tag = mergedOptions.find((option) => option.id === id);
              return tag ? (
                <Tag key={id} style={tagColorStyle(tag)}>
                  {tag.name}
                </Tag>
              ) : null;
            })}
          </Space>
        )}
        {canQuickCreate && mode === 'assign' && !quickCreateOpen && (
          <Button
            type="link"
            size="small"
            onClick={() => {
              setQuickCreateOpen(true);
              void enterpriseResourceServiceListEnterpriseTagGroups().then(
                (response) => {
                  setGroupOptions(
                    (response.data ?? []).map((group) => ({
                      label: group.name ?? '',
                      value: group.id ?? '',
                      color: group.color,
                    })),
                  );
                },
              );
            }}
          >
            + 新增标签
          </Button>
        )}
        {canQuickCreate && mode === 'assign' && quickCreateOpen && (
          <Form
            form={quickForm}
            layout="inline"
            onFinish={() => void submitQuickCreate()}
          >
            <Form.Item
              name="groupId"
              rules={[{ required: true, message: '选择标签组' }]}
            >
              <Select
                style={{ width: 160 }}
                placeholder="标签组"
                showSearch
                optionFilterProp="label"
                options={groupOptions}
              />
            </Form.Item>
            <Form.Item
              name="name"
              rules={[{ required: true, message: '输入标签名称' }]}
            >
              <Input placeholder="标签名称" maxLength={200} />
            </Form.Item>
            <Button htmlType="submit">创建</Button>
          </Form>
        )}
      </Space>
    </Modal>
  );
}
