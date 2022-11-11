// adb操作相关

package adb

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var adbClient *Client

type Client struct {
	adbPath string
}

type Forward struct {
	Serial     string
	LocalPort  int64
	DevicePort int64
}

func GetClient() *Client {
	if adbClient == nil {
		adbClient = newAdbClient()
	}
	return adbClient
}

func newAdbClient() *Client {
	a := &Client{}
	a.Init()
	return a
}

// Init adb path
func (a *Client) Init() {
	if runtime.GOOS == "windows" {
		os.Setenv("PATH", os.Getenv("PATH")+";"+"C:\\Program Files\\Nox\\bin")
		os.Setenv("PATH", os.Getenv("PATH")+";"+"D:\\Program Files\\Nox\\bin")
		os.Setenv("PATH", os.Getenv("PATH")+";"+"C:\\Program Files (x86)\\Nox\\bin")
		os.Setenv("PATH", os.Getenv("PATH")+";"+"D:\\Program Files (x86)\\Nox\\bin")
	}
	pwd, _ := os.Getwd()
	if runtime.GOOS == "windows" {
		os.Setenv("PATH", os.Getenv("PATH")+";"+pwd+"\\bin")
	} else {
		os.Setenv("PATH", os.Getenv("PATH")+":"+pwd+"/bin")
	}

	adbPath, err := exec.LookPath("adb")
	if adbPath == "" || err != nil {
		fmt.Println("adb not found, download it...")
		if err := downloadAdb(); err != nil {
			fmt.Println("adb download failed")
			os.Exit(1)
		}
		fmt.Println("adb download success")
		adbPath, err = exec.LookPath("adb")
		fmt.Println(adbPath)
	}
	if err != nil {
		fmt.Println("Error adb not found... exit")
		os.Exit(1)
	}
	fmt.Println(fmt.Sprintf("Found adb: %s", adbPath))
	a.adbPath = adbPath
}

// GetDevices adb devices
func (a *Client) GetDevices() ([]string, error) {
	var devices []string
	cmd := exec.Command(a.adbPath, "devices")
	result, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}
	lines := strings.Split(string(result), "\n")
	for _, line := range lines {
		tabs := strings.Split(line, "\t")
		if len(tabs) > 1 && !strings.Contains(tabs[1], "offline") {
			devices = append(devices, tabs[0])
		}
	}
	return devices, nil
}

// Forward adb forward
func (a *Client) Forward(f *Forward) (bool, error) {
	cmd := exec.Command(a.adbPath, "-s", f.Serial, "forward", "--no-rebind", "tcp:"+fmt.Sprintf("%d", f.LocalPort), "tcp:"+fmt.Sprintf("%d", f.DevicePort))
	bs, err := cmd.Output()
	if err != nil {
		return false, err
	}
	if strings.Contains(string(bs), "error") {
		return false, fmt.Errorf(string(bs))
	}
	return true, nil
}

// ForwardList adb forward --list
func (a *Client) ForwardList() ([]*Forward, error) {
	cmd := exec.Command(a.adbPath, "forward", "--list")
	bs, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var fs []*Forward
	lines := strings.Split(string(bs), "\n")
	for _, line := range lines {
		line = strings.Trim(line, "")
		tabs := strings.Split(line, "\t")
		if len(tabs) <= 1 {
			tabs = strings.Split(line, " ")
		}
		if len(tabs) > 1 && !strings.Contains(tabs[1], "offline") {
			localPort, err := strconv.ParseInt(tabs[1][4:], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing the local port is abnormal. Procedure")
			}
			devicePort, err := strconv.ParseInt(tabs[2][4:], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing the device port is abnormal. Procedure")
			}
			f := &Forward{
				Serial:     tabs[0],
				LocalPort:  localPort,
				DevicePort: devicePort,
			}
			fs = append(fs, f)
		}
	}
	return fs, nil
}

// Connect adb connect
func (a *Client) Connect(ip string, port int64) (string, error) {
	cmd := exec.Command(a.adbPath, "connect", ip+":"+fmt.Sprintf("%d", port))
	bs, err := cmd.Output()
	if strings.Contains(string(bs), "connected") {
		return ip + ":" + fmt.Sprintf("%d", port), nil
	}
	cmd = exec.Command(a.adbPath, "disconnect", ip+":"+fmt.Sprintf("%d", port))
	cmd.Output()
	if err != nil {
		return "", err
	}
	return "", nil
}

// Restart adb kill-server then adb start-server
func (a *Client) Restart() {
	cmd := exec.Command(a.adbPath, "kill-server")
	cmd.Output()
	time.Sleep(time.Second)
	cmd = exec.Command(a.adbPath, "start-server")
	cmd.Output()
}

// TCPIP adb tcpip
func (a *Client) TCPIP(serial, port string) error {
	cmd := exec.Command(a.adbPath, "-s", serial, "tcpip", port)
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

// Shell adb shell
func (a *Client) Shell(serial, command string) (string, error) {
	cmd := exec.Command(a.adbPath, "-s", serial, "shell", command)

	bs, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.Trim(bs, "\r\n ")), nil
}

// Install adb install
func (a *Client) Install(serial, apk string) (string, error) {
	cmd := exec.Command(a.adbPath, "-s", serial, "install", "-r", apk)

	bs, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if strings.Contains(string(bs), "error") {
		return "", nil
	}
	return string(bytes.Trim(bs, "\r\n ")), nil
}

// Uninstall adb uninstall
func (a *Client) Uninstall(serial, apkID string) (string, error) {
	cmd := exec.Command(a.adbPath, "-s", serial, "uninstall", apkID)

	bs, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if strings.Contains(string(bs), "error") {
		return "", nil
	}
	return string(bytes.Trim(bs, "\r\n ")), nil
}

// GetPids adb shell "ps | grep process"  (app_process)
func (a *Client) GetPids(serial string, process string) ([]string, error) {
	cmd := exec.Command(a.adbPath, "-s", serial, "shell", "ps | grep "+process)

	bs, err := cmd.Output()
	if err != nil {
		if !strings.Contains(err.Error(), "exit") {
			return nil, err
		}
	}
	result := string(bs)
	// try ps -A
	if result == "" {
		cmd := exec.Command(a.adbPath, "-s", serial, "shell", "ps -A | grep "+process)

		bs, err := cmd.Output()
		if err != nil {
			if !strings.Contains(err.Error(), "exit") {
				return nil, err
			}
		}
		result = string(bs)
	}

	pids := []string{}
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		tabs := strings.Split(line, " ")
		for i, tab := range tabs {
			if i > 0 && tab != "" {
				pids = append(pids, tab)
				break
			}
		}
	}
	return pids, nil
}

// IsFileExist check is file exist in device
func (a *Client) IsFileExist(serial, path string) bool {
	cmd := exec.Command(a.adbPath, "-s", serial, "shell", "ls "+path)

	bs, err := cmd.Output()
	if err != nil {
		return false
	}
	result := string(bs)

	if strings.Contains(result, "No such file") {
		return false
	}
	return true
}

// GetDeviceABI get device abi, arm64-v8a, armeabi-v7a or x86
func (a *Client) GetDeviceABI(serial string) (string, error) {
	return a.Shell(serial, "getprop ro.product.cpu.abi")
}

// GetAppProcess for robotmon starting service
func (a *Client) GetAppProcess(serial string) (bool, bool, bool) {
	isExist := a.IsFileExist(serial, "/system/bin/app_process")
	isExist32 := a.IsFileExist(serial, "/system/bin/app_process32")
	isExist64 := a.IsFileExist(serial, "/system/bin/app_process64")
	return isExist, isExist32, isExist64
}

// GetApkPath get apk installed path (com.r2studio.robotmon)
func (a *Client) GetApkPath(serial, packageName string) (string, error) {
	cmd := exec.Command(a.adbPath, "-s", serial, "shell", "pm path "+packageName)

	bs, err := cmd.Output()
	if err != nil {
		if !strings.Contains(err.Error(), "exit status 1") {
			return "", err
		}
	}
	result := string(bs)
	result = strings.Trim(result, "\r\n")
	paths := strings.Split(string(result), ":")
	if len(paths) < 2 {
		return "", fmt.Errorf("Can not find robotmon package, please install it first")
	}
	return paths[1], nil
}

func (a *Client) getNohubPath(serial string) string {
	nohup := "" // some device not exist nohup
	if a.IsFileExist(serial, "/system/bin/nohup") || a.IsFileExist(serial, "/system/xbin/nohup") {
		nohup = "nohup"
	} else if a.IsFileExist(serial, "/system/bin/daemonize") || a.IsFileExist(serial, "/system/xbin/daemonize") {
		nohup = "daemonize"
	}
	return nohup
}

// GetApkAbi get apk abi
func (a *Client) GetApkAbi(serial, packageName string) string {
	cmd := exec.Command(a.adbPath, "-s", serial, "shell", "pm dump "+packageName)

	bs, err := cmd.Output()
	if err != nil {
		return ""
	}
	result := string(bs)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "primaryCpuAbi") {
			continue
		}
		if strings.Contains(line, "x86") {
			return "x86"
		} else if strings.Contains(line, "arm64-v8a") {
			return "arm64-v8a"
		}
	}
	return "armeabi-v7a"
}

// GetRobotmonStartCommand getRobotmonStartCommand
func (a *Client) GetRobotmonStartCommand(serial string) (string, []string, error) {
	nohup := a.getNohubPath(serial)
	apk, err := a.GetApkPath(serial, "com.r2studio.robotmon")
	if err != nil {
		return "", nil, err
	}
	apkDir := path.Dir(apk)

	// abi, err := a.GetDeviceABI(serial)
	// if err != nil {
	// 	return "", nil, err
	// }
	abi := a.GetApkAbi(serial, "com.r2studio.robotmon")

	app, app32, app64 := a.GetAppProcess(serial)
	classPath := "CLASSPATH=" + apk
	ldPath := "LD_LIBRARY_PATH="
	appProcess := ""
	if abi == "arm64-v8a" {
		ldPath += "/system/lib64:/system/lib:"
		ldPath += apkDir + "/lib:" + apkDir + "/lib/arm64"
		if app64 {
			appProcess = "app_process64"
		} else if app {
			appProcess = "app_process"
		} else {
			appProcess = "app_process32"
		}
	} else if abi == "x86" {
		ldPath += "/system/lib:/data/data/com.r2studio.robotmon/lib:"
		ldPath += apkDir + "/lib:" + apkDir + "/lib/x86"
		if app32 {
			appProcess = "app_process32"
		} else {
			appProcess = "app_process"
		}
	} else {
		ldPath += "/system/lib:/data/data/com.r2studio.robotmon/lib:"
		ldPath += apkDir + "/lib:" + apkDir + "/lib/arm"
		if app32 {
			appProcess = "app_process32"
		} else {
			appProcess = "app_process"
		}
	}
	baseCommand := fmt.Sprintf("%s %s %s /system/bin com.r2studio.robotmon.Main $@", ldPath, classPath, appProcess)
	command := fmt.Sprintf("%s sh -c \"%s\" > /dev/null 2> /dev/null && sleep 1 &", nohup, baseCommand)
	fmt.Println("============================\n", ldPath)
	fmt.Println("============================\n", classPath)
	fmt.Println("============================\n", appProcess)
	fmt.Println("============================\n", baseCommand)
	fmt.Println("============================\n", command)
	fmt.Println("============================")
	details := []string{
		ldPath, classPath, appProcess, baseCommand, command,
	}
	return command, details, err
}

func (a *Client) StartRobotmonService(serial string) ([]string, error) {
	pids, err := a.GetPids(serial, "app_process")
	if err != nil {
		return nil, err
	}
	if len(pids) > 0 {
		return pids, nil
	}
	command, _, err := a.GetRobotmonStartCommand(serial)
	if err != nil {
		return nil, err
	}
	// try 3 times
	for i := 0; i < 3; i++ {
		cv := make(chan bool, 1)
		go func() {
			cmd := exec.Command(a.adbPath, "-s", serial, "shell", command)

			_, err := cmd.Output()
			if err != nil {
				cv <- false
			}
			cv <- true
		}()

		// wait for running command
		select {
		case <-cv:
		case <-time.After(5 * time.Second):
		}

		// check pids
		pids, err := a.GetPids(serial, "app_process")
		if err != nil {
			return nil, err
		}
		if len(pids) > 1 {
			return pids, nil
		}
		time.Sleep(1000 * time.Millisecond)
	}
	return nil, fmt.Errorf("Start service failed")
}

func (a *Client) StopService(serial string) error {
	pids, err := a.GetPids(serial, "app_process")
	if err != nil {
		return err
	}
	for _, pid := range pids {
		cmd := exec.Command(a.adbPath, "-s", serial, "shell", "kill "+pid)

		_, err := cmd.Output()
		if err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
	pids, err = a.GetPids(serial, "app_process")
	if len(pids) == 0 {
		return nil
	}
	for _, pid := range pids {
		cmd := exec.Command(a.adbPath, "-s", serial, "shell", "kill -9 "+pid)

		_, err := cmd.Output()
		if err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func (a *Client) GetIPAddress(serial string) string {
	output, _ := a.Shell(serial, "ifconfig")
	lines := strings.Split(output, "\n")
	ip := ""
	for _, line := range lines {
		if !strings.Contains(line, "inet ") || strings.Contains(line, "127.0.0.1") || strings.Contains(line, "0.0.0.0") || strings.Contains(line, ":172.") {
			continue
		}
		fmt.Sscanf(strings.Trim(line, " \t"), "inet addr:%s ", &ip)
		if ip != "" {
			return ip
		}
	}
	output, _ = a.Shell(serial, "netcfg")
	lines = strings.Split(output, "\n")
	for _, line := range lines {
		ss := strings.Split(line, " ")
		for _, s := range ss {
			if len(s) > 10 && strings.Contains(s, "/") {
				tmpIP := s[0:strings.Index(s, "/")]
				if tmpIP == "0.0.0.0" || tmpIP == "127.0.0.1" || strings.HasPrefix(tmpIP, "172.") {
					continue
				}
				return ip
			}
		}
	}
	return ip
}
