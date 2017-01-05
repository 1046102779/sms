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

type SmsSendRecords struct {
	Id              int       `orm:"column(sms_send_record_id);auto"`
	SmsTemplateId   int       `orm:"column(sms_template_id);null"`
	CompanyId       int       `orm:"column(company_id);null"`
	Content         string    `orm:"column(content);size(1000);null"`
	ReceiverMobiles string    `orm:"column(receiver_mobiles);size(2000);null"`
	SendStatus      string    `orm:"column(send_status);size(20);null"`
	Count           int       `orm:"column(count);null"`
	CountPerContent int16     `orm:"column(count_per_content);null"`
	MessageId       string    `orm:"column(message_id);size(100);null"`
	SendAt          time.Time `orm:"column(send_at);type(datetime);null"`
}

func (t *SmsSendRecords) TableName() string {
	return "sms_send_records"
}

func (t *SmsSendRecords) InsertSmsSendRecordNoLock(o *orm.Ormer) (retcode int, err error) {
	Logger.Info("[%v] enter InsertSmsSendRecordNoLock.", t.MessageId)
	defer Logger.Info("[%v] left InsertSmsSendRecordNoLock.", t.MessageId)
	if o == nil {
		err = errors.New("param `orm.Ormer` ptr empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	if _, err = (*o).Insert(t); err != nil {
		err = errors.Wrap(err, "InsertSmsSendRecordNoLock")
		retcode = utils.DB_INSERT_ERROR
		return
	}
	return
}
func init() {
	orm.RegisterModel(new(SmsSendRecords))
}

// GetAllSmsSendRecords retrieves all SmsSendRecords matches certain condition. Returns empty list if
// no records exist
func GetAllSmsSendRecords(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(SmsSendRecords))
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

	var l []SmsSendRecords
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
