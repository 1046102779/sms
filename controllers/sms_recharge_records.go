package controllers

import (
	"fmt"
	"time"

	utils "github.com/1046102779/common"
	. "github.com/1046102779/common/utils"
	pb "github.com/1046102779/igrpc"
	"github.com/1046102779/sms/conf"
	. "github.com/1046102779/sms/logger"
	"github.com/1046102779/sms/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

// SmsRechargeRecordsController operations for SmsRechargeRecords
type SmsRechargeRecordsController struct {
	beego.Controller
}

type RechargingInfo struct {
	PayType int16 `json:"pay_type"` // 10: 微信公众号支付 Native, 11：微信公众号支付 JSAPI
	Money   int   `json:"money"`    // 充值金额，单位：分
}

// SaaS平台下账户短信充值
// @router /recharging [POST]
func (t *SmsRechargeRecordsController) SmsRecharge() {
	var (
		companyId, userId int             // 从header头部获取公司ID和用户ID
		rechargingInfo    *RechargingInfo = new(RechargingInfo)
	)
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
		userId = info.UserId
	} else {
		err := errors.New("please login homepage")
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.USER_LOGGED_IN,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	if err := jsoniter.Unmarshal(t.Ctx.Input.RequestBody, rechargingInfo); err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": utils.SOURCE_DATA_ILLEGAL,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	o := orm.NewOrm()
	now := time.Now()
	record := &models.SmsRechargeRecords{
		PayType:       rechargingInfo.PayType,
		CompanyId:     companyId,
		UserId:        userId,
		RechargeMoney: rechargingInfo.Money,
		OutTradeNo:    fmt.Sprintf("%s-%s", now.Format("20060102150405"), GetRandomString(4)),
		PayStatus:     int16(models.SMS_PAY_TOBEPAY),
		Status:        utils.STATUS_VALID,
		UpdatedAt:     now,
		CreatedAt:     now,
	}
	if retcode, err := record.InsertSmsRechargeRecordNoLock(&o); err != nil {
		Logger.Error(err.Error())
		t.Data["json"] = map[string]interface{}{
			"err_code": retcode,
			"err_msg":  errors.Cause(err).Error(),
		}
		t.ServeJSON()
		return
	}
	// rpc调用公众号平台充值
	// 获取充值内容，支付金额，订单号，支付方式，用户openid
	in := &pb.SmsRechargeInfo{
		Money:             int64(rechargingInfo.Money),
		TradeNo:           record.OutTradeNo,
		Title:             fmt.Sprintf("进销存系统短信充值平台"),
		OfficialAccountId: 1,
		//	Openid:  openid,
	}
	outJSAPI := new(pb.WechatJSAPIParamInfo)
	outNative := new(pb.WechatNativeParamInfo)
	switch rechargingInfo.PayType {
	case int16(models.WECHAT_TRADE_TYPE_JSAPI):
		// JSAPI 支付需要获取用户openid, 从公众号第三方平台获取
		openidIn := &pb.UserOpenidInfo{
			UserId:    int64(userId),
			CompanyId: 1, // 盈创丰茂
		}
		conf.OfficialAccountClient.Call(fmt.Sprintf("%s.%s", "official_accounts", "GetOpenid"), openidIn, openidIn)
		// 调用JSAPI，获取微信支付参数
		in.Openid = openidIn.Openid
		conf.OfficialAccountClient.Call(fmt.Sprintf("%s.%s", "official_accounts", "GetSmsRechargePayJsapiParams"), in, outJSAPI)
	case int16(models.WECHAT_TRADE_TYPE_NATIVE):
		// Native二维码支付不需要用户openid
		conf.OfficialAccountClient.Call(fmt.Sprintf("%s.%s", "official_accounts", "GetSmsRechargePayNativeParams"), in, outNative)
	}
	t.Data["json"] = map[string]interface{}{
		"err_code":    0,
		"err_msg":     "",
		"jsapi_info":  *outJSAPI,
		"native_info": *outNative,
	}
	t.ServeJSON()
	return
}
