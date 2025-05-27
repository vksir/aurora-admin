package database

import (
	"aurora-admin/internal/entity/dstety"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

var DB *gorm.DB

var migrate = []any{
	&dstety.DontStarveArchive{},
	&dstety.DontStarveWorld{},
	&dstety.DontStarveMod{},
	&dstety.DontStarveModInArchive{},
}

func Init(dbPath string) {
	sql := sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath,
	}

	db, err := gorm.Open(sql, &gorm.Config{})
	errutil.Check(err)
	DB = db

	err = DB.AutoMigrate(migrate...)
	errutil.Check(err)
}
