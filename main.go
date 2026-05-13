//go:build linux

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"os/exec"
	"os/signal"

	"syscall"

	"github.com/peterbull/vesselrt/hub"
	"github.com/peterbull/vesselrt/network"
	"github.com/peterbull/vesselrt/state"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
)

var rootCmd = &cobra.Command{
	Use:   "vesselrt",
	Short: "A container runtime",
}

var runCmd = &cobra.Command{
	Use:   "run [image] [command]",
	Short: "Run a command in a container",
	Args:  cobra.MinimumNArgs(2),
	Run:   runContainer,
}

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running containers",
	Run:   listContainers,
}

var killCmd = &cobra.Command{
	Use:   "kill [id]",
	Short: "Kill a running container",
	Args:  cobra.ExactArgs(1),
	Run:   killContainer,
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(killCmd)
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
	containerId, err := createID(12)
	if err != nil {
		log.Fatalf("error generating container id: %v", err)
	}

	cmd := exec.Command("/proc/self/exe", append([]string{"child", containerId}, args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("error starting container: %v", err)
	}
	pid := cmd.Process.Pid
	nsFile, err := os.Open(fmt.Sprintf("/proc/%d/ns/net", pid))
	defer nsFile.Close()
	if err != nil {
		log.Fatalf("error getting proc namespace: %v", err)
	}
	local, peer := network.CreateVethPairs()

	nsFd := nsFile.Fd()

	netlink.LinkSetNsFd(peer, int(nsFd))
	defer netlink.LinkDel(local)

	containerState := state.ContainerState{
		ID:      containerId,
		PID:     cmd.Process.Pid,
		Status:  "Running",
		Image:   "alpine",
		Rootfs:  "images/alpine/rootfs",
		Created: time.Now(),
	}

	if err := state.WriteState(containerState); err != nil {
		log.Printf("error writing state: %v", err)
	}
	defer state.DeleteState(containerId)
	if err := cmd.Wait(); err != nil {
		log.Printf("err, exiting %v", err)
	}

	syscall.Unmount("/home/peterbull.guest/rootfs", syscall.MNT_DETACH)
	os.Remove(fmt.Sprintf("/sys/fs/cgroup/%s", containerId))
}

func runContainer(cliCmd *cobra.Command, args []string) {
	spawnAsChild(args)
}

func listContainers(cliCmd *cobra.Command, args []string) {
	baseDir := "/run/vesselrt"
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		fmt.Printf("error listing containers: %v", err)
	}
	for _, entry := range entries {

		file, err := os.ReadFile(fmt.Sprintf("%s/%s/state.json", baseDir, entry.Name()))
		if err != nil {
			fmt.Printf("error reading state from file: %v", err)
		}
		if entry.IsDir() {
			fmt.Printf("state: %s", string(file))
		}
	}
}

func killContainer(cliCmd *cobra.Command, args []string) {
	containerId := args[0]
	containerState, err := state.ReadState(containerId)
	if err != nil {
		fmt.Printf("error killing container: %v", err)
		return
	}
	process, err := os.FindProcess(containerState.PID)
	if err != nil {
		fmt.Printf("error finding process: %v", err)
		return
	}
	err = process.Signal(os.Interrupt)
	if err != nil {
		fmt.Printf("error killing process: %v", err)
	}
	fmt.Printf("sigint sent to process: %d", containerState.PID)
}

func createID(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type ContainerStatus string

const (
	Running ContainerStatus = "Running"
	Stopped ContainerStatus = "Stopped"
)

func runAsChild() error {
	syscall.Sethostname([]byte("vessel"))
	containerID := os.Args[2]
	command := os.Args[3]
	cmdArgs := os.Args[4:]

	cmd := exec.Command(command, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	imgName := "alpine"
	imgPath := fmt.Sprintf("images/%s", imgName)
	_, err := os.Stat(imgPath)
	if os.IsNotExist(err) {
		hub.PullImage(imgName)
	}

	rootfs := fmt.Sprintf("images/%s/rootfs", imgName)
	const oldRoot = ".old_root"
	intPID := os.Getpid()
	pid := []byte(strconv.Itoa(intPID))
	must(os.MkdirAll(fmt.Sprintf("/sys/fs/cgroup/%s", containerID), 0755))

	// cgroups v2 requires explicit enable
	must(os.WriteFile("/sys/fs/cgroup/cgroup.subtree_control", []byte("+cpu +memory +pids"), 0700))

	must(os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/cgroup.procs", containerID), pid, 0700))
	must(os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/pids.max", containerID), []byte("20"), 0700))

	// 100 mb
	must(os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/memory.max", containerID), []byte("104857600"), 0700))

	// half of a core
	must(os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/cpu.max", containerID), []byte("50000 100000"), 0700))

	// have to make private before bind mount
	must(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""), "make / private")
	must(syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""), "bind mount")
	if err := os.MkdirAll(rootfs+"/"+oldRoot, 0755); err != nil {
		if !os.IsExist(err) {
			must(err)

		}
	}

	// pivot root
	must(syscall.Chdir(rootfs), "chdir to rootfs")
	must(syscall.PivotRoot(".", oldRoot), "pivot_root")
	must(syscall.Chdir("/"), "chdir to new /")
	must(syscall.Unmount(oldRoot, syscall.MNT_DETACH), "unmount old root")
	must(syscall.Rmdir(oldRoot), "rmdir old root")
	must(syscall.Mount("proc", "/proc", "proc", 0, ""), "mount proc")

	if err := cmd.Run(); err != nil {
		fmt.Printf("error, exiting... %v\n", err)
	}

	return nil
}

func main() {
	_, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == "child" {
		if err := runAsChild(); err != nil {
			log.Fatalf("big error: %v", err)
		}
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: {%v}", err)
		os.Exit(1)
	}
}
