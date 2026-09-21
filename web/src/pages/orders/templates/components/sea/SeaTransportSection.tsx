import { ProFormDateTimePicker, ProFormText } from '@ant-design/pro-components';
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import isoWeek from 'dayjs/plugin/isoWeek';
import React, { useCallback, useEffect, useState } from 'react';
import { FormRow, ProFormSearchableSelect } from '@/components/ui';
import { SeaDocumentStructure } from '@/enums.generated';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import { containerOwnershipOptions } from '../../../common';
import {
  OrderContainerRequestFields,
  type SelectOption,
} from '../../../order-plan-fields';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';
import type { TemplateProps } from '../../types';

dayjs.extend(isoWeek);

export function splitSeaVesselVoyage(value?: string) {
  const normalized = value?.trim() || '';
  if (!normalized) return { vesselName: undefined, voyageNo: undefined };
  const slashIndex = normalized.indexOf('/');
  if (slashIndex >= 0) {
    return {
      vesselName: normalized.slice(0, slashIndex).trim() || undefined,
      voyageNo: normalized.slice(slashIndex + 1).trim() || undefined,
    };
  }
  const lastSpaceIndex = normalized.lastIndexOf(' ');
  if (lastSpaceIndex >= 0) {
    return {
      vesselName: normalized.slice(0, lastSpaceIndex).trim() || undefined,
      voyageNo: normalized.slice(lastSpaceIndex + 1).trim() || undefined,
    };
  }
  return { vesselName: normalized, voyageNo: undefined };
}

export function SeaMasterBillFields({
  disabled = false,
  isDetail = false,
}: {
  disabled?: boolean;
  isDetail?: boolean;
}) {
  const form = Form.useFormInstance();
  const masterNo = Form.useWatch('seaMasterBillMasterNo', form);
  // 候选标识由匹配结果写入 store，没有对应 Form.Item，需监听未注册字段。
  const candidateId = Form.useWatch('seaMasterBillCandidateId', {
    form,
    preserve: true,
  });
  const candidateTeId = Form.useWatch('seaMasterBillCandidateTeId', {
    form,
    preserve: true,
  });
  const existingMbl = Form.useWatch('seaMasterBill', {
    form,
    preserve: true,
  }) as API.SeaMasterBillSummary | undefined;

  const originLocationId = Form.useWatch('originLocationId', form);
  const dischargeLocationId = Form.useWatch('dischargeLocationId', form);
  const transitLocationId = Form.useWatch('transitLocationId', form);
  const shippingLineId = Form.useWatch('shippingLineId', form);
  const vesselVoyage = Form.useWatch('vesselVoyage', form);
  const etd = Form.useWatch('etd', form);
  const eta = Form.useWatch('eta', form);

  const [candidate, setCandidate] = useState<API.SeaMasterBillCandidate | null>(
    null,
  );
  const [conflicts, setConflicts] = useState<API.SeaVoyageConflict[]>([]);
  const [candidateMatched, setCandidateMatched] = useState(false);
  const [candidateMatching, setCandidateMatching] = useState(false);
  const [candidateMatchError, setCandidateMatchError] = useState<string>();
  // 命中批次含直单成员：全员分单制下禁止加拼，横幅转红色阻断文案。
  const [batchDirectBlocked, setBatchDirectBlocked] = useState(false);

  // 多成员 MBL 检查（锁定修改）
  const isMultiMemberLocked =
    isDetail && !!existingMbl && (existingMbl.memberCount ?? 0) > 1;
  const isSingleMemberCorrection =
    isDetail &&
    !!existingMbl &&
    (existingMbl.memberCount ?? 0) <= 1 &&
    ((masterNo && masterNo !== existingMbl.masterNo) ||
      (shippingLineId && shippingLineId !== existingMbl.shippingLineId));

  useEffect(() => {
    const rawMasterNo = masterNo || '';
    const selectedShippingLineId = shippingLineId;

    if (
      !rawMasterNo ||
      !selectedShippingLineId ||
      !/^[A-Za-z0-9]+$/.test(rawMasterNo)
    ) {
      setCandidate(null);
      setConflicts([]);
      setCandidateMatched(false);
      setCandidateMatching(false);
      setCandidateMatchError(undefined);
      setBatchDirectBlocked(false);
      form?.setFieldValue('seaMasterBillBatchHouseNos', undefined);
      return;
    }

    // 若详情页未变更主单号与船公司，无需展示候选关联
    if (
      isDetail &&
      existingMbl &&
      existingMbl.masterNo === rawMasterNo &&
      existingMbl.shippingLineId === selectedShippingLineId
    ) {
      setCandidate(null);
      setConflicts([]);
      setCandidateMatched(false);
      setCandidateMatching(false);
      setCandidateMatchError(undefined);
      setBatchDirectBlocked(false);
      form?.setFieldValue('seaMasterBillBatchHouseNos', undefined);
      return;
    }

    const currentMblVersion = isDetail ? existingMbl?.version : undefined;
    form?.setFieldValue('seaMasterBillCandidateId', undefined);
    form?.setFieldValue('seaMasterBillCandidateTeId', undefined);
    form?.setFieldValue('seaMasterBillExpectedCandidateTeVersion', undefined);
    form?.setFieldValue('seaMasterBillBatchHouseNos', undefined);
    form?.setFieldValue(
      'seaMasterBillExpectedCandidateVersion',
      currentMblVersion,
    );
    setCandidate(null);
    setConflicts([]);
    setCandidateMatched(false);
    setBatchDirectBlocked(false);
    setCandidateMatching(true);
    setCandidateMatchError(undefined);

    let isSubscribed = true;
    const timer = setTimeout(async () => {
      try {
        const etdStr = etd ? dayjs(etd).toISOString() : undefined;
        const etaStr = eta ? dayjs(eta).toISOString() : undefined;
        const { vesselName, voyageNo } = splitSeaVesselVoyage(vesselVoyage);
        const resp = await orderServiceMatchSeaMasterBillCandidate({
          masterNo: rawMasterNo,
          shippingLineId: selectedShippingLineId,
          originLocationId: originLocationId || undefined,
          dischargeLocationId: dischargeLocationId || undefined,
          transitLocationId: transitLocationId || undefined,
          vesselName,
          voyageNo,
          etd: etdStr,
          eta: etaStr,
        });
        if (!isSubscribed) return;
        setCandidateMatching(false);
        if (resp?.matched && resp.candidate) {
          setCandidate(resp.candidate);
          setConflicts(resp.conflicts || []);
          setCandidateMatched(true);
          // 批次内全部规范化分单号（含作废）写入表单，供分单号失焦即时排重提示；
          // 服务端事务内预查为权威口径。
          form?.setFieldValue(
            'seaMasterBillBatchHouseNos',
            resp.candidate.batchNormalizedHouseNos ?? [],
          );
          const hasDirectMember = (resp.candidate.members ?? []).some(
            (member) =>
              member.documentStructure ===
              SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          );
          setBatchDirectBlocked(hasDirectMember);
          const availableExecutions = resp.candidate.transportExecutions ?? [];
          if (availableExecutions.length === 1) {
            form?.setFieldValue(
              'seaMasterBillCandidateTeId',
              availableExecutions[0].id,
            );
            form?.setFieldValue(
              'seaMasterBillExpectedCandidateTeVersion',
              availableExecutions[0].version,
            );
          }
          // 全员分单制：新建页命中全 HOUSE 批次且无航程冲突时自动关联，
          // 候选 ID + 版本自动带入保存请求；服务端事务内复验兜底。
          if (
            !isDetail &&
            !hasDirectMember &&
            (resp.conflicts ?? []).length === 0 &&
            availableExecutions.length >= 1
          ) {
            form?.setFieldValue('seaMasterBillCandidateId', resp.candidate.id);
            form?.setFieldValue(
              'seaMasterBillExpectedCandidateVersion',
              resp.candidate.version,
            );
          }
        } else {
          setCandidate(null);
          setConflicts([]);
          setCandidateMatched(false);
          setBatchDirectBlocked(false);
          form?.setFieldValue('seaMasterBillBatchHouseNos', undefined);
          form?.setFieldValue('seaMasterBillCandidateId', undefined);
          form?.setFieldValue(
            'seaMasterBillExpectedCandidateVersion',
            currentMblVersion,
          );
        }
      } catch (error: unknown) {
        if (!isSubscribed) return;
        const requestError = error as Error;
        setCandidate(null);
        setConflicts([]);
        setCandidateMatched(false);
        setBatchDirectBlocked(false);
        setCandidateMatching(false);
        setCandidateMatchError(
          requestError.message || '主单候选查询失败，请重试后再保存',
        );
      }
    }, 300);

    return () => {
      isSubscribed = false;
      clearTimeout(timer);
    };
  }, [
    masterNo,
    originLocationId,
    dischargeLocationId,
    transitLocationId,
    shippingLineId,
    vesselVoyage,
    etd,
    eta,
    isDetail,
    existingMbl,
    form,
  ]);

  const validateMasterBillCandidate = useCallback(async () => {
    if (candidateMatching) {
      throw new Error('正在核对已有主单，请稍候');
    }
    if (candidateMatchError) {
      throw new Error(candidateMatchError);
    }
    if (conflicts.length > 0) {
      throw new Error('航程信息与已有主单冲突，不能确认关联');
    }
    if (batchDirectBlocked) {
      throw new Error('该主单已被直单订单占用，如需拼单请先将其转为分单');
    }
    if (candidateMatched && isSingleMemberCorrection) {
      throw new Error('新主单身份已存在，当前阶段不允许直接合并');
    }
    if (candidateMatched && !candidateId) {
      throw new Error('发现已有主单，请明确确认关联后再保存');
    }
  }, [
    candidateMatching,
    candidateMatchError,
    conflicts.length,
    batchDirectBlocked,
    candidateMatched,
    isSingleMemberCorrection,
    candidateId,
  ]);

  useEffect(() => {
    // 候选核对状态不在表单 store 中，变化后需重验已交互字段，避免残留等待提示。
    // 执行完整 rules，保留必填、格式与真实业务阻断错误。
    void form
      .validateFields(['seaMasterBillMasterNo'], { dirty: true })
      .catch(() => undefined);
  }, [form, validateMasterBillCandidate]);

  const selectedCandidateTe = candidate?.transportExecutions?.find(
    (item) => item.id === candidateTeId,
  );

  return (
    <>
      <div style={{ gridColumn: 'span 2' }}>
        <ProFormText
          name="seaMasterBillMasterNo"
          label="MBL 主单号"
          placeholder="请输入主单号"
          disabled={disabled || isMultiMemberLocked}
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
            {
              validator: validateMasterBillCandidate,
            },
          ]}
          normalize={(value) =>
            typeof value === 'string' ? value.toUpperCase() : value
          }
          fieldProps={{
            style: { textTransform: 'uppercase' },
          }}
        />
      </div>

      {isSingleMemberCorrection && (
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

      {candidateMatched && candidate && (
        <div style={{ gridColumn: '1 / -1', marginBottom: 16 }}>
          <Card
            size="small"
            style={{
              background: batchDirectBlocked ? '#fff2f0' : '#f6ffed',
              borderColor: batchDirectBlocked ? '#ffccc7' : '#b7eb8f',
            }}
          >
            <Space orientation="vertical" style={{ width: '100%' }}>
              {batchDirectBlocked ? (
                <Alert
                  type="error"
                  showIcon
                  title="该主单已被直单订单占用，如需拼单请先将其转为分单"
                  description={`共享批次：${candidate.masterNo} | 成员：${candidate.memberCount ?? 0} 票（含直单订单）`}
                />
              ) : (
                <div>
                  <span style={{ fontWeight: 600, color: '#389e0d' }}>
                    {isDetail
                      ? '🔍 匹配到已有共享 MBL'
                      : '✅ 已自动关联共享主单批次'}
                    （{candidate.memberCount ?? 0} 票）：{candidate.masterNo}
                  </span>
                  <div style={{ fontSize: 12, color: '#595959', marginTop: 4 }}>
                    船公司: {candidate.shippingLineName || '-'} | 版本: v
                    {candidate.version}
                  </div>
                </div>
              )}

              <Select
                value={candidateTeId}
                placeholder="请选择本票关联的实际航次"
                disabled={batchDirectBlocked}
                options={(candidate.transportExecutions ?? []).map((te) => ({
                  value: te.id,
                  label: `${te.vesselName || '-'} / ${te.voyageNo || '-'} / ${te.etd || '无 ETD'}`,
                }))}
                onChange={(value) => {
                  const te = candidate.transportExecutions?.find(
                    (item) => item.id === value,
                  );
                  form?.setFieldValue('seaMasterBillCandidateTeId', value);
                  form?.setFieldValue(
                    'seaMasterBillExpectedCandidateTeVersion',
                    te?.version,
                  );
                }}
              />

              {selectedCandidateTe && (
                <div style={{ fontSize: 13, color: '#595959' }}>
                  <span>运输执行：</span>
                  <span>
                    船名航次: {selectedCandidateTe.vesselName || '-'} /{' '}
                    {selectedCandidateTe.voyageNo || '-'} |{' '}
                  </span>
                  <span>
                    起运港: {selectedCandidateTe.originLocationName || '-'} |{' '}
                  </span>
                  <span>
                    卸货港: {selectedCandidateTe.dischargeLocationName || '-'} |{' '}
                  </span>
                  <span>ETD: {selectedCandidateTe.etd || '-'} | </span>
                  <span>ETA: {selectedCandidateTe.eta || '-'}</span>
                </div>
              )}

              {conflicts.length > 0 && (
                <Alert
                  type="warning"
                  showIcon
                  title="检测到航程信息冲突，不能关联此主单"
                  description={
                    <ul style={{ margin: 0, paddingLeft: 16 }}>
                      {conflicts.map((c) => (
                        <li key={`${c.field}-${c.message}`}>
                          {c.message} (主单值: {c.masterValue || '空'}, 当前值:{' '}
                          {c.orderValue || '空'})
                        </li>
                      ))}
                    </ul>
                  }
                />
              )}
            </Space>
          </Card>
        </div>
      )}

      {candidateMatchError && (
        <div style={{ gridColumn: '1 / -1', marginBottom: 16 }}>
          <Alert
            type="error"
            showIcon
            title="主单候选查询失败"
            description={candidateMatchError}
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

export function SeaContainerPlanFields({
  options,
}: {
  options: SelectOption[];
}) {
  const form = Form.useFormInstance();
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
              <Button
                danger
                size="small"
                htmlType="button"
                onClick={() => form?.setFieldValue('containerRequests', [])}
              >
                清空箱量计划
              </Button>
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
  const weekValue =
    etd && dayjs(etd).isValid() ? `W${dayjs(etd).isoWeek()}` : '';

  return (
    <>
      <ProFormDateTimePicker
        name="etd"
        label="ETD"
        fieldProps={{
          style: { width: '100%' },
          onChange: (date) => {
            if (date) {
              const currentEta = form?.getFieldValue('eta');
              if (currentEta?.isBefore(date)) {
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
      <ProFormDateTimePicker
        name="eta"
        label="ETA"
        dependencies={['etd']}
        rules={[
          ({ getFieldValue }) => ({
            validator(_, value) {
              const etdVal = getFieldValue('etd');
              if (
                !value ||
                !etdVal ||
                value.isAfter(etdVal) ||
                value.isSame(etdVal)
              ) {
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

export function getSeaTransportFields(
  props: TemplateProps,
  createLayout = false,
) {
  const { locationOptions, searchLocations, containerSpecOptions, isDetail } =
    props;
  return {
    containers: <SeaContainerPlanFields options={containerSpecOptions} />,
    master: <SeaMasterBillFields isDetail={isDetail} />,
    ownership: (
      <ProFormSearchableSelect
        name="containerOwnership"
        label="货主箱标记"
        options={containerOwnershipOptions}
        placeholder="请选择 COC / SOC"
      />
    ),
    vessel: (
      <div style={{ gridColumn: 'span 2' }}>
        <ProFormText
          name="vesselVoyage"
          label="船名航次"
          placeholder="请输入船名航次"
          fieldProps={{
            suffix: (
              <Tooltip title="船期与船舶实时动态追踪">
                <a
                  href="https://www.shipxy.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  tabIndex={-1}
                  style={{ fontSize: 12, color: '#1677ff' }}
                  onClick={(e) => e.stopPropagation()}
                >
                  船在哪儿
                </a>
              </Tooltip>
            ),
          }}
        />
      </div>
    ),
    ports: (
      <FormRow cols={createLayout ? 4 : 6}>
        <div style={{ gridColumn: createLayout ? undefined : 'span 2' }}>
          <ProFormSearchableSelect
            name="originLocationId"
            label="起运港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择起运港或地点"
          />
        </div>
        <div style={{ gridColumn: createLayout ? undefined : 'span 2' }}>
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
          options={locationOptions}
          request={async ({ keyWords }) => searchLocations(keyWords)}
          fieldProps={{ filterOption: false }}
          placeholder="请选择卸货港"
        />
        <ProFormSearchableSelect
          name="transitLocationId"
          label="中转港"
          options={locationOptions}
          request={async ({ keyWords }) => searchLocations(keyWords)}
          fieldProps={{ filterOption: false }}
          placeholder="请选择中转港"
        />
      </FormRow>
    ),
    schedule: (
      <FormRow cols={3}>
        <SeaScheduleDateFields />
      </FormRow>
    ),
    cutoffs: (
      <FormRow cols={4}>
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
      </FormRow>
    ),
  };
}

export function buildSeaTransportSection(props: TemplateProps) {
  const fields = getSeaTransportFields(props);
  return {
    key: 'transportInfo',
    title: '配舱信息',
    content: (
      <div style={{ display: 'grid', gap: 12, width: '100%' }}>
        {fields.containers}
        <FormRow cols={6}>
          {fields.master}
          {fields.ownership}
          {fields.vessel}
        </FormRow>
        {fields.ports}
        {fields.schedule}
        {fields.cutoffs}
      </div>
    ),
  };
}
