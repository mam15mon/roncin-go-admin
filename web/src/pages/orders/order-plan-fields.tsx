import {
  DeleteOutlined,
  PlusCircleFilled,
} from '@ant-design/icons';
import { Button, Col, Form, InputNumber, Select } from 'antd';
import React from 'react';
import OrderMasterDocGroupCard from './components/docs/OrderMasterDocGroupCard';
import {
  SEA_HOUSE_RELEASE_TYPE_OPTIONS,
  type MasterDocGroup,
  type SelectOption,
} from './order-plan-constants';
import {
  getDuplicateMasterNo,
  getShippingDocumentsValidationMessage,
  groupsToRawDocs,
  nextDocKey,
  rawDocsToGroups,
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
  const [groups, setGroups] = React.useState<MasterDocGroup[]>(() => {
    const initial = form?.getFieldValue('shippingDocuments');
    return rawDocsToGroups(initial);
  });

  const watchedDocuments = Form.useWatch('shippingDocuments', {
    form,
    preserve: true,
  }) as ShippingDocumentFormValue[] | undefined;

  // 同步表单回显、重置与切换编辑记录带来的外部变化。
  React.useEffect(() => {
    const current = watchedDocuments;
    if (Array.isArray(current)) {
      const currentFlat = groupsToRawDocs(groups);
      const isSame =
        current.length === currentFlat.length &&
        current.every(
          (doc, idx) =>
            doc.id === currentFlat[idx]?.id &&
            (doc.masterNo || '') === (currentFlat[idx]?.masterNo || '') &&
            (doc.masterDocumentType || '') ===
              (currentFlat[idx]?.masterDocumentType || '') &&
            (doc.masterReleaseMethod || '') ===
              (currentFlat[idx]?.masterReleaseMethod || '') &&
            (doc.houseNo || '') === (currentFlat[idx]?.houseNo || '') &&
            (doc.releaseType || '') === (currentFlat[idx]?.releaseType || '') &&
            (doc.note || '') === (currentFlat[idx]?.note || ''),
        );
      if (!isSame) {
        setGroups(rawDocsToGroups(current));
      }
    }
  }, [groups, watchedDocuments]);

  const updateGroups = (nextGroups: MasterDocGroup[]) => {
    setGroups(nextGroups);
    const flat = groupsToRawDocs(nextGroups);
    form?.setFieldValue('shippingDocuments', flat);
  };

  const handleMasterChange = (groupIndex: number, val: string) => {
    const next = groups.map((g, idx) => {
      if (idx === groupIndex) {
        return { ...g, masterNo: val };
      }
      return g;
    });
    updateGroups(next);
  };

  const handleMasterAttributeChange = (
    groupIndex: number,
    field: 'masterDocumentType' | 'masterReleaseMethod',
    val: string,
  ) => {
    const next = groups.map((group, index) =>
      index === groupIndex ? { ...group, [field]: val } : group,
    );
    updateGroups(next);
  };

  const handleAddGroup = () => {
    const next = [
      ...groups,
      {
        key: nextDocKey('mg'),
        masterNo: '',
        houses: [
          {
            key: nextDocKey('h'),
            houseNo: '',
            releaseType: undefined,
            note: '',
          },
        ],
      },
    ];
    updateGroups(next);
  };

  const handleRemoveGroup = (groupIndex: number) => {
    const next = groups.filter((_, idx) => idx !== groupIndex);
    updateGroups(
      next.length > 0
        ? next
        : [
            {
              key: nextDocKey('mg'),
              masterNo: '',
              houses: [
                {
                  key: nextDocKey('h'),
                  houseNo: '',
                  releaseType: undefined,
                  note: '',
                },
              ],
            },
          ],
    );
  };

  const handleAddHouse = (groupIndex: number) => {
    const next = groups.map((g, idx) => {
      if (idx === groupIndex) {
        return {
          ...g,
          houses: [
            ...g.houses,
            {
              key: nextDocKey('h'),
              houseNo: '',
              releaseType: undefined,
              note: '',
              omitWhenEmpty: true,
            },
          ],
        };
      }
      return g;
    });
    updateGroups(next);
  };

  const handleRemoveHouse = (groupIndex: number, houseIndex: number) => {
    const next = groups.map((g, idx) => {
      if (idx === groupIndex) {
        const filtered = g.houses.filter((_, hIdx) => hIdx !== houseIndex);
        return {
          ...g,
          houses:
            filtered.length > 0
              ? filtered
              : [
                  {
                    key: nextDocKey('h'),
                    houseNo: '',
                    releaseType: undefined,
                    note: '',
                    omitWhenEmpty: true,
                  },
                ],
        };
      }
      return g;
    });
    updateGroups(next);
  };

  const handleHouseFieldChange = (
    groupIndex: number,
    houseIndex: number,
    field: 'houseNo' | 'releaseType' | 'note',
    val: string,
  ) => {
    const next = groups.map((g, idx) => {
      if (idx === groupIndex) {
        const newHouses = g.houses.map((h, hIdx) => {
          if (hIdx === houseIndex) {
            return { ...h, [field]: val };
          }
          return h;
        });
        return { ...g, houses: newHouses };
      }
      return g;
    });
    updateGroups(next);
  };

  const documentLabels =
    transportMode === 'air'
      ? { master: 'MAWB', house: 'HAWB' }
      : { master: 'MBL', house: 'HBL' };
  const releaseTypeOptions =
    transportMode === 'sea' ? SEA_HOUSE_RELEASE_TYPE_OPTIONS : [];
  const masterNoCounts = new Map<string, number>();
  const houseNoCounts = new Map<string, number>();
  for (const group of groups) {
    const normalizedMasterNo = group.masterNo.trim().toLowerCase();
    if (normalizedMasterNo) {
      masterNoCounts.set(
        normalizedMasterNo,
        (masterNoCounts.get(normalizedMasterNo) || 0) + 1,
      );
    }
    for (const house of group.houses) {
      const normalizedHouseNo = house.houseNo.trim().toLowerCase();
      if (normalizedHouseNo) {
        houseNoCounts.set(
          normalizedHouseNo,
          (houseNoCounts.get(normalizedHouseNo) || 0) + 1,
        );
      }
    }
  }

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
              const duplicateMasterNo = getDuplicateMasterNo(groups);
              if (duplicateMasterNo) {
                throw new Error(`主单号 ${duplicateMasterNo} 已在当前操作票中`);
              }
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
      {groups.map((group, groupIdx) => (
        <OrderMasterDocGroupCard
          key={group.key}
          group={group}
          groupIdx={groupIdx}
          totalGroups={groups.length}
          disabled={disabled}
          transportMode={transportMode}
          documentLabels={documentLabels}
          releaseTypeOptions={releaseTypeOptions}
          masterNoCounts={masterNoCounts}
          houseNoCounts={houseNoCounts}
          onMasterChange={handleMasterChange}
          onMasterAttributeChange={handleMasterAttributeChange}
          onRemoveGroup={handleRemoveGroup}
          onAddHouse={handleAddHouse}
          onRemoveHouse={handleRemoveHouse}
          onHouseFieldChange={handleHouseFieldChange}
          onAddGroup={handleAddGroup}
        />
      ))}
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
