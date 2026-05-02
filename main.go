//go:build linux

package main

import (
	"fmt"
	"log"
	"os"

	"os/exec"

	"syscall"

	"github.com/spf13/cobra"
)

var (
	runCmd  string
	image   string
	command string
)

var rootCmd = &cobra.Command{
	Use:   "run [image] [command]",
	Short: "run images",
	Args:  cobra.MinimumNArgs(3),
	Run:   parseArgs,
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func parseArgs(cliCmd *cobra.Command, args []string) {
	runCmd = args[0]
	image = args[1]
	command = args[2]
	cmdArgs := args[3:]

	_ = cmdArgs
	cmd := exec.Command(command, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr =
		&syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS,
		}
	must(cmd.Run())

	for _, arg := range args {
		fmt.Printf("arg: %v\n", arg)
	}
	_ = cliCmd
}
func init() {
	// flags will go here
}
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: {%v}", err)
		os.Exit(1)
	}
	fmt.Println("hello vessel")
}
