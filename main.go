package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfg config
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfg.Address, "address", "a", "", "fake address of server to test")
	rootCmd.PersistentFlags().StringVarP(&cfg.Proto, "proto", "p", "", "fake protocol of server to test")
	rootCmd.PersistentFlags().StringVar(&cfg.UserAgent, "user-agent", "svck", "user agent string to be sent in requests")
	rootCmd.PersistentFlags().StringVarP(&cfg.Output, "output", "o", "default", "output format, can be 'default' or 'json'")

	rootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().IntVarP(&cfg.Workers, "workers", "w", 3, "number of workers to execute requests concurrently")
	runCmd.PersistentFlags().IntVarP(&cfg.Timeout, "timeout", "t", 10, "number of seconds until a single request runs in a timeout")
	runCmd.PersistentFlags().BoolVar(&cfg.NoProgress, "no-progress", false, "do not display progess bar")

	rootCmd.AddCommand(curlCmd)
	curlCmd.PersistentFlags().BoolVar(&cfg.NoBashComments, "no-bash-comments", false, "do not print bash comments")
}

var rootCmd = &cobra.Command{
	Use:   "svck",
	Short: "Runs http requests and checks the responses",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run checks",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg.ServiceFiles = args
		err := cfg.ReadServiceFiles()
		if err != nil {
			panic(err)
		}

		checks, err := NewChecks(cfg.services, cfg.Address, cfg.Proto, cfg.UserAgent, cfg.Timeout)
		if err != nil {
			panic(err)
		}
		p := NewPool(checks, cfg.Workers)
		p.Run(!cfg.NoProgress)

		success := 0
		fail := 0
		if cfg.Output == "json" {
			out, err := json.MarshalIndent(checks, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Println(string(out))
		} else {
			for _, check := range checks {
				if check.success {
					success++
				} else {
					fail++
					fmt.Printf("Failed check: %s\n\tURL: %s\n\tREQUEST_HEADERS: %s\n\tRESPONSE_HEADERS: %s\n\tREPRODUCE: %s\n\tREASON: %s\n\n", check.name(), check.request.URL.String(), check.requestHeaders(""), check.responseHeaders(""), check.asCurl(), check.reason)
				}
			}

			fmt.Printf("Summary:\n\tFailed: %d\n\tSuccessful: %d\n", fail, success)
		}
	},
}

var curlCmd = &cobra.Command{
	Use:   "curl",
	Short: "Print all checks as curl commands",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg.ServiceFiles = args
		err := cfg.ReadServiceFiles()
		if err != nil {
			fmt.Println(err)
		}

		checks, err := NewChecks(cfg.services, cfg.Address, cfg.Proto, cfg.UserAgent, cfg.Timeout)
		if err != nil {
			fmt.Println(err)
		}

		if !cfg.NoBashComments {
			fmt.Printf("#!/bin/bash\n")
		}
		for _, check := range checks {
			if !cfg.NoBashComments {
				fmt.Printf("\n# %s\n", check.name())
			}
			fmt.Sprintf("%s\n", check.asCurl())
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
