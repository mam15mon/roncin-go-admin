import {
  ProFormDateTimePicker,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Radio,
  Space,
  Tag,
  Typography,
} from 'antd';
import { useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { FormCell, FormRow, ProFormSearchableSelect } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import { history } from '@/router/history';
import { airExportServiceGetAirMasterWaybill } from '@/services/roncin/airExportService';
import { masterDataServiceListAirlines } from '@/services/roncin/masterDataService';
import { ensureListResponse } from '@/utils/api';
import { formatDate, trimDecimal } from '@/utils/format';
import {
  airBusinessTypeCode,
  requireAirMasterIdentity,
} from '../../../order-kinds/air-export/identity';
import type { TemplateProps } from '../../types';
import { TemplateFieldGroup } from '../FieldGroup';
import { SeaCollapsibleNotesField } from '../sea/SeaCollapsibleNotesField';
import { AirNotesFields, getAirCommercialFields } from './AirCommercialFields';
import { useAirMasterCandidate } from './useAirMasterCandidate';

type AirMasterFieldProps = Pick<
  TemplateProps,
  'searchLocations' | 'locationOptions' | 'readonly'
>;

/** 无关键字查询时把主单已选值并入候选，保证回显名称而不是 ID。 */
function withSeedOption<T extends { value?: unknown; label?: unknown }>(
  options: T[],
  value?: string,
  label?: string,
  keyWords?: string,
): (T | { value: string; label: string })[] {
  if (
    !value ||
    !label ||
    keyWords ||
    options.some((item) => item.value === value)
  )
    return options;
  return [{ value, label }, ...options];
}

/** 主单身份与航线字段：主运单号、航空公司、起运/目的机场与航班号。 */
export function AirMasterRouteFields({
  props,
  prefix = ['airMaster'],
  includeIdentity = true,
  master,
  masterNoEditable = false,
  masterNoValidator,
}: {
  props: AirMasterFieldProps;
  prefix?: string[];
  includeIdentity?: boolean;
  master?: API.AirMasterWaybill;
  /** 新建页关联主单后仍允许改号以解除关联；维护场景主单号随主单锁定。 */
  masterNoEditable?: boolean;
  /** 新建页同号主单核对结论的保存前校验。 */
  masterNoValidator?: () => Promise<void>;
}) {
  const name = (key: string) => [...prefix, key];
  return (
    <FormRow cols={6} style={{ rowGap: 4 }}>
      {includeIdentity && (
        <FormCell span={2}>
          <ProFormText
            name={name('masterNo')}
            label="主运单号（MAWB）"
            rules={[
              { required: true, message: '请输入主运单号' },
              {
                validator: async (_, value) => {
                  const normalized = String(value || '').replace(/[\s-]/g, '');
                  if (
                    normalized &&
                    (!/^\d{11}$/.test(normalized) ||
                      Number(normalized.slice(3, 10)) % 7 !==
                        Number(normalized[10]))
                  )
                    throw new Error(
                      '请输入 11 位有效航空主单号（含模 7 校验位）',
                    );
                },
              },
              ...(masterNoValidator
                ? [{ validator: () => masterNoValidator() }]
                : []),
            ]}
            fieldProps={{ maxLength: 20 }}
            disabled={
              masterNoEditable ? false : props.readonly || Boolean(master)
            }
          />
        </FormCell>
      )}
      <FormCell span={2}>
        <ProFormSearchableSelect
          name={name('airlineId')}
          label="航空公司"
          rules={[{ required: true, message: '请选择航空公司' }]}
          disabled={props.readonly}
          // 设置 request 后 ProFormSelect 不读静态 options：主单已选值须并入请求结果才能显示名称。
          params={{ seedMasterId: master?.id }}
          fieldProps={{ filterOption: false }}
          request={async ({ keyWords }) => {
            const response = ensureListResponse(
              await masterDataServiceListAirlines({
                page: 1,
                pageSize: 50,
                keyword: keyWords,
                enabled: true,
              }),
              '航空公司加载失败',
            );
            return withSeedOption(
              (response.data ?? []).map((item) => ({
                value: item.id || '',
                label: `${[item.nameZh, item.nameEn].filter(Boolean).join(' / ')} (${item.iataCode || item.icaoCode || ''})`,
              })),
              master?.content?.airlineId,
              master?.airlineName,
              keyWords,
            );
          }}
        />
      </FormCell>
      <FormCell span={2}>
        <ProFormText
          name={name('flightNo')}
          label="航班号"
          disabled={props.readonly}
          fieldProps={{ maxLength: 40 }}
        />
      </FormCell>
      {(['originAirportId', 'destinationAirportId'] as const).map(
        (key, index) => (
          <FormCell key={key} span={3}>
            <ProFormSearchableSelect
              name={name(key)}
              label={index === 0 ? '起运机场' : '目的机场'}
              options={props.locationOptions}
              rules={[{ required: true, message: '请选择机场' }]}
              disabled={props.readonly}
              params={{ seedMasterId: master?.id }}
              fieldProps={{ filterOption: false }}
              request={async ({ keyWords }) =>
                withSeedOption(
                  await props.searchLocations(keyWords),
                  master?.content?.[key],
                  index === 0
                    ? master?.originAirportName
                    : master?.destinationAirportName,
                  keyWords,
                )
              }
            />
          </FormCell>
        ),
      )}
    </FormRow>
  );
}

/** 班期字段：计划起飞/到达与实际起飞/到达。 */
export function AirMasterScheduleFields({
  props,
  prefix = ['airMaster'],
}: {
  props: AirMasterFieldProps;
  prefix?: string[];
}) {
  const name = (key: string) => [...prefix, key];
  return (
    <FormRow cols={6} style={{ rowGap: 4 }}>
      <ProFormDateTimePicker
        name={name('etd')}
        label="计划起飞"
        disabled={props.readonly}
        fieldProps={{ style: { width: '100%' } }}
      />
      <ProFormDateTimePicker
        name={name('eta')}
        label="计划到达"
        disabled={props.readonly}
        fieldProps={{ style: { width: '100%' } }}
      />
      <ProFormDateTimePicker
        name={name('actualDepartureAt')}
        label="实际起飞"
        disabled={props.readonly}
        fieldProps={{ style: { width: '100%' } }}
      />
      <ProFormDateTimePicker
        name={name('actualArrivalAt')}
        label="实际到达"
        disabled={props.readonly}
        fieldProps={{ style: { width: '100%' } }}
      />
    </FormRow>
  );
}

/** 申报字段：截单/报关截止、主单单证状态与主单申报件重尺。 */
export function AirMasterDeclarationFields({
  props,
  prefix = ['airMaster'],
}: {
  props: AirMasterFieldProps;
  prefix?: string[];
}) {
  const name = (key: string) => [...prefix, key];
  // 截单、报关截止、单证状态与申报件重尺恰好六格，同一网格一行排满。
  return (
    <FormRow cols={6} style={{ rowGap: 4 }}>
      <ProFormDateTimePicker
        name={name('documentCutoffAt')}
        label="截单时间"
        disabled={props.readonly}
        fieldProps={{ style: { width: '100%' } }}
      />
      <ProFormDateTimePicker
        name={name('customsCutoffAt')}
        label="报关截止"
        disabled={props.readonly}
        fieldProps={{ style: { width: '100%' } }}
      />
      <ProFormSearchableSelect
        name={name('status')}
        label="主单单证状态"
        initialValue="DRAFT"
        options={[
          { label: '资料待完善', value: 'DRAFT' },
          { label: '已确认', value: 'CONFIRMED' },
          { label: '已签发', value: 'RELEASED' },
        ]}
        disabled={props.readonly}
      />
      <ProFormDigit
        name={name('declaredPackages')}
        label="主单申报件数"
        min={0}
        disabled={props.readonly}
        fieldProps={{ precision: 0 }}
      />
      <ProFormDigit
        name={name('declaredGrossWeightKg')}
        label="主单申报毛重（KG）"
        min={0}
        disabled={props.readonly}
        fieldProps={{ precision: 3 }}
      />
      <ProFormDigit
        name={name('declaredVolumeCbm')}
        label="主单申报体积（m³）"
        min={0}
        disabled={props.readonly}
        fieldProps={{ precision: 3 }}
      />
    </FormRow>
  );
}

/** 共同资料只有新建主单或主单明确编辑命令能够写入。 */
export function AirMasterFields({
  props,
  prefix = ['airMaster'],
  includeIdentity = true,
  master,
}: {
  props: AirMasterFieldProps;
  prefix?: string[];
  includeIdentity?: boolean;
  master?: API.AirMasterWaybill;
}) {
  return (
    <>
      <AirMasterRouteFields
        props={props}
        prefix={prefix}
        includeIdentity={includeIdentity}
        master={master}
      />
      <AirMasterScheduleFields props={props} prefix={prefix} />
      <AirMasterDeclarationFields props={props} prefix={prefix} />
      <ProFormTextArea
        name={[...prefix, 'notes']}
        label="主单备注"
        disabled={props.readonly}
        fieldProps={{ maxLength: 1000 }}
      />
    </>
  );
}

const AIR_DOCUMENT_MODE_OUTPUT_ID = 'airDocumentModeDisplay';

/** 详情只读展示单证模式，与同行输入框等高对齐（同海运提单模式）。 */
function AirDocumentModeValue({ direct }: { direct: boolean }) {
  return (
    <output
      id={AIR_DOCUMENT_MODE_OUTPUT_ID}
      aria-label="当前航空单证模式"
      style={{ display: 'flex', alignItems: 'center', height: 32 }}
    >
      <Tag color="blue" style={{ margin: 0, paddingInline: 10 }}>
        {direct ? '直单（DIRECT，无 HAWB）' : '主分单（HOUSE）'}
      </Tag>
    </output>
  );
}

/**
 * 「订舱与运输」卡片：与海运同节奏，主单与订舱 / 航线与班期 / 时刻与申报 /
 * 备注四个小节。新建页只有一个主运单号入口：输入后由 useAirMasterCandidate
 * 自动核对同号主单并就地给出结论（新建 / 加入 / 不可用），不再要求先选
 * 「新建/关联」；详情页展示共享主单摘要与运输维护入口。
 */
export function AirMasterTransportSection({ props }: { props: TemplateProps }) {
  const form = Form.useFormInstance();
  const documentMode = Form.useWatch('airDocumentMode', form);
  const air = Form.useWatch('airExport', { form, preserve: true }) as
    | API.AirOrderInfo
    | undefined;
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const businessType = props.businessType;
  const businessTypeCode = airBusinessTypeCode(businessType);
  const identity = `${organizationId}:${businessType}`;
  const preselectedApplied = useRef<string | undefined>(undefined);
  const [params] = useSearchParams();
  const requestedMode = params.get('documentMode');
  const requestedMasterId = props.isDetail ? undefined : params.get('masterId');
  const isCreate = !props.isDetail;
  const candidateState = useAirMasterCandidate({
    enabled: isCreate,
    organizationId,
    businessType,
    locationOptions: props.locationOptions,
  });
  const preselected = useQuery({
    queryKey: [
      'air-orders',
      'preselected-master',
      requestedMasterId,
      { organizationId, businessType },
    ],
    enabled: Boolean(requestedMasterId && organizationId && businessTypeCode),
    queryFn: async ({ signal }) => {
      const response = await airExportServiceGetAirMasterWaybill(
        { id: requestedMasterId || '', page: 1, pageSize: 1 },
        { signal },
      );
      if (response?.success === false) throw new Error('主单加载失败');
      return requireAirMasterIdentity(
        response?.data,
        requestedMasterId ?? undefined,
        organizationId,
        businessType,
      );
    },
  });
  // 主单工作台带 masterId 建票：只填入主单号与单证模式，由同号核对自然完成关联。
  useEffect(() => {
    const selectionKey = `${identity}:${requestedMasterId}:${requestedMode}`;
    if (
      preselected.data &&
      isCreate &&
      preselectedApplied.current !== selectionKey
    ) {
      preselectedApplied.current = selectionKey;
      form.setFieldsValue({
        airMaster: { masterNo: preselected.data.content?.masterNo },
        airDocumentMode:
          requestedMode === 'DIRECT' || requestedMode === 'HOUSE'
            ? requestedMode
            : preselected.data.documentMode || 'HOUSE',
      });
    }
  }, [
    form,
    preselected.data,
    isCreate,
    requestedMode,
    identity,
    requestedMasterId,
  ]);
  const master = props.isDetail
    ? (props.airTransportSnapshot ?? air?.master)
    : undefined;
  const isImport = businessType === OrderBusinessType.BUSINESS_TYPE_AI;
  const commercial = getAirCommercialFields(props);
  // 已关联主单时共同资料以主单为准：主单字段只读，主运单号仍可修改以解除关联。
  const sharedProps = {
    ...props,
    readonly: props.readonly || candidateState.linked,
  };
  return (
    <div style={{ display: 'grid', gap: 12, width: '100%' }}>
      {preselected.isError && (
        <Alert
          type="error"
          showIcon
          title="所选主单加载失败，请重新输入主运单号。"
        />
      )}
      <TemplateFieldGroup title={isImport ? '承运与主单' : '主单与订舱'}>
        <Form.Item name="airMaster" hidden>
          <span />
        </Form.Item>
        <Form.Item name="airExport" hidden>
          <span />
        </Form.Item>
        <Form.Item name="airMasterSelection" hidden>
          <span />
        </Form.Item>
        <Form.Item name="airMasterMode" hidden>
          <span />
        </Form.Item>
        <Form.Item name="airMasterId" hidden>
          <span />
        </Form.Item>
        <Form.Item name="airMasterExpectedVersion" hidden>
          <span />
        </Form.Item>
        {/* 与海运「提单模式」同规格：单证模式入首行字段格，订舱号与代理同行。 */}
        <FormRow cols={6} style={{ rowGap: 4 }}>
          <FormCell span={2}>
            <Form.Item
              name="airDocumentMode"
              label="航空单证模式"
              htmlFor={AIR_DOCUMENT_MODE_OUTPUT_ID}
              rules={[{ required: true, message: '请选择航空单证模式' }]}
            >
              {props.isDetail ? (
                <AirDocumentModeValue
                  direct={
                    documentMode === 'DIRECT' || air?.documentMode === 'DIRECT'
                  }
                />
              ) : (
                <Radio.Group
                  id={AIR_DOCUMENT_MODE_OUTPUT_ID}
                  aria-label="航空单证模式"
                  disabled={props.readonly}
                  options={[
                    { label: '主分单（HOUSE）', value: 'HOUSE' },
                    { label: '直单（DIRECT，无 HAWB）', value: 'DIRECT' },
                  ]}
                />
              )}
            </Form.Item>
          </FormCell>
          <FormCell span={2}>
            <ProFormText
              name="bookingNo"
              label="订舱号"
              disabled={props.readonly}
            />
          </FormCell>
          {commercial.agents}
        </FormRow>
        {props.isDetail && master?.id && (
          <Space wrap>
            <Typography.Text>
              共享主单 {master.content?.masterNo || '-'}
            </Typography.Text>
            <Button
              type="link"
              disabled={false}
              onClick={() =>
                history.push(`/orders/${props.orderKind}/masters/${master.id}`)
              }
            >
              查看主单与成员
            </Button>
          </Space>
        )}
        {isCreate && (
          <>
            <AirMasterRouteFields
              props={sharedProps}
              masterNoEditable={!props.readonly}
              masterNoValidator={candidateState.validate}
              master={
                candidateState.linked ? candidateState.candidate : undefined
              }
            />
            <AirMasterCandidatePanel
              state={candidateState}
              documentMode={documentMode}
            />
          </>
        )}
      </TemplateFieldGroup>
      {isCreate && (
        <>
          <TemplateFieldGroup title="航线与班期">
            <AirMasterScheduleFields props={sharedProps} />
          </TemplateFieldGroup>
          <TemplateFieldGroup title="时刻与申报">
            <AirMasterDeclarationFields props={sharedProps} />
          </TemplateFieldGroup>
          <SeaCollapsibleNotesField
            name={['airMaster', 'notes']}
            label="主单备注"
            placeholder="请输入主单备注"
            disabled={sharedProps.readonly}
          />
        </>
      )}
      {props.isDetail && master && (
        <TemplateFieldGroup
          title="共同航线与班期"
          extra={props.airTransportMaintenance}
        >
          <AirMasterTransportSummary master={master} />
        </TemplateFieldGroup>
      )}
      <TemplateFieldGroup title="备注">
        <AirNotesFields disabled={props.readonly} />
      </TemplateFieldGroup>
    </div>
  );
}

/** 就地核对结论卡片：与海运「已找到主单」卡同一信息层级。 */
function AirMasterCandidatePanel({
  state,
  documentMode,
}: {
  state: ReturnType<typeof useAirMasterCandidate>;
  documentMode?: string;
}) {
  const { status, candidate, differences } = state;
  if (status === 'idle') return null;
  if (status === 'checking')
    return (
      <Typography.Text type="secondary" data-testid="air-master-candidate">
        正在核对同号主单…
      </Typography.Text>
    );
  if (status === 'error')
    return (
      <Alert
        data-testid="air-master-candidate"
        type="error"
        showIcon
        title={state.error || '同号主单核对失败，请重试后再保存'}
      />
    );
  if (status === 'none')
    return (
      <Alert
        data-testid="air-master-candidate"
        type="info"
        showIcon
        title="未找到同号主单，保存后将新建主单"
      />
    );
  if (!candidate) return null;
  const memberCount = candidate.memberCount ?? 0;
  const heading = `已找到主单 ${candidate.content?.masterNo || ''} · ${candidate.airlineName || '航空公司未填写'} · 已有 ${memberCount} 票`;
  if (status === 'blocked')
    return (
      <Alert
        data-testid="air-master-candidate"
        type="error"
        showIcon
        title={candidate.attachBlockedReason || '该主单不允许关联'}
        description={`${heading}。请核对主运单号，或更换为其他主单号。`}
      />
    );
  return (
    <Card size="small" data-testid="air-master-candidate">
      <Space orientation="vertical" style={{ width: '100%' }}>
        <div>
          <Typography.Text strong>{heading}</Typography.Text>
          <div>本票保存后将加入该主单，共 {memberCount + 1} 票。</div>
          <Typography.Text type="secondary">
            {[
              candidate.content?.flightNo &&
                `航班 ${candidate.content.flightNo}`,
              [candidate.originAirportName, candidate.destinationAirportName]
                .filter(Boolean)
                .join(' → '),
              candidate.content?.etd &&
                `计划起飞 ${formatDate(candidate.content.etd, 'minute')}`,
            ]
              .filter(Boolean)
              .join('；') || '主单共同资料未填写'}
          </Typography.Text>
        </div>
        {status === 'modeMismatch' && (
          <Alert
            type="warning"
            showIcon
            title={`该主单为${state.documentModeLabel(candidate.documentMode)}，本票当前为${state.documentModeLabel(documentMode)}`}
            action={
              <Button htmlType="button" onClick={state.switchMode}>
                切换为主单模式
              </Button>
            }
          />
        )}
        {status === 'conflict' && (
          <Alert
            type="warning"
            showIcon
            title="本票已填主单资料与已有主单不同，请确认后采用"
            description={
              <>
                <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
                  {differences.map((difference) => (
                    <li key={difference.key}>
                      {difference.label}：本票 {difference.currentValue}；主单{' '}
                      {difference.sharedValue}
                    </li>
                  ))}
                </ul>
                <Button htmlType="button" onClick={state.adopt}>
                  采用主单资料
                </Button>
              </>
            }
          />
        )}
        {status === 'ready' && (
          <Typography.Text type="secondary">
            已关联该主单，共同运输资料以主单为准（只读）；如需修改请到主单工作台维护。
            客户、分单、货物与费用仍由本票独立维护。
          </Typography.Text>
        )}
      </Space>
    </Card>
  );
}

/** 展示权威主单资料，不注册可随本票整单保存的副本字段。 */
export function AirMasterTransportSummary({
  master,
}: {
  master: API.AirMasterWaybill;
}) {
  const content = master.content;
  return (
    <Descriptions
      size="small"
      layout="vertical"
      column={{ xs: 1, md: 3, xl: 6 }}
      items={[
        {
          key: 'masterNo',
          label: '主运单号（MAWB）',
          children: content?.masterNo || '-',
        },
        {
          key: 'airline',
          label: '航空公司',
          children: master.airlineName || '-',
        },
        { key: 'flight', label: '航班号', children: content?.flightNo || '-' },
        {
          key: 'origin',
          label: '起运机场',
          children: master.originAirportName || '-',
        },
        {
          key: 'destination',
          label: '目的机场',
          children: master.destinationAirportName || '-',
        },
        {
          key: 'status',
          label: '主单单证状态',
          children:
            content?.status === 'RELEASED'
              ? '已签发'
              : content?.status === 'CONFIRMED'
                ? '已确认'
                : content?.status === 'DRAFT'
                  ? '资料待完善'
                  : '-',
        },
        ...(
          [
            ['etd', '计划起飞'],
            ['eta', '计划到达'],
            ['actualDepartureAt', '实际起飞'],
            ['actualArrivalAt', '实际到达'],
            ['documentCutoffAt', '截单时间'],
            ['customsCutoffAt', '报关截止'],
          ] as const
        ).map(([key, label]) => ({
          key,
          label,
          children: formatDate(content?.[key], 'minute'),
        })),
        {
          key: 'packages',
          label: '主单申报件数',
          children: trimDecimal(content?.declaredPackages),
        },
        {
          key: 'gross',
          label: '主单申报毛重（KG）',
          children: trimDecimal(content?.declaredGrossWeightKg),
        },
        {
          key: 'volume',
          label: '主单申报体积（m³）',
          children: trimDecimal(content?.declaredVolumeCbm),
        },
      ]}
    />
  );
}
