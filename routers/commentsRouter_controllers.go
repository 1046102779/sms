package routers

import (
	"github.com/astaxie/beego"
)

func init() {

	beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:ChuanglanSmsController"] = append(beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:ChuanglanSmsController"],
		beego.ControllerComments{
			Method: "ReceivedNotification",
			Router: `/callback`,
			AllowHTTPMethods: []string{"GET"},
			Params: nil})

	beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:ChuanglanSmsController"] = append(beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:ChuanglanSmsController"],
		beego.ControllerComments{
			Method: "QueryBalance",
			Router: `/querybalance`,
			AllowHTTPMethods: []string{"GET"},
			Params: nil})

	beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:SmsController"] = append(beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:SmsController"],
		beego.ControllerComments{
			Method: "MobileVerificationCode",
			Router: `/mobile_verification_code`,
			AllowHTTPMethods: []string{"post"},
			Params: nil})

	beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:SmsController"] = append(beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:SmsController"],
		beego.ControllerComments{
			Method: "SendMarketingSms",
			Router: `/marketing`,
			AllowHTTPMethods: []string{"POST"},
			Params: nil})

	beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:SmsRechargeRecordsController"] = append(beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:SmsRechargeRecordsController"],
		beego.ControllerComments{
			Method: "SmsRecharge",
			Router: `/recharging`,
			AllowHTTPMethods: []string{"POST"},
			Params: nil})

	beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:YunpianSmsController"] = append(beego.GlobalControllerRouter["github.com/1046102779/sms/controllers:YunpianSmsController"],
		beego.ControllerComments{
			Method: "ReceivedNotification",
			Router: `/yunpian/callback`,
			AllowHTTPMethods: []string{"POST"},
			Params: nil})

}
