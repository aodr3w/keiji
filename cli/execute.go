package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aodr3w/keiji-core/bus"
	"github.com/aodr3w/keiji-core/common"
	c "github.com/aodr3w/keiji-core/constants"
	"github.com/aodr3w/keiji-core/db"
	"github.com/aodr3w/keiji-core/logging"
	"github.com/aodr3w/keiji-core/paths"
	"github.com/aodr3w/keiji-core/utils"
	cmdErrors "github.com/aodr3w/keiji/errors"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var bc = bus.NewBusClient()
var cmdRepo *db.Repo

// serviceLogsMapping is mapping of service to logsPath
var serviceLogsMapping = map[c.Service]string{
	c.SCHEDULER: paths.SCHEDULER_LOGS,
	c.TCP_BUS:   paths.BUS_LOGS,
}

// serviceRepos is a mapping of service to github repo
var serviceRepos = map[c.Service]string{
	c.SCHEDULER: "github.com/aodr3w/keiji-scheduler",
	c.TCP_BUS:   "github.com/aodr3w/keiji-bus",
}

var rootCmd = &cobra.Command{
	Use:   "keiji",
	Short: "keiji CLI",
	Long:  "Keji CLI to manage services and tasks",
}

type Editor string

const (
	VIM           Editor = "vim"
	NANO          Editor = "nano"
	CODE          Editor = "code"
	maxRetries           = 10
	retryInterval        = 1 * time.Second
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
	rootCmd.AddCommand(NewTaskCMD())
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
	gopath, err := getGoPath()
	if err != nil {
		return "", err
	}
	repoPath := ""
	filepath.Walk(filepath.Join(gopath, "pkg", "mod", "github.com", "aodr3w"),
		func(path string, info fs.FileInfo, err error) error {
			if strings.Contains(info.Name(), "keiji-core") {
				repoPath = path
				return filepath.SkipAll
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
	err = utils.CopyFile(filepath.Join(repoPath, "templates", "settings.conf"), paths.WORKSPACE_SETTINGS, 0644)
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
func getGoPath() (string, error) {
	gopath := os.Getenv("GOPATH")
	if !valid(gopath) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		gopath = filepath.Join(homeDir, "go")
	}
	ok, err := utils.PathExists(gopath)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", common.ErrPathNotFound(gopath)
	}
	return gopath, nil
}
func isServiceInstalled(service c.Service) (bool, error) {
	_, err := getServicePath(service)
	if err != nil {
		if errors.Is(err, cmdErrors.ErrServiceNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func InstallService(service c.Service, update bool, cc bool) error {
	ok, err := isServiceInstalled(service)
	if err != nil {
		return err
	}
	if ok && !update {
		logWarn(fmt.Sprintf("service %s is already installed, provide updated=true to update service\n", service))
		return nil
	}
	if cc {
		logWarn("cleaning modcache...")
		err = runCMD(paths.WORKSPACE, true, "go", "clean", "-modcache")
		if err != nil {
			return err
		}
	}

	repoURL, ok := serviceRepos[service]
	if !ok {
		return fmt.Errorf("please provide repo url for %s", service)
	}

	if update {
		logInfo(fmt.Sprintf("updating %s", service))
	} else {
		logInfo(fmt.Sprintf("installing %s", service))
	}
	err = runCMD(paths.WORKSPACE, true, "go", "install", fmt.Sprintf("%v@main", repoURL))
	if err != nil {
		return err
	}

	logInfo("ok")

	if !update {
		return nil
	} else {
		logInfo("restarting service")
		return restartService(service)
	}

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
				installed, err := isServiceInstalled(service)
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
				}
			}
			if !allInstalled {
				for _, s := range missingServices {
					err := InstallService(s, false, false)
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
				_, err := logging.NewFileLogger(logPath)
				if err != nil {
					logError(err)
				}
			}
			return nil
		},
	}
}

func installAllServices(update bool, cc bool) error {
	if cc {
		err := clearCache()
		if err != nil {
			return err
		}
	}
	logWarn("installing all services...")
	for _, s := range c.SERVICES {
		err := InstallService(s, update, false)
		if err != nil {
			return err
		}
	}
	return nil
}
func clearCache() error {
	logWarn("cleaning modcache...")
	return runCMD(paths.WORKSPACE, true, "go", "clean", "-modcache")
}

/*
uninstalls specific service
*/
func uninstallService(service c.Service) error {
	logWarn(fmt.Sprintf("uninstalling service %s", service))
	servicePath, err := getServicePath(service)
	if err != nil {
		return err
	}
	return os.RemoveAll(servicePath)
}

/*
uninstall all keiji related packages
*/
func getPkgPath() (string, error) {
	goPath, err := getGoPath()
	if err != nil {
		return "", err
	}
	pkgPath := filepath.Join(goPath, "pkg", "mod", "github.com", "aodr3w")
	exists, err := utils.PathExists(pkgPath)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", common.ErrPathNotFound(pkgPath)
	}
	return pkgPath, nil
}
func getServicePath(service c.Service) (string, error) {
	goPath, err := getGoPath()
	if err != nil {
		return "", err
	}
	main := "keiji"
	var binPath string
	if strings.EqualFold(string(service), main) {
		binPath = filepath.Join(goPath, "bin", main)
	} else {
		binPath = filepath.Join(goPath, "bin", fmt.Sprintf("%v-%v", main, service))
	}

	ok, err := utils.PathExists(binPath)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", cmdErrors.ErrServiceNotFound
	}
	return binPath, nil
}
func uninstallSystem() error {
	//confirm use of sudo priviledges
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must be run as root. Please use sudo")
	}
	//remove all services
	for _, service := range c.SERVICES {
		err := uninstallService(service)
		if err != nil {
			if errors.Is(err, cmdErrors.ErrServiceNotFound) {
				logError(err)
				continue
			}
			return err
		}
	}
	//delete workspace
	logWarn("removing workspace")
	if err := os.RemoveAll(paths.WORKSPACE); err != nil {
		return err
	}

	//deleting hidden folders
	logWarn("deleting system folder")
	if err := os.RemoveAll(paths.SYSTEM_ROOT); err != nil {
		return err
	}
	//delete packages
	path, err := getPkgPath()
	if err != nil {
		return err
	}
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if strings.Contains(err.Error(), "no such file or directory") {
				return nil
			}
			return err
		}
		if d.IsDir() && strings.Contains(d.Name(), "keiji") || strings.Contains(d.Name(), "logger") {
			//delete the dir
			//call os.RemoveAll() here
			logWarn(fmt.Sprintf("deleting %v", d.Name()))
			removeErr := os.RemoveAll(path)
			if removeErr != nil {
				return removeErr
			}
		}
		return nil
	})

	if err != nil {
		return err
	}
	//remove main programm
	err = uninstallService("keiji")
	if err != nil {
		return err
	}
	logInfo("uninstall complete")
	logInfo("good bye :-)")
	return nil
}

/*
returns a boolean denoting wether a service is running or not
*/
func isServiceRunning(service c.Service) bool {
	pidPath := paths.PID_PATH(service)
	pid, err := readPID(pidPath)
	if err != nil {
		logError(fmt.Sprintf("%s: %v", pidPath, err))
		return false
	}
	err = syscall.Kill(pid, 0)
	if err != nil {
		if err == syscall.ESRCH {
			return false
		} else if err == syscall.EPERM {
			logError("permission denied")
			return false
		}
		logError(fmt.Sprintf("%d: %v", pid, err))
		return false
	}
	return true
}

/*
get status of all services installed or not, running or not
*/
func getServiceInfo() {
	report := make(map[c.Service]c.ServiceStatus)
	for _, service := range c.SERVICES {
		ok, err := isServiceInstalled(service)
		if err != nil {
			logError(err)
			continue
		}
		if !ok {
			logError(fmt.Sprintf("service %s not found", service))
			continue
		}
		ok = isServiceRunning(service)
		if ok {
			report[service] = c.ONLINE
		} else {
			report[service] = c.OFFLINE
		}
	}

	// Print header
	fmt.Println(strings.Repeat("=", 8), "SERVICES", strings.Repeat("=", 8))
	fmt.Printf("%-18s %-18s\n", "NAME", "STATUS")

	// Print each service status
	for k, v := range report {
		fmt.Printf("%-18s %-18s\n", k, v)
	}

	// Print footer
	fmt.Println(strings.Repeat("=", 8), "SERVICES", strings.Repeat("=", 8))
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

func NewTaskCMD() *cobra.Command {
	var create, build, disable, enable, delete, restart, get, force, resolve bool
	var logs, code, vim, nano bool
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
			if !valid(name) && !get {
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
			} else if enable {
				taskError = enableTask(name)
			} else if get {
				taskError = getTask(name)
			} else if delete {
				taskError = deleteTask(name)
			} else if build {
				taskError = buildTask(name, restart)
			} else if resolve {
				taskError = resolveError(name)
			} else if logs {
				taskError = handleGetTaskLogs(name, code, vim, nano)
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
	taskCMD.Flags().BoolVar(&build, "build", false, "provide true to rebuild task executable")
	taskCMD.Flags().BoolVar(&resolve, "resolve", false, "provide true to resolve task.isError")
	taskCMD.Flags().BoolVar(&enable, "enable", false, "provide true to set task.IsDisabled to False")
	taskCMD.Flags().BoolVar(&logs, "logs", false, "returns last 100 log lines for service")
	taskCMD.Flags().BoolVar(&code, "code", false, "opens service logs in vscode")
	taskCMD.Flags().BoolVar(&vim, "vim", false, "opens service logs in vim")
	taskCMD.Flags().BoolVar(&nano, "nano", false, "opens service logs in nano")
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
	//create destination folder
	err = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if d.Name() == "templates" {
			return utils.CopyDir(filepath.Join(path, "tasks"), taskPath, 0644)
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = writeEnvFile(taskPath, name, description)
	if err != nil {
		return err
	}
	logWarn(fmt.Sprintf("creating task %v", name))
	err = runCMD(paths.WORKSPACE, true, "go", "mod", "tidy")
	if err != nil {
		return err
	}
	log.Println(aurora.Green("ok"))
	return nil
}

func enableTask(name string) error {
	_, err := cmdRepo.SetIsDisabled(name, false)
	return err
}

func disableTask(name string) error {
	task, err := cmdRepo.GetTaskByName(name)
	if err != nil {
		return err
	}
	return bc.StopTask(task.TaskId, true, false)
}

func deleteTask(name string) error {
	task, err := cmdRepo.GetTaskByName(name)
	if err != nil {
		return err
	}
	/**check if task is disabled or inError state, if so
	we can delete without sending a message to the scheduler
	because such tasks are ignored by the scheduler
	**/
	if task.IsDisabled || task.IsError {
		err := utils.DeleteTaskExecutable(task.Executable)
		if err != nil {
			return err
		}
		err = utils.DeleteTaskLog(task.LogPath)
		if err != nil {
			return err
		}
		return cmdRepo.DeleteTask(task)
	}
	return bc.StopTask(task.TaskId, false, true)
}

func restartTask(name string) error {
	logWarn(fmt.Sprintf("restarting task %v\n", name))
	task, err := cmdRepo.GetTaskByName(name)
	if err != nil {
		return err
	}
	return bc.StopTask(task.TaskId, false, false)
}

func resolveError(name string) error {
	model, err := cmdRepo.SetIsError(name, false, "")
	log.Printf("resolved : %v, %v\n", model, err)
	return err
}

func getTask(name string) error {
	if valid(name) {
		task, err := cmdRepo.GetTaskByName(name)
		if err != nil {
			return err
		}
		fmt.Println(task)
	} else {
		tasks, err := cmdRepo.GetAllTasks()
		if err != nil {
			return err
		}
		for _, task := range tasks {
			fmt.Println(strings.Repeat("=", 100))
			fmt.Println(task)
			fmt.Println(strings.Repeat("=", 100))
		}
	}
	return nil
}

func buildTask(name string, restart bool) error {
	taskPath := filepath.Join(paths.TASKS_PATH, name)
	exists, err := utils.PathExists(taskPath)
	if err != nil {
		return err
	}

	if !exists {
		return common.ErrPathNotFound(taskPath)
	}
	logInfo("task found , building...")
	err = runCMD(taskPath, false, "go", "run", "main.go", "schedule.go", "function.go", "--schedule")
	if err != nil {
		return err
	}
	if restart {
		return restartTask(name)
	}
	return nil
}

func NewSystemCMD() *cobra.Command {
	//start stop update system services
	var start, stop, logs, update, uninstall, cc bool
	var scheduler, bus, status, restart bool
	var code, vim, nano bool
	systemCMD := cobra.Command{
		Use:   "system",
		Short: "manage system services",
		Long:  "commands start, stop and diagnose system services",
		RunE: func(cmd *cobra.Command, args []string) error {

			if uninstall {
				//first stop the system
				err := stopAllServices()
				if err != nil {
					logError(err)
				}
				//uninstalls all services
				err = uninstallSystem()
				if err != nil {
					logError(err)
				}
				return nil
			}
			if err := checkWorkSpace(); err != nil {
				logError(err)
				return nil
			}
			if start {
				var startError error
				if scheduler {
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
				if scheduler {
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
				if scheduler {
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
				if scheduler {
					updateError = InstallService(c.SCHEDULER, update, cc)
				} else if bus {
					updateError = InstallService(c.TCP_BUS, update, cc)
				} else {
					updateError = installAllServices(update, cc)
				}
				if updateError != nil {
					logError(updateError)
				}
				return nil
			} else if status {
				getServiceInfo()
				return nil
			} else if restart {
				var restartError error
				if scheduler {
					restartError = restartService(c.SCHEDULER)
				} else if bus {
					restartError = restartService(c.TCP_BUS)
				} else {
					restartError = restartAllServices()
				}
				if restartError != nil {
					logError(restartError)
				}
				return nil
			}
			return fmt.Errorf("no flag provided")
		},
	}
	systemCMD.Flags().BoolVar(&scheduler, "scheduler", false, "manage scheduler service")
	systemCMD.Flags().BoolVar(&bus, "bus", false, "manage tcp-bus service")
	systemCMD.Flags().BoolVar(&start, "start", false, "starts system services")
	systemCMD.Flags().BoolVar(&stop, "stop", false, "stops system services")
	systemCMD.Flags().BoolVar(&logs, "logs", false, "returns last 100 log lines for service")
	systemCMD.Flags().BoolVar(&code, "code", false, "opens service logs in vscode")
	systemCMD.Flags().BoolVar(&vim, "vim", false, "opens service logs in vim")
	systemCMD.Flags().BoolVar(&nano, "nano", false, "opens service logs in nano")
	systemCMD.Flags().BoolVar(&update, "update", false, "updates service is specified otherwise all")
	systemCMD.Flags().BoolVar(&uninstall, "uninstall", false, "uinstalls all services and packages")
	systemCMD.Flags().BoolVar(&status, "status", false, "get status of system services")
	systemCMD.Flags().BoolVar(&restart, "restart", false, "restart all services")
	systemCMD.Flags().BoolVar(&cc, "cc", false, "clears go mod cache")
	return &systemCMD
}

func getServiceLogPath(service c.Service) (string, error) {
	switch service {
	case c.SCHEDULER:
		return paths.SCHEDULER_LOGS, nil
	case c.TCP_BUS:
		return paths.BUS_LOGS, nil
	default:
		return "", fmt.Errorf("invalid service name %v", service)
	}
}

func startService(service c.Service) error {
	//check if service is running first
	isRunning := isServiceRunning(service)
	if isRunning {
		logWarn("service already running")
		return nil
	}
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
		return -1, cmdErrors.ErrPIDNotFound
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
func restartAllServices() error {
	logWarn("restarting all services")
	for _, service := range c.SERVICES {
		err := restartService(service)
		if err != nil {
			return err
		}
	}
	logInfo("ok")
	return nil
}
func restartService(service c.Service) error {
	isRunning := isServiceRunning(service)
	if !isRunning {
		logWarn("service is not running.")
		return nil
	}
	err := stopService(service)
	if err != nil {
		return err
	}
	return startService(service)
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

	for c := 0; c < maxRetries; c++ {
		r := isServiceRunning(service)
		if !r {
			logInfo(fmt.Sprintf("%s stopped successfully", service))
			return nil
		}
		time.Sleep(retryInterval)
	}
	return fmt.Errorf("failed to stop service after %d retries, run ps aux to inspect", maxRetries)
}

func handleGetTaskLogs(name string, code, vim, nano bool) error {
	task, err := cmdRepo.GetTaskByName(name)
	if err != nil {
		return err
	}
	return handleGetLogs(task.LogPath, code, vim, nano)
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
	osType := runtime.GOOS
	switch editor {
	case VIM, NANO:
		switch osType {
		case "linux":
			cmd = exec.Command(string(editor), path)
		case "darwin":
			// Use osascript to open a new terminal window and run the editor
			script := fmt.Sprintf(`tell application "Terminal"
			        do script "%s %s"
			        activate
			    end tell`, editor, path)
			cmd = exec.Command("osascript", "-e", script)
		default:
			return fmt.Errorf("unsupported operating system: %s", osType)
		}
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
	_, err = logging.NewFileLogger(logsPath)
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
	if !silence && len(strings.Trim(outputStr, "")) > 0 {
		fmt.Println(string(output))
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

func Execute() {
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
