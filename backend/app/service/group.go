package service

import (
	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
)

type IGroupService interface {
	Create(req dto.GroupCreate) error
	Update(req dto.GroupUpdate) error
	Delete(id uint) error
	GetList(groupType string) ([]dto.GroupInfo, error)
}

type GroupService struct {
	groupRepo repo.IGroupRepo
}

func NewIGroupService() IGroupService {
	return &GroupService{groupRepo: repo.NewIGroupRepo()}
}

func (s *GroupService) Create(req dto.GroupCreate) error {
	group := model.Group{Name: req.Name, Type: req.Type}
	if err := s.groupRepo.Create(&group); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

func (s *GroupService) Update(req dto.GroupUpdate) error {
	updates := map[string]interface{}{"name": req.Name}
	if err := s.groupRepo.Update(req.ID, updates); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

func (s *GroupService) Delete(id uint) error {
	return s.groupRepo.Delete(repo.WithByID(id))
}

func (s *GroupService) GetList(groupType string) ([]dto.GroupInfo, error) {
	groups, err := s.groupRepo.GetList(repo.WithByType(groupType))
	if err != nil {
		return nil, err
	}
	var items []dto.GroupInfo
	for _, g := range groups {
		items = append(items, dto.GroupInfo{ID: g.ID, Name: g.Name, Type: g.Type})
	}
	return items, nil
}
