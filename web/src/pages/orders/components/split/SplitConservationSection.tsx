import {
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { Alert, Card, Col, Row, Spin, Tag, Typography } from 'antd';
import Decimal from 'decimal.js';
import { SectionCard } from '@/components/ui';

const { Text } = Typography;

interface SplitConservationSectionProps {
  previewData: API.SeaOrderSplitPreviewData | null;
  previewing: boolean;
}

/** 拆票页区块 8：实时守恒与配载重算校验结果展示。 */
export default function SplitConservationSection({
  previewData,
  previewing,
}: SplitConservationSectionProps) {
  return (
    <SectionCard
      title="实时守恒与配载重算校验"
      extra={
        previewing ? (
          <Spin size="small" />
        ) : previewData?.conservationPassed && previewData?.isValid ? (
          <Tag color="success" icon={<CheckCircleOutlined />}>
            守恒与门禁校验通过
          </Tag>
        ) : (
          <Tag color="error" icon={<ExclamationCircleOutlined />}>
            校验未通过
          </Tag>
        )
      }
    >
      {previewData?.validationErrors &&
        previewData.validationErrors.length > 0 && (
          <Alert
            type="error"
            showIcon
            message="阻断原因提示"
            description={
              <div>
                {previewData.validationErrors.map((err) => (
                  <div key={`${err.reason}-${err.message}`}>
                    <Text strong>[{err.reason}]</Text>{' '}
                    <span>{err.message}</span>
                  </div>
                ))}
              </div>
            }
            style={{ marginBottom: 16 }}
          />
        )}

      <Row gutter={16}>
        <Col span={8}>
          <Card size="small" title="原始基线总量" type="inner">
            <div>件数：{previewData?.baseline?.packageCount ?? '-'} 件</div>
            <div>毛重：{previewData?.baseline?.grossWeightKg ?? '-'} KGS</div>
            <div>体积：{previewData?.baseline?.volumeCbm ?? '-'} CBM</div>
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title="各票分配累计" type="inner">
            <div>件数：{previewData?.allocated?.packageCount ?? '-'} 件</div>
            <div>毛重：{previewData?.allocated?.grossWeightKg ?? '-'} KGS</div>
            <div>体积：{previewData?.allocated?.volumeCbm ?? '-'} CBM</div>
          </Card>
        </Col>
        <Col span={8}>
          {(() => {
            const remPkg = previewData?.remaining?.packageCount;
            let pkgColor = '#8c8c8c';
            let pkgBg = '#fafafa';
            let pkgStatusText = '未计算';
            if (remPkg !== undefined && remPkg !== null) {
              if (remPkg === 0) {
                pkgColor = '#52c41a';
                pkgBg = '#f6ffed';
                pkgStatusText = '已完全分配 (守恒)';
              } else if (remPkg > 0) {
                pkgColor = '#1677ff';
                pkgBg = '#e6f4ff';
                pkgStatusText = `分配进行中 (待分配 ${remPkg} 件)`;
              } else {
                pkgColor = '#f5222d';
                pkgBg = '#fff1f0';
                pkgStatusText = `分配超出 (超出 ${Math.abs(remPkg)} 件)`;
              }
            }

            const remainingWeightValue = previewData?.remaining?.grossWeightKg;
            const remWt =
              remainingWeightValue === undefined
                ? undefined
                : new Decimal(remainingWeightValue);
            let wtColor = '#8c8c8c';
            let wtStatusText = '未计算';
            if (remWt) {
              if (remWt.isZero()) {
                wtColor = '#52c41a';
                wtStatusText = '已完全分配 (守恒)';
              } else if (remWt.isPositive()) {
                wtColor = '#1677ff';
                wtStatusText = `进行中 (待分配 ${remainingWeightValue} KGS)`;
              } else {
                wtColor = '#f5222d';
                wtStatusText = `超出分配 (超出 ${remWt.abs().toFixed(3)} KGS)`;
              }
            }

            const remainingVolumeValue = previewData?.remaining?.volumeCbm;
            const remVol =
              remainingVolumeValue === undefined
                ? undefined
                : new Decimal(remainingVolumeValue);
            let volColor = '#8c8c8c';
            let volStatusText = '未计算';
            if (remVol) {
              if (remVol.isZero()) {
                volColor = '#52c41a';
                volStatusText = '已完全分配 (守恒)';
              } else if (remVol.isPositive()) {
                volColor = '#1677ff';
                volStatusText = `进行中 (待分配 ${remainingVolumeValue} CBM)`;
              } else {
                volColor = '#f5222d';
                volStatusText = `超出分配 (超出 ${remVol.abs().toFixed(6)} CBM)`;
              }
            }

            return (
              <Card
                size="small"
                title="未分配差额 (零误差守恒校验)"
                type="inner"
                style={{ background: pkgBg }}
              >
                <div style={{ marginBottom: 4 }}>
                  件数差额：
                  <Text strong style={{ color: pkgColor }}>
                    {previewData?.remaining?.packageCount ?? '-'} 件
                  </Text>
                  <Tag
                    color={
                      remPkg === 0
                        ? 'success'
                        : remPkg && remPkg > 0
                          ? 'processing'
                          : 'error'
                    }
                    style={{ marginLeft: 8 }}
                  >
                    {pkgStatusText}
                  </Tag>
                </div>
                <div style={{ marginBottom: 4 }}>
                  毛重差额：
                  <Text strong style={{ color: wtColor }}>
                    {previewData?.remaining?.grossWeightKg ?? '-'} KGS
                  </Text>
                  <Tag
                    color={
                      remWt?.isZero()
                        ? 'success'
                        : remWt?.isPositive()
                          ? 'processing'
                          : 'error'
                    }
                    style={{ marginLeft: 8 }}
                  >
                    {wtStatusText}
                  </Tag>
                </div>
                <div>
                  体积差额：
                  <Text strong style={{ color: volColor }}>
                    {previewData?.remaining?.volumeCbm ?? '-'} CBM
                  </Text>
                  <Tag
                    color={
                      remVol?.isZero()
                        ? 'success'
                        : remVol?.isPositive()
                          ? 'processing'
                          : 'error'
                    }
                    style={{ marginLeft: 8 }}
                  >
                    {volStatusText}
                  </Tag>
                </div>
              </Card>
            );
          })()}
        </Col>
      </Row>

      {previewData?.results && (
        <div style={{ marginTop: 16 }}>
          <Text
            strong
            style={{
              fontSize: 13,
              marginBottom: 8,
              display: 'block',
            }}
          >
            拆票结果票规划与集装箱计划自动重算：
          </Text>
          <Row gutter={[12, 12]}>
            {previewData.results.map((pr) => (
              <Col span={12} key={pr.clientResultKey}>
                <Card
                  size="small"
                  type="inner"
                  title={`${pr.resultRole === 'ORIGINAL' ? '原票' : '新票'}: ${pr.clientResultKey}`}
                >
                  <div>
                    分配货物：{pr.packageCount} 件 / {pr.grossWeightKg} KGS /{' '}
                    {pr.volumeCbm} CBM
                  </div>
                  <div>
                    分单号：{pr.houseNo || '无'} | 归属费用：{pr.feeCount} 笔
                  </div>
                  <div style={{ marginTop: 6 }}>
                    <Text type="secondary">自动重算箱计划：</Text>
                    {pr.containerPlans && pr.containerPlans.length > 0 ? (
                      pr.containerPlans.map((cp) => (
                        <Tag
                          key={cp.containerSpecId || cp.containerSpecName}
                          color="blue"
                        >
                          {cp.containerSpecName}: {cp.quantity} 箱
                        </Tag>
                      ))
                    ) : (
                      <Text type="secondary">无箱计划</Text>
                    )}
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      )}
    </SectionCard>
  );
}
