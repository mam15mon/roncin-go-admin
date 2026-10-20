import {
  type ActionType,
  PageContainer,
  type ProFormInstance,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Descriptions, Result, Space, Tag } from 'antd';
import dayjs from 'dayjs';
import { useRef, useState } from 'react';
import { useParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  GuardedModalForm,
  PageHeaderShell,
  ProFormSearchableSelect,
  ResponsiveProTable,
  SectionCard,
} from '@/components/ui';
import { getOrderFlowStatusText } from '@/constants/statusMeta';
import { OrderBusinessType } from '@/enums.generated';
import { history } from '@/router/history';
import {
  airExportServiceGetAirMasterWaybill,
  airExportServiceLinkAirMasterWaybill,
  airExportServiceListAirLinkCandidates,
  airExportServicePreviewAirMasterWaybillUpdate,
  airExportServiceUpdateAirMasterWaybill,
} from '@/services/roncin/airExportService';
import { ensureListResponse, isWriteResponse } from '@/utils/api';
import { formatDate, trimDecimal } from '@/utils/format';
import { searchOrderLocations } from './common';
import AirWeightPanel from './order-kinds/air-export/AirWeightPanel';
import { serializeAirMaster } from './order-kinds/air-export/form-adapter';
import { AirMasterFields } from './templates/components/air/AirMasterFields';

type MasterEditValues = { input: API.AirMasterWaybillInput; reason: string };

/** 摘要沿用货量表单的三位精度，消除 double 汇总与差值的二进制尾数，仅用于展示。 */
function formatCargoSummary(value?: number): string {
  return value === undefined ? '-' : trimDecimal(Number(value.toFixed(3)));
}

export default function AirMasterPage() {
  const { masterId = '' } = useParams<{ masterId: string }>();
  const access = useAccess();
  const queryClient = useQueryClient();
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const canRead = access.canOrder(OrderBusinessType.BUSINESS_TYPE_AE, 'read');
  const canUpdate = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_AE,
    'update',
  );
  const canRestore =
    canUpdate &&
    access.canOrder(OrderBusinessType.BUSINESS_TYPE_AE, 'transition');
  const [editOpen, setEditOpen] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [impact, setImpact] = useState<{
    values: MasterEditValues;
    response: API.PreviewAirMasterWaybillUpdateResponse;
  }>();
  const memberRef = useRef<ActionType | undefined>(undefined);
  const linkForm = useRef<
    | ProFormInstance<{
        orderId: string;
        expectedOrderVersion: string;
        houseNo: string;
        reason: string;
      }>
    | undefined
  >(undefined);
  const query = useQuery({
    queryKey: ['air-export', 'master', masterId, { organizationId }],
    enabled: Boolean(masterId && canRead && organizationId),
    queryFn: async ({ signal }) => {
      const response = await airExportServiceGetAirMasterWaybill(
        { id: masterId, page: 1, pageSize: 20 },
        { signal },
      );
      if (!response?.data) throw new Error('主单加载失败');
      return response;
    },
  });
  const current = query.isError ? undefined : query.data;
  const master = current?.data;
  const input = master?.content;
  const totals = current?.totals;
  const reload = async () => {
    await Promise.all([
      query.refetch(),
      queryClient.invalidateQueries({ queryKey: ['air-weight'] }),
    ]);
    memberRef.current?.reload();
  };
  if (!canRead) return <Result status="403" title="无权查看空运主单" />;
  if (query.isError)
    return (
      <Result
        status="error"
        title="主单加载失败"
        extra={<Button onClick={() => void query.refetch()}>重试</Button>}
      />
    );
  return (
    <PageContainer
      title={false}
      breadcrumbRender={false}
      loading={query.isPending}
    >
      <PageHeaderShell
        title={input?.masterNo || '共享航空主运单'}
        subTitle="共同运输事实、真实成员与计费重量"
        onBack={() => history.push('/orders/air-export')}
        tags={<Tag color="blue">MAWB · v{master?.version}</Tag>}
        actions={
          <Space>
            {access.canOrder(OrderBusinessType.BUSINESS_TYPE_AE, 'create') && (
              <Button
                type="primary"
                onClick={() =>
                  history.push(`/orders/air-export/new?masterId=${masterId}`)
                }
              >
                从本主单继续建票
              </Button>
            )}
            {canUpdate && (
              <>
                {canRestore && (
                  <Button onClick={() => setLinkOpen(true)}>
                    恢复关联已终止订单
                  </Button>
                )}
                <Button
                  onClick={() => {
                    setImpact(undefined);
                    setEditOpen(true);
                  }}
                >
                  修改共同资料
                </Button>
              </>
            )}
            <Button onClick={() => void reload()}>刷新</Button>
          </Space>
        }
      />
      <Space
        orientation="vertical"
        size={12}
        style={{ width: '100%', marginTop: 12 }}
      >
        <SectionCard title="共同运输与单证">
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, lg: 4 }}
            items={[
              {
                key: 'airline',
                label: '航空公司',
                children: master?.airlineName || '-',
              },
              {
                key: 'origin',
                label: '起运机场',
                children: master?.originAirportName || '-',
              },
              {
                key: 'destination',
                label: '目的机场',
                children: master?.destinationAirportName || '-',
              },
              {
                key: 'flight',
                label: '航班号',
                children: input?.flightNo || '-',
              },
              {
                key: 'etd',
                label: '计划起飞',
                children: formatDate(input?.etd, 'minute'),
              },
              {
                key: 'eta',
                label: '计划到达',
                children: formatDate(input?.eta, 'minute'),
              },
              {
                key: 'departure',
                label: '实际起飞',
                children: formatDate(input?.actualDepartureAt, 'minute'),
              },
              {
                key: 'arrival',
                label: '实际到达',
                children: formatDate(input?.actualArrivalAt, 'minute'),
              },
              {
                key: 'cutoff',
                label: '截单时间',
                children: formatDate(input?.documentCutoffAt, 'minute'),
              },
              {
                key: 'customs',
                label: '报关截止',
                children: formatDate(input?.customsCutoffAt, 'minute'),
              },
              {
                key: 'status',
                label: '主单单证',
                children:
                  input?.status === 'RELEASED'
                    ? '已签发'
                    : input?.status === 'CONFIRMED'
                      ? '已确认'
                      : '资料待完善',
              },
              { key: 'notes', label: '备注', children: input?.notes || '-' },
            ]}
          />
          {[
            ['截单', input?.documentCutoffAt],
            ['报关', input?.customsCutoffAt],
          ]
            .filter(([, at]) => at && dayjs(at).isBefore(dayjs()))
            .map(([label, at]) => (
              <Alert
                key={label}
                type="warning"
                showIcon
                title={`${label}截止时间已过：${formatDate(at, 'minute')}`}
                description="请在推进对应节点时记录处理原因。资料缺失仍会阻止操作；超时提示不代表已向外部提交成功。"
                style={{ marginTop: 8 }}
              />
            ))}
        </SectionCard>
        <SectionCard title="成员货量与主单申报核对">
          {totals?.complete ? (
            <Descriptions
              size="small"
              column={{ xs: 1, sm: 3 }}
              items={[
                {
                  key: 'packages',
                  label: '成员订单申报件数合计',
                  children:
                    totals.packagesByUnit
                      ?.map((item) => `${item.packages} ${item.unit}`)
                      .join(' / ') || '-',
                },
                {
                  key: 'gross',
                  label: '成员订单申报毛重 KG',
                  children: formatCargoSummary(totals.grossWeightKg),
                },
                {
                  key: 'volume',
                  label: '成员订单申报体积 m³',
                  children: formatCargoSummary(totals.volumeCbm),
                },
                {
                  key: 'declaredPackages',
                  label: '主单申报件数',
                  children: input?.declaredPackages ?? '-',
                },
                {
                  key: 'declaredGross',
                  label: '主单申报毛重 KG / 差异',
                  children: `${formatCargoSummary(input?.declaredGrossWeightKg)} / ${input?.declaredGrossWeightKg === undefined ? '-' : formatCargoSummary(input.declaredGrossWeightKg - (totals.grossWeightKg || 0))}`,
                },
                {
                  key: 'declaredVolume',
                  label: '主单申报体积 m³ / 差异',
                  children: `${formatCargoSummary(input?.declaredVolumeCbm)} / ${input?.declaredVolumeCbm === undefined ? '-' : formatCargoSummary(input.declaredVolumeCbm - (totals.volumeCbm || 0))}`,
                },
              ]}
            />
          ) : (
            <Alert
              type="warning"
              title="无法完整核对批次"
              description="当前权限不足以查看完整成员及汇总；不会展示隐藏成员的数据。"
            />
          )}
          <Alert
            type="info"
            title="申报值和成员合计独立保留，差异仅提示"
            description="以上为订单摘要，下面的重量参考取实际货物明细；不会自动覆盖成员订单或分摊费用。"
            style={{ marginTop: 8 }}
          />
        </SectionCard>
        <SectionCard title="真实活动成员">
          <ResponsiveProTable<API.AirMasterMember>
            rowKey="orderId"
            actionRef={memberRef}
            search={false}
            options={false}
            size="small"
            pagination={{ defaultPageSize: 20 }}
            request={async ({ current: page, pageSize }) => {
              const response = await airExportServiceGetAirMasterWaybill({
                id: masterId,
                page: page || 1,
                pageSize: pageSize || 20,
              });
              if (!response?.data) throw new Error('成员列表加载失败');
              return {
                data: response.members || [],
                total: response.total || 0,
                success: true,
              };
            }}
            columns={[
              {
                title: '订单',
                dataIndex: 'orderNo',
                render: (_, item) => (
                  <a
                    onClick={() =>
                      history.push(`/orders/air-export/${item.orderId}`)
                    }
                  >
                    {item.orderNo}
                  </a>
                ),
              },
              { title: '客户', dataIndex: 'customerName', ellipsis: true },
              { title: '分单号', dataIndex: 'houseNo' },
              {
                title: '计划确认计费重 KG',
                render: (_, item) =>
                  `${trimDecimal(item.confirmedChargeableWeightKg)}${item.confirmationStale ? '（依据已变化）' : ''}`,
              },
              {
                title: '件数',
                render: (_, item) =>
                  `${item.packages ?? '-'} ${item.packageUnit || ''}`,
              },
              {
                title: '毛重 KG',
                render: (_, item) => trimDecimal(item.grossWeightKg),
              },
              {
                title: '体积 m³',
                render: (_, item) => trimDecimal(item.volumeCbm),
              },
              {
                title: '状态',
                render: (_, item) =>
                  `${getOrderFlowStatusText(item.flowStatus, 'AE')} · ${item.locked ? '已锁定' : '未锁定'}`,
              },
              {
                title: '分单单证',
                render: (_, item) =>
                  item.houseStatus === 'RELEASED'
                    ? '已签发'
                    : item.houseStatus === 'CONFIRMED'
                      ? '已确认'
                      : '待完善',
              },
            ]}
          />
        </SectionCard>
        <AirWeightPanel
          key={masterId}
          scope="MASTER"
          targetId={masterId}
          canUpdate={canUpdate && !current?.blockedReasons?.length}
          onConfirmed={() => {
            memberRef.current?.reload();
          }}
        />
        <SectionCard title="主单及关联历史">
          <ResponsiveProTable<API.AirMasterHistory>
            rowKey="id"
            search={false}
            options={false}
            size="small"
            pagination={{ defaultPageSize: 20 }}
            request={async ({ current: page, pageSize }) => {
              const response = await airExportServiceGetAirMasterWaybill({
                id: masterId,
                page: page || 1,
                pageSize: pageSize || 20,
              });
              if (!response?.data) throw new Error('主单历史加载失败');
              return {
                data: response.history || [],
                total: response.historyTotal || 0,
                success: true,
              };
            }}
            params={{ version: master?.version }}
            columns={[
              { title: '版本', dataIndex: 'version', width: 70 },
              { title: '操作', dataIndex: 'action' },
              { title: '原因', dataIndex: 'reason', ellipsis: true },
              {
                title: '时间',
                render: (_, item) => formatDate(item.occurredAt, 'datetime'),
                width: 180,
              },
            ]}
          />
        </SectionCard>
      </Space>
      {editOpen && master && (
        <GuardedModalForm<MasterEditValues>
          title="修改共享主单共同资料"
          open
          onOpenChange={setEditOpen}
          guardIdentity={masterId}
          width={1000}
          initialValues={{ input: master.content }}
          modalProps={{ destroyOnHidden: true }}
          onValuesChange={() => setImpact(undefined)}
          submitter={{
            searchConfig: {
              submitText: impact?.response.canExecute
                ? '确认影响并提交'
                : '预览影响',
            },
          }}
          onFinish={async (values) => {
            const payload = {
              id: masterId,
              expectedVersion: master.version,
              input: serializeAirMaster(values.input),
            };
            if (!impact) {
              const response =
                await airExportServicePreviewAirMasterWaybillUpdate(
                  { id: masterId },
                  payload,
                );
              if (!response?.inputFingerprint) return false;
              setImpact({ values, response });
              return false;
            }
            if (!impact.response.canExecute) return false;
            const response = await airExportServiceUpdateAirMasterWaybill(
              { id: masterId },
              {
                ...payload,
                inputFingerprint: impact.response.inputFingerprint,
                reason: values.reason,
              },
            );
            if (!isWriteResponse(response, true)) return false;
            setEditOpen(false);
            await reload();
            return true;
          }}
        >
          <AirMasterFields
            props={{
              searchLocations: (keyword) =>
                searchOrderLocations('air', keyword),
              locationOptions: [],
            }}
            prefix={['input']}
            master={master}
          />
          <ProFormTextArea
            name="reason"
            label="修改原因"
            rules={[
              { required: true, whitespace: true, message: '请输入修改原因' },
            ]}
            fieldProps={{ maxLength: 500 }}
          />
          {impact && (
            <Alert
              type={impact.response.canExecute ? 'info' : 'error'}
              showIcon
              title={
                impact.response.canExecute
                  ? `影响 ${impact.response.members?.length || 0} 票订单，确认后统一修改`
                  : '共享修改被阻止'
              }
              description={
                <>
                  <div>
                    {impact.response.members
                      ?.map((item) => item.orderNo)
                      .join('、')}
                  </div>
                  {impact.response.blockedReasons?.map((reason) => (
                    <div key={reason}>{reason}</div>
                  ))}
                </>
              }
            />
          )}
        </GuardedModalForm>
      )}
      {linkOpen && master && canRestore && (
        <GuardedModalForm<{
          orderId: string;
          expectedOrderVersion: string;
          houseNo: string;
          reason: string;
        }>
          title="明确恢复关联已终止订单"
          formRef={linkForm}
          open
          onOpenChange={setLinkOpen}
          guardIdentity={masterId}
          modalProps={{ destroyOnHidden: true }}
          onFinish={async (values) => {
            if (!canRestore) return false;
            const response = await airExportServiceLinkAirMasterWaybill(
              { id: masterId },
              {
                id: masterId,
                ...values,
                expectedMasterVersion: master.version,
              },
            );
            if (!isWriteResponse(response, true)) return false;
            setLinkOpen(false);
            await reload();
            return true;
          }}
        >
          <Alert
            type="info"
            title="恢复该票在本主单下的有效关联并保留历史"
            description="仅可选择原属本主单且已终止的订单；恢复时建立新关联并保留历史，不支持跨主单改配。服务端会复核所有受影响成员及版本。"
            style={{ marginBottom: 12 }}
          />
          <ProFormSearchableSelect
            name="orderId"
            label="选择空运订单"
            rules={[{ required: true, message: '请选择订单' }]}
            request={async ({ keyWords }) => {
              const response = ensureListResponse(
                await airExportServiceListAirLinkCandidates({
                  id: masterId,
                  keyword: keyWords,
                  page: 1,
                  pageSize: 50,
                }),
                '订单候选加载失败',
              );
              return (response.data || []).map((item) => ({
                value: item.orderId || '',
                label: `${item.orderNo} · ${item.customerName || ''}`,
                order: item,
              }));
            }}
            fieldProps={{
              onChange: (_value, option) => {
                const selected = Array.isArray(option) ? option[0] : option;
                if (selected && 'order' in selected) {
                  const order = selected.order as API.AirMasterMember;
                  linkForm.current?.setFieldsValue({
                    expectedOrderVersion: order.orderVersion,
                    houseNo: order.houseNo,
                  });
                }
              },
            }}
          />
          <ProFormText name="expectedOrderVersion" hidden />
          <ProFormText
            name="houseNo"
            label="目标主单内分单号"
            rules={[{ required: true, message: '请输入分单号' }]}
          />
          <ProFormTextArea
            name="reason"
            label="关联原因"
            rules={[{ required: true, message: '请输入关联原因' }]}
          />
        </GuardedModalForm>
      )}
    </PageContainer>
  );
}
