package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type feeLedgerPreferenceRepoStub struct {
	value *FeeLedgerPreference
}

func (stub *feeLedgerPreferenceRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*FeeLedgerPreference, error) {
	return stub.value, nil
}

func (stub *feeLedgerPreferenceRepoStub) Save(_ context.Context, value *FeeLedgerPreference) (*FeeLedgerPreference, error) {
	stub.value = value
	return value, nil
}

func (stub *feeLedgerPreferenceRepoStub) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID, version uint64) error {
	if stub.value == nil {
		if version == 0 {
			return nil
		}
		return ErrFeeLedgerPreferenceConflict
	}
	if version != stub.value.Version {
		return ErrFeeLedgerPreferenceConflict
	}
	stub.value = nil
	return nil
}

func TestFeeLedgerPreferenceGetReturnsSystemDefaults(t *testing.T) {
	usecase := NewFeeLedgerPreferenceUsecase(&feeLedgerPreferenceRepoStub{})
	value, err := usecase.Get(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("读取默认表头设置失败: %v", err)
	}
	if value.Customized || value.Version != 0 || value.PageSize != 40 || value.RowColors.Completed != "#F6FFED" {
		t.Fatalf("默认表头设置不正确: %+v", value)
	}
}

func TestNormalizeFeeLedgerPreferencePreservesColumnOrder(t *testing.T) {
	organizationID, userID := uuid.New(), uuid.New()
	value, err := normalizeFeeLedgerPreference(organizationID, userID, validFeeLedgerPreference())
	if err != nil {
		t.Fatalf("标准化表头设置失败: %v", err)
	}
	if value.Columns[0].FieldKey != "orderNo" || value.Columns[1].FieldKey != "customerName" || value.SortDirection != FeeLedgerSortDescending {
		t.Fatalf("列顺序或排序设置未保留: %+v", value)
	}
}

func TestFeeLedgerPreferenceResetReturnsDefaults(t *testing.T) {
	stub := &feeLedgerPreferenceRepoStub{value: &FeeLedgerPreference{Version: 3}}
	usecase := NewFeeLedgerPreferenceUsecase(stub)
	value, err := usecase.Reset(context.Background(), uuid.New(), uuid.New(), 3)
	if err != nil {
		t.Fatalf("重置表头设置失败: %v", err)
	}
	if stub.value != nil || value.Customized || value.Version != 0 || value.PageSize != 40 {
		t.Fatalf("重置后未恢复系统默认值: %+v", value)
	}
}

func TestNormalizeFeeLedgerPreferenceRejectsInvalidSettings(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*FeeLedgerPreference)
	}{
		{name: "不支持的分页行数", modify: func(value *FeeLedgerPreference) { value.PageSize = 20 }},
		{name: "重复字段", modify: func(value *FeeLedgerPreference) { value.Columns[1].FieldKey = value.Columns[0].FieldKey }},
		{name: "没有显示字段", modify: func(value *FeeLedgerPreference) {
			for index := range value.Columns {
				value.Columns[index].Visible = false
			}
		}},
		{name: "排序字段不存在", modify: func(value *FeeLedgerPreference) { value.SortField = "missingField" }},
		{name: "颜色格式错误", modify: func(value *FeeLedgerPreference) { value.RowColors.Completed = "green" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validFeeLedgerPreference()
			test.modify(value)
			if _, err := normalizeFeeLedgerPreference(uuid.New(), uuid.New(), value); err != ErrFeeLedgerPreferenceInvalidArgument {
				t.Fatalf("错误 = %v，期望 ErrFeeLedgerPreferenceInvalidArgument", err)
			}
		})
	}
}

func validFeeLedgerPreference() *FeeLedgerPreference {
	return &FeeLedgerPreference{
		Columns: []FeeLedgerColumnPreference{
			{FieldKey: "orderNo", Visible: true},
			{FieldKey: "customerName", Visible: true},
		},
		PageSize:      60,
		SortField:     "orderNo",
		SortDirection: FeeLedgerSortDescending,
		RowColors:     defaultFeeLedgerRowColors(),
	}
}
