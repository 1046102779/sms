package models

import (
	"errors"
	"reflect"
	"strings"

	utils "github.com/1046102779/common"
	. "github.com/1046102779/sms/logger"
	"github.com/astaxie/beego/orm"
)

type SmsReceiptFailedRecords struct {
	Id            int    `orm:"column(sms_receipt_failed_record_id);pk"`
	MessageId     string `orm:"column(message_id);size(100);null"`
	Mobile        string `orm:"column(mobile);size(20);null"`
	ReceiptStatus int16  `orm:"column(receipt_status);null"`
	ReceiptAt     string `orm:"column(receipt_at);size(30);null"`
}

func (t *SmsReceiptFailedRecords) TableName() string {
	return "sms_receipt_failed_records"
}

func (t *SmsReceiptFailedRecords) InsertSmsReceiptFailedRecordNoLock(o *orm.Ormer) (retcode int, err error) {
	Logger.Info("[%v] enter InsertSmsReceiptFailedRecordNoLock.", t.MessageId)
	defer Logger.Info("[%v] left InsertSmsReceiptFailedRecordNoLock.", t.MessageId)
	if o == nil {
		err = errors.New("param `orm.Ormer` ptr empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	if _, err = (*o).Insert(t); err != nil {
		retcode = utils.DB_INSERT_ERROR
		return
	}
	return
}

func init() {
	orm.RegisterModel(new(SmsReceiptFailedRecords))
}

// GetAllSmsReceiptFailedRecords retrieves all SmsReceiptFailedRecords matches certain condition. Returns empty list if
// no records exist
func GetAllSmsReceiptFailedRecords(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(SmsReceiptFailedRecords))
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

	var l []SmsReceiptFailedRecords
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
