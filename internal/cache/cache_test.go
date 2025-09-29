package cache

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

// helper function to create a temporary directory and return its path
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cache_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

func TestCache(t *testing.T) {
	t.Run("New_Set_Get_Close", func(t *testing.T) {
		dir := tempDir(t)
		defer os.RemoveAll(dir)
		dbPath := filepath.Join(dir, "test.db")

		// 1. Test New
		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if c == nil {
			t.Fatal("New() returned nil cache")
		}
		defer c.Close()

		// 2. Test Set
		n := uint64(100)
		val := big.NewInt(3542248481792619150) // F(100)
		err = c.Set(n, val)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// 3. Test Get (existing key)
		retrievedVal, found, err := c.Get(n)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if !found {
			t.Fatal("Get() should have found the value, but it didn't")
		}
		if retrievedVal.Cmp(val) != 0 {
			t.Errorf("Get() retrieved = %v, want %v", retrievedVal, val)
		}

		// 4. Test Get (non-existent key)
		_, found, err = c.Get(999)
		if err != nil {
			t.Fatalf("Get() for non-existent key error = %v", err)
		}
		if found {
			t.Error("Get() should not have found a value for a non-existent key, but it did")
		}
	})

	t.Run("Persistence", func(t *testing.T) {
		dir := tempDir(t)
		defer os.RemoveAll(dir)
		dbPath := filepath.Join(dir, "test.db")

		// Create cache, set a value, and close it
		c1, err := New(dbPath)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		n := uint64(200)
		val := big.NewInt(2805711729925101400) // F(200), a different value
		err = c1.Set(n, val)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if err := c1.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		// Re-open the cache and check if the value is still there
		c2, err := New(dbPath)
		if err != nil {
			t.Fatalf("New() on existing db error = %v", err)
		}
		defer c2.Close()

		retrievedVal, found, err := c2.Get(n)
		if err != nil {
			t.Fatalf("Get() after reopen error = %v", err)
		}
		if !found {
			t.Fatal("Get() after reopen should have found the value, but it didn't")
		}
		if retrievedVal.Cmp(val) != 0 {
			t.Errorf("Get() after reopen retrieved = %v, want %v", retrievedVal, val)
		}
	})
}