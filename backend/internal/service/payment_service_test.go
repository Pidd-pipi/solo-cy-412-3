package service

import (
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"testing"
)

func TestPaymentPayTable(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&model.User{}, &model.Payment{})
	u := model.User{Phone: "2", Role: "resident"}
	db.Create(&u)
	p := model.Payment{UserID: u.ID, FeeType: "物业费", Amount: 1, Month: "2026-08", Status: "unpaid"}
	db.Create(&p)
	s := NewPaymentService(repository.NewPaymentRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, tt := range []struct {
		id   uint
		want string
	}{{p.ID, "paid"}, {p.ID, "paid"}} {
		v, e := s.Pay(tt.id, u.ID, "resident")
		if e != nil || v.Status != tt.want {
			t.Fatalf("got %s %v", v.Status, e)
		}
	}
}
