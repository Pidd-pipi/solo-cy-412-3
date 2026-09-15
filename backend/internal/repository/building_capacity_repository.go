package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

type BuildingCapacityRepository struct{ DB *gorm.DB }

func NewBuildingCapacityRepository(db *gorm.DB) *BuildingCapacityRepository {
	return &BuildingCapacityRepository{db}
}

// GetOrCreate 返回楼栋容量配置；未配置时回退缺省上限（不落库，避免登记新楼栋时产生隐式写入）。
// 传入 tx 时在同一事务/连接内查询，避免与外层事务争用连接。
func (r *BuildingCapacityRepository) GetOrCreate(building string, tx *gorm.DB) (model.BuildingCapacity, error) {
	var v model.BuildingCapacity
	q := r.DB
	if tx != nil {
		q = tx
	}
	e := q.Where("building = ?", building).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return model.BuildingCapacity{Building: building, DailyLimit: constants.DefaultBuildingDailyLimit}, nil
	}
	return v, e
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
