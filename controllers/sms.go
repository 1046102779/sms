package controllers

import (
	"encoding/json"
	"fmt"
	"time"

	utils "github.com/1046102779/common"
	. "github.com/1046102779/common/utils"
	. "github.com/1046102779/sms/logger"
	"github.com/1046102779/sms/models"
	"github.com/astaxie/beego"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

const (
	TAG_GROUPS_TYPE      = 1
	ACTIVITY_GROUPS_TYPE = 2
	CUSTOM_GROUPS_TYPE   = 3

	// 错误码
	SMS_MASS_SEND_BLACKWORDS = 12030 // 屏蔽词过滤

	ACTIVITY_TYPE_QUESTIONNAIRE = 4

	STAR_TAG_WHITE  = 1 // 非星标记
	STAR_TAG_YELLOW = 2 // 星标记
)

type SmsController struct {
	beego.Controller
}

// 生成验证码，并把验证码保存到redis，且发送短信验证码
// @router /mobile_verification_code [post]
func (t *SmsController) MobileVerificationCode() {
	type MobileInfo struct {
		Mobile string `json:"mobile"`
	}
	var (
		mobiles []string
		key     string
		info    *MobileInfo = new(MobileInfo)
	)
	if err := json.Unmarshal(t.Ctx.Input.RequestBody, info); err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.JSON_PARSE_FAILED,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}

	// 生成四位验证码
	code := GetRandomString(4)
	key = fmt.Sprintf("SMS:%s:LOGIN", info.Mobile)
	if err := RedisClient.Set(key, code, 600*time.Second).Err(); err != nil {
		Logger.Error("set redis failed. " + err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.REDIS_SET_FAILED,
			"err_msg":  "store redis error:" + errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	mobiles = append(mobiles, info.Mobile)
	// 发送短信验证码
	chuanglan := ChuanglanSmsController{}
	countPerSingle, smsSendCount, msgid, content, templateId, retcode, err := chuanglan.SendVerificationSms(code, mobiles)
	if err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": retcode,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	fmt.Printf("countPerSingle=%d, smsSendCount=%d, msgid=%s, content=%s, templateId=%d\n",
		countPerSingle, smsSendCount, msgid, content, templateId)
	// 扣除该公司营销所发送的短信和平台短信数量
	models.UpdateChuanglanRemaingSMS(-1, 0, int64(-1*smsSendCount), 0)
	// 发送验证码
	t.Data["json"] = map[string]interface{}{
		"err_code": 0,
		"err_msg":  "",
	}
	t.ServeJSON()
	return
}

// 营销类短信，主动推送给用户，用户被动接受且可以退订
// @router /marketing [POST]
func (t *SmsController) SendMarketingSms() {
	type SmsInfo struct {
		Content    string   `json:"content"`
		Mobiles    []string `json:"mobiles"`
		TemplateId int      `json:"template_id"`
	}
	var (
		info      *SmsInfo = new(SmsInfo)
		companyId int
	)
	if err := jsoniter.Unmarshal(t.Ctx.Input.RequestBody, info); err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.JSON_PARSE_FAILED,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	// 获取user_id和company_id
	if info, retcode, err := GetHeaderParams(t.Ctx.Request); err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": retcode,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	} else if info != nil && info.CompanyId > 0 {
		companyId = info.CompanyId
	} else {
		err := errors.New("please login homepage")
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.USER_LOGGED_IN,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	_, platformMarketingCount, companySmsRemainingCount := models.GetChuanglanRemainingSMS(int64(companyId))
	if platformMarketingCount <= 0 || companySmsRemainingCount <= 0 {
		err := errors.New("sms remaining count not enough")
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.SMS_CHUANGLAN_REMAINING_NOT_ENOUGH,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	chuanglan := &ChuanglanSmsController{}
	countPerSingle, smsSendCount, msgid, retcode, err := chuanglan.SendMarketingSms(companyId, info.TemplateId, info.Content, info.Mobiles)
	if err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": retcode,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	// 扣除该公司营销所发送的短信和平台短信数量
	models.UpdateChuanglanRemaingSMS(int64(companyId), 0, int64(-1*smsSendCount), int64(-1*smsSendCount))
	fmt.Printf("countPerSingle=%d, smsSendCount=%d, msgid=%s\n", countPerSingle, smsSendCount, msgid)
	t.Data["json"] = map[string]interface{}{
		"err_code": 0,
		"err_msg":  "",
	}
	t.ServeJSON()
	return
}
