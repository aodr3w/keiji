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
}

func getTemplateRepoPath() (string, error) {
	err := runCMD(paths.WORKSPACE, "go", "get", "-u", "github.com/aodr3w/keiji-core")
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
	err = runCMD(paths.WORKSPACE, "go", "mod", "init", "workspace")
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
func NewInitCMD() *cobra.Command {
	initCMD := &cobra.Command{
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
				err = runCMD(paths.WORKSPACE, "go", "get", "github.com/aodr3w/keiji-tasks@latest")
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
	return initCMD
}

func runCMD(targetDir string, ss ...string) error {
	if len(ss) == 0 {
		return fmt.Errorf("no command provided")
	}
	cmd := exec.Command(ss[0], ss[1:]...)
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	out := string(output)
	log.Println(out)
	if err != nil {
		return fmt.Errorf("failed to run go command: %v , output: %s", err, out)
	}
	return nil
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
