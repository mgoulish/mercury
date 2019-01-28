package main

import (
  "bufio"
  "fmt"
  "os"
  "regexp"
  "time"
  "math/rand"
  "errors"

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










/*=====================================================================
  Main
======================================================================*/


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


  // verbose command -------------------------------------------------------
  cmd := context.add_command ( "verbose",
                               verbose,
                               "Turn verbosity 'on' or 'off'." )
  cmd.add_arg ( "state",
                true,        // unlabelable
                "string",
                "on",
                "'on' or 'off" )

  // paths command -------------------------------------------------------
  // Is the dispatch path still used ??? 
  cmd = context.add_command ( "paths",
                              paths,
                              "Define dispatch, proton, and mercury paths." )


  // dispatch_version command -------------------------------------------------------
  cmd = context.add_command ( "dispatch_version",
                              dispatch_version,
                              "Define different version of the dispatch code." )


  // linear command -------------------------------------------------------
  cmd = context.add_command ( "linear",
                              linear,
                              "Create a linear router network." )
  cmd.add_arg ( "count",
                true,   // unlabelable
                "int",
                "3",    // default is 3 routers
                "How many edge routers to create in the linear network." )

  cmd.add_arg ( "version",
                true,
                "string",
                "",
                "Which version of the dispatch code to use.\n" +
                "Defaults to the first version you defined." )




  // edges command -------------------------------------------------------
  cmd = context.add_command ( "edges",
                              edges,
                              "Create edge router on a given interior router." )
  cmd.add_arg ( "count",
                true,   // unlabelable
                "int",
                "",
                "How many edge routers to create." )

  cmd.add_arg ( "router",
                true,   // unlabelable
                "string",
                "",
                "Which interior router to add the edges to." )

  cmd.add_arg ( "version",
                false,
                "string",
                "",
                "Which version of the dispatch code to use.\n" +
                "Defaults to the first version you defined." )


  // send command -------------------------------------------------------
  cmd = context.add_command ( "send",
                              send,
                              "Create message-sending clients." )

  cmd.add_arg ( "router",
                true,   // unlabelable
                "string",
                "",
                "Which router the senders should attach to." )

  cmd.add_arg ( "count",
                true,   // unlabelable
                "int",
                "1",
                "How many senders to make." )

  cmd.add_arg ( "n_messages",
                false,
                "int",
                "100000",
                "How many messages to send." )

  cmd.add_arg ( "max_message_length",
                false,
                "int",
                "1000",
                "Max length for each messages. " +
                "Lengths will be random, and average will be half this." )

  cmd.add_arg ( "edges",
                false,
                "string",
                "",
                "Add senders to the edges of this router. " +
                "i.e. 'edges A' means add senders to edges of router A." )

  cmd.add_arg ( "throttle",
                false,
                "string",    // Just ... don't ask.
                "0",
                "How many msec between each sent message. " +
                "0 means send as fast as possible." )

  cmd.add_arg ( "address",
                false,
                "string",
                "my_address",
                "Address to send to. Embed a '%d' if you " +
                "want addresses to count up." )

  cmd.add_arg ( "start_at",
                false,
                "int",
                "1",
                "If you use %d in address, use this to tell what int " +
                "the counting should start with." )




  // recv command -------------------------------------------------------
  cmd = context.add_command ( "recv",
                              recv,
                              "Create message-receiving clients." )

  cmd.add_arg ( "router",
                true,   // unlabelable
                "string",
                "",
                "Which router the senders should attach to." )

  cmd.add_arg ( "count",
                true,   // unlabelable
                "int",
                "1",
                "How many senders to make." )

  cmd.add_arg ( "n_messages",
                false,
                "int",
                "100000",
                "How many messages to send." )

  cmd.add_arg ( "edges",
                false,
                "string",
                "",
                "Add senders to the edges of this router. " +
                "i.e. 'edges A' means add senders to edges of router A." )

  cmd.add_arg ( "address",
                false,
                "string",
                "my_address",
                "Address to send to. Embed a '%d' if you " +
                "want addresses to count up." )

  cmd.add_arg ( "start_at",
                false,
                "int",
                "1",
                "If you use %d in address, use this to tell what int " +
                "the counting should start with." )

  cmd.add_arg ( "max_message_length",
                false,
                "int",
                "1000",
                "Max length for each messages. " )





  // run command -------------------------------------------------------
  cmd = context.add_command ( "run",
                              run,
                              "Start the network of routers and clients." )


  // quit command -------------------------------------------------------
  cmd = context.add_command ( "quit",
                              quit,
                              "Shut down the network and halt Mercury." )



  // console_ports command -------------------------------------------------------
  cmd = context.add_command ( "console_ports",
                              console_ports,
                              "Show the console ports for all routers." )



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





