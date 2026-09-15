package repository

import (
	"errors"

	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BuildingCapacityRepository struct{ DB *gorm.DB }

func NewBuildingCapacityRepository(db *gorm.DB) *BuildingCapacityRepository {
	return &BuildingCapacityRepository{db}
}

// EnsureLock 在调用方事务内确保该楼栋存在容量行，并对该行加写锁（MySQL 为 SELECT … FOR UPDATE）。
// 同一楼栋的登记/审核/进入/离开都先获取此锁，从而在数据库层串行化，杜绝"读后判"并发竞态。
// SQLite 不支持行锁，FOR UPDATE 被方言忽略，其单写者语义配合 busy_timeout 同样保证串行。
func (r *BuildingCapacityRepository) EnsureLock(tx *gorm.DB, building string, defaultLimit int) (model.BuildingCapacity, error) {
	seed := model.BuildingCapacity{Building: building, DailyLimit: defaultLimit}
	if e := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "building"}},
		DoNothing: true,
	}).Create(&seed).Error; e != nil {
		return seed, e
	}
	q := tx.Where("building = ?", building)
	if r.DB.Dialector.Name() == "mysql" {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var cfg model.BuildingCapacity
	if e := q.First(&cfg).Error; e != nil {
		return cfg, e
	}
	return cfg, nil
}

// Upsert 物业调整楼栋当日容量上限。
func (r *BuildingCapacityRepository) Upsert(building string, limit int, operatorID uint) (model.BuildingCapacity, error) {
	var v model.BuildingCapacity
	e := r.DB.Where("building = ?", building).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		v = model.BuildingCapacity{Building: building, DailyLimit: limit, UpdatedBy: operatorID}
		if e = r.DB.Create(&v).Error; e != nil {
			return v, e
		}
		return v, nil
	}
	if e != nil {
		return v, e
	}
	v.DailyLimit = limit
	v.UpdatedBy = operatorID
	if e = r.DB.Save(&v).Error; e != nil {
		return v, e
	}
	return v, nil
}

func (r *BuildingCapacityRepository) List() (out []model.BuildingCapacity, e error) {
	e = r.DB.Order("building asc").Find(&out).Error
	return
}
