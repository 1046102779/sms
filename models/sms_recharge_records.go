package models

import (
	"reflect"
	"strings"
	"time"

	utils "github.com/1046102779/common"
	. "github.com/1046102779/sms/logger"
	"github.com/astaxie/beego/orm"
	"github.com/pkg/errors"
)

var (
	// 短信充值支付状态：
	SMS_PAY_TOBEPAY = 10 // 未支付
	SMS_PAY_PAYED   = 20 // 已支付

	// 10: 微信公众号支付 Native, 11：微信公众号支付 JSAPI, 12: 微信公众号支付 APP
	WECHAT_TRADE_TYPE_NATIVE = 10
	WECHAT_TRADE_TYPE_JSAPI  = 11
	WECHAT_TRADE_TYPE_APP    = 12
)

type SmsRechargeRecords struct {
	Id            int       `orm:"column(sms_recharge_record_id);auto"`
	CompanyId     int       `orm:"column(company_id);null"`
	UserId        int       `orm:"column(user_id);null"`
	RechargeMoney int       `orm:"column(recharge_money);null"`
	OutTradeNo    string    `orm:"column(out_trade_no);size(50);null"`
	TransactionId string    `orm:"column(transaction_id);size(100);null"`
	PayType       int16     `orm:"column(pay_type);null"`
	PayStatus     int16     `orm:"column(pay_status);null"`
	Status        int16     `orm:"column(status);null"`
	UpdatedAt     time.Time `orm:"column(updated_at);type(datetime);null"`
	CreatedAt     time.Time `orm:"column(created_at);type(datetime);null"`
}

func (t *SmsRechargeRecords) TableName() string {
	return "sms_recharge_records"
}

func (t *SmsRechargeRecords) UpdateSmsRechargeInfoNoLock(o *orm.Ormer) (retcode int, err error) {
	Logger.Info("[%v] enter UpdateSmsRechargeInfoNoLock.", t.Id)
	defer Logger.Info("[%v] enter UpdateSmsRechargeInfoNoLock.", t.Id)
	if o == nil {
		err = errors.New("param `orm.Ormer` ptr empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	if _, err = (*o).Update(t); err != nil {
		err = errors.Wrap(err, "UpdateSmsRechargeInfoNoLock")
		retcode = utils.DB_UPDATE_ERROR
		return
	}
	return
}
func (t *SmsRechargeRecords) InsertSmsRechargeRecordNoLock(o *orm.Ormer) (retcode int, err error) {
	Logger.Info("[%v.%v] enter InsertSmsRechargeRecordNoLock.", t.CompanyId, t.UserId)
	defer Logger.Info("[%v.%v] enter InsertSmsRechargeRecordNoLock.", t.CompanyId, t.UserId)
	if o == nil {
		err = errors.New("param `orm.Ormer` ptr empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	if _, err = (*o).Insert(t); err != nil {
		err = errors.Wrap(err, "InsertSmsRechargeRecordNoLock")
		retcode = utils.DB_INSERT_ERROR
		return
	}
	return
}

// 通过订单号out_trade_no，更新短信订单
func (t *SmsRechargeRecords) UpdateSmsRechargeInfoByOutTradeNoNoLock(o *orm.Ormer) (retcode int, err error) {
	Logger.Info("[%v] enter UpdateSmsRechargeInfoByOutTradeNoNoLock.", t.OutTradeNo)
	defer Logger.Info("[%v] left UpdateSmsRechargeInfoByOutTradeNoNoLock.", t.OutTradeNo)
	var (
		smsRechargeRecords []SmsRechargeRecords = []SmsRechargeRecords{}
		num                int64
	)
	now := time.Now()
	num, err = (*o).QueryTable(t.TableName()).Filter("out_trade_no", t.OutTradeNo).Filter("pay_status", SMS_PAY_TOBEPAY).Filter("status", utils.STATUS_VALID).All(&smsRechargeRecords)
	if err != nil {
		err = errors.Wrap(err, "UpdateSmsRechargeInfoByOutTradeNoNoLock")
		retcode = utils.DB_READ_ERROR
		return
	}
	if num > 0 {
		smsRechargeRecords[0].PayStatus = int16(SMS_PAY_PAYED)
		smsRechargeRecords[0].UpdatedAt = now
		smsRechargeRecords[0].RechargeMoney = t.RechargeMoney
		smsRechargeRecords[0].TransactionId = t.TransactionId
		if retcode, err = smsRechargeRecords[0].UpdateSmsRechargeInfoNoLock(o); err != nil {
			err = errors.Wrap(err, "UpdateSmsRechargeInfoByOutTradeNoNoLock")
			return
		}
	}
	return
}

func init() {
	orm.RegisterModel(new(SmsRechargeRecords))
}

// GetAllSmsRechargeRecords retrieves all SmsRechargeRecords matches certain condition. Returns empty list if
// no records exist
func GetAllSmsRechargeRecords(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(SmsRechargeRecords))
	// query k=v
	for k, v := range query {
		// rewrite dot-notation to Object__Attribute
		k = strings.Replace(k, ".", "__", -1)
		if strings.Contains(k, "isnull") {
			qs = qs.Filter(k, (v == "true" || v == "1"))
		} else {
			qs = qs.Filter(k, v)
		}
	}
	// order by:
	var sortFields []string
	if len(sortby) != 0 {
		if len(sortby) == len(order) {
			// 1) for each sort field, there is an associated order
			for i, v := range sortby {
				orderby := ""
				if order[i] == "desc" {
					orderby = "-" + v
				} else if order[i] == "asc" {
					orderby = v
				} else {
					return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
			qs = qs.OrderBy(sortFields...)
		} else if len(sortby) != len(order) && len(order) == 1 {
			// 2) there is exactly one order, all the sorted fields will be sorted by this order
			for _, v := range sortby {
				orderby := ""
				if order[0] == "desc" {
					orderby = "-" + v
				} else if order[0] == "asc" {
					orderby = v
				} else {
					return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
		} else if len(sortby) != len(order) && len(order) != 1 {
			return nil, errors.New("Error: 'sortby', 'order' sizes mismatch or 'order' size is not 1")
		}
	} else {
		if len(order) != 0 {
			return nil, errors.New("Error: unused 'order' fields")
		}
	}

	var l []SmsRechargeRecords
	qs = qs.OrderBy(sortFields...)
	if _, err = qs.Limit(limit, offset).All(&l, fields...); err == nil {
		if len(fields) == 0 {
			for _, v := range l {
				ml = append(ml, v)
			}
		} else {
			// trim unused fields
			for _, v := range l {
				m := make(map[string]interface{})
				val := reflect.ValueOf(v)
				for _, fname := range fields {
					m[fname] = val.FieldByName(fname).Interface()
				}
				ml = append(ml, m)
			}
		}
		return ml, nil
	}
	return nil, err
}
