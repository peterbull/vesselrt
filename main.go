//go:build linux

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"os/exec"
	"os/signal"

	"syscall"

	"github.com/peterbull/vesselrt/hub"
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

func must(err error, msg ...string) {
	if err != nil {
		if len(msg) > 0 {
			log.Panicf("%s: %v", msg[0], err)
		} else {
			log.Panic(err)
		}
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

	syscall.Unmount("/home/peterbull.guest/rootfs", syscall.MNT_DETACH)
	os.Remove("/sys/fs/cgroup/mycontainer")
}

func parseArgs(cliCmd *cobra.Command, args []string) {
	spawnAsChild(args)
}
func runAsChild(ctx context.Context) {

	syscall.Sethostname([]byte("vessel"))

	command := os.Args[3]
	cmdArgs := os.Args[4:]

	cmd := exec.Command(command, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	hub.PullImage("alpine")
	const rootfs = "/home/peterbull.guest/rootfs"
	const oldRoot = ".old_root"
	pid := []byte(strconv.Itoa(os.Getpid()))
	must(os.MkdirAll("/sys/fs/cgroup/mycontainer", 0755))

	// cgroups v2 requires explicit enable
	must(os.WriteFile("/sys/fs/cgroup/cgroup.subtree_control", []byte("+cpu +memory +pids"), 0700))

	must(os.WriteFile("/sys/fs/cgroup/mycontainer/cgroup.procs", pid, 0700))
	must(os.WriteFile("/sys/fs/cgroup/mycontainer/pids.max", []byte("20"), 0700))

	// 100 mb
	must(os.WriteFile("/sys/fs/cgroup/mycontainer/memory.max", []byte("104857600"), 0700))
	// half of a core
	must(os.WriteFile("/sys/fs/cgroup/mycontainer/cpu.max", []byte("50000 100000"), 0700))

	// have to make private before bind mount
	must(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""), "make / private")
	must(syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""), "bind mount")
	if err := os.MkdirAll(rootfs+"/"+oldRoot, 0755); err != nil {
		if !os.IsExist(err) {
			must(err)
		}
	}

	must(syscall.Chdir(rootfs), "chdir to rootfs")
	must(syscall.PivotRoot(".", oldRoot), "pivot_root")
	must(syscall.Chdir("/"), "chdir to new /")
	must(syscall.Unmount(oldRoot, syscall.MNT_DETACH), "unmount old root")
	must(syscall.Rmdir(oldRoot), "rmdir old root")
	must(syscall.Mount("proc", "/proc", "proc", 0, ""), "mount proc")

	must(cmd.Run())
}

func init() {

	// flags will go here
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == "child" {
		runAsChild(ctx)
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: {%v}", err)
		os.Exit(1)
	}
}
