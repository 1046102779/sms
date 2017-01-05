package models

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
)

var (
	// 服务是否已启用:10: 未启用；20：已启用
	SMS_SERVICE_INVALID = 10
	SMS_SERVICE_VALID   = 20

	// 10:创蓝253短信提供商；20: 云片网短信提供商
	SMS_SERVICE_PROVIDER_TYPE_253_CHUANGLAN = 10
	SMS_SERVICE_PROVIDER_TYPE_YUNPIAN       = 10
)

type SmsServiceProviders struct {
	Id                 int       `orm:"column(sms_service_provider_id);auto"`
	Type               int16     `orm:"column(type);null"`
	Name               string    `orm:"column(name);size(100);null"`
	Code               string    `orm:"column(code);size(50);null"`
	SignName           string    `orm:"column(sign_name);size(50);null"`
	SingleSmsMaxLength int       `orm:"column(single_sms_max_length);null"`
	IsValid            int16     `orm:"column(is_valid);null"`
	Status             int16     `orm:"column(status);null"`
	UpdatedAt          time.Time `orm:"column(updated_at);type(datetime);null"`
	CreatedAt          time.Time `orm:"column(created_at);type(datetime);null"`
}

func (t *SmsServiceProviders) TableName() string {
	return "sms_service_providers"
}

func init() {
	orm.RegisterModel(new(SmsServiceProviders))
}

// GetAllSmsServiceProviders retrieves all SmsServiceProviders matches certain condition. Returns empty list if
// no records exist
func GetAllSmsServiceProviders(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(SmsServiceProviders))
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

	var l []SmsServiceProviders
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
