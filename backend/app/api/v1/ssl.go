package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"
	sslutil "xpanel/utils/ssl"

	"github.com/gin-gonic/gin"
)

type SSLAPI struct{}

// --- 证书 ---

func (a *SSLAPI) SearchCertificate(c *gin.Context) {
	var req dto.SearchCertReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewICertificateService().SearchWithPage(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *SSLAPI) CreateCertificate(c *gin.Context) {
	var req dto.CertificateCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertificateService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) UpdateCertificate(c *gin.Context) {
	var req dto.CertificateUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertificateService().Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) UploadCertificate(c *gin.Context) {
	var req dto.CertificateUpload
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertificateService().Upload(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) DeleteCertificate(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertificateService().Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) GetCertificateDetail(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	detail, err := service.NewICertificateService().GetDetail(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, detail)
}

func (a *SSLAPI) ApplyCertificate(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertificateService().Apply(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) RenewCertificate(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertificateService().Renew(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) GetCertificateLog(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	logContent, err := service.NewICertificateService().GetLog(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, logContent)
}

// --- ACME 账户 ---

func (a *SSLAPI) ListAcmeAccount(c *gin.Context) {
	items, err := service.NewIAcmeAccountService().GetList()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *SSLAPI) CreateAcmeAccount(c *gin.Context) {
	var req dto.AcmeAccountCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIAcmeAccountService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) DeleteAcmeAccount(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIAcmeAccountService().Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- DNS 账户 ---

func (a *SSLAPI) ListDnsAccount(c *gin.Context) {
	items, err := service.NewIDnsAccountService().GetList()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *SSLAPI) CreateDnsAccount(c *gin.Context) {
	var req dto.DnsAccountCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIDnsAccountService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) UpdateDnsAccount(c *gin.Context) {
	var req dto.DnsAccountUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIDnsAccountService().Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) DeleteDnsAccount(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIDnsAccountService().Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- 导入导出 ---

func (a *SSLAPI) ExportAccounts(c *gin.Context) {
	data, err := service.NewIAccountExportService().Export()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *SSLAPI) ImportAccounts(c *gin.Context) {
	var req dto.AccountExport
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIAccountExportService().Import(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- SSL 路径与 DNS 提供商列表 ---

func (a *SSLAPI) GetSSLDir(c *gin.Context) {
	dir := service.NewICertificateService().GetSSLDir()
	helper.SuccessWithData(c, dir)
}

func (a *SSLAPI) UpdateSSLDir(c *gin.Context) {
	var req dto.SSLDirUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertificateService().UpdateSSLDir(req.Dir); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSLAPI) GetDnsProviders(c *gin.Context) {
	helper.SuccessWithData(c, sslutil.SupportedDNSProviders())
}
