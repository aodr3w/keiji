# KEIJI - 計時 (TIME KEEPING)
- keiji is the main entry point to the keiji task scheduling system. it provides a cobra-based cli that provides control commands for managing both services and tasks.

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


# create tasks


# start system


# change task schedule / source code / restart task


# view task logs


# disable a task


# delete a task


# view system logs


# stop system


# restart system


# uninstall system


