package models

// 公共短信接口列表,类似多态
type ISMS interface {
	SendSMS(interface{}) (int, error)
}
