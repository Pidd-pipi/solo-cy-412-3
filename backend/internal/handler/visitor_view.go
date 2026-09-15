package handler

import (
	"time"

	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/util"
)

func mapUser(u model.User) dto.PassUserView {
	return dto.PassUserView{ID: u.ID, Nickname: u.Nickname, Phone: u.Phone}
}

func mapUserPtr(u *model.User) *dto.PassUserView {
	if u == nil || u.ID == 0 {
		return nil
	}
	v := mapUser(*u)
	return &v
}

// optVisit 把可空时刻格式化为社区钟面串，nil/零值返回空串。
func optVisit(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return util.FormatVisit(*t)
}

// mapPass 把凭证模型转为响应视图：所有时刻统一为社区时区钟面串，
// 使业主、物业、门岗在任意浏览器时区下看到同一本地时刻；旧记录按原值渲染。
func mapPass(p model.VisitorPass) dto.PassView {
	return dto.PassView{
		ID:             p.ID,
		PassNo:         p.PassNo,
		ResidentID:     p.ResidentID,
		Resident:       mapUser(p.Resident),
		VisitorName:    p.VisitorName,
		VisitorPhone:   p.VisitorPhone,
		Building:       p.Building,
		Reason:         p.Reason,
		StartTime:      util.FormatVisit(p.StartTime),
		EndTime:        util.FormatVisit(p.EndTime),
		Status:         p.Status,
		ReviewerID:     p.ReviewerID,
		Reviewer:       mapUserPtr(p.Reviewer),
		ReviewRemark:   p.ReviewRemark,
		ReviewedAt:     optVisit(p.ReviewedAt),
		CheckInAt:      optVisit(p.CheckInAt),
		CheckOutAt:     optVisit(p.CheckOutAt),
		Checkpoint:     p.Checkpoint,
		ExpireMarkedAt: optVisit(p.ExpireMarkedAt),
		CreatedAt:      util.FormatVisit(p.CreatedAt),
	}
}

func mapPasses(ps []model.VisitorPass) []dto.PassView {
	out := make([]dto.PassView, 0, len(ps))
	for _, p := range ps {
		out = append(out, mapPass(p))
	}
	return out
}

func mapEvents(es []model.VisitorEvent) []dto.PassEventView {
	out := make([]dto.PassEventView, 0, len(es))
	for _, e := range es {
		out = append(out, dto.PassEventView{
			ID:         e.ID,
			PassID:     e.PassID,
			Action:     e.Action,
			ActorID:    e.ActorID,
			Actor:      mapUserPtr(e.Actor),
			FromStatus: e.FromStatus,
			ToStatus:   e.ToStatus,
			Detail:     e.Detail,
			CreatedAt:  util.FormatVisit(e.CreatedAt),
		})
	}
	return out
}
