import {
  ProFormDateTimePicker,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Alert, Button, Form, Radio, Space, Typography } from 'antd';
import { useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { FormRow, ProFormSearchableSelect } from '@/components/ui';
import { history } from '@/router/history';
import {
  airExportServiceGetAirMasterWaybill,
  airExportServiceListAirMasterWaybills,
} from '@/services/roncin/airExportService';
import { masterDataServiceListAirlines } from '@/services/roncin/masterDataService';
import { ensureListResponse } from '@/utils/api';
import type { TemplateProps } from '../../types';
import { TemplateControlGroup, TemplateFieldGroup } from '../FieldGroup';
import { SeaCollapsibleNotesField } from '../sea/SeaCollapsibleNotesField';
import { AirWaybillFields } from './AirWaybillFields';

type AirMasterFieldProps = Pick<
  TemplateProps,
  'searchLocations' | 'locationOptions' | 'readonly'
>;

/** 主单身份与航线字段：主运单号、航空公司、起运/目的机场与航班号。 */
export function AirMasterRouteFields({
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
  const name = (key: string) => [...prefix, key];
  return (
    <FormRow cols={4}>
      {includeIdentity && (
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
          ]}
          fieldProps={{ maxLength: 20 }}
          disabled={props.readonly || Boolean(master)}
        />
      )}
      <ProFormSearchableSelect
        name={name('airlineId')}
        label="航空公司"
        rules={[{ required: true, message: '请选择航空公司' }]}
        disabled={props.readonly}
        options={
          master?.content?.airlineId
            ? [
                {
                  value: master.content.airlineId,
                  label: master.airlineName || '',
                },
              ]
            : undefined
        }
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
          return (response.data ?? []).map((item) => ({
            value: item.id || '',
            label: `${item.nameZh || item.nameEn || ''} (${item.iataCode || item.icaoCode || ''})`,
          }));
        }}
      />
      {(['originAirportId', 'destinationAirportId'] as const).map(
        (key, index) => (
          <ProFormSearchableSelect
            key={key}
            name={name(key)}
            label={index === 0 ? '起运机场' : '目的机场'}
            options={[
              ...props.locationOptions,
              ...(master?.content?.[key]
                ? [
                    {
                      value: master.content[key] || '',
                      label:
                        index === 0
                          ? master.originAirportName || ''
                          : master.destinationAirportName || '',
                    },
                  ]
                : []),
            ]}
            rules={[{ required: true, message: '请选择机场' }]}
            disabled={props.readonly}
            request={({ keyWords }) => props.searchLocations(keyWords)}
          />
        ),
      )}
      <ProFormText
        name={name('flightNo')}
        label="航班号"
        disabled={props.readonly}
        fieldProps={{ maxLength: 40 }}
      />
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
    <FormRow cols={4}>
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
  return (
    <>
      <FormRow cols={4}>
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
      </FormRow>
      <FormRow cols={3}>
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
    </>
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

/**
 * 「订舱与运输」卡片：主单与订舱 / 航线与班期 / 时刻与申报三个小节，
 * 主单备注与操作备注使用可折叠交互。主单关联方式、共享主单选择、
 * masterId 预选与同号提示逻辑原样保留在「主单与订舱」小节。
 */
export function AirMasterTransportSection({ props }: { props: TemplateProps }) {
  const form = Form.useFormInstance();
  const mode = Form.useWatch('airMasterMode', form) || 'NEW';
  const documentMode = Form.useWatch('airDocumentMode', form);
  const selectedIdentity = Form.useWatch('airMasterSelection', {
    form,
    preserve: true,
  }) as API.AirMasterWaybill | undefined;
  const selectedMasterId = Form.useWatch('airMasterId', {
    form,
    preserve: true,
  });
  const selectedMaster = Form.useWatch('airMaster', { form, preserve: true }) as
    | API.AirMasterWaybillInput
    | undefined;
  const air = Form.useWatch('airExport', { form, preserve: true }) as
    | API.AirOrderInfo
    | undefined;
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const organizationRef = useRef(organizationId);
  organizationRef.current = organizationId;
  const documentModeRef = useRef(documentMode);
  documentModeRef.current = documentMode;
  const [params] = useSearchParams();
  const requestedMode = params.get('documentMode');
  const requestedMasterId = props.isDetail ? undefined : params.get('masterId');
  const preselected = useQuery({
    queryKey: [
      'air-export',
      'preselected-master',
      requestedMasterId,
      { organizationId },
    ],
    enabled: Boolean(requestedMasterId && organizationId),
    queryFn: async ({ signal }) => {
      const response = await airExportServiceGetAirMasterWaybill(
        { id: requestedMasterId || '', page: 1, pageSize: 1 },
        { signal },
      );
      if (!response?.data) throw new Error('主单加载失败');
      return response.data;
    },
  });
  useEffect(() => {
    if (preselected.data && !props.isDetail)
      form.setFieldsValue({
        airMasterMode: 'EXISTING',
        airMasterId: preselected.data.id,
        airMasterExpectedVersion: preselected.data.version,
        airMaster: preselected.data.content,
        airMasterSelection: preselected.data,
        airDocumentMode:
          requestedMode === 'DIRECT' || requestedMode === 'HOUSE'
            ? requestedMode
            : preselected.data.documentMode || 'HOUSE',
      });
  }, [form, preselected.data, props.isDetail, requestedMode]);
  const masterEditable = !props.isDetail && mode === 'NEW';
  return (
    <div style={{ display: 'grid', gap: 12, width: '100%' }}>
      <TemplateFieldGroup title="主单与订舱">
        <TemplateControlGroup title="航空单证模式" required>
          <Form.Item
            name="airDocumentMode"
            rules={[{ required: true, message: '请选择航空单证模式' }]}
          >
            <Radio.Group
              aria-label="航空单证模式"
              disabled={props.readonly || props.isDetail}
              options={[
                { label: '主分单（HOUSE）', value: 'HOUSE' },
                { label: '直单（DIRECT，无 HAWB）', value: 'DIRECT' },
              ]}
              onChange={() => {
                if (mode === 'EXISTING')
                  form.setFieldsValue({
                    airMasterId: undefined,
                    airMasterExpectedVersion: undefined,
                    airMasterSelection: undefined,
                    airMaster: undefined,
                  });
              }}
            />
          </Form.Item>
        </TemplateControlGroup>
        <Form.Item name="airMaster" hidden>
          <span />
        </Form.Item>
        <Form.Item name="airExport" hidden>
          <span />
        </Form.Item>
        <Form.Item name="airMasterSelection" hidden>
          <span />
        </Form.Item>
        {props.isDetail && (
          <>
            <Form.Item name="airMasterId" hidden>
              <span />
            </Form.Item>
            <Form.Item name="airMasterExpectedVersion" hidden>
              <span />
            </Form.Item>
          </>
        )}
        {!props.isDetail && (
          <TemplateControlGroup title="主单关联方式">
            <Form.Item name="airMasterMode" style={{ marginBottom: 0 }}>
              <Radio.Group
                aria-label="主单关联方式"
                options={[
                  { label: '新建主单', value: 'NEW' },
                  { label: '明确关联已有主单', value: 'EXISTING' },
                ]}
              />
            </Form.Item>
          </TemplateControlGroup>
        )}
        {!props.isDetail && mode === 'EXISTING' && (
          <>
            <ProFormSearchableSelect
              name="airMasterId"
              label="选择共享主单"
              params={{ documentMode }}
              options={
                selectedMasterId
                  ? [
                      {
                        value: selectedMasterId,
                        label:
                          selectedMaster?.masterNo ||
                          preselected.data?.content?.masterNo ||
                          '',
                      },
                    ]
                  : undefined
              }
              rules={[
                { required: true, message: '请选择已有主单' },
                {
                  validator: async () => {
                    if (
                      selectedMasterId &&
                      (selectedIdentity?.canAttach !== true ||
                        (selectedIdentity.documentMode &&
                          selectedIdentity.documentMode !== documentMode))
                    ) {
                      throw new Error(
                        selectedIdentity?.attachBlockedReason ||
                          '所选主单不允许按当前单证模式建票，请重新选择',
                      );
                    }
                  },
                },
              ]}
              request={async ({ keyWords }) => {
                if (!organizationId) return [];
                const requestedDocumentMode = documentMode;
                const response = ensureListResponse(
                  await airExportServiceListAirMasterWaybills({
                    businessType: 'AE',
                    keyword: keyWords,
                    page: 1,
                    pageSize: 50,
                  }),
                  '主单候选加载失败',
                );
                if (
                  organizationRef.current !== organizationId ||
                  documentModeRef.current !== requestedDocumentMode
                )
                  return [];
                return (response.data ?? []).map((master) => ({
                  value: master.id || '',
                  label: `${master.content?.masterNo} · ${master.airlineName || ''} · ${master.documentMode || '未定模式'} · ${master.memberCount || 0} 票${master.attachBlockedReason ? ` · ${master.attachBlockedReason}` : ''}`,
                  disabled:
                    master.canAttach !== true ||
                    Boolean(
                      master.documentMode &&
                        master.documentMode !== requestedDocumentMode,
                    ),
                  master,
                }));
              }}
              fieldProps={{
                onChange: (_value, option) => {
                  const selected = Array.isArray(option) ? option[0] : option;
                  if (selected && 'master' in selected) {
                    const master = selected.master as API.AirMasterWaybill;
                    form.setFieldsValue({
                      airMasterExpectedVersion: master.version,
                      airMaster: master.content,
                      airMasterSelection: master,
                    });
                  } else
                    form.setFieldsValue({
                      airMasterExpectedVersion: undefined,
                      airMasterSelection: undefined,
                      airMaster: undefined,
                    });
                },
              }}
            />
            <Form.Item name="airMasterExpectedVersion" hidden>
              <input />
            </Form.Item>
            <Alert
              type="info"
              showIcon
              title="关联后共同运输资料以主单为准"
              description="同号主单不会自动合并；保存时再次核验你选择的主单版本。共同资料请在主单工作台维护。"
            />
            {selectedIdentity &&
              (selectedIdentity.canAttach !== true ||
                (selectedIdentity.documentMode &&
                  selectedIdentity.documentMode !== documentMode)) && (
                <Alert
                  type="error"
                  showIcon
                  title={
                    selectedIdentity.attachBlockedReason ||
                    '该主单与当前单证模式不匹配，请重新选择'
                  }
                />
              )}
          </>
        )}
        {props.isDetail && (
          <Alert
            type="info"
            showIcon
            title={`共享主单 ${air?.master?.content?.masterNo || '-'}`}
            description={
              <Space wrap>
                <Typography.Text>
                  {air?.master?.airlineName} · {air?.master?.originAirportName}{' '}
                  → {air?.master?.destinationAirportName} ·{' '}
                  {air?.master?.memberCount || 0} 票
                </Typography.Text>
                <a
                  onClick={() =>
                    history.push(
                      `/orders/air-export/masters/${air?.master?.id}`,
                    )
                  }
                >
                  查看主单与成员
                </a>
              </Space>
            }
          />
        )}
        <FormRow cols={4}>
          <ProFormText
            name="bookingNo"
            label="订舱号"
            disabled={props.readonly}
          />
        </FormRow>
        {masterEditable && (
          <>
            <AirMasterRouteFields props={props} />
            <AirMasterCandidateHint organizationId={organizationId} />
          </>
        )}
      </TemplateFieldGroup>
      {masterEditable && (
        <>
          <TemplateFieldGroup title="航线与班期">
            <AirMasterScheduleFields props={props} />
          </TemplateFieldGroup>
          <TemplateFieldGroup title="时刻与申报">
            <AirMasterDeclarationFields props={props} />
          </TemplateFieldGroup>
          <SeaCollapsibleNotesField
            name={['airMaster', 'notes']}
            label="主单备注"
            placeholder="请输入主单备注"
            disabled={props.readonly}
          />
          <TemplateFieldGroup title="主运单正文">
            <AirWaybillFields
              prefix={['airMaster', 'waybillContent']}
              disabled={props.readonly}
            />
          </TemplateFieldGroup>
        </>
      )}
      <SeaCollapsibleNotesField
        name="operationNotes"
        label="操作备注"
        placeholder="请输入操作备注"
        disabled={props.readonly}
        sectionHeading
      />
    </div>
  );
}

function AirMasterCandidateHint({
  organizationId,
}: {
  organizationId?: string;
}) {
  const form = Form.useFormInstance();
  const documentMode = Form.useWatch('airDocumentMode', form);
  const value = Form.useWatch(['airMaster', 'masterNo'], form);
  const masterNo = String(value || '').replace(/[\s-]/g, '');
  const query = useQuery({
    queryKey: [
      'air-export',
      'identity-candidates',
      { organizationId, masterNo },
    ],
    enabled: Boolean(organizationId && /^\d{11}$/.test(masterNo)),
    queryFn: async ({ signal }) => {
      const response = ensureListResponse(
        await airExportServiceListAirMasterWaybills(
          {
            businessType: 'AE',
            keyword: masterNo,
            page: 1,
            pageSize: 20,
          },
          { signal },
        ),
        '主单身份核对失败',
      );
      return (response.data || []).filter(
        (candidate) => candidate.content?.masterNo === masterNo,
      );
    },
  });
  const candidate =
    query.isFetching || query.isError ? undefined : query.data?.[0];
  return candidate ? (
    <Alert
      type="warning"
      showIcon
      title="同号主单已存在，请明确选择关联"
      description={`${candidate.content?.masterNo} · ${candidate.airlineName} · ${candidate.documentMode || '未定模式'} · ${candidate.memberCount || 0} 票${candidate.attachBlockedReason ? ` · ${candidate.attachBlockedReason}` : ''}`}
      action={
        <Button
          disabled={
            candidate.canAttach !== true ||
            Boolean(
              candidate.documentMode && candidate.documentMode !== documentMode,
            )
          }
          onClick={() =>
            form.setFieldsValue({
              airMasterMode: 'EXISTING',
              airMasterId: candidate.id,
              airMasterExpectedVersion: candidate.version,
              airMaster: candidate.content,
              airMasterSelection: candidate,
            })
          }
        >
          选择此主单
        </Button>
      }
    />
  ) : null;
}
