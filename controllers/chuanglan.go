package controllers

import (
	"fmt"
	"strings"
	"time"

	utils "github.com/1046102779/common"
	. "github.com/1046102779/sms/logger"
	"github.com/1046102779/sms/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/pkg/errors"
)

type ChuanglanSmsController struct {
	beego.Controller
}

// 创蓝短信发送状态回调响应地址
// @router /callback [GET]
func (t *ChuanglanSmsController) ReceivedNotification() {
	msgid := t.GetString("msgid")
	reportTime := t.GetString("reportTime")
	mobile := t.GetString("mobile")
	code := t.GetString("status")
	instance := models.GetChuanglanInstance()
	instance.ReceivedNotification(mobile, msgid, code, reportTime)
	t.Data["json"] = map[string]interface{}{
		"err_code": 0,
		"err_msg":  "",
	}
	t.ServeJSON()
	return
}

// 额度查询接口
// @router /querybalance [GET]
func (t *ChuanglanSmsController) QueryBalance() {
	accountType, _ := t.GetInt("account_type")
	if accountType != models.SMS_CHUANGLAN_VERIFICATION_TYPE && accountType != models.SMS_CHUANGLAN_MARKETING_TYPE {
		err := errors.New("param `account_type` is illegal!")
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.SOURCE_DATA_ILLEGAL,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	instance := models.GetChuanglanInstance()
	if instance == nil {
		err := errors.New("chuanglan sms service is unabled.")
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.SMS_SERVICE_253_CHUANGLAN_UNABLED,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	remainingCount, retcode, err := instance.QueryBalance(int16(accountType))
	if err != nil {
		t.Data["json"] = map[string]interface{}{
			"err_code": retcode,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	t.Data["json"] = map[string]interface{}{
		"err_code":        0,
		"err_msg":         "",
		"remaining_count": remainingCount,
	}
	t.ServeJSON()
	return
}

// 营销类短信，主动推送给用户，用户被动接受且可以退订
/*
	>>	本接口支持营销类短信两类:
		1. 采用模板和参数，形成短信内容
		2. 直接发送自定义内容，无模板
*/
func (t *ChuanglanSmsController) SendMarketingSms(companyId int, templateId int, content string, mobiles []string, args ...interface{}) (countPerSingle, smsSendCount int, msgid string, retcode int, err error) {
	Logger.Info("[%v] enter SendMarketingSms.", templateId)
	defer Logger.Info("[%v] left SendMarketingSms.", templateId)
	var (
		template   *models.SmsTemplates
		smsContent string
	)
	if (strings.TrimSpace(content) == "" && templateId <= 0) || mobiles == nil || len(mobiles) <= 0 {
		err = errors.New("param `content || mobiles` empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	instance := models.GetChuanglanInstance()
	if instance == nil {
		err = errors.New("chuanglan sms service is unabled.")
		retcode = utils.SMS_SERVICE_253_CHUANGLAN_UNABLED
		return
	}
	// 如果模板ID不为空，则采用模板发送短信
	if templateId > 0 {
		o := orm.NewOrm()
		template = &models.SmsTemplates{
			Id: templateId,
		}
		if retcode, err = template.ReadSmsTemplateNoLock(&o); err != nil {
			err = errors.Wrap(err, "SendMarketingSms")
			return
		}
		newArgs := append([]interface{}{instance.SignName}, args...)
		smsContent = fmt.Sprintf(template.TemplateContent, newArgs...)
	} else {
		smsContent = fmt.Sprintf("【%s】%s。回复TD退订", instance.SignName, content)
	}
	countPerSingle, smsSendCount, msgid, retcode, err = instance.SendMarketingSms(smsContent, mobiles)
	// 增加短信发送记录
	now := time.Now()
	o := orm.NewOrm()
	record := &models.SmsSendRecords{
		SmsTemplateId:   templateId,
		CompanyId:       companyId,
		Content:         smsContent,
		ReceiverMobiles: strings.Join(mobiles, ","),
		SendStatus:      fmt.Sprintf("%d", retcode),
		Count:           smsSendCount,
		CountPerContent: int16(countPerSingle),
		MessageId:       msgid,
		SendAt:          now,
	}
	if retcode, err = record.InsertSmsSendRecordNoLock(&o); err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": retcode,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	return
}

func (t *ChuanglanSmsController) SendVerificationSms(code string, mobiles []string) (countPerSingle, smsSendCount int, msgid string, content string, templateId int, retcode int, err error) {
	Logger.Info("[chuanglan] enter SendVerificationSms.")
	defer Logger.Info("[chuanglan] left SendVerificationSms.")
	var (
		template *models.SmsTemplates
	)
	instance := models.GetChuanglanInstance()
	if instance == nil {
		err = errors.New("chuanglan sms service is unabled.")
		retcode = utils.SMS_SERVICE_253_CHUANGLAN_UNABLED
		return
	}
	// 拿到短信验证码模板
	template, retcode, err = models.GetSmsTemplate(instance.SmsServiceProviderId, models.MOBILE_VERIFICATION_CODE_CONTENT)
	if err != nil {
		err = errors.Wrap(err, "SendSms[chuanglan]")
		return
	}

	templateId = template.Id
	content = fmt.Sprintf(template.TemplateContent, instance.SignName, code)
	countPerSingle, smsSendCount, msgid, retcode, err = instance.SendVerificationSms(content, mobiles)
	// 增加短信发送记录
	now := time.Now()
	o := orm.NewOrm()
	record := &models.SmsSendRecords{
		SmsTemplateId:   templateId,
		CompanyId:       -1,
		Content:         content,
		ReceiverMobiles: strings.Join(mobiles, ","),
		SendStatus:      fmt.Sprintf("%d", retcode),
		Count:           smsSendCount,
		CountPerContent: int16(countPerSingle),
		MessageId:       msgid,
		SendAt:          now,
	}
	if retcode, err = record.InsertSmsSendRecordNoLock(&o); err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": retcode,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	return
}
