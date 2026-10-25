package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func seed(t testing.TB, db *DB, archives, modsPerArchive int) {
	t.Helper()
	ctx := context.Background()
	for a := 0; a < archives; a++ {
		var mods []*Mod
		for m := 0; m < modsPerArchive; m++ {
			mod, err := db.CreateMod(ctx, &Mod{WorkshopID: fmt.Sprintf("%d-%d", a, m)})
			assert.Nil(t, err)
			mods = append(mods, mod)
		}
		_, err := db.CreateArchive(ctx, &Archive{Remark: "bench", Mods: mods})
		assert.Nil(t, err)
	}
}

func benchDB(b *testing.B) *DB {
	b.Helper()
	db, err := Open(fmt.Sprintf("%s/bench.db", b.TempDir()))
	assert.Nil(b, err)
	assert.Nil(b, db.Migrate(context.Background()))
	return db
}

// 基准：单条 JOIN 查询（优化后 ListArchives 实现）。
func BenchmarkListArchivesJoin(b *testing.B) {
	db := benchDB(b)
	defer db.Close()
	seed(b, db, 50, 5)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.ListArchives(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// 基准：逐个 GetArchive 模拟优化前的 N+1 行为，用于对比。
func BenchmarkListArchivesNPlus1(b *testing.B) {
	db := benchDB(b)
	defer db.Close()
	seed(b, db, 50, 5)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := db.Query(ctx, `SELECT id FROM dont_starve_archives`)
		if err != nil {
			b.Fatal(err)
		}
		var ids []string
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			if _, err := db.GetArchive(ctx, id); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// 基准：写入吞吐对比 —— WAL+synchronous(NORMAL)（新 DSN） vs 默认 journal（旧 DSN）。
func benchmarkWrites(b *testing.B, dsn string) {
	db, err := openDSN(dsn)
	assert.Nil(b, err)
	defer db.Close()
	assert.Nil(b, db.Migrate(context.Background()))

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.CreateArchive(ctx, &Archive{Remark: "bench"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDBWritesWAL(b *testing.B) {
	benchmarkWrites(b, fmt.Sprintf("file:%s/wal.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(1)", b.TempDir()))
}

func BenchmarkDBWritesLegacyJournal(b *testing.B) {
	benchmarkWrites(b, fmt.Sprintf("file:%s/legacy.db?_pragma=foreign_keys(1)", b.TempDir()))
}
