package models

import (
	"fmt"
	"strings"

	"github.com/astaxie/beego/orm"

	utils "github.com/1046102779/common"
	. "github.com/1046102779/common/utils"
	pb "github.com/1046102779/igrpc"
	. "github.com/1046102779/sms/logger"
)

type SmsServer struct{}

func (t *SmsServer) SendSingleSms(in *pb.SmsRequest, out *pb.CodeReply) (err error) {
	Logger.Info("enter SendSingleSms.")
	defer Logger.Info("left SendSingleSms.")
	defer func() {
		err = nil
	}()
	if in.Mobiles == nil || len(in.Mobiles) <= 0 || in.Contents == nil || len(in.Contents) <= 0 {
		out.RetCode = utils.SOURCE_DATA_ILLEGAL
		out.ErrMsg = fmt.Sprintf("param `mobile | content` empty")
		return
	}
	return
}

func (t *SmsServer) UpdateSmsRechargeInfo(in *pb.SmsRechargeOrderInfo, out *pb.SmsRechargeOrderInfo) (err error) {
	Logger.Info("[%v.%v] enter UpdateSmsRechargeInfo.", in.OutTradeNo, in.Money)
	defer Logger.Info("[%v.%v] left UpdateSmsRechargeInfo.", in.OutTradeNo, in.Money)
	defer func() {
		err = nil
	}()
	o := orm.NewOrm()
	record := &SmsRechargeRecords{
		OutTradeNo:    in.OutTradeNo,
		RechargeMoney: int(in.Money),
		TransactionId: in.TransactionId,
	}
	if _, err = record.UpdateSmsRechargeInfoByOutTradeNoNoLock(&o); err != nil {
		Logger.Error(err.Error())
		return
	}
	return
}

func (t *SmsServer) CodeMatch(in *pb.CodeRequest, reply *pb.CodeReply) (err error) {
	Logger.Info("[%v.%v] enter CodeMatch", in.Mobile, in.Code)
	defer Logger.Info("[%v.%v] left CodeMatch", in.Mobile, in.Code)
	var (
		code string
	)
	if strings.TrimSpace(in.Mobile) == "" || strings.TrimSpace(in.Code) == "" {
		reply.ErrMsg = "param `mobile or code` empty!"
		return
	}
	key := "SMS:" + in.Mobile + ":LOGIN"
	if code, err = GetCode(key); err != nil {
		Logger.Error("get redis err: " + err.Error())
		*reply = pb.CodeReply{
			RetCode: utils.REDIS_GET_FAILED,
			ErrMsg:  err.Error(),
		}
		err = nil
		return
	}
	if code != in.Code {
		*reply = pb.CodeReply{
			RetCode: utils.VERIFICATION_NOT_MATCH,
			ErrMsg:  "verification code unmatched",
		}

	}
	return
}
