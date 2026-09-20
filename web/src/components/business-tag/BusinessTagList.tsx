import { Tag } from 'antd';

type BusinessTagListProps = {
  tags?: API.BusinessTagSummary[] | null;
};

/**
 * 业务标签列表渲染：分组色描边 Tag，空列表显示 '-'。
 * 列宽、搜索与页面操作配置留在调用方列定义中。
 */
export function BusinessTagList({ tags }: BusinessTagListProps) {
  if (!tags?.length) {
    return <>-</>;
  }
  return (
    <>
      {tags.map((tag) => (
        <Tag
          key={tag.id}
          style={
            tag.groupColor
              ? {
                  color: tag.groupColor,
                  borderColor: tag.groupColor,
                  marginInlineEnd: 4,
                }
              : { marginInlineEnd: 4 }
          }
        >
          {tag.name}
        </Tag>
      ))}
    </>
  );
}
