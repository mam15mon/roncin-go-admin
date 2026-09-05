import React, { useEffect, useRef, useState } from 'react';
import {
  Modal,
  Form,
  Input,
  Select,
  Radio,
  DatePicker,
  Table,
  Tag,
  Typography,
  Space,
  Alert,
  Row,
  Col,
  App,
  Spin,
  Button,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import {
  seaOrderChangeServicePreviewSeaOrderReassignment,
  seaOrderChangeServiceExecuteSeaOrderReassignment,
} from '@/services/roncin/seaOrderChangeService';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import type { DefaultOptionType } from 'antd/es/select';
import { computeCanonicalSha256 } from '@/utils/hash';

const { Text } = Typography;
const { TextArea } = Input;

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const messageValue = (error as { message?: unknown }).message;
    if (typeof messageValue === 'string' && messageValue) return messageValue;
  }
  return fallback;
}

interface SeaOrderReassignmentModalProps {
  orderId: string;
  orderNo?: string;
  open: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onClose: () => void;
  onSuccess: () => void;
  searchCarriers?: (keyword?: string) => Promise<DefaultOptionType[]>;
  searchIssuers?: (keyword?: string) => Promise<DefaultOptionType[]>;
  searchLocations?: (keyword?: string) => Promise<DefaultOptionType[]>;
}

export const SeaOrderReassignmentModal: React.FC<SeaOrderReassignmentModalProps> = ({
  orderId,
  orderNo,
  open,
  disabled = false,
  disabledReason,
  onClose,
  onSuccess,
  searchCarriers,
  searchIssuers,
  searchLocations,
}) => {
  const [form] = Form.useForm();
  const { message, modal } = App.useApp();

  const [submitting, setSubmitting] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [previewData, setPreviewData] = useState<API.SeaOrderReassignmentPreviewData | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [orderVersion, setOrderVersion] = useState<string>('0');
  const [linkVersion, setLinkVersion] = useState<string>('0');
  const [targetType, setTargetType] = useState<'candidate' | 'new'>('new');
  const [candidateMatched, setCandidateMatched] = useState<API.SeaMasterBillCandidate | null>(null);
  const disabledRef = useRef({ disabled, reason: disabledReason });
  disabledRef.current = { disabled, reason: disabledReason };
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  // 选项缓存
  const [carrierOptions, setCarrierOptions] = useState<DefaultOptionType[]>([]);
  const [issuerOptions, setIssuerOptions] = useState<DefaultOptionType[]>([]);
  const [originPortOptions, setOriginPortOptions] = useState<DefaultOptionType[]>([]);
  const [dischargePortOptions, setDischargePortOptions] = useState<DefaultOptionType[]>([]);
  const [transitPortOptions, setTransitPortOptions] = useState<DefaultOptionType[]>([]);

  useEffect(() => {
    if (open && disabled) {
      onCloseRef.current();
      return;
    }
    if (open && orderId) {
      form.resetFields();
      setCandidateMatched(null);
      setPreviewData(null);
      setPreviewError(null);
      setTargetType('new');

      triggerPreview();
    }
  }, [disabled, open, orderId]);

  // 实时触发比对预览
  const triggerPreview = async (customTarget?: API.SeaOrderReassignmentTargetInput) => {
    if (!orderId || disabledRef.current.disabled) return;
    setPreviewing(true);
    setPreviewError(null);
    try {
      const values = form.getFieldsValue();
      const targetInput: API.SeaOrderReassignmentTargetInput = customTarget || {
        targetType: targetType === 'candidate' && candidateMatched?.id ? 'CANDIDATE' : 'NEW',
        candidateId: targetType === 'candidate' ? candidateMatched?.id : undefined,
        candidateVersion: targetType === 'candidate' && candidateMatched?.version ? String(candidateMatched.version) : undefined,
        candidateTeId: targetType === 'candidate' ? candidateMatched?.transportExecution?.id : undefined,
        candidateTeVersion:
          targetType === 'candidate' && candidateMatched?.transportExecution?.version
            ? String(candidateMatched.transportExecution.version)
            : undefined,
        masterNo: values.masterNo || '',
        issuerPartnerId: values.issuerPartnerId,
        carrierId: values.carrierId,
        vesselName: values.vesselName?.trim(),
        voyageNo: values.voyageNo?.trim(),
        originLocationId: values.originLocationId,
        dischargeLocationId: values.dischargeLocationId,
        transitLocationId: values.transitLocationId,
        etd: values.etd ? dayjs(values.etd).format('YYYY-MM-DD HH:mm:ss') : undefined,
        eta: values.eta ? dayjs(values.eta).format('YYYY-MM-DD HH:mm:ss') : undefined,
      };

      const resp = await seaOrderChangeServicePreviewSeaOrderReassignment(
        { orderId },
        { orderId, target: targetInput },
      );
      if (resp?.data) {
        setPreviewData(resp.data);
        if (resp.data.orderVersion) {
          setOrderVersion(String(resp.data.orderVersion));
        }
        if (resp.data.currentLinkVersion) {
          setLinkVersion(String(resp.data.currentLinkVersion));
        }
      }
    } catch (error: unknown) {
      setPreviewError(getErrorMessage(error, '改配比对校验失败'));
      setPreviewData(null);
    } finally {
      setPreviewing(false);
    }
  };

  // 匹配候选母单
  const handleMatchCandidate = async () => {
    if (disabledRef.current.disabled) {
      message.warning(disabledRef.current.reason || '订单当前不可编辑');
      return;
    }
    const masterNo = form.getFieldValue('masterNo');
    const issuerPartnerId = form.getFieldValue('issuerPartnerId');
    if (!masterNo) {
      message.warning('请先输入提单号(MBL)');
      return;
    }
    if (!/^[A-Za-z0-9]+$/.test(masterNo)) {
      message.warning('提单号只能包含英文字母和阿拉伯数字，不能包含空格或符号');
      return;
    }
    if (!issuerPartnerId) {
      message.warning('请先选择签发方');
      return;
    }
    try {
      const resp = await orderServiceMatchSeaMasterBillCandidate({
        masterNo,
        issuerPartnerId,
      });
      if (resp?.matched && resp.candidate) {
        const c = resp.candidate;
        const te = c.transportExecution;
        if (!c.id || !c.version || !te?.id || !te.version) {
          message.error('候选母单或运输执行缺少版本信息，无法选择！');
          return;
        }
        setCandidateMatched(c);
        setTargetType('candidate');
        // 自动填充信息
        form.setFieldsValue({
          masterNo: c.masterNo,
          issuerPartnerId: c.issuerPartnerId,
          carrierId: te?.carrierId,
          vesselName: te?.vesselName,
          voyageNo: te?.voyageNo,
          originLocationId: te?.originLocationId,
          dischargeLocationId: te?.dischargeLocationId,
          transitLocationId: te?.transitLocationId,
          etd: te?.etd ? dayjs(te.etd) : undefined,
          eta: te?.eta ? dayjs(te.eta) : undefined,
        });
        message.success(`成功匹配到共享母单 [${c.masterNo}]，当前已有 ${c.memberCount || 1} 票订单`);
        const nextTargetInput: API.SeaOrderReassignmentTargetInput = {
          targetType: 'CANDIDATE',
          candidateId: c.id,
          candidateVersion: String(c.version),
          candidateTeId: te.id,
          candidateTeVersion: String(te.version),
          masterNo: c.masterNo,
          issuerPartnerId: c.issuerPartnerId,
          carrierId: te?.carrierId,
          vesselName: te?.vesselName,
          voyageNo: te?.voyageNo,
          originLocationId: te?.originLocationId,
          dischargeLocationId: te?.dischargeLocationId,
          transitLocationId: te?.transitLocationId,
          etd: te?.etd ? dayjs(te.etd).format('YYYY-MM-DD HH:mm:ss') : undefined,
          eta: te?.eta ? dayjs(te.eta).format('YYYY-MM-DD HH:mm:ss') : undefined,
        };
        triggerPreview(nextTargetInput);
      } else {
        message.info('未找到匹配的已有母单，将作为新母单创建');
        setCandidateMatched(null);
        setTargetType('new');
        const values = form.getFieldsValue();
        const nextTargetInput: API.SeaOrderReassignmentTargetInput = {
          targetType: 'NEW',
          masterNo: values.masterNo || '',
          issuerPartnerId: values.issuerPartnerId,
          carrierId: values.carrierId,
          vesselName: values.vesselName?.trim(),
          voyageNo: values.voyageNo?.trim(),
          originLocationId: values.originLocationId,
          dischargeLocationId: values.dischargeLocationId,
          transitLocationId: values.transitLocationId,
          etd: values.etd ? dayjs(values.etd).format('YYYY-MM-DD HH:mm:ss') : undefined,
          eta: values.eta ? dayjs(values.eta).format('YYYY-MM-DD HH:mm:ss') : undefined,
        };
        triggerPreview(nextTargetInput);
      }
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '匹配候选母单失败'));
    }
  };

  // 提交执行改配
  const handleExecute = async () => {
    if (disabledRef.current.disabled) {
      message.warning(disabledRef.current.reason || '订单当前不可编辑');
      return;
    }
    if (!orderVersion || orderVersion === '0' || !linkVersion || linkVersion === '0') {
      message.error('未获取到有效的订单或母单关联版本，请刷新重试！');
      return;
    }
    if (
      targetType === 'candidate' &&
      (!candidateMatched?.id ||
        !candidateMatched?.version ||
        !candidateMatched.transportExecution?.id ||
        !candidateMatched.transportExecution.version)
    ) {
      message.error('候选母单或运输执行缺少有效版本，无法提交！');
      return;
    }

    try {
      const values = await form.validateFields();

      modal.confirm({
        title: '确认提交整票改配？',
        content: (
          <div>
            <p>
              保留当前操作票号与流程状态，不会复制订单；当前航程将切换为目标 MBL 权威航程。
            </p>
            <p style={{ color: '#fa8c16' }}>
              请核对下方航程要素变动表。执行后将生成不可变改配历史记录。
            </p>
          </div>
        ),
        okText: '确认改配',
        cancelText: '取消',
        onOk: async () => {
          if (disabledRef.current.disabled) {
            message.warning(disabledRef.current.reason || '订单当前不可编辑');
            return;
          }
          setSubmitting(true);
          try {
            const targetInput: API.SeaOrderReassignmentTargetInput = {
              targetType: targetType === 'candidate' && candidateMatched?.id ? 'CANDIDATE' : 'NEW',
              candidateId: targetType === 'candidate' ? candidateMatched?.id : undefined,
              candidateVersion: targetType === 'candidate' && candidateMatched?.version ? String(candidateMatched.version) : undefined,
              candidateTeId: targetType === 'candidate' && candidateMatched?.transportExecution?.id ? candidateMatched.transportExecution.id : undefined,
              candidateTeVersion: targetType === 'candidate' && candidateMatched?.transportExecution?.version ? String(candidateMatched.transportExecution.version) : undefined,
              masterNo: values.masterNo || '',
              issuerPartnerId: values.issuerPartnerId,
              carrierId: values.carrierId,
              vesselName: values.vesselName?.trim(),
              voyageNo: values.voyageNo?.trim(),
              originLocationId: values.originLocationId,
              dischargeLocationId: values.dischargeLocationId,
              transitLocationId: values.transitLocationId,
              etd: values.etd ? dayjs(values.etd).format('YYYY-MM-DD HH:mm:ss') : undefined,
              eta: values.eta ? dayjs(values.eta).format('YYYY-MM-DD HH:mm:ss') : undefined,
            };

            const payloadForHash = {
              orderId,
              target: targetInput,
              reason: values.reason?.trim() || '',
              responsibilityType: values.responsibilityType,
              responsiblePartnerId: values.responsiblePartnerId || undefined,
              expectedOrderVersion: orderVersion,
              expectedLinkVersion: linkVersion,
              expectedCandidateMblVersion:
                targetType === 'candidate' ? String(candidateMatched?.version) : undefined,
              expectedCandidateTeVersion:
                targetType === 'candidate'
                  ? String(candidateMatched?.transportExecution?.version)
                  : undefined,
            };
            const hash = computeCanonicalSha256(payloadForHash);
            const fingerprint = `reassign-fp:${hash}`;
            const idempotencyKey = `reassign:${orderId}:${hash}`;

            await seaOrderChangeServiceExecuteSeaOrderReassignment(
              { orderId },
              {
                orderId,
                idempotencyKey,
                requestFingerprint: fingerprint,
                target: targetInput,
                reason: values.reason.trim(),
                responsibilityType: values.responsibilityType,
                responsiblePartnerId: values.responsiblePartnerId || undefined,
                expectedOrderVersion: orderVersion,
                expectedLinkVersion: linkVersion,
                expectedCandidateMblVersion:
                  targetType === 'candidate' ? String(candidateMatched?.version) : undefined,
                expectedCandidateTeVersion:
                  targetType === 'candidate'
                    ? String(candidateMatched?.transportExecution?.version)
                    : undefined,
              },
            );

            message.success('整票改配成功！航程与母单已更新');
            onSuccess();
            onClose();
          } catch (error: unknown) {
            message.error(getErrorMessage(error, '执行改配失败'));
          } finally {
            setSubmitting(false);
          }
        },
      });
    } catch (_error: unknown) {
      // 表单校验未通过
    }
  };

  const diffColumns: ColumnsType<API.VoyageDifferenceItem> = [
    {
      title: '航程要素',
      dataIndex: 'label',
      width: 140,
    },
    {
      title: '当前原值',
      dataIndex: 'currentValue',
      render: (val: string) => <Text type="secondary">{val || '未填写'}</Text>,
    },
    {
      title: '改配目标值',
      dataIndex: 'targetValue',
      render: (val: string, record) => {
        if (record.isDifferent) {
          return <Text strong style={{ color: '#cf1322' }}>{val || '未填写'}</Text>;
        }
        return <Text>{val || '未填写'}</Text>;
      },
    },
    {
      title: '变更状态',
      width: 90,
      render: (_, record) => {
        if (record.isDifferent) {
          return <Tag color="error">发生变动</Tag>;
        }
        return <Tag color="default">保持一致</Tag>;
      },
    },
  ];

  return (
    <Modal
      title={`订单整票改配 - ${orderNo || orderId}`}
      open={open}
      onCancel={onClose}
      width={900}
      destroyOnClose={false}
      confirmLoading={submitting}
      onOk={handleExecute}
      okButtonProps={{ disabled }}
      okText="确认改配"
      cancelText="取消"
    >
      <Alert
        type="info"
        showIcon
        message="改配说明"
        description="改配操作将把当前订单完整切换到新航程或目标母单，同步更新船名航次、起运港/卸货港等航程要素。若目标母单已存在，将自动加入共享母单组。"
        style={{ marginBottom: 16 }}
      />

      {previewError && (
        <Alert
          type="error"
          showIcon
          message="改配校验未通过"
          description={previewError}
          style={{ marginBottom: 16 }}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        initialValues={{
          responsibilityType: 'OWN_COMPANY',
        }}
        onValuesChange={() => triggerPreview()}
      >
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item label="母单提供方式">
              <Radio.Group
                value={targetType}
                onChange={(e) => {
                  setTargetType(e.target.value);
                  if (e.target.value === 'new') {
                    setCandidateMatched(null);
                  }
                  triggerPreview();
                }}
              >
                <Radio.Button value="new">手工录入新母单</Radio.Button>
                <Radio.Button value="candidate">匹配已有共享母单</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="masterNo"
              label="目标提单号 (MBL No)"
              rules={[
                { required: true, message: '请输入目标提单号' },
                {
                  pattern: /^[A-Za-z0-9]+$/,
                  message: '提单号只能包含英文字母和阿拉伯数字，不能包含空格或符号',
                },
              ]}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Input placeholder="请输入目标提单号" />
                <Button type="primary" onClick={handleMatchCandidate}>
                  检查/匹配
                </Button>
              </Space.Compact>
            </Form.Item>
          </Col>
        </Row>

        {candidateMatched && (
          <Alert
            type="success"
            showIcon
            message={`已关联到现有共享母单：${candidateMatched.masterNo}`}
            description={`发单人：${candidateMatched.issuerPartnerName || '无'}，当前已有 ${candidateMatched.memberCount} 票成员订单。`}
            style={{ marginBottom: 16 }}
          />
        )}

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item
              name="issuerPartnerId"
              label="发单人 / 船代"
              rules={[{ required: true, message: '请选择发单人' }]}
            >
              <Select
                showSearch
                placeholder="搜索选择发单人"
                filterOption={false}
                options={issuerOptions}
                onSearch={async (k) => {
                  if (searchIssuers) {
                    const opts = await searchIssuers(k);
                    setIssuerOptions(opts);
                  }
                }}
              />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="carrierId" label="承运人 / 船东">
              <Select
                showSearch
                allowClear
                placeholder="搜索选择船东"
                filterOption={false}
                options={carrierOptions}
                onSearch={async (k) => {
                  if (searchCarriers) {
                    const opts = await searchCarriers(k);
                    setCarrierOptions(opts);
                  }
                }}
              />
            </Form.Item>
          </Col>
          <Col span={4}>
            <Form.Item name="vesselName" label="船名">
              <Input placeholder="船名" />
            </Form.Item>
          </Col>
          <Col span={4}>
            <Form.Item name="voyageNo" label="航次">
              <Input placeholder="航次" />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item name="originLocationId" label="起运港 (POL)">
              <Select
                showSearch
                allowClear
                placeholder="搜索起运港"
                filterOption={false}
                options={originPortOptions}
                onSearch={async (k) => {
                  if (searchLocations) {
                    const opts = await searchLocations(k);
                    setOriginPortOptions(opts);
                  }
                }}
              />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="dischargeLocationId" label="卸货港 (POD)">
              <Select
                showSearch
                allowClear
                placeholder="搜索卸货港"
                filterOption={false}
                options={dischargePortOptions}
                onSearch={async (k) => {
                  if (searchLocations) {
                    const opts = await searchLocations(k);
                    setDischargePortOptions(opts);
                  }
                }}
              />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="transitLocationId" label="中转港">
              <Select
                showSearch
                allowClear
                placeholder="搜索中转港"
                filterOption={false}
                options={transitPortOptions}
                onSearch={async (k) => {
                  if (searchLocations) {
                    const opts = await searchLocations(k);
                    setTransitPortOptions(opts);
                  }
                }}
              />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="etd" label="预计离港时间 (ETD)">
              <DatePicker showTime style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="eta" label="预计到港时间 (ETA)">
              <DatePicker showTime style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item
              name="responsibilityType"
              label="责任归属类型"
              rules={[{ required: true, message: '请选择责任归属' }]}
            >
              <Select
                options={[
                  { label: '我司原因 (OWN_COMPANY)', value: 'OWN_COMPANY' },
                  { label: '客户原因 (CUSTOMER)', value: 'CUSTOMER' },
                  { label: '船东原因 (CARRIER)', value: 'CARRIER' },
                  { label: '海关原因 (CUSTOMS)', value: 'CUSTOMS' },
                  { label: '不可抗力 (FORCE_MAJEURE)', value: 'FORCE_MAJEURE' },
                  { label: '其他原因 (OTHER)', value: 'OTHER' },
                ]}
              />
            </Form.Item>
          </Col>
          <Col span={16}>
            <Form.Item
              name="reason"
              label="改配原因说明"
              rules={[{ required: true, message: '请输入详细改配原因' }]}
            >
              <TextArea rows={2} placeholder="详细记录改配发生的背景与原因（将记录进不可变事件历史与审计日志）" />
            </Form.Item>
          </Col>
        </Row>
      </Form>

      {/* 航程要素比对表格 */}
      <div style={{ marginTop: 12 }}>
        <Text strong style={{ fontSize: 14, marginBottom: 8, display: 'block' }}>
          航程要素变动实时比对：
        </Text>
        {previewing ? (
          <div style={{ textAlign: 'center', padding: 20 }}>
            <Spin description="计算比对差异..." />
          </div>
        ) : (
          <Table<API.VoyageDifferenceItem>
            columns={diffColumns}
            dataSource={previewData?.differences || []}
            rowKey="fieldName"
            pagination={false}
            size="small"
            bordered
          />
        )}
      </div>
    </Modal>
  );
};

export default SeaOrderReassignmentModal;
