import { CopyOutlined, EditOutlined } from '@ant-design/icons';
import { Button, Card, Space, Switch, Tag, Tooltip } from 'antd';
import React from 'react';
import {
  DATE_FORMATS,
  type DocTypeMeta,
  RESET_POLICIES,
  generatePreviewNumber,
} from './numberRulesConstants';

interface NumberRuleCardProps {
  rule: API.NumberRule;
  meta: DocTypeMeta;
  onToggleActive: (rule: API.NumberRule, checked: boolean) => void;
  onOpenEdit: (rule: API.NumberRule) => void;
  onCopyPreview: (text: string) => void;
}

export default function NumberRuleCard({
  rule,
  meta,
  onToggleActive,
  onOpenEdit,
  onCopyPreview,
}: NumberRuleCardProps) {
  const sample = generatePreviewNumber(rule);
  const dateMeta = DATE_FORMATS[rule.dateFormat as any] || DATE_FORMATS[4];
  const resetMeta =
    RESET_POLICIES[rule.resetPolicy as any] || RESET_POLICIES[4];

  return (
    <Card
      size="small"
      variant="borderless"
      style={{
        borderRadius: 8,
        border: '1px solid #e8e8e8',
        overflow: 'hidden',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        boxShadow: '0 2px 6px rgba(0, 0, 0, 0.02)',
        transition: 'all 0.3s cubic-bezier(0.645, 0.045, 0.355, 1)',
      }}
      styles={{
        body: {
          padding: '16px',
          display: 'flex',
          flexDirection: 'column',
          flex: 1,
          justifyContent: 'space-between',
        },
      }}
    >
      {/* Card Top Strip */}
      <div>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            marginBottom: 12,
          }}
        >
          <div>
            <Tag
              color={meta.color}
              style={{
                fontSize: 12,
                fontWeight: 600,
                padding: '2px 8px',
                marginBottom: 4,
              }}
            >
              {meta.shortLabel}
            </Tag>
            <div
              style={{
                fontSize: 14,
                fontWeight: 600,
                color: '#262626',
              }}
            >
              {meta.label}
            </div>
          </div>

          {/* Status Switch */}
          <Space size={6} align="center">
            <span
              style={{
                fontSize: 12,
                color: rule.enabled ? '#52c41a' : '#8c8c8c',
                fontWeight: 500,
              }}
            >
              {rule.enabled ? '启用' : '停用'}
            </span>
            <Switch
              size="small"
              checked={Boolean(rule.enabled)}
              onChange={(checked) => onToggleActive(rule, checked)}
            />
          </Space>
        </div>

        {/* Rule Attributes Table List */}
        <div
          style={{
            backgroundColor: '#fafafa',
            borderRadius: 6,
            padding: '8px 12px',
            fontSize: 12,
            marginBottom: 14,
            display: 'flex',
            flexDirection: 'column',
            gap: 6,
          }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <span style={{ color: '#8c8c8c' }}>前缀代码：</span>
            {meta.numValue === 1 ? (
              <Tooltip title="订单前缀由系统根据订单流向 (如 SE/SI/AE/AI) 自动匹配">
                <Tag
                  color="blue"
                  style={{
                    fontFamily: 'ui-monospace, monospace',
                    fontWeight: 600,
                    margin: 0,
                  }}
                >
                  {rule.prefix ? `${rule.prefix}-{业务代码}` : '{业务代码}'}
                </Tag>
              </Tooltip>
            ) : (
              <Tag
                style={{
                  fontFamily: 'ui-monospace, monospace',
                  fontWeight: 600,
                  color: rule.prefix ? '#1677ff' : '#8c8c8c',
                  margin: 0,
                }}
              >
                {rule.prefix || '无'}
              </Tag>
            )}
          </div>
          {meta.businessCodes && (
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: '#8c8c8c' }}>业务流向：</span>
              <span
                style={{
                  fontFamily: 'ui-monospace, monospace',
                  color: '#262626',
                  fontSize: 11,
                }}
              >
                {meta.businessCodes.join(' / ')}
              </span>
            </div>
          )}
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: '#8c8c8c' }}>日期格式：</span>
            <span style={{ color: '#262626' }}>{dateMeta.label}</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: '#8c8c8c' }}>流水位数：</span>
            <span style={{ color: '#262626' }}>
              {rule.sequenceLength || 4} 位数字
            </span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: '#8c8c8c' }}>重置策略：</span>
            <span style={{ color: '#262626' }}>{resetMeta.label}</span>
          </div>
        </div>
      </div>

      {/* Card Bottom: Live Preview Box & Actions */}
      <div>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            fontSize: 11,
            color: '#8c8c8c',
            marginBottom: 4,
          }}
        >
          <span>单号示例预览：</span>
          {sample.isDynamicType && (
            <span style={{ fontSize: 11, color: '#1677ff' }}>
              SE 为示例流向
            </span>
          )}
        </div>
        <div
          style={{
            backgroundColor: '#f6ffed',
            border: '1px dashed #b7eb8f',
            borderRadius: 6,
            padding: '6px 10px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 12,
          }}
        >
          <span
            style={{
              fontFamily:
                'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
              fontSize: 13,
              fontWeight: 600,
              color: '#389e0d',
              letterSpacing: '0.5px',
            }}
          >
            {sample.text}
          </span>
          <Tooltip title="复制示例">
            <Button
              type="text"
              size="small"
              icon={
                <CopyOutlined
                  style={{ fontSize: 12, color: '#52c41a' }}
                />
              }
              onClick={() => onCopyPreview(sample.text)}
              style={{ height: 22, width: 22, padding: 0 }}
            />
          </Tooltip>
        </div>

        {/* Edit Button */}
        <Button
          type="default"
          block
          icon={<EditOutlined />}
          onClick={() => onOpenEdit(rule)}
          style={{ borderRadius: 6 }}
        >
          修改规则
        </Button>
      </div>
    </Card>
  );
}
