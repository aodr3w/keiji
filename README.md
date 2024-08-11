# KEIJI - 計時 (time keeping)

## MOTIVIATION

- Develop a concurrent task scheduling system in go.


## SYSTEM OVERVIEW
![Keiji Scheduling System Overview](images/KEIJI-SCHEDULING-SYSTEM-OVERVIEW.png)

### Description

```
describe the system overview
```

## INSTALLATION

```
go install github.com/aodr3w/keiji@latest
```

After install, `keiji command should be available`


## USAGE

## STEP 1: Initializing a workspace

**command**

```
keiji init
```
 
**output**

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


## STEP 2: create tasks

**command**

```
keiji task --create --name=ping_google --desc="pings google"
```

**output**

```
2024/08/10 19:27:09 creating task ping_google
2024/08/10 19:27:09 ok

```
**result**

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
- A task can can be schedule to run on a fixed interval defined `Seconds, Minutes or Hours` e.g

```
return tasks.NewSchedule().Run().Every(10).Seconds().Build()
```

- Alternatively, it can be scheduled to run on a specific day at a specific time e.g

```
return tasks.NewSchedule().On().Friday().At("10:00PM").Build()
```
- The time value can be either `12hour or 24hour format`.

## STEP 3: build & save task

**command**

```
keiji task --name=ping_google --build
```

**output**

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

## STEP 4: check task details

**command**

```
keiji task --get --name=ping_google
```

**output**

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

## STEP 5: start system

**start system command**

```
keiji system --start
```
**output**

```
2024/08/10 19:51:44 /Users/andrewodiit/.keiji/exec/services/bus.pid: PID file not found
2024/08/10 19:51:44 service started with pid 72628
2024/08/10 19:51:44 /Users/andrewodiit/.keiji/exec/services/scheduler.pid: open /Users/andrewodiit/.keiji/exec/services/scheduler.pid: no such file or directory
2024/08/10 19:51:44 service started with pid 72630
```

**check system status command**

```
keiji system --status
```
**output**

```
======== SERVICES ========
NAME               STATUS            
bus                ONLINE            
scheduler          ONLINE            
======== SERVICES ========
```


## STEP 6: view logs

**bus logs**
```
keiji system --logs --bus 
```
**output**
```
2024/08/10 19:51:44 waiting for termination signal
2024/08/10 19:51:44 Server started at :8005
2024/08/10 19:51:44 Server started at :8006
```

**scheduler logs**

```
keiji system --logs --scheduler
```
**output**

```
2024/08/10 19:51:45 waiting for termination signal
time=2024-08-10T19:51:45.330+03:00 level=INFO msg="starting tcp-bus listener"
time=2024-08-10T19:51:45.330+03:00 level=INFO msg="running start function"
```

**task logs**

```
keiji task --logs --name=ping_google
```

**output**

```
time=2024-08-11T11:18:54.687+03:00 level=INFO msg="Task Next Execution Time: 2024-08-16 22:00:00 +0300 +0300"
````



## STEP 7: modify task functionality

**update source code**

**update schedule**

**check task details**


## STEP 8: disable a task

```
keiji task --disable --name=ping_google`
```

## STEP 9: delete a task

```
keiji task --delete --name=ping_google
```


## STEP 10: changing database

```
modify settings.conf in workspace && restart system
```


## STEP 9: enable/disable log rotation

```
modify settings.conf in workspace && restart system
```

## STEP 10: stop system

```
keiji system --stop
```

## STEP 12: uninstall system

```
keiji system --uninstall
```


