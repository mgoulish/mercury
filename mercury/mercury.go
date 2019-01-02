package main

import (
  "bufio"
  "fmt"
  "os"
  "strings"
  "regexp"
  "strconv"
  "time"

  "utils"
  rn "router_network"
)



var fp = fmt.Fprintf
var mercury = '\u263F'



// This is a description of an argument for a
// command. This is not what gets passed to the
// live instance of the running command.
type arg_descriptor struct {
  name, data_type, explanation, default_value string
  set bool
}


// This is the actual argument value that gets passed 
// to the running command. (Except in a map.)
type argval struct {
  value    string
  // Was it set explicitly, or is this just the default?
  explicit bool 
}


// This is the map of live values that gets passed
// to a running command function.
type argmap map [ string ] argval



// The function that actually gets called to
// execute an instance of a command.
type command_fn func ( * Context, argmap )


// The structure that describes a command 
// and its arguments.
type command struct {
  name                 string
  abbreviations   []   string
  arg_descriptors [] * arg_descriptor
  fn                   command_fn
}


// Actions to be executed in the future, and
// maybe repeatedly. All we need is the wait time
// before execution (which is also the cycle time
// for repeated action), and the command line for
// the command to be called.
type action struct {
  delay           int 
  repeat          bool
  command_line [] string
}


type Context struct {
  line_rx                  * regexp.Regexp
  commands              [] * command

  dispatch_install_root      string
  proton_install_root        string
  mercury_root               string
  result_path                string
  router_path                string
  config_path                string
  log_path                   string
  client_path                string

  verbose                    bool

  n_worker_threads           int

  network                  * rn.Router_Network
  network_running            bool
  function_argmap            argmap
  actions               [] * action
}






/*=====================================================================
  Helper Functions
======================================================================*/


// To create a command, call this function to get it started
// with just the name and the executable function, and then 
// repeatedly call command.add_args() to add all the arguments.
func new_command ( name string, fn command_fn ) ( * command ) {
  var c * command
  c = & command { name : name,
                  fn   : fn }
  return c
}





func new_action ( cycle_time int, repeat bool, command_line [] string ) ( * action ) {
  var a * action
  a = & action { delay        : cycle_time,
                 repeat       : repeat,
                 command_line : command_line }
  return a
}





func ( cmd * command ) add_arg ( name, data_type, explanation, default_value string ) {
  a := & arg_descriptor { name          : name,
                          data_type     : data_type,
                          explanation   : explanation,
                          default_value : default_value }
  cmd.arg_descriptors = append ( cmd.arg_descriptors, a )
}





func read_file ( context * Context, file_name string ) {

  file, err := os.Open ( file_name )
  if err != nil {
    panic ( err )
  }
  defer file.Close()

  scanner := bufio.NewScanner ( file )
  for scanner.Scan() {
    process_line ( context, scanner.Text() )
  }

  if err := scanner.Err(); err != nil {
    panic ( err )
  }
}





func process_line ( context * Context, line string ) {

  /*----------------------------------------
    Clean up the line
  -----------------------------------------*/
  line = strings.Replace ( line, "\n", "", -1 )
  line = context.line_rx.ReplaceAllString ( line, " " )
  words := strings.Split ( line, " " )

  /*----------------------------------------
    The first word should be the name of 
    a function. Call it.
  -----------------------------------------*/
  call_command ( context, words )
}





func print_usage ( context * Context, cmd * command ) {
  fp ( os.Stderr, "Usage for command |%s|\n", cmd.name )
}





func get_command ( context * Context, target_name string ) ( * command ) {
  for _, cmd := range context.commands {
    if target_name == cmd.name {
      return cmd
    }
  }

  return nil
}





func make_default_args ( cmd * command ) ( argmap ) {
  am := make ( argmap, len(cmd.arg_descriptors) )
  for _, argd := range cmd.arg_descriptors {
    av := argval { value : argd.default_value, explicit : false }
    am [ argd.name ] = av
  }
  return am
}





/*=======================================================
  Make the arg-map that will be handed to the running
  instance of the given functions. All Args will get
  default values at first, but those values will then
  be over-written by whatever values the user supplied 
  on the command line.
=======================================================*/
func make_arg_map ( context * Context, cmd * command, command_line [] string ) ( argmap )  {

  am := make_default_args ( cmd )

  // Look at each arg on the command line.
  for i := 1; i < len(command_line); i ++ {
    arg_name := command_line [ i ]
    arg := cmd.get_arg ( arg_name )

    if arg == nil {
      // There is no such arg for this command.
      fp ( os.Stderr, "Bad arg: |%s|\n", arg_name )
      print_usage ( context, cmd )
      return nil
    } 

    // This is a valid arg for this command.
    if arg.data_type == "flag" {
      // Flags are special because they take no value.
      av := argval { value : "true", explicit : true }
      am [ arg_name ] = av
    } else {
      // This arg is a non-flag type and must take a value.
      if i == len(command_line) - 1 {
        fp ( os.Stderr, "mercury error: no value for argument |%s| in command |%s|\n", cmd.name, arg_name )
        return nil
      }
      av := argval { value : command_line [ i + 1 ], explicit : true }
      am [ arg_name ] = av
      // Advance the loop variable over the value we just consumed.
      i ++
    }
  }

  return am
}





func call_command ( context * Context, command_line [] string ) {

  if command_line[0] == "action" {
    create_action ( context, command_line )
    return
  } 

  cmd := get_command ( context, command_line[0] )
  if cmd == nil {
    fp ( os.Stderr, "mercury error: unknown command: |%s|\n", command_line[0] )
    return
  }

  am := make_arg_map ( context, cmd, command_line )

  cmd.fn ( context, am )
}





func call_command_repeatedly ( context * Context, command_line [] string, cycle_time int, repeat bool ) {
  for {
    time.Sleep ( time.Duration(cycle_time) * time.Second )
    call_command ( context, command_line )
    if ! repeat {
      break
    }
  }
}





func init_context ( context * Context ) {
  context.verbose                        = false
  context.network_running                = false
  context.n_worker_threads               = 4
}





func create_network (context * Context ) {

  context.result_path   = context.mercury_root + "/mercury/results"
  context.config_path   = context.mercury_root + "/mercury/config"
  context.log_path      = context.mercury_root + "/mercury/log"
  context.client_path   = context.mercury_root + "/clients/c_proactor_client"

  utils.Find_or_create_dir ( context.result_path )
  utils.Find_or_create_dir ( context.config_path )
  utils.Find_or_create_dir ( context.log_path )
  // Don't try to create the client path. That's an executable, not a directory.

  if context.verbose {
    fp ( os.Stdout, "  create_network: result_path == |%s|\n", context.result_path )
    fp ( os.Stdout, "  create_network: config_path == |%s|\n", context.config_path )
    fp ( os.Stdout, "  create_network: log_path    == |%s|\n", context.log_path )
    fp ( os.Stdout, "  create_network: client_path == |%s|\n", context.client_path )
    fp ( os.Stdout, "\n" )
  }

  context.network = rn.New_Router_Network ( "mercury_router_network",
                                            context.n_worker_threads,
                                            context.result_path,
                                            context.router_path,
                                            context.config_path,
                                            context.log_path,
                                            context.client_path,
                                            context.dispatch_install_root,
                                            context.proton_install_root,
                                            context.verbose,
                                            0 )
}








// Is the given string one of my args?
func ( cmd * command ) get_arg ( possible_arg_name string ) ( * arg_descriptor ) {
  for _, arg := range cmd.arg_descriptors {
    if arg.name == possible_arg_name {
      return arg
    }
  }
  return nil
}





func create_action ( context * Context, command_line [] string ) {
  // The first arg and val must be for 'cycle_time'.
  arg_name := "cycle_time"
  if command_line[1] != arg_name {
    fp ( os.Stderr, "%c error: action must have first arg == %s\n", mercury, arg_name )
    return
  }
  cycle_time, _ := strconv.Atoi ( command_line[2] )

  a := new_action ( cycle_time, true, command_line[3:] )
  context.actions = append ( context.actions, a )
}





/*=====================================================================
  Command Functions
======================================================================*/


func router ( context * Context, am argmap ) {
  fp ( os.Stderr, "router fn called!  with %d args\n", len(am) )
}





func quit ( context * Context, am argmap ) {
  if context.network_running {
    context.network.Halt ( )
  }
  os.Exit ( 0 )
}





func echo ( context * Context, am argmap ) {
  fp ( os.Stderr, "echo fn called!  with %d args\n", len(am) )
  if len(am) > 0 {
    fp ( os.Stderr, "%s\n", am["message"].value )
  }
}





func start_actions ( context * Context, am argmap ) {
  for _, a := range context.actions {
    fp ( os.Stderr, " start_actions: delay is %d comline is |%V|\n", a.delay, a.command_line )
    go call_command_repeatedly ( context, a.command_line, a.delay, a.repeat ) 
  }
}





func verbose ( context * Context, am argmap ) {
  if a := am["on"]; a.value == "true" {
    context.verbose = true
    fp ( os.Stdout, "%c: verbose on.\n", mercury )
    return
  }

  if a := am["off"]; a.value == "true" {
    context.verbose = false
  }
}





func set_paths ( context * Context, am argmap ) {
  context.dispatch_install_root = am["dispatch"].value
  context.proton_install_root   = am["proton"  ].value
  context.mercury_root          = am["mercury" ].value

  if context.verbose {
    fp ( os.Stderr, " set_paths: dispatch |%s|  proton |%s|    mercury |%s|\n", context.dispatch_install_root, context.proton_install_root, context.mercury_root )
  }
}





func main() {
  
  var context Context
  init_context   ( & context )

  context.line_rx   = regexp.MustCompile(`\s+`)

  /*-------------------------------------------
    Make commands.
  -------------------------------------------*/
  c := new_command ( "router", router )
  c.add_arg ( "count", "int",    "how many routers to create", "1" )
  context.commands = append ( context.commands, c )

  c = new_command ( "quit", quit )
  context.commands = append ( context.commands, c )

  c = new_command ( "echo", echo )
  c.add_arg ( "message", "string", "The message for echo to echo.", "Hello, Mercury!" )
  context.commands = append ( context.commands, c )

  c = new_command ( "start_actions", start_actions )
  context.commands = append ( context.commands, c )

  c = new_command ( "verbose", verbose )
  c.add_arg ( "on",  "flag", "Turn verbosity on. That is, invite, nay *command* Mercury to explain every little thing. i.e. every detail of its operation.", "" )
  c.add_arg ( "off", "flag", "Turn verbosity off.", "" )
  context.commands = append ( context.commands, c )

  c = new_command ( "set_paths", set_paths )
  c.add_arg ( "dispatch", "string", "The path to the dispatch install directory.", "none" )
  c.add_arg ( "proton",   "string", "The path to the proton install directory.",   "none" )
  c.add_arg ( "mercury",  "string", "The path to the mercury directory.",          "none" )
  context.commands = append ( context.commands, c )

  

  /*--------------------------------------------
    Process files named on command line.
  --------------------------------------------*/
  for i := 1; i < len(os.Args); i ++ {
    read_file ( & context, os.Args[i] )
  }

  /*--------------------------------------------
    Prompt for and read lines of input until
    the user tells us to quit.
  --------------------------------------------*/
  reader := bufio.NewReader ( os.Stdin )
  for {
    fp ( os.Stdout, "%c ", mercury )
    line, _ := reader.ReadString ( '\n' )

    process_line ( & context, line )
  }
}





