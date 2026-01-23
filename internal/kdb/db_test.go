package kdb_test

import (
	"database/sql"
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/testkit"
)

func TestCreatePrincipal(t *testing.T) {
	h := testkit.NewHarness(t)

	// 1. Create a valid principal
	p := h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "testuser",
		Instance:    "",
		Realm:       "ATHENA.MIT.EDU",
		KeyBytes:    []byte("secret_key_bytes"),
		Kvno:        1,
	})
	assert.Equal(t, p.PrimaryName, "testuser")
	assert.Equal(t, p.Kvno, int64(1))

	// 2. Duplicate constraint violation
	_, err := kdb.Query.CreatePrincipal(t.Context(), h.DB, kdb.CreatePrincipalParams{
		PrimaryName: "testuser",
		Instance:    "",
		Realm:       "ATHENA.MIT.EDU",
		KeyBytes:    []byte("other_key"),
		Kvno:        1,
	})
	// Should fail with a constraint error (sqlite3 returns non-nil)
	if err == nil {
		t.Fatal("expected error on duplicate principal, got nil")
	}

	// 3. CHECK constraint violation (empty primary name)
	_, err = kdb.Query.CreatePrincipal(t.Context(), h.DB, kdb.CreatePrincipalParams{
		PrimaryName: "",
		Instance:    "",
		Realm:       "REALM",
		KeyBytes:    []byte("key"),
		Kvno:        1,
	})
	if err == nil {
		t.Fatal("expected error on empty primary_name, got nil")
	}
}

func TestGetPrincipal(t *testing.T) {
	h := testkit.NewHarness(t)

	// Insert
	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "service",
		Instance:    "http",
		Realm:       "REALM",
		KeyBytes:    []byte("my_key"),
		Kvno:        2,
	})

	// Get - Success
	row, err := kdb.Query.GetPrincipal(t.Context(), h.DB, kdb.GetPrincipalParams{
		PrimaryName: "service",
		Instance:    "http",
		Realm:       "REALM",
	})
	assert.Err(t, err, nil)
	assert.Equal(t, string(row.KeyBytes), "my_key")
	assert.Equal(t, row.Kvno, int64(2))

	// Get - Not Found
	_, err = kdb.Query.GetPrincipal(t.Context(), h.DB, kdb.GetPrincipalParams{
		PrimaryName: "ghost",
		Instance:    "",
		Realm:       "REALM",
	})
	assert.Err(t, err, sql.ErrNoRows)
}

func TestListPrincipals(t *testing.T) {
	h := testkit.NewHarness(t)

	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "alice",
		Instance:    "",
		Realm:       "R",
		KeyBytes:    []byte("k"),
		Kvno:        1,
	})
	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "bob",
		Instance:    "",
		Realm:       "R",
		KeyBytes:    []byte("k"),
		Kvno:        1,
	})

	list, err := kdb.Query.ListPrincipals(t.Context(), h.DB)
	assert.Err(t, err, nil)

	assert.Equal(t, len(list), 2)
	assert.Equal(t, list[0].PrimaryName, "alice")
	assert.Equal(t, list[1].PrimaryName, "bob")
}
