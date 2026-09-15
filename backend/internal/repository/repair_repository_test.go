package repository

import (
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestRepairRepositoryListTable(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&model.User{}, &model.Repair{})
	u := model.User{Phone: "1", Nickname: "u", Role: "resident"}
	db.Create(&u)
	db.Create(&model.Repair{UserID: u.ID, Title: "A", Description: "d", Type: "水电", Status: "pending"})
	r := NewRepairRepository(db)
	for _, tt := range []struct {
		status string
		want   int
	}{{"", 1}, {"pending", 1}, {"done", 0}} {
		got, e := r.List(tt.status)
		if e != nil || len(got) != tt.want {
			t.Fatalf("status %s got %d err %v", tt.status, len(got), e)
		}
	}
}
