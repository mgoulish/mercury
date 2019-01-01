package main

import (
  "bufio"
  "fmt"
  "os"
  "strings"
  "regexp"

  "utils"
  rn "router_network"
)



var fp = fmt.Fprintf



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


type Context struct {
  line_rx                      * regexp.Regexp
  commands                  [] * command

  dispatch_install_root          string
  proton_install_root            string
  mercury_root                   string
  result_path                    string
  router_path                    string
  config_path                    string
  log_path                       string
  client_path                    string

  verbose                        bool

  n_worker_threads               int
  resource_measurement_frequency int

  network                      * rn.Router_Network
  network_running                bool
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
    fp ( os.Stderr, "make_default_args argd.name == |%s|\n", argd.name )
    am [ argd.name ] = av
  }
  return am
}




func call_command ( context * Context, command_line [] string ) {

  cmd := get_command ( context, command_line[0] )
  if cmd == nil {
    fp ( os.Stderr, "mercury error: unknown command: |%s|\n", command_line[0] )
    return
  }

  // We found the command, now run it.
  am := make_default_args ( cmd )

  // Look at each arg on the command line.
  for i := 1; i < len(command_line); i ++ {
    arg_name := command_line [ i ]
    arg := cmd.get_arg ( arg_name )
    if arg == nil {
      // There is no such arg for this command.
      fp ( os.Stderr, "Bad arg: |%s|\n", arg_name )
      print_usage ( context, cmd )
      return
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
        return
      }
      av := argval { value : command_line [ i + 1 ], explicit : true }
      am [ arg_name ] = av
      // Advance the loop variable over the value we just consumed.
      i ++
    }
  }
  cmd.fn ( context, am )
}





func init_context ( context * Context ) {
  context.verbose                        = false
  context.network_running                = false
  context.n_worker_threads               = 4
  context.resource_measurement_frequency = 0
}





func create_network (context * Context ) {

  context.result_path             = context.mercury_root + "/mercury/results"
  context.config_path             = context.mercury_root + "/mercury/config"
  context.log_path                = context.mercury_root + "/mercury/log"
  context.client_path             = context.mercury_root + "/clients/c_proactor_client"

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
                                            context.resource_measurement_frequency )
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





/*=====================================================================
  Command Functions
======================================================================*/


func router ( context * Context, argmap argmap ) {
  fp ( os.Stderr, "router fn called!  with %d args\n", len(argmap) )
}


func quit ( context * Context, argmap argmap ) {
  fp ( os.Stderr, "quit fn called!  with %d args\n", len(argmap) )

  if context.network_running {
    context.network.Halt ( )
  }

  os.Exit ( 0 )
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

  

  /*--------------------------------------------
    Process files named on command line.
  --------------------------------------------*/
  for i := 1; i < len(os.Args); i ++ {
    read_file ( & context, os.Args[i] )
  }

  /*--------------------------------------------
    Prompt for and read the next line of input.
  --------------------------------------------*/
  reader := bufio.NewReader ( os.Stdin )
  for {
    fp ( os.Stdout, "%c ", '\u263F' )
    line, _ := reader.ReadString ( '\n' )

    process_line ( & context, line )
  }
}





