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
	TEMPLATE_SMS_CHECKING = 10 // 模板审核中
	TEMPLATE_SMS_SUCCESS  = 20 // 模板审核通过
	TEMPLATE_SMS_FAIL     = 30 // 模板审核拒绝

	//  短信模板编码
	MOBILE_VERIFICATION_CODE_CONTENT = "MOBILE_VERIFICATION_CODE_CONTENT"
)

type SmsTemplates struct {
	Id                   int       `orm:"column(sms_template_id);auto"`
	SmsServiceProviderId int       `orm:"column(sms_service_provider_id);null"`
	TemplateName         string    `orm:"column(template_name);size(50);null"`
	TemplateContent      string    `orm:"column(template_content);size(1000);null"`
	CheckStatus          int16     `orm:"column(check_status);null"`
	Status               int16     `orm:"column(status);null"`
	UpdatedAt            time.Time `orm:"column(updated_at);type(datetime);null"`
	CreatedAt            time.Time `orm:"column(created_at);type(datetime);null"`
}

func (t *SmsTemplates) TableName() string {
	return "sms_templates"
}

func (t *SmsTemplates) ReadSmsTemplateNoLock(o *orm.Ormer) (retcode int, err error) {
	Logger.Info("[%v] enter ReadSmsTemplateNoLock.", t.Id)
	defer Logger.Info("[%v] left ReadSmsTemplateNoLock.", t.Id)
	if o == nil {
		err = errors.New("param `orm.Ormer` ptr empty")
		retcode = utils.SOURCE_DATA_ILLEGAL
		return
	}
	if err = (*o).Read(t); err != nil {
		err = errors.Wrap(err, "ReadSmsTemplateNoLock")
		retcode = utils.DB_READ_ERROR
		return
	}
	return
}
func init() {
	orm.RegisterModel(new(SmsTemplates))
}

func GetSmsTemplate(smsServiceProviderId int, templateName string) (template *SmsTemplates, retcode int, err error) {
	Logger.Info("[%v.%v] enter GetSmsTemplate.", smsServiceProviderId, templateName)
	defer Logger.Info("[%v.%v] left GetSmsTemplate.", smsServiceProviderId, templateName)
	var (
		num       int64
		templates []SmsTemplates = []SmsTemplates{}
	)
	o := orm.NewOrm()
	num, err = o.QueryTable((&SmsTemplates{}).TableName()).Filter("sms_service_provider_id", smsServiceProviderId).Filter("template_name", templateName).Filter("check_status", TEMPLATE_SMS_SUCCESS).Filter("status", utils.STATUS_VALID).All(&templates)
	if err != nil {
		err = errors.Wrap(err, "GetSmsTemplate")
		retcode = utils.DB_READ_ERROR
		return
	}
	if num > 0 {
		return &templates[0], 0, nil
	}
	return
}

// GetAllSmsTemplates retrieves all SmsTemplates matches certain condition. Returns empty list if
// no records exist
func GetAllSmsTemplates(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(SmsTemplates))
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

	var l []SmsTemplates
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
