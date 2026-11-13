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
import { useAccess } from '@/app/access';
import { FormCell, FormRow, ProFormSearchableSelect } from '@/components/ui';
import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
import { history } from '@/router/history';
import { formatDate } from '@/utils/format';
import { containerOwnershipOptions } from '../../../common';
import {
  OrderContainerRequestFields,
  type SelectOption,
} from '../../../order-plan-fields';
import { buildSeaHouseOrderCreatePath } from '../../../sea-house-order-context';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';
import type { TemplateProps } from '../../types';
import {
  HouseBillNumberField,
  isTerminalHouseBill,
} from './HouseBillIdentityFields';
import { SeaFinalDestinationField } from './SeaFinalDestinationField';
import { seaVoyageMissingFields } from './sea-shared-voyage';
import {
  useSeaMasterBillCandidate,
  useSharedSeaVoyage,
} from './useSeaMasterBillCandidate';

dayjs.extend(isoWeek);

export function SeaMasterBillFields({
  disabled = false,
  isDetail = false,
  locationOptions = [],
  seaHouseOrderCreateSource,
  seaHouseOrderSourceOrderId,
  organizationId,
}: {
  disabled?: boolean;
  isDetail?: boolean;
  locationOptions?: SelectOption[];
} & Pick<
  TemplateProps,
  'seaHouseOrderCreateSource' | 'seaHouseOrderSourceOrderId' | 'organizationId'
>) {
  const { token } = theme.useToken();
  const state = useSeaMasterBillCandidate(
    isDetail,
    locationOptions,
    seaHouseOrderCreateSource,
  );
  const form = Form.useFormInstance();
  const structure = Form.useWatch('seaDocumentStructure', {
    form,
    preserve: true,
  });
  const { candidate, selected, existing } = state;
  const masterNo = Form.useWatch('seaMasterBillMasterNo', {
    form,
    preserve: true,
  });
  const shippingLineId = Form.useWatch('shippingLineId', {
    form,
    preserve: true,
  });
  const canCreateHouse = Boolean(
    isDetail &&
      seaHouseOrderSourceOrderId &&
      structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE &&
      existing?.masterBillId &&
      existing.status !== 'VOIDED' &&
      existing.masterNo === masterNo &&
      existing.shippingLineId === shippingLineId &&
      existing.masterNo &&
      existing.shippingLineId &&
      existing.transportExecutionId,
  );
  const isMultiMemberLocked = isDetail && (existing?.memberCount ?? 0) > 1;
  const memberLinks =
    candidate?.members?.filter((member) => member.orderId) ?? [];
  const access = useAccess();
  const missing = selected ? seaVoyageMissingFields(selected).join('、') : '';
  const missingRequiredDates = Boolean(
    selected && (!selected.etd || !selected.eta),
  );
  const canMaintain =
    access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'reassign') &&
    access.canOperateOrganization(organizationId);
  const canReadMembers = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'read',
  );

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
      <SeaAssociatedHouseBillsField isDetail={isDetail} disabled={disabled} />
      {canCreateHouse && seaHouseOrderSourceOrderId && (
        <div style={{ gridColumn: '1 / -1', marginBottom: 12 }}>
          <SeaCreateHouseOrderButton
            sourceOrderId={seaHouseOrderSourceOrderId}
            organizationId={organizationId}
          />
        </div>
      )}
      {isMultiMemberLocked && existing && (
        <div style={{ gridColumn: '1 / -1' }}>
          <Alert
            type="info"
            showIcon
            title={`共用主单 ${existing.masterNo}，共 ${existing.memberCount} 票`}
            description={
              <Space orientation="vertical" size={4}>
                <span>
                  船名、航次、起运港、卸货港、中转港和船期由这些订单共同使用；最终目的港由本票独立维护。
                </span>
              </Space>
            }
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
              {((candidate.transportExecutions?.length ?? 0) > 1 ||
                (state.sourceError &&
                  (candidate.transportExecutions?.length ?? 0) > 0)) && (
                <Form.Item label="本票共用航程" style={{ marginBottom: 0 }}>
                  <Select
                    aria-label="本票共用航程"
                    placeholder="该主单存在多个航程，请核实后选择"
                    value={selected?.id}
                    disabled={state.batchDirectBlocked || isDetail || undefined}
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
                    船名：{selected.vesselName || '未填写'}；航次：
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
                          disabled={state.requiresHouse || undefined}
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
              {missing && !state.batchDirectBlocked && (
                <Alert
                  type={missingRequiredDates ? 'warning' : 'info'}
                  showIcon
                  title={`共同航程尚缺：${missing}`}
                  description={
                    <Space orientation="vertical" size={4}>
                      <span>
                        {missingRequiredDates
                          ? 'ETD、ETA 为开单必填，须先补齐共同航程后才能保存新票。'
                          : '以上为资料提示，不单独阻止保存；ETD、ETA 仍为必填。'}
                      </span>
                      <span>新票不会修改已有共同航程。</span>
                      {canMaintain &&
                      canReadMembers &&
                      memberLinks.length > 0 ? (
                        <Typography.Link
                          href={`/orders/sea-export/${memberLinks[0].orderId}`}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          前往已有订单维护共同航程
                        </Typography.Link>
                      ) : (
                        <Typography.Text type="secondary">
                          {!canMaintain
                            ? '当前账号无共同航程维护权限，请联系有权限的人员补齐。'
                            : !canReadMembers
                              ? '当前账号无已有订单查看权限，请联系有权限的人员补齐。'
                              : '暂无可查看的已有订单，请联系负责该主单的人员补齐。'}
                        </Typography.Text>
                      )}
                    </Space>
                  }
                />
              )}
              {!state.batchDirectBlocked &&
                canReadMembers &&
                memberLinks.length > 0 && (
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
            title={
              state.sourceError ? '来源主单或航程不可用' : '主单候选查询失败'
            }
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

function SeaCreateHouseOrderButton({
  sourceOrderId,
  organizationId,
}: {
  sourceOrderId: string;
  organizationId?: string;
}) {
  const access = useAccess();
  if (
    !access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'create') ||
    !access.canOperateOrganization(organizationId)
  )
    return null;
  return (
    <Button
      htmlType="button"
      // 此入口仅在已有订单详情展示，创建另一票不修改当前锁定订单。
      disabled={false}
      onClick={() => history.push(buildSeaHouseOrderCreatePath(sourceOrderId))}
    >
      从此主单新增分单
    </Button>
  );
}

export function SeaAssociatedHouseBillsField({
  isDetail = true,
  disabled = false,
}: {
  isDetail?: boolean;
  disabled?: boolean;
}) {
  const displayId = React.useId();
  const form = Form.useFormInstance();
  const watchedHouseBill = (Form.useWatch('seaHouseBill', {
    form,
    preserve: true,
  }) ?? form?.getFieldValue('seaHouseBill')) as
    | {
        id?: string;
        houseNo?: string;
        expectedVersion?: string;
        status?: number;
      }
    | undefined;
  const watchedStructure = (Form.useWatch('seaDocumentStructure', {
    form,
    preserve: true,
  }) ?? form?.getFieldValue('seaDocumentStructure')) as number | undefined;
  const watchedDocSummary = (Form.useWatch('seaDocumentSummary', {
    form,
    preserve: true,
  }) ?? form?.getFieldValue('seaDocumentSummary')) as
    | API.SeaOrderDocumentSummary
    | undefined;

  const houseNo =
    watchedHouseBill?.houseNo?.trim() || watchedDocSummary?.houseNo?.trim();

  const structure = watchedStructure ?? watchedDocSummary?.documentStructure;
  const isDirect =
    structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;

  if (structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE) {
    const documentReady = Boolean(
      watchedHouseBill?.id &&
        watchedHouseBill.expectedVersion &&
        watchedHouseBill.status !== undefined,
    );
    return (
      <HouseBillNumberField
        fieldKey="seaHouseBill"
        showCheckFeedback={!isDetail}
        disabled={
          disabled ||
          (isDetail &&
            (!documentReady || isTerminalHouseBill(watchedHouseBill?.status)))
        }
        extra={
          isDetail ? '修改后点击页面「保存」，与订单资料一起提交。' : undefined
        }
      />
    );
  }
  return (
    <Form.Item label="关联分单号" htmlFor={displayId}>
      <output
        id={displayId}
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
      </output>
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
        tooltip="WEEK 依据 ETD 自动计算，显示在日期框右侧"
        disabled={readonly || undefined}
        rules={[{ required: true, message: '请选择 ETD（预计离港时间）' }]}
        fieldProps={{
          style: { width: '100%' },
          // WEEK 是 ETD 的派生值，以后缀展示，不再单独占一列。
          suffixIcon: weekValue ? (
            <span
              data-testid="sea-etd-week"
              title={`WEEK ${weekValue}`}
              style={{ fontSize: 12, fontWeight: 500, color: '#5b6b82' }}
            >
              {weekValue}
            </span>
          ) : undefined,
        }}
      />
      <ProFormDatePicker
        name="eta"
        label="ETA"
        tooltip="预计到达共同航程卸货港的时间，不代表本票最终目的港的到达时间"
        disabled={readonly || undefined}
        dependencies={['etd']}
        rules={[
          { required: true, message: '请选择 ETA（预计到达卸货港时间）' },
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

function SeaVesselFields() {
  const { readonly } = useSharedSeaVoyage();
  const form = Form.useFormInstance();
  const vesselValue = Form.useWatch('vesselName', {
    form,
    preserve: true,
  }) as string | undefined;
  const { token } = theme.useToken();
  const vesselName = vesselValue?.trim();
  return (
    <>
      <ProFormText
        name="vesselName"
        label="船名"
        placeholder="请输入船名"
        disabled={readonly || undefined}
        // 长船名与查询入口需要两倍宽度；FormRow 窄屏规则会恢复单列。
        formItemProps={{ style: { gridColumn: 'span 2', minWidth: 0 } }}
        fieldProps={{
          suffix: (
            <Tooltip
              title={
                vesselName
                  ? '在船讯网查询当前船名'
                  : '打开船讯网首页，手动查询船舶'
              }
            >
              <a
                href={
                  vesselName
                    ? `https://www.shipxy.com/?kw=${encodeURIComponent(vesselName)}`
                    : 'https://www.shipxy.com/'
                }
                target="_blank"
                rel="noopener noreferrer"
                style={{ fontSize: 12, color: token.colorLink }}
                onClick={(event) => event.stopPropagation()}
              >
                船在哪儿
              </a>
            </Tooltip>
          ),
        }}
      />
      <ProFormText
        name="voyageNo"
        label="航次"
        placeholder="请输入航次"
        disabled={readonly || undefined}
      />
    </>
  );
}

function SeaPortFields({
  locationOptions,
  searchLocations,
}: Pick<TemplateProps, 'locationOptions' | 'searchLocations'>) {
  const { readonly, voyage } = useSharedSeaVoyage();
  const form = Form.useFormInstance();
  const options = [...locationOptions];
  for (const [value, label, fieldLabel] of [
    [voyage?.originLocationId, voyage?.originLocationName, '起运港'],
    [voyage?.dischargeLocationId, voyage?.dischargeLocationName, '卸货港'],
    [voyage?.transitLocationId, voyage?.transitLocationName, '中转港'],
  ]) {
    if (value && !options.some((option) => option.value === value))
      options.push({ value, label: label || `${fieldLabel}名称读取失败` });
  }
  // 直接落入外层 6 列网格：中转港多为空，排在船期行尾；起运港、卸货港名称较长，各跨两列。
  return (
    <>
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
        placeholder="请选择中转港"
      />
      <FormCell span={2}>
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
      </FormCell>
      <FormCell span={2}>
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
          fieldProps={{
            filterOption: false,
            onChange: (value, option) => {
              const selected = Array.isArray(option) ? option[0] : option;
              form.setFieldValue(
                'seaDischargePortOption',
                value && selected?.label
                  ? { value, label: String(selected.label) }
                  : undefined,
              );
            },
          }}
          placeholder="请选择卸货港或地点"
        />
      </FormCell>
    </>
  );
}

export function getSeaTransportFields(props: TemplateProps) {
  const { locationOptions, searchLocations, containerSpecOptions, isDetail } =
    props;
  return {
    containers: (
      <FormRow cols={2}>
        <div>
          <Typography.Text strong>计划箱型箱量</Typography.Text>
          <SeaContainerPlanFields options={containerSpecOptions} />
        </div>
        <div>
          <Typography.Text strong>实际装箱</Typography.Text>
          {props.isDetail ? (
            (props.actualContainers ?? (
              <Typography.Paragraph type="secondary">
                实际箱资料暂不可用
              </Typography.Paragraph>
            ))
          ) : (
            <Typography.Paragraph type="secondary">
              保存订单后，可在此维护实际箱型、箱号、封号及件重尺。计划箱量与实际装箱分别维护。
            </Typography.Paragraph>
          )}
        </div>
      </FormRow>
    ),
    master: (
      <SeaMasterBillFields
        isDetail={isDetail}
        disabled={props.readonly}
        seaHouseOrderCreateSource={props.seaHouseOrderCreateSource}
        seaHouseOrderSourceOrderId={props.seaHouseOrderSourceOrderId}
        organizationId={props.organizationId}
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
    vessel: <SeaVesselFields />,
    ports: (
      <SeaPortFields
        locationOptions={locationOptions}
        searchLocations={searchLocations}
      />
    ),
    destination: (
      <SeaFinalDestinationField
        locationOptions={locationOptions}
        searchLocations={searchLocations}
        isDetail={isDetail}
        readonly={props.readonly}
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
