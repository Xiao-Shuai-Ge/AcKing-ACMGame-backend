package email

import (
	"crypto/tls"
	"fmt"
	"tgwp/global"
	"tgwp/log/zlog"

	"gopkg.in/gomail.v2"
)

func Send(to []string, subject string, message string) error {
	host := global.Config.Email.Host
	port := global.Config.Email.Port
	userName := global.Config.Email.UserName
	password := global.Config.Email.Password

	m := gomail.NewMessage()
	m.SetHeader("From", userName)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", message)

	d := gomail.NewDialer(
		host,
		port,
		userName,
		password,
	)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		zlog.Errorf("邮件发送失败：%v", err)
		return err
	}
	return nil
}

func SendCode(to string, code int64) error {
	message := `
	<p style="text-indent:2em;">你的邮箱验证码为: %06d </p> 
	<p style="text-indent:2em;">此验证码的有效期为5分钟，请尽快使用。</p>
	`
	return Send([]string{to}, "[ACM-GAME] [邮箱验证码]", fmt.Sprintf(message, code))
}
