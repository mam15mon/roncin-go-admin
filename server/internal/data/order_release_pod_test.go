package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderreleasepodent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderreleasepod"
)

func setupTestOrderReleasePodRepo(t *testing.T) (biz.OrderReleasePodRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))
	repo := NewOrderReleasePodRepo(&Data{db: client, sqlDB: db})
	cleanup := func() {
		_ = client.Close()
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func orderRows(id, orgID uuid.UUID) *sqlmock.Rows {
	return orderRowsWithShipmentType(id, orgID, "FCL")
}

func orderRowsWithShipmentType(id, orgID uuid.UUID, shipmentType string) *sqlmock.Rows {
	now := time.Now()
	rows := sqlmock.NewRows(orderent.Columns)
	values := make([]driver.Value, len(orderent.Columns))
	for i, col := range orderent.Columns {
		switch col {
		case orderent.FieldID:
			values[i] = id
		case orderent.FieldOrganizationID:
			values[i] = orgID
		case orderent.FieldOrderNo:
			values[i] = "ORD-20260821-0001"
		case orderent.FieldBusinessType:
			values[i] = "SE"
		case orderent.FieldTradeDirection:
			values[i] = "export"
		case orderent.FieldTradeTerm:
			values[i] = "FOB"
		case orderent.FieldPaymentTerm:
			values[i] = "PREPAID"
		case orderent.FieldShipmentType:
			values[i] = shipmentType
		case orderent.FieldContainerOwnership:
			values[i] = "COC"
		case orderent.FieldShipmentMode:
			values[i] = "TRADITIONAL_FORWARDING"
		case orderent.FieldFlowStatus:
			values[i] = "CONFIRMED"
		case orderent.FieldCreatedAt, orderent.FieldUpdatedAt:
			values[i] = now
		default:
			values[i] = nil
		}
	}
	rows.AddRow(values...)
	return rows
}

func releasePodRows(id, orderID uuid.UUID, status string, signedAt *time.Time, signedBy *uuid.UUID) *sqlmock.Rows {
	now := time.Now()
	rows := sqlmock.NewRows(orderreleasepodent.Columns)
	values := make([]driver.Value, len(orderreleasepodent.Columns))
	for i, col := range orderreleasepodent.Columns {
		switch col {
		case orderreleasepodent.FieldID:
			values[i] = id
		case orderreleasepodent.FieldCreatedAt, orderreleasepodent.FieldUpdatedAt:
			values[i] = now
		case orderreleasepodent.FieldOrderID:
			values[i] = orderID
		case orderreleasepodent.FieldShippingDocumentID:
			values[i] = nil
		case orderreleasepodent.FieldReleaseNo:
			values[i] = "REL-001"
		case orderreleasepodent.FieldPodNo:
			values[i] = "POD-001"
		case orderreleasepodent.FieldStatus:
			values[i] = status
		case orderreleasepodent.FieldSignedAt:
			values[i] = signedAt
		case orderreleasepodent.FieldSignedBy:
			values[i] = signedBy
		case orderreleasepodent.FieldNote:
			values[i] = "test note"
		default:
			values[i] = nil
		}
	}
	rows.AddRow(values...)
	return rows
}

func TestOrderReleasePodRepo_Transition_PendingToSigned_Success(t *testing.T) {
	repo, mock, cleanup := setupTestOrderReleasePodRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	podID := uuid.New()
	actorID := uuid.New()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.release_pod.transition",
		Result:         "success",
		RequestID:      "req-123",
		TraceID:        "trace-123",
		IPAddress:      "127.0.0.1",
		Details: map[string]string{
			"from_status": string(biz.OrderReleasePodStatusPending),
			"to_status":   string(biz.OrderReleasePodStatusSigned),
		},
	}

	// 1. 校验所属订单存在
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "orders"."id"`)).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 2. 开启事务
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id".*FOR UPDATE`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 3. ForUpdate 查询当前放货凭证
	mock.ExpectQuery(`SELECT "order_release_pods"\."id".*FOR UPDATE`).
		WithArgs(podID, orderID).
		WillReturnRows(releasePodRows(podID, orderID, "PENDING", nil, nil))

	// 4. 执行更新 SQL：status -> SIGNED, signed_at, signed_by
	signedAt := time.Now()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "order_release_pods" SET`)).
		WithArgs(
			sqlmock.AnyArg(), // updated_at
			"SIGNED",         // status
			sqlmock.AnyArg(), // signed_at
			actorID,          // signed_by
			podID,            // id
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 5. Ent 更新后重新加载实体
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id", "created_at", "updated_at", "order_id", "shipping_document_id", "sea_master_bill_id", "sea_house_bill_id", "release_no", "pod_no", "status", "signed_at", "signed_by", "note" FROM "order_release_pods" WHERE "id" = $1`)).
		WithArgs(podID).
		WillReturnRows(releasePodRows(podID, orderID, "SIGNED", &signedAt, &actorID))

	// 6. 写入审计日志 SQL
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WithArgs(
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			orgID,            // organization_id
			actorID,          // user_id
			audit.Action,     // action
			"success",        // result
			audit.RequestID,  // request_id
			audit.TraceID,    // trace_id
			audit.IPAddress,  // ip_address
			sqlmock.AnyArg(), // details json
			sqlmock.AnyArg(), // id
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 7. 提交事务
	mock.ExpectCommit()

	result, err := repo.Transition(
		context.Background(),
		orgID,
		orderID,
		podID,
		biz.OrderReleasePodStatusPending,
		biz.OrderReleasePodStatusSigned,
		actorID,
		audit,
	)
	if err != nil {
		t.Fatalf("Transition 期望成功，实际返回错误: %v", err)
	}
	if result == nil {
		t.Fatal("Transition 期望返回实体，实际为 nil")
	}
	if result.Status != biz.OrderReleasePodStatusSigned {
		t.Fatalf("期望状态为 SIGNED，实际为 %s", result.Status)
	}
	if result.SignedAt == nil {
		t.Fatal("期望 SignedAt 已持久化填充，实际为 nil")
	}
	if result.SignedBy == nil || *result.SignedBy != actorID {
		t.Fatalf("期望 SignedBy 为 %s，实际为 %v", actorID, result.SignedBy)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func TestOrderReleasePodRepo_Transition_BusinessSQLError_Rollback(t *testing.T) {
	repo, mock, cleanup := setupTestOrderReleasePodRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	podID := uuid.New()
	actorID := uuid.New()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.release_pod.transition",
		Result:         "success",
	}

	// 1. 校验所属订单存在
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "orders"."id"`)).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 2. 开启事务
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id".*FOR UPDATE`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 3. ForUpdate 查询当前放货凭证
	mock.ExpectQuery(`SELECT "order_release_pods"\."id".*FOR UPDATE`).
		WithArgs(podID, orderID).
		WillReturnRows(releasePodRows(podID, orderID, "PENDING", nil, nil))

	// 4. 更新业务 SQL 发生数据库错误
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "order_release_pods" SET`)).
		WithArgs(
			sqlmock.AnyArg(),
			"SIGNED",
			sqlmock.AnyArg(),
			actorID,
			podID,
		).
		WillReturnError(errors.New("db update failed"))

	// 5. 预期事务回滚
	mock.ExpectRollback()

	result, err := repo.Transition(
		context.Background(),
		orgID,
		orderID,
		podID,
		biz.OrderReleasePodStatusPending,
		biz.OrderReleasePodStatusSigned,
		actorID,
		audit,
	)
	if err == nil {
		t.Fatal("期望业务 SQL 错误时返回 error，实际为 nil")
	}
	if result != nil {
		t.Fatalf("期望失败时返回 nil 实体，实际为 %#v", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func TestOrderReleasePodRepo_Transition_AuditSQLError_Rollback(t *testing.T) {
	repo, mock, cleanup := setupTestOrderReleasePodRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	podID := uuid.New()
	actorID := uuid.New()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.release_pod.transition",
		Result:         "success",
		RequestID:      "req-123",
		TraceID:        "trace-123",
		IPAddress:      "127.0.0.1",
		Details: map[string]string{
			"from_status": string(biz.OrderReleasePodStatusPending),
			"to_status":   string(biz.OrderReleasePodStatusSigned),
		},
	}

	// 1. 校验所属订单存在
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "orders"."id"`)).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 2. 开启事务
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id".*FOR UPDATE`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 3. ForUpdate 查询当前放货凭证
	mock.ExpectQuery(`SELECT "order_release_pods"\."id".*FOR UPDATE`).
		WithArgs(podID, orderID).
		WillReturnRows(releasePodRows(podID, orderID, "PENDING", nil, nil))

	// 4. 业务更新成功
	signedAt := time.Now()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "order_release_pods" SET`)).
		WithArgs(
			sqlmock.AnyArg(),
			"SIGNED",
			sqlmock.AnyArg(),
			actorID,
			podID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 5. Ent 更新后重新加载实体
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id", "created_at", "updated_at", "order_id", "shipping_document_id", "sea_master_bill_id", "sea_house_bill_id", "release_no", "pod_no", "status", "signed_at", "signed_by", "note" FROM "order_release_pods" WHERE "id" = $1`)).
		WithArgs(podID).
		WillReturnRows(releasePodRows(podID, orderID, "SIGNED", &signedAt, &actorID))

	// 6. 写入审计日志 SQL 发生错误
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			orgID,
			actorID,
			audit.Action,
			"success",
			audit.RequestID,
			audit.TraceID,
			audit.IPAddress,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnError(errors.New("db audit insert failed"))

	// 7. 预期事务回滚
	mock.ExpectRollback()

	result, err := repo.Transition(
		context.Background(),
		orgID,
		orderID,
		podID,
		biz.OrderReleasePodStatusPending,
		biz.OrderReleasePodStatusSigned,
		actorID,
		audit,
	)
	if err == nil {
		t.Fatal("期望审计 SQL 错误时返回 error，实际为 nil")
	}
	if result != nil {
		t.Fatalf("期望失败时返回 nil 实体，实际为 %#v", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func TestOrderReleasePodRepo_Transition_StatusConflict_Rollback(t *testing.T) {
	repo, mock, cleanup := setupTestOrderReleasePodRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	podID := uuid.New()
	actorID := uuid.New()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.release_pod.transition",
		Result:         "success",
	}

	// 1. 校验所属订单存在
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "orders"."id"`)).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 2. 开启事务
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id".*FOR UPDATE`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 3. ForUpdate 查询到当前状态为 RETURNED（与期望的 from=PENDING 冲突）
	mock.ExpectQuery(`SELECT "order_release_pods"\."id".*FOR UPDATE`).
		WithArgs(podID, orderID).
		WillReturnRows(releasePodRows(podID, orderID, "RETURNED", nil, nil))

	// 4. 预期事务回滚
	mock.ExpectRollback()

	result, err := repo.Transition(
		context.Background(),
		orgID,
		orderID,
		podID,
		biz.OrderReleasePodStatusPending,
		biz.OrderReleasePodStatusSigned,
		actorID,
		audit,
	)
	if !errors.Is(err, biz.ErrOrderReleasePodStatusConflict) {
		t.Fatalf("期望返回 ErrOrderReleasePodStatusConflict，实际为: %v", err)
	}
	if result != nil {
		t.Fatalf("期望失败时返回 nil 实体，实际为 %#v", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func TestOrderReleasePodRepo_Transition_NotFound_Rollback(t *testing.T) {
	repo, mock, cleanup := setupTestOrderReleasePodRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	podID := uuid.New()
	actorID := uuid.New()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.release_pod.transition",
		Result:         "success",
	}

	// 1. 校验所属订单存在
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "orders"."id"`)).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 2. 开启事务
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id".*FOR UPDATE`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// 3. ForUpdate 查询未找到记录
	mock.ExpectQuery(`SELECT "order_release_pods"\."id".*FOR UPDATE`).
		WithArgs(podID, orderID).
		WillReturnRows(sqlmock.NewRows(orderreleasepodent.Columns))

	// 4. 预期事务回滚
	mock.ExpectRollback()

	result, err := repo.Transition(
		context.Background(),
		orgID,
		orderID,
		podID,
		biz.OrderReleasePodStatusPending,
		biz.OrderReleasePodStatusSigned,
		actorID,
		audit,
	)
	if !errors.Is(err, biz.ErrOrderReleasePodNotFound) {
		t.Fatalf("期望返回 ErrOrderReleasePodNotFound，实际为: %v", err)
	}
	if result != nil {
		t.Fatalf("期望失败时返回 nil 实体，实际为 %#v", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}
