package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

func searchKeywordsField() ent.Field {
	return field.Text("search_keywords").Default("")
}

func searchKeywordsHook(sourceFields ...string) ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			sourceChanged := false
			for _, name := range sourceFields {
				if _, exists := mutation.Field(name); exists || mutation.FieldCleared(name) {
					sourceChanged = true
					break
				}
			}
			if !sourceChanged {
				return next.Mutate(ctx, mutation)
			}
			// 批量更新无法读取每条记录的旧字段。当前批量写路径只切换启用状态，
			// 不应使用不完整字段覆盖已有检索键。
			if mutation.Op().Is(ent.OpUpdate) {
				return next.Mutate(ctx, mutation)
			}

			values := make([]string, 0, len(sourceFields))
			for _, name := range sourceFields {
				value, exists := mutation.Field(name)
				if !exists && mutation.Op().Is(ent.OpUpdateOne) && !mutation.FieldCleared(name) {
					oldValue, err := mutation.OldField(ctx, name)
					if err != nil {
						return nil, err
					}
					value, exists = oldValue, true
				}
				if !exists {
					continue
				}
				switch typed := value.(type) {
				case string:
					values = append(values, typed)
				case *string:
					if typed != nil {
						values = append(values, *typed)
					}
				}
			}
			if err := mutation.SetField("search_keywords", searchtext.Build(values...)); err != nil {
				return nil, err
			}
			return next.Mutate(ctx, mutation)
		})
	}
}
