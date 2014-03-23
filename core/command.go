// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/thoughtmonster/sleepy/core/user"
	"github.com/spf13/cobra"
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

func init() {
	rootCmd.PersistentFlags().StringVarP(&flags.config, "config", "c", "/etc/sleepy/sleepy.conf", "Main configuration file to read from")
	rootCmd.PersistentFlags().Int64VarP(&flags.connections, "max-connections", "m", 0, "Max concurrent connections to server")
	userCmd.Flags().BoolP("add", "a", true, "Add user to server")
	userCmd.Flags().IntP("remove", "r", 0, "Remove user from server")
	userCmd.Flags().BoolP("list", "l", true, "List users on server")

	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(versionCmd)

}
