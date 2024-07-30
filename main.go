package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	cmdErrors "github.com/aodr3w/keiji-cli/errors"
	c "github.com/aodr3w/keiji-core/constants"
	"github.com/aodr3w/keiji-core/db"
	"github.com/aodr3w/keiji-core/paths"
	"github.com/aodr3w/keiji-core/utils"
	"github.com/aodr3w/logger"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var cmdRepo *db.Repo

// serviceLogsMapping is mapping of service to logsPath
var serviceLogsMapping = map[c.Service]string{
	c.SERVER:    paths.HTTP_SERVER_LOGS,
	c.SCHEDULER: paths.SCHEDULER_LOGS,
	c.TCP_BUS:   paths.TCP_BUS_LOGS,
}

// serviceRepos is a mapping of service to github repo
var serviceRepos = map[c.Service]string{
	c.SCHEDULER: "github.com/aodr3w/keiji-scheduler",
	c.SERVER:    "github.com/aodr3w/keiji-server",
	c.TCP_BUS:   "github.com/aodr3w/keiji-bus",
}

var rootCmd = &cobra.Command{
	Use:   "keiji",
	Short: "keiji CLI",
	Long:  "Keji CLI to manage services and tasks",
}

type Editor string

const (
	VIM  Editor = "vim"
	NANO Editor = "nano"
	CODE Editor = "code"
)

/*
newRepo is a factory function for repo instances.
The caller of this function must remember to call instance.close()
when done with the instance to prevent memory leaks
*/
func newRepo() (*db.Repo, error) {
	return db.NewRepo()
}

/*
check workSpace confirms wether or not a workspace folder
has been successfully initialized by the user
*/
func checkWorkSpace() error {
	var err error
	if !utils.IsInit() {
		return fmt.Errorf("please initialize your workspace to continue")
	}
	if cmdRepo == nil {
		cmdRepo, err = newRepo()
	}
	return err
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	rootCmd.AddCommand(NewInitCMD())
	rootCmd.AddCommand(taskCMD())
	rootCmd.AddCommand(NewSystemCMD())
}

/*
getTemplateRepoPath returns a path and error.
The path points to the location of the workspace templates
*/
func getTemplateRepoPath(get bool) (string, error) {
	if get {
		err := runCMD(paths.WORKSPACE, true, "go", "get", "-u", "github.com/aodr3w/keiji-core")
		if err != nil {
			fmt.Printf("Error pulling repository: %v\n", err)
			return "", err
		}
	}

	//step 2: Locate the repository in the GoPATH
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}
	repoPath := ""
	filepath.Walk(filepath.Join(gopath, "pkg", "mod", "github.com", "aodr3w"),
		func(path string, info fs.FileInfo, err error) error {
			if strings.Contains(info.Name(), "keiji-core") {
				repoPath = path
				return filepath.SkipDir
			}
			return nil
		},
	)
	if repoPath == "" {
		return "", fmt.Errorf("could not find the keiji-core repository in the GoPath")
	}
	return repoPath, nil
}

/*
createWorkSpace function creates the workspace folder in the $HOME directory
*/
func createWorkSpace() error {
	err := os.MkdirAll(paths.TASKS_PATH, 0755)
	if err != nil {
		return err
	}
	//do a go mod init workspace
	err = runCMD(paths.WORKSPACE, true, "go", "mod", "init", "workspace")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	repoPath, err := getTemplateRepoPath(true)
	if err != nil {
		return err
	}
	//copy settings.conf file to workSpace
	err = utils.CopyFile(filepath.Join(repoPath, "templates", "settings.conf"), paths.WORKSPACE_SETTINGS)
	if err != nil {
		return err
	}
	exists, err := utils.PathExists(paths.TASKS_PATH)
	if !exists || err != nil {
		return cmdErrors.ErrWorkSpaceInit("work space creation error: path: %v exists: %v err: %v", paths.TASKS_PATH, exists, err)
	}
	exists, err = utils.PathExists(paths.WORKSPACE)
	if !exists || err != nil {
		return fmt.Errorf("work space creation error: path: %v exists: %v err: %v", paths.WORKSPACE, exists, err)
	}
	exists, err = utils.PathExists(paths.WORKSPACE_SETTINGS)
	if !exists || err != nil {
		return fmt.Errorf("work space creation error: path: %v exists: %v err: %v", paths.WORKSPACE_SETTINGS, exists, err)
	}
	exists, err = utils.PathExists(paths.WORKSPACE_MODULE)
	if !exists || err != nil {
		return fmt.Errorf("work space creation error: path: %v exists: %v err: %v", paths.WORKSPACE_MODULE, exists, err)
	}
	return nil
}

/*
isService installed checks wether or not a service has been installed
in the gopath
*/
func isServiceInstalled(service c.Service) (error, bool) {
	gopath := os.Getenv("GOPATH")
	if !valid(gopath) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err, false
		}
		gopath = filepath.Join(homeDir, "go")
	}
	binPath := filepath.Join(gopath, "bin", fmt.Sprintf("%v-%v", "keiji", service))
	ok, err := utils.PathExists(binPath)
	if err != nil {
		return err, false
	}

	if !ok {
		return nil, false
	}

	return nil, true
}

func InstallService(service c.Service, update bool) error {
	logWarn("cleaning modcache...")
	err := runCMD(paths.WORKSPACE, true, "go", "clean", "-modcache")
	if err != nil {
		return err
	}

	logWarn("running go mod tidy...")
	err = runCMD(paths.WORKSPACE, true, "go", "mod", "tidy")
	if err != nil {
		return err
	}
	repoURL, ok := serviceRepos[service]
	if !ok {
		return fmt.Errorf("please provide repo url for %s", service)
	}
	err, ok = isServiceInstalled(service)
	if err != nil {
		return err
	}
	if ok && !update {
		logWarn(fmt.Sprintf("service %s is already installed, provide updated=true to update service\n", service))
	}
	logWarn(fmt.Sprintf("installing or updating service %s", service))
	if update {
		err := runCMD(paths.WORKSPACE, true, "go", "get", "-u", fmt.Sprintf("%v@latest", repoURL))
		if err != nil {
			return err
		}
	}
	err = runCMD(paths.WORKSPACE, true, "go", "install", fmt.Sprintf("%v@latest", repoURL))
	if err != nil {
		return err
	}

	return nil
}
func NewInitCMD() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "initialize workspace",
		Long:  "initializes workspace by creating required directories and installing services",
		RunE: func(cmd *cobra.Command, args []string) error {
			//initialize work space folder
			if !utils.IsInit() {
				logWarn("Initializing work space...")
				err := createWorkSpace()
				if err != nil {
					logError(err)
					return nil
				}
				err = runCMD(paths.WORKSPACE, true, "go", "get", "github.com/aodr3w/keiji-tasks@latest")
				if err != nil {
					logError(err)
					return nil
				}
				cmdRepo, err = newRepo()
				if err != nil {
					logError(err)
					return nil
				}
			} else {
				logInfo("workspace already initialized.")
			}
			//install services after initializing work space
			missingServices := make([]c.Service, 0)
			allInstalled := true
			for service := range serviceRepos {
				err, installed := isServiceInstalled(service)
				if err != nil {
					logError(err)
					return nil
				}
				if !installed {
					logError(fmt.Sprintf("service %s not found", service))
					missingServices = append(missingServices, service)
					if allInstalled {
						allInstalled = false
					}
				} else {
					logInfo(fmt.Sprintf("service %s already installed", service))
				}
			}
			if !allInstalled {
				for _, s := range missingServices {
					err := InstallService(s, false)
					if err != nil {
						logError(err)
						return nil
					}
				}
			} else {
				if len(missingServices) > 0 {
					logError(fmt.Errorf("missing services %v", missingServices))
				}
			}

			//init logfiles for service if not already present
			for _, service := range c.SERVICES {
				//initialize service logFile here
				logPath, ok := serviceLogsMapping[service]
				if !ok {
					logError(fmt.Errorf("logPath not found for service %v", service))
				}
				_, err := logger.NewFileLogger(logPath)
				if err != nil {
					logError(err)
				}
			}
			return nil
		},
	}
}

func installAllServices(update bool) error {
	logWarn("installing all services...")
	for _, s := range c.SERVICES {
		err := InstallService(s, update)
		if err != nil {
			return err
		}
	}
	return nil
}
func logInfo(msg interface{}) {
	log.Println(aurora.Green(msg))
}
func logWarn(msg interface{}) {
	log.Println(aurora.Yellow(msg))
}

func logError(msg interface{}) {
	log.Println(aurora.Red(msg))
}

func taskCMD() *cobra.Command {
	var create, disable, delete, restart, get, force bool
	var name, description string
	taskCMD := cobra.Command{
		Use:   "task",
		Short: "keiji task management",
		Long:  "cobra commands to create, update, deploy, or delete tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			var taskError error
			if err := checkWorkSpace(); err != nil {
				logError(err)
				return nil
			}
			if !valid(name) {
				logError(fmt.Errorf("please provide name for your task"))
				return nil
			}
			if create {
				if !valid(description) {
					taskError = fmt.Errorf("please provide a description for your task")
				} else {
					taskError = createTask(name, description, force)
				}
			} else if disable {
				taskError = disableTask(name)
			} else if get {
				taskError = getTask(name)
			} else if delete {
				taskError = deleteTask(name)
			} else if restart {
				taskError = restartTask(name)
			} else {
				return fmt.Errorf("please pass a valid command")
			}
			if taskError != nil {
				logError(taskError)
			}
			return nil
		},
	}
	taskCMD.Flags().StringVar(&name, "name", "", "provid a name for your task")
	taskCMD.Flags().StringVar(&description, "desc", "", "provid a description for your task")
	taskCMD.Flags().BoolVar(&create, "create", false, "provide true to create task")
	taskCMD.Flags().BoolVar(&disable, "disable", false, "provide true to disable task")
	taskCMD.Flags().BoolVar(&delete, "delete", false, "provide true to delete task")
	taskCMD.Flags().BoolVar(&restart, "restart", false, "provide true to restart task")
	taskCMD.Flags().BoolVar(&get, "get", false, "provid true to get task info (returns all tasks if name not provided)")
	taskCMD.Flags().BoolVar(&force, "force", false, "provid true to force createTask operation")
	return &taskCMD
}
func createTask(name string, description string, force bool) error {
	//check if task exists
	taskPath := filepath.Join(paths.TASKS_PATH, name)
	exists, err := utils.PathExists(taskPath)
	if err != nil {
		return err
	}
	if exists {
		if force {
			//delete task folder
			err := os.RemoveAll(taskPath)
			if err != nil {
				return err
			}
		} else {
			log.Println("task already exists in TASK_PATH")
			return nil
		}

	}
	//create task
	err = os.MkdirAll(taskPath, 0755)
	if err != nil {
		return err
	}
	repoPath, err := getTemplateRepoPath(false)
	if err != nil {
		return err
	}
	log.Println("copying template files")
	for _, f := range []string{"function", "schedule"} {
		dstPath := filepath.Join(taskPath, f)
		err = os.MkdirAll(dstPath, 0755)
		if err != nil {
			return err
		}
		err = utils.CopyFile(
			filepath.Join(repoPath, "templates", "tasks", fmt.Sprintf("%v/main.go", f)),
			fmt.Sprintf("%v/main.go", dstPath),
		)
		if err != nil {
			return err
		} else {
			log.Println(aurora.Green(fmt.Sprintf("%v copied", f)))
		}
		err := runCMD(paths.WORKSPACE, true, "go", "mod", "tidy")
		if err != nil {
			return err
		} else {
			log.Println(aurora.Green(fmt.Sprintln("task created")))
		}
	}

	return writeEnvFile(taskPath, name, description)
}

func disableTask(name string) error {
	log.Printf("disabling task %v\n", name)
	return nil
}

func deleteTask(name string) error {
	log.Printf("deleting task %v\n", name)
	return nil
}
func restartTask(name string) error {
	log.Printf("restarting task %v\n", name)
	return nil
}

func getTask(name string) error {
	if valid(name) {
		log.Printf("getting task %v\n", name)
	} else {
		log.Printf("getting all tasks..")
	}
	return nil
}

func NewSystemCMD() *cobra.Command {
	//start stop update system services
	var start, stop, logs, update bool
	var server, scheduler, bus bool
	var code, vim, nano bool
	systemCMD := cobra.Command{
		Use:   "system",
		Short: "manage system services",
		Long:  "commands start, stop and diagnose system services",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkWorkSpace(); err != nil {
				logError(err)
				return nil
			}
			if start {
				var startError error
				if server {
					startError = startService(c.SERVER)
				} else if scheduler {
					startError = startService(c.SCHEDULER)
				} else if bus {
					startError = startService(c.TCP_BUS)
				} else {
					startError = startAllServices()
				}
				if startError != nil {
					logError(startError)
				}
				return nil
			} else if stop {
				var stopError error
				if server {
					stopError = stopService(c.SERVER)
				} else if scheduler {
					stopError = stopService(c.SCHEDULER)
				} else if bus {
					stopError = stopService(c.TCP_BUS)
				} else {
					stopError = stopAllServices()
				}
				if stopError != nil {
					logError(stopError)
				}
				return nil
			} else if logs {
				var logsError error
				if server {
					logsError = handleGetServiceLogs(c.SERVER, code, vim, nano)
				} else if scheduler {
					logsError = handleGetServiceLogs(c.SCHEDULER, code, vim, nano)
				} else if bus {
					logsError = handleGetServiceLogs(c.TCP_BUS, code, vim, nano)
				} else {
					return fmt.Errorf("no flag provided")
				}
				if logsError != nil {
					logError(logsError)
				}
				return nil
			} else if update {
				var updateError error
				if server {
					updateError = InstallService(c.SERVER, update)
				} else if scheduler {
					updateError = InstallService(c.SCHEDULER, update)
				} else if bus {
					updateError = InstallService(c.TCP_BUS, update)
				} else {
					updateError = installAllServices(update)
				}
				if updateError != nil {
					logError(updateError)
				}
				return nil
			}
			return fmt.Errorf("no flag provided")
		},
	}
	systemCMD.Flags().BoolVar(&server, "server", false, "manage server service")
	systemCMD.Flags().BoolVar(&scheduler, "scheduler", false, "manage scheduler service")
	systemCMD.Flags().BoolVar(&bus, "bus", false, "manage tcp-bus service")
	systemCMD.Flags().BoolVar(&start, "start", false, "starts system services")
	systemCMD.Flags().BoolVar(&stop, "stop", false, "stops system services")
	systemCMD.Flags().BoolVar(&logs, "logs", false, "returns last 100 log lines for service")
	systemCMD.Flags().BoolVar(&code, "code", false, "opens service logs in vscode")
	systemCMD.Flags().BoolVar(&vim, "vim", false, "opens service logs in vim")
	systemCMD.Flags().BoolVar(&nano, "nano", false, "opens service logs in nano")
	systemCMD.Flags().BoolVar(&update, "update", false, "updates service is specified otherwise all")
	return &systemCMD
}

func getServiceLogPath(service c.Service) (string, error) {
	switch service {
	case c.SERVER:
		return paths.HTTP_SERVER_LOGS, nil
	case c.SCHEDULER:
		return paths.SCHEDULER_LOGS, nil
	case c.TCP_BUS:
		return paths.TCP_BUS_LOGS, nil
	default:
		return "", fmt.Errorf("invalid service name %v", service)
	}
}

func startService(service c.Service) error {
	//we need to run the service while also retrieving its pid
	//get service log file
	err := runServiceCMD(service)
	if err != nil {
		return err
	}
	pid, err := readPID(paths.PID_PATH(service))
	if err != nil {
		return err
	}
	log.Printf("service started with pid %v\n", pid)
	return nil
}
func readPID(pidPath string) (int, error) {
	exists, err := utils.PathExists(filepath.Dir(pidPath))

	if err != nil {
		return -1, err
	}

	if !exists {
		return -1, fmt.Errorf("pid path not found")
	}
	f, err := os.Open(pidPath)
	if err != nil {
		return -1, err
	}
	reader := bufio.NewReader(f)
	data, err := reader.ReadBytes('\n')
	if err != nil {
		return -1, err
	}
	pid, err := strconv.Atoi(strings.ReplaceAll(string(data), "\n", ""))

	if err != nil {
		return -1, err
	}

	return pid, nil
}
func stopService(service c.Service) error {
	//stop service using its pid
	pidPath := paths.PID_PATH(service)
	exists, err := utils.PathExists(pidPath)
	if err != nil {
		return fmt.Errorf("error retrieving pidPath: %v", err)
	}
	if !exists {
		return fmt.Errorf("pid path not found for service: %v", service)
	}
	PID, err := readPID(pidPath)
	if err != nil {
		return fmt.Errorf("error reading service PID: %v", err)
	}
	err = syscall.Kill(PID, syscall.SIGINT)
	if err != nil {
		return fmt.Errorf("kill error: %v", err)
	}
	return nil
}

func handleGetServiceLogs(service c.Service, code, vim, nano bool) error {
	path := serviceLogsMapping[service]
	if valid(path) {
		return handleGetLogs(path, code, vim, nano)
	}
	return fmt.Errorf("logs path for service %v not found", service)
}

func handleGetLogs(path string, code, vim, nano bool) error {
	var editor Editor
	if code {
		editor = CODE
	}
	if vim {
		editor = VIM
	}
	if nano {
		editor = NANO
	}
	if valid(editor) {
		return OpenInEditor(editor, path)
	}
	logsLines, err := utils.GetLogLines(path)
	if err != nil {
		return err
	}
	for _, line := range logsLines.Content {
		fmt.Println(line)
	}
	return nil
}

func OpenInEditor(editor Editor, path string) error {
	var cmd *exec.Cmd

	switch editor {
	case VIM, NANO:
		// Use osascript to open a new terminal window and run the editor
		script := fmt.Sprintf(`tell application "Terminal"
            do script "%s %s"
            activate
        end tell`, editor, path)
		cmd = exec.Command("osascript", "-e", script)
	case CODE:
		cmd = exec.Command(string(editor), path)
	default:
		return fmt.Errorf("unsupported editor: %s", editor)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startAllServices() error {
	for _, service := range c.SERVICES {
		err := startService(service)
		if err != nil {
			return err
		}
	}
	return nil
}
func stopAllServices() error {
	for _, service := range c.SERVICES {
		err := stopService(service)
		if err != nil {
			return err
		}
	}
	return nil
}

func valid(data interface{}) bool {
	switch v := data.(type) {
	case c.Service:
		return len(v) > 0
	case Editor:
		return len(v) > 0 && v == "code" || v == "vim" || v == "nano"
	case string:
		return len(v) > 0
	case int:
		return v >= 1
	case bool:
		return true
	default:
		return false
	}
}

func runServiceCMD(service c.Service) error {
	logsPath, err := getServiceLogPath(service)
	if err != nil {
		return err
	}
	_, err = logger.NewFileLogger(logsPath)
	if err != nil {
		return err
	}
	pidPath := paths.PID_PATH(service)
	err = os.MkdirAll(filepath.Dir(pidPath), 0755)
	if err != nil {
		return err
	}
	cmdStr := fmt.Sprintf("keiji-%s > %v 2>&1 & echo $! > %v", service, logsPath, pidPath)
	cmd := exec.Command("sh", "-c", cmdStr)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}
	return nil
}

func runCMD(targetDir string, silence bool, ss ...string) error {
	if len(ss) == 0 {
		return fmt.Errorf("no command provided")
	}
	cmd := exec.Command(ss[0], ss[1:]...)
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if !silence && len(outputStr) > 0 {
		log.Printf("[runCMD]: %v\n", string(output))
	}
	if err != nil {
		return fmt.Errorf("failed to run go command: %v , output: %s", err, string(output))
	}
	return nil
}

func writeEnvFile(taskDir, task, description string) error {
	envFilePath := filepath.Join(taskDir, ".env")
	envFile, err := os.Create(envFilePath)
	if err != nil {
		return err
	}
	defer envFile.Close()
	_, err = envFile.WriteString(fmt.Sprintf("TASK_NAME='%s'\nTASK_DESCRIPTION='%s'\n", task, description))
	return err
}

func main() {
	defer func() {
		if cmdRepo != nil {
			cmdRepo.Close()
		}
	}()
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
