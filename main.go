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
func spawnAsChild(args []string) {
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr =
		&syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC,
		}

	must(cmd.Run())
}

func parseArgs(cliCmd *cobra.Command, args []string) {
	spawnAsChild(args)

}
func runAsChild() {
	syscall.Sethostname([]byte("vessel"))

	command := os.Args[3]
	cmdArgs := os.Args[4:]

	cmd := exec.Command(command, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(cmd.Run())
}
func init() {

	// flags will go here
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		runAsChild()
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: {%v}", err)
		os.Exit(1)
	}
}
