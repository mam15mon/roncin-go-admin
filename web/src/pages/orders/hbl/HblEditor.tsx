import { EyeOutlined, ReloadOutlined, SaveOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Segmented,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  lazy,
  Suspense,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { useAccess } from '@/app/access';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { useTabCloseGuard } from '@/components/layout/tabCloseGuard';
import { useRouteLeaveGuard } from '@/components/layout/useRouteLeaveGuard';
import {
  OrderFormTemplate,
  PageHeaderShell,
  StickyFooterBar,
} from '@/components/ui';
import type { OrderFormTemplateActions } from '@/components/ui/order-template/types';
import {
  MasterDataKind,
  OrderBusinessType,
  SeaDocumentAction,
  SeaHouseBillStatus,
  ShipmentType,
} from '@/enums.generated';
import { getMasterDataOptions } from '@/features/orders/options';
import { history } from '@/router/history';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { seaDocumentServiceUpdateSeaHouseBill } from '@/services/roncin/seaDocumentService';
import { seaSharedContainerServiceListSeaSharedContainers } from '@/services/roncin/seaSharedContainerService';
import { isWriteResponse } from '@/utils/api';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from '../use-order-lock-state';
import { buildHblFormSections } from './HblFormSections';
import {
  buildHblFormValues,
  formatHblContainers,
  type HblBundle,
  type HblFormValues,
  hblFormPayload,
  hblUnavailableReason,
  hblValuesFingerprint,
} from './model';
import {
  HBL_PDF_VARIANTS,
  type HblPdfModel,
  type HblPdfVariant,
} from './pdf/types';
import { fetchHblBundle, useHblChangeNotice } from './use-hbl-data';

const HblPdfPreview = lazy(() => import('./pdf'));

export default function HblEditor({
  initial,
  scope,
}: {
  initial: HblBundle;
  scope: string;
}) {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const client = useQueryClient();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const actionsRef = useRef<
    OrderFormTemplateActions<HblFormValues> | undefined
  >(undefined);
  const [baseline, setBaseline] = useState(initial);
  const [initialValues] = useState(() => buildHblFormValues(initial));
  const [baselineValues, setBaselineValues] = useState(initialValues);
  const [variant, setVariant] = useState<HblPdfVariant>('draft');
  const [preview, setPreview] = useState<HblPdfModel>();
  const [refreshing, setRefreshing] = useState(false);
  const [syncError, setSyncError] = useState(false);
  const [checkingPreview, setCheckingPreview] = useState(false);
  const alive = useRef(true);
  const confirmation = useRef<ReturnType<typeof modal.confirm> | undefined>(
    undefined,
  );
  useLayoutEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
      confirmation.current?.destroy();
    };
  }, []);

  const { order, documents } = baseline;
  const house = documents.houseBill;
  const orderId = order.id as string;
  const path = `/orders/sea-export/${orderId}/hbl`;
  const tabKey = resolveTabKey(path);
  const lock = useOrderLockState(orderId);
  const policy = getOrderBusinessWritePolicy({
    state: lock.state,
    loading: lock.loading,
    error: lock.error,
    order,
    canOperate: access.canOperateOrganization(order.organizationId),
  });
  const canUpdate =
    access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'update') &&
    documents.allowedActions?.includes(
      SeaDocumentAction.SEA_DOCUMENT_ACTION_UPDATE_HOUSE_BILL,
    ) === true &&
    house?.status !== SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_RELEASED;
  const saveMutation = useMutation({
    mutationFn: (input: API.UpdateSeaHouseBillRequest) =>
      seaDocumentServiceUpdateSeaHouseBill(
        { orderId: input.orderId, id: input.id },
        input,
      ),
  });
  const busy = saveMutation.isPending || refreshing || checkingPreview;
  const notice = useHblChangeNotice(baseline, scope, busy);
  const disabled =
    !canUpdate ||
    policy.disabled ||
    busy ||
    syncError ||
    Boolean(hblUnavailableReason(baseline));
  const readValues = () =>
    formRef.current?.getFieldsValue(true) as HblFormValues | undefined;
  const hasChanges = () => {
    const values = readValues();
    return values
      ? hblValuesFingerprint(values) !== hblValuesFingerprint(baselineValues)
      : false;
  };
  useRouteLeaveGuard(tabKey, hasChanges);
  // 模板只在输入事件标脏；显式带入也是编辑，此守卫按完整表单快照保护工作区切换。
  useTabCloseGuard({ tabKey, isDirty: hasChanges });

  const acceptBundle = (bundle: HblBundle) => {
    if (!alive.current) return;
    const values = buildHblFormValues(bundle);
    setBaseline(bundle);
    setBaselineValues(values);
    setSyncError(false);
    setPreview(undefined);
    actionsRef.current?.resetTo(values);
  };

  const refresh = async () => {
    if (busy) return;
    setRefreshing(true);
    try {
      const bundle = await client.fetchQuery({
        queryKey: ['orders', 'hbl-production', { id: orderId, scope }],
        queryFn: ({ signal }) => fetchHblBundle(orderId, signal),
        staleTime: 0,
      });
      acceptBundle(bundle);
      if (alive.current) await lock.refresh();
    } catch {
      if (alive.current)
        message.error('读取最新资料失败，当前输入已保留，请重试。');
    } finally {
      if (alive.current) setRefreshing(false);
    }
  };

  const confirmReplace = (
    content: string,
    action: () => void | Promise<void>,
  ) => {
    confirmation.current = modal.confirm({
      title: '确认替换当前内容？',
      content,
      okText: '确定替换',
      cancelText: '保留当前内容',
      onOk: action,
    });
  };
  const requestRefresh = () => {
    if (hasChanges())
      confirmReplace('刷新会覆盖未保存的制作修改，是否继续？', refresh);
    else void refresh();
  };

  const save = async (): Promise<boolean> => {
    const values = readValues();
    if (!values || disabled || !house?.id) return false;
    if (
      !values.source?.houseVersion ||
      !values.source?.linkVersion ||
      values.source.houseId !== house.id
    ) {
      message.error(
        '制作草稿与当前 HBL 不一致，输入已保留，请刷新资料后重试。',
      );
      return false;
    }
    setRefreshing(true);
    try {
      const response = await saveMutation.mutateAsync({
        orderId,
        id: house.id,
        expectedVersion: values.source.houseVersion,
        expectedLinkVersion: values.source.linkVersion,
        houseBill: {
          id: house.id,
          houseNo: house.houseNo ?? '',
          issuerSource: house.issuerSource ?? 0,
          note: house.note,
          ...hblFormPayload(values),
        },
      });
      if (!alive.current || !isWriteResponse(response, true)) return false;
      const saved = response.data as API.SeaHouseBill;
      const savedBundle = {
        order,
        documents: { ...documents, houseBill: saved },
      };
      acceptBundle(savedBundle);
      message.success('HBL 制作资料已保存');
      // 写入已成功，重读失败仅冻结后续保存，不能把已落库结果报告成失败。
      try {
        // 写入前启动的后台读取不能成为本次保存后的同步快照。
        await client.cancelQueries({
          queryKey: ['orders', 'hbl-production', { id: orderId, scope }],
          exact: true,
        });
        if (!alive.current) return true;
        const next = await client.fetchQuery({
          queryKey: ['orders', 'hbl-production', { id: orderId, scope }],
          queryFn: ({ signal }) => fetchHblBundle(orderId, signal),
          staleTime: 0,
        });
        acceptBundle(next);
      } catch {
        if (alive.current) setSyncError(true);
      }
      if (alive.current) {
        void client.invalidateQueries({ queryKey: ['order-detail'] });
        void client.invalidateQueries({ queryKey: ['orders'] });
      }
      return true;
    } catch {
      return false;
    } finally {
      if (alive.current) setRefreshing(false);
    }
  };

  const containers = useQuery({
    queryKey: [
      'orders',
      'hbl-container-print',
      { orderId, scope, execution: order.seaMasterBill?.transportExecutionId },
    ],
    enabled: false,
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const execution = order.seaMasterBill?.transportExecutionId;
      const [exclusive, shared, specs] = await Promise.all([
        order.shipmentType === ShipmentType.SHIPMENT_TYPE_FCL
          ? orderContainerServiceListContainers(
              { orderId },
              { signal, skipErrorHandler: true },
            )
          : Promise.resolve({ success: true, data: [] }),
        execution
          ? seaSharedContainerServiceListSeaSharedContainers(
              {
                orderId,
                transportExecutionId: execution,
                page: 1,
                pageSize: 200,
              },
              { signal, skipErrorHandler: true },
            )
          : Promise.resolve({ success: true, data: [], total: 0 }),
        getMasterDataOptions(order.organizationId as string),
      ]);
      const valid = (
        response: { success?: boolean; data?: unknown[] } | undefined,
      ) =>
        Boolean(
          response &&
            response.success !== false &&
            (Array.isArray(response.data) ||
              (response.success === true && response.data === undefined)),
        );
      if (!valid(exclusive) || !valid(shared))
        throw new Error('实际箱资料读取失败，请重试。');
      if ((shared?.total ?? 0) > 200)
        throw new Error(
          '当前航次共享箱超过可带入范围，请核对装箱资料后手工填写本票箱号。',
        );
      return formatHblContainers(
        orderId,
        exclusive?.data ?? [],
        shared?.data ?? [],
        specs.filter(
          (item) =>
            item.kind === MasterDataKind.MASTER_DATA_KIND_CONTAINER_SPEC,
        ),
      );
    },
  });

  const importContainers = async () => {
    if (
      disabled ||
      !access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'container.read')
    )
      return;
    const result = await containers.refetch();
    if (!alive.current) return;
    if (result.error || result.data === undefined) {
      message.error(
        result.error instanceof Error
          ? result.error.message
          : '实际箱资料读取失败，请重试。',
      );
      return;
    }
    const text = result.data;
    if (!text) {
      message.info('当前订单尚无已维护的实际箱或本票共享箱分配。');
      return;
    }
    const apply = () => {
      actionsRef.current?.applyValues({
        printDetails: { containersText: text },
      });
    };
    if (formRef.current?.getFieldValue(['printDetails', 'containersText']))
      confirmReplace('带入实际箱会替换当前箱号／封号／箱型文字。', apply);
    else apply();
  };

  const importTransport = () => {
    if (disabled) return;
    confirmReplace(
      '将使用当前已保存的订单运输资料替换对应印字，不修改共同航程。',
      () => {
        const details = buildHblFormValues({
          ...baseline,
          documents: {
            ...documents,
            houseBill: { ...house, printDetails: undefined },
          },
        }).printDetails;
        actionsRef.current?.applyValues({
          printDetails: {
            bookingNoText: details.bookingNoText,
            vesselVoyageText: details.vesselVoyageText,
            portOfLoadingText: details.portOfLoadingText,
            portOfDischargeText: details.portOfDischargeText,
            placeOfDeliveryText: details.placeOfDeliveryText,
          },
        });
      },
    );
  };

  const requestPreview = async () => {
    if (hasChanges()) {
      message.warning('请先保存制作修改，再预览、打印或下载。');
      return;
    }
    setCheckingPreview(true);
    try {
      if (!(await notice.verifyCurrent()) || !alive.current) return;
      setPreview({
        houseNo: house?.houseNo ?? '',
        ...hblFormPayload(baselineValues),
      });
    } catch (error) {
      if (alive.current)
        message.error(
          error instanceof Error ? error.message : '预览资料检查失败',
        );
    } finally {
      if (alive.current) setCheckingPreview(false);
    }
  };

  const unavailable = hblUnavailableReason(baseline);
  useEffect(() => {
    window.dispatchEvent(
      new CustomEvent('roncin:update-tab-title', {
        detail: { path, title: `HBL ${house?.houseNo ?? ''}` },
      }),
    );
  }, [path, house?.houseNo]);

  const previewButton = (
    <Button
      icon={<EyeOutlined />}
      loading={checkingPreview}
      disabled={busy || Boolean(unavailable)}
      onClick={() => void requestPreview()}
    >
      预览／打印／下载
    </Button>
  );
  const saveButton = (
    <Tooltip
      title={!canUpdate ? '当前账号或单证状态不允许修改' : policy.reason}
    >
      <span>
        <Button
          type="primary"
          icon={<SaveOutlined />}
          loading={saveMutation.isPending}
          disabled={disabled}
          onClick={() => formRef.current?.submit()}
        >
          保存制作资料
        </Button>
      </span>
    </Tooltip>
  );

  return (
    <>
      <OrderFormTemplate<HblFormValues>
        formRef={formRef}
        actionsRef={actionsRef}
        initialValues={initialValues}
        readonly={!canUpdate || policy.disabled}
        editingDisabled={disabled}
        submitter={false}
        onFinish={save}
        tabKey={tabKey}
        draftPathname={path}
        draftScope={scope}
        showAnchorNav={false}
        header={
          <PageHeaderShell
            title="HBL 制作"
            subTitle={order.orderNo}
            onBack={() => history.push('/orders/sea-export')}
            breadcrumbs={[
              { label: '海运出口', href: '/orders/sea-export' },
              {
                label: order.orderNo || orderId,
                href: `/orders/sea-export/${orderId}`,
              },
            ]}
            tags={<Tag color="blue">{house?.houseNo}</Tag>}
            extra={
              <Space wrap>
                <Button
                  icon={<ReloadOutlined />}
                  loading={refreshing}
                  disabled={busy}
                  onClick={requestRefresh}
                >
                  刷新资料
                </Button>
                {previewButton}
                {saveButton}
              </Space>
            }
          />
        }
        prependSections={[
          {
            key: 'output',
            title: '输出版本',
            content: (
              <div style={{ width: '100%', display: 'grid', gap: 12 }}>
                <Segmented
                  aria-label="HBL 输出版本"
                  options={HBL_PDF_VARIANTS}
                  value={variant}
                  onChange={setVariant}
                  disabled={false}
                />
                <Typography.Text type="secondary">
                  {house?.issuerOrganizationName} ·
                  四种版本可直接选择，输出不会改变订单审核或放单状态。
                </Typography.Text>
                {notice.changed && (
                  <Alert
                    type="warning"
                    showIcon
                    title="这个订单已经发生变动，请重新刷新。"
                    action={
                      <Button onClick={requestRefresh} disabled={busy}>
                        刷新
                      </Button>
                    }
                  />
                )}
                {syncError && (
                  <Alert
                    type="warning"
                    showIcon
                    title="HBL 已保存，最新资料同步失败，请刷新后继续编辑。"
                    action={<Button onClick={requestRefresh}>重试同步</Button>}
                  />
                )}
                {unavailable && (
                  <Alert type="error" showIcon title={unavailable} />
                )}
                {(policy.disabled || !canUpdate) && (
                  <Alert
                    type="info"
                    showIcon
                    title={policy.reason ?? '当前 HBL 仅可查看，不能直接修改。'}
                    description="仍可选择四种版本进行预览、打印和下载。"
                    action={
                      lock.error ? (
                        <Button onClick={() => void lock.refresh()}>
                          重试审核状态
                        </Button>
                      ) : undefined
                    }
                  />
                )}
              </div>
            ),
          },
        ]}
        sections={buildHblFormSections({
          containerAction: (
            <Button
              size="small"
              disabled={
                disabled ||
                !access.canOrder(
                  OrderBusinessType.BUSINESS_TYPE_SE,
                  'container.read',
                )
              }
              loading={containers.isFetching}
              onClick={() => void importContainers()}
            >
              带入实际箱
            </Button>
          ),
          sourceAction: (
            <Button size="small" disabled={disabled} onClick={importTransport}>
              重新带入订单运输资料
            </Button>
          ),
        })}
        footer={
          <StickyFooterBar
            style={{ width: '100%', flexWrap: 'wrap', gap: 12 }}
            info={
              <Typography.Text>
                {house?.houseNo} · {order.orderNo}
              </Typography.Text>
            }
          >
            <Space wrap>
              <Button
                disabled={false}
                onClick={() => history.push(`/orders/sea-export/${orderId}`)}
              >
                返回订单
              </Button>
              {previewButton}
              {saveButton}
            </Space>
          </StickyFooterBar>
        }
      />
      {preview && (
        <Suspense fallback={<Alert type="info" title="正在加载提单预览…" />}>
          <HblPdfPreview
            open
            onClose={() => setPreview(undefined)}
            model={preview}
            variant={variant}
          />
        </Suspense>
      )}
    </>
  );
}
