import { Col, Form, Input, Row, Select } from 'antd';
import React, { useEffect, useState } from 'react';
import { MODAL_SIZE, SearchableSelect } from '@/components/ui';
import { QuickCreateModal } from '@/components/ui/quick-create-modal';
import { MASTER_DATA_KINDS } from '@/pages/orders/common';
import { feeCatalogServiceCreateFeeSetting } from '@/services/roncin/feeCatalogService';
import { masterDataServiceListItems } from '@/services/roncin/masterDataService';

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
  const [chargeCategories, setChargeCategories] = useState<
    API.MasterDataItem[]
  >([]);

  useEffect(() => {
    if (!open) {
      return;
    }
    let cancelled = false;
    masterDataServiceListItems({
      kind: MASTER_DATA_KINDS.SERVICE_TYPE,
      enabled: true,
      page: 1,
      pageSize: 200,
    })
      .then((res) => {
        if (!cancelled) {
          setChargeCategories(res.data ?? []);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setChargeCategories([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  return (
    <QuickCreateModal<API.CreateFeeSettingRequest, API.FeeSetting>
      title="快捷新增费用科目"
      open={open}
      width={MODAL_SIZE.SM}
      onCancel={onCancel}
      onSuccess={onSuccess}
      onSubmit={async (values) => {
        const res = await feeCatalogServiceCreateFeeSetting(values);
        return res.data;
      }}
    >
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
          <Form.Item
            name="chargeCategoryId"
            label="费用大类"
            rules={[{ required: true, message: '请选择费用大类' }]}
          >
            <SearchableSelect
              placeholder="请选择费用大类"
              options={chargeCategories.map((item) => ({
                label: item.code
                  ? `${item.name} (${item.code})`
                  : item.name || '',
                value: item.id ?? '',
              }))}
            />
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
            <SearchableSelect
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
            <SearchableSelect
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
            <SearchableSelect
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
    </QuickCreateModal>
  );
}
