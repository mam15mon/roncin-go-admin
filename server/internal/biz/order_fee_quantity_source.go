package biz

import (
	"github.com/google/uuid"
	"strings"
	"unicode/utf8"
)

// FeeQuantitySelection 是费用数量采用方式；数值与单位快照只由持久化层生成。
type FeeQuantitySelection struct {
	Mode                string
	ConfirmationID      uuid.UUID
	ConfirmationVersion uint64
	Reason              string
	AdoptedQuantity     string
	Unit                string
	ReferenceQuantity   string
	Source              string
	ConfirmedAt         string
}

func NormalizeFeeQuantitySelection(source *FeeQuantitySelection) (*FeeQuantitySelection, error) {
	if source == nil {
		return &FeeQuantitySelection{Mode: "MANUAL"}, nil
	}
	result := *source
	result.Reason = strings.TrimSpace(result.Reason)
	if utf8.RuneCountInString(result.Reason) > 500 {
		return nil, ErrOrderFeeInvalidArgument
	}
	switch result.Mode {
	case "MANUAL":
		if result.ConfirmationID != uuid.Nil || result.ConfirmationVersion != 0 {
			return nil, ErrOrderFeeInvalidArgument
		}
	case "DIRECT":
		if result.ConfirmationID == uuid.Nil || result.ConfirmationVersion == 0 {
			return nil, ErrOrderFeeInvalidArgument
		}
	case "MANUAL_ADJUSTED":
		if result.ConfirmationID == uuid.Nil || result.ConfirmationVersion == 0 || result.Reason == "" {
			return nil, ErrOrderFeeInvalidArgument
		}
	default:
		return nil, ErrOrderFeeInvalidArgument
	}
	result.AdoptedQuantity = ""
	result.Unit = ""
	result.ReferenceQuantity = ""
	result.Source = ""
	result.ConfirmedAt = ""
	return &result, nil
}
func FeeQuantitySelectionsEqual(a, b *FeeQuantitySelection) bool {
	if a == nil {
		a = &FeeQuantitySelection{Mode: "MANUAL"}
	}
	if b == nil {
		b = &FeeQuantitySelection{Mode: "MANUAL"}
	}
	return a.Mode == b.Mode && a.ConfirmationID == b.ConfirmationID && a.ConfirmationVersion == b.ConfirmationVersion && a.Reason == b.Reason
}
