package models

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	utils "github.com/1046102779/common"
	"github.com/1046102779/common/httpRequest"
	"github.com/1046102779/sms/conf"
	. "github.com/1046102779/sms/logger"
	"github.com/astaxie/beego/orm"
	"github.com/pkg/errors"

	pb "github.com/1046102779/igrpc"
)

/*
	创蓝短信服务提供商
	提供的服务列表：
	>> 一、短信验证码
	1. 普通短信发送
	2. 状态报告推送
	3. 短信接收
	4. 额度查询接口

	>> 二、会员营销短信
	1. 普通短信发送
	2. 状态报告推送
	3. 短信接收
	4. 额度查询接口
*/

var (
	SMS_CHUANGLAN_VERIFICATION_TYPE = 1 // 验证码短信，不可退订的, 独立账户
	SMS_CHUANGLAN_MARKETING_TYPE    = 2 // 营销短信，可退订的, 独立账户
)

type ChuanglanInfo struct {
	HttpApi              string // 创蓝253短信服务HTTP API
	QueryBalanceHttpApi  string // 额度查询接口
	ReceiverHttpApi      string // 接收响应状态请求地址
	VerificationAccount  string // 必填参数。用户验证码账号
	VerificationPassword string // 必填参数。用户验证码密码
	MarketingAccount     string // 必填参数。用户营销账号
	MarketingPassword    string // 必填参数。用户营销密码
	Mobiles              string // 必填参数。合法的手机号码，号码间用英文逗号分隔
	SingleSmsMaxLength   int    // 创蓝单条短信最大长度，超过此长度，则分条发送
	SmsContent           string // 必填参数。短信内容，短信内容长度不能超过536个字符。使用URL方式编码为UTF-8格式。短信内容超过70个字符（企信通是60个字符）时，会被拆分成多条，然后以长短信的格式发送。
	ReceivedStatus       int16  // 必填参数。是否需要状态报告，0表示不需要，1表示需要
	Extend               int16  // 可选参数，扩展码，用户定义扩展码,扩展码的长度将直接影响短信上行接收的接收。固需要传扩展码参数时，请提前咨询客服相关设置问题。
	SignName             string // 短信服务应用签名
	SmsServiceProviderId int    // 内部短信服务商ID
}

var (
	chuanglanService *ChuanglanInfo
	mutex            sync.Mutex
)

func GetChuanglanInstance() (instance *ChuanglanInfo) {
	Logger.Info("enter GetChuanglanInstance.")
	defer Logger.Info("left GetChuanglanInstance.")
	mutex.Lock()
	defer mutex.Unlock()
	if chuanglanService == nil {
		// 调用rpcx服务，获取系统配置的253创蓝账号和密码
		systemConfInfo := &pb.ChuanglanConfInfo{}
		err := conf.AccountClient.Call(fmt.Sprintf("%s.%s", "accounts", "GetChuanglanAccountInfo"), systemConfInfo, systemConfInfo)
		if err != nil {
			fmt.Println(err.Error())
		}
		// 获取创蓝服务提供商，单条短信最大长度
		var (
			smsServiceProviders []SmsServiceProviders = []SmsServiceProviders{}
		)
		o := orm.NewOrm()
		num, err := o.QueryTable((&SmsServiceProviders{}).TableName()).Filter("type", SMS_SERVICE_PROVIDER_TYPE_253_CHUANGLAN).Filter("status", utils.STATUS_VALID).Filter("is_valid", SMS_SERVICE_VALID).All(&smsServiceProviders)
		if err != nil {
			Logger.Error(err.Error())
			return
		}
		if num <= 0 {
			return nil // 尚未启用创蓝短信服务
		}
		chuanglanService = &ChuanglanInfo{
			VerificationAccount:  systemConfInfo.VerificationAccount,
			VerificationPassword: systemConfInfo.VerificationPassword,
			MarketingAccount:     systemConfInfo.MarketingAccount,
			MarketingPassword:    systemConfInfo.MarketingPassword,
			HttpApi:              systemConfInfo.HttpApi,
			ReceiverHttpApi:      systemConfInfo.ReceiverHttpApi,
			QueryBalanceHttpApi:  systemConfInfo.QueryBalanceHttpApi,
			SingleSmsMaxLength:   smsServiceProviders[0].SingleSmsMaxLength,
			SignName:             smsServiceProviders[0].SignName,
			SmsServiceProviderId: smsServiceProviders[0].Id,
			ReceivedStatus:       1,
		}
		fmt.Println("*chuanglanService: ", *chuanglanService)
	}
	return chuanglanService
}

// rpc获取平台创蓝剩余短信数量和公司剩余短信数量
func GetChuanglanRemainingSMS(companyId int64) (platformVerificationCount int64, platformMarketingCount int64, companySmsRemainingCount int64) {
	in := &pb.ChuanglanSmsInfo{
		CompanyId: companyId,
	}
	conf.AccountClient.Call(fmt.Sprintf("%s.%s", "accounts", "GetChuanglanRemainingSMS"), in, in)
	platformMarketingCount = in.PlatformMarketingCount
	platformVerificationCount = in.PlatformVerificationCount
	companySmsRemainingCount = in.CompanySmsRemainingCount
	return
}

func UpdateChuanglanRemaingSMS(companyId int64, platformVerificationInc int64, platformMarketingInc int64, companySmsInc int64) {
	in := &pb.ChuanglanSmsInfo{
		CompanyId:                 companyId,
		PlatformVerificationCount: platformVerificationInc,
		PlatformMarketingCount:    platformMarketingInc,
		CompanySmsRemainingCount:  companySmsInc,
	}
	conf.AccountClient.Call(fmt.Sprintf("%s.%s", "accounts", "UpdateChuanglanSmsCount"), in, in)
	return
}

/*
 格式说明：
	 短信提交响应分为两行:
		 第一行为响应时间和提交状态，
		 第二行为服务器给出提交messageid。
		 无论发送的号码是多少，一个发送请求只返回一个messageid，如果响应的状态不是“0”，则没有messageid即第二行数据。（每行以换行符(0x0a,即\n)分割）
 返回输出：
	20161025170822,0
	16102517082223817
 备注：
	响应时间为20161025170822，响应状态为0 表明那个成功提交到服务器；16102517082223817为返回的messageid，这个供状态报告匹配时使用。
*/
func parseChuanglanBody(bodyData []byte) (timeStr string, retcode int, msgid string) {
	if bodyData == nil && len(bodyData) > 0 {
		return
	}
	twolines := strings.Split(string(bodyData), "\n")
	if len(twolines) > 1 {
		firstline := strings.Split(twolines[0], ",")
		if len(firstline) > 1 {
			timeStr = firstline[0]
			result, _ := strconv.ParseInt(firstline[1], 10, 64)
			retcode = int(result)
		}
		msgid = twolines[1]
	}
	return
}

// 发送专用通道短信：是不可退订的
func (t *ChuanglanInfo) SendVerificationSms(content string, mobiles []string) (countPerSingle int, smsSendCount int, msgid string, retcode int, err error) {
	Logger.Info("enter SendVerificationSms.")
	defer Logger.Info("left SendVerificationSms.")
	var (
		bodyData []byte
	)
	if mobiles == nil || len(mobiles) <= 0 || strings.TrimSpace(content) == "" || t.SingleSmsMaxLength <= 0 {
		return
	}
	charCount := utf8.RuneCountInString(content)
	countPerSingle = charCount / t.SingleSmsMaxLength
	if charCount%t.SingleSmsMaxLength > 0 {
		countPerSingle += 1
	}
	smsSendCount = countPerSingle * len(mobiles)
	httpStr := fmt.Sprintf("%s?account=%s&pswd=%s&mobile=%s&msg=%s&needstatus=true", t.HttpApi, t.VerificationAccount, t.VerificationPassword, strings.Join(mobiles, ","), url.QueryEscape(content))
	fmt.Println("uri: ", httpStr)
	bodyData, err = httpRequest.HttpGetBody(httpStr)
	if err != nil {
		err = errors.Wrap(err, "SendVerificationSms")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	fmt.Println("body : ", string(bodyData))
	_, retcode, msgid = parseChuanglanBody(bodyData)
	if retcode != 0 {
		err = t.getErrorMessage(retcode)
	}
	return
}

// 发送营销短信：是指可以退订的
func (t *ChuanglanInfo) SendMarketingSms(content string, mobiles []string) (countPerSingle int, smsSendCount int, msgid string, retcode int, err error) {
	Logger.Info("enter SendMarketingSms.")
	defer Logger.Info("left SendMarketingSms.")
	var (
		bodyData []byte
	)
	if mobiles == nil || len(mobiles) <= 0 || strings.TrimSpace(content) == "" || t.SingleSmsMaxLength <= 0 {
		return
	}
	charCount := utf8.RuneCountInString(content)
	countPerSingle = charCount / t.SingleSmsMaxLength
	if charCount%t.SingleSmsMaxLength > 0 {
		countPerSingle += 1
	}
	smsSendCount = countPerSingle * len(mobiles)
	httpStr := fmt.Sprintf("%s?account=%s&pswd=%s&mobile=%s&msg=%s&needstatus=true", t.HttpApi, t.MarketingAccount, t.MarketingPassword, strings.Join(mobiles, ","), url.QueryEscape(content))
	bodyData, err = httpRequest.HttpGetBody(httpStr)
	if err != nil {
		err = errors.Wrap(err, "SendVerificationSms")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	fmt.Println("bodyData:", string(bodyData))
	_, retcode, msgid = parseChuanglanBody(bodyData)
	if retcode != 0 {
		err = t.getErrorMessage(retcode)
	}
	return
}

func (t *ChuanglanInfo) ReceivedNotification(mobile string, msgid string, code string, reportTime string) (retcode int, err error) {
	Logger.Info("enter ReceivedNotification.")
	defer Logger.Info("left ReceivedNotification.")
	status := t.getReportErrorMessage(code)
	if status > 0 {
		o := orm.NewOrm()
		smsReceiptFailedRecord := &SmsReceiptFailedRecords{
			MessageId:     msgid,
			Mobile:        mobile,
			ReceiptStatus: status,
			ReceiptAt:     fmt.Sprintf("20%s", reportTime),
		}
		retcode, err = smsReceiptFailedRecord.InsertSmsReceiptFailedRecordNoLock(&o)
		if err != nil {
			err = errors.Wrap(err, "ReceivedNotification")
			return
		}
	}
	return
}

// 额度查询接口
// @param accountType : 账户类型，1.验证码短信是不可退订的，属于verification_account
//								  2.营销短信是可退订的，属于marketing_account
func (t *ChuanglanInfo) QueryBalance(accountType int16) (remainingCount int, retcode int, err error) {
	Logger.Info("enter QueryBalance.")
	defer Logger.Info("left QueryBalance.")
	var (
		account, password string
		bodyData          []byte
		remainingCountStr string
	)
	if int(accountType) == SMS_CHUANGLAN_VERIFICATION_TYPE {
		account = t.VerificationAccount
		password = t.VerificationPassword
	} else {
		account = t.MarketingAccount
		password = t.MarketingPassword
	}
	httpStr := fmt.Sprintf("%s?account=%s&pswd=%s", t.QueryBalanceHttpApi, account, password)
	fmt.Println("http uri: ", httpStr)
	bodyData, err = httpRequest.HttpGetBody(httpStr)
	if err != nil {
		err = errors.Wrap(err, "QueryBalance")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	_, retcode, remainingCountStr = parseChuanglanBody(bodyData)
	remainingCountTemp, _ := strconv.ParseInt(strings.Split(remainingCountStr, ",")[1], 10, 64)
	remainingCount = int(remainingCountTemp)
	return
}

func (t *ChuanglanInfo) getReportErrorMessage(code string) (status int16) {
	switch code {
	case "DELIVRD": // 消息送达成功
		status = 0
	case "EXPIRED": // 短消息超过有效期
		status = 11
	case "UNDELIV": // 短消息是不可达的
		status = 12
	case "UNKNOWN": // 未知短消息状态
		status = 13
	case "REJECTD": // 短消息被短信中心拒绝
		status = 14
	case "DTBLACK": // 目的号码是黑名单号码
		status = 15
	case "ERR:104": // 系统忙
		status = 16
	case "REJECT":
		status = 17
	default: // 网关内部状态
		status = 18
	}
	return
}

func (t *ChuanglanInfo) getErrorMessage(errcode int) (err error) {
	var (
		message string
	)
	switch errcode {
	case 101:
		message = "创蓝短信错误：无此用户"
	case 102:
		message = "创蓝短信错误：密码错误"
	case 103:
		message = "创蓝短信错误：提交过快（提交速度超过流速限制）"
	case 104:
		message = "创蓝短信错误：系统忙（因平台侧原因，暂时无法处理提交的短信）"
	case 105:
		message = "创蓝短信错误：敏感短信（短信内容包含敏感词）"
	case 106:
		message = "创蓝短信错误：消息长度错（>536或<=0）"
	case 107:
		message = "创蓝短信错误：包含错误的手机号码"
	case 108:
		message = "创蓝短信错误：手机号码个数错（群发>50000或<=0）"
	case 109:
		message = "创蓝短信错误：无发送额度（该用户可用短信数已使用完）"
	case 110:
		message = "创蓝短信错误：不在发送时间内"
	case 113:
		message = "创蓝短信错误: extno格式错（非数字或者长度不对）"
	case 116:
		message = "创蓝短信错误: 签名不合法或未带签名（用户必须带签名的前提下）"
	case 117:
		message = "创蓝短信错误: IP地址认证错,请求调用的IP地址不是系统登记的IP地址"
	case 118:
		message = "创蓝短信错误: 用户没有相应的发送权限（账号被禁止发送）"
	case 119:
		message = "创蓝短信错误: 用户已过期"
	case 120:
		message = "创蓝短信错误: 违反放盗用策略(日发限制) --自定义添加"
	case 121:
		message = "创蓝短信错误: 必填参数。是否需要状态报告，取值true或false"
	case 122:
		message = "创蓝短信错误: 5分钟内相同账号提交相同消息内容过多"
	case 123:
		message = "创蓝短信错误: 发送类型错误"
	case 124:
		message = "创蓝短信错误: 白模板匹配错误"
	case 125:
		message = "创蓝短信错误: 匹配驳回模板，提交失败"
	case 126:
		message = "创蓝短信错误: 审核通过模板匹配错误"
	}
	if message != "" {
		err = errors.New(message)
	}
	return err
}
