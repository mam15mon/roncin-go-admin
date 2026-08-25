import {
  DeleteOutlined,
  FileTextOutlined,
  InfoCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import {
  ProFormDigit,
  ProFormList,
  ProFormSelect,
  ProFormText,
} from '@ant-design/pro-components';
import {
  AutoComplete,
  Button,
  Col,
  Form,
  Input,
  Popconfirm,
  Tag,
  Tooltip,
} from 'antd';
import React from 'react';

type SelectOption = { label: string; value: string | number };

export interface HouseDocItem {
  key: string;
  id?: string;
  houseNo: string;
  releaseType?: string;
  note?: string;
  status?: number;
  omitWhenEmpty?: boolean;
}

export interface MasterDocGroup {
  key: string;
  masterNo: string;
  houses: HouseDocItem[];
}

const RELEASE_TYPE_OPTIONS = [
  { label: '电放 (TELEX)', value: 'TELEX' },
  { label: '正本 (ORIGINAL)', value: 'ORIGINAL' },
  { label: '海运单 (SEAWAYBILL)', value: 'SEAWAYBILL' },
  { label: '放行条 (RELEASE)', value: 'RELEASE' },
  { label: '正本寄单 (MAILED)', value: 'MAILED' },
];

type ShippingDocumentFormValue = API.OrderShippingDocumentInput & {
  status?: number;
};

type OrderShippingDocumentFieldsProps = {
  disabled?: boolean;
  transportMode?: 'sea' | 'air';
};

let docKeySeq = 0;
const nextKey = (prefix: string) => `${prefix}_${Date.now()}_${++docKeySeq}`;

function rawDocsToGroups(
  rawDocs?: ShippingDocumentFormValue[],
): MasterDocGroup[] {
  if (!rawDocs || rawDocs.length === 0) {
    return [
      {
        key: nextKey('mg'),
        masterNo: '',
        houses: [
          { key: nextKey('h'), houseNo: '', releaseType: undefined, note: '' },
        ],
      },
    ];
  }

  const groupMap = new Map<string, MasterDocGroup>();
  const groups: MasterDocGroup[] = [];

  for (const doc of rawDocs) {
    const rawMaster = doc.masterNo || '';
    const masterKey = rawMaster.trim().toLowerCase();

    let group = groupMap.get(masterKey);
    if (!group) {
      group = {
        key: nextKey('mg'),
        masterNo: rawMaster,
        houses: [],
      };
      groupMap.set(masterKey, group);
      groups.push(group);
    }

    group.houses.push({
      key: nextKey('h'),
      id: doc.id,
      houseNo: doc.houseNo || '',
      releaseType: doc.releaseType,
      note: doc.note,
      status: doc.status,
      omitWhenEmpty: false,
    });
  }

  for (const g of groups) {
    if (g.houses.length === 0) {
      g.houses.push({
        key: nextKey('h'),
        houseNo: '',
        releaseType: undefined,
        note: '',
      });
    }
  }

  return groups.length > 0
    ? groups
    : [
        {
          key: nextKey('mg'),
          masterNo: '',
          houses: [
            {
              key: nextKey('h'),
              houseNo: '',
              releaseType: undefined,
              note: '',
            },
          ],
        },
      ];
}

function groupsToRawDocs(
  groups: MasterDocGroup[],
): API.OrderShippingDocumentInput[] {
  const result: API.OrderShippingDocumentInput[] = [];
  for (const g of groups) {
    const masterNo = g.masterNo;
    for (const h of g.houses) {
      const isEmptyPlaceholder =
        !h.id &&
        !h.houseNo.trim() &&
        !h.releaseType?.trim() &&
        !h.note?.trim() &&
        (!masterNo.trim() || h.omitWhenEmpty);
      if (isEmptyPlaceholder) {
        continue;
      }
      result.push({
        id: h.id,
        masterNo,
        houseNo: h.houseNo,
        releaseType: h.releaseType,
        note: h.note,
      });
    }
  }
  return result;
}

function getShippingDocumentsValidationMessage(
  docs?: ShippingDocumentFormValue[],
): string | undefined {
  const houseNos = new Set<string>();

  for (const doc of docs || []) {
    if (!doc.masterNo?.trim()) {
      return '请填写主单号';
    }
    const houseNo = doc.houseNo?.trim();
    if (!houseNo) {
      return '请填写分单号';
    }

    const normalizedHouseNo = houseNo.toLowerCase();
    if (houseNos.has(normalizedHouseNo)) {
      return `分单号 ${houseNo} 重复`;
    }
    houseNos.add(normalizedHouseNo);
  }

  return undefined;
}

function getDuplicateMasterNo(groups: MasterDocGroup[]): string | undefined {
  const masterNos = new Set<string>();

  for (const group of groups) {
    const masterNo = group.masterNo.trim();
    if (!masterNo) {
      continue;
    }
    const normalizedMasterNo = masterNo.toLowerCase();
    if (masterNos.has(normalizedMasterNo)) {
      return masterNo;
    }
    masterNos.add(normalizedMasterNo);
  }

  return undefined;
}

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

  const handleAddGroup = () => {
    const next = [
      ...groups,
      {
        key: nextKey('mg'),
        masterNo: '',
        houses: [
          { key: nextKey('h'), houseNo: '', releaseType: undefined, note: '' },
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
              key: nextKey('mg'),
              masterNo: '',
              houses: [
                {
                  key: nextKey('h'),
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
              key: nextKey('h'),
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
                    key: nextKey('h'),
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
    transportMode === 'sea' ? RELEASE_TYPE_OPTIONS : [];
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
    <Col span={24} style={{ marginBottom: 16 }}>
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
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
        }}
      >
        {groups.map((group, groupIdx) => {
          const masterTrimmed = group.masterNo?.trim();
          const groupHasReleased = group.houses.some(
            (house) => house.status === 3,
          );
          const groupHasContent = group.houses.some(
            (house) =>
              house.id ||
              house.houseNo.trim() ||
              house.releaseType?.trim() ||
              house.note?.trim(),
          );
          const masterMissing = !masterTrimmed && Boolean(groupHasContent);
          const normalizedMasterNo = masterTrimmed.toLowerCase();
          const masterDuplicate =
            Boolean(normalizedMasterNo) &&
            (masterNoCounts.get(normalizedMasterNo) || 0) > 1;
          const removeGroupButton = (
            <Button
              type="text"
              danger
              size="small"
              icon={<DeleteOutlined />}
              disabled={groupHasReleased}
              onClick={() => handleRemoveGroup(groupIdx)}
              style={{ fontSize: 12 }}
            >
              删除该主单组
            </Button>
          );
          return (
            <div
              key={group.key}
              style={{
                background: '#ffffff',
                border: '1px solid #f0f0f0',
                borderRadius: 6,
                padding: '12px 16px',
                boxShadow: '0 1px 2px rgba(0, 0, 0, 0.02)',
              }}
            >
              {/* 主单行 Header */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  marginBottom: 12,
                  flexWrap: 'wrap',
                }}
              >
                <div
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 6,
                    fontSize: 13,
                    fontWeight: 600,
                    color: '#262626',
                    minWidth: 100,
                  }}
                >
                  <FileTextOutlined
                    style={{ color: '#1677ff', fontSize: 14 }}
                  />
                  <span>主单 ({documentLabels.master})</span>
                  {groups.length > 1 && (
                    <Tag
                      bordered={false}
                      color="default"
                      style={{ marginInlineStart: 2, fontSize: 11 }}
                    >
                      #{groupIdx + 1}
                    </Tag>
                  )}
                </div>

                <div style={{ flex: '1 1 260px', maxWidth: 420 }}>
                  <Input
                    value={group.masterNo}
                    onChange={(e) =>
                      handleMasterChange(groupIdx, e.target.value)
                    }
                    placeholder={`请输入主单号 (如 ${documentLabels.master}-001)`}
                    maxLength={64}
                    disabled={disabled || groupHasReleased}
                    status={
                      masterMissing || masterDuplicate ? 'error' : undefined
                    }
                    allowClear
                  />
                  {(masterMissing || masterDuplicate) && (
                    <div
                      style={{ color: '#ff4d4f', fontSize: 12, marginTop: 2 }}
                    >
                      {masterDuplicate
                        ? '该主单已在当前操作票中，请在原主单组下添加分单'
                        : '请填写主单号'}
                    </div>
                  )}
                </div>

                <Tooltip title="当前操作票可加拼多张不同主单；同一主单下请直接添加分单。其他操作票使用相同主单号时，系统会归入同一主单批次（忽略大小写与首尾空格）。">
                  <div
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 4,
                      fontSize: 12,
                      color: '#8c8c8c',
                      cursor: 'help',
                    }}
                  >
                    <InfoCircleOutlined style={{ color: '#1677ff' }} />
                    <span>一主多分</span>
                  </div>
                </Tooltip>

                {masterTrimmed && (
                  <Tag
                    color="blue"
                    bordered={false}
                    style={{ margin: 0, fontSize: 11 }}
                  >
                    批次标识: {masterTrimmed.toUpperCase()}
                  </Tag>
                )}

                <div style={{ flex: 1 }} />

                {groups.length > 1 && !disabled && (
                  <Tooltip
                    title={
                      groupHasReleased
                        ? '组内存在已放货分单，不能删除该主单组'
                        : undefined
                    }
                  >
                    <span>
                      {group.houses.some((house) => house.id) &&
                      !groupHasReleased ? (
                        <Popconfirm
                          title="确认删除该主单组？"
                          description="保存订单后，组内已有分单会被删除。"
                          onConfirm={() => handleRemoveGroup(groupIdx)}
                          okText="删除"
                          cancelText="取消"
                        >
                          {React.cloneElement(removeGroupButton, {
                            onClick: undefined,
                          })}
                        </Popconfirm>
                      ) : (
                        removeGroupButton
                      )}
                    </span>
                  </Tooltip>
                )}
              </div>

              {/* 分单列表 Sub-Table */}
              <div
                style={{
                  border: '1px solid #f0f0f0',
                  borderRadius: 4,
                  overflow: 'hidden',
                  background: '#fafafa',
                }}
              >
                {/* 列表表头 */}
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    padding: '6px 12px',
                    background: '#fafafa',
                    borderBottom: '1px solid #f0f0f0',
                    fontSize: 12,
                    fontWeight: 500,
                    color: '#595959',
                  }}
                >
                  <div
                    style={{ width: 28, textAlign: 'center', color: '#8c8c8c' }}
                  >
                    #
                  </div>
                  <div
                    style={{
                      flex: '1 1 180px',
                      minWidth: 140,
                      paddingInline: 6,
                    }}
                  >
                    <span style={{ color: '#ff4d4f', marginRight: 4 }}>*</span>
                    分单号 ({documentLabels.house})
                  </div>
                  <div
                    style={{
                      flex: '0 1 180px',
                      minWidth: 140,
                      paddingInline: 6,
                    }}
                  >
                    放货类型
                  </div>
                  <div
                    style={{
                      flex: '2 1 240px',
                      minWidth: 160,
                      paddingInline: 6,
                    }}
                  >
                    分单备注
                  </div>
                  <div style={{ width: 44, textAlign: 'center' }}>操作</div>
                </div>

                {/* 列表内容 */}
                <div style={{ background: '#ffffff', padding: '6px 12px' }}>
                  {group.houses.map((house, houseIdx) => {
                    const isOnlyOne =
                      group.houses.length === 1 && groups.length === 1;
                    const isReleased = house.status === 3;
                    const normalizedHouseNo = house.houseNo
                      .trim()
                      .toLowerCase();
                    const isDuplicate =
                      Boolean(normalizedHouseNo) &&
                      (houseNoCounts.get(normalizedHouseNo) || 0) > 1;
                    const houseHasContent = Boolean(
                      house.id ||
                        (!house.omitWhenEmpty && group.masterNo.trim()) ||
                        house.houseNo.trim() ||
                        house.releaseType?.trim() ||
                        house.note?.trim(),
                    );
                    const houseMissing =
                      houseHasContent && !house.houseNo.trim();
                    const removeHouseButton = (
                      <Button
                        type="text"
                        size="small"
                        danger
                        aria-label={`删除分单 ${house.houseNo || houseIdx + 1}`}
                        disabled={isReleased}
                        icon={<DeleteOutlined style={{ fontSize: 12 }} />}
                        onClick={() => handleRemoveHouse(groupIdx, houseIdx)}
                        style={{
                          height: 24,
                          width: 24,
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          padding: 0,
                        }}
                      />
                    );
                    return (
                      <div
                        key={house.key}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          paddingBlock: 4,
                          gap: 10,
                        }}
                      >
                        <div
                          style={{
                            width: 28,
                            textAlign: 'center',
                            fontSize: 12,
                            color: '#8c8c8c',
                            fontWeight: 500,
                          }}
                        >
                          {houseIdx + 1}
                        </div>

                        <div style={{ flex: '1 1 180px', minWidth: 140 }}>
                          <Input
                            size="small"
                            value={house.houseNo}
                            onChange={(e) =>
                              handleHouseFieldChange(
                                groupIdx,
                                houseIdx,
                                'houseNo',
                                e.target.value,
                              )
                            }
                            placeholder={`请输入分单号 (如 ${documentLabels.house}-001)`}
                            maxLength={64}
                            disabled={disabled || isReleased}
                            status={
                              houseMissing || isDuplicate ? 'error' : undefined
                            }
                          />
                          {(houseMissing || isDuplicate) && (
                            <div
                              style={{
                                color: '#ff4d4f',
                                fontSize: 12,
                                marginTop: 2,
                              }}
                            >
                              {isDuplicate ? '分单号重复' : '请填写分单号'}
                            </div>
                          )}
                        </div>

                        <div style={{ flex: '0 1 180px', minWidth: 140 }}>
                          <AutoComplete
                            size="small"
                            value={house.releaseType}
                            options={releaseTypeOptions}
                            onChange={(val) =>
                              handleHouseFieldChange(
                                groupIdx,
                                houseIdx,
                                'releaseType',
                                val,
                              )
                            }
                            placeholder={
                              transportMode === 'sea'
                                ? '如 电放 / 正本'
                                : '请输入放货类型'
                            }
                            disabled={disabled || isReleased}
                            filterOption={(inputValue, option) =>
                              Boolean(
                                option?.value
                                  ?.toString()
                                  .toUpperCase()
                                  .includes(inputValue.toUpperCase()) ||
                                  option?.label
                                    ?.toString()
                                    .toUpperCase()
                                    .includes(inputValue.toUpperCase()),
                              )
                            }
                            allowClear
                            style={{ width: '100%' }}
                          />
                        </div>

                        <div style={{ flex: '2 1 240px', minWidth: 160 }}>
                          <Input
                            size="small"
                            value={house.note}
                            onChange={(e) =>
                              handleHouseFieldChange(
                                groupIdx,
                                houseIdx,
                                'note',
                                e.target.value,
                              )
                            }
                            placeholder="分单备注说明 (选填)"
                            maxLength={500}
                            disabled={disabled || isReleased}
                            allowClear
                          />
                        </div>

                        <div style={{ width: 44, textAlign: 'center' }}>
                          {!disabled && (
                            <Tooltip
                              title={
                                isReleased
                                  ? '已放货分单不能修改或删除'
                                  : isOnlyOne
                                    ? '清空该行'
                                    : '删除该分单'
                              }
                            >
                              <span>
                                {house.id && !isReleased ? (
                                  <Popconfirm
                                    title="确认删除该分单？"
                                    description="保存订单后，该分单会被删除。"
                                    onConfirm={() =>
                                      handleRemoveHouse(groupIdx, houseIdx)
                                    }
                                    okText="删除"
                                    cancelText="取消"
                                  >
                                    {React.cloneElement(removeHouseButton, {
                                      onClick: undefined,
                                    })}
                                  </Popconfirm>
                                ) : (
                                  removeHouseButton
                                )}
                              </span>
                            </Tooltip>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>

                {/* 添加分单按钮 */}
                {!disabled && (
                  <div
                    style={{
                      padding: '6px 12px 8px',
                      background: '#ffffff',
                      borderTop: '1px dashed #f0f0f0',
                    }}
                  >
                    <Button
                      type="dashed"
                      size="small"
                      icon={<PlusOutlined style={{ fontSize: 11 }} />}
                      onClick={() => handleAddHouse(groupIdx)}
                      style={{ fontSize: 12 }}
                    >
                      添加分单 ({documentLabels.house})
                    </Button>
                  </div>
                )}
              </div>
            </div>
          );
        })}

        {/* 加拼更多不同主单 */}
        {!disabled && (
          <Button
            type="dashed"
            icon={<PlusOutlined />}
            onClick={handleAddGroup}
            style={{
              width: '100%',
              borderRadius: 6,
              height: 36,
              color: '#1677ff',
              borderColor: '#b7d4ff',
              background: '#f9fcff',
            }}
          >
            加拼主单 ({documentLabels.master})
          </Button>
        )}
      </div>
    </Col>
  );
}

export function OrderContainerRequestFields({
  options,
}: {
  options: SelectOption[];
}) {
  return (
    <Col span={24}>
      <ProFormList
        name="containerRequests"
        label="计划箱型箱量"
        tooltip="这里只维护订单级配箱计划；实际箱号、封号与箱货分配在箱货信息中维护"
        creatorButtonProps={{ creatorButtonText: '新增计划箱型箱量' }}
        creatorRecord={{ containerSpecId: '', quantity: 1 }}
        copyIconProps={false}
        deleteIconProps={{ tooltipText: '删除该计划箱型箱量' }}
        itemContainerRender={(doms) => (
          <div style={{ width: '100%' }}>{doms}</div>
        )}
      >
        <ProFormText name="id" hidden />
        <ProFormSelect
          name="containerSpecId"
          label="箱型"
          options={options}
          placeholder="请选择箱型"
          rules={[{ required: true, message: '请选择箱型' }]}
          fieldProps={{ showSearch: true, optionFilterProp: 'label' }}
          width="md"
        />
        <ProFormDigit
          name="quantity"
          label="箱量"
          min={1}
          max={999}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入箱量' }]}
          width="sm"
        />
      </ProFormList>
    </Col>
  );
}
