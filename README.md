# KEIJI - 計時 (time keeping)

## MOTIVIATION

- Develop an easy to use `task scheduling system` in go.


## SYSTEM OVERVIEW
![Keiji Scheduling System Overview](images/KEIJI-SCHEDULING-SYSTEM-OVERVIEW.png)


## INSTALLATION

```
go install github.com/aodr3w/keiji@latest
```

## USAGE

## STEP 1: Initializing a workspace

 - `keiji init`
 
```
workspace structure
```


## STEP 2: create tasks

```
keiji task --create --name=ping_google --desc="pings google"
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

```
keiji task --name=ping_google --build
```


## STEP 4: check task details

```
keiji task --get --name=ping_google
```


## STEP 5: start system

```
keiji system --start
```

```
keiji system --status
```

## STEP 6: view logs

```
keiji system --logs --bus 
```

```
keiji system --logs --scheduler --vim
```

```
keiji task --logs --name=ping_google --nano
```




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


