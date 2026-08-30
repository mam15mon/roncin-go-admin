import {
  DeleteOutlined,
  FileTextOutlined,
  InfoCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import {
  Alert,
  AutoComplete,
  Button,
  Input,
  Popconfirm,
  Select,
  Tag,
  Tooltip,
} from 'antd';
import React from 'react';
import { OrderShippingDocumentStatus } from '@/enums.generated';
import {
  SEA_MASTER_DOCUMENT_TYPE_OPTIONS,
  SEA_MASTER_RELEASE_METHOD_OPTIONS,
  type MasterDocGroup,
} from '../../order-plan-constants';

type OrderMasterDocGroupCardProps = {
  group: MasterDocGroup;
  groupIdx: number;
  totalGroups: number;
  disabled: boolean;
  transportMode: 'sea' | 'air';
  documentLabels: { master: string; house: string };
  releaseTypeOptions: { label: string; value: string }[];
  masterNoCounts: Map<string, number>;
  houseNoCounts: Map<string, number>;
  onMasterChange: (groupIdx: number, val: string) => void;
  onMasterAttributeChange: (
    groupIdx: number,
    field: 'masterDocumentType' | 'masterReleaseMethod',
    val: string,
  ) => void;
  onRemoveGroup: (groupIdx: number) => void;
  onAddHouse: (groupIdx: number) => void;
  onRemoveHouse: (groupIdx: number, houseIdx: number) => void;
  onHouseFieldChange: (
    groupIdx: number,
    houseIdx: number,
    field: 'houseNo' | 'releaseType' | 'note',
    val: string,
  ) => void;
};

export default function OrderMasterDocGroupCard({
  group,
  groupIdx,
  totalGroups,
  disabled,
  transportMode,
  documentLabels,
  releaseTypeOptions,
  masterNoCounts,
  houseNoCounts,
  onMasterChange,
  onMasterAttributeChange,
  onRemoveGroup,
  onAddHouse,
  onRemoveHouse,
  onHouseFieldChange,
}: OrderMasterDocGroupCardProps) {
  const masterTrimmed = group.masterNo?.trim();
  const groupHasReleased = group.houses.some(
    (house) =>
      house.status ===
      OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED,
  );
  const groupHasContent =
    group.houses.some(
      (house) =>
        house.id ||
        house.houseNo.trim() ||
        house.releaseType?.trim() ||
        house.note?.trim(),
    ) ||
    Boolean(
      group.masterDocumentType?.trim() || group.masterReleaseMethod?.trim(),
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
      onClick={() => onRemoveGroup(groupIdx)}
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
          <FileTextOutlined style={{ color: '#1677ff', fontSize: 14 }} />
          <span>主单 ({documentLabels.master})</span>
          {totalGroups > 1 && (
            <Tag
              variant="filled"
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
            onChange={(e) => onMasterChange(groupIdx, e.target.value)}
            placeholder={`请输入主单号 (如 ${documentLabels.master}-001)`}
            maxLength={64}
            disabled={disabled || groupHasReleased}
            status={masterMissing || masterDuplicate ? 'error' : undefined}
            allowClear
          />
          {(masterMissing || masterDuplicate) && (
            <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 2 }}>
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
            variant="filled"
            style={{ margin: 0, fontSize: 11 }}
          >
            批次标识: {masterTrimmed.toUpperCase()}
          </Tag>
        )}

        <div style={{ flex: 1 }} />

        {totalGroups > 1 && !disabled && (
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
                  onConfirm={() => onRemoveGroup(groupIdx)}
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

      {transportMode === 'sea' && (
        <div style={{ marginBottom: 12 }}>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 320px))',
              gap: 12,
              marginBottom: 8,
            }}
          >
            <div>
              <div
                style={{
                  color: '#595959',
                  fontSize: 12,
                  marginBottom: 4,
                }}
              >
                主单单证类型
              </div>
              <Select
                value={group.masterDocumentType}
                options={SEA_MASTER_DOCUMENT_TYPE_OPTIONS}
                placeholder="请选择主单单证类型"
                disabled={disabled || groupHasReleased}
                onChange={(value) =>
                  onMasterAttributeChange(
                    groupIdx,
                    'masterDocumentType',
                    value,
                  )
                }
                style={{ width: '100%' }}
              />
            </div>
            <div>
              <div
                style={{
                  color: '#595959',
                  fontSize: 12,
                  marginBottom: 4,
                }}
              >
                主单签放方式
              </div>
              <Select
                value={group.masterReleaseMethod}
                options={SEA_MASTER_RELEASE_METHOD_OPTIONS}
                placeholder="请选择主单签放方式"
                disabled={disabled || groupHasReleased}
                onChange={(value) =>
                  onMasterAttributeChange(
                    groupIdx,
                    'masterReleaseMethod',
                    value,
                  )
                }
                style={{ width: '100%' }}
              />
            </div>
          </div>
          <Alert
            type="warning"
            showIcon
            title="主单属性属于共享主单批次，修改后会影响其他引用同一主单的操作票。"
          />
        </div>
      )}

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
            分单签放方式
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
            const isOnlyOne = group.houses.length === 1 && totalGroups === 1;
            const isReleased =
              house.status ===
              OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED;
            const normalizedHouseNo = house.houseNo.trim().toLowerCase();
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
            const houseMissing = houseHasContent && !house.houseNo.trim();
            const removeHouseButton = (
              <Button
                type="text"
                size="small"
                danger
                aria-label={`删除分单 ${house.houseNo || houseIdx + 1}`}
                disabled={isReleased}
                icon={<DeleteOutlined style={{ fontSize: 12 }} />}
                onClick={() => onRemoveHouse(groupIdx, houseIdx)}
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
                      onHouseFieldChange(
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
                      onHouseFieldChange(
                        groupIdx,
                        houseIdx,
                        'releaseType',
                        val,
                      )
                    }
                    placeholder={
                      transportMode === 'sea'
                        ? '请选择或输入分单签放方式'
                        : '请输入分单签放方式'
                    }
                    disabled={disabled || isReleased}
                    showSearch={{
                      filterOption: (inputValue, option) =>
                        Boolean(
                          option?.value
                            ?.toString()
                            .toUpperCase()
                            .includes(inputValue.toUpperCase()) ||
                            option?.label
                              ?.toString()
                              .toUpperCase()
                              .includes(inputValue.toUpperCase()),
                        ),
                    }}
                    allowClear
                    style={{ width: '100%' }}
                  />
                </div>

                <div style={{ flex: '2 1 240px', minWidth: 160 }}>
                  <Input
                    size="small"
                    value={house.note}
                    onChange={(e) =>
                      onHouseFieldChange(
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
                              onRemoveHouse(groupIdx, houseIdx)
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
              onClick={() => onAddHouse(groupIdx)}
              style={{ fontSize: 12 }}
            >
              添加分单 ({documentLabels.house})
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
