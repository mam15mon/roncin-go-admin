import { Form, InputNumber, Select, Space } from 'antd';
import React, { useMemo, useState } from 'react';
import { PACKAGE_UNIT_OPTIONS } from './packageUnits';
import type { PackageCountInputProps } from './types';

/**
 * PackageCountInput
 *
 * 复合连体件数与包装单位输入组件：
 * - 遵循与 CurrencyAmountInput 一致的连体容器标准（Space.Compact），保证高度 100% 对齐；
 * - 左侧输入件数（整数/非负），右侧紧凑选择/搜索包装单位（支持 500+ 国际海运与内贸标准单位种子，支持自由输入）；
 * - 独立绑定 countName 与 unitName，无需改造后端 DTO。
 */
export const PackageCountInput: React.FC<PackageCountInputProps> = ({
  countName,
  unitName,
  countPlaceholder = '件数',
  unitPlaceholder = '单位',
  disabled = false,
  unitWidth = 130,
  unitOptions = PACKAGE_UNIT_OPTIONS,
  min = 0,
  style,
  className,
}) => {
  const form = Form.useFormInstance();
  const [searchValue, setSearchValue] = useState('');

  // 监听当前单位值，确保不在预置字典中的自定义值也能正确回显
  const currentUnitVal = Form.useWatch(unitName, form);

  const mergedOptions = useMemo(() => {
    const list = [...unitOptions];
    const valSet = new Set(list.map((o) => o.value.toLowerCase()));

    // 1. 如果已有自定义值不在选项中，追加到选项
    if (
      currentUnitVal &&
      typeof currentUnitVal === 'string' &&
      currentUnitVal.trim() !== ''
    ) {
      const trimmed = currentUnitVal.trim();
      if (!valSet.has(trimmed.toLowerCase())) {
        list.unshift({ label: trimmed, value: trimmed });
        valSet.add(trimmed.toLowerCase());
      }
    }

    // 2. 如果正在搜索且搜索词不在选项中，在顶部提供该自定义值以供一键确认录入
    const trimmedSearch = searchValue.trim();
    if (trimmedSearch && !valSet.has(trimmedSearch.toLowerCase())) {
      list.unshift({
        label: `${trimmedSearch} (自定义)`,
        value: trimmedSearch,
      });
    }

    return list;
  }, [currentUnitVal, searchValue, unitOptions]);

  const filterOption = (
    input: string,
    option?: { label?: React.ReactNode; value?: string },
  ) => {
    if (!input || !option?.value) return true;
    const normInput = input.replace(/[\s\-_/]/g, '').toLowerCase();
    const normVal = option.value.replace(/[\s\-_/]/g, '').toLowerCase();
    const normLabel = String(option.label ?? '')
      .replace(/[\s\-_/]/g, '')
      .toLowerCase();
    return normVal.includes(normInput) || normLabel.includes(normInput);
  };

  return (
    <Space.Compact
      block
      className={`roncin-package-count-input ${className || ''}`}
      style={{ width: '100%', ...style }}
    >
      {/* 1. 左侧件数数字输入 */}
      <Form.Item noStyle name={countName}>
        <InputNumber
          min={min}
          precision={0}
          placeholder={countPlaceholder}
          disabled={disabled}
          style={{
            width: `calc(100% - ${unitWidth}px)`,
            textAlign: 'left',
          }}
        />
      </Form.Item>

      {/* 2. 右侧包装单位搜索选择器 */}
      <Form.Item noStyle name={unitName}>
        <Select
          showSearch
          allowClear
          placeholder={unitPlaceholder}
          disabled={disabled}
          options={mergedOptions}
          filterOption={filterOption}
          onSearch={setSearchValue}
          onSelect={() => setSearchValue('')}
          onBlur={() => setSearchValue('')}
          popupMatchSelectWidth={false}
          style={{ width: unitWidth }}
        />
      </Form.Item>
    </Space.Compact>
  );
};
