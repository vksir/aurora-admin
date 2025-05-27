package repo

import (
	"aurora-admin/internal/entity/dstety"
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Constraint interface {
	dstety.DontStarveArchive | dstety.DontStarveWorld |
		dstety.DontStarveMod | dstety.DontStarveModInArchive |
		dstety.DontStarveAdmin
}

type Repo[T Constraint] struct {
	Db *gorm.DB
}

func NewRepo[T Constraint](db *gorm.DB) *Repo[T] {
	return &Repo[T]{Db: db}
}

// Get 查询多个数据，预加载所有关联表
func (r *Repo[T]) Get(ctx context.Context, ids []string) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}

	var ts []T
	res := r.Db.WithContext(ctx).Preload(clause.Associations).
		Where("id IN ?", ids).Find(&ts)
	return ts, res.Error
}

// GetOne 查询单个数据，预加载所有关联表
func (r *Repo[T]) GetOne(ctx context.Context, id string) (T, error) {
	var t T
	res := r.Db.WithContext(ctx).Preload(clause.Associations).
		Where("id = ?", id).Take(&t)
	return t, res.Error
}

func (r *Repo[T]) GetOneByKey(ctx context.Context, k string, v any) (T, error) {
	var t T
	res := r.Db.WithContext(ctx).Preload(clause.Associations).
		Where(fmt.Sprintf("%s = ?", k), v).Take(&t)
	return t, res.Error
}

func (r *Repo[T]) GetOneWithNestedPreload(ctx context.Context, id string, preload ...string) (T, error) {
	var t T
	db := r.Db.WithContext(ctx).Preload(clause.Associations)
	for _, pre := range preload {
		db = db.Preload(pre)
	}
	res := db.Where("id = ?", id).Take(&t)
	return t, res.Error
}

func (r *Repo[T]) Exist(ctx context.Context, id string) bool {
	_, err := r.GetOne(ctx, id)
	return !errors.Is(err, gorm.ErrRecordNotFound)
}

func (r *Repo[T]) ExistByKey(ctx context.Context, k string, v any) bool {
	_, err := r.GetOneByKey(ctx, k, v)
	return !errors.Is(err, gorm.ErrRecordNotFound)
}

// GetAll 查询所有数据，预加载所有关联表
func (r *Repo[T]) GetAll(ctx context.Context) ([]T, error) {
	var ts []T
	res := r.Db.WithContext(ctx).Preload(clause.Associations).
		Find(&ts)
	return ts, res.Error
}

// Create 创建完后，ts 中会携带最新数据
func (r *Repo[T]) Create(ctx context.Context, ts ...*T) error {
	if len(ts) == 0 {
		return nil
	}
	res := r.Db.WithContext(ctx).Create(&ts)
	return res.Error
}

// Delete 删除多个数据，删除所有关联关系，并且删除所有关联数据
func (r *Repo[T]) Delete(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	var t T
	res := r.Db.WithContext(ctx).Select(clause.Associations).Unscoped().
		Where("id IN ?", ids).Delete(&t)
	return res.Error
}

func (r *Repo[T]) Update(ctx context.Context, ts ...*T) error {
	res := r.Db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).
		Updates(ts)
	return res.Error
}
