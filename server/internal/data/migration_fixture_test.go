package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	entgo "entgo.io/ent"
	"github.com/google/uuid"
)

// insertMigrationFixture 仅用于正式迁移前态：写入夹具显式声明的旧字段，
// 未指定字段使用该时点数据库默认值，避免现行 Ent 默认值与 hook 写入未来新增列。
// 返回值仅作事实快照，不携带 Ent 客户端；查询和更新仍显式走 data.db。
func insertMigrationFixture[T any](ctx context.Context, d *Data, table string, m entgo.Mutation) (*T, error) {
	fields := m.Fields()
	args := make([]any, 0, len(fields)+3)
	names := make([]string, 0, len(fields)+3)
	placeholders := make([]string, 0, len(fields)+3)
	appendValue := func(name string, value any) {
		names = append(names, `"`+name+`"`)
		args = append(args, value)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	for _, name := range fields {
		value, _ := m.Field(name)
		appendValue(name, value)
	}
	for name, value := range map[string]any{"id": uuid.New(), "created_at": time.Now(), "updated_at": time.Now()} {
		if _, ok := m.Field(name); !ok {
			appendValue(name, value)
		}
	}
	query := fmt.Sprintf(`INSERT INTO "%s"(%s) VALUES(%s) RETURNING to_jsonb("%s")`, table, strings.Join(names, ","), strings.Join(placeholders, ","), table)
	var raw []byte
	if err := d.sqlDB.QueryRowContext(ctx, query, args...).Scan(&raw); err != nil {
		return nil, err
	}
	// PostgreSQL numeric 在 JSON 中是数字，Ent decimal 投影为字符串，保留原始精度。
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	values := make(map[string]any)
	if err := decoder.Decode(&values); err != nil {
		return nil, err
	}
	var entity T
	typ := reflect.TypeOf(entity)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Type.Kind() != reflect.String {
			continue
		}
		key := strings.Split(field.Tag.Get("json"), ",")[0]
		if value, ok := values[key].(json.Number); ok {
			values[key] = value.String()
		}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

func mustMigrationFixture[T any](value *T, err error) *T {
	if err != nil {
		panic(err)
	}
	return value
}
