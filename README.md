# KEIJI - 計時 (time keeping)

## MOTIVIATION

- Develop an easy to use `task scheduling system` in go.


## SYSTEM OVERVIEW
![Keiji Scheduling System Overview](images/KEIJI-SCHEDULING-SYSTEM-OVERVIEW.png)


## INSTALLATION

`go install github.com/aodr3w/keiji@latest`

## USAGE

## STEP 1: Initializing a workspace

 - `keiji init`
 
```
workspace structure

```


## STEP 2: create tasks

```
# keiji task --create --name=ping_google --desc="pings google"

```

```
$WORKSPACE/
└── tasks/
    └── ping_google/
        ├── schedule.go
        ├── main.go
        └── function.go
```





## STEP 3: build & save task


## STEP 4: check task details


## STEP 5: start system


## STEP 6: view logs


## STEP 7: modify task


## STEP 8: disable a task

`keiji task --disable --name=<task_name>`

## STEP 9: delete a task
`keiji task --delete --name=<task_name>`

## STEP 10: stop system
`keiji system --stop`


## STEP 11: restart system
`keiji system --restart`


## STEP 12: uninstall system
`keiji system --uninstall`


