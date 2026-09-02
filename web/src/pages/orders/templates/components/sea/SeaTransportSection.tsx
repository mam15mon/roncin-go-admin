import { ProFormDateTimePicker, ProFormText } from '@ant-design/pro-components';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Input,
  Space,
  Tooltip,
} from 'antd';
import dayjs from 'dayjs';
import isoWeek from 'dayjs/plugin/isoWeek';
import React, { useEffect, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import { searchPartnerOptions } from '@/utils/options';
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
  searchIssuers,
}: {
  disabled?: boolean;
  isDetail?: boolean;
  searchIssuers?: (keyword?: string) => Promise<SelectOption[]>;
}) {
  const form = Form.useFormInstance();
  const masterNo = Form.useWatch('seaMasterBillMasterNo', form);
  const issuerPartnerId = Form.useWatch('seaMasterBillIssuerPartnerId', form);
  const candidateId = Form.useWatch('seaMasterBillCandidateId', form);
  const existingMbl = Form.useWatch('seaMasterBill', form) as
    | API.SeaMasterBillSummary
    | undefined;

  const originLocationId = Form.useWatch('originLocationId', form);
  const dischargeLocationId = Form.useWatch('dischargeLocationId', form);
  const transitLocationId = Form.useWatch('transitLocationId', form);
  const carrierId = Form.useWatch('carrierId', form);
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

  // 多成员 MBL 检查（锁定修改）
  const isMultiMemberLocked =
    isDetail && !!existingMbl && (existingMbl.memberCount ?? 0) > 1;
  const isSingleMemberCorrection =
    isDetail &&
    !!existingMbl &&
    (existingMbl.memberCount ?? 0) <= 1 &&
    ((masterNo && masterNo !== existingMbl.masterNo) ||
      (issuerPartnerId && issuerPartnerId !== existingMbl.issuerPartnerId));

  useEffect(() => {
    const rawMasterNo = masterNo || '';
    const partnerId = issuerPartnerId;

    if (!rawMasterNo || !partnerId || !/^[A-Za-z0-9]+$/.test(rawMasterNo)) {
      setCandidate(null);
      setConflicts([]);
      setCandidateMatched(false);
      setCandidateMatching(false);
      setCandidateMatchError(undefined);
      return;
    }

    // 若详情页未变更主单号与签发方，无需展示候选关联
    if (
      isDetail &&
      existingMbl &&
      existingMbl.masterNo === rawMasterNo &&
      existingMbl.issuerPartnerId === partnerId
    ) {
      setCandidate(null);
      setConflicts([]);
      setCandidateMatched(false);
      setCandidateMatching(false);
      setCandidateMatchError(undefined);
      return;
    }

    const currentMblVersion = isDetail ? existingMbl?.version : undefined;
    form?.setFieldValue('seaMasterBillCandidateId', undefined);
    form?.setFieldValue(
      'seaMasterBillExpectedCandidateVersion',
      currentMblVersion,
    );
    setCandidate(null);
    setConflicts([]);
    setCandidateMatched(false);
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
          issuerPartnerId: partnerId,
          originLocationId: originLocationId || undefined,
          dischargeLocationId: dischargeLocationId || undefined,
          transitLocationId: transitLocationId || undefined,
          carrierId: carrierId || undefined,
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
        } else {
          setCandidate(null);
          setConflicts([]);
          setCandidateMatched(false);
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
    issuerPartnerId,
    originLocationId,
    dischargeLocationId,
    transitLocationId,
    carrierId,
    vesselVoyage,
    etd,
    eta,
    isDetail,
    existingMbl,
    form,
  ]);

  const isConfirmed = !!candidateId;

  return (
    <>
      <Form.Item name="seaMasterBillCandidateId" hidden>
        <Input type="hidden" />
      </Form.Item>
      <Form.Item name="seaMasterBillExpectedCandidateVersion" hidden>
        <Input type="hidden" />
      </Form.Item>

      <Col className="col-5">
        <ProFormText
          name="seaMasterBillMasterNo"
          label="MBL 主单号"
          placeholder="请输入主单号 (仅大写字母与数字)"
          disabled={disabled || isMultiMemberLocked}
          tooltip={
            isMultiMemberLocked
              ? '该主单已关联多票订单，禁止直接在此修改主单号或签发主体'
              : undefined
          }
          rules={[
            { required: true, message: '请输入海运出口 MBL 主单号' },
            {
              pattern: /^[A-Za-z0-9]+$/,
              message: '主单号仅允许包含英文字母和数字，禁止包含空格或特殊字符',
            },
            {
              validator: async () => {
                if (candidateMatching) {
                  throw new Error('正在核对已有主单，请稍候');
                }
                if (candidateMatchError) {
                  throw new Error(candidateMatchError);
                }
                if (conflicts.length > 0) {
                  throw new Error('航程信息与已有主单冲突，不能确认关联');
                }
                if (candidateMatched && isSingleMemberCorrection) {
                  throw new Error('新主单身份已存在，当前阶段不允许直接合并');
                }
                if (candidateMatched && !candidateId) {
                  throw new Error('发现已有主单，请明确确认关联后再保存');
                }
              },
            },
          ]}
          fieldProps={{
            onChange: (e) => {
              const val = e.target.value.replace(/[a-z]/g, (char) =>
                char.toUpperCase(),
              );
              form?.setFieldValue('seaMasterBillMasterNo', val);
            },
          }}
        />
      </Col>

      <Col className="col-5">
        <ProFormSearchableSelect
          name="seaMasterBillIssuerPartnerId"
          label="实际签发/承运主体"
          placeholder="请选择主单签发方"
          disabled={disabled || isMultiMemberLocked}
          tooltip={
            isMultiMemberLocked
              ? '该主单已关联多票订单，禁止直接在此修改主单号或签发主体'
              : undefined
          }
          rules={[{ required: true, message: '请选择主单签发/承运主体' }]}
          request={async ({ keyWords }) => {
            if (searchIssuers) {
              return searchIssuers(keyWords);
            }
            return searchPartnerOptions(keyWords);
          }}
        />
      </Col>

      {isSingleMemberCorrection && (
        <Col className="col-5">
          <ProFormText
            name="seaMasterBillCorrectionReason"
            label="主单更正原因"
            placeholder="请输入主单号/签发方更正原因"
            rules={[
              {
                required: true,
                message: '单票修改主单号或签发方必须填写更正原因',
              },
            ]}
          />
        </Col>
      )}

      {candidateMatched && candidate && (
        <Col span={24} style={{ marginBottom: 16 }}>
          <Card
            size="small"
            style={{
              background: '#f6ffed',
              borderColor: '#b7eb8f',
            }}
          >
            <Space direction="vertical" style={{ width: '100%' }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  flexWrap: 'wrap',
                  gap: 8,
                }}
              >
                <span style={{ fontWeight: 600, color: '#389e0d' }}>
                  🔍 匹配到已有共享 MBL：{candidate.masterNo} (签发方:{' '}
                  {candidate.issuerPartnerName} | 版本: v{candidate.version} |
                  成员: {candidate.memberCount} 票)
                </span>
                <Checkbox
                  checked={isConfirmed}
                  disabled={conflicts.length > 0 || isSingleMemberCorrection}
                  onChange={(e) => {
                    if (e.target.checked) {
                      form?.setFieldValue(
                        'seaMasterBillCandidateId',
                        candidate.id,
                      );
                      form?.setFieldValue(
                        'seaMasterBillExpectedCandidateVersion',
                        candidate.version,
                      );
                    } else {
                      form?.setFieldValue(
                        'seaMasterBillCandidateId',
                        undefined,
                      );
                      form?.setFieldValue(
                        'seaMasterBillExpectedCandidateVersion',
                        undefined,
                      );
                    }
                  }}
                >
                  <strong
                    style={{ color: isConfirmed ? '#1677ff' : '#cf1322' }}
                  >
                    {conflicts.length > 0
                      ? '航程冲突，禁止关联'
                      : isSingleMemberCorrection
                        ? '新主单身份已存在，禁止直接合并'
                        : '确认关联已有主单'}
                  </strong>
                </Checkbox>
              </div>

              {candidate.transportExecution && (
                <div style={{ fontSize: 13, color: '#595959' }}>
                  <span>运输执行：</span>
                  <span>
                    船名航次: {candidate.transportExecution.vesselName || '-'} /{' '}
                    {candidate.transportExecution.voyageNo || '-'} |{' '}
                  </span>
                  <span>
                    起运港:{' '}
                    {candidate.transportExecution.originLocationName || '-'} |{' '}
                  </span>
                  <span>
                    卸货港:{' '}
                    {candidate.transportExecution.dischargeLocationName || '-'}{' '}
                    |{' '}
                  </span>
                  <span>ETD: {candidate.transportExecution.etd || '-'} | </span>
                  <span>ETA: {candidate.transportExecution.eta || '-'}</span>
                </div>
              )}

              {conflicts.length > 0 && (
                <Alert
                  type="warning"
                  showIcon
                  message="检测到航程信息冲突，不能关联此主单"
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

              {!isConfirmed && (
                <div style={{ color: '#fa8c16', fontSize: 12 }}>
                  ⚠️
                  提示：系统检测到已有相同主单。若不勾选“确认关联已有主单”，保存时将被服务端拦截。
                </div>
              )}
            </Space>
          </Card>
        </Col>
      )}

      {candidateMatchError && (
        <Col span={24} style={{ marginBottom: 16 }}>
          <Alert
            type="error"
            showIcon
            message="主单候选查询失败"
            description={candidateMatchError}
          />
        </Col>
      )}
    </>
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
      <Col span={24}>
        <Alert
          type={containerRequests.length > 0 ? 'warning' : 'info'}
          showIcon
          title="散杂货不使用箱型箱量、箱号或封号配置"
          description={
            containerRequests.length > 0
              ? '切换托运类型前已经录入箱型箱量，请确认后清空，系统不会静默删除已有数据。'
              : '页面已隐藏集装箱专属配置，货物将按件数、毛重、体积和计费吨管理。'
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
          style={{ marginBottom: 16 }}
        />
      </Col>
    );
  }
  return <OrderContainerRequestFields options={options} />;
}

export function SeaScheduleDateFields() {
  const form = Form.useFormInstance();
  const etd = Form.useWatch('etd', form);
  const weekValue =
    etd && dayjs(etd).isValid() ? `W${dayjs(etd).isoWeek()}` : '';

  return (
    <>
      <Col className="col-5">
        <ProFormDateTimePicker
          name="etd"
          label="ETD (预计开航)"
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
      </Col>
      <Col className="col-5">
        <Form.Item label="WEEK" style={{ marginInline: 0 }}>
          <Input
            value={weekValue}
            placeholder="依据 ETD 自动生成"
            disabled
            style={{ width: '100%' }}
          />
        </Form.Item>
      </Col>
      <Col className="col-5">
        <ProFormDateTimePicker
          name="eta"
          label="ETA (预计到达)"
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
      </Col>
    </>
  );
}

export function buildSeaTransportSection(props: TemplateProps) {
  const {
    locationOptions,
    searchLocations,
    containerSpecOptions,
    searchIssuers,
    isDetail,
  } = props;

  return {
    key: 'transportInfo',
    title: '配舱信息',
    content: (
      <>
        {/* 第 1 行：海运出口共享 MBL 主单号与实际签发主体 */}
        <SeaMasterBillFields
          isDetail={isDetail}
          searchIssuers={searchIssuers}
        />

        {/* 第 2 行：箱型箱量；HBL 只在独立“提单信息”区块维护 */}
        <SeaContainerPlanFields options={containerSpecOptions} />

        {/* 第 3 行：航线 4 港口（起运港、目的港、卸货港、中转港） */}
        <Col className="col-5">
          <ProFormSearchableSelect
            name="originLocationId"
            label="起运港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择起运港或地点"
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="destinationLocationId"
            label="目的港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择目的港或地点"
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="dischargeLocationId"
            label="卸货港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择卸货港"
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="transitLocationId"
            label="中转港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择中转港"
          />
        </Col>
        <Col className="col-5" />

        {/* 第 4 行：货主箱标记、船名航次、ETD、WEEK、ETA */}
        <Col className="col-5">
          <ProFormSearchableSelect
            name="containerOwnership"
            label="货主箱标记"
            options={containerOwnershipOptions}
            placeholder="请选择 COC / SOC"
          />
        </Col>
        <Col className="col-5">
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
                    style={{ fontSize: 12, color: '#1677ff' }}
                    onClick={(e) => e.stopPropagation()}
                  >
                    船在哪儿
                  </a>
                </Tooltip>
              ),
            }}
          />
        </Col>
        <SeaScheduleDateFields />

        {/* 第 5 行：SI截关时间、单证截关时间(截单时间)、报关截关时间(截关时间)、VGM截关时间 */}
        <Col className="col-5">
          <ProFormDateTimePicker
            name="siCutoff"
            label="SI截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5">
          <ProFormDateTimePicker
            name="docCutoff"
            label="单证截关时间"
            tooltip="即截单时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5">
          <ProFormDateTimePicker
            name="customsCutoff"
            label="报关截关时间"
            tooltip="即截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5">
          <ProFormDateTimePicker
            name="vgmCutoff"
            label="VGM截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5" />
      </>
    ),
  };
}
