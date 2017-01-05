# 公共短信服务平台
该服务采用比较流行的微服务思想，主要提供创蓝253、云片网短信服务。服务列表:
* [创蓝253](https://www.253.com)短信服务提供商
       1、短信验证码
       1.1. 普通短信发送
       1.2. 状态报告推送
       1.3. 短信接收
       1.4. 额度查询接口
       2、会员营销短信
       2.1. 普通短信发送
       2.2. 状态报告推送
       2.3. 短信接收
       2.4. 额度查询接口
* [云片网](http://www.yunpian.com)短信服务：
      2. 发送短信服务列表
      2.1 [单条发送](https://sms.yunpian.com/v2/sms/single_send.json)
      2.2 [批量发送相同内容](https://sms.yunpian.com/v2/sms/batch_send.json)
      2.3 [批量发送不同内容](https://sms.yunpian.com/v2/sms/multi_send.json)
      2.4 推送状态报告
      3. 模板接口
      3.1 [添加模板](https://sms.yunpian.com/v2/tpl/add.json)
      3.2 [取模板](https://sms.yunpian.com/v2/tpl/get.json)
      3.3 [修改模板](https://sms.yunpian.com/v2/tpl/update.json)
      3.4 [删除模板](https://sms.yunpian.com/v2/tpl/del.json)
      4. 签名接口
      4.1 [添加签名](https://sms.yunpian.com/v2/sign/add.json)
      4.2 [获取签名](https://sms.yunpian.com/v2/sign/get.json)
      4.3 [修改签名](https://sms.yunpian.com/v2/sign/update.json)
      5. [查短信发送记录](https://sms.yunpian.com/v2/sms/get_record.json)
      6. [查屏蔽词](https://sms.yunpian.com/v2/sms/get_black_word.json)
* 短信`充值/扣费`服务，提供[**商家**]入驻短信平台,

该服务依赖于[igrpc](../igrpc)模块，[微信公众号第三方平台](../official_accounts)和[common](../common)组成

## 库表设计

[`短信服务库表`](./table.md)
[`全局配置库`](./table.md)

## 说明

`希望与大家一起成长，有任何该服务运行或者代码问题，可以及时找我沟通，喜欢开源，热爱开源, 欢迎多交流`
`联系方式：cdh_cjx@163.com`