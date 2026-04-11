package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLiteStoreWriteSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO audit_entries").
		WithArgs("id-1", "cli", "sync", "org/repo", "ok", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := NewSQLiteStore(db)
	err = s.Write(context.Background(), Entry{
		ID: "id-1", Actor: "cli", Action: "sync",
		Target: "org/repo", Outcome: "ok", CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteStoreWriteRejectsInvalid(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	s := NewSQLiteStore(db)
	if err := s.Write(context.Background(), Entry{}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSQLiteStoreWriteMarshalError(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	orig := marshalMetadata
	marshalMetadata = func(any) ([]byte, error) { return nil, errors.New("marshal boom") }
	defer func() { marshalMetadata = orig }()

	s := NewSQLiteStore(db)
	err := s.Write(context.Background(), Entry{Actor: "cli", Action: "sync"})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestSQLiteStoreWriteWrapsDBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectExec("INSERT INTO audit_entries").
		WillReturnError(errors.New("boom"))

	s := NewSQLiteStore(db)
	err := s.Write(context.Background(), Entry{Actor: "cli", Action: "sync"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSQLiteStoreListReturnsEntries(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	rows := sqlmock.NewRows([]string{"id", "actor", "action", "target", "outcome", "metadata", "created_at"}).
		AddRow("id-1", "cli", "sync", "org/repo", "ok", `{"k":"v"}`, time.Now())
	mock.ExpectQuery("SELECT (.+) FROM audit_entries").
		WithArgs(10).
		WillReturnRows(rows)

	s := NewSQLiteStore(db)
	entries, err := s.List(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Actor != "cli" {
		t.Fatalf("got %+v", entries)
	}
	if entries[0].Metadata["k"] != "v" {
		t.Fatalf("metadata not parsed: %+v", entries[0].Metadata)
	}
}

func TestSQLiteStoreListDefaultsLimit(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	rows := sqlmock.NewRows([]string{"id", "actor", "action", "target", "outcome", "metadata", "created_at"})
	mock.ExpectQuery("SELECT (.+) FROM audit_entries").
		WithArgs(100).
		WillReturnRows(rows)

	s := NewSQLiteStore(db)
	if _, err := s.List(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteStoreListQueryError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("SELECT (.+) FROM audit_entries").
		WillReturnError(errors.New("boom"))

	s := NewSQLiteStore(db)
	if _, err := s.List(context.Background(), 5); err == nil {
		t.Fatal("expected error")
	}
}

func TestSQLiteStoreListScanError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	// Return wrong column count to force Scan error.
	rows := sqlmock.NewRows([]string{"id"}).AddRow("id-1")
	mock.ExpectQuery("SELECT (.+) FROM audit_entries").
		WillReturnRows(rows)

	s := NewSQLiteStore(db)
	if _, err := s.List(context.Background(), 5); err == nil {
		t.Fatal("expected scan error")
	}
}

func TestSQLiteStoreListRowsErr(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	rows := sqlmock.NewRows([]string{"id", "actor", "action", "target", "outcome", "metadata", "created_at"}).
		AddRow("id-1", "cli", "sync", "org/repo", "ok", "", time.Now()).
		RowError(0, errors.New("row boom"))
	mock.ExpectQuery("SELECT (.+) FROM audit_entries").
		WillReturnRows(rows)

	s := NewSQLiteStore(db)
	if _, err := s.List(context.Background(), 5); err == nil {
		t.Fatal("expected rows error")
	}
}

func TestSQLiteStoreCloseIsNoOp(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	s := NewSQLiteStore(db)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
