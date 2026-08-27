package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

const revisionTableExistsQuery = "SELECT to_regclass(current_schema() || '.schema_migrations') IS NOT NULL"

func TestReadFilesSortsAndChecksums(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "20260821150000_second.sql"), []byte("SELECT 2;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "20260821140000_first.sql"), []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := readFiles(dir)
	if err != nil {
		t.Fatalf("readFiles() error = %v", err)
	}
	if files[0].version != "20260821140000_first" || files[1].version != "20260821150000_second" {
		t.Fatalf("files = %#v", files)
	}
	hash := sha256.Sum256([]byte("SELECT 1;"))
	if files[0].checksum != hex.EncodeToString(hash[:]) {
		t.Fatalf("checksum = %s", files[0].checksum)
	}
}

func TestApplyExecutesNewMigrationInTransaction(t *testing.T) {
	dir := t.TempDir()
	statement := "CREATE TABLE test_table (id uuid PRIMARY KEY);"
	name := "20260821140000_create_test.sql"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(statement), 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(statement))
	checksum := hex.EncodeToString(hash[:])
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(revisionTableExistsQuery)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT "version", "checksum"`).WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(statement)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO "schema_migrations"`).WithArgs("20260821140000_create_test", checksum).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyCreatesRevisionTableIfNotExists(t *testing.T) {
	dir := t.TempDir()
	statement := "CREATE TABLE test_table (id uuid PRIMARY KEY);"
	name := "20260821140000_create_test.sql"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(statement), 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(statement))
	checksum := hex.EncodeToString(hash[:])
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(revisionTableExistsQuery)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`CREATE TABLE "schema_migrations"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT "version", "checksum"`).WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(statement)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO "schema_migrations"`).WithArgs("20260821140000_create_test", checksum).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRejectsChangedMigration(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "20260821140000_create_test.sql"), []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(revisionTableExistsQuery)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT "version", "checksum"`).WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}).AddRow("20260821140000_create_test", "different"))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Apply(context.Background(), db, dir); err == nil {
		t.Fatal("Apply() error = nil, want checksum mismatch")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompatibleChecksumOnlyAcceptsRegisteredHistoricalRepair(t *testing.T) {
	const oldChecksum = "d50b2a09d9b4d640285f3abb43d2d9ed05e7c701a1296363b7ab3c333cc6617c"
	const orderFeeOldChecksum = "eec00e191b2ff7429c7469316f2b2cbc3cd77f7c98ecb2fb6373b53c4c96989a"
	if !isCompatibleChecksum("20260824043000_global_exchange_rates", oldChecksum) {
		t.Fatal("已登记的共享汇率迁移旧校验和应被接受")
	}
	if !isCompatibleChecksum("20260826150000_order_fee_finance_foundation", orderFeeOldChecksum) {
		t.Fatal("已登记的费用基础迁移旧校验和应被接受")
	}
	if isCompatibleChecksum("20260824043000_global_exchange_rates", "unknown") {
		t.Fatal("未知共享汇率迁移校验和不应被接受")
	}
	if isCompatibleChecksum("20260821140000_create_test", oldChecksum) {
		t.Fatal("兼容校验和不应扩散到其他迁移")
	}
}

func TestApplyRollsBackFailedMigration(t *testing.T) {
	dir := t.TempDir()
	statement := "CREATE TABLE broken;"
	if err := os.WriteFile(filepath.Join(dir, "20260821140000_broken.sql"), []byte(statement), 0o600); err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(revisionTableExistsQuery)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT "version", "checksum"`).WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(statement)).WillReturnError(errors.New("invalid SQL"))
	mock.ExpectRollback()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Apply(context.Background(), db, dir); err == nil {
		t.Fatal("Apply() error = nil, want SQL error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyValidatesCompleteHistoryBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "20260821150000_new.sql"), []byte("SELECT 2;"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(revisionTableExistsQuery)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT "version", "checksum"`).WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}).AddRow("20260821140000_missing", "checksum"))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Apply(context.Background(), db, dir); err == nil {
		t.Fatal("Apply() error = nil, want missing history error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRejectsNonLinearMigration(t *testing.T) {
	dir := t.TempDir()
	oldStatement := []byte("SELECT 1;")
	newStatement := []byte("SELECT 2;")
	if err := os.WriteFile(filepath.Join(dir, "20260821140000_inserted.sql"), oldStatement, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "20260821150000_applied.sql"), newStatement, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(newStatement)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(revisionTableExistsQuery)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT "version", "checksum"`).WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}).AddRow("20260821150000_applied", hex.EncodeToString(hash[:])))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Apply(context.Background(), db, dir); err == nil {
		t.Fatal("Apply() error = nil, want non-linear migration error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
