package data

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	personnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
)

func TestOrderListResourcesPostgres(t *testing.T) {
	f := newSharedContainerFixture(t)
	ctx := context.Background()
	client, err := f.data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	shared := f.createContainer(t, "LIST")
	_, err = client.SeaSharedContainerAllocation.Create().SetOrganizationID(f.orgID).
		SetOrderID(f.order1.ID).SetHouseBillID(f.hbl1.ID).SetCargoItemID(f.cargo1ID).
		SetSharedContainerID(shared.ID).SetPackageCount(1).SetGrossWeightKg("1").SetVolumeCbm("1").Save(ctx)
	must(err)
	for _, number := range []string{shared.ContainerNo, "OWN-LIST-001"} {
		_, err = client.OrderContainer.Create().SetOrganizationID(f.orgID).SetOrderID(f.order1.ID).
			SetContainerSpecID(f.specID).SetContainerNo(number).SetPackageCount(1).SetGrossWeightKg(1).SetVolumeCbm(1).Save(ctx)
		must(err)
	}
	for _, name := range []string{"A.pdf", "B.pdf"} {
		asset, err := client.OrderAttachmentAsset.Create().SetOrganizationID(f.orgID).SetObjectKey(uuid.NewString()).SetFileName(name).SetMimeType("application/pdf").SetFileSize(1).Save(ctx)
		must(err)
		_, err = client.OrderAttachment.Create().SetOrderID(f.order1.ID).SetAssetID(asset.ID).SetDocType("OTHER").SetIdempotencyKey(uuid.NewString()).Save(ctx)
		must(err)
	}
	_, err = client.OrderPersonnel.Create().SetOrderID(f.order1.ID).SetOrganizationID(f.orgID).SetUserID(f.userID).SetRole(personnelent.RoleCUSTOMER_SERVICE).Save(ctx)
	must(err)
	_, err = client.User.UpdateOneID(f.userID).SetEnabled(false).Save(ctx)
	must(err)
	_, err = client.Order.UpdateOneID(f.order1.ID).SetBookingNotes("订舱备注").SetAllocationNotes("配舱备注").SetOperationNotes("操作备注").Save(ctx)
	must(err)
	scopes := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{f.orgID}}}
	result, err := NewOrderRepo(f.data).List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessSE, Page: 1, PageSize: 20})
	must(err)
	if len(result.Items) != 2 {
		t.Fatalf("订单数 = %d", len(result.Items))
	}
	for _, item := range result.Items {
		if item.ID == f.order1.ID {
			want := []string{"OWN-LIST-001", shared.ContainerNo}
			slices.Sort(want)
			if !slices.Equal(item.ContainerNos, want) || item.AttachmentCount == nil || *item.AttachmentCount != 2 || item.CustomerServiceName != "共享箱测试操作员" {
				t.Fatalf("本票摘要错误：箱%v 附件%v 客服%s", item.ContainerNos, item.AttachmentCount, item.CustomerServiceName)
			}
			if item.BookingNotes != "订舱备注" || item.AllocationNotes != "配舱备注" || item.OperationNotes != "操作备注" {
				t.Fatal("备注丢失")
			}
		} else if len(item.ContainerNos) != 0 || item.AttachmentCount == nil || *item.AttachmentCount != 0 {
			t.Fatalf("同主单另一票不应带入箱与附件：%#v", item)
		}
	}
	hidden, err := NewOrderRepo(f.data).List(ctx, []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{uuid.New()}}}, biz.OrderListOptions{BusinessType: biz.OrderBusinessSE, Page: 1, PageSize: 20})
	must(err)
	if len(hidden.Items) != 0 {
		t.Fatal("跨组织数据泄露")
	}
}

func TestOrderListSortPostgres(t *testing.T) {
	f := newSharedContainerFixture(t)
	ctx := context.Background()
	client, err := f.data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.Order.UpdateOneID(f.order1.ID).SetEtd("2026-12-01T10:00:00+08:00").SetEta("2026-12-02T10:00:00+08:00").Save(ctx)
	must(err)
	_, err = client.Order.UpdateOneID(f.order2.ID).SetEtd("2026-12-01T03:00:00Z").SetEta("2026-12-02T02:00:00Z").Save(ctx)
	must(err)
	// A采用共享航程01:00，B的独立航程没有时刻，沿响应规则采用本票03:00。
	_, err = client.SeaTransportExecution.UpdateOneID(f.teID).SetEtd(at.AddDate(0, 1, 0).Add(time.Hour)).Save(ctx)
	must(err)
	currentTE, err := client.SeaTransportExecution.Get(ctx, f.teID)
	must(err)
	te2, err := client.SeaTransportExecution.Create().SetOrganizationID(f.orgID).SetShippingLineID(currentTE.ShippingLineID).Save(ctx)
	must(err)
	_, err = client.SeaMasterBillOrderLink.UpdateOneID(f.link2.ID).SetTransportExecutionID(te2.ID).Save(ctx)
	must(err)
	third, err := client.Order.Create().SetIdempotencyKey(uuid.NewString()).SetOrganizationID(f.orgID).SetCustomerID(f.partnerID).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetPaymentTerm(orderent.PaymentTermPREPAID).SetOrderNo("ZZZ").SetCreatedAt(time.Now().Add(time.Hour)).Save(ctx)
	must(err)
	scopes := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{f.orgID}}}
	repo := NewOrderRepo(f.data)
	ids := func(items []*biz.Order) []uuid.UUID {
		result := make([]uuid.UUID, len(items))
		for i, item := range items {
			result[i] = item.ID
		}
		return result
	}
	numberOrder := []uuid.UUID{f.order1.ID, f.order2.ID}
	if f.order1.OrderNo > f.order2.OrderNo {
		slices.Reverse(numberOrder)
	}
	numberDesc := slices.Clone(numberOrder)
	slices.Reverse(numberDesc)
	for _, tc := range []struct {
		field, direction string
		want             []uuid.UUID
	}{
		{"etd", "ascend", []uuid.UUID{f.order1.ID, f.order2.ID, third.ID}},
		{"etd", "descend", []uuid.UUID{f.order2.ID, f.order1.ID, third.ID}},
		{"orderNo", "descend", append([]uuid.UUID{third.ID}, numberDesc...)},
		{"orderNo", "ascend", append(slices.Clone(numberOrder), third.ID)},
	} {
		result, err := repo.List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessSE, Page: 1, PageSize: 20, SortBy: tc.field, SortOrder: tc.direction})
		must(err)
		if !slices.Equal(ids(result.Items), tc.want) {
			t.Fatalf("%s %s排序错误: %v want %v", tc.field, tc.direction, ids(result.Items), tc.want)
		}
	}
	// ETA的两个不同RFC3339字符串表示同一时刻，分页按ID稳定排序，空串始终最后。
	tied := []uuid.UUID{f.order1.ID, f.order2.ID}
	slices.SortFunc(tied, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	for _, direction := range []string{"ascend", "descend"} {
		for i, want := range append(tied, third.ID) {
			result, err := repo.List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessSE, Page: i + 1, PageSize: 1, SortBy: "eta", SortOrder: direction})
			must(err)
			if len(result.Items) != 1 || result.Items[0].ID != want {
				t.Fatalf("ETA相同时刻分页不稳定：%s 第%d页 %v", direction, i+1, ids(result.Items))
			}
		}
	}
	result, err := repo.List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessSE, Page: 1, PageSize: 20})
	must(err)
	if result.Items[0].ID != third.ID {
		t.Fatal("默认排序必须创建时间倒序")
	}
}
