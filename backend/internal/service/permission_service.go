package service

import "github.com/smartestate/smartestate/internal/constants"

type PermissionService struct{}

func NewPermissionService() *PermissionService { return &PermissionService{} }
func (s *PermissionService) Has(role, permission string) bool {
	if role == "admin" {
		return true
	}
	if role == "staff" {
		return permission == constants.PermissionRepairManage || permission == constants.PermissionPaymentManage || permission == constants.PermissionAnnouncementPublish
	}
	return false
}
