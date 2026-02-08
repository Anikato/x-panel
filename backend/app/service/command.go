package service

import (
	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

type ICommandService interface {
	Create(req dto.CommandCreate) error
	Update(req dto.CommandUpdate) error
	Delete(id uint) error
	SearchWithPage(req dto.SearchCommandReq) (int64, []dto.CommandInfo, error)
	GetTree() ([]dto.CommandTree, error)
}

type CommandService struct {
	cmdRepo   repo.ICommandRepo
	groupRepo repo.IGroupRepo
}

func NewICommandService() ICommandService {
	return &CommandService{
		cmdRepo:   repo.NewICommandRepo(),
		groupRepo: repo.NewIGroupRepo(),
	}
}

func (s *CommandService) Create(req dto.CommandCreate) error {
	cmd := model.Command{
		GroupID: req.GroupID,
		Name:    req.Name,
		Command: req.Command,
	}
	if err := s.cmdRepo.Create(&cmd); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	global.LOG.Infof("Command created: %s", req.Name)
	return nil
}

func (s *CommandService) Update(req dto.CommandUpdate) error {
	updates := map[string]interface{}{
		"group_id": req.GroupID,
		"name":     req.Name,
		"command":  req.Command,
	}
	if err := s.cmdRepo.Update(req.ID, updates); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

func (s *CommandService) Delete(id uint) error {
	return s.cmdRepo.Delete(repo.WithByID(id))
}

func (s *CommandService) SearchWithPage(req dto.SearchCommandReq) (int64, []dto.CommandInfo, error) {
	var opts []repo.DBOption
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	if req.GroupID > 0 {
		opts = append(opts, repo.WithByGroupID(req.GroupID))
	}
	total, cmds, err := s.cmdRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.CommandInfo
	for _, c := range cmds {
		items = append(items, dto.CommandInfo{
			ID:      c.ID,
			GroupID: c.GroupID,
			Name:    c.Name,
			Command: c.Command,
		})
	}
	return total, items, nil
}

func (s *CommandService) GetTree() ([]dto.CommandTree, error) {
	groups, _ := s.groupRepo.GetList(repo.WithByType("command"))
	cmds, err := s.cmdRepo.GetList()
	if err != nil {
		return nil, err
	}

	groupMap := make(map[uint][]dto.CommandInfo)
	for _, c := range cmds {
		groupMap[c.GroupID] = append(groupMap[c.GroupID], dto.CommandInfo{
			ID:      c.ID,
			GroupID: c.GroupID,
			Name:    c.Name,
			Command: c.Command,
		})
	}

	var tree []dto.CommandTree
	if items, ok := groupMap[0]; ok {
		tree = append(tree, dto.CommandTree{Label: "Default", Value: "default", Children: items})
	}
	for _, g := range groups {
		if items, ok := groupMap[g.ID]; ok {
			tree = append(tree, dto.CommandTree{ID: g.ID, Label: g.Name, Value: g.Name, Children: items})
		}
	}
	return tree, nil
}
