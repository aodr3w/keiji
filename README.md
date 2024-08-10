# KEIJI - 計時 (time keeping)

## MOTIVIATION

- Develop an easy to use `task scheduling system` in go.


# SYSTEM OVERVIEW
![Keiji Scheduling System Overview](images/KEIJI-SCHEDULING-SYSTEM-OVERVIEW.png)


## INSTALLATION

`go install https://github.com/aodr3w/keiji@latest`

## USAGE

# Initializing a workspace
 - Before you can do anything on keiji, you have to initialize a workspace, which is a folder where all your tasks as well as settings related to `DATABASE, LOG MANAGEMENT & TIMEZONE preferences` will be found. this can be done with the `init command`

**command**

`keiji init`

After running the init command there should be a workspace folder at `$HOME/keiji`. Inside this folder you should see the following:
- `setting.conf` -> contains settings for `DATABASE`, `LOGGING` && `TIMEZONE`.
- `/tasks` -> created tasks go here
- `go.mod` -> installed packages used by tasks go here

the `init` command does other things such as:
- installing the scheduler & bus services.
- initializing log files for services located in `$HOME/.keiji/logs`


## create tasks

**command**
creating a task requires the following command;

```
# keiji task --create --name=<task_name> --desc=<task_description>
```

here is an example;

```
# keiji task --create --name=ping_google --desc="pings google"
```

once done , check your $WORKSPACE/tasks folder , you should find a folder with the
task name containing 3 files;

```
$WORKSPACE/
└── tasks/
    └── ping_google/
        ├── schedule.go
        ├── main.go
        └── function.go
```
**schedule.go**
 - the scheduling logic for your task goes here. Tasks can be of 2 types, `HMS tasks` or `DayTime tasks`. 

 - `HMS` --> `Hours or Minutes or Seconds` --> a task run on a fixed interval e.g

 ```
 tasks.NewSchedule().Run().Every(10).Seconds().Build()

 ```
 
 - `DayTime` --> `Day and Time` --> a task run on a specific day at a specific time e.g

 ```
 return tasks.NewSchedule().On().Monday().At("10:00PM").Build()

 ```
 - the time value passed must in a valid format i.e 12h or 24h
 
 ## build / save task
 - before a task is run it has to be `built`. Building at a task, creates/updates the task binary located at $HOME/.keiji/exec
 and stores task scheduling information in the database , so it can be picked up by the scheduler.

`command` : `keiji task --build --name=<task_name>`

## start system
- inorder for tasks to be scheduled, the system (`scheduler and bus service`) have to both be running.

`command` : `keiji system --start`


## view logs
- you can confirm the system is running by checking the logs

- scheduler logs: `keiji system --logs --scheduler`
- bus logs: `keiji system --logs --bus`
- you can also open the logs files in one of 3 editors, namely; `vim`, `nano` or `code` simply add the flag to
the command e.g `keiji system --logs --scheduler --code`

## view task logs
- `keiji task --logs --name=<task_name> --<editor>`

## change task schedule / source code / restart task
to change a task;
- open workspace in your favourite editor
- change the task `logic (function.go)` or `schedule (schedule.go)`
- build task with restart flag; `keiji task --build --name=<task_name> --restart`
- check task logs to confirm your changes have taken effect e.g `keiji task --logs --name=<task_name> --code`


## disable a task
`keiji task --disable --name=<task_name>`

## delete a task
`keiji task --delete --name=<task_name>`

## stop system
`keiji system --stop`


## restart system
`keiji system --restart`


## uninstall system
`keiji system --uninstall`


