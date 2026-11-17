import { ReloadOutlined } from '@ant-design/icons';
import {
  PageContainer,
  ProForm,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Form, Space, Spin, Tag, Typography } from 'antd';
import {
  lazy,
  Suspense,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { useLocation } from 'react-router';
import { useAccess } from '@/app/access';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { useTabCloseGuard } from '@/components/layout/tabCloseGuard';
import { useRouteLeaveGuard } from '@/components/layout/useRouteLeaveGuard';
import {
  GuardedModalForm,
  PageHeaderShell,
  SectionCard,
  StickyFooterBar,
} from '@/components/ui';
import { history } from '@/router/history';
import {
  airExportServicePreviewAirMasterWaybillUpdate,
  airExportServiceUpdateAirMasterWaybill,
} from '@/services/roncin/airExportService';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import { isWriteResponse } from '@/utils/api';
import { sameAirWaybillContent } from '../order-kinds/air-export/air-save-baseline';
import { serializeAirWaybillContent } from '../order-kinds/air-export/form-adapter';
import type { OrderKindDefinition } from '../order-kinds/types';
import { AirWaybillFields } from '../templates/components/air/AirWaybillFields';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from '../use-order-lock-state';
import {
  type AwbBundle,
  type AwbFormValues,
  type AwbIdentity,
  acceptsAwbHouseSave,
  acceptsAwbMasterSave,
  awbContent,
  awbFingerprint,
  awbHouseSavePayload,
  awbOutputBlockedReason,
  buildAwbPdfModel,
  requireAwbBundle,
} from './model';
import type { AwbPdfModel } from './pdf/types';
import { fetchAwbBundle, useAwbChangeNotice } from './use-awb-data';

const AwbPdfPreview = lazy(() => import('./pdf/AwbPdfPreview'));
type Impact = {
  values: AwbFormValues;
  input: API.AirMasterWaybillInput;
  fingerprint: string;
  revision: number;
  members: API.AirMasterMember[];
};

export default function AwbEditor({
  initial,
  identity: sourceIdentity,
  definition,
}: {
  initial: AwbBundle;
  identity: AwbIdentity;
  definition: OrderKindDefinition;
}) {
  const identity =
    sourceIdentity.target === 'MAWB'
      ? { ...sourceIdentity, orderId: undefined, masterId: initial.master.id }
      : sourceIdentity;
  const { modal, message } = App.useApp();
  const access = useAccess();
  const client = useQueryClient();
  const location = useLocation();
  const [path] = useState(location.pathname);
  const [form] = Form.useForm<AwbFormValues>();
  const [baseline, setBaseline] = useState(() =>
    structuredClone(
      identity.target === 'MAWB'
        ? { master: initial.master, blockedReasons: initial.blockedReasons }
        : initial,
    ),
  );
  const [session, setSession] = useState(0);
  const [syncError, setSyncError] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [checking, setChecking] = useState(false);
  const [impact, setImpact] = useState<Impact>();
  const [preview, setPreview] = useState<AwbPdfModel>();
  const revision = useRef(0);
  const alive = useRef(true);
  const confirmation = useRef<ReturnType<typeof modal.confirm> | undefined>(
    undefined,
  );
  useLayoutEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
      revision.current += 1;
      confirmation.current?.destroy();
    };
  }, []);
  const { target } = identity;
  const house = baseline.order?.airExport?.house;
  const lock = useOrderLockState(
    target === 'HAWB' ? identity.orderId : undefined,
  );
  const policy = getOrderBusinessWritePolicy({
    state: lock.state,
    loading: lock.loading,
    error: lock.error,
    order: baseline.order,
    canOperate: access.canOperateOrganization(identity.organizationId),
  });
  const canUpdate =
    access.canOperateOrganization(identity.organizationId) &&
    access.canOrder(definition.businessType, 'update') &&
    (target === 'MAWB'
      ? baseline.blockedReasons.length === 0
      : !policy.disabled && baseline.order?.airExport?.active === true);
  const reasonRequired = target === 'MAWB' || house?.status === 'RELEASED';
  const readContent = () =>
    serializeAirWaybillContent(
      form.getFieldValue('content') as API.AirWaybillContent | undefined,
    ) ?? {};
  const hasChanges = () =>
    !sameAirWaybillContent(readContent(), awbContent(baseline, target));
  const tabKey = resolveTabKey(location.pathname);
  useRouteLeaveGuard(tabKey, hasChanges);
  useTabCloseGuard({ tabKey, isDirty: hasChanges });
  const accept = (bundle: AwbBundle) => {
    setBaseline(structuredClone(bundle));
    form.resetFields();
    form.setFieldsValue({
      content: awbContent(bundle, target),
      reason: undefined,
    });
    revision.current += 1;
    setSession((value) => value + 1);
    setSyncError(false);
    setPreview(undefined);
    setImpact(undefined);
  };
  const save = useMutation({
    mutationFn: async (request: {
      values: AwbFormValues;
      impact?: Impact;
    }): Promise<AwbBundle> => {
      if (target === 'HAWB') {
        const payload = awbHouseSavePayload(baseline, request.values);
        const response = await orderServiceUpdateOrder(
          { id: identity.orderId as string },
          payload,
        );
        if (!isWriteResponse(response, true) || !response.data)
          throw new Error('正文保存失败，当前输入已保留。');
        if (!acceptsAwbHouseSave(baseline, payload, response.data))
          throw new Error(
            '保存后资料已发生后续变动，请刷新核对；当前输入已保留。',
          );
        return requireAwbBundle(
          {
            order: response.data,
            master: response.data.airExport?.master as API.AirMasterWaybill,
            blockedReasons: [],
          },
          identity,
        );
      }
      if (!request.impact) throw new Error('请先确认共享影响。');
      const response = await airExportServiceUpdateAirMasterWaybill(
        { id: baseline.master.id as string },
        {
          id: baseline.master.id,
          expectedVersion: baseline.master.version,
          input: request.impact.input,
          inputFingerprint: request.impact.fingerprint,
          reason: request.values.reason,
        },
      );
      if (
        !isWriteResponse(response, true) ||
        !acceptsAwbMasterSave(baseline, request.values, response.data)
      )
        throw new Error(
          '保存失败或保存后资料已发生后续变动，请刷新核对；当前输入已保留。',
        );
      // 主单独立制作无需重读订单，不把更晚的订单/分单版本借给旧输入。
      return requireAwbBundle(
        {
          master: response.data as API.AirMasterWaybill,
          blockedReasons: response.blockedReasons ?? [],
        },
        { ...identity, orderId: undefined, masterId: baseline.master.id },
      );
    },
    meta: { silent: true },
  });
  const previewImpact = useMutation({
    mutationFn: async (values: AwbFormValues) => {
      const input = {
        ...structuredClone(baseline.master.content),
        waybillContent: values.content,
      };
      const response = await airExportServicePreviewAirMasterWaybillUpdate(
        { id: baseline.master.id as string },
        {
          id: baseline.master.id,
          expectedVersion: baseline.master.version,
          input,
        },
      );
      if (response?.success === false || !response?.inputFingerprint)
        throw new Error('共享修改预览失败，当前输入已保留。');
      if (response.canExecute !== true)
        throw new Error(
          response.blockedReasons?.join('；') || '共享成员当前不允许修改。',
        );
      return { input, response };
    },
    meta: { silent: true },
  });
  const busy =
    save.isPending || previewImpact.isPending || refreshing || checking;
  const notice = useAwbChangeNotice(
    baseline,
    identity,
    session,
    !busy && location.pathname === path,
  );
  useEffect(() => {
    if (notice.changed || notice.checkError) setPreview(undefined);
  }, [notice.changed, notice.checkError]);
  const executeSave = async (values: AwbFormValues, confirmed?: Impact) => {
    if (!canUpdate || notice.changed || syncError) return false;
    const currentRevision = revision.current;
    try {
      const result = await save.mutateAsync({
        values: structuredClone(values),
        impact: confirmed,
      });
      if (!alive.current) return false;
      if (revision.current !== currentRevision) {
        setSyncError(true);
        setImpact(undefined);
        message.warning(
          '保存期间输入已继续修改，当前草稿已保留，请刷新核对后再保存。',
        );
        return false;
      }
      accept(result);
      void client.invalidateQueries({ queryKey: ['orders'] });
      void client.invalidateQueries({ queryKey: ['air-orders', 'master'] });
      message.success('航空运单正文已保存');
      return true;
    } catch (error) {
      if (alive.current) {
        setSyncError(true);
        setImpact(undefined);
        message.error(
          error instanceof Error
            ? error.message
            : '正文保存失败，当前输入已保留。',
        );
      }
      return false;
    }
  };
  const submit = async (raw: AwbFormValues) => {
    if (!canUpdate || busy || notice.changed || syncError) return false;
    const values = structuredClone({
      ...raw,
      content: serializeAirWaybillContent(raw.content) ?? {},
    });
    if (target === 'HAWB') return executeSave(values);
    const currentRevision = revision.current;
    try {
      const result = await previewImpact.mutateAsync(values);
      if (!alive.current || revision.current !== currentRevision) return false;
      setImpact({
        values,
        input: result.input,
        fingerprint: result.response.inputFingerprint as string,
        revision: currentRevision,
        members: result.response.members ?? [],
      });
    } catch (error) {
      if (alive.current)
        message.error(
          error instanceof Error ? error.message : '共享修改预览失败。',
        );
    }
    return false;
  };
  const refresh = async () => {
    if (busy) return;
    const currentRevision = revision.current;
    setRefreshing(true);
    try {
      const result = await client.fetchQuery({
        queryKey: [
          'air-orders',
          'awb-refresh',
          { identity, session, revision: currentRevision },
        ],
        queryFn: ({ signal }) => fetchAwbBundle(identity, signal),
        staleTime: 0,
        meta: { silent: true },
      });
      if (!alive.current || currentRevision !== revision.current) return;
      accept(result);
      if (target === 'HAWB') await lock.refresh();
    } catch (error) {
      if (alive.current)
        message.error(
          error instanceof Error ? error.message : '刷新失败，当前输入已保留。',
        );
    } finally {
      if (alive.current) setRefreshing(false);
    }
  };
  const requestRefresh = () => {
    if (busy) return;
    if (hasChanges())
      confirmation.current = modal.confirm({
        title: '刷新会覆盖尚未保存的航空运单正文，是否继续？',
        okText: '确认刷新',
        cancelText: '保留编辑',
        onOk: refresh,
      });
    else void refresh();
  };
  const output = async () => {
    if (busy || syncError || notice.changed) return;
    if (hasChanges()) {
      message.warning('请先保存正文后再输出。');
      return;
    }
    const currentRevision = revision.current;
    setChecking(true);
    try {
      if (
        !(await notice.verifyCurrent()) ||
        !alive.current ||
        currentRevision !== revision.current ||
        hasChanges()
      )
        return;
      setPreview(buildAwbPdfModel(baseline, target));
    } catch (error) {
      if (alive.current)
        message.error(
          error instanceof Error ? error.message : '输出检查失败。',
        );
    } finally {
      if (alive.current) setChecking(false);
    }
  };
  const blockOutput = awbOutputBlockedReason(baseline, target);
  const listPath = `/orders/${definition.kind}`;
  const self = target === 'HAWB' && baseline.master.businessType === 'AE';
  return (
    <PageContainer title={false} breadcrumbRender={false}>
      <PageHeaderShell
        title={`${target} 制作${self ? '' : '归档'}`}
        subTitle={
          target === 'HAWB' ? house?.houseNo : baseline.master.content?.masterNo
        }
        onBack={() => history.push(listPath)}
        breadcrumbs={[{ label: definition.navigationTitle, href: listPath }]}
        tags={<Tag color="blue">{target}</Tag>}
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={requestRefresh}
            loading={refreshing}
          >
            刷新
          </Button>
        }
      />
      <Space
        orientation="vertical"
        size={12}
        style={{ width: '100%', marginTop: 12 }}
      >
        <Alert
          type="info"
          showIcon
          title={self ? '本公司航空分运单' : '外部航空运单资料归档'}
          description={
            self
              ? `抬头使用实际经营公司英文名称：${baseline.master.organizationNameEn || '尚未维护'}。保存正文、登记签发事实与打印分别进行。`
              : '按已保存的真实印发人输出归档文件；打印不会签发、放货、加电子签章或外发。'
          }
        />
        {notice.changed && (
          <Alert
            type="warning"
            showIcon
            title="这个航空运单已经发生变动，请重新刷新。"
            description="当前输入与原提交基线已保留，旧预览已失效。"
          />
        )}
        {notice.checkError && (
          <Alert
            type="warning"
            title="航空运单变动检查失败，当前输入已保留；输出前将再次检查。"
          />
        )}
        {syncError && (
          <Alert
            type="warning"
            showIcon
            title="请刷新后核对保存结果，当前输入已保留。"
          />
        )}
        {blockOutput && <Alert type="warning" title={blockOutput} />}
        {!canUpdate && (
          <Alert
            type="info"
            title={
              target === 'HAWB'
                ? policy.reason || '当前仅可查看和输出已保存正文。'
                : baseline.blockedReasons.join('；') ||
                  '当前仅可查看和输出已保存正文。'
            }
          />
        )}
        <SectionCard title="航空运单印字正文">
          <ProForm<AwbFormValues>
            form={form}
            initialValues={{ content: awbContent(initial, target) }}
            submitter={false}
            onFinish={submit}
            onValuesChange={() => {
              revision.current += 1;
              setImpact(undefined);
              setPreview(undefined);
            }}
          >
            <AirWaybillFields prefix={['content']} disabled={!canUpdate} />
            {canUpdate && (
              <ProFormTextArea
                name="reason"
                label="修改原因"
                rules={[
                  {
                    required: reasonRequired,
                    whitespace: reasonRequired,
                    message: '请输入修改原因',
                  },
                ]}
                fieldProps={{ maxLength: 500 }}
              />
            )}
          </ProForm>
        </SectionCard>
      </Space>
      <StickyFooterBar
        info={
          <Typography.Text>
            {target === 'HAWB'
              ? house?.houseNo
              : baseline.master.content?.masterNo}{' '}
            · {self ? '本公司制作' : '资料归档'}
          </Typography.Text>
        }
      >
        <Button onClick={() => history.push(listPath)}>返回列表</Button>
        <Button
          onClick={output}
          loading={checking}
          disabled={busy || syncError || notice.changed || Boolean(blockOutput)}
        >
          预览 / 打印 / 下载
        </Button>
        {canUpdate && (
          <Button
            type="primary"
            loading={save.isPending || previewImpact.isPending}
            disabled={busy || notice.changed || syncError}
            onClick={() => form.submit()}
          >
            保存正文
          </Button>
        )}
      </StickyFooterBar>
      {impact && (
        <GuardedModalForm
          key={impact.fingerprint}
          title="确认主单正文共享修改"
          open
          modalProps={{ destroyOnHidden: true }}
          onOpenChange={(open) => {
            if (!open && !save.isPending) setImpact(undefined);
          }}
          submitter={{ searchConfig: { submitText: '确认影响并保存' } }}
          onFinish={async () => {
            if (revision.current !== impact.revision) return false;
            return executeSave(impact.values, impact);
          }}
        >
          <Alert
            type="info"
            title={`影响 ${impact.members.length} 票订单`}
            description="仅更新本次主单正文；共享运输输入保持打开时快照，服务端将再次复验全部成员和预览指纹。"
          />
          <Typography.Paragraph>
            {impact.members.map((member) => member.orderNo).join('、') ||
              '当前无活动成员'}
          </Typography.Paragraph>
        </GuardedModalForm>
      )}
      {preview && !notice.changed && (
        <Suspense fallback={<Spin />}>
          <AwbPdfPreview
            open
            onClose={() => setPreview(undefined)}
            model={preview}
            fingerprint={awbFingerprint(baseline)}
          />
        </Suspense>
      )}
    </PageContainer>
  );
}
