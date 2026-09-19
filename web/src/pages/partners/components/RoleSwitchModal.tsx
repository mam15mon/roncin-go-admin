import { SwapOutlined } from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Checkbox,
  Form,
  Modal,
  Space,
  Tag,
  Typography,
} from 'antd';
import React, { useEffect } from 'react';
import { PartnerRoleType } from '@/enums.generated';
import { partnerServiceUpdatePartner } from '@/services/roncin/partnerService';
import { getErrorMessage } from '@/utils/errorMessage';

const { Text } = Typography;

const ROLE_OPTIONS = [
  { label: '客户', value: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER },
  { label: '供应商', value: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER },
  { label: '国外代理', value: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT },
];

const ROLE_LABELS: Record<number, string> = {
  [PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER]: '客户',
  [PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER]: '供应商',
  [PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT]: '国外代理',
};

interface RoleSwitchModalProps {
  open: boolean;
  partner: API.Partner | null;
  onClose: () => void;
  onSuccess: () => void;
}

export default function RoleSwitchModal({
  open,
  partner,
  onClose,
  onSuccess,
}: RoleSwitchModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = React.useState(false);

  useEffect(() => {
    if (open && partner) {
      const activeRoleTypes = (partner.roles ?? [])
        .filter((r) => r.enabled)
        .map((r) => r.type as number);
      form.setFieldsValue({
        roleTypes: activeRoleTypes.length > 0 ? activeRoleTypes : [1],
      });
    }
  }, [open, partner, form]);

  const handleQuickSelect = (type: PartnerRoleType) => {
    form.setFieldsValue({ roleTypes: [type] });
  };

  const handleSubmit = async () => {
    if (!partner?.id) return;
    try {
      const values = await form.validateFields();
      const selectedTypes: number[] = values.roleTypes || [];
      if (selectedTypes.length === 0) {
        message.warning('请至少保留一个业务角色身份');
        return;
      }

      setSubmitting(true);

      // 构建角色输入
      const roleInputs: API.PartnerRoleInput[] = selectedTypes.map((type) => ({
        type,
        enabled: true,
      }));

      await partnerServiceUpdatePartner(
        { id: partner.id },
        {
          id: partner.id,
          legalName: partner.legalName ?? '',
          code: partner.code ?? '',
          unifiedSocialCreditCode: partner.unifiedSocialCreditCode,
          registeredAddress: partner.registeredAddress,
          enabled: partner.enabled ?? true,
          roles: roleInputs,
        },
      );

      message.success('业务角色转换成功');
      onSuccess();
      onClose();
    } catch (e) {
      message.error(getErrorMessage(e, '角色变更失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const activeRoleNames = (partner?.roles ?? [])
    .filter((r) => r.enabled)
    .map((r) => ROLE_LABELS[r.type ?? 0] || '未知')
    .join('、');

  return (
    <Modal
      title={
        <Space size={8}>
          <SwapOutlined style={{ color: '#1677ff' }} />
          <span>业务角色转换 / 身份管理</span>
        </Space>
      }
      open={open}
      onCancel={onClose}
      destroyOnClose
      footer={[
        <Button key="cancel" onClick={onClose} disabled={submitting}>
          取消
        </Button>,
        <Button
          key="submit"
          type="primary"
          onClick={handleSubmit}
          loading={submitting}
        >
          确认转换
        </Button>,
      ]}
    >
      <div style={{ marginTop: 16 }}>
        <div
          style={{
            padding: '12px 16px',
            backgroundColor: '#f5f7fa',
            borderRadius: 6,
            marginBottom: 16,
          }}
        >
          <div style={{ marginBottom: 4 }}>
            <Text type="secondary">当前单位：</Text>
            <Text strong style={{ fontFamily: 'monospace', marginRight: 8 }}>
              {partner?.code}
            </Text>
            <Text strong>{partner?.legalName}</Text>
          </div>
          <div>
            <Text type="secondary">当前生效角色：</Text>
            <Tag color="blue">{activeRoleNames || '无'}</Tag>
          </div>
        </div>

        <Form form={form} layout="vertical">
          <Form.Item
            name="roleTypes"
            label="目标业务角色 (可单选转换或多选并存)"
            rules={[{ required: true, message: '请至少选择一个业务角色' }]}
            style={{ marginBottom: 12 }}
          >
            <Checkbox.Group options={ROLE_OPTIONS} />
          </Form.Item>

          <div style={{ marginBottom: 16 }}>
            <Text type="secondary" style={{ fontSize: 12, marginRight: 8 }}>
              一键转换快捷方式：
            </Text>
            <Space size={8}>
              <Button
                size="small"
                onClick={() =>
                  handleQuickSelect(PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER)
                }
              >
                仅设为客户
              </Button>
              <Button
                size="small"
                onClick={() =>
                  handleQuickSelect(PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER)
                }
              >
                仅设为供应商
              </Button>
              <Button
                size="small"
                onClick={() =>
                  handleQuickSelect(
                    PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
                  )
                }
              >
                仅设为国外代理
              </Button>
            </Space>
          </div>

          <Alert
            type="info"
            showIcon
            message="角色转换规则说明"
            description="往来单位在客户、供应商与国外代理之间可自由互转或多重身份并存。转换后该企业将立即在对应角色的档案列表中可见，未勾选的旧角色将被停用。"
          />
        </Form>
      </div>
    </Modal>
  );
}
