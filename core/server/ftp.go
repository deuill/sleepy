// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/thoughtmonster/sleepy/core/user"
)

type ftpSession struct {
	conn net.Conn
	data net.Listener
	user *user.User
}

func ServeFTP(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		session := &ftpSession{conn, nil, nil}
		go session.serve()
	}
}

func (s *ftpSession) serve() {
	buf := bufio.NewReader(s.conn)
	s.respond("220 Connection established")

	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			continue
		}

		params := strings.Fields(line)
		if len(params) == 0 {
			continue
		}

		switch strings.ToUpper(params[0]) {
		case "TYPE", "MODE", "STRU":
			s.respond("200 Command OK")
		case "PWD":
			s.respond("257 \"/\" is the current directory")
		case "PORT":
			s.respond("421 Cannot use active mode, use passive mode instead")
		case "PASV":
			s.data, err = net.Listen("tcp", ":0")
			if err != nil {
				s.respond("421 Could not start in passive mode, creating socket failed")
				goto quit
			}

			_, port, _ := net.SplitHostPort(s.data.Addr().String())
			t, _ := strconv.ParseInt(port, 10, 64)
			p := strconv.FormatInt(t/256, 10) + "," + strconv.FormatInt(t%256, 10)
			s.respond("227 Entering Passive Mode (127,0,0,1," + p + ")")
		case "USER":
			if len(params) < 2 {
				s.respond("501 USER expects an SHA1 authkey, none given")
				break
			}

			u, _ := user.Auth(params[1])
			if u == nil {
				s.respond("530 Login failed")
				break
			}

			s.user = u
			s.respond("230 Login successful")
		case "STOR":
			if s.user == nil {
				s.respond("532 You need to login to access this command")
				break
			}

			if len(params) < 2 {
				s.respond("501 STOR expects a name for the file, none given")
				break
			}

			s.respond("150 File transfer starting")
			go s.storeFile(params[1])
		case "QUIT":
			s.respond("221 Closing connection")
			goto quit
		default:
			s.respond("502 Command not implemented")
		}
	}

quit:
	if s.data != nil {
		s.data.Close()
	}

	s.conn.Close()
}

func (s *ftpSession) respond(msg string) {
	fmt.Fprintln(s.conn, msg)
}

func (s *ftpSession) storeFile(name string) {
	defer s.data.Close()

	conn, err := s.data.Accept()
	if err != nil {
		s.respond("451 Could not establish connection to server")
		return
	}

	defer conn.Close()
	path := os.TempDir() + "/sleepy/" + strconv.FormatInt(int64(s.user.Id), 10)

	err = os.MkdirAll(path, 0755)
	if err != nil {
		s.respond("451 Could not create temporary directory")
		return
	}

	file, err := os.Create(path + "/" + name)
	if err != nil {
		s.respond("451 Could not create remote file")
		return
	}

	defer file.Close()

	io.Copy(file, conn)
	s.respond("226 File transfer successful")
}
