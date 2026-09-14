import { GridContext } from '@ant-design/pro-components';
import { createStyles } from 'antd-style';
import React from 'react';

/**
 * FormRow 表单行模板（纯白高密度企业级视觉规范）。
 *
 * 基于 CSS Grid 的四种列模板，跨行共享列边界（row-2 中线 = row-4 第二列边界）：
 * - row-2 对半：一行两等分（对应旧 24 栅格 12/12）；
 * - row-3 三分：一行三等分（对应旧 8/8/8）；
 * - row-4 四分：一行四等分（对应旧 6/6/6/6）；
 * - row-5 五分：一行五等分。
 * 子元素按顺序直接落入网格（字段组件自带 Form.Item，无需再包 Row/Col）；
 * 992px 以下自动降级为单列。
 */
const useStyles = createStyles(({ css }) => ({
  grid: css`
    display: grid;
    width: 100%;
    /* 行间距 12 / 列间距 16 */
    gap: 12px 16px;
  `,
  // row-2 对半
  row2: css`
    grid-template-columns: repeat(2, minmax(0, 1fr));
    @media (max-width: 992px) {
      grid-template-columns: minmax(0, 1fr);
    }
  `,
  // row-3 三分
  row3: css`
    grid-template-columns: repeat(3, minmax(0, 1fr));
    @media (max-width: 992px) {
      grid-template-columns: minmax(0, 1fr);
    }
  `,
  // row-4 四分
  row4: css`
    grid-template-columns: repeat(4, minmax(0, 1fr));
    @media (max-width: 992px) {
      grid-template-columns: minmax(0, 1fr);
    }
  `,
  // row-5 五分
  row5: css`
    grid-template-columns: repeat(5, minmax(0, 1fr));
    @media (max-width: 992px) {
      grid-template-columns: minmax(0, 1fr);
    }
  `,
}));

export interface FormRowProps {
  /** 列数：2 对半 / 3 三分 / 4 四分 / 5 五分，默认 4 */
  cols?: 2 | 3 | 4 | 5;
  className?: string;
  children?: React.ReactNode;
}

const COLS_CLASS_MAP = {
  2: 'row2',
  3: 'row3',
  4: 'row4',
  5: 'row5',
} as const;

// FormRow 内关闭 pro-form 的 grid 列包装：分列由网格自身负责，
// 字段直接作为网格项铺满列宽，避免字段被包上 Col 后继承外层
// Row 的 gutter 内边距造成 8px 错位。
const NON_GRID_CONTEXT = { grid: false };

export function FormRow({ cols = 4, className, children }: FormRowProps) {
  const { styles, cx } = useStyles();
  return (
    <GridContext.Provider value={NON_GRID_CONTEXT}>
      <div className={cx(styles.grid, styles[COLS_CLASS_MAP[cols]], className)}>
        {children}
      </div>
    </GridContext.Provider>
  );
}
