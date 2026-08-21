package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

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
	mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass('public.schema_migrations') IS NOT NULL")).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
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
	mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass('public.schema_migrations') IS NOT NULL")).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT "version", "checksum"`).WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}).AddRow("20260821140000_create_test", "different"))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).WithArgs(advisoryLockKey).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Apply(context.Background(), db, dir); err == nil {
		t.Fatal("Apply() error = nil, want checksum mismatch")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
