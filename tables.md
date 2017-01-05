# 创建短信服务库表

## 创建短信库
CREATE DATABASE IF NOT EXISTS ycfm_sms DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

### 创建发送失败记录表
```
CREATE TABLE IF NOT EXISTS `sms_receipt_failed_records` (
  `sms_receipt_failed_record_id` int(11) NOT NULL COMMENT '自增主键',
  `message_id` varchar(100) DEFAULT NULL COMMENT '第三方短信消息ID',
  `mobile` varchar(20) DEFAULT NULL COMMENT ' 手机号码',
  `receipt_status` smallint(6) DEFAULT NULL COMMENT '11:短消息超过有效期;12:短消息是不可达的;13:未知短消息状态;14:短消息被短信中心拒绝;15:目的号码是黑名单号码;;16:系统忙;17:审核驳回;18:网关内部状态',
  `receipt_at` varchar(30) DEFAULT NULL COMMENT '短信回执接收时间',
  PRIMARY KEY (`sms_receipt_failed_record_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 创建短信服务充值记录表
```
CREATE TABLE IF NOT EXISTS `sms_recharge_records` (
  `sms_recharge_record_id` int(11) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `company_id` int(11) DEFAULT NULL COMMENT '公司ID',
  `user_id` int(11) NOT NULL DEFAULT '0' COMMENT '用户ID',
  `recharge_money` int(11) DEFAULT NULL COMMENT '充值金额，单位：分',
  `out_trade_no` varchar(50) DEFAULT NULL COMMENT '订单号:时间序列',
  `transaction_id` varchar(100) DEFAULT NULL COMMENT '微信支付号',
  `pay_type` smallint(6) DEFAULT NULL COMMENT '10: 微信公众号支付 Native, 11：微信公众号支付 JSAPI, 12: 微信公众号
支付 APP',
  `pay_status` smallint(6) DEFAULT NULL COMMENT '10: 未支付, 20：已支付',
  `status` smallint(6) DEFAULT NULL COMMENT '状态：-20:逻辑删除；10: 有效',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`sms_recharge_record_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4
```

### 创建短信发送记录表
```
 CREATE TABLE IF NOT EXISTS `sms_send_records` (
  `sms_send_record_id` int(11) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `sms_template_id` int(11) DEFAULT NULL COMMENT '短信模板ID',
  `company_id` int(11) DEFAULT NULL COMMENT '公司ID',
  `content` varchar(1000) DEFAULT NULL COMMENT '短信内容',
  `receiver_mobiles` varchar(2000) DEFAULT NULL COMMENT '短信接收者手机号列表',
  `send_status` varchar(20) DEFAULT NULL COMMENT '短信发送响应状态',
  `count` int(11) DEFAULT NULL COMMENT '短信使用条数=count_per_content*receiver_mobiles',
  `count_per_content` smallint(6) DEFAULT NULL COMMENT '短信内容被分隔的条数 int  一般65个字符一条短信',
  `message_id` varchar(100) DEFAULT NULL COMMENT '第三方短信消息ID',
  `send_at` datetime DEFAULT NULL COMMENT '短信发送时间',
  PRIMARY KEY (`sms_send_record_id`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4
```

### 创建短信服务提供商表
```
CREATE TABLE IF NOT EXISTS `sms_service_providers` (
  `sms_service_provider_id` int(11) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `type` smallint(6) DEFAULT NULL COMMENT '10:创蓝253短信提供商；20: 云片网短信提供商',
  `name` varchar(100) DEFAULT NULL COMMENT '短信服务商公司名称',
  `code` varchar(50) DEFAULT NULL COMMENT '253_CHUANGLAN_SMS_SERVICE：253创蓝短信服务；YUNPIAN_SMS_SERVICE：云片网
短信服务',
  `sign_name` varchar(50) NOT NULL COMMENT '短信服务应用签名',
  `single_sms_max_length` int(11) DEFAULT NULL COMMENT '单条短信最大字符长度',
  `is_valid` smallint(6) DEFAULT NULL COMMENT '服务是否已启用:10: 未启用；20：已启用',
  `status` smallint(6) DEFAULT NULL COMMENT '状态：-20:逻辑删除；10: 有效',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`sms_service_provider_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4
```

### 短信模板表
```
CREATE TABLE IF NOT EXISTS `sms_templates` (
  `sms_template_id` int(11) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `sms_service_provider_id` int(11) DEFAULT NULL COMMENT '短信服务提供商ID',
  `template_id` int(11) NOT NULL COMMENT '第三方模板ID',
  `template_name` varchar(50) DEFAULT NULL COMMENT '短信模板名称：MOBILE_VERIFICATION_CODE_CONTENT: 短信验证码模板
名称等',
  `template_content` varchar(1000) DEFAULT NULL COMMENT '短信模板内容：MOBILE_VERIFICATION_CODE_CONTENT:【%s】%s（
动态登录验证码），请勿向任何人泄漏。',
  `check_status` smallint(6) NOT NULL COMMENT '审核状态：10: 审核中；20:审核通过；30: 审核拒绝',
  `status` smallint(6) DEFAULT NULL COMMENT '状态：-20:逻辑删除；10: 有效',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`sms_template_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4
```

## 创建全局配置库

CREATE DATABASE IF NOT EXISTS ycfm_accounts DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

###  创建全局配置表
```
CREATE TABLE IF NOT EXISTS `company_system_confs` (
  `company_system_conf_id` int(11) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `company_id` int(11) DEFAULT NULL COMMENT '公司ID',
  `param_key` varchar(100) DEFAULT NULL COMMENT '配置参数名称',
  `param_value` varchar(100) DEFAULT NULL COMMENT '配置参数值',
  `status` smallint(6) DEFAULT NULL COMMENT '状态：-20：逻辑删除；10：有效',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`company_system_conf_id`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4
```
