import { Col, Form, Input, Modal, Row, Select } from 'antd';
import React, { useState } from 'react';
import { feeCatalogServiceCreateFeeSetting } from '@/services/roncin/feeCatalogService';

type QuickAddFeeModalProps = {
  open: boolean;
  onCancel: () => void;
  onSuccess: (newFeeSetting: API.FeeSetting) => void;
  currencies: API.Currency[];
  billingUnits: API.BillingUnit[];
  taxableServices: API.TaxableService[];
};

export default function QuickAddFeeModal({
  open,
  onCancel,
  onSuccess,
  currencies,
  billingUnits,
  taxableServices,
}: QuickAddFeeModalProps) {
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);
      const res = await feeCatalogServiceCreateFeeSetting(values);
      if (res.data) {
        onSuccess(res.data);
      }
    } catch {
      // Form validation error
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title="快捷新增费用科目"
      open={open}
      confirmLoading={saving}
      okText="保存并选用"
      cancelText="取消"
      onOk={() => void handleSave()}
      onCancel={onCancel}
      destroyOnHidden
      width={580}
    >
      <Form form={form} layout="vertical" preserve={false}>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="feeCode"
              label="科目代码"
              rules={[
                { required: true, whitespace: true, message: '请输入科目代码' },
                { max: 30, message: '不能超过 30 字符' },
              ]}
            >
              <Input placeholder="例如：THC、OFRT、CUSTOMS" maxLength={30} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="nameZh"
              label="科目中文名称"
              rules={[
                { required: true, whitespace: true, message: '请输入中文名称' },
                { max: 100, message: '不能超过 100 字符' },
              ]}
            >
              <Input placeholder="例如：码头操作费、海运费" maxLength={100} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="nameEn" label="英文名称（选填）">
              <Input
                placeholder="例如：Terminal Handling Charge"
                maxLength={100}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="defaultCurrency"
              label="默认币种"
              rules={[{ required: true, message: '请选择币种' }]}
            >
              <Select
                options={currencies.map((c) => ({
                  label: `${c.code} (${c.name})`,
                  value: c.code ?? '',
                }))}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="billingUnitId"
              label="默认计费单位"
              rules={[{ required: true, message: '请选择计费单位' }]}
            >
              <Select
                options={billingUnits.map((u) => ({
                  label: u.name ?? '',
                  value: u.id ?? '',
                }))}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="taxRate"
              label="默认增值税税率"
              rules={[{ required: true, message: '请选择税率' }]}
            >
              <Select
                options={[
                  { label: '0% (零税率/免税)', value: '0' },
                  { label: '6% (现代服务业/货运代理)', value: '0.06' },
                  { label: '9% (基础交通运输)', value: '0.09' },
                  { label: '13% (商品贸易/修箱)', value: '0.13' },
                ]}
              />
            </Form.Item>
          </Col>
          <Col span={24}>
            <Form.Item name="taxableServiceId" label="应税服务类别">
              <Select
                placeholder="选择税目分类"
                options={taxableServices.map((s) => ({
                  label: s.goodsCode
                    ? `${s.name} (${s.goodsCode})`
                    : s.name || '',
                  value: s.id ?? '',
                }))}
              />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
