package models

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	utils "github.com/1046102779/common"
	"github.com/1046102779/common/httpRequest"
	. "github.com/1046102779/common/utils"
	pb "github.com/1046102779/igrpc"
	"github.com/1046102779/sms/conf"
	. "github.com/1046102779/sms/logger"
	"github.com/astaxie/beego/orm"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

/*
	云片网短信服务：
	1. 发送短信服务列表
		1.1 单条发送 https://sms.yunpian.com/v2/sms/single_send.json
		1.2 批量发送相同内容 https://sms.yunpian.com/v2/sms/batch_send.json
		1.3 批量发送不同内容 https://sms.yunpian.com/v2/sms/multi_send.json
		1.4 推送状态报告
	2. 模板接口
		2.1 添加模板 https://sms.yunpian.com/v2/tpl/add.json
		2.2 取模板 https://sms.yunpian.com/v2/tpl/get.json
		2.3 修改模板 https://sms.yunpian.com/v2/tpl/update.json
		2.4 删除模板 https://sms.yunpian.com/v2/tpl/del.json
	3. 签名接口
		3.1 添加签名 https://sms.yunpian.com/v2/sign/add.json
		3.2 获取签名 https://sms.yunpian.com/v2/sign/get.json
		3.3 修改签名 https://sms.yunpian.com/v2/sign/update.json
	4. 查短信发送记录 https://sms.yunpian.com/v2/sms/get_record.json
	5. 查屏蔽词 https://sms.yunpian.com/v2/sms/get_black_word.json

	说明：请求方法全部为POST请求
*/

type YunpianInfo struct {
	SingleApiKey       string // 普通发送apikey值
	GroupApiKey        string // 群发短信apikey值
	HttpApi            string // 短信服务调用Http api地址
	ReceiverHttpApi    string // 短信服务系统接收地址
	SingleSmsMaxLength int    // 云片网单条短信最大长度，超过此长度，则分条发送
}

type YunpianSingleSendInfo struct {
	ApiKey      string `json:"apikey"`
	Mobile      string `json:"mobile"`
	Text        string `json:"text"`
	CallbackUrl string `json:"callback_url"`
}

type YunpianSingleSendRespInfo struct {
	Code  int     `json:"code"`  // 0代表发送成功，其他code代表出错，详细见"返回值说明"页面
	Msg   string  `json:"msg"`   // 例如""发送成功""，或者相应错误信息
	Count int     `json:"count"` // 发送成功短信的计费条数(计费条数：70个字一条，超出70个字时按每67字一条计费)
	Fee   float64 `json:"fee"`   // 扣费金额，单位：元，类型：双精度浮点型/double
	Sid   int64   `json:"sid"`   // 短信id，64位整型， 对应Java和C#的Long，不可用int解析
}

// 批量发送接口返回结构体
type SendSmsRespInfo struct {
	Code   int     `json:"code"`
	Msg    string  `json:"msg"`
	Count  int     `json:"count"`
	Fee    float64 `json:"fee"`
	Unit   string  `json:"unit"`
	Mobile string  `json:"mobile"`
	Sid    int64   `json:"sid"`
}

type BatchSmsSendRespInfo struct {
	TotalCount int               `json:"total_count"`
	TotalFee   string            `json:"total_fee"`
	Unit       string            `json:"unit"`
	Datas      []SendSmsRespInfo `json:"data"`
}

var (
	yunpianService *YunpianInfo
)

func GetYunpianInstance() (instance *YunpianInfo) {
	Logger.Info("enter GetYunpianInstance.")
	defer Logger.Info("left GetYunpianInstance.")
	mutex.Lock()
	defer mutex.Unlock()
	if yunpianService == nil {
		// 调用rpcx服务，获取系统配置的云片网appkey列表
		systemConfInfo := &pb.YunpianConfInfo{}
		conf.AccountClient.Call(fmt.Sprintf("%s.%s", "accounts", "GetYunpianAccountInfo"), systemConfInfo, systemConfInfo)
		// 获取云片网服务提供商，单条短信最大长度
		singleSmsMaxLength := 0
		var (
			smsServiceProviders []SmsServiceProviders = []SmsServiceProviders{}
		)
		o := orm.NewOrm()
		num, err := o.QueryTable((&SmsServiceProviders{}).TableName()).Filter("type", 20).Filter("status", utils.STATUS_VALID).Filter("is_valid", 20).All(&smsServiceProviders)
		if err != nil {
			Logger.Error(err.Error())
			return
		}
		if num > 0 {
			singleSmsMaxLength = smsServiceProviders[0].SingleSmsMaxLength
		}
		yunpianService = &YunpianInfo{
			SingleApiKey:       systemConfInfo.SingleApiKey,
			GroupApiKey:        systemConfInfo.GroupApiKey,
			HttpApi:            systemConfInfo.HttpApi,
			ReceiverHttpApi:    systemConfInfo.ReceiverHttpApi,
			SingleSmsMaxLength: singleSmsMaxLength,
		}
	}
	return yunpianService
}

// 1.1 单条发送 https://sms.yunpian.com/v2/sms/single_send.json
func (t *YunpianInfo) SendSingleSms(content string, mobile string) (count int, fee int, msgid string, retcode int, err error) {
	Logger.Info("[%v] enter SendSingleSms.", mobile)
	defer Logger.Info("[%v] enter SendSingleSms.", mobile)
	var (
		bodyData, body []byte
		singleSendInfo *YunpianSingleSendInfo
		respInfo       *YunpianSingleSendRespInfo = new(YunpianSingleSendRespInfo)
	)
	if strings.TrimSpace(content) == "" || strings.TrimSpace(mobile) == "" {
		err = errors.New("param `content || mobile` empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sms/single_send.json")
	singleSendInfo = &YunpianSingleSendInfo{
		ApiKey:      t.SingleApiKey,
		Mobile:      mobile,
		Text:        content,
		CallbackUrl: t.ReceiverHttpApi,
	}
	body, _ = json.Marshal(*singleSendInfo)
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "SendSingleSms.")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	// bodyData 解析
	if err = jsoniter.Unmarshal(bodyData, respInfo); err != nil {
		err = errors.Wrap(err, "SendSingleSms")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	if respInfo.Code != 0 {
		err = t.getErrorMessage(respInfo.Code)
		retcode = respInfo.Code
		return
	}
	fee = int(respInfo.Fee * 100) // 转化为分
	msgid = fmt.Sprintf("%d", respInfo.Sid)
	count = respInfo.Count
	return
}

// 1.2 批量发送相同内容 https://sms.yunpian.com/v2/sms/batch_send.json
func (t *YunpianInfo) SendBatchSms(content string, mobiles []string) (count int, totalFee int, retcode int, err error) {
	Logger.Info("enter SendBatchSms.")
	defer Logger.Info("left SendBatchSms.")
	var (
		batchSmsRespInfo *BatchSmsSendRespInfo = new(BatchSmsSendRespInfo)
		body, bodyData   []byte
	)
	if strings.TrimSpace(content) == "" || mobiles == nil || len(mobiles) <= 0 {
		err = errors.New("param `content || mobiles` empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sms/batch_send.json")
	batchSmsInfo := &YunpianSingleSendInfo{
		ApiKey:      t.GroupApiKey,
		Mobile:      strings.Join(mobiles, ","),
		Text:        content,
		CallbackUrl: t.ReceiverHttpApi,
	}
	body, _ = json.Marshal(*batchSmsInfo)
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "SendBatchSms")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, batchSmsRespInfo); err != nil {
		err = errors.Wrap(err, "SendBatchSms")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	for index := 0; batchSmsRespInfo.Datas != nil && index < len(batchSmsRespInfo.Datas); index++ {
		if batchSmsRespInfo.Datas[index].Code != 0 {
			err = t.getErrorMessage(batchSmsRespInfo.Datas[index].Code)
			retcode = batchSmsRespInfo.Datas[index].Code
			return
		}
	}
	count = batchSmsRespInfo.TotalCount
	totalFeeTemp, _ := strconv.ParseFloat(batchSmsRespInfo.TotalFee, 64)
	totalFee = int(totalFeeTemp * 100)
	return
}

func (t *YunpianInfo) SendMultiSms(contents []string, mobiles []string) (count int, totalFee int, retcode int, err error) {
	Logger.Info("enter SendMultiSms.")
	defer Logger.Info("left SendMultiSms.")
	var (
		content          string                // 短信内容，多个短信内容请使用UTF-8做urlencode；使用逗号分隔，一次不要超过1000条且短信内容条数必须与手机号个数相等
		multiSmsRespInfo *BatchSmsSendRespInfo = new(BatchSmsSendRespInfo)
		body, bodyData   []byte
	)
	// 短信内容条数必须与手机号个数相等
	if contents == nil || len(contents) <= 0 || mobiles == nil || len(mobiles) <= 0 || len(contents) != len(mobiles) {
		err = errors.New("param `contents || mobiles` empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	for index := 0; index < len(contents); index++ {
		if strings.TrimSpace(contents[index]) == "" || strings.TrimSpace(mobiles[index]) == "" {
			err = errors.New("param `content || mobile` len(mobile||content)<=0")
			retcode = utils.SOURCE_DATA_ILLEGAL
			return
		}
		if content == "" {
			content = fmt.Sprintf("%s", url.QueryEscape(contents[index]))
		} else {
			content = fmt.Sprintf("%s,%s", content, url.QueryEscape(contents[index]))
		}
	}
	if len(mobiles) > 1000 {
		err = errors.New("param the length `mobiles` beyond max 1000")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	multiSmsInfo := &YunpianSingleSendInfo{
		ApiKey:      t.GroupApiKey,
		Mobile:      strings.Join(mobiles, ","),
		Text:        content,
		CallbackUrl: t.ReceiverHttpApi,
	}
	body, _ = json.Marshal(*multiSmsInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sms/multi_send.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "SendMultiSms")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, multiSmsRespInfo); err != nil {
		err = errors.Wrap(err, "SendMultiSms")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	for index := 0; multiSmsRespInfo.Datas != nil && index < len(multiSmsRespInfo.Datas); index++ {
		if multiSmsRespInfo.Datas[index].Code != 0 {
			err = t.getErrorMessage(multiSmsRespInfo.Datas[index].Code)
			retcode = multiSmsRespInfo.Datas[index].Code
			return
		}
	}
	count = multiSmsRespInfo.TotalCount
	totalFeeTemp, _ := strconv.ParseFloat(multiSmsRespInfo.TotalFee, 64)
	totalFee = int(totalFeeTemp * 100)
	return
}

// 推送状态报告
// 说明：参数名 = 经过urlencode编码的数据
//      curl --data "sms_status=url_encode_json" http://your_receive_url_address
/*
	sms_status:[
		{
			"sid": 9527, //短信id （数据类型：64位整型，对应Java和C#的long，不可用int解析)
			"uid": null, //用户自定义id
			"user_receive_time": "2014-03-17 22:55:21", //用户接受时间
			"error_msg": "", //运营商返回的代码，如："DB:0103"
			"mobile": "15205201314", //接受手机号
			"report_status": "SUCCESS" //接收状态有:SUCCESS/FAIL
		},
		{
			......
		},
	]
*/

// 云片回执推送报告结构体
type YunpianReceiptInfo struct {
	Sid             int64     `json:"sid"`
	UserReceiveTime time.Time `json:"user_receive_time"`
	ErrMsg          string    `json:"error_msg"`
	Mobile          string    `json:"mobile"`
	ReportStatus    string    `json:"report_status"`
}

type YunpianReceipt struct {
	SmsStatus []YunpianReceiptInfo `json:"sms_status"`
}

func (t *YunpianInfo) ReceivedNotification(yunpianReceipt *YunpianReceipt) (retcode int, err error) {
	Logger.Info("enter ReceivedNotification.")
	defer Logger.Info("left ReceivedNotification.")
	var (
		smsReceiptFailedRecord *SmsReceiptFailedRecords
	)
	if yunpianReceipt == nil {
		Logger.Error("urlencode need to parse.")
		return
	}
	o := orm.NewOrm()
	for index := 0; index < len(yunpianReceipt.SmsStatus); index++ {
		if yunpianReceipt.SmsStatus[index].ReportStatus == "SUCCESS" {
			smsReceiptFailedRecord = &SmsReceiptFailedRecords{
				MessageId:     fmt.Sprintf("%d", yunpianReceipt.SmsStatus[index].Sid),
				Mobile:        yunpianReceipt.SmsStatus[index].Mobile,
				ReceiptStatus: 12, // FAIL： 在创蓝253的错误码短消息是不可达的
				ReceiptAt:     yunpianReceipt.SmsStatus[index].UserReceiveTime.Format("20060102150405"),
			}
			retcode, err = smsReceiptFailedRecord.InsertSmsReceiptFailedRecordNoLock(&o)
			if err != nil {
				err = errors.Wrap(err, "ReceivedNotification")
				return
			}
		}
	}
	return
}

type YunpianTempateInfo struct {
	ApiKey     string `json:"apikey"`
	TplContent string `json:"tpl_content"`
	NotifyType int16  `json:"notify_type"`
}

type YunpianTemplateRespInfo struct {
	TplId       int64  `json:"tpl_id"`       // 模板id
	TplContent  string `json:"tpl_content"`  // 模板内容：【云片网】您的验证码是#code#"
	CheckStatus string `json:"check_status"` // 审核状态：CHECKING/SUCCESS/FAIL
	Reason      string `json:"reason"`       // 审核未通过的原因
}

// 2. 模板接口列表
// 2.1 添加模板
func (t *YunpianInfo) InsertSmsTemplate(templateContent string, notifyType int16) (tplId int64, checkStatus int, reason string, retcode int, err error) {
	Logger.Info("enter InsertSmsTemplate.")
	defer Logger.Info("left InsertSmsTemplate.")
	var (
		body, bodyData          []byte
		yunpianTemplateRespInfo *YunpianTemplateRespInfo = new(YunpianTemplateRespInfo)
	)
	if templateContent == "" || notifyType < 0 || notifyType > 3 {
		err = errors.New("param `template content | notify_type`  empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	yunpianTplInfo := &YunpianTempateInfo{
		ApiKey:     t.SingleApiKey,
		TplContent: templateContent,
		NotifyType: notifyType,
	}
	body, _ = json.Marshal(*yunpianTplInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/tpl/add.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "InsertSmsTemplate")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, yunpianTemplateRespInfo); err != nil {
		err = errors.Wrap(err, "InsertSmsTemplate")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	checkStatus = t.getCheckStatus(yunpianTemplateRespInfo.CheckStatus)
	reason = yunpianTemplateRespInfo.Reason
	tplId = yunpianTemplateRespInfo.TplId
	return
}

// 取指定模板
func (t *YunpianInfo) GetTemplateByTplId(tplId int64) (resp *YunpianTemplateRespInfo, retcode int, err error) {
	Logger.Info("[%v] enter GetTemplateByTplId.", tplId)
	defer Logger.Info("[%v] left GetTemplateByTplId.", tplId)
	type YunpianTemplateInfo struct {
		ApiKey string `json:"apikey"`
		TplId  int64  `json:"tpl_id"`
	}
	var (
		yunpianTplInfo *YunpianTemplateInfo
		body, bodyData []byte
	)
	resp = new(YunpianTemplateRespInfo)
	yunpianTplInfo = &YunpianTemplateInfo{
		ApiKey: t.SingleApiKey,
		TplId:  tplId,
	}
	body, _ = json.Marshal(*yunpianTplInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/tpl/get.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "GetTemplateByTplId")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, resp); err != nil {
		err = errors.Wrap(err, "GetTemplateByTplId")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	return
}

// 获取云片网账户下所有模板
func (t *YunpianInfo) GetAllTemplates() (resp []*YunpianTemplateRespInfo, retcode int, err error) {
	Logger.Info("enter GetAllTemplates.")
	defer Logger.Info("left GetAllTemplates.")
	type YunpianTemplateInfo struct {
		ApiKey string `json:"apikey"`
		TplId  int64  `json:"tpl_id"`
	}
	var (
		yunpianTplInfo *YunpianTemplateInfo
		body, bodyData []byte
	)
	yunpianTplInfo = &YunpianTemplateInfo{
		ApiKey: t.SingleApiKey,
	}
	body, _ = json.Marshal(*yunpianTplInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/tpl/get.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "GetTemplateByTplId")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, resp); err != nil {
		err = errors.Wrap(err, "GetTemplateByTplId")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	return
}

// 修改模版
func (t *YunpianInfo) ModifyTemplate(tplId int64, tplContent string) (checkStatus int, reason string, retcode int, err error) {
	Logger.Info("[%v] enter ModifyTemplate.", tplId)
	defer Logger.Info("[%v] left ModifyTemplate.", tplId)
	type TemplateInfo struct {
		ApiKey     string `json:"apikey"`
		TplId      int64  `json:"tpl_id"`
		TplContent string `json:"tpl_content"`
	}
	var (
		templateInfo   *TemplateInfo
		resp           *YunpianTemplateRespInfo = new(YunpianTemplateRespInfo)
		body, bodyData []byte
	)
	if tplId <= 0 || tplContent == "" {
		err = errors.New("param `template_id | content` empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	templateInfo = &TemplateInfo{
		ApiKey:     t.SingleApiKey,
		TplId:      tplId,
		TplContent: tplContent,
	}
	body, _ = json.Marshal(*templateInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/tpl/update.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "ModifyTemplate")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, resp); err != nil {
		err = errors.Wrap(err, "ModifyTemplate")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	checkStatus = t.getCheckStatus(resp.CheckStatus)
	reason = resp.Reason
	return
}

// 删除模板
func (t *YunpianInfo) DeleteTemplate(tplId int64) (checkStatus int, reason string, retcode int, err error) {
	Logger.Info("[%v] enter DeleteTemplate.", tplId)
	defer Logger.Info("[%v] left DeleteTemplate.", tplId)
	type TemplateInfo struct {
		ApiKey string `json:"apikey"`
		TplId  int64  `json:"tpl_id"`
	}
	var (
		body, bodyData []byte
		resp           *YunpianTemplateRespInfo = new(YunpianTemplateRespInfo)
	)
	if tplId <= 0 {
		err = errors.New("param `template_id` empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	templateInfo := &TemplateInfo{
		ApiKey: t.SingleApiKey,
		TplId:  tplId,
	}
	body, _ = json.Marshal(*templateInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/tpl/del.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "DeleteTemplate")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, resp); err != nil {
		err = errors.Wrap(err, "DeleteTemplate")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	checkStatus = t.getCheckStatus(resp.CheckStatus)
	reason = resp.Reason
	return
}

// 3.签名接口
//   - 3.1 添加签名
/*
	@param sign 签名内容
	@param notify 是否短信通知结果，默认true
	@param applyVip 是否申请专用通道，默认false
	@param isOnlyGlobal 是否仅发国际短信，默认false
	@param industryType 所属行业，默认“其它”
*/
func (t *YunpianInfo) InsertSign(sign string, notify bool, applyVip bool, industry string) (checkStatus int, retcode int, err error) {
	Logger.Info("[%v] enter InsertSign.", sign)
	defer Logger.Info("[%v] left InsertSign.", sign)
	type SignInfo struct {
		ApiKey       string `json:"apikey"`
		Sign         string `json:"sign"`
		Notify       bool   `json:"notify"`
		ApplyVip     bool   `json:"applyVip"`
		IsOnlyGlobal bool   `json:"isOnlyGlobal"`
		Industry     string `json:"industryType"`
	}
	var (
		body    []byte
		retJson map[string]interface{}
		isMap   bool = false
	)
	signInfo := &SignInfo{
		ApiKey:       t.SingleApiKey,
		Sign:         sign,
		Notify:       notify,
		ApplyVip:     applyVip,
		IsOnlyGlobal: false,
		Industry:     industry,
	}
	body, _ = json.Marshal(*signInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sign/add.json")
	if retJson, err = httpRequest.HttpPostJson(httpStr, body); err != nil {
		err = errors.Wrap(err, "InsertSign")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if retJson["code"].(int) == 0 {
		if retJson, isMap = ConvertInterfaceToMap(retJson["sign"]); !isMap {
			err = errors.New("http parse failed.")
			retcode = utils.JSON_PARSE_FAILED
			return
		}
		checkStatus = t.getCheckStatus(retJson["apply_state"].(string))
		return
	} else {
		retcode = retJson["code"].(int)
		err = errors.New(retJson["detail"].(string))
		return
	}
	return
}

// 修改签名
func (t *YunpianInfo) UpdateSign(oldSign string, newSign string, notify bool, applyVip bool, industry string) (checkStatus int, retcode int, err error) {
	Logger.Info("[%v] enter UpdateSign.", newSign)
	defer Logger.Info("[%v] left UpdateSign.", newSign)
	type SignInfo struct {
		ApiKey       string `json:"apikey"`
		OldSign      string `json:"oldSign"`
		Sign         string `json:"sign"`
		Notify       bool   `json:"notify"`
		ApplyVip     bool   `json:"applyVip"`
		IsOnlyGlobal bool   `json:"isOnlyGlobal"`
		Industry     string `json:"industryType"`
	}
	var (
		retJson map[string]interface{} = map[string]interface{}{}
		body    []byte
		isMap   bool = false
	)
	signInfo := &SignInfo{
		ApiKey:       t.SingleApiKey,
		OldSign:      oldSign,
		Sign:         newSign,
		Notify:       notify,
		ApplyVip:     applyVip,
		IsOnlyGlobal: false,
		Industry:     industry,
	}
	body, _ = json.Marshal(*signInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sign/update.json")
	if retJson, err = httpRequest.HttpPostJson(httpStr, body); err != nil {
		err = errors.Wrap(err, "UpdateSign")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if retJson["code"].(int) == 0 {
		if retJson, isMap = ConvertInterfaceToMap(retJson["sign"]); !isMap {
			err = errors.New("http parse failed.")
			retcode = utils.JSON_PARSE_FAILED
			return
		}
		checkStatus = t.getCheckStatus(retJson["apply_state"].(string))
		return
	} else {
		retcode = retJson["code"].(int)
		err = errors.New(retJson["msg"].(string))
		return
	}
	return
}

type YunpianSignInfo struct {
	Chan        string `json:"chan"`          // 通道类型，'NONE'暂未分配，'GLOBAL'国际短信通道，'MARKET'营销通道，'VIP'专用通道，'NORMAL'普通通道
	CheckStatus string `json:"check_status"`  // 当前状态,'CHECKING'审核中，'FAIL'审核失败，'SUCCESS'审核成功，'APLLYING_VIP'普通通道审核成功申请升级专用通道
	Enabled     bool   `json:"enabled"`       // 当前签名是否启用
	Extend      string `json:"extend"`        // 扩展号，为空表示暂未分配
	Industry    string `json:"industry_type"` // "商业服务",行业
	OnlyGlobal  bool   `json:"only_global"`   // 是否用于国际短信
	Remark      string `json:"remark"`        // 客服给的审核结果解释，一般见于审核失败
	Sign        string `json:"sign"`          // 签名
	Vip         bool   `json:"vip"`           // 是否专用通道
}
type YunpianSignRespInfo struct {
	Code  int               `json:"code"`
	Total int               `json:"total"`
	Sign  []YunpianSignInfo `json:"sign"`
}

// 搜索签名
func (t *YunpianInfo) SearchSign(sign string, pageIndex int64, pageSize int64) (yunpianSignInfos []YunpianSignInfo, count int, retcode int, err error) {
	Logger.Info("[%v] enter SearchSign.", sign)
	defer Logger.Info("[%v] left SearchSign.", sign)
	type SignInfo struct {
		ApiKey    string `json:"apikey"`
		Sign      string `json:"sign"`
		PageIndex int64  `json:"pageNo"`
		PageSize  int64  `json:"pageSize"`
	}
	var (
		signResp       *YunpianSignRespInfo = new(YunpianSignRespInfo)
		body, bodyData []byte
	)
	signInfo := &SignInfo{
		ApiKey:    t.SingleApiKey,
		Sign:      sign,
		PageIndex: pageIndex,
		PageSize:  pageSize,
	}
	body, _ = json.Marshal(*signInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sign/get.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "SearchSign")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, signResp); err != nil {
		err = errors.Wrap(err, "SearchSign")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	if signResp.Code != 0 {
		retcode = signResp.Code
		err = t.getErrorMessage(signResp.Code)
		return
	}
	count = signResp.Total
	yunpianSignInfos = signResp.Sign
	return
}

type YunpianSendRecordInfo struct {
	MsgId           string    `json:"sid"`
	Mobile          string    `json:"mobile"`
	SendTime        time.Time `json:"send_time"`
	Content         string    `json:"text"`
	SendStatus      string    `json:"send_status"`
	ReportStatus    string    `json:"report_status"`
	Fee             int       `json:"report_status"`
	UserReceiveTime time.Time `json:"user_receive_time"`
	ErrMsg          string    `json:"error_msg"`
}

// 查短信发送记录
func (t *YunpianInfo) GetRecords(searchMobile string, startTime time.Time, endTime time.Time, pageIndex int64, pageSize int64) (infos []YunpianSendRecordInfo, retcode int, err error) {
	Logger.Info("enter GetRecords.")
	defer Logger.Info("enter GetRecords.")
	type SearchingInfo struct {
		ApiKey    string `json:"apikey"`
		Mobile    string `json:"mobile"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		PageIndex int64  `json:"page_num"`
		PageSize  int64  `json:"page_size"`
	}
	var (
		body, bodyData []byte
	)
	searchingInfo := &SearchingInfo{
		ApiKey:    t.SingleApiKey,
		Mobile:    searchMobile,
		StartTime: startTime.Format("2006-01-02 15:04:05"),
		EndTime:   endTime.Format("2006-01-02 15:04:05"),
		PageIndex: pageIndex,
		PageSize:  pageSize,
	}
	body, _ = json.Marshal(*searchingInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sms/get_record.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "GetRecords")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	if err = jsoniter.Unmarshal(bodyData, &infos); err != nil {
		err = errors.Wrap(err, "GetRecords")
		retcode = utils.JSON_PARSE_FAILED
		return
	}
	return
}

// 查屏蔽词
func (t *YunpianInfo) CheckBlackWord(content string) (blackWords []string, retcode int, err error) {
	Logger.Info("enter CheckBlackWord.")
	defer Logger.Info("left CheckBlackWord.")
	type BlackInfo struct {
		ApiKey  string `json:"apikey"`
		Content string `json:"text"`
	}
	var (
		body, bodyData []byte
	)
	blackInfo := &BlackInfo{
		ApiKey:  t.SingleApiKey,
		Content: content,
	}
	body, _ = json.Marshal(*blackInfo)
	httpStr := fmt.Sprintf("https://sms.yunpian.com/v2/sms/get_black_word.json")
	if bodyData, err = httpRequest.HttpPostBody(httpStr, body); err != nil {
		err = errors.Wrap(err, "CheckBlackWord")
		retcode = utils.HTTP_CALL_FAILD_EXTERNAL
		return
	}
	blackWords = append(blackWords, strings.Split(string(bodyData), ",")...)
	return
}
func (t *YunpianInfo) getCheckStatus(code string) (status int) {
	switch code {
	case "CHECKING":
		status = TEMPLATE_SMS_CHECKING
	case "SUCCESS":
		status = TEMPLATE_SMS_SUCCESS
	case "FAIL":
		status = TEMPLATE_SMS_FAIL
	}
	return
}

func (t *YunpianInfo) getErrorMessage(errcode int) (err error) {
	var (
		message string
	)
	switch errcode {
	case -1:
		message = "云片网短信错误：非法的apikey"
	case -2:
		message = "云片网短信错误：API没有权限"
	case -3:
		message = "云片网短信错误：IP没有权限"
	case -4:
		message = "云片网短信错误：访问次数超限"
	case -5:
		message = "云片网短信错误：访问频率超限"
	case -50:
		message = "云片网短信错误：未知异常"
	case -51:
		message = "云片网短信错误：系统繁忙"
	case -52:
		message = "云片网短信错误：充值失败"
	case -53:
		message = "云片网短信错误：提交短信失败"
	case -54:
		message = "云片网短信错误：记录已存在"
	case -55:
		message = "云片网短信错误：记录不存在"
	case -57:
		message = "云片网短信错误：用户开通过固定签名功能，但签名未设置"
	case 1:
		message = "云片网短信错误：请求参数缺失"
	case 2:
		message = "云片网短信错误：请求参数格式错误"
	case 3:
		message = "云片网短信错误：账户余额不足"
	case 4:
		message = "云片网短信错误：关键词屏蔽"
	case 5:
		message = "云片网短信错误：未找到对应id的模板"
	case 6:
		message = "云片网短信错误：添加模板失败"
	case 7:
		message = "云片网短信错误：模板不可用"
	case 8:
		message = "云片网短信错误：同一手机号30秒内重复提交相同的内容"
	case 9:
		message = "云片网短信错误：同一手机号5分钟内重复提交相同的内容超过3次"
	case 10:
		message = "云片网短信错误：手机号黑名单过滤"
	case 11:
		message = "云片网短信错误：接口不支持GET方式调用"
	case 12:
		message = "云片网短信错误：接口不支持POST方式调用"
	case 13:
		message = "云片网短信错误：营销短信暂停发送"
	case 14:
		message = "云片网短信错误：解码失败"
	case 15:
		message = "云片网短信错误：签名不匹配"
	case 16:
		message = "云片网短信错误：签名格式不正确"
	case 17:
		message = "云片网短信错误：24小时内同一手机号发送次数超过限制"
	case 18:
		message = "云片网短信错误：签名校验失败"
	case 19:
		message = "云片网短信错误：请求已失效"
	case 20:
		message = "云片网短信错误：不支持的国家地区"
	case 21:
		message = "云片网短信错误：解密失败"
	case 22:
		message = "云片网短信错误：1小时内同一手机号发送次数超过限制"
	case 23:
		message = "云片网短信错误：发往模板支持的国家列表之外的地区"
	case 24:
		message = "云片网短信错误：添加告警设置失败"
	case 25:
		message = "云片网短信错误：手机号和内容个数不匹配"
	case 26:
		message = "云片网短信错误：流量包错误"
	case 27:
		message = "云片网短信错误：未开通金额计费"
	case 28:
		message = "云片网短信错误：运营商错误"
	case 33:
		message = "云片网短信错误：超过频率"
	}
	if message != "" {
		err = errors.New(message)
	}
	return
}
