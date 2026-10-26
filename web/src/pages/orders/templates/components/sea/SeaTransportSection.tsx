import { CheckCircleFilled, LoadingOutlined } from '@ant-design/icons';
import {
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormText,
} from '@ant-design/pro-components';
import {
  Alert,
  Button,
  Card,
  Form,
  type FormInstance,
  Input,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  theme,
} from 'antd';
import dayjs from 'dayjs';
import isoWeek from 'dayjs/plugin/isoWeek';
import React from 'react';
import { FormRow, ProFormSearchableSelect } from '@/components/ui';
import { SeaDocumentStructure } from '@/enums.generated';
import { formatDate } from '@/utils/format';
import { containerOwnershipOptions } from '../../../common';
import {
  OrderContainerRequestFields,
  type SelectOption,
} from '../../../order-plan-fields';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';
import type { TemplateProps } from '../../types';
import {
  useSeaMasterBillCandidate,
  useSharedSeaVoyage,
} from './useSeaMasterBillCandidate';

export { splitSeaVesselVoyage } from './sea-shared-voyage';

dayjs.extend(isoWeek);

export function SeaMasterBillFields({
  disabled = false,
  isDetail = false,
  locationOptions = [],
}: {
  disabled?: boolean;
  isDetail?: boolean;
  locationOptions?: SelectOption[];
}) {
  const { token } = theme.useToken();
  const state = useSeaMasterBillCandidate(isDetail, locationOptions);
  const { candidate, selected, existing } = state;
  const isMultiMemberLocked = isDetail && (existing?.memberCount ?? 0) > 1;
  const memberLinks =
    candidate?.members?.filter((member) => member.orderId) ?? [];
  const incomplete =
    selected &&
    (!selected.vesselName ||
      !selected.voyageNo ||
      !selected.originLocationId ||
      !selected.dischargeLocationId ||
      !selected.etd ||
      !selected.eta);

  return (
    <>
      <div>
        <Form.Item
          name="seaMasterBillMasterNo"
          label="MBL 主单号"
          help={
            state.matching
              ? false
              : state.waitingForCarrier
                ? '待选择船公司后核对主单占用情况'
                : undefined
          }
          validateStatus={state.matching ? 'validating' : undefined}
          tooltip={
            isMultiMemberLocked
              ? '该主单已关联多票订单，禁止从单票页面修改主单号；如需变更请走共享主单改派流程'
              : undefined
          }
          rules={[
            { required: true, message: '请输入 MBL 主单号' },
            {
              pattern: /^[A-Za-z0-9]+$/,
              message: '主单号仅允许包含英文字母和数字，禁止包含空格或特殊字符',
            },
            { validator: state.validate },
          ]}
          normalize={(value) =>
            typeof value === 'string' ? value.toUpperCase() : value
          }
        >
          <Input
            placeholder="请输入主单号"
            disabled={disabled || isMultiMemberLocked || undefined}
            style={{ textTransform: 'uppercase' }}
            suffix={
              state.matching ? (
                <LoadingOutlined
                  aria-label="MBL 主单号核对中"
                  style={{ color: token.colorPrimary }}
                />
              ) : state.checked &&
                !state.error &&
                !state.batchDirectBlocked &&
                !state.requiresHouse &&
                !state.isSingleMemberCorrection &&
                (!candidate || state.ready) ? (
                <CheckCircleFilled
                  aria-label="MBL 主单号核对通过"
                  style={{ color: token.colorSuccess }}
                />
              ) : (
                <span />
              )
            }
          />
        </Form.Item>
      </div>
      {state.isSingleMemberCorrection && (
        <div style={{ gridColumn: 'span 2' }}>
          <ProFormText
            name="seaMasterBillCorrectionReason"
            label="主单更正原因"
            placeholder="请输入主单号/船公司更正原因"
            rules={[
              {
                required: true,
                message: '单票修改主单号或船公司必须填写更正原因',
              },
            ]}
          />
        </div>
      )}
      <SeaAssociatedHouseBillsField />
      {isMultiMemberLocked && existing && (
        <div style={{ gridColumn: '1 / -1' }}>
          <Alert
            type="info"
            showIcon
            title={`共用主单 ${existing.masterNo}，共 ${existing.memberCount} 票`}
            description="船名航次、起运港、卸货港、中转港和船期由共同航程维护。如需修改，请使用顶部「更多操作 → 修改共享航程」；可用操作取决于当前权限与订单状态。"
          />
        </div>
      )}
      {candidate && (
        <div style={{ gridColumn: '1 / -1', marginBottom: 16 }}>
          <Card size="small">
            <Space orientation="vertical" style={{ width: '100%' }}>
              {state.batchDirectBlocked ? (
                <Alert
                  type="error"
                  showIcon
                  title="该主单已被直单订单占用，如需拼单请先将其转为分单"
                  description={`主单：${candidate.masterNo}，已有 ${candidate.memberCount ?? 0} 票订单（含直单订单）`}
                />
              ) : (
                <div>
                  <Typography.Text
                    strong
                  >{`已找到主单 ${candidate.masterNo}，已有 ${candidate.memberCount ?? 0} 票订单`}</Typography.Text>
                  {!isDetail && (
                    <div>
                      本票保存后将加入该主单，共{' '}
                      {(candidate.memberCount ?? 0) + 1} 票。
                    </div>
                  )}
                  <Typography.Text type="secondary">
                    船公司：{candidate.shippingLineName || '未填写'}
                  </Typography.Text>
                </div>
              )}
              {state.requiresHouse && (
                <Alert
                  type="warning"
                  showIcon
                  title="共享主单仅支持有货代分单（HOUSE），请选择分单模式"
                />
              )}
              {(candidate.transportExecutions?.length ?? 0) > 1 && (
                <Form.Item label="本票共用航程" style={{ marginBottom: 0 }}>
                  <Select
                    aria-label="本票共用航程"
                    placeholder="该主单存在多个航程，请核实后选择"
                    value={selected?.id}
                    disabled={state.batchDirectBlocked || isDetail}
                    options={candidate.transportExecutions?.map((voyage) => ({
                      value: voyage.id,
                      label: `${voyage.vesselName || '船名未填写'} / ${voyage.voyageNo || '航次未填写'} / ${voyage.etd ? formatDate(voyage.etd, 'date') : 'ETD未填写'}`,
                    }))}
                    onChange={state.select}
                  />
                </Form.Item>
              )}
              {selected && (
                <div style={{ fontSize: 13 }}>
                  <span>共用航程：</span>
                  <span>
                    船名航次：{selected.vesselName || '未填写'} /{' '}
                    {selected.voyageNo || '未填写'}；
                  </span>
                  <span>
                    起运港：
                    {selected.originLocationName ||
                      (selected.originLocationId ? '名称读取失败' : '未填写')}
                    ；
                  </span>
                  <span>
                    卸货港：
                    {selected.dischargeLocationName ||
                      (selected.dischargeLocationId
                        ? '名称读取失败'
                        : '未填写')}
                    ；
                  </span>
                  <span>
                    ETD：
                    {selected.etd ? formatDate(selected.etd, 'date') : '未填写'}
                    ；
                  </span>
                  <span>
                    ETA：
                    {selected.eta ? formatDate(selected.eta, 'date') : '未填写'}
                  </span>
                </div>
              )}
              {state.differences.length > 0 &&
                !state.batchDirectBlocked &&
                !isDetail && (
                  <Alert
                    type="warning"
                    showIcon
                    title="本票已填航程与主单不同，请确认后采用"
                    description={
                      <>
                        <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
                          {state.differences.map((difference) => (
                            <li key={difference.field}>
                              {difference.label}：本票 {difference.currentValue}
                              ；主单 {difference.sharedValue}
                            </li>
                          ))}
                        </ul>
                        <Button
                          htmlType="button"
                          disabled={state.requiresHouse}
                          onClick={state.adopt}
                        >
                          采用主单航程
                        </Button>
                      </>
                    }
                  />
                )}
              {state.ready && (
                <Typography.Text type="secondary">
                  已带入主单共享航程，航程字段只读；客户、分单、货物、最终目的地与费用仍由本票独立维护。
                </Typography.Text>
              )}
              {incomplete && !state.batchDirectBlocked && (
                <Alert
                  type="info"
                  showIcon
                  title="已有主单的航程资料尚未填全"
                  description="如需补全，请在已有订单的「更多操作 → 修改共享航程」中统一维护；新票填写不会补写已有共享航程。"
                />
              )}
              {!state.batchDirectBlocked && memberLinks.length > 0 && (
                <Space wrap>
                  <Typography.Text type="secondary">
                    查看已有订单：
                  </Typography.Text>
                  {memberLinks.map((member) => (
                    <Typography.Link
                      key={member.orderId}
                      href={`/orders/sea-export/${member.orderId}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {member.orderNo || '查看订单'}
                    </Typography.Link>
                  ))}
                </Space>
              )}
              {!selected && candidate.transportExecutions?.length === 0 && (
                <Alert
                  type="warning"
                  showIcon
                  title="该主单暂无可关联航程，请先在已有订单中核实"
                />
              )}
            </Space>
          </Card>
        </div>
      )}
      {state.error && (
        <div style={{ gridColumn: '1 / -1', marginBottom: 16 }}>
          <Alert
            type="error"
            showIcon
            title="主单候选查询失败"
            description={state.error}
            action={
              <Button htmlType="button" onClick={() => void state.retry()}>
                重新核对
              </Button>
            }
          />
        </div>
      )}
    </>
  );
}

export function SeaAssociatedHouseBillsField() {
  const form = Form.useFormInstance();
  const watchedHouseBill = (Form.useWatch('seaHouseBill', form) ??
    form?.getFieldValue('seaHouseBill')) as { houseNo?: string } | undefined;
  const watchedStructure = (Form.useWatch('seaDocumentStructure', form) ??
    form?.getFieldValue('seaDocumentStructure')) as number | undefined;
  const watchedDocSummary = (Form.useWatch('seaDocumentSummary', form) ??
    form?.getFieldValue('seaDocumentSummary')) as
    | API.SeaOrderDocumentSummary
    | undefined;

  const houseNo =
    watchedHouseBill?.houseNo?.trim() || watchedDocSummary?.houseNo?.trim();

  const structure = watchedStructure ?? watchedDocSummary?.documentStructure;
  const isDirect =
    structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;

  return (
    <Form.Item label="关联分单号">
      <div
        data-testid="associated-hbl-display"
        style={{
          minHeight: 32,
          padding: '4px 11px',
          backgroundColor: '#fafafa',
          border: '1px solid #d9d9d9',
          borderRadius: 6,
          display: 'flex',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: '4px',
          fontSize: 13,
          lineHeight: 1.5,
        }}
      >
        {isDirect ? (
          <Tag color="success" style={{ margin: 0 }}>
            直单，无HBL
          </Tag>
        ) : houseNo ? (
          <Tag color="processing" style={{ margin: 0 }}>
            {houseNo}
          </Tag>
        ) : (
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            暂未录入分单号
          </Typography.Text>
        )}
      </div>
    </Form.Item>
  );
}

function ClearContainerRequests({
  onChange,
}: {
  onChange?: (value: API.OrderContainerRequestInput[]) => void;
}) {
  return (
    <Button
      danger
      size="small"
      htmlType="button"
      onClick={() => onChange?.([])}
    >
      清空箱量计划
    </Button>
  );
}

export function SeaContainerPlanFields({
  options,
}: {
  options: SelectOption[];
}) {
  const shipmentType = Form.useWatch('shipmentType');
  const containerRequests = (Form.useWatch('containerRequests') ??
    []) as API.OrderContainerRequestInput[];
  const policy = resolveSeaOrderFormPolicy({ shipmentType });

  if (!policy.showContainerPlan) {
    return (
      <div style={{ width: '100%', marginBottom: 12 }}>
        <Alert
          type={containerRequests.length > 0 ? 'warning' : 'info'}
          showIcon
          title={
            <span style={{ fontSize: 13 }}>
              <span>散杂货不使用箱型箱量、箱号或封号配置</span>
              <span style={{ color: '#8c8c8c', marginLeft: 8, fontSize: 12 }}>
                {containerRequests.length > 0
                  ? '已录入箱量计划，请确认清空'
                  : '页面已隐藏集装箱专属配置，货物按件数、毛重、体积和计费吨管理'}
              </span>
            </span>
          }
          action={
            containerRequests.length > 0 ? (
              <Form.Item name="containerRequests" noStyle>
                <ClearContainerRequests />
              </Form.Item>
            ) : undefined
          }
          style={{ padding: '4px 12px' }}
        />
      </div>
    );
  }
  return (
    <div style={{ width: '100%' }}>
      <OrderContainerRequestFields options={options} />
    </div>
  );
}

export function SeaScheduleDateFields() {
  const form = Form.useFormInstance();
  const etd = Form.useWatch('etd', form);
  const { readonly } = useSharedSeaVoyage();
  const weekValue =
    etd && dayjs(etd).isValid() ? `W${dayjs(etd).isoWeek()}` : '';

  return (
    <>
      <ProFormDatePicker
        name="etd"
        label="ETD"
        disabled={readonly || undefined}
        rules={[{ required: true, message: '请选择 ETD（预计离港时间）' }]}
        fieldProps={{
          style: { width: '100%' },
          onChange: (date: dayjs.Dayjs | null) => {
            if (date) {
              const currentEta = form?.getFieldValue('eta');
              if (currentEta?.isBefore(date, 'day')) {
                form?.setFieldValue('eta', undefined);
              }
            }
          },
        }}
      />
      <Form.Item label="WEEK">
        <Input
          value={weekValue}
          placeholder="依据 ETD 自动生成"
          disabled
          style={{ width: '100%' }}
        />
      </Form.Item>
      <ProFormDatePicker
        name="eta"
        label="ETA"
        disabled={readonly || undefined}
        dependencies={['etd']}
        rules={[
          { required: true, message: '请选择 ETA（预计到港时间）' },
          ({ getFieldValue }: FormInstance) => ({
            validator(_: unknown, value?: dayjs.Dayjs) {
              const etdVal = getFieldValue('etd');
              if (!value || !etdVal || !value.isBefore(etdVal, 'day')) {
                return Promise.resolve();
              }
              return Promise.reject(new Error('ETA 不能早于 ETD'));
            },
          }),
        ]}
        fieldProps={{ style: { width: '100%' } }}
      />
    </>
  );
}

function SeaVesselVoyageField() {
  const { readonly } = useSharedSeaVoyage();
  return (
    <div>
      <ProFormText
        name="vesselVoyage"
        label="船名航次"
        placeholder="请输入船名航次"
        disabled={readonly || undefined}
        fieldProps={{
          suffix: (
            <Tooltip title="船期与船舶实时动态追踪">
              <a
                href="https://www.shipxy.com/"
                target="_blank"
                rel="noopener noreferrer"
                tabIndex={-1}
                style={{ fontSize: 12, color: '#1677ff' }}
                onClick={(event) => event.stopPropagation()}
              >
                船在哪儿
              </a>
            </Tooltip>
          ),
        }}
      />
    </div>
  );
}

function SeaPortFields({
  locationOptions,
  searchLocations,
}: Pick<TemplateProps, 'locationOptions' | 'searchLocations'>) {
  const { readonly, voyage } = useSharedSeaVoyage();
  const options = [...locationOptions];
  for (const [value, label, fieldLabel] of [
    [voyage?.originLocationId, voyage?.originLocationName, '起运港'],
    [voyage?.dischargeLocationId, voyage?.dischargeLocationName, '卸货港'],
    [voyage?.transitLocationId, voyage?.transitLocationName, '中转港'],
  ]) {
    if (value && !options.some((option) => option.value === value))
      options.push({ value, label: label || `${fieldLabel}名称读取失败` });
  }
  return (
    <FormRow cols={4}>
      <div>
        <ProFormSearchableSelect
          name="originLocationId"
          label="起运港"
          options={options}
          disabled={readonly || undefined}
          request={
            readonly
              ? undefined
              : async ({ keyWords }) => searchLocations(keyWords)
          }
          fieldProps={{ filterOption: false }}
          placeholder="请选择起运港或地点"
        />
      </div>
      <div>
        <ProFormSearchableSelect
          name="destinationLocationId"
          label="目的港"
          options={locationOptions}
          request={async ({ keyWords }) => searchLocations(keyWords)}
          fieldProps={{ filterOption: false }}
          placeholder="请选择目的港或地点"
        />
      </div>
      <ProFormSearchableSelect
        name="dischargeLocationId"
        label="卸货港"
        options={options}
        disabled={readonly || undefined}
        request={
          readonly
            ? undefined
            : async ({ keyWords }) => searchLocations(keyWords)
        }
        fieldProps={{ filterOption: false }}
        placeholder="请选择卸货港或地点"
      />
      <ProFormSearchableSelect
        name="transitLocationId"
        label="中转港"
        options={options}
        disabled={readonly || undefined}
        request={
          readonly
            ? undefined
            : async ({ keyWords }) => searchLocations(keyWords)
        }
        fieldProps={{ filterOption: false }}
        placeholder="请选择中转港或地点"
      />
    </FormRow>
  );
}

export function getSeaTransportFields(props: TemplateProps) {
  const { locationOptions, searchLocations, containerSpecOptions, isDetail } =
    props;
  return {
    containers: <SeaContainerPlanFields options={containerSpecOptions} />,
    master: (
      <SeaMasterBillFields
        isDetail={isDetail}
        locationOptions={locationOptions}
      />
    ),
    ownership: (
      <ProFormSearchableSelect
        name="containerOwnership"
        label="货主箱标记"
        options={containerOwnershipOptions}
        placeholder="请选择 COC / SOC"
      />
    ),
    vessel: <SeaVesselVoyageField />,
    ports: (
      <SeaPortFields
        locationOptions={locationOptions}
        searchLocations={searchLocations}
      />
    ),
    schedule: <SeaScheduleDateFields />,
    cutoffs: (
      <>
        <ProFormDateTimePicker
          name="siCutoff"
          label="SI截关时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDateTimePicker
          name="docCutoff"
          label="单证截关时间"
          tooltip="即截单时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDateTimePicker
          name="customsCutoff"
          label="报关截关时间"
          tooltip="即截关时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDateTimePicker
          name="vgmCutoff"
          label="VGM截关时间"
          fieldProps={{ style: { width: '100%' } }}
        />
      </>
    ),
  };
}
