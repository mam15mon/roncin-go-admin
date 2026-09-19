import { Col, Row, Statistic } from 'antd';
import { SectionCard } from '@/components/ui';

interface SplitBaselineSectionProps {
  previewData: API.SeaOrderSplitPreviewData | null;
  splitContext: API.SeaOrderSplitContextData | null;
}

/** 拆票页区块 1：原始订单基线汇总（件重尺、箱数、货物项、草稿费用统计）。 */
export default function SplitBaselineSection({
  previewData,
  splitContext,
}: SplitBaselineSectionProps) {
  return (
    <SectionCard title="原始订单基线汇总" collapsible defaultCollapsed={false}>
      <Row gutter={16}>
        <Col span={4}>
          <Statistic
            title="总件数 (Packages)"
            value={previewData?.baseline?.packageCount ?? '-'}
            suffix="件"
          />
        </Col>
        <Col span={4}>
          <Statistic
            title="总毛重 (Gross Weight)"
            value={previewData?.baseline?.grossWeightKg ?? '-'}
            suffix="KGS"
          />
        </Col>
        <Col span={4}>
          <Statistic
            title="总体积 (Volume)"
            value={previewData?.baseline?.volumeCbm ?? '-'}
            suffix="CBM"
          />
        </Col>
        <Col span={4}>
          <Statistic
            title="集装箱总数"
            value={splitContext?.containers?.length || 0}
            suffix="箱"
          />
        </Col>
        <Col span={4}>
          <Statistic
            title="货物项总数"
            value={splitContext?.cargoItems?.length || 0}
            suffix="项"
          />
        </Col>
        <Col span={4}>
          <Statistic
            title="可分配草稿费用"
            value={splitContext?.draftFees?.length || 0}
            suffix="笔"
          />
        </Col>
      </Row>
    </SectionCard>
  );
}
