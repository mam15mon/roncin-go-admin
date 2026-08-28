import {
  Alert,
  Checkbox,
  Col,
  Form,
  type FormInstance,
  Input,
  Modal,
  Row,
  Select,
} from 'antd';
import React from 'react';

type QuickAddInvoiceProfileModalProps = {
  open: boolean;
  partnerName: string;
  saving: boolean;
  form: FormInstance;
  onOk: () => void;
  onCancel: () => void;
};

export default function QuickAddInvoiceProfileModal({
  open,
  partnerName,
  saving,
  form,
  onOk,
  onCancel,
}: QuickAddInvoiceProfileModalProps) {
  return (
    <Modal
      title={`为【${partnerName}】新增开票抬头`}
      open={open}
      confirmLoading={saving}
      okText="保存并选用"
      cancelText="取消"
      onOk={onOk}
      onCancel={onCancel}
      destroyOnHidden
      width={620}
    >
      <Form form={form} layout="vertical" preserve={false}>
        <Alert
          type="info"
          showIcon
          title="新增的开票抬头将自动保存至该客户的主档案中，并自动选中为当前账单的对账抬头。"
          style={{ marginBottom: 16 }}
        />
        <Row gutter={16}>
          <Col span={24}>
            <Form.Item
              name="invoiceTitle"
              label="发票抬头全称"
              rules={[
                {
                  required: true,
                  whitespace: true,
                  message: '请输入发票抬头全称',
                },
                { max: 200, message: '抬头全称不能超过 200 字' },
              ]}
            >
              <Input placeholder="公司注册全称或开票抬头" maxLength={200} />
            </Form.Item>
          </Col>
          <Col span={14}>
            <Form.Item
              name="taxpayerIdentificationNo"
              label="统一社会信用代码 / 税号"
              rules={[
                {
                  required: true,
                  whitespace: true,
                  message: '请输入纳税人识别号/税号',
                },
                { max: 50, message: '税号不能超过 50 位' },
              ]}
            >
              <Input
                placeholder="18 位纳税人识别号（自动大写）"
                maxLength={50}
              />
            </Form.Item>
          </Col>
          <Col span={10}>
            <Form.Item
              name="defaultInvoiceType"
              label="默认发票类型"
              rules={[{ required: true, message: '请选择发票类型' }]}
            >
              <Select
                options={[
                  { label: '增值税普通发票', value: 'NORMAL' },
                  { label: '增值税专用发票', value: 'SPECIAL' },
                ]}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="bankName" label="开户银行（选填）">
              <Input placeholder="例如：中国工商银行上海分行" maxLength={100} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="bankAccount" label="银行账号（选填）">
              <Input placeholder="开户行银行账号" maxLength={50} />
            </Form.Item>
          </Col>
          <Col span={15}>
            <Form.Item name="registeredAddress" label="开票地址（选填）">
              <Input placeholder="企业注册地址" maxLength={200} />
            </Form.Item>
          </Col>
          <Col span={9}>
            <Form.Item name="registeredPhone" label="开票电话（选填）">
              <Input placeholder="注册联系电话" maxLength={50} />
            </Form.Item>
          </Col>
          <Col span={24}>
            <Form.Item
              name="isDefault"
              valuePropName="checked"
              style={{ marginBottom: 0 }}
            >
              <Checkbox>设为该客户的默认首选开票抬头</Checkbox>
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
