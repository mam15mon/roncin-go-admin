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

export default function AirMasterIdentityFields({
  props,
}: {
  props: TemplateProps;
}) {
  const form = Form.useFormInstance();
  const mode = Form.useWatch('airMasterMode', form) || 'NEW';
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
  const [params] = useSearchParams();
  const requestedMasterId = props.isDetail ? undefined : params.get('masterId');
  const preselected = useQuery({
    queryKey: ['air-export', 'master', requestedMasterId, { organizationId }],
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
      });
  }, [form, preselected.data, props.isDetail]);
  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      <Form.Item name="airMaster" hidden>
        <span />
      </Form.Item>
      <Form.Item name="airExport" hidden>
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
        <Form.Item name="airMasterMode" label="主单关联方式">
          <Radio.Group
            options={[
              { label: '新建主单', value: 'NEW' },
              { label: '明确关联已有主单', value: 'EXISTING' },
            ]}
          />
        </Form.Item>
      )}
      {!props.isDetail && mode === 'EXISTING' && (
        <>
          <ProFormSearchableSelect
            name="airMasterId"
            label="选择共享主单"
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
            rules={[{ required: true, message: '请选择已有主单' }]}
            request={async ({ keyWords }) => {
              const response = ensureListResponse(
                await airExportServiceListAirMasterWaybills({
                  keyword: keyWords,
                  page: 1,
                  pageSize: 50,
                }),
                '主单候选加载失败',
              );
              if (organizationRef.current !== organizationId) return [];
              return (response.data ?? []).map((master) => ({
                value: master.id || '',
                label: `${master.content?.masterNo} · ${master.airlineName || ''} · ${master.memberCount || 0} 票 · v${master.version}`,
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
                  });
                } else
                  form.setFieldValue('airMasterExpectedVersion', undefined);
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
        </>
      )}
      {props.isDetail ? (
        <Alert
          type="info"
          showIcon
          title={`共享主单 ${air?.master?.content?.masterNo || '-'}`}
          description={
            <Space wrap>
              <Typography.Text>
                {air?.master?.airlineName} · {air?.master?.originAirportName} →{' '}
                {air?.master?.destinationAirportName} ·{' '}
                {air?.master?.memberCount || 0} 票
              </Typography.Text>
              <a
                onClick={() =>
                  history.push(`/orders/air-export/masters/${air?.master?.id}`)
                }
              >
                查看主单与成员
              </a>
            </Space>
          }
        />
      ) : (
        mode === 'NEW' && (
          <>
            <AirMasterFields props={props} />
            <AirMasterCandidateHint organizationId={organizationId} />
          </>
        )
      )}
      <FormRow cols={3}>
        <ProFormText
          name="airHouseNo"
          label="本票分运单号（HAWB）"
          rules={[
            { required: true, whitespace: true, message: '请输入本票分运单号' },
          ]}
          disabled={props.readonly}
          fieldProps={{ maxLength: 64 }}
          tooltip="同一主单内不允许重号，不同主单可重复；号码会统一大写。"
        />
        <ProFormSearchableSelect
          name="airHouseStatus"
          label="本票分单状态"
          initialValue="DRAFT"
          options={[
            { label: '资料待完善', value: 'DRAFT' },
            { label: '已确认', value: 'CONFIRMED' },
            { label: '已签发', value: 'RELEASED' },
          ]}
          disabled={props.readonly}
        />
        <ProFormTextArea
          name="airHouseContent"
          label="本票单证资料"
          disabled={props.readonly}
          fieldProps={{ maxLength: 4000, autoSize: { minRows: 1, maxRows: 4 } }}
        />
      </FormRow>
      {props.isDetail && air?.houseStatus === 'RELEASED' && (
        <ProFormTextArea
          name="airHouseChangeReason"
          label="已签发分单修改原因"
          tooltip="修改已签发分单号、内容或状态时必填；仅保存本票客户、货量等字段时可空。"
          fieldProps={{ maxLength: 500 }}
          disabled={props.readonly}
        />
      )}
    </Space>
  );
}

function AirMasterCandidateHint({
  organizationId,
}: {
  organizationId?: string;
}) {
  const form = Form.useFormInstance();
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
          { keyword: masterNo, page: 1, pageSize: 20 },
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
      description={`${candidate.content?.masterNo} · ${candidate.airlineName} · ${candidate.memberCount || 0} 票 · v${candidate.version}`}
      action={
        <Button
          onClick={() =>
            form.setFieldsValue({
              airMasterMode: 'EXISTING',
              airMasterId: candidate.id,
              airMasterExpectedVersion: candidate.version,
              airMaster: candidate.content,
            })
          }
        >
          选择此主单
        </Button>
      }
    />
  ) : null;
}
