import {
  DeleteOutlined,
  InfoCircleOutlined,
  PlusCircleFilled,
} from '@ant-design/icons';
import {
  Alert,
  AutoComplete,
  Button,
  Col,
  Form,
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
  onAddGroup?: () => void;
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
  onAddGroup,
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

  const firstHouse = group.houses[0] || {
    key: 'h-0',
    houseNo: '',
    releaseType: undefined,
    note: '',
  };
  const firstHouseTrimmed = firstHouse.houseNo.trim().toLowerCase();
  const firstHouseDuplicate =
    Boolean(firstHouseTrimmed) && (houseNoCounts.get(firstHouseTrimmed) || 0) > 1;
  const firstHouseHasContent = Boolean(
    firstHouse.id ||
      (!firstHouse.omitWhenEmpty && group.masterNo.trim()) ||
      firstHouse.houseNo.trim() ||
      firstHouse.releaseType?.trim() ||
      firstHouse.note?.trim(),
  );
  const firstHouseMissing = firstHouseHasContent && !firstHouse.houseNo.trim();
  const firstHouseReleased =
    firstHouse.status ===
    OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED;

  const removeGroupButton = (
    <Button
      type="link"
      danger
      size="small"
      disabled={disabled || groupHasReleased}
      icon={<DeleteOutlined style={{ fontSize: 12 }} />}
      onClick={() => onRemoveGroup(groupIdx)}
      style={{ padding: '0 2px', fontSize: 12, height: 32 }}
    >
      删除该主单组
    </Button>
  );

  const removeFirstHouseButton = (
    <Button
      type="link"
      danger
      size="small"
      disabled={disabled || firstHouseReleased}
      aria-label={`删除分单 ${firstHouse.houseNo || 1}`}
      icon={<DeleteOutlined style={{ fontSize: 12 }} />}
      onClick={() => onRemoveHouse(groupIdx, 0)}
      style={{ padding: '0 2px', height: 32, fontSize: 12, flexShrink: 0 }}
    >
      删除分单
    </Button>
  );

  return (
    <>
      {/* 主单与首条分单：5 列紧凑横排 */}
      <Col className="col-5">
        <Form.Item
          label={
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
              <span>{`主单 (${documentLabels.master})`}</span>
              {totalGroups > 1 && (
                <Tag color="blue" variant="filled" style={{ margin: 0, fontSize: 10, padding: '0 4px' }}>
                  #{groupIdx + 1}
                </Tag>
              )}
              {groupIdx === 0 && (
                <Tooltip title="当前操作票可加拼多张不同主单；同一主单下请直接添加分单。其他操作票使用相同主单号时，系统会归入同一主单批次（一主多分）。">
                  <span style={{ color: '#8c8c8c', cursor: 'help', fontSize: 12 }}>
                    <InfoCircleOutlined style={{ color: '#1677ff' }} />
                    <span style={{ display: 'none' }}>一主多分</span>
                  </span>
                </Tooltip>
              )}
            </span>
          }
          style={{ marginBottom: masterMissing || masterDuplicate ? 24 : 12 }}
          validateStatus={masterMissing || masterDuplicate ? 'error' : undefined}
          help={
            masterDuplicate
              ? '该主单已在当前操作票中，请在原主单组下添加分单'
              : masterMissing
                ? '请填写主单号'
                : undefined
          }
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <Input
              value={group.masterNo}
              onChange={(e) => onMasterChange(groupIdx, e.target.value)}
              placeholder={`请输入主单号 (如 ${documentLabels.master}-001)`}
              maxLength={64}
              disabled={disabled || groupHasReleased}
              status={masterMissing || masterDuplicate ? 'error' : undefined}
              allowClear
            />
            {groupIdx === 0 && !disabled && onAddGroup && (
              <Button
                type="link"
                icon={<PlusCircleFilled style={{ color: '#1677ff' }} />}
                onClick={onAddGroup}
                style={{ padding: '0 2px', height: 32, fontSize: 12, flexShrink: 0 }}
              >
                加拼主单 ({documentLabels.master})
              </Button>
            )}
            {totalGroups > 1 && (
              <span>
                {group.houses.some((house) => house.id) && !groupHasReleased && !disabled ? (
                  <Popconfirm
                    title="确认删除该主单组？"
                    description="保存订单后，组内已有分单会被删除。"
                    onConfirm={() => onRemoveGroup(groupIdx)}
                    okText="删除"
                    cancelText="取消"
                  >
                    {React.cloneElement(removeGroupButton, { onClick: undefined })}
                  </Popconfirm>
                ) : (
                  removeGroupButton
                )}
              </span>
            )}
          </div>
        </Form.Item>
      </Col>

      <Col className="col-5">
        <Form.Item
          label={`分单号 (${documentLabels.house})`}
          style={{ marginBottom: firstHouseMissing || firstHouseDuplicate ? 24 : 12 }}
          validateStatus={firstHouseMissing || firstHouseDuplicate ? 'error' : undefined}
          help={
            firstHouseDuplicate
              ? '分单号重复'
              : firstHouseMissing
                ? '请填写分单号'
                : undefined
          }
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <Input
              value={firstHouse.houseNo}
              onChange={(e) =>
                onHouseFieldChange(groupIdx, 0, 'houseNo', e.target.value)
              }
              placeholder={`请输入分单号 (如 ${documentLabels.house}-001)`}
              maxLength={64}
              disabled={disabled || firstHouseReleased}
              status={firstHouseMissing || firstHouseDuplicate ? 'error' : undefined}
              allowClear
            />
            {!disabled && (
              <Button
                type="link"
                icon={<PlusCircleFilled style={{ color: '#1677ff' }} />}
                onClick={() => onAddHouse(groupIdx)}
                style={{ padding: '0 2px', height: 32, fontSize: 12, flexShrink: 0 }}
              >
                添加分单 ({documentLabels.house})
              </Button>
            )}
            {(group.houses.length > 1 || firstHouse.id || firstHouseReleased || totalGroups > 1) && (
              <span>
                {firstHouse.id && !firstHouseReleased && !disabled ? (
                  <Popconfirm
                    title="确认删除该分单？"
                    description="保存订单后，该分单会被删除。"
                    onConfirm={() => onRemoveHouse(groupIdx, 0)}
                    okText="删除"
                    cancelText="取消"
                  >
                    {React.cloneElement(removeFirstHouseButton, { onClick: undefined })}
                  </Popconfirm>
                ) : (
                  removeFirstHouseButton
                )}
              </span>
            )}
          </div>
        </Form.Item>
      </Col>

      {transportMode === 'sea' && (
        <>
          <Col className="col-5">
            <Form.Item label="主单单证类型" style={{ marginBottom: 12 }}>
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
                allowClear
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>

          <Col className="col-5">
            <Form.Item label="主单签放方式" style={{ marginBottom: 12 }}>
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
                allowClear
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>

          <Col className="col-5">
            <Form.Item label="分单签放方式" style={{ marginBottom: 12 }}>
              <AutoComplete
                value={firstHouse.releaseType}
                options={releaseTypeOptions}
                onChange={(val) =>
                  onHouseFieldChange(groupIdx, 0, 'releaseType', val)
                }
                placeholder="请选择或输入分单签放方式"
                disabled={disabled || firstHouseReleased}
                allowClear
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
        </>
      )}

      {/* 共享主单批次说明 */}
      {transportMode === 'sea' && groupIdx === 0 && (
        <Col span={24} style={{ marginTop: -6, marginBottom: 8 }}>
          <Alert
            type="warning"
            showIcon
            title="主单属性属于共享主单批次，修改后会影响其他引用同一主单的操作票。"
            style={{ padding: '2px 8px', fontSize: 12, borderRadius: 4 }}
          />
        </Col>
      )}

      {/* 附加分单行（如果有 2 张及以上分单） */}
      {group.houses.slice(1).map((extraHouse, offset) => {
        const houseIdx = offset + 1;
        const normalizedNo = extraHouse.houseNo.trim().toLowerCase();
        const isDuplicate =
          Boolean(normalizedNo) && (houseNoCounts.get(normalizedNo) || 0) > 1;
        const hasContent = Boolean(
          extraHouse.id ||
            (!extraHouse.omitWhenEmpty && group.masterNo.trim()) ||
            extraHouse.houseNo.trim() ||
            extraHouse.releaseType?.trim() ||
            extraHouse.note?.trim(),
        );
        const isMissing = hasContent && !extraHouse.houseNo.trim();
        const isReleased =
          extraHouse.status ===
          OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED;

        return (
          <React.Fragment key={extraHouse.key}>
            <Col className="col-5">
              <Form.Item label="所属主单" style={{ marginBottom: 12 }}>
                <Input value={group.masterNo || '-'} disabled style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col className="col-5">
              <Form.Item
                label={`分单号 #${houseIdx + 1}`}
                style={{ marginBottom: isMissing || isDuplicate ? 24 : 12 }}
                validateStatus={isMissing || isDuplicate ? 'error' : undefined}
                help={isDuplicate ? '分单号重复' : isMissing ? '请填写分单号' : undefined}
              >
                <Input
                  value={extraHouse.houseNo}
                  onChange={(e) =>
                    onHouseFieldChange(groupIdx, houseIdx, 'houseNo', e.target.value)
                  }
                  placeholder={`请输入分单号 (如 ${documentLabels.house}-001)`}
                  maxLength={64}
                  disabled={disabled || isReleased}
                  status={isMissing || isDuplicate ? 'error' : undefined}
                  allowClear
                />
              </Form.Item>
            </Col>
            <Col className="col-5">
              <Form.Item label="分单签放方式" style={{ marginBottom: 12 }}>
                <AutoComplete
                  value={extraHouse.releaseType}
                  options={releaseTypeOptions}
                  onChange={(val) =>
                    onHouseFieldChange(groupIdx, houseIdx, 'releaseType', val)
                  }
                  placeholder="请选择或输入分单签放方式"
                  disabled={disabled || isReleased}
                  allowClear
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
            <Col className="col-5">
              <Form.Item label="分单备注" style={{ marginBottom: 12 }}>
                <Input
                  value={extraHouse.note}
                  onChange={(e) =>
                    onHouseFieldChange(groupIdx, houseIdx, 'note', e.target.value)
                  }
                  placeholder="分单备注说明 (选填)"
                  maxLength={500}
                  disabled={disabled || isReleased}
                  allowClear
                />
              </Form.Item>
            </Col>
            <Col className="col-5">
              <Form.Item label="操作" style={{ marginBottom: 12 }}>
                <Button
                  type="link"
                  danger
                  disabled={disabled || isReleased}
                  aria-label={`删除分单 ${extraHouse.houseNo || houseIdx + 1}`}
                  icon={<DeleteOutlined />}
                  onClick={() => onRemoveHouse(groupIdx, houseIdx)}
                >
                  删除分单
                </Button>
              </Form.Item>
            </Col>
          </React.Fragment>
        );
      })}
    </>
  );
}
