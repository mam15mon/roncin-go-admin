package biz

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOrderFeeSaveRowKeyStable(t *testing.T) {
	a := OrderFeeSaveRowKey("req-1", "row-1")
	if a != OrderFeeSaveRowKey("req-1", "row-1") || a == OrderFeeSaveRowKey("req-1", "row-2") || a == OrderFeeSaveRowKey("req-2", "row-1") {
		t.Fatal("幂等键应对相同输入稳定、对不同输入不同")
	}
	if len(a) != 64 {
		t.Fatalf("应为 64 位十六进制: %d", len(a))
	}
}

func TestOrderFeeCurrencySetsMatchAfterSave(t *testing.T) {
	r1 := &OrderFee{ID: uuid.New(), Direction: OrderFeeReceivable, Currency: "USD"}
	p1 := &OrderFee{ID: uuid.New(), Direction: OrderFeePayable, Currency: "USD"}
	cases := []struct {
		name      string
		existing  []*OrderFee
		submitted []*OrderFee
		want      bool
	}{
		{"提交后两侧一致", []*OrderFee{r1}, []*OrderFee{{ID: uuid.New(), Direction: OrderFeePayable, Currency: "USD"}}, true},
		{"提交行取代既有行后失配", []*OrderFee{r1, p1}, []*OrderFee{{ID: p1.ID, Direction: OrderFeePayable, Currency: "EUR"}}, false},
		{"一侧为空", []*OrderFee{r1}, []*OrderFee{{ID: uuid.New(), Direction: OrderFeeReceivable, Currency: "USD"}}, false},
		{"新增币种使集合不等", []*OrderFee{r1, p1}, []*OrderFee{{ID: uuid.New(), Direction: OrderFeePayable, Currency: "EUR"}}, false},
	}
	for _, c := range cases {
		if got := OrderFeeCurrencySetsMatchAfterSave(c.existing, c.submitted); got != c.want {
			t.Errorf("%s: 得到 %v，期望 %v", c.name, got, c.want)
		}
	}
}

func TestNewOrderFeeSaveFailureMetadata(t *testing.T) {
	first, ok := NewOrderFeeSaveRowFailure("r1", ErrOrderFeeVersionConflict)
	if !ok {
		t.Fatal("业务错误应可转为行级失败")
	}
	second, _ := NewOrderFeeSaveRowFailure("r2", ErrOrderFeeTaxRateForbidden)
	err := NewOrderFeeSaveFailure([]OrderFeeSaveRowFailure{first, second})
	if !strings.Contains(err.Error(), "ORDER_FEE_VERSION_CONFLICT") {
		t.Fatalf("应保留第一条错误语义: %v", err)
	}
	if _, isRow := NewOrderFeeSaveRowFailure("r3", strings.NewReader("").UnreadByte()); isRow {
		t.Fatal("非业务错误不得伪装为行级失败")
	}
}
