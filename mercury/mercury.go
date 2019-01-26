package main

import (
  "bufio"
  "fmt"
  "os"
  "regexp"
  "time"
  "math/rand"
  "errors"
  "strconv"
  "strings"

  "utils"
  rn "router_network"
  "lisp"
)



var fp = fmt.Fprintf
var mercury = '\u263F'


type command_fn func ( * Context, * lisp.List )


//---------------------------------------------------
// This is an argument that can be provided 
// by the user to a command.
//---------------------------------------------------
type arg_descriptor struct {
  name          string // The label of the arg, if it appears.
  unlabelable   bool   // Can it be unlabeled?
  data_type     string // How should the value be interpreted?
  default_value string // If user does not specify, it gets this value.
  help          string // Help string to show the user.

  // All fields below this point are filled in by a particular
  // instance of this command. I.e., when the user types it on
  // the command line.

  string_value  string // The string that the caller gave.
  int_value     int    // The integer value (if any) of the string.
  explicit      bool   // Did the caller explicity specify? (Or was this default?)
}



//---------------------------------------------------
// This is a command that the user can call from
// the command line.
// Each command has a maximum of two magic args that
// can be unlabeled. One an int, 
// and the other a string.
//---------------------------------------------------
type command struct {
  name                    string
  fn                      command_fn
  help                    string

  // Above this line are all the fields provided by the
  // caller when the command is first created. Below are
  // fields that are filled in later, as args are added.
  argmap                  map [ string ] * arg_descriptor
  unlabelable_int       * arg_descriptor
  unlabelable_string    * arg_descriptor
}





type Context struct {
  session_name               string

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
  receiver_count             int
  sender_count               int
  edge_count                 int

  mercury_log_name           string
  mercury_log_file         * os.File
  first_nonwhitespace_rgx  * regexp.Regexp
  line_rgx                 * regexp.Regexp
  
  first_version_name         string

  // This should probably change to commands, not fns.
  commands                   map [ string ] * command
}






/*=====================================================================
  Helper Functions
======================================================================*/



func m_info ( context * Context, format string, args ...interface{}) {
  if ! context.verbose {
    return
  }
  new_format := fmt.Sprintf ( "    %c info: %s\n", mercury, format )
  fp ( os.Stdout, new_format, args ... )
}





func m_error ( format string, args ...interface{}) {
  new_format := fmt.Sprintf ( "    %c error: %s", mercury, format + "\n" )
  fp ( os.Stdout, "\n------------------------------------------------\n" )
  fp ( os.Stdout, new_format, args ... )
  fp ( os.Stdout,   "------------------------------------------------\n\n" )
}





func ( context * Context ) add_command ( name      string, 
                                         fn        command_fn,
                                         help      string ) ( * command ) {
  cmd := & command { name : name,
                     fn   : fn,
                     help : help }
  cmd.argmap = make ( map [ string ] * arg_descriptor, 0 )

  context.commands [ name ] = cmd
  return cmd
}





func (c * command) add_arg ( name          string,
                             unlabelable   bool,
                             data_type     string,
                             default_value string,
                             help          string ) {
  a := & arg_descriptor { name          : name,
                          unlabelable   : unlabelable,
                          data_type     : data_type,
                          default_value : default_value,
                          help          : help }

  // If this arg is one of the two allowable unlabelables,
  // make sure that the spot it wants is not already filled.
  if unlabelable {
    if data_type == "string" {
      if c.unlabelable_string != nil {
        m_error ( "command |%s| already has an unlabelable string arg: |%s|", 
                  c.name, 
                  c.unlabelable_string.name )
        return
      }
      c.unlabelable_string = a
    } else if data_type == "int" {
      if c.unlabelable_int != nil {
        m_error ( "command |%s| already has an unlabelable int arg: |%s|",
                  c.name,
                  c.unlabelable_int.name )
        return
      }
      c.unlabelable_int = a
    } else {
      m_error ( "add_arg: unknown arg data type: |%s|", data_type )
      return
    }
  }

  c.argmap [ name ] = a
}





func parse_command_line ( context *      Context, 
                          cmd          * command, 
                          command_line * lisp.List ) {

  var err error

  // Fill in all args with their default values.
  // First the unlabelables
  if cmd.unlabelable_int != nil {
    cmd.unlabelable_int.int_value, _ = strconv.Atoi(cmd.unlabelable_int.default_value)
  }
  if cmd.unlabelable_string != nil {
    cmd.unlabelable_string.string_value = cmd.unlabelable_string.default_value
  }

  // And now all the labeled args.
  for _, arg := range cmd.argmap {
    if arg.data_type == "string " {
      arg.string_value = arg.default_value
    } else {
      arg.int_value, _ = strconv.Atoi ( arg.default_value )
    }
  }

  // Process the command line.
  // Get all the labeled args from the command line.
  // They and their values are removed as they are parsed.
  // If there are any unlabeled args, they will be left over after 
  // these are removed.
  for _, arg := range cmd.argmap {
    str_val := command_line.Get_value_and_remove ( arg.name )
    if str_val != "" {
      if arg.data_type == "string" {
        arg.string_value = str_val
        arg.explicit = true
      } else {
        arg.int_value, err = strconv.Atoi ( str_val )
        if err != nil {
          m_error ( "parse_command_line: error reading int from |%s|", str_val )
          return
        }
        arg.explicit = true
      }
    }
  }

  // If this command has unlabelable args, get them last.
  // Get the unlabelable string.
  if cmd.unlabelable_string != nil {
    ul_str, e2 := command_line.Get_string_cdr ( )
    if e2 == nil {
      // Fill in the value, so the command can get at it.
      cmd.unlabelable_string.string_value = ul_str
      cmd.unlabelable_string.explicit     = true
    }
  }

  // Get the unlabelable int.
  if cmd.unlabelable_int != nil {
    name := cmd.unlabelable_int.name
    ul_str, e2 := command_line.Get_int_cdr ( ) 
    if e2 == nil {
      var err error
      cmd.unlabelable_int.int_value, err = strconv.Atoi ( ul_str )
      if err != nil {
        m_error ( "parse_command_line: error reading value for |%s| : |%s|", 
                  name, 
                  err.Error() )
        cmd.unlabelable_int.explicit = false
      } else {
        cmd.unlabelable_int.explicit = true
      }
    }
  }
}





func call_command ( context * Context, command_line * lisp.List ) {
  cmd_name, err := command_line.Get_atom ( 0 ) 
  if err != nil {
    fp ( os.Stdout, "\n--------------------------------------\n" )
    fp ( os.Stdout, "    %c error: call_command: |%s|\n", mercury, err.Error() )
    fp ( os.Stdout, "--------------------------------------\n\n" )
    return
  }

  cmd := context.commands [ cmd_name ]
  if cmd == nil {
    fp ( os.Stdout, "\n--------------------------------------\n" )
    fp ( os.Stdout, "    %c error: no such command: |%s|\n", mercury, cmd_name )
    fp ( os.Stdout, "--------------------------------------\n\n" )
    return
  }

  cmd.fn ( context, command_line )
}





func init_context ( context * Context ) {
  context.verbose                        = false
  context.network_running                = false
  context.n_worker_threads               = 4
}





func create_network ( context * Context ) {

  context.router_path   = context.dispatch_install_root + "/sbin/qdrouterd"
  context.client_path   = context.mercury_root + "/clients/c_proactor_client"

  context.result_path   = context.session_name + "/" + "results"
  context.config_path   = context.session_name + "/" + "config"
  context.log_path      = context.session_name + "/" + "log"

  utils.Find_or_create_dir ( context.result_path )
  utils.Find_or_create_dir ( context.config_path )
  utils.Find_or_create_dir ( context.log_path )
  // Don't try to create the client path. That's an executable, not a directory.

  m_info ( context, "create_network: result_path : |%s|", context.result_path )
  m_info ( context, "create_network: config_path : |%s|", context.config_path )
  m_info ( context, "create_network: log_path    : |%s|", context.log_path )
  m_info ( context, "create_network: client_path : |%s|", context.client_path )
  m_info ( context, "create_network: result_path : |%s|", context.result_path )
  m_info ( context, "create_network: router_path : |%s|", context.router_path )

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





func get_next_interior_router_name ( context * Context ) ( string ) {
  routers_so_far := context.network.N_interior_routers()
  name := fmt.Sprintf ( "%c", 'A' + byte(routers_so_far) )
  return name
}





func this_is_an_interior_router_name ( context * Context, name string ) ( bool ) {
  routers_so_far := context.network.N_interior_routers()
  byte_array := []byte(name)
  router_name_byte := byte_array[0] 

  if 'A' <= router_name_byte &&  router_name_byte <= 'Z' {
    this_router_index := byte_array[0] - 'A'
    if int(this_router_index) < routers_so_far {
      return true
    }
  }

  return false
}





/*=====================================================================
  Command Functions
======================================================================*/


func verbose ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "verbose" ]
  parse_command_line ( context, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    context.verbose = true
    m_info ( context, "verbose: on" )
  } else if val == "off" {
    context.verbose = false
  } else {
    fp ( os.Stdout, " ERROR do something here.\n" )
  }
}





func edges ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "edges" ]
  parse_command_line ( context, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value
  count       := cmd.unlabelable_int.int_value

  version_name := context.first_version_name  // This will be the default.
  version_arg  := cmd.argmap [ "version" ]
  if version_arg.explicit {
    // The user entered a value.
    version_name = version_arg.string_value
  }

  // Make the edges.
  var edge_name string
  for i := 0; i < count; i ++ {
    context.edge_count ++
    edge_name = fmt.Sprintf ( "edge_%04d", context.edge_count )
    context.network.Add_edge ( edge_name, version_name )
    context.network.Connect_router ( edge_name, router_name )
    m_info ( context, 
             "edges: added edge %s with version %s to router %s", 
             edge_name, 
             version_name, 
             router_name )
  }
}





func paths ( context * Context, arglist * lisp.List ) {

  dispatch_path := arglist.Match_atom ( "dispatch" )
  proton_path   := arglist.Match_atom ( "proton" )
  mercury_path  := arglist.Match_atom ( "mercury" )

  trouble := 0

  if dispatch_path == "" {
    m_error ( "paths: dispatch path missing." )
    trouble ++
  }
  if _, err := os.Stat ( dispatch_path ); os.IsNotExist ( err ) {
    m_error ( "paths: dispatch path does not exist: |%s|.", dispatch_path )
    trouble ++
  }

  if proton_path == "" {
    m_error ( "paths: proton path missing." )
    trouble ++
  }
  if _, err := os.Stat ( mercury_path ); os.IsNotExist ( err ) {
    m_error ( "paths: mercury path does not exist: |%s|.", mercury_path )
    trouble ++
  }

  if mercury_path == "" {
    m_error ( "paths: mercury path missing." )
    trouble ++
  }
  if _, err := os.Stat ( proton_path ); os.IsNotExist ( err ) {
    m_error ( "paths: proton path does not exist: |%s|.", proton_path )
    trouble ++
  }

  if trouble > 0 {
    os.Exit ( 1 )
  }

  context.dispatch_install_root = dispatch_path
  context.proton_install_root   = proton_path
  context.mercury_root          = mercury_path

  m_info ( context, "paths: dispatch_path : |%s|", dispatch_path )
  m_info ( context, "paths: proton_path   : |%s|", proton_path   )
  m_info ( context, "paths: mercury_path  : |%s|", mercury_path  )

  // Now that paths are set, the network can be created.
  create_network ( context )
}





func senders ( context * Context, arglist * lisp.List ) {

  //-------------------------------------------------
  // Here we try to get all the args by name.
  //-------------------------------------------------
  n_messages   := arglist.Get_value_and_remove ( "n_messages" )
  max_message_length   := arglist.Get_value_and_remove ( "max_message_length" )
  router       := arglist.Get_value_and_remove ( "router" )
  edges        := arglist.Get_value_and_remove ( "edges" )  // The name of the router whose edges we want.
  throttle     := arglist.Get_value_and_remove ( "throttle" )
  address      := arglist.Get_value_and_remove ( "address" )
  start_at     := arglist.Get_value_and_remove ( "start_at" )
  count        := arglist.Get_value_and_remove ( "count" )

  //-------------------------------------------------
  // Now we convert all ints to ints.
  // Maybe do them as separate lists?
  //-------------------------------------------------
  n_messages_int, e1 := strconv.Atoi ( n_messages )
  if e1 != nil {
    m_error ( "senders: error reading n_messages: |%s|", e1.Error() )
    return
  }

  _, e2 := strconv.Atoi ( throttle )
  if e2 != nil {
    m_error ( "senders: error reading throttle: |%s|", e2.Error() )
    return
  }

  max_message_length_int, e3 := strconv.Atoi ( max_message_length )
  if e3 != nil {
    m_error ( "senders: error reading max_message_length: |%s|", e3.Error() )
    return
  }

  start_at_int := 1
  if start_at != "" {
    start_at_int, e2 = strconv.Atoi ( start_at )
    if e1 != nil {
      m_error ( "senders: error reading start_at: |%s|", e2.Error() )
      return
    }
  }

  //-------------------------------------------------
  // Finally, we get the unlabeled ones.
  // There can be only one unlabeled per type (string, int)
  //-------------------------------------------------

  //-------------------------------------------------
  // The unlabeled int.
  //-------------------------------------------------
  // If the count was unlabeled, it will still 
  // be in the arglist as an integer-string.
  if count == "" {
    count, _ = arglist.Get_int_cdr ( )
    if count == "" {
      m_error ( "senders: no count argument." )
      return
    }
  }
  // And now get the integer value.
  count_int, err := strconv.Atoi ( count )
  if err != nil {
    m_error ( "senders: error reading count: |%s|", err.Error() )
    return
  }

  //-------------------------------------------------
  // The unlabeled string.
  //-------------------------------------------------
  // If at this point we have no destination,
  // the user must have put an unlabeled router 
  // name in the arglist.
  if router == "" && edges == "" {
    router, _ = arglist.Get_string_cdr ( )
    if router == "" {
      m_error ( "senders: needs a destination." )
      return
    }
  }

  //--------------------------------------------------------
  // And now , fn-specific logic to remove contradictions.
  //--------------------------------------------------------
  // We can't have the user specifying both an unterior router,
  // and the edges of some router as the destinations.
  if router != "" && edges != "" {
    m_error (  "senders: Two destinations: router %s edges %s", router, edges )
    return
  }


  // Get the list of the routers we are going to distribute 
  // these senders over. If the user specified an interior
  // router, this will be a list of length one.
  var router_list [] string
  if router != "" {
    router_list = append ( router_list, router )
  } else {
    // Then we are doing edges.
    router_list = context.network.Get_router_edges ( edges )
  }

  final_addr := address
  variable_address := false
  if strings.Contains ( address, "%d" ) {
    variable_address = true
  }

  router_index := 0

  for i := 0; i < count_int; i ++ {
    context.sender_count ++
    sender_name := fmt.Sprintf ( "sender_%04d", context.sender_count )

    if variable_address {
      final_addr = fmt.Sprintf ( address, start_at_int )
      start_at_int ++
    }

    router_name := router_list[router_index]

    context.network.Add_sender ( sender_name,
                                 n_messages_int,
                                 max_message_length_int,
                                 router_name,
                                 final_addr,
                                 throttle )


    m_info ( context,
             "senders: added sender |%s| with addr |%s| to router |%s|.", 
             sender_name,
             final_addr,
             router_name )

    router_index ++
    if router_index >= len(router_list) {
      router_index = 0
    }
  }

  fp ( os.Stdout, " ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n\n\n" )
}





func dispatch_version ( context * Context, arglist * lisp.List ) {
  version_name, err := arglist.Get_atom ( 1 )
  if err != nil {
    m_error ( "dispatch_version: error on version name: %s", err.Error() )
    return
  }

  path, err := arglist.Get_atom ( 2 )
  if err != nil {
    m_error ( "dispatch_version: error on path: %s", err.Error() )
    return
  }

  if _, err := os.Stat ( path ); os.IsNotExist ( err ) {
    m_error ( "dispatch_version: %s version path does not exist: |%s|.", version_name, path )
    return
  }

  context.network.Add_dispatch_version ( version_name, path )
  m_info ( context, "dispatch_version: added version %s with path %s", version_name, path )

  // If this one is first, store it.
  // It will become the default.
  if context.first_version_name == "" {
    context.first_version_name = version_name
    m_info ( context, "dispatch_version: version %s is default.", context.first_version_name )
  }
}





func get_version_name ( context  * Context, input_name string ) ( string, error ) {

  var output_name string

  if input_name == "" {
    output_name = context.first_version_name
  }

  if output_name == "" {
    m_error ( "get_version_name: there are no dispatch versions defined. Use command dispatch_version.")
    return "", errors.New ( "No defined versions." )
  }

  m_info ( context, "dispatch version for this command set to %s", output_name )

  return output_name, nil
}





func linear ( context  * Context, arglist * lisp.List ) {

  count, e1 := arglist.Get_int ( ) 
  if e1 != nil {
    // The user did not specify a count.
    count = 3
    m_info ( context, "linear network count defaults to %d.", count )
  }

  version, e2 := arglist.Get_string_cdr ( )
  if e2 != nil {
    // The user did not specify a version.
    version = context.first_version_name
    if version == "" {
      m_error ( "linear: No dispatch version available." )
      return
    }

    m_info ( context, "linear network dispatch version defaults to %s.", version )
  }


  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( context )
    context.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    m_info ( context, "linear: added router |%s| with version |%s|.", router_name, version )
  }

  for index, name := range temp_names {
    if index < len(temp_names) - 1 {
      pitcher := name
      catcher := temp_names [ index + 1 ]
      context.network.Connect_router ( pitcher, catcher )
      m_info ( context, "linear: connected router |%s| to router |%s|", pitcher, catcher )
    }
  }
}





func run ( context  * Context, list * lisp.List ) {
  context.network.Init ( )
  context.network.Run  ( )

  context.network_running = true
  m_info ( context, "run: network is running." )
}






func main() {

  rand.Seed ( int64 ( os.Getpid()) )

  cwd, err := os.Getwd()
  if err != nil {
    m_error ( "Can't get cwd path for program name %s", os.Args[0] )
  }

  
  var context Context
  init_context ( & context )

  context.session_name = cwd + "/sessions/session_" + time.Now().Format ( "2006_01_02_1504" )
  utils.Find_or_create_dir ( context.session_name )

  context.mercury_log_name = context.session_name + "/mercury_log"
  context.mercury_log_file, _ = os.Create ( context.mercury_log_name )
  defer context.mercury_log_file.Close()

  context.line_rgx                = regexp.MustCompile(`\s+`)
  context.first_nonwhitespace_rgx = regexp.MustCompile(`\S`)


  /*===========================================
    Make commands. 
  ============================================*/

  context.commands = make ( map[string] * command, 0 )


  // verbose -------------------------------------------------------
  cmd := context.add_command ( "verbose",
                               verbose,
                               "Turn verbosity 'on' or 'off'." )
  cmd.add_arg ( "state",
                true,
                "string",
                "on",
                "'on' or 'off" )

  // paths -------------------------------------------------------
  // Is the dispatch path still used ??? 
  cmd = context.add_command ( "paths",
                              paths,
                              "Define dispatch, proton, and mercury paths." )


  // dispatch_version -------------------------------------------------------
  cmd = context.add_command ( "dispatch_version",
                              dispatch_version,
                              "Define different version of the dispatch code." )


  // linear -------------------------------------------------------
  cmd = context.add_command ( "linear",
                              linear,
                              "Create a linear router network." )


  // edges -------------------------------------------------------
  cmd = context.add_command ( "edges",
                              edges,
                              "Create edge router on a given interior router." )
  cmd.add_arg ( "count",
                true,
                "int",
                "",
                "How many edge routers to create." )

  cmd.add_arg ( "router",
                true,
                "string",
                "",
                "Which interior router to add the edges to." )

  cmd.add_arg ( "version",
                false,
                "string",
                "",
                "Which version of the dispatch code to use.\n" +
                "Defaults to the first version you defined." )


  // senders -------------------------------------------------------
  cmd = context.add_command ( "senders",
                              senders,
                              "Create message-sending clients." )

  // run -------------------------------------------------------
  cmd = context.add_command ( "run",
                              run,
                              "Start the network of routers and clients." )



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
    fp ( os.Stdout, "%c ", mercury )  // prompt
    line, _ := reader.ReadString ( '\n' )
    process_line ( & context, line )
  }
}





