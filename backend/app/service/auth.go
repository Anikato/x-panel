package service

import (
	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/utils/encrypt"
	jwtUtil "xpanel/utils/jwt"
)

// IAuthService 认证服务接口
type IAuthService interface {
	Login(info dto.Login) (*dto.UserLoginInfo, error)
	InitUser(info dto.InitUser) error
	IsInitialized() bool
	UpdatePassword(userName string, info dto.PasswordUpdate) error
	GetLoginSetting() (*dto.LoginSetting, error)
}

// NewIAuthService 创建认证服务实例
func NewIAuthService() IAuthService {
	return &AuthService{}
}

type AuthService struct{}

var settingRepo = repo.NewISettingRepo()
var logRepo = repo.NewILogRepo()

func (a *AuthService) Login(info dto.Login) (*dto.UserLoginInfo, error) {
	// 校验用户名
	nameSetting, err := settingRepo.Get(repo.WithByKey("UserName"))
	if err != nil {
		return nil, buserr.New(constant.ErrUserNotFound)
	}
	if nameSetting.Value != info.Name {
		return nil, buserr.New(constant.ErrAuth)
	}

	// 校验密码
	passwordSetting, err := settingRepo.Get(repo.WithByKey("Password"))
	if err != nil {
		return nil, buserr.New(constant.ErrAuth)
	}
	if passwordSetting.Value == "" {
		return nil, buserr.New(constant.ErrInitialPassword)
	}
	if !encrypt.CheckPassword(info.Password, passwordSetting.Value) {
		return nil, buserr.New(constant.ErrAuth)
	}

	// 检查 MFA 状态
	mfaSetting, _ := settingRepo.Get(repo.WithByKey("MFAStatus"))
	if mfaSetting.Value == constant.StatusEnable {
		return &dto.UserLoginInfo{
			Name:      nameSetting.Value,
			MfaStatus: mfaSetting.Value,
		}, nil
	}

	// 生成 JWT Token
	token, err := jwtUtil.GenerateToken(info.Name)
	if err != nil {
		return nil, buserr.WithErr(constant.ErrInternalServer, err)
	}

	return &dto.UserLoginInfo{
		Name:  nameSetting.Value,
		Token: token,
	}, nil
}

func (a *AuthService) InitUser(info dto.InitUser) error {
	// 检查是否已初始化
	passwordSetting, err := settingRepo.Get(repo.WithByKey("Password"))
	if err != nil {
		return err
	}
	if passwordSetting.Value != "" {
		return buserr.New(constant.ErrRecordExist)
	}

	// 哈希密码
	hashed, err := encrypt.HashPassword(info.Password)
	if err != nil {
		return buserr.WithErr(constant.ErrInternalServer, err)
	}

	// 更新用户名和密码
	if err := settingRepo.Update("UserName", info.Name); err != nil {
		return err
	}
	return settingRepo.Update("Password", hashed)
}

func (a *AuthService) IsInitialized() bool {
	passwordSetting, err := settingRepo.Get(repo.WithByKey("Password"))
	if err != nil {
		return false
	}
	return passwordSetting.Value != ""
}

func (a *AuthService) UpdatePassword(userName string, info dto.PasswordUpdate) error {
	// 校验旧密码
	passwordSetting, err := settingRepo.Get(repo.WithByKey("Password"))
	if err != nil {
		return err
	}
	if !encrypt.CheckPassword(info.OldPassword, passwordSetting.Value) {
		return buserr.New(constant.ErrPasswordWrong)
	}

	// 哈希新密码
	hashed, err := encrypt.HashPassword(info.NewPassword)
	if err != nil {
		return buserr.WithErr(constant.ErrInternalServer, err)
	}
	return settingRepo.Update("Password", hashed)
}

func (a *AuthService) GetLoginSetting() (*dto.LoginSetting, error) {
	language, _ := settingRepo.GetValueByKey("Language")
	panelName, _ := settingRepo.GetValueByKey("PanelName")
	theme, _ := settingRepo.GetValueByKey("Theme")

	return &dto.LoginSetting{
		Language:  language,
		PanelName: panelName,
		Theme:     theme,
	}, nil
}

// SaveLoginLog 保存登录日志（供 API 层调用）
func SaveLoginLog(ip, agent string, err error) {
	log := &model.LoginLog{
		IP:    ip,
		Agent: agent,
	}
	if err != nil {
		log.Status = constant.StatusFailed
		log.Message = err.Error()
	} else {
		log.Status = constant.StatusSuccess
	}
	if e := logRepo.CreateLoginLog(log); e != nil {
		global.LOG.Errorf("Failed to save login log: %v", e)
	}
}
