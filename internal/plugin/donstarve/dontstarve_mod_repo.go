package donstarve

import (
	"aurora-admin/internal/entity/dstety"
	"context"
	"gorm.io/gorm"
)

type DontStarveModRepo struct {
	*gorm.DB
}

func NewDontStarveModRepo(db *gorm.DB) *DontStarveModRepo {
	return &DontStarveModRepo{db}
}

func (r *DontStarveModRepo) GetMod(ctx context.Context, id string) (dstety.DontStarveMod, error) {
	var e dstety.DontStarveMod
	e.ID = id
	res := r.WithContext(ctx).Take(&e)
	return e, res.Error
}

func (r *DontStarveModRepo) GetMods(ctx context.Context) ([]dstety.DontStarveMod, error) {
	var entities []dstety.DontStarveMod
	res := r.WithContext(ctx).Find(&entities)
	return entities, res.Error
}

func (r *DontStarveModRepo) GetModsByModIds(ctx context.Context, modIds []string) ([]dstety.DontStarveMod, error) {
	if len(modIds) == 0 {
		return []dstety.DontStarveMod{}, nil
	}

	var mods []dstety.DontStarveMod
	res := r.WithContext(ctx).
		Where("mod_id IN ?", modIds).Find(&mods)
	return mods, res.Error
}

func (r *DontStarveModRepo) CreateMods(ctx context.Context, mods []dstety.DontStarveMod) error {
	res := r.WithContext(ctx).Create(&mods)
	return res.Error
}

func (r *DontStarveModRepo) DeleteModsByModIds(ctx context.Context, modIds []string) error {
	if len(modIds) == 0 {
		return nil
	}

	res := r.WithContext(ctx).
		Where("mod_id IN ?", modIds).Delete(&dstety.DontStarveMod{})
	return res.Error
}

func (r *DontStarveModRepo) UpdateMods(ctx context.Context, mods []dstety.DontStarveMod) error {
	res := r.WithContext(ctx).Updates(&mods)
	return res.Error
}
