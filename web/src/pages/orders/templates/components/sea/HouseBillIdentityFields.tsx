import { ProFormText } from '@ant-design/pro-components';
import { Col, Form, Radio, Row } from 'antd';
import React, { useEffect } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import { searchPartnerOptions } from '@/features/partners';

/** 与服务端 NormalizeSeaHouseNo 同口径的展示层归一化：NFC + 去首尾空白 + 大写。
 * 仅用于批次排重的即时提示（体验层），服务端事务内预查为权威。 */
export function normalizeSeaHouseNoForCompare(value: string): string {
  return value.normalize('NFC').trim().toUpperCase();
}

type HouseBillFormKey = 'seaHouseBill' | 'newHouseBill';

export function HouseBillIdentityFields({
  fieldKey,
  disabled = false,
}: {
  fieldKey: HouseBillFormKey;
  disabled?: boolean;
}) {
  const form = Form.useFormInstance();
  // 批次分单号清单（候选响应）晚于分单号录入到达时补一次重复校验：
  // form.setFieldValue 写入不触发 antd dependencies 重校验，这里显式补检；
  // 分单号为空时不触发，避免提前抛出必填错误。
  // 该字段无 Form.Item 注册，必须以 preserve 读取全量 store 值。
  const batchHouseNos = Form.useWatch('seaMasterBillBatchHouseNos', {
    form,
    preserve: true,
  });
  useEffect(() => {
    if (!batchHouseNos?.length) return;
    const value = form.getFieldValue([fieldKey, 'houseNo']);
    if (typeof value === 'string' && normalizeSeaHouseNoForCompare(value)) {
      void form.validateFields([[fieldKey, 'houseNo']]).catch(() => undefined);
    }
  }, [batchHouseNos, fieldKey, form]);
  return (
    <Row gutter={[16, 0]}>
      <Col xs={24} md={8}>
        <ProFormText
          name={[fieldKey, 'houseNo']}
          label="分单号 (HBL No.)"
          placeholder="请输入分单号"
          disabled={disabled}
          layout="vertical"
          rules={[
            { required: true, whitespace: true, message: '分单号不能为空' },
            ({ getFieldValue }) => ({
              validator(_rule, value) {
                const normalized = normalizeSeaHouseNoForCompare(
                  typeof value === 'string' ? value : '',
                );
                if (!normalized) return Promise.resolve();
                // 命中共享主单批次时与批次内兄弟票分单号（含作废）即时比对。
                const batchHouseNos = (getFieldValue(
                  'seaMasterBillBatchHouseNos',
                ) ?? []) as string[];
                if (batchHouseNos.includes(normalized)) {
                  return Promise.reject(
                    new Error(
                      `分单号 ${normalized} 在该主单批次内已存在（含作废），请更换分单号`,
                    ),
                  );
                }
                return Promise.resolve();
              },
            }),
          ]}
          fieldProps={{ maxLength: 128 }}
        />
      </Col>
      <Col xs={24} md={16}>
        <ProFormText
          name={[fieldKey, 'note']}
          label="分单备注"
          placeholder="请输入分单备注"
          disabled={disabled}
          layout="vertical"
          fieldProps={{ maxLength: 500 }}
        />
      </Col>
      <Col xs={24}>
        <Form.Item
          name={[fieldKey, 'issuerSource']}
          label="分单签发主体"
          required
          rules={[{ required: true, message: '请选择分单签发主体' }]}
          layout="vertical"
          style={{ marginBottom: 16 }}
        >
          <Radio.Group
            disabled={disabled}
            onChange={() =>
              form.setFieldValue([fieldKey, 'issuerPartnerId'], undefined)
            }
          >
            <Radio
              value={
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION
              }
            >
              本公司
            </Radio>
            <Radio
              value={
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER
              }
            >
              委托单位
            </Radio>
            <Radio
              value={
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
              }
            >
              指定合作方
            </Radio>
          </Radio.Group>
        </Form.Item>
      </Col>
      <Form.Item
        noStyle
        shouldUpdate={(previous, current) =>
          previous?.[fieldKey]?.issuerSource !==
          current?.[fieldKey]?.issuerSource
        }
      >
        {({ getFieldValue }) => {
          const issuerSource = getFieldValue([fieldKey, 'issuerSource']);
          if (
            issuerSource !==
            SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
          ) {
            return null;
          }
          return (
            <Col xs={24} md={8}>
              <ProFormSearchableSelect
                name={[fieldKey, 'issuerPartnerId']}
                label="签发合作方"
                placeholder="请选择签发主体合作伙伴"
                disabled={disabled}
                rules={[{ required: true, message: '请选择签发主体合作伙伴' }]}
                request={async ({ keyWords }) => searchPartnerOptions(keyWords)}
                fieldProps={{ filterOption: false }}
                formItemProps={{
                  style: { marginBottom: 24 },
                }}
              />
            </Col>
          );
        }}
      </Form.Item>
    </Row>
  );
}

export function isTerminalHouseBill(status?: number) {
  return status === SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED;
}

export function houseBillStatusPresentation(status?: number) {
  switch (status) {
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_RELEASED:
      return { color: 'success', text: '已签发' };
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_CONFIRMED:
      return { color: 'blue', text: '已确认' };
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED:
      return { color: 'error', text: '已作废' };
    default:
      return { color: 'default', text: '草稿' };
  }
}
