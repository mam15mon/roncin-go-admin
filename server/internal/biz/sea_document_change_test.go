package biz

import (
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

func validAmendmentCommand() *SeaDocumentAmendmentCommand {
	value := "更新后的发货人"
	return &SeaDocumentAmendmentCommand{
		OrderID:                  uuid.New(),
		DocumentType:             SeaDocumentTypeMasterBill,
		DocumentID:               uuid.New(),
		ExpectedOrderVersion:     3,
		ExpectedDocumentVersion:  2,
		ExpectedCurrentVersionID: uuid.New(),
		Reason:                   " 客户书面更正 ",
		IdempotencyKey:           "amend-001",
		Input:                    &SeaDocumentAmendmentInput{MasterBillContent: &SeaBillContent{ShipperText: &value}},
	}
}

func TestValidateSeaDocumentChangeCommands(t *testing.T) {
	t.Run("Preview 允许无幂等键但规范化原因", func(t *testing.T) {
		cmd := validAmendmentCommand()
		cmd.IdempotencyKey = ""
		got, err := validateAmendmentCommand(cmd, false)
		if err != nil || got.Reason != "客户书面更正" || got.IdempotencyKey != "" {
			t.Fatalf("Preview 命令校验失败: got=%+v err=%v", got, err)
		}
	})

	t.Run("Execute 强制原因和幂等键", func(t *testing.T) {
		for name, mutate := range map[string]func(*SeaDocumentAmendmentCommand){
			"空原因":  func(cmd *SeaDocumentAmendmentCommand) { cmd.Reason = "  " },
			"空幂等键": func(cmd *SeaDocumentAmendmentCommand) { cmd.IdempotencyKey = "" },
		} {
			t.Run(name, func(t *testing.T) {
				cmd := validAmendmentCommand()
				mutate(cmd)
				_, err := validateAmendmentCommand(cmd, true)
				if !kratoserrors.IsBadRequest(err) {
					t.Fatalf("应返回参数错误，实际 %v", err)
				}
			})
		}
	})

	t.Run("Execute 强制 expected version 和当前不可变版本", func(t *testing.T) {
		cmd := validAmendmentCommand()
		cmd.ExpectedOrderVersion = 0
		if _, err := validateAmendmentCommand(cmd, true); !kratoserrors.IsBadRequest(err) {
			t.Fatalf("expected_order_version=0 应被拒绝: %v", err)
		}
		cmd = validAmendmentCommand()
		cmd.ExpectedCurrentVersionID = uuid.Nil
		if _, err := validateAmendmentCommand(cmd, true); !kratoserrors.IsBadRequest(err) {
			t.Fatalf("空当前不可变版本应被拒绝: %v", err)
		}
	})

	t.Run("Switch 校验真实新 HBL", func(t *testing.T) {
		cmd := &SeaHouseBillSwitchCommand{
			OrderID: uuid.New(), OldHouseBillID: uuid.New(), ExpectedOrderVersion: 1,
			ExpectedHouseBillVersion: 1, ExpectedCurrentVersionID: uuid.New(),
			Reason: "换单", IdempotencyKey: "switch-001",
			NewHouseBill: &SeaHouseBillInput{HouseNo: " hbl-001 ", IssuerSource: SeaHouseBillIssuerSourceCustomerPartner},
		}
		got, err := validateSwitchCommand(cmd, true)
		if err != nil || got.NewHouseBill.HouseNo != " hbl-001 " {
			t.Fatalf("Switch 命令校验失败: got=%+v err=%v", got, err)
		}
		cmd.NewHouseBill = nil
		if _, err = validateSwitchCommand(cmd, true); !kratoserrors.IsBadRequest(err) {
			t.Fatalf("缺少新 HBL 应被拒绝: %v", err)
		}
	})
}
