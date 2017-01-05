package controllers

import (
	. "github.com/1046102779/sms/logger"
	"github.com/1046102779/sms/models"
	"github.com/astaxie/beego"
	jsoniter "github.com/json-iterator/go"
)

type YunpianSmsController struct {
	beego.Controller
}

// 推送状态报告
// @router /yunpian/callback [POST]
func (t *YunpianSmsController) ReceivedNotification() {
	// 解析获取
	var (
		yunpianReceipt *models.YunpianReceipt = new(models.YunpianReceipt)
	)
	if err := jsoniter.Unmarshal(t.Ctx.Input.RequestBody, yunpianReceipt); err != nil {
		Logger.Error(err.Error())
		t.Ctx.Output.Body([]byte("SUCCESS"))
		return
	}
	instance := models.GetYunpianInstance()
	if _, err := instance.ReceivedNotification(yunpianReceipt); err != nil {
		Logger.Error(err.Error())
	}
	t.Ctx.Output.Body([]byte("SUCCESS"))
	return
}
