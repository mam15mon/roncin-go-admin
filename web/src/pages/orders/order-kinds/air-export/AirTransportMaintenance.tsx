import {
  type ProFormInstance,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Space, Spin } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router';
import { GuardedModalForm } from '@/components/ui';
import {
  airExportServiceGetAirMasterWaybill,
  airExportServicePreviewAirMasterWaybillUpdate,
  airExportServiceUpdateAirMasterWaybill,
} from '@/services/roncin/airExportService';
import { isWriteResponse } from '@/utils/api';
import { generateUUID } from '@/utils/uuid';
import { useAirMasterChangeNotice } from '../../air-master-change-notice';
import { AirMasterFields } from '../../templates/components/air/AirMasterFields';
import type { OrderDetailFeaturesContext } from '../types';
import { serializeAirMaster } from './form-adapter';
import { requireAirMasterIdentity } from './identity';

type TransportValues = { input: API.AirMasterWaybillInput; reason: string };
type EditorProps = { context: OrderDetailFeaturesContext; onClose: () => void };

/** 单票中办理共享运输修改；只负责此独立命令，不重置订单主表。 */
export default function AirTransportMaintenance({
  context,
}: {
  context: OrderDetailFeaturesContext;
}) {
  const [session, setSession] = useState(0);
  const [open, setOpen] = useState(false);
  if (!context.order.airExport?.master?.id || !context.canOrder('update'))
    return null;
  return (
    <>
      <Button
        disabled={context.businessWritesDisabled}
        onClick={() => {
          if (!context.ensureBusinessWriteAllowed()) return;
          setSession((value) => value + 1);
          setOpen(true);
        }}
      >
        修改共同运输资料
      </Button>
      {open && (
        <AirTransportLoader
          key={`${context.orderFormIdentity}:${session}`}
          context={context}
          onClose={() => setOpen(false)}
        />
      )}
    </>
  );
}

function AirTransportLoader({ context, onClose }: EditorProps) {
  const owner = useMemo(() => generateUUID(), []);
  const masterId = context.order.airExport?.master?.id || '';
  const organizationId = context.order.organizationId;
  const businessType = context.order.businessType;
  const load = useQuery({
    queryKey: [
      'air-orders',
      'transport-editor',
      { masterId, organizationId, businessType, owner },
    ],
    enabled: Boolean(masterId && organizationId && context.canOrder('read')),
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response = await airExportServiceGetAirMasterWaybill(
        { id: masterId, page: 1, pageSize: 1 },
        { signal, skipErrorHandler: true },
      );
      if (!response?.data || response.success === false)
        throw new Error('共同运输资料加载失败');
      requireAirMasterIdentity(
        response.data,
        masterId,
        organizationId,
        businessType,
      );
      if (!response.data.content) throw new Error('主单缺少共同运输资料');
      return response;
    },
  });
  if (!load.data)
    return (
      <GuardedModalForm
        open
        title="修改共同运输资料"
        onOpenChange={(value) => {
          if (!value) onClose();
        }}
        submitter={false}
      >
        {load.isError ? (
          <Alert
            type="error"
            title="共同运输资料加载失败"
            action={<Button onClick={() => void load.refetch()}>重试</Button>}
          />
        ) : (
          <Spin description="正在读取共同运输资料">
            <div style={{ minHeight: 100 }} />
          </Spin>
        )}
      </GuardedModalForm>
    );
  return (
    <AirTransportEditor
      context={context}
      onClose={onClose}
      initialResponse={load.data}
      reload={async () => {
        const result = await load.refetch();
        return result.isSuccess ? result.data : undefined;
      }}
    />
  );
}

function AirTransportEditor({
  context,
  onClose,
  initialResponse,
  reload,
}: EditorProps & {
  initialResponse: API.GetAirMasterWaybillResponse;
  reload: () => Promise<API.GetAirMasterWaybillResponse | undefined>;
}) {
  const { modal, message } = App.useApp();
  const queryClient = useQueryClient();
  const { pathname } = useLocation();
  const [baseline, setBaseline] = useState(() =>
    structuredClone(initialResponse),
  );
  const [session, setSession] = useState(1);
  const [dirty, setDirty] = useState(false);
  const [refreshFailed, setRefreshFailed] = useState(false);
  const [impact, setImpact] =
    useState<API.PreviewAirMasterWaybillUpdateResponse>();
  const form = useRef<ProFormInstance<TransportValues> | undefined>(undefined);
  const revision = useRef(0);
  const alive = useRef(true);
  const activeContext = useRef(context);
  activeContext.current = context;
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  const master = baseline.data;
  const masterId = master?.id || '';
  const changed = useAirMasterChangeNotice({
    baseline,
    session,
    organizationId: context.order.organizationId,
    businessType: context.order.businessType,
    enabled: pathname === `/orders/${context.kind}/${context.orderId}`,
  });
  const allowed = () => {
    const current = activeContext.current;
    return (
      alive.current &&
      current.orderFormIdentity === context.orderFormIdentity &&
      current.order.organizationId === context.order.organizationId &&
      current.order.airExport?.master?.id === masterId &&
      current.canOrder('update') &&
      !current.businessWritesDisabled
    );
  };
  const refresh = async () => {
    const submittedRevision = revision.current;
    const next = await reload();
    if (!alive.current || submittedRevision !== revision.current) return;
    if (!next?.data) {
      setRefreshFailed(true);
      return;
    }
    setBaseline(structuredClone(next));
    setSession((value) => value + 1);
    setDirty(false);
    setImpact(undefined);
    setRefreshFailed(false);
  };
  const requestRefresh = () => {
    if (!dirty) {
      void refresh();
      return;
    }
    const confirmation = modal.confirm({
      title: '刷新会覆盖尚未保存的运输修改，是否继续？',
      okText: '确认刷新',
      cancelText: '保留编辑',
      onOk: refresh,
    });
    void confirmation.then(
      () => {},
      () => {},
    );
  };
  return (
    <GuardedModalForm<TransportValues>
      key={session}
      title="修改共同运输资料"
      open
      width={1000}
      formRef={form}
      guardIdentity={`${context.orderFormIdentity}:transport:${session}`}
      modalProps={{ destroyOnHidden: true }}
      initialValues={{ input: structuredClone(master?.content) }}
      onOpenChange={(value) => {
        if (!value) {
          alive.current = false;
          onClose();
        }
      }}
      onValuesChange={() => {
        revision.current += 1;
        setDirty(true);
        setImpact(undefined);
      }}
      submitter={{
        searchConfig: {
          submitText: impact?.canExecute ? '确认影响并提交' : '预览影响',
        },
        submitButtonProps: {
          disabled:
            changed ||
            refreshFailed ||
            context.businessWritesDisabled ||
            !context.canOrder('update'),
        },
      }}
      onFinish={async (values) => {
        if (!master || !allowed() || changed || refreshFailed) return false;
        const submittedRevision = revision.current;
        const current = () =>
          allowed() && submittedRevision === revision.current;
        // 运输窗口没有注册正文控件，明确从打开快照保留完整正文，避免空值或字段裁剪覆盖。
        const input = serializeAirMaster({
          ...master.content,
          ...values.input,
          waybillContent: master.content?.waybillContent,
        });
        const payload = {
          id: masterId,
          expectedVersion: master.version,
          input,
        };
        if (!impact) {
          const response = await airExportServicePreviewAirMasterWaybillUpdate(
            { id: masterId },
            payload,
          );
          if (
            !current() ||
            !response?.inputFingerprint ||
            response.success === false
          )
            return false;
          setImpact(response);
          return false;
        }
        if (impact.canExecute !== true) return false;
        const response = await airExportServiceUpdateAirMasterWaybill(
          { id: masterId },
          {
            ...payload,
            inputFingerprint: impact.inputFingerprint,
            reason: values.reason,
          },
        );
        if (!isWriteResponse(response, true) || !current()) return false;
        requireAirMasterIdentity(
          response.data,
          masterId,
          context.order.organizationId,
          context.order.businessType,
        );
        try {
          await Promise.all([
            context.refreshOrderAndLock(),
            queryClient.invalidateQueries({ queryKey: ['air-orders'] }),
            queryClient.invalidateQueries({ queryKey: ['air-weight'] }),
          ]);
        } catch {
          if (current())
            message.warning(
              '共同运输资料已保存，订单摘要读取失败，请刷新。未保存的订单输入已保留。',
            );
        }
        if (!current()) return false;
        message.success(
          '共同运输资料已保存。订单中未保存的输入已保留，请刷新核对最新资料。',
        );
        return true;
      }}
    >
      <Space
        orientation="vertical"
        size={12}
        style={{ width: '100%', marginBottom: 12 }}
      >
        <Alert
          type="info"
          showIcon
          title="本次仅保存共同运输资料"
          description="会影响该主单的关联订单；订单内其他未保存输入不会一并提交。运单印字正文保持独立。"
        />
        {(changed || refreshFailed) && (
          <Alert
            type="warning"
            showIcon
            title={
              changed
                ? '这个主单已经发生变动，请重新刷新。'
                : '运输资料刷新失败，当前输入已保留。'
            }
            action={<Button onClick={requestRefresh}>刷新运输资料</Button>}
          />
        )}
      </Space>
      <AirMasterFields
        props={{
          searchLocations: context.searchLocations,
          locationOptions: [],
          readonly: context.businessWritesDisabled,
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
      <Button onClick={requestRefresh} style={{ marginBottom: 12 }}>
        重新读取运输资料
      </Button>
      {impact && (
        <Alert
          type={impact.canExecute ? 'info' : 'error'}
          showIcon
          title={
            impact.canExecute
              ? `影响 ${impact.members?.length || 0} 票订单，确认后统一修改`
              : '共享修改被阻止'
          }
          description={
            <>
              <div>
                {impact.members?.map((item) => item.orderNo).join('、')}
              </div>
              {impact.blockedReasons?.map((reason) => (
                <div key={reason}>{reason}</div>
              ))}
            </>
          }
        />
      )}
    </GuardedModalForm>
  );
}
