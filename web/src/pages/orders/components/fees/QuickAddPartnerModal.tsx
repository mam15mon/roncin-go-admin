import { Col, Form, Input, Row, Select } from 'antd';
import React from 'react';
import { QuickCreateModal } from '@/components/ui/quick-create-modal';
import {
  PartnerRoleType,
  type PartnerRoleType as PartnerRoleTypeValue,
} from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';

type QuickAddPartnerFormValues = {
  legalName: string;
  roles: PartnerRoleTypeValue[];
};

type QuickAddPartnerModalProps = {
  open: boolean;
  onCancel: () => void;
  onSuccess: (newPartner: { id: string; name: string; code?: string }) => void;
};

export default function QuickAddPartnerModal({
  open,
  onCancel,
  onSuccess,
}: QuickAddPartnerModalProps) {
  return (
    <QuickCreateModal<
      QuickAddPartnerFormValues,
      {
        id: string;
        name: string;
        code?: string;
      }
    >
      title="快捷新建往来单位"
      open={open}
      width={540}
      onCancel={onCancel}
      onSuccess={onSuccess}
      onSubmit={async (values) => {
        const res = await partnerServiceCreatePartner({
          legalName: values.legalName.trim(),
          roles: values.roles.map((type) => ({ type, enabled: true })),
        });
        if (res.data?.id) {
          return {
            id: res.data.id,
            name: res.data.legalName ?? values.legalName.trim(),
            code: res.data.code,
          };
        }
        throw new Error('创建结果缺少伙伴 ID，请重试');
      }}
    >
      <Row gutter={16}>
        <Col span={24}>
          <Form.Item
            name="legalName"
            label="单位全称"
            rules={[
              { required: true, whitespace: true, message: '请输入单位全称' },
              { max: 200, message: '不能超过 200 字符' },
            ]}
          >
            <Input placeholder="工商登记全称或客商名称" maxLength={200} />
          </Form.Item>
        </Col>
        <Col span={24}>
          <Form.Item
            name="roles"
            label="客商类型"
            rules={[{ required: true, message: '请选择至少一种类型' }]}
          >
            <Select
              mode="multiple"
              options={[
                {
                  label: '客户 (委托单位/收发通)',
                  value: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
                },
                {
                  label: '供应商 (船东/车队/报关行/码头)',
                  value: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
                },
              ]}
            />
          </Form.Item>
        </Col>
      </Row>
    </QuickCreateModal>
  );
}
