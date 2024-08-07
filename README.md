# KEIJI - 計時 (TIME KEEPING)
keiji is command line based go application that allows for creation and scheduling of golang functions.

# installation
- go install https://github.com/aodr3w/keiji@latest

# USAGE
# step 1: initialize a workspace
inorder to use keiji you need to first initialize a workspace , which is a folder in your home directory where your database settings and function source code will go.
 - keiji init

# step 2: create a task
to create task run the following command
- keiji task --create --name=<task_name> --desc=<task_description> 
this command will create a task called <task_name> in your workspace
for examples on how to create and schedule tasks, check the examples folder