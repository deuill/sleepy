// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package email provides methods for sending email messages, with
// support for multiple To, Cc, Bcc fields, attachments etc.
package email

import (
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/deuill/sleepy/core/config"
	"github.com/deuill/sleepy/core/server"
)

type Email struct {
	// Contains private or unexported fields.
	host     string
	port     string
	username string
	password string
}

type Request struct {
	Subject string
	Message struct {
		Content string
		Type    string
	}
	From struct {
		Address string
		Name    string
	}
	To     []string
	Cc     []string
	Bcc    []string
	Attach []struct {
		Filename string
		Type     string
		Data     string
	}
}

func (e *Email) Send(p Request) (bool, error) {
	var body, boundary, ctype string

	var auth smtp.Auth
	if e.username != "" && e.password != "" {
		auth = smtp.PlainAuth(
			"",
			e.username,
			e.password,
			e.host,
		)
	}

	from := mail.Address{p.From.Name, p.From.Address}
	for i, addr := range p.To {
		t := &mail.Address{"", addr}
		p.To[i] = t.String()
	}

	body += "From: " + from.String() + "\r\n"
	body += "To: " + strings.Join(p.To, ", ") + "\r\n"
	body += "Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"
	body += "Subject: =?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(p.Subject)) + "?=\r\n"
	body += "MIME-Version: 1.0\r\n"

	switch p.Message.Type {
	case "html":
		ctype = "text/html"
	case "text":
		ctype = "text/plain"
	default:
		ctype = "text/plain"
	}

	if len(p.Attach) > 0 {
		boundary = "______(-_-,)_zZzZZ__"
		body += "Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n"
		body += "\r\n--" + boundary + "\r\n"
		body += "Content-Type: " + ctype + "; charset=\"UTF-8\"\r\n"
		body += "Content-Transfer-Encoding: base64\r\n"
		body += "\r\n" + base64.StdEncoding.EncodeToString([]byte(p.Message.Content))
		body += "\r\n--" + boundary

		for _, a := range p.Attach {
			body += "\r\n" + "Content-Type: " + a.Type + "; name=\"" + a.Filename + "\"\r\n"
			body += "Content-Transfer-Encoding: base64\r\n"
			body += "Content-Disposition: attachment; filename=\"" + a.Filename + "\"\r\n"
			body += "\r\n" + a.Data
			body += "\r\n--" + boundary
		}

		body += "--"
	} else {
		body += "Content-Type: " + ctype + "; charset=\"UTF-8\"\r\n"
		body += "Content-Transfer-Encoding: base64\r\n"
		body += "\r\n" + base64.StdEncoding.EncodeToString([]byte(p.Message.Content))
	}

	err := sendMail(e.host+":"+e.port, auth, from.Address, p.To, []byte(body))
	if err != nil {
		return false, err
	}

	return true, nil
}

func sendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		host, _, _ := net.SplitHostPort(addr)
		conf := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: true,
		}
		if err = c.StartTLS(conf); err != nil {
			return err
		}
	}
	if a != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(a); err != nil {
				return err
			}
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

func (e *Email) Setup(config *config.Config) error {
	e.host, _ = config.String("email", "host")
	e.port, _ = config.String("email", "port")
	e.username, _ = config.String("auth", "username")
	e.password, _ = config.String("auth", "password")

	return nil
}

func init() {
	server.Register(&Email{})
}
