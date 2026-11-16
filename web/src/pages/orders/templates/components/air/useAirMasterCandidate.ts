import { useQuery } from '@tanstack/react-query';
import { Form } from 'antd';
import dayjs from 'dayjs';
import { useCallback, useEffect, useState } from 'react';
import { airExportServiceListAirMasterWaybills } from '@/services/roncin/airExportService';
import { formatDate, trimDecimal } from '@/utils/format';
import {
  airBusinessTypeCode,
  requireAirMasterIdentity,
} from '../../../order-kinds/air-export/identity';
import type { SelectOption } from '../../types';

/** 规范化主运单号：去空格与连字符。 */
export function normalizeAirMasterNo(value: unknown) {
  return String(value || '').replace(/[\s-]/g, '');
}

/** 11 位且满足模 7 校验位才视为有效 MAWB。 */
export function isValidAirMasterNo(masterNo: string) {
  return (
    /^\d{11}$/.test(masterNo) &&
    Number(masterNo.slice(3, 10)) % 7 === Number(masterNo[10])
  );
}

type FieldKind = 'text' | 'id' | 'time' | 'number';

/**
 * 主单共同资料中由本票主单字段承载、需要与已有主单逐项核对的字段。
 * 主单状态不参与差异：新建表单默认「资料待完善」，关联时直接以主单为准。
 */
const COMPARED_FIELDS: {
  key: keyof API.AirMasterWaybillInput;
  label: string;
  kind: FieldKind;
}[] = [
  { key: 'airlineId', label: '航空公司', kind: 'id' },
  { key: 'flightNo', label: '航班号', kind: 'text' },
  { key: 'originAirportId', label: '起运机场', kind: 'id' },
  { key: 'destinationAirportId', label: '目的机场', kind: 'id' },
  { key: 'etd', label: '计划起飞', kind: 'time' },
  { key: 'eta', label: '计划到达', kind: 'time' },
  { key: 'actualDepartureAt', label: '实际起飞', kind: 'time' },
  { key: 'actualArrivalAt', label: '实际到达', kind: 'time' },
  { key: 'documentCutoffAt', label: '截单时间', kind: 'time' },
  { key: 'customsCutoffAt', label: '报关截止', kind: 'time' },
  { key: 'declaredPackages', label: '主单申报件数', kind: 'number' },
  { key: 'declaredGrossWeightKg', label: '主单申报毛重', kind: 'number' },
  { key: 'declaredVolumeCbm', label: '主单申报体积', kind: 'number' },
  { key: 'notes', label: '主单备注', kind: 'text' },
];

function isEmptyValue(value: unknown) {
  return (
    value === undefined ||
    value === null ||
    (typeof value === 'string' && value.trim() === '')
  );
}

function sameValue(kind: FieldKind, current: unknown, shared: unknown) {
  if (isEmptyValue(current) && isEmptyValue(shared)) return true;
  if (isEmptyValue(current) || isEmptyValue(shared)) return false;
  if (kind === 'time')
    return dayjs(current as string).isSame(dayjs(shared as string), 'minute');
  if (kind === 'number') return Number(current) === Number(shared);
  return String(current).trim() === String(shared).trim();
}

/** 主单内容转表单值：时间字段转 Dayjs，供日期控件直接回显。 */
function toFormValue(kind: FieldKind, value: unknown) {
  if (kind === 'time')
    return isEmptyValue(value) ? undefined : dayjs(value as string);
  return isEmptyValue(value) ? undefined : value;
}

export type AirMasterDifference = {
  key: string;
  label: string;
  currentValue: string;
  sharedValue: string;
};

export type AirMasterCandidateStatus =
  | 'idle'
  | 'checking'
  | 'error'
  | 'none'
  | 'blocked'
  | 'modeMismatch'
  | 'conflict'
  | 'ready';

const documentModeLabel = (mode?: string) =>
  mode === 'DIRECT' ? '直单（DIRECT）' : '主分单（HOUSE）';

/**
 * 空运新建页同号主单自动判断（对齐海运 useSeaMasterBillCandidate）：
 * 输入有效 MAWB 后防抖精确核对；命中可关联主单时自动带入空字段并在无差异时
 * 自动确认关联，差异、模式不符、不可关联与核对中均在保存前由 validate 拦截。
 * 关联结论只写入内部字段 airMasterMode/airMasterId/airMasterExpectedVersion/
 * airMasterSelection，提交载荷仍由表单适配器决定。
 */
export function useAirMasterCandidate({
  enabled: hookEnabled,
  organizationId,
  businessType,
  locationOptions,
}: {
  enabled: boolean;
  organizationId?: string;
  businessType?: number;
  locationOptions: SelectOption[];
}) {
  const form = Form.useFormInstance();
  const businessTypeCode = airBusinessTypeCode(businessType);
  const rawMasterNo = Form.useWatch(['airMaster', 'masterNo'], {
    form,
    preserve: true,
  });
  const masterNo = normalizeAirMasterNo(rawMasterNo);
  const documentMode = Form.useWatch('airDocumentMode', {
    form,
    preserve: true,
  }) as string | undefined;
  const current = Form.useWatch('airMaster', { form, preserve: true }) as
    | API.AirMasterWaybillInput
    | undefined;
  const valid = isValidAirMasterNo(masterNo);
  const [debouncedMasterNo, setDebouncedMasterNo] = useState('');

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedMasterNo(masterNo), 300);
    return () => clearTimeout(timer);
  }, [masterNo]);

  const enabled = Boolean(
    hookEnabled &&
      valid &&
      organizationId &&
      businessTypeCode &&
      debouncedMasterNo === masterNo,
  );
  const query = useQuery({
    queryKey: [
      'air-orders',
      'master-candidate',
      { organizationId, businessType, masterNo: debouncedMasterNo },
    ],
    enabled,
    queryFn: async ({ signal }) => {
      const response = await airExportServiceListAirMasterWaybills(
        {
          businessType: businessTypeCode,
          keyword: debouncedMasterNo,
          page: 1,
          pageSize: 20,
        },
        { signal, skipErrorHandler: true },
      );
      if (!response || response.success === false)
        throw new Error('同号主单核对失败，请重试后再保存');
      // 列表为包含匹配；完整 11 位号按完全相等筛选即为精确核对。
      const matched = (response.data ?? [])
        .map((item) =>
          requireAirMasterIdentity(item, item.id, organizationId, businessType),
        )
        .find((item) => item.content?.masterNo === debouncedMasterNo);
      return matched ?? null;
    },
    meta: { silent: true },
  });

  const checking =
    hookEnabled && valid && (!enabled || query.isPending || query.isFetching);
  const candidate =
    enabled && query.isSuccess && !query.isFetching
      ? (query.data ?? undefined)
      : undefined;
  const error = enabled && query.isError ? query.error.message : undefined;
  const blocked = Boolean(candidate && candidate.canAttach !== true);
  const modeMismatch = Boolean(
    candidate &&
      !blocked &&
      candidate.documentMode &&
      documentMode &&
      candidate.documentMode !== documentMode,
  );
  const shared = candidate?.content;
  const differences: AirMasterDifference[] =
    candidate && !blocked && !modeMismatch
      ? COMPARED_FIELDS.filter(
          ({ key, kind }) =>
            !isEmptyValue(current?.[key]) &&
            !sameValue(kind, current?.[key], shared?.[key]),
        ).map(({ key, label, kind }) => ({
          key,
          label,
          currentValue: displayValue(kind, key, current?.[key], {
            locationOptions,
          }),
          sharedValue: displayValue(kind, key, shared?.[key], {
            locationOptions,
            candidate,
          }),
        }))
      : [];
  const differencesKey = JSON.stringify(differences.map((item) => item.key));

  let status: AirMasterCandidateStatus = 'idle';
  if (!hookEnabled || !valid) status = 'idle';
  else if (checking) status = 'checking';
  else if (error) status = 'error';
  else if (!candidate) status = 'none';
  else if (blocked) status = 'blocked';
  else if (modeMismatch) status = 'modeMismatch';
  else if (differences.length > 0) status = 'conflict';
  else status = 'ready';

  const candidateKey = candidate ? `${candidate.id}:${candidate.version}` : '';

  useEffect(() => {
    if (!hookEnabled) return;
    function setValue(name: string | string[], value: unknown) {
      if (JSON.stringify(form.getFieldValue(name)) !== JSON.stringify(value))
        form.setFieldValue(name, value);
    }
    // 命中可用主单：本票空字段自动带入主单值；覆盖非空差异必须由「采用主单资料」触发。
    if (candidate && !blocked && !modeMismatch) {
      for (const { key, kind } of COMPARED_FIELDS) {
        if (
          isEmptyValue(form.getFieldValue(['airMaster', key])) &&
          !isEmptyValue(shared?.[key])
        )
          form.setFieldValue(
            ['airMaster', key],
            toFormValue(kind, shared?.[key]),
          );
      }
    }
    if (status === 'ready' && candidate) {
      setValue(['airMaster', 'status'], shared?.status);
      setValue('airMasterMode', 'EXISTING');
      setValue('airMasterId', candidate.id);
      setValue('airMasterExpectedVersion', candidate.version);
      if (form.getFieldValue('airMasterSelection')?.id !== candidate.id)
        form.setFieldValue('airMasterSelection', candidate);
      return;
    }
    setValue('airMasterMode', 'NEW');
    setValue('airMasterId', undefined);
    setValue('airMasterExpectedVersion', undefined);
    if (form.getFieldValue('airMasterSelection') !== undefined)
      form.setFieldValue('airMasterSelection', undefined);
  }, [
    form,
    hookEnabled,
    status,
    candidateKey,
    blocked,
    modeMismatch,
    differencesKey,
  ]);

  const adopt = () => {
    if (!candidate || blocked || modeMismatch) return;
    form.setFieldsValue({
      airMaster: Object.fromEntries(
        COMPARED_FIELDS.map(({ key, kind }) => [
          key,
          toFormValue(kind, shared?.[key]),
        ]),
      ),
    });
  };
  const switchMode = () => {
    if (candidate?.documentMode)
      form.setFieldValue('airDocumentMode', candidate.documentMode);
  };

  const validate = useCallback(async () => {
    if (!hookEnabled) return;
    // 校验可先于 useWatch/Effect 渲染；直接读取当前号码，禁止旧结论穿过该窗口。
    const currentNo = normalizeAirMasterNo(
      form.getFieldValue(['airMaster', 'masterNo']),
    );
    if (!isValidAirMasterNo(currentNo)) return;
    if (currentNo !== masterNo || status === 'checking')
      throw new Error('正在核对已有主单，请稍候');
    if (status === 'error') throw new Error(error);
    if (status === 'blocked')
      throw new Error(
        candidate?.attachBlockedReason || '该主单不允许关联，请更换主单号',
      );
    if (status === 'modeMismatch')
      throw new Error(
        `该主单为${documentModeLabel(candidate?.documentMode)}，请切换单证模式后再保存`,
      );
    if (status === 'conflict')
      throw new Error('本票主单资料与已有主单不同，请确认采用主单资料');
  }, [form, hookEnabled, masterNo, status, error, candidate]);

  useEffect(() => {
    if (!hookEnabled) return;
    void form
      .validateFields([['airMaster', 'masterNo']], { dirty: true })
      .catch(() => undefined);
  }, [form, hookEnabled, validate]);

  return {
    status,
    candidate,
    differences,
    error,
    linked: status === 'ready',
    adopt,
    switchMode,
    validate,
    documentModeLabel,
  };
}

function displayValue(
  kind: FieldKind,
  key: keyof API.AirMasterWaybillInput,
  value: unknown,
  context: {
    locationOptions: SelectOption[];
    candidate?: API.AirMasterWaybill;
  },
) {
  if (isEmptyValue(value)) return '未填写';
  if (kind === 'time')
    return formatDate(dayjs(value as string).toDate(), 'minute');
  if (kind === 'number') return trimDecimal(String(value));
  if (kind === 'id') {
    if (context.candidate) {
      if (key === 'airlineId') return context.candidate.airlineName || '已选';
      if (key === 'originAirportId')
        return context.candidate.originAirportName || '已选';
      if (key === 'destinationAirportId')
        return context.candidate.destinationAirportName || '已选';
    }
    const option = context.locationOptions.find((item) => item.value === value);
    return option ? String(option.label) : '已选其他值';
  }
  return String(value);
}
