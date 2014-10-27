// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package main is the constitutes the starting poing for Sleepy. It
// contains code for setting run-time options and some setting up the
// operation of the server.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/deuill/sleepy/core/config"
	"github.com/deuill/sleepy/core/server"
	"github.com/deuill/sleepy/core/user"

	// Internal modules
	_ "github.com/deuill/sleepy/modules/auth"
	_ "github.com/deuill/sleepy/modules/database"
	_ "github.com/deuill/sleepy/modules/email"
	_ "github.com/deuill/sleepy/modules/file"
	_ "github.com/deuill/sleepy/modules/image"
	_ "github.com/deuill/sleepy/modules/template"
	_ "github.com/deuill/sleepy/modules/user"
)

var flags struct {
	config      string
	connections int64
}

var rootCmd = &cobra.Command{
	Use:   "sleepyd",
	Short: "Sleepy - the lightweight web application server",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Provides methods for adding, removing and listing users",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := setup(flags.config, false); err != nil {
			fmt.Printf("Unable to initialize environment: %s\n", err)
			os.Exit(1)
		}

		if cmd.Flags().Lookup("add").Changed {
			u, err := user.Save()
			if err != nil {
				fmt.Printf("Unable to add user: %s\n", err)
				os.Exit(1)
			}

			fmt.Printf("User with id '%d', authkey '%s' added successfully.\n", u.Id, u.Authkey)
			os.Exit(0)
		}

		if cmd.Flags().Lookup("remove").Changed {
			id, _ := strconv.Atoi(cmd.Flags().Lookup("remove").Value.String())
			_, err := user.Remove(id)
			if err != nil {
				fmt.Printf("Unable to remove user: %s\n", err)
				os.Exit(1)
			}

			fmt.Printf("User with id '%d' removed successfully.\n", id)
			os.Exit(0)
		}

		if cmd.Flags().Lookup("list").Changed {
			l, err := user.List()
			if err != nil {
				fmt.Printf("Unable to list users: %s\n", err)
				os.Exit(1)
			}

			fmt.Println("#\tID\tAuthkey")
			for i, u := range l {
				fmt.Printf("%d\t%d\t%s\n", (i + 1), u.Id, u.Authkey)
			}

			os.Exit(0)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the program name and version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Sleepy version 0.5.0")
	},
}

func setup(conf string, remote bool) (net.Listener, error) {
	var err error

	// Load main configuration file.
	c, err := config.Load(conf)
	if err != nil {
		fmt.Println("Unable to read file '" + conf + "'.")
		fmt.Println("Please specify a valid configuration file using the '--config' command-line option.")
		os.Exit(1)
	}

	// Set up the system directories as needed.
	tmpdir := c.S("directories", "tmp")
	if _, err = os.Stat(tmpdir); err != nil {
		if err = os.Mkdir(tmpdir, 0755); err != nil {
			return nil, err
		}
	}

	// Connect to system database.
	datadir := c.S("directories", "data")
	err = user.Setup(datadir, c.S("sqlite", "filename"))
	if err != nil {
		return nil, err
	}

	// Write our PID to a file.
	ioutil.WriteFile(tmpdir+"/sleepy.pid", []byte(strconv.Itoa(os.Getpid())), 0644)

	// Handle SIGINT and SIGTERM signals.
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		<-sigchan

		log.Println("Shutting down Sleepy...")
		os.Exit(0)
	}()

	// Initialize networking parts if not running a local operation.
	if remote == true {
		// Setup our internal modules.
		err = server.Setup(c)
		if err != nil {
			return nil, err
		}

		// Set up TCP socket.
		ln, err := net.Listen("tcp", c.S("sleepy", "address")+":"+c.S("sleepy", "port"))
		if err != nil {
			return nil, err
		}

		// Register the RPC method receiver for external method calls.
		rpc.RegisterName("Sleepy", &server.Server{})

		// Start embedded HTTP server.
		go func() {
			http.Handle("/", server.HTTPHandler(datadir+"/serve/"))
			http.ListenAndServe(":"+c.S("http", "port"), nil)
		}()

		// Start embedded FTP server.
		go func() {
			server.ServeFTP(c.S("ftp", "address") + ":" + c.S("ftp", "port"))
		}()

		// Get limit for maximum concurrent connections to server.
		if flags.connections == 0 {
			flags.connections = c.I("sleepy", "max-connections")
		}

		return ln, nil
	}

	return nil, nil
}

func run() {
	// Setup core environment.
	ln, err := setup(flags.config, true)
	if err != nil {
		fmt.Printf("Unable to initialize environment: %s\n", err)
		os.Exit(1)
	}

	defer ln.Close()

	// Start serving connections.
	log.Println("Staring Sleepy...")
	queue := make(chan bool, flags.connections)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to handle connection: %s", err)
			continue
		}

		queue <- true
		go func(conn net.Conn) {
			jsonrpc.ServeConn(conn)
			<-queue
		}(conn)
	}
}

func main() {
	rootCmd.PersistentFlags().StringVarP(&flags.config, "config", "c", "/etc/sleepy/sleepy.conf", "Main configuration file to read from")
	rootCmd.PersistentFlags().Int64VarP(&flags.connections, "max-connections", "m", 0, "Max concurrent connections to server")
	userCmd.Flags().BoolP("add", "a", true, "Add user to server")
	userCmd.Flags().IntP("remove", "r", 0, "Remove user from server")
	userCmd.Flags().BoolP("list", "l", true, "List users on server")

	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.Execute()
}
