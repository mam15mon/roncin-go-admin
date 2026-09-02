import {
  DeleteOutlined,
  PlusCircleFilled,
} from '@ant-design/icons';
import { Button, Col, Form, Input, InputNumber, Select } from 'antd';
import React from 'react';
import { OrderShippingDocumentStatus } from '@/enums.generated';
import {
  SEA_HOUSE_RELEASE_TYPE_OPTIONS,
  type HouseDocItem,
  type SelectOption,
} from './order-plan-constants';
import {
  getShippingDocumentsValidationMessage,
  housesToRawDocs,
  nextDocKey,
  rawDocsToHouses,
  type ShippingDocumentFormValue,
} from './order-shipping-doc-helpers';

export * from './order-plan-constants';
export * from './order-shipping-doc-helpers';

type OrderShippingDocumentFieldsProps = {
  disabled?: boolean;
  transportMode?: 'sea' | 'air';
};

function ShippingDocumentsFormControl(_props: {
  value?: ShippingDocumentFormValue[];
  onChange?: (value: ShippingDocumentFormValue[]) => void;
}) {
  return null;
}

export function OrderShippingDocumentFields({
  disabled = false,
  transportMode = 'sea',
}: OrderShippingDocumentFieldsProps = {}) {
  const form = Form.useFormInstance();
  const [houses, setHouses] = React.useState<HouseDocItem[]>(() => {
    const initial = form?.getFieldValue('shippingDocuments');
    return rawDocsToHouses(initial);
  });

  const watchedDocuments = Form.useWatch('shippingDocuments', {
    form,
    preserve: true,
  }) as ShippingDocumentFormValue[] | undefined;

  // 同步表单回显、重置与切换编辑记录带来的外部变化。
  React.useEffect(() => {
    const current = watchedDocuments;
    if (Array.isArray(current)) {
      const currentFlat = housesToRawDocs(houses);
      const isSame =
        current.length === currentFlat.length &&
        current.every(
          (doc, idx) =>
            doc.id === currentFlat[idx]?.id &&
            (doc.houseNo || '') === (currentFlat[idx]?.houseNo || '') &&
            (doc.releaseType || '') === (currentFlat[idx]?.releaseType || '') &&
            (doc.note || '') === (currentFlat[idx]?.note || ''),
        );
      if (!isSame) {
        setHouses(rawDocsToHouses(current));
      }
    }
  }, [houses, watchedDocuments]);

  const updateHouses = (nextHouses: HouseDocItem[]) => {
    setHouses(nextHouses);
    const flat = housesToRawDocs(nextHouses);
    form?.setFieldValue('shippingDocuments', flat);
  };

  const handleAddHouse = () => {
    const next = [
      ...houses,
      {
        key: nextDocKey('h'),
        houseNo: '',
        releaseType: undefined,
        note: '',
      },
    ];
    updateHouses(next);
  };

  const handleRemoveHouse = (index: number) => {
    const next = houses.filter((_, idx) => idx !== index);
    updateHouses(
      next.length > 0
        ? next
        : [
            {
              key: nextDocKey('h'),
              houseNo: '',
              releaseType: undefined,
              note: '',
              omitWhenEmpty: true,
            },
          ],
    );
  };

  const handleFieldChange = (
    index: number,
    field: 'houseNo' | 'releaseType' | 'note',
    val: string | undefined,
  ) => {
    const next = houses.map((item, idx) => {
      if (idx === index) {
        return { ...item, [field]: val, omitWhenEmpty: false };
      }
      return item;
    });
    updateHouses(next);
  };

  const releaseTypeOptions =
    transportMode === 'sea' ? SEA_HOUSE_RELEASE_TYPE_OPTIONS : [];

  return (
    <>
      <Form.Item
        name="shippingDocuments"
        hidden
        rules={[
          {
            validator: async (
              _,
              value: ShippingDocumentFormValue[] | undefined,
            ) => {
              const message = getShippingDocumentsValidationMessage(value);
              if (message) {
                throw new Error(message);
              }
            },
          },
        ]}
      >
        <ShippingDocumentsFormControl />
      </Form.Item>

      <Col span={24} style={{ marginBottom: 12 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 8,
          }}
        >
          <span style={{ fontWeight: 600 }}>分单信息 (HBL)</span>
          {!disabled && (
            <Button
              type="dashed"
              size="small"
              icon={<PlusCircleFilled />}
              onClick={handleAddHouse}
            >
              添加分单 (HBL)
            </Button>
          )}
        </div>

        {houses.map((house, idx) => {
          const isReleased =
            house.status ===
            OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED;
          return (
            <div
              key={house.key}
              style={{
                display: 'flex',
                gap: 12,
                alignItems: 'center',
                marginBottom: 8,
                background: '#fafafa',
                padding: '8px 12px',
                borderRadius: 4,
                border: '1px solid #f0f0f0',
              }}
            >
              <div style={{ flex: 1 }}>
                <Input
                  placeholder="分单号 (HBL)"
                  value={house.houseNo}
                  disabled={disabled || isReleased}
                  onChange={(e) =>
                    handleFieldChange(idx, 'houseNo', e.target.value)
                  }
                />
              </div>
              <div style={{ width: 180 }}>
                <Select
                  placeholder="分单签放方式"
                  value={house.releaseType}
                  options={releaseTypeOptions}
                  disabled={disabled || isReleased}
                  allowClear
                  style={{ width: '100%' }}
                  onChange={(val) =>
                    handleFieldChange(idx, 'releaseType', val)
                  }
                />
              </div>
              <div style={{ flex: 1 }}>
                <Input
                  placeholder="备注"
                  value={house.note}
                  disabled={disabled || isReleased}
                  onChange={(e) =>
                    handleFieldChange(idx, 'note', e.target.value)
                  }
                />
              </div>
              {!disabled && !isReleased && houses.length > 1 && (
                <Button
                  type="text"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => handleRemoveHouse(idx)}
                />
              )}
            </div>
          );
        })}
      </Col>
    </>
  );
}

export function OrderContainerRequestFields({
  options,
}: {
  options: SelectOption[];
}) {
  const form = Form.useFormInstance();
  const containerRequests = (Form.useWatch('containerRequests', form) ??
    []) as API.OrderContainerRequestInput[];

  const handleAdd = () => {
    const next = [
      ...containerRequests,
      { containerSpecId: options[0]?.value || '', quantity: 1 },
    ];
    form?.setFieldValue('containerRequests', next);
  };

  const handleRemove = (index: number) => {
    const next = containerRequests.filter((_, idx) => idx !== index);
    form?.setFieldValue('containerRequests', next);
  };

  const handleChange = (
    index: number,
    field: 'containerSpecId' | 'quantity',
    val: any,
  ) => {
    const next = containerRequests.map((item, idx) => {
      if (idx === index) {
        return { ...item, [field]: val };
      }
      return item;
    });
    form?.setFieldValue('containerRequests', next);
  };

  return (
    <Col span={24}>
      <Form.Item
        name="containerRequests"
        hidden
      >
        <input type="hidden" />
      </Form.Item>
      <Form.Item
        label="箱型箱量"
        tooltip="这里只维护订单级配箱计划；实际箱号、封号与箱货分配在箱货信息中维护"
        style={{ marginBottom: 12 }}
      >
        <span style={{ display: 'none' }}>计划箱型箱量</span>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            flexWrap: 'wrap',
          }}
        >
          {containerRequests.length === 0 ? (
            <div style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
              <Select
                placeholder="请选择"
                options={options}
                showSearch
                optionFilterProp="label"
                style={{ width: 140 }}
                onChange={(val) => {
                  form?.setFieldValue('containerRequests', [
                    { containerSpecId: val, quantity: 1 },
                  ]);
                }}
              />
              <InputNumber
                min={1}
                max={999}
                defaultValue={1}
                precision={0}
                style={{ width: 90 }}
                disabled
              />
              <Button
                type="link"
                icon={<PlusCircleFilled style={{ color: '#1677ff' }} />}
                onClick={handleAdd}
                style={{ padding: 0, height: 32, fontSize: 13 }}
              >
                新增计划箱型箱量
              </Button>
            </div>
          ) : (
            <>
              {containerRequests.map((req, idx) => (
                <div
                  key={req.id || `cr-${req.containerSpecId}-${idx}`}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 6,
                    background: '#fafafa',
                    padding: '2px 8px',
                    borderRadius: 4,
                    border: '1px solid #f0f0f0',
                  }}
                >
                  <Select
                    value={req.containerSpecId || undefined}
                    placeholder="请选择"
                    options={options}
                    showSearch
                    optionFilterProp="label"
                    style={{ width: 130 }}
                    onChange={(val) =>
                      handleChange(idx, 'containerSpecId', val)
                    }
                  />
                  <InputNumber
                    value={req.quantity || 1}
                    min={1}
                    max={999}
                    precision={0}
                    style={{ width: 80 }}
                    onChange={(val) =>
                      handleChange(idx, 'quantity', val ?? 1)
                    }
                  />
                  <Button
                    type="text"
                    danger
                    size="small"
                    icon={<DeleteOutlined style={{ fontSize: 12 }} />}
                    onClick={() => handleRemove(idx)}
                    style={{ padding: '0 4px', height: 24 }}
                  />
                </div>
              ))}
              <Button
                type="link"
                icon={<PlusCircleFilled style={{ color: '#1677ff' }} />}
                onClick={handleAdd}
                style={{ padding: 0, height: 32, fontSize: 13 }}
              >
                新增计划箱型箱量
              </Button>
            </>
          )}
        </div>
      </Form.Item>
    </Col>
  );
}
