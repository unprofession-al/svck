package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/unprofession-al/workerpool"
)

type App struct {
	cfg config

	// entry point
	Execute func() error
}

func NewApp() *App {
	a := &App{}

	// root
	rootCmd := &cobra.Command{
		Use:   "svck",
		Short: "Runs http requests and checks the responses",
	}
	rootCmd.PersistentFlags().StringVarP(&a.cfg.Address, "address", "a", "", "fake address of server to test")
	rootCmd.PersistentFlags().StringVarP(&a.cfg.Proto, "proto", "p", "", "fake protocol of server to test")
	rootCmd.PersistentFlags().StringVar(&a.cfg.UserAgent, "user-agent", "svck", "user agent string to be sent in requests")
	rootCmd.PersistentFlags().StringVarP(&a.cfg.Output, "output", "o", "default", "output format, can be 'default' or 'json'")
	a.Execute = rootCmd.Execute

	// run
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run checks",
		Args:  cobra.MinimumNArgs(1),
		Run:   a.runCmd,
	}
	runCmd.PersistentFlags().IntVarP(&a.cfg.Workers, "workers", "w", 3, "number of workers to execute requests concurrently")
	runCmd.PersistentFlags().IntVarP(&a.cfg.Timeout, "timeout", "t", 10, "number of seconds until a single request runs in a timeout")
	runCmd.PersistentFlags().BoolVar(&a.cfg.NoProgress, "no-progress", false, "do not display progess bar")
	rootCmd.AddCommand(runCmd)

	// curl
	var curlCmd = &cobra.Command{
		Use:   "curl",
		Short: "Print all checks as curl commands",
		Args:  cobra.MinimumNArgs(1),
		Run:   a.curlCmd,
	}
	curlCmd.PersistentFlags().BoolVar(&a.cfg.NoBashComments, "no-bash-comments", false, "do not print bash comments")
	rootCmd.AddCommand(curlCmd)

	// version
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run:   a.versionCmd,
	}
	rootCmd.AddCommand(versionCmd)

	return a
}

func (a *App) runCmd(cmd *cobra.Command, args []string) {
	a.cfg.ServiceFiles = args
	err := a.cfg.ReadServiceFiles()
	if err != nil {
		panic(err)
	}

	checks, err := NewChecks(a.cfg.services, a.cfg.Address, a.cfg.Proto, a.cfg.UserAgent, a.cfg.Timeout)
	if err != nil {
		panic(err)
	}

	p := workerpool.New(a.cfg.Workers)
	for _, check := range checks {
		p.Add(check.run)
	}
	p.Run(!a.cfg.NoProgress)

	success := 0
	fail := 0
	if a.cfg.Output == "json" {
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
}

func (a *App) curlCmd(cmd *cobra.Command, args []string) {
	a.cfg.ServiceFiles = args
	err := a.cfg.ReadServiceFiles()
	if err != nil {
		fmt.Println(err)
	}

	checks, err := NewChecks(a.cfg.services, a.cfg.Address, a.cfg.Proto, a.cfg.UserAgent, a.cfg.Timeout)
	if err != nil {
		fmt.Println(err)
	}

	if !a.cfg.NoBashComments {
		fmt.Printf("#!/bin/bash\n")
	}
	for _, check := range checks {
		if !a.cfg.NoBashComments {
			fmt.Printf("\n# %s\n", check.name())
		}
		fmt.Printf("%s\n", check.asCurl())
	}
}

func (a *App) versionCmd(cmd *cobra.Command, args []string) {
	fmt.Println(versionInfo())
}
