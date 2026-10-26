import { useQuery } from '@tanstack/react-query';
import { Form } from 'antd';
import dayjs from 'dayjs';
import { useCallback, useEffect, useState } from 'react';
import { SeaDocumentStructure } from '@/enums.generated';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import type { SelectOption } from '../../types';
import {
  isEmptySeaVoyageValue,
  type SeaSharedVoyageSource,
  type SeaVoyageValues,
  seaMasterBillIdentity,
  seaVoyageDifferences,
  seaVoyageFields,
  seaVoyageFormValues,
  splitSeaVesselVoyage,
} from './sea-shared-voyage';

export function useSharedSeaVoyage() {
  const form = Form.useFormInstance();
  const source = Form.useWatch('seaMasterBillSharedVoyage', {
    form,
    preserve: true,
  }) as SeaSharedVoyageSource | undefined;
  const masterNo = Form.useWatch('seaMasterBillMasterNo', {
    form,
    preserve: true,
  });
  const shippingLineId = Form.useWatch('shippingLineId', {
    form,
    preserve: true,
  });
  const existing = Form.useWatch('seaMasterBill', { form, preserve: true }) as
    | API.SeaMasterBillSummary
    | undefined;
  const identity = seaMasterBillIdentity(masterNo, shippingLineId);
  const inherited = source?.identity === identity ? source.voyage : undefined;
  const existingShared =
    (existing?.memberCount ?? 0) > 1 &&
    existing?.masterNo === masterNo &&
    existing?.shippingLineId === shippingLineId;
  return {
    readonly: Boolean(inherited || existingShared),
    voyage: inherited || (existingShared ? existing : undefined),
    existingShared,
  };
}

export function useSeaMasterBillCandidate(
  isDetail: boolean,
  locationOptions: SelectOption[],
) {
  const form = Form.useFormInstance();
  const masterNo = Form.useWatch('seaMasterBillMasterNo', {
    form,
    preserve: true,
  }) as string | undefined;
  const shippingLineId = Form.useWatch('shippingLineId', {
    form,
    preserve: true,
  }) as string | undefined;
  const structure = Form.useWatch('seaDocumentStructure', {
    form,
    preserve: true,
  });
  const existing = Form.useWatch('seaMasterBill', { form, preserve: true }) as
    | API.SeaMasterBillSummary
    | undefined;
  const values = Form.useWatch(
    (store) => ({
      originLocationId: store.originLocationId,
      dischargeLocationId: store.dischargeLocationId,
      transitLocationId: store.transitLocationId,
      vesselVoyage: store.vesselVoyage,
      etd: store.etd,
      eta: store.eta,
    }),
    { form, preserve: true },
  ) as SeaVoyageValues | undefined;
  const candidateId = Form.useWatch('seaMasterBillCandidateId', {
    form,
    preserve: true,
  });
  const identity = seaMasterBillIdentity(masterNo, shippingLineId);
  const unchanged = Boolean(
    isDetail &&
      existing?.masterNo === masterNo &&
      existing?.shippingLineId === shippingLineId,
  );
  const isSingleMemberCorrection = Boolean(
    isDetail && existing && (existing.memberCount ?? 0) <= 1 && !unchanged,
  );
  const [debouncedIdentity, setDebouncedIdentity] = useState('');
  const [selection, setSelection] = useState<{
    identity: string;
    id: string;
  }>();

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedIdentity(identity), 300);
    return () => clearTimeout(timer);
  }, [identity]);

  useEffect(() => setSelection(undefined), [identity]);

  const enabled = Boolean(
    identity && identity === debouncedIdentity && !unchanged,
  );
  const query = useQuery({
    queryKey: ['orders', 'sea-master-bill-candidate', debouncedIdentity],
    enabled,
    queryFn: async ({ signal, queryKey }) => {
      const queryIdentity = queryKey[2];
      const separator = queryIdentity.indexOf(':');
      const queryShippingLineId = queryIdentity.slice(0, separator);
      const queryMasterNo = queryIdentity.slice(separator + 1);
      // 候选身份只有船公司和主单号。回填航程不重新查询，差异在当前选定航次上即时展示。
      const current = form.getFieldsValue(true) as SeaVoyageValues;
      const { vesselName, voyageNo } = splitSeaVesselVoyage(
        current.vesselVoyage,
      );
      const response = await orderServiceMatchSeaMasterBillCandidate(
        {
          masterNo: queryMasterNo,
          shippingLineId: queryShippingLineId,
          originLocationId: current.originLocationId || undefined,
          dischargeLocationId: current.dischargeLocationId || undefined,
          transitLocationId: current.transitLocationId || undefined,
          vesselName,
          voyageNo,
          etd: current.etd
            ? dayjs(current.etd).startOf('day').toISOString()
            : undefined,
          eta: current.eta
            ? dayjs(current.eta).startOf('day').toISOString()
            : undefined,
        },
        { skipErrorHandler: true, signal },
      );
      if (!response || response.success === false)
        throw new Error('主单候选查询失败，请重试后再保存');
      return response;
    },
    meta: { silent: true },
  });
  const matching = Boolean(
    identity && !unchanged && (!enabled || query.isPending || query.isFetching),
  );
  const checked = enabled && query.isSuccess && !query.isFetching;
  const error = enabled && query.isError ? query.error.message : undefined;
  const candidate =
    checked && query.data.matched ? query.data.candidate : undefined;
  const executions = candidate?.transportExecutions ?? [];
  const selected =
    executions.length === 1
      ? executions[0]
      : selection?.identity === identity
        ? executions.find((item) => item.id === selection.id)
        : undefined;
  const batchDirectBlocked = (candidate?.members ?? []).some(
    (member) =>
      member.documentStructure ===
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
  );
  const requiresHouse = Boolean(
    candidate &&
      structure !== SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
  );
  const differences = selected
    ? seaVoyageDifferences(values ?? {}, selected, locationOptions)
    : [];
  const differencesKey = JSON.stringify(differences);
  const candidateReady = Boolean(
    candidate?.id &&
      candidate.version &&
      selected?.id &&
      selected.version &&
      !batchDirectBlocked &&
      !requiresHouse &&
      differences.length === 0 &&
      !isDetail,
  );
  const selectedKey = selected ? JSON.stringify(selected) : '';

  useEffect(() => {
    function setValue(name: string, value: unknown) {
      if (JSON.stringify(form.getFieldValue(name)) !== JSON.stringify(value))
        form.setFieldValue(name, value);
    }
    setValue('seaMasterBillBatchHouseNos', candidate?.batchNormalizedHouseNos);
    setValue(
      'seaMasterBillCandidateId',
      candidateReady ? candidate?.id : undefined,
    );
    setValue(
      'seaMasterBillExpectedCandidateVersion',
      candidateReady
        ? candidate?.version
        : isDetail
          ? existing?.version
          : undefined,
    );
    setValue(
      'seaMasterBillCandidateTeId',
      candidateReady ? selected?.id : undefined,
    );
    setValue(
      'seaMasterBillExpectedCandidateTeVersion',
      candidateReady ? selected?.version : undefined,
    );
    setValue(
      'seaMasterBillSharedVoyage',
      candidateReady && selected ? { identity, voyage: selected } : undefined,
    );
    if (!selected || batchDirectBlocked || requiresHouse || isDetail) return;
    const target = seaVoyageFormValues(selected);
    // 自动带入只填空值；覆盖任何非空差异都必须由采用按钮明确触发。
    for (const field of seaVoyageFields) {
      if (
        isEmptySeaVoyageValue(form.getFieldValue(field)) &&
        !isEmptySeaVoyageValue(target[field])
      )
        setValue(field, target[field]);
    }
  }, [
    form,
    identity,
    candidate,
    candidateReady,
    selectedKey,
    differencesKey,
    batchDirectBlocked,
    requiresHouse,
    isDetail,
    existing?.version,
  ]);

  const adopt = () => {
    if (!selected || batchDirectBlocked || requiresHouse || isDetail) return;
    form.setFieldsValue(seaVoyageFormValues(selected));
  };
  const validate = useCallback(async () => {
    // Form 校验可先于 useWatch/Effect 渲染；直接读取当前身份，禁止旧确认穿过该窗口。
    const currentIdentity = seaMasterBillIdentity(
      form.getFieldValue('seaMasterBillMasterNo'),
      form.getFieldValue('shippingLineId'),
    );
    if (!currentIdentity) return;
    if (currentIdentity !== identity)
      throw new Error('正在核对已有主单，请稍候');
    if (matching) throw new Error('正在核对已有主单，请稍候');
    if (error) throw new Error(error);
    if (batchDirectBlocked)
      throw new Error('该主单已被直单订单占用，如需拼单请先将其转为分单');
    if (
      candidate &&
      form.getFieldValue('seaDocumentStructure') !==
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
    )
      throw new Error('共享主单仅支持有货代分单（HOUSE），请选择分单模式');
    if (candidate && isSingleMemberCorrection)
      throw new Error('新主单身份已存在，当前阶段不允许直接合并');
    if (candidate && !selected) throw new Error('请选择本票共用的航程');
    if (differences.length > 0)
      throw new Error('航程信息与已有主单冲突，请确认采用主单航程');
    if (candidate && !form.getFieldValue('seaMasterBillCandidateId'))
      throw new Error('发现已有主单，请明确确认关联后再保存');
  }, [
    form,
    identity,
    matching,
    error,
    batchDirectBlocked,
    requiresHouse,
    candidate,
    isSingleMemberCorrection,
    selected,
    differences.length,
    candidateId,
  ]);

  useEffect(() => {
    void form
      .validateFields(['seaMasterBillMasterNo'], { dirty: true })
      .catch(() => undefined);
  }, [form, validate]);

  return {
    candidate,
    selected,
    matching,
    checked,
    error,
    batchDirectBlocked,
    requiresHouse,
    differences,
    isSingleMemberCorrection,
    validate,
    adopt,
    select: (id: string) => setSelection({ identity, id }),
    retry: () => query.refetch(),
    waitingForCarrier: Boolean(masterNo && !shippingLineId),
    ready: candidateReady && Boolean(candidateId),
    existing,
  };
}
