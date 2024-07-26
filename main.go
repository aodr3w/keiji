package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cmdErrors "github.com/aodr3w/keiji-cli/errors"
	c "github.com/aodr3w/keiji-core/constants"
	"github.com/aodr3w/keiji-core/db"
	"github.com/aodr3w/keiji-core/paths"
	"github.com/aodr3w/keiji-core/utils"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var cmdRepo *db.Repo

// var serviceLogsMapping = map[c.Service]string{
// 	c.HTTP:      paths.HTTP_SERVER_LOGS,
// 	c.SCHEDULER: paths.SCHEDULER_LOGS,
// 	c.CLEANER:   paths.CLEANER_LOGS,
// 	c.TCP_BUS:   paths.TCP_BUS_LOGS,
// }

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

func getRepo() (*db.Repo, error) {
	return db.NewRepo()
}

func checkWorkSpace() error {
	fmt.Println("checking workspace")
	var err error
	if !utils.IsInit() {
		err := fmt.Errorf("unitialized workspace error")
		log.Println(aurora.Red("please initialize your workspace to continue"))
		return err
	}
	if cmdRepo == nil {
		cmdRepo, err = getRepo()
	}
	return err
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	rootCmd.AddCommand(NewInitCMD())
	rootCmd.AddCommand(taskCMD())
}

func getTemplateRepoPath() (string, error) {
	err := runCMD(paths.WORKSPACE, false, "go", "get", "-u", "github.com/aodr3w/keiji-core")
	if err != nil {
		fmt.Printf("Error pulling repository: %v\n", err)
		return "", err
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
func createWorkSpace() error {
	err := os.MkdirAll(paths.TASKS_PATH, 0755)
	if err != nil {
		return err
	}
	//do a go mod init workspace
	err = runCMD(paths.WORKSPACE, false, "go", "mod", "init", "workspace")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	repoPath, err := getTemplateRepoPath()
	if err != nil {
		return err
	}
	//copy settings.conf file to workSpace
	err = utils.CopyFile(filepath.Join(repoPath, "templates", "settings.conf"), paths.WORKSPACE_SETTINGS)
	if err != nil {
		return err
	}
	exists, err := utils.DirectoryExists(paths.TASKS_PATH)
	if !exists || err != nil {
		return cmdErrors.ErrWorkSpaceInit("work space creation error: path: %v exists: %v err: %v", paths.TASKS_PATH, exists, err)
	}
	exists, err = utils.DirectoryExists(paths.WORKSPACE)
	if !exists || err != nil {
		return fmt.Errorf("work space creation error: path: %v exists: %v err: %v", paths.WORKSPACE, exists, err)
	}
	exists, err = utils.DirectoryExists(paths.WORKSPACE_SETTINGS)
	if !exists || err != nil {
		return fmt.Errorf("work space creation error: path: %v exists: %v err: %v", paths.WORKSPACE_SETTINGS, exists, err)
	}
	exists, err = utils.DirectoryExists(paths.WORKSPACE_MODULE)
	if !exists || err != nil {
		return fmt.Errorf("work space creation error: path: %v exists: %v err: %v", paths.WORKSPACE_MODULE, exists, err)
	}
	return nil
}

func allServicesInstalled() ([]c.Service, bool) {
	missingServices := make([]c.Service, 0)
	ok := true
	for service := range serviceRepos {
		if !isServiceInstalled(service) {
			log.Println(aurora.Red(fmt.Sprintf("service %s not found", service)))
			missingServices = append(missingServices, service)
			if ok {
				ok = false
			}
		} else {
			log.Println(aurora.Green(fmt.Sprintf("service %s already installed", service)))
		}
	}
	return missingServices, ok
}

func isServiceInstalled(service c.Service) bool {
	gopath := os.Getenv("GOPATH")
	if !valid(gopath) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Println(aurora.Red(fmt.Sprintf("failed to get home directory: %v", err)))
			return false
		}
		gopath = filepath.Join(homeDir, "go")
	}
	binPath := filepath.Join(gopath, "bin", fmt.Sprintf("%v-%v", "keiji", service))
	ok, err := utils.DirectoryExists(binPath)
	if err != nil {
		log.Println(aurora.Red(fmt.Sprintf("%v", err)))
	}
	return err == nil && ok
}
func InstallService(service c.Service, update bool) error {
	repoURL, ok := serviceRepos[service]
	if !ok {
		return fmt.Errorf("please provide repo url for %s", service)
	}
	if isServiceInstalled(service) && !update {
		log.Println(
			aurora.BrightGreen(fmt.Sprintf("service %s is already installed, provide updated=true to update service\n", service)))
	} else {
		log.Println(aurora.Yellow(fmt.Sprintf("installing or updating service %s", service)))
		err := runCMD(paths.WORKSPACE, true, "go", "install", fmt.Sprintf("%v@latest", repoURL))
		if err != nil {
			return err
		}
	}
	return nil
}
func NewInitCMD() *cobra.Command {
	//TODO
	/*add service installion to init process
	 */
	return &cobra.Command{
		Use:   "init",
		Short: "initialize workspace",
		Long:  "initializes workspace by creating required directories and installing services",
		RunE: func(cmd *cobra.Command, args []string) error {
			//initialize work space folder
			if !utils.IsInit() {
				log.Println(aurora.Yellow("Initializing work space..."))
				err := createWorkSpace()
				if err != nil {
					return err
				}
				err = runCMD(paths.WORKSPACE, false, "go", "get", "github.com/aodr3w/keiji-tasks@latest")
				if err != nil {
					return err
				}
				cmdRepo, err = getRepo()
				if err != nil {
					return err
				}
			}
			//install services after initializing work space
			ms, ok := allServicesInstalled()
			if !ok {
				//clear mod cache first
				err := runCMD(paths.WORKSPACE, true, "go", "clean", "-modcache")
				if err != nil {
					return err
				}
				for _, s := range ms {
					err := InstallService(s, false)
					if err != nil {
						return err
					}
				}
			} else {
				if len(ms) > 0 {
					return fmt.Errorf("missing services %v", ms)
				}
			}
			return nil
		},
	}
}

func taskCMD() *cobra.Command {
	var create, disable, delete, restart, get, force bool
	var name, description string
	taskCMD := cobra.Command{
		Use:   "task",
		Short: "keiji task management",
		Long:  "cobra commands to create, update, deploy, or delete tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := checkWorkSpace()
			if err != nil {
				return err
			}
			if !valid(name) {
				return fmt.Errorf("please provide name for your task")
			}
			if create {
				if !valid(description) {
					return fmt.Errorf("please provide a description for your task")
				}
				return createTask(name, description, force)
			}
			if disable {
				return disableTask(name)
			}
			if get {
				return getTask(name)
			}
			if delete {
				return deleteTask(name)
			}
			if restart {
				return restartTask(name)
			}
			return fmt.Errorf("please pass a valid command")

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
	exists, err := utils.DirectoryExists(taskPath)
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
	repoPath, err := getTemplateRepoPath()
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

func runCMD(targetDir string, silence bool, ss ...string) error {
	if len(ss) == 0 {
		return fmt.Errorf("no command provided")
	}
	cmd := exec.Command(ss[0], ss[1:]...)
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if !silence {
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
