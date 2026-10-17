import { RightOutlined } from '@ant-design/icons';
import { App, Spin } from 'antd';
import React, { useState } from 'react';
import { authServiceSwitchOrganization } from '@/services/roncin/authService';
import styles from './organization-picker.module.less';

/** 登录后组织选择视图使用的候选组织（与登录响应 OrganizationChoice 同构）。 */
export interface LoginOrganizationOption {
  organizationId: string;
  organizationName: string;
  organizationCode: string;
}

/** 密码登录使用服务端候选，钉钉登录使用相同排序口径的工作台集合。 */
export function resolveLoginOrganizationOptions(
  organizationChoices: API.OrganizationChoice[] | undefined,
  currentUser?: API.CurrentUser,
): LoginOrganizationOption[] {
  if (organizationChoices && organizationChoices.length > 0) {
    return organizationChoices
      .filter((choice) => choice.organizationId)
      .map((choice) => ({
        organizationId: choice.organizationId,
        organizationName: choice.organizationName,
        organizationCode: choice.organizationCode,
      }));
  }
  return (currentUser?.organizations ?? [])
    .filter((organization) => organization.id)
    .map((organization) => ({
      organizationId: organization.id as string,
      organizationName: organization.name ?? '',
      organizationCode: organization.code ?? '',
    }));
}

interface OrganizationPickerProps {
  /** 候选组织（调用方保证多于 1 个才渲染本组件）。 */
  options: LoginOrganizationOption[];
  currentOrganizationId?: string;
  /** 进入所选组织后完成登录导航。 */
  onEnter: () => void;
}

/** 用户明确选择当前会话工作台时直接进入，其他选择通过会话轮转进入。 */
export function OrganizationPicker({
  options,
  currentOrganizationId,
  onEnter,
}: OrganizationPickerProps) {
  const { message } = App.useApp();
  const [switchingOrgId, setSwitchingOrgId] = useState<string>();

  const handleSelect = async (
    option: LoginOrganizationOption,
  ): Promise<false | undefined> => {
    if (switchingOrgId) return;
    if (option.organizationId === currentOrganizationId) {
      onEnter();
      return;
    }
    setSwitchingOrgId(option.organizationId);
    try {
      const response = await authServiceSwitchOrganization(
        { organizationId: option.organizationId },
        { skipErrorHandler: true },
      );
      if (!response || response.success === false) return;
      if (!response.data?.id) {
        message.error('进入所选组织失败，请稍后重试');
        return;
      }
      onEnter();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : '进入所选组织失败，请稍后重试',
      );
    } finally {
      setSwitchingOrgId(undefined);
    }

    return undefined;
  };

  return (
    <div className={styles.picker}>
      <h1 className={styles.title}>选择进入的组织</h1>
      <p className={styles.subtitle}>
        你的账号属于多个组织，请选择本次进入的组织
      </p>
      <div className={styles.grid}>
        {options.map((option) => {
          const switching = switchingOrgId === option.organizationId;
          return (
            <button
              key={option.organizationId}
              type="button"
              className={styles.orgCard}
              disabled={Boolean(switchingOrgId)}
              aria-label={`进入${option.organizationName}`}
              title={option.organizationName}
              onClick={() => {
                void handleSelect(option);
              }}
            >
              <div className={styles.orgCardBody}>
                <div className={styles.orgCardHeader}>
                  <span className={styles.orgName}>
                    {option.organizationName}
                  </span>
                </div>
                <span className={styles.orgCode}>
                  {option.organizationCode}
                </span>
              </div>
              <div className={styles.orgCardAction}>
                {switching ? (
                  <Spin size="small" className={styles.cardSpin} />
                ) : (
                  <RightOutlined className={styles.enterIcon} />
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
