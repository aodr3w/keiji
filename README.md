# KEIJI - 計時 (time keeping)

## MOTIVIATION

- Develop a concurrent task scheduling system in go.


## SYSTEM OVERVIEW
![Keiji Scheduling System Overview](images/KEIJI-SCHEDULING-SYSTEM-OVERVIEW.png)

### description
keiji is a concurrent task scheduling system written in Go, designed to efficiently manage and execute tasks on a scheduled basis. The system consists of five main components that work together to provide robust task scheduling and management.

### components

#### keiji-main

Provides a command line interface (CLI) for interacting with tasks and services.
Users can create, manage, and monitor tasks using straightforward CLI commands.
Supports commands for initializing workspaces, creating tasks, and more.

####  keiji-core

Acts as the backbone of the entire system by providing essential services such as logging, storage, and utility functions.
Ensures consistent and reusable functionality across all components.
Handles database access, task creation APIs, templates, and more.

####  keiji-bus

Facilitates inter-process communication between the CLI and the scheduler. The bus acts as a bridge, relaying commands such as stop, disable, and delete from the CLI to the scheduler. The scheduler then listens for these commands on the bus and translates them into actionable directives that are executed as needed. These directives can be task specific or system wide.

####  keiji-scheduler

Manages the execution of tasks based on the schedule defined for the task.
Reads commands from the bus and applies them as needed. These commands may be task directives e.g disable delete, stop or system level directives e.g shutdown.
Logs all activities to provide detailed insights into task execution and system status.

####  task-binary

Each task, once created, is compiled into a binary executable.
These binaries are executed on their scheduled intervals, with the scheduler managing their lifecycle.

#### work-space

The workspace is a folder located at `$HOME/keiji`, designated for the development of tasks. It is created when the `init` command is run.


## INSTALLATION

```
go install github.com/aodr3w/keiji@latest
```

After install, the keiji command should be available.


## USAGE

### STEP 1: Initializing a workspace

#### command

```
keiji init
```
 
#### output

```
2024/08/10 18:43:43 open /Users/andrewodiit/keiji/settings.conf: no such file or directory
2024/08/10 18:43:43 Initializing work space...
2024/08/10 18:43:53 service scheduler not found
2024/08/10 18:43:53 service bus not found
2024/08/10 18:43:53 installing scheduler
2024/08/10 18:43:58 ok
2024/08/10 18:43:58 installing bus
2024/08/10 18:44:01 ok
```

after `initialization` you should have the following folder structures in your system 

`$HOME/keiji`

```
keiji
├── go.mod
├── go.sum
├── settings.conf
└── tasks
```
- `go.mod & go.sum` - all workspace dependencies used by your tasks.

- `settings.conf` - database , timezone & log rotation settings.

- `tasks/` - task source code is stored here after creation.


`$HOME/.keiji`

```
.keiji
├── db
│   └── keiji.db
└── logs
    └── services
        ├── bus
        │   └── bus.log
        ├── repo
        │   └── repo.log
        └── scheduler
            └── scheduler.log

```

`db` - `default sqllite3 storage` created during the initialization process. It can be changed to postgresql in `settings.conf`

`logs` - contains log files for `services`. Once tasks are created, a folder for `tasks` will appear here.

NB: Folder structure is required for clear separation of concerns. This is particularly important once log rotation is enabled.

**WARNING ⚠️ : DO NOT MODIFY THE STRUCTURE OF THESE DIRECTORIES**


### STEP 2: create a task

####  command

```
keiji task --create --name=ping_google --desc="pings google"
```

####  output

```
2024/08/10 19:27:09 creating task ping_google
2024/08/10 19:27:09 ok

```
####  result

```
keiji
└── tasks
    └── ping_google
        ├── .env
        ├── function.go
        ├── main.go
        └── schedule.go
```

`function.go`

- Tasks logic goes here e.g
 
 ```
 func Function() error {
	/*
		please put the logic you wish to execute in this function.
	*/
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get("https://www.google.com/")
	fmt.Println("status: ", resp.StatusCode)
	if err != nil {
		return err
	}

	return nil
}
```

`schedule.go`
- Tasks schedule goes here e.g

```
package main

import (
	"log"

	"github.com/aodr3w/keiji-core/tasks"
)

func Schedule() error {
	/*
		    DEFINE FUNCTION SCHEDULE HERE
			example;
			 core.NewSchedule().Run().Every(10).Seconds().Build()
			)
	*/
	log.Println("scheduling function...")
	return tasks.NewSchedule().Run().Every(10).Seconds().Build()
}
```
- ☝🏾 A task can can be schedule to run on a fixed interval defined as `Seconds, Minutes or Hours` (as shown in code above)

- Alternatively, it can be scheduled to run on a specific day at a specific time e.g

```
return tasks.NewSchedule().On().Friday().At("10:00PM").Build()
```
- The time value can be either `12hour or 24hour format`.

### STEP 3: build & save a task

#### command

```
keiji task --name=ping_google --build
```

####  output

```
2024/08/10 19:46:30 task found , building...
2024/08/10 19:46:32 scheduling function...
2024/08/10 19:46:32 task saved
sourcePath:  /Users/andrewodiit/keiji/tasks/ping_google
2024/08/10 19:46:32 executable path /Users/andrewodiit/.keiji/exec/tasks/ping_google.bin
2024/08/10 19:46:32 source path /Users/andrewodiit/keiji/tasks/ping_google
2024/08/10 19:46:35 executable created

2024/08/10 19:46:35 /Users/andrewodiit/go/pkg/mod/github.com/aodr3w/keiji-core@v0.2.2/db/repo.go:184 record not found
[0.219ms] [rows:0] SELECT * FROM `task_models` WHERE name = "ping_google" AND `task_models`.`deleted_at` IS NULL ORDER BY `task_models`.`id` LIMIT 1
2024/08/10 19:46:35 task saved
```

### STEP 4: check task details

#### command

```
keiji task --get --name=ping_google
```

#### output

```
TaskModel:
ID: 1
TaskId: ac965b18-b229-4360-a294-72913115bb45
Name: ping_google
Description: pings google
Schedule: day:Friday,time:10:00PM
LastExecutionTime: N/A
NextExecutionTime: N/A
LogPath: /Users/andrewodiit/.keiji/logs/tasks/ping_google/ping_google.log
Slug: ping_google
Type: DayTime
Executable: /Users/andrewodiit/.keiji/exec/tasks/ping_google.bin
IsRunning: false
IsQueued: false
IsError: false
IsDisabled: false
ErrorTxt: 
```

### STEP 5: start system

#### start system command

```
keiji system --start
```
#### output

```
2024/08/10 19:51:44 /Users/andrewodiit/.keiji/exec/services/bus.pid: PID file not found
2024/08/10 19:51:44 service started with pid 72628
2024/08/10 19:51:44 /Users/andrewodiit/.keiji/exec/services/scheduler.pid: open /Users/andrewodiit/.keiji/exec/services/scheduler.pid: no such file or directory
2024/08/10 19:51:44 service started with pid 72630
```

#### check system status command

```
keiji system --status
```
#### output

```
======== SERVICES ========
NAME               STATUS            
bus                ONLINE            
scheduler          ONLINE            
======== SERVICES ========
```


### STEP 6: view logs

#### bus logs
```
keiji system --logs --bus 
```
#### output
```
2024/08/10 19:51:44 waiting for termination signal
2024/08/10 19:51:44 Server started at :8005
2024/08/10 19:51:44 Server started at :8006
```

#### scheduler logs

```
keiji system --logs --scheduler
```
#### output

```
2024/08/10 19:51:45 waiting for termination signal
time=2024-08-10T19:51:45.330+03:00 level=INFO msg="starting tcp-bus listener"
time=2024-08-10T19:51:45.330+03:00 level=INFO msg="running start function"
```

#### task logs

```
keiji task --logs --name=ping_google
```

#### output

```
time=2024-08-11T11:18:54.687+03:00 level=INFO msg="Task Next Execution Time: 2024-08-16 22:00:00 +0300 +0300"
```


### STEP 7: modify task functionality

- In the example below, i added a `fmt.Println("Pinging Google....")` statement in function.go
and changed the `schedule.go` to run every 10 seconds.

#### updated source code

```
package main

import (
	"fmt"
	"net/http"
	"time"
)

func Function() error {
	/*
		please put the logic you wish to execute in this function.
	*/
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	fmt.Println("Pinging Google....")
	resp, err := client.Get("https://www.google.com/")
	fmt.Println("status: ", resp.StatusCode)
	if err != nil {
		return err
	}

	return nil
}

```

#### updated schedule

```
return tasks.NewSchedule().Run().Every(10).Seconds().Build()
```

- rebuild & restart task 

```
keiji task --name=ping_google --build --restart
```

```
2024/08/11 13:31:39 task found , building...
2024/08/11 13:31:42 scheduling function...
2024/08/11 13:31:42 task saved
sourcePath:  /Users/andrewodiit/keiji/tasks/ping_google
2024/08/11 13:31:42 executable path /Users/andrewodiit/.keiji/exec/tasks/ping_google.bin
2024/08/11 13:31:42 source path /Users/andrewodiit/keiji/tasks/ping_google
2024/08/11 13:31:44 executable created
2024/08/11 13:31:44 task saved

2024/08/11 13:31:44 restarting task ping_google
```

#### check task details

```
keiji task --get --name=ping_google
```

```
TaskModel:
ID: 1
TaskId: ac965b18-b229-4360-a294-72913115bb45
Name: ping_google
Description: pings google
Schedule: units:seconds,interval:10
LastExecutionTime: N/A
NextExecutionTime: N/A
LogPath: /Users/andrewodiit/.keiji/logs/tasks/ping_google/ping_google.log
Slug: ping_google
Type: HMS
Executable: /Users/andrewodiit/.keiji/exec/tasks/ping_google.bin
IsRunning: true
IsQueued: false
IsError: false
IsDisabled: false
ErrorTxt: 
```

- ☝🏾 task type has changed from `DayTime` to `HMS` because the schedule has been changed.

```
time=2024-08-11T13:31:44.532+03:00 level=INFO msg="task terminated, exiting..."
time=2024-08-11T13:31:45.535+03:00 level=INFO msg="task ping_google interval: 10"
time=2024-08-11T13:31:57.164+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:31:57.164+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:32:06.488+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:32:06.488+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:32:16.414+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:32:16.414+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:32:26.453+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:32:26.454+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:32:36.487+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:32:36.487+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:32:46.396+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:32:46.397+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:32:56.337+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:32:56.338+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:33:06.392+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:33:06.392+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:33:16.524+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:33:16.525+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:33:26.460+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:33:26.460+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:33:36.491+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:33:36.491+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:33:46.424+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:33:46.424+03:00 level=INFO msg="status:  200"
time=2024-08-11T13:33:56.460+03:00 level=INFO msg="Pinging Google...."
time=2024-08-11T13:33:56.460+03:00 level=INFO msg="status:  200"
```

- ☝🏾 task logs clearly indicate that the task was restarted successfully, and our modifications
have taken full effect

### STEP 8: disable a task


```
keiji task --disable --name=ping_google
```


```
time=2024-08-11T14:20:07.119+03:00 level=INFO msg="status:  200"
time=2024-08-11T14:20:10.828+03:00 level=INFO msg="task disabled, exiting..."
```

☝🏾 **disable signal received by running task**

```
====================================================================================================
TaskModel:
ID: 1
TaskId: ac965b18-b229-4360-a294-72913115bb45
Name: ping_google
Description: pings google
Schedule: units:seconds,interval:10
LastExecutionTime: N/A
NextExecutionTime: N/A
LogPath: /Users/andrewodiit/.keiji/logs/tasks/ping_google/ping_google.log
Slug: ping_google
Type: HMS
Executable: /Users/andrewodiit/.keiji/exec/tasks/ping_google.bin
IsRunning: false
IsQueued: false
IsError: false
IsDisabled: true
ErrorTxt: 

====================================================================================================
```

☝🏾 task record in database is marked as disabled so the scheduler will not attempt to pick it up. it can be enabled using the command `keiji task --enable --name=ping_google`, which makes the task runnable.

### STEP 9: enable task

```
keiji task --enable --name=ping_google
```
### STEP 10: delete a task

```
keiji task --delete --name=ping_google
```

### STEP 11: stop system

```
keiji system --stop
```

### STEP 12: uninstall system

```
keiji system --uninstall
```

## Q/A
**How do i change my workspace settings ?**

Open settings.conf , you can modify one of 4 settings:

- `DB_URL` - must be one of 2 values, `default` (sqllite3) or `a valid postgresql URL`.

- `TIME_ZONE` - must be a valid `timezone string` in the format `Continent/City`.

- `ROTATE_LOGS` - must be an integer either `1 (true)` or `0 (False)`.

- `LOG_MAX_SIZE` - must be a non negative integer.


**Can i use a database other than `sqllite3` and `postgresql` ?**

- keiji only supports `sqllite3` and `postgresql`, you would need to fork the repository and modify it as you wish.
