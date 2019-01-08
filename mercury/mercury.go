package main

import (
  "bufio"
  "fmt"
  "os"
  "strings"
  "regexp"
  "sort"
  "strconv"
  "time"
  "math/rand"

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
  help                 string
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
  receiver_count             int
  sender_count               int
  edge_count                 int

  mercury_log_name           string
  mercury_log_file         * os.File
  first_nonwhitespace_rgx  * regexp.Regexp
  line_rgx                 * regexp.Regexp
}






/*=====================================================================
  Helper Functions
======================================================================*/

// first -- make it ignore comment lines
/*
func mlog 
func Print_log ( format string, args ...interface{} ) {
  ts := Timestamp()
  fp ( os.Stdout, ts + " : %s : " + format + "\n", args... )
}
*/


// To create a command, call this function to get it started
// with just the name and the executable function, and then 
// repeatedly call command.add_args() to add all the arguments.
func new_command ( name string, fn command_fn, help string ) ( * command ) {
  var c * command
  c = & command { name : name,
                  fn   : fn,
                  help : help }
  return c
}





// commands_by_name implements the sort interface
// for [] command , sorting by ... wait for it ... names!
type Commands_by_name [] * command

// 'ca' is a command array
func ( ca Commands_by_name ) Len  ( ) int           { return len ( ca ) }
func ( ca Commands_by_name ) Swap ( i, j int )      { ca[i], ca[j] = ca[j], ca[i] }
func ( ca Commands_by_name ) Less ( i, j int ) bool { return ca[i].name < ca[j].name }




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

  first_nonwhitespace := context.first_nonwhitespace_rgx.FindString ( line )
  if first_nonwhitespace == "" {
    // If the line is just empty, don't even echo it to the log.
    // The user just hit 'enter'.
    return
  }

  // Except for emppty lines, echo everything else, 
  // including comments, to the log.
  fmt.Fprintf ( context.mercury_log_file, "%s\n", line )

  if first_nonwhitespace == "#" {
    // This is a comment.
    return
  }

  /*----------------------------------------
    Clean up the line
  -----------------------------------------*/
  line = strings.Replace ( line, "\n", "", -1 )
  line = context.line_rgx.ReplaceAllString ( line, " " )
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





func create_network ( context * Context ) {

  context.router_path   = context.dispatch_install_root + "/sbin/qdrouterd"
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








// Is the given string one of this command's args?
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





func add_command ( context * Context, cmd * command ) {
  context.commands = append ( context.commands, cmd )
}





/*=====================================================================
  Command Functions
======================================================================*/


func add_routers ( context * Context, am argmap ) {

  routers_so_far := context.network.N_routers()

  if routers_so_far >= 26 {
    fp ( os.Stdout, 
         "%c error: You can't have any more routers. You have %d already.\n", 
         mercury, 
         routers_so_far )
    return
  }

  count, _ := strconv.Atoi ( am["count"].value )

  if count + routers_so_far >= 26 {
    count = 26 - routers_so_far
  }

  for i:= 0; i < count; i ++ {
    router_name := fmt.Sprintf ( "%c", 'A' + byte(i) + byte(routers_so_far) )

    context.network.Add_router ( router_name )

    if context.verbose {
      fp ( os.Stderr, "%c info: made router %s. Network now has %d routers.\n", mercury, router_name, context.network.N_routers() )
    }
  }
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





func add_edges ( context * Context, am argmap ) {

  count_str  := am["count"].value
  router_arg := am["router"].value
  if count_str == "" || router_arg == "" {
    print_usage ( context, get_command (context, "add_edges" ) )
    return
  }

  var target_router string
  count, err := strconv.Atoi ( count_str )
  if err != nil {
    fp ( os.Stdout, "%c add_edges: error on count: |%s|\n", mercury, err.Error() )
    return
  }

  var edge_name string
  for i := 0; i < count; i ++ {
    if router_arg == "RANDOM" {
      interior_router_count := context.network.How_many_interior_routers()
      random_index := rand.Intn ( interior_router_count )
      target_router = context.network.Get_nth_interior_router_name ( random_index )
    } else {
      target_router = router_arg
    }

    edge_name = fmt.Sprintf ( "edge_%04d", context.edge_count )
    context.network.Add_edge ( edge_name )
    context.network.Connect_router ( edge_name, target_router )
    if context.verbose {
      fp ( os.Stdout, 
           "    %c info: add_edges: added |%s| to |%s|\n", 
           mercury, edge_name, target_router )
    }
    context.edge_count ++
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





func help ( context * Context, am argmap ) {
  for _, cmd := range context.commands  {
    fp ( os.Stdout, "---------------------\n" )
    fp ( os.Stdout, "%s\n",   cmd.name )
    fp ( os.Stdout, "---------------------\n" )
    fp ( os.Stdout, "  %s\n", cmd.help )
    for _, ad := range cmd.arg_descriptors {
      fp ( os.Stdout, "\n" )
      fp ( os.Stdout, "  arg         : %s\n", ad.name )
      fp ( os.Stdout, "  type        : %s\n", ad.data_type )
      fp ( os.Stdout, "  explanation : %s\n", ad.explanation )
      fp ( os.Stdout, "  default     : %s\n", ad.default_value )
    }
    fp ( os.Stdout, "\n\n" )
  }
}





func network ( context * Context, am argmap ) {
  create_network ( context )
}





func sleep ( context * Context, am argmap ) {
  how_long, _ := strconv.Atoi ( am["duration"].value )

  if context.verbose {
    fp ( os.Stdout, "%c: Sleeping for %d seconds.\n", mercury, how_long )
  }

  time.Sleep ( time.Second * time.Duration ( how_long ) )
}





func connect ( context * Context, am argmap ) {
  pitcher := am["from"].value
  catcher := am["to"].value

  context.network.Connect_router ( pitcher, catcher )

  if context.verbose {
    fp ( os.Stdout, "    %c info: connected %s to %s\n", mercury, pitcher, catcher )
  }
}





func add_receivers ( context * Context, am argmap ) {
  router_name           := am["router"].value
  count, _              := strconv.Atoi ( am["count"].value )
  n_messages, _         := strconv.Atoi ( am["n_messages"].value )
  max_message_length, _ := strconv.Atoi ( am["max_message_length"].value )
  address               := am["address"].value
  fp ( os.Stderr, " recv: addr: |%s| explicit: %t\n", address, am["address"].explicit )

  var receiver_name string
  var i int
  for i = 0; i < count; i ++ {
    receiver_name = fmt.Sprintf ( "receiver_%04d", i + context.receiver_count )
    context.network.Add_receiver ( receiver_name, n_messages, max_message_length, router_name, address )

    if context.verbose {
      fp ( os.Stdout, "  %c info: created receiver %s attached to router %s.\n", mercury, receiver_name, router_name )
    }
  }
  context.receiver_count += i
}





func add_senders ( context * Context, am argmap ) {
  router_name           := am["router"].value
  count, _              := strconv.Atoi ( am["count"].value )
  n_messages, _         := strconv.Atoi ( am["n_messages"].value )
  max_message_length, _ := strconv.Atoi ( am["max_message_length"].value )
  address               := am["address"].value
  fp ( os.Stderr, " send: addr: |%s| explicit: %t\n", address, am["address"].explicit )

  var sender_name string
  var i int
  for i = 0; i < count; i ++ {
    sender_name = fmt.Sprintf ( "sender_%04d", i + context.sender_count )
    context.network.Add_sender ( sender_name, n_messages, max_message_length, router_name, address )

    if context.verbose {
      fp ( os.Stdout, "  %c info: created sender %s attached to router %s.\n", mercury, sender_name, router_name )
    }
  }
  context.sender_count += i
}





func report ( context * Context, am argmap ) {
  fp ( os.Stdout, "report --------------------------------------------\n" )
  fp ( os.Stdout, "-----------------------------------------end report\n" )
}





func run ( context  * Context, am argmap ) {
  context.network.Init ( )
  context.network.Run  ( )

  if context.verbose {
    fp ( os.Stdout, "    %c info: network is running.\n", mercury )
  }
}





func main() {
  rand.Seed ( int64 ( os.Getpid()) )
  
  var context Context
  init_context ( & context )

  context.mercury_log_name = "./mercury_" + time.Now().Format ( "2006_01_02_1504" ) + ".log"
  context.mercury_log_file, _ = os.Create ( context.mercury_log_name )
  defer context.mercury_log_file.Close()

  context.line_rgx                = regexp.MustCompile(`\s+`)
  context.first_nonwhitespace_rgx = regexp.MustCompile(`\S`)

  /*-------------------------------------------
    Make commands.
  -------------------------------------------*/
  c := new_command ( "add_routers", 
                     add_routers, 
                     "Add one or more internal routers to the network, up to 26.\n  Names will be A, B, ... Z." )
  c.add_arg ( "count", "int",    "how many routers to create", "1" )
  add_command ( & context, c )


  c = new_command ( "quit", 
                    quit, 
                    "Shut down the network and halt Mercury." )
  add_command ( & context, c )


  c = new_command ( "echo", 
                    echo, 
                    "Echo the given string." )
  c.add_arg ( "message", "string", "The message for echo to echo.", "Hello, Mercury!" )
  add_command ( & context, c )


  c = new_command ( "start_actions", 
                    start_actions, 
                    "Start running all actions that have already been registered." )
  add_command ( & context, c )


  c = new_command ( "verbose", 
                    verbose, 
                    "Tell Mercury to turn verbose mode on or off." )
  c.add_arg ( "on",  "flag", "Turn verbosity on. That is, invite, nay *command*\n  Mercury to explain every little thing. i.e. every detail of its operation.", "" )
  c.add_arg ( "off", "flag", "Turn verbosity off.", "" )
  add_command ( & context, c )


  c = new_command ( "set_paths", 
                    set_paths, 
                    "Define the paths that Mercury needs to run."  )
  c.add_arg ( "dispatch", "string", "The path to the dispatch install directory.", "none" )
  c.add_arg ( "proton",   "string", "The path to the proton install directory.",   "none" )
  c.add_arg ( "mercury",  "string", "The path to the mercury directory.",          "none" )
  add_command ( & context, c )


  c = new_command ( "network", 
                    network, 
                    "Create the initial network. Do this after paths are defined,\n  and before you start adding routers and clients." )
  add_command ( & context, c )


  c = new_command ( "sleep", 
                    sleep, 
                    "Tell the main thread to sleep the given number of seconds.\n  Repeating actions will continue running." )
  c.add_arg ( "duration", "string", "How long to sleep.", "10" )
  add_command ( & context, c )


  c = new_command ( "connect", 
                    connect, 
                    "Connect the 'from' router to the 'to' router." )
  c.add_arg ( "from", "string", "The router that will initiate the connection.", "" )
  c.add_arg ( "to",   "string", "The router that will accept the connection.",   "" )
  add_command ( & context, c )


  c = new_command ( "run", 
                    run, 
                    "Start the network running. Before doing this, you should add\n  all the internal routers and connect them as desired." )
  add_command ( & context, c )


  c = new_command ( "help", 
                    help, 
                    "Print all command and argument help." )
  add_command ( & context, c )


  c = new_command ( "add_receivers",
                    add_receivers,
                    "Add receivers to a given router." )
  c.add_arg ( "count", "string", "How many senders to create.", "1" )
  c.add_arg ( "router", "string", "The router that the receivers will attach to.", "" )
  c.add_arg ( "n_messages", "string", "How many messages to receive before quitting.", "100" )
  c.add_arg ( "max_message_length", "string", "Average length of messages will be about half of this.", "100" )
  c.add_arg ( "address", "string", "Address to receive from.", "my_address" )
  add_command ( & context, c )


  c = new_command ( "add_senders",
                    add_senders,
                    "Add senders to a given router." )
  c.add_arg ( "count", "string", "How many senders to create.", "1" )
  c.add_arg ( "router", "string", "The router that the senders will attach to.", "" )
  c.add_arg ( "n_messages", "string", "How many messages to send.", "100" )
  c.add_arg ( "max_message_length", "string", "Average length of messages will be about half of this.", "100" )
  c.add_arg ( "address", "string", "Address to send to.", "my_address" )
  add_command ( & context, c )


  c = new_command ( "report",
                    report,
                    "Report on the status of all running processes." )
  add_command ( & context, c )


  c = new_command ( "add_edges",
                    add_edges,
                    "Add edge-routers to the given interior router." )
  c.add_arg ( "count",  "string", "How many edge routers to add.", "1" )
  c.add_arg ( "router", "string", "Which router to add them to.", "" )
  add_command ( & context, c )





  // Get the commands into alphabetical order.
  sort.Sort ( Commands_by_name ( context.commands ) )



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




