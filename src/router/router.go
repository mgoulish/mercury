/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */


/*
  Package router implements a dispatch router.
  The normal order for operations on a router is:
    * create
    * connect to other routers
    * init
    * run
    * halt
  It is during the initialization step that the 
  configuration file is written that will be read at 
  router startup. So any connecting that you want to do
  should be done before you call Init.
  Every router is created with a normal-mode listener, so 
  you can always attach a client to it, or send it 
  management commands.
*/
package router

import ( "errors"
         "fmt"
         "os"
         "os/exec"
         "strconv"
         "strings"
         "time"
         "utils"
         qdo "qdstat_output"
       )


var fp          = fmt.Fprintf
var upl         = utils.Print_log
var module_name = "router"





type router_state int

const (
  none         router_state = iota
  initialized
  running                 
  halted
)





func state_to_string ( state router_state ) ( string ) {
  switch state {
    case initialized :
        return "initialized"
    case running :
        return "running"
    case halted :
      return "halted"
    default :
      return "none"
  }
}





/*
  The Router struct represents a Dispatch Router,
  maintains state about it, remembers all the paths 
  it needs to know, and contains the exec command 
  that is used to manipulate the running process.

  The router runs as a separate process from the main 
  program.
*/
type Router struct {
  name                           string
  version                        string
  router_type                    string
  worker_threads                 int
  result_path                    string
  executable_path                string
  config_path                    string
  log_path                       string
  config_file_path               string
  dispatch_install_root          string
  proton_install_root            string
  client_port                    string
  console_port                   string
  router_port                    string
  edge_port                      string
  verbose                        bool

  pid                            int
  state                          router_state            
  cmd                          * exec.Cmd
  i_connect_to_ports            [] string
  i_connect_to_names            [] string

  connect_to_me_interior        [] string
  connect_to_me_edge            [] string

  resource_usage_dir             string
  mem_usage_file_name            string
  cpu_usage_file_name            string
  resource_measurement_frequency int
  resource_ticker              * time.Ticker

  qdstat_outputs            [] * qdo.Qdstat_output
  qdstat_output_filename         string
}





/*
  Create a new router.
  The caller supplies all the paths and ports.
  Every router will be given a listener for clients
  to connect to. Even if a particular test does not
  need clients to connect to this router, the support
  libraries will sometimes need to talk to this port.

  Interior routers will also be given a listener for 
  other interior routers, and a separate listener for
  edge routers.
*/
func New_Router ( name                        string, 
                  version                     string,
                  router_type                 string,
                  worker_threads              int,
                  result_path                 string,
                  executable_path             string,
                  config_path                 string,
                  log_path                    string,
                  dispatch_install_root       string,
                  proton_install_root         string,
                  client_port                 string,
                  console_port                string,
                  router_port                 string,
                  edge_port                   string,
                  verbose                     bool,
                  usage_measurement_frequency int ) * Router {
  var r * Router

  r = & Router { name                           : name, 
                 version                        : version,
                 router_type                    : router_type,
                 worker_threads                 : worker_threads,
                 result_path                    : result_path,
                 executable_path                : executable_path,
                 config_path                    : config_path,
                 log_path                       : log_path,
                 dispatch_install_root          : dispatch_install_root,
                 proton_install_root            : proton_install_root,
                 client_port                    : client_port,
                 console_port                   : console_port,
                 router_port                    : router_port,
                 edge_port                      : edge_port,
                 verbose                        : verbose,
                 resource_measurement_frequency : usage_measurement_frequency }

  r.qdstat_output_filename = r.result_path + "/qdstat_output_" + r.name

  if ! utils.Path_exists ( executable_path ) {

    fp ( os.Stdout, 
         "    router error: executable path |%s| does not exist.\n", 
         executable_path )
    os.Exit ( 1 )
  }

  return r
}




func ( r * Router ) Edges ( ) ( [] string ) {
  return r.connect_to_me_edge
}





func ( r * Router ) Connected_to_you ( router_name string, edge bool ) {
  if edge {
    r.connect_to_me_edge = append ( r.connect_to_me_edge, router_name )
  } else {
    r.connect_to_me_interior = append ( r.connect_to_me_interior, router_name )
  }
}





func ( r * Router ) Print_console_port () {
  fp ( os.Stdout, "  router %s console port %s\n", r.name, r.console_port )
}





func ( r * Router ) Print () {
  fp ( os.Stdout, "router %s -------------\n", r.name )
  fp ( os.Stdout, "  PID: %d\n", r.pid )
  fp ( os.Stdout, "  state:  %s\n", state_to_string ( r.state ) )
  fp ( os.Stdout, "  client  port: %s\n", r.client_port )
  fp ( os.Stdout, "  router  port: %s\n", r.router_port )
  fp ( os.Stdout, "  edge    port: %s\n", r.edge_port )
  fp ( os.Stdout, "  console port: %s\n", r.console_port )
  fp ( os.Stdout, "\n" )

  if 0 < len(r.i_connect_to_names) {
    fp ( os.Stdout, "  Routers that I connect to:\n" )
    for i, name := range r.i_connect_to_names {
      fp ( os.Stdout, "    %s %s\n", name, r.i_connect_to_ports [ i ] )
    }
  }

  if 0 < len(r.connect_to_me_interior) {
    fp ( os.Stdout, "  Interior routers that connect to me:\n" )
    for _, name := range r.connect_to_me_interior {
      fp ( os.Stdout, "    %s\n", name )
    }
  }

  if 0 < len(r.connect_to_me_edge) {
    fp ( os.Stdout, "  Edge routers that connect to me:\n" )
    for _, name := range r.connect_to_me_edge {
      fp ( os.Stdout, "    %s\n", name )
    }
  }

  fp ( os.Stdout, "\n" )
}





// Tell the router a port number (represented as a string)
// that it should attach to.
func ( r * Router ) Connect_to ( name string, port string ) {
  r.i_connect_to_ports = append ( r.i_connect_to_ports, port )
  r.i_connect_to_names = append ( r.i_connect_to_names, name )
}





// Get the router's name.
func ( r * Router ) Name ( ) string  {
  return r.name
}





func ( r * Router ) Is_interior () (bool) {
  return r.router_type == "interior"
}





/*
  Get the router's type, i.e. interior or edge.
*/
func ( r * Router ) Type ( ) string  {
  return r.router_type
}





/*
  Get the router's client port number (as a string).
*/
func ( r * Router ) Client_port ( ) string {
  return r.client_port
}





/*
  Get the router's router port number (as a string).
  Or nil if this is an edge router.
*/
func ( r * Router ) Router_port ( ) string {
  return r.router_port
}





/*
  Get the router's edge port number (as a string).
  Or nil if this is an edge router.
*/
func ( r * Router ) Edge_port ( ) string {
  return r.edge_port
}





/*
  Initialization of a router does whatever is needed 
  to get ready to launch the router, i.e. write the
  configuration file.
*/
func ( r * Router ) Init ( ) error {
  if r.state >= initialized {
    return nil
  }

  r.config_file_path = r.config_path + "/" + r.name + ".conf"
  r.state = initialized
  return r.write_config_file ( )
}





func ( r * Router ) write_config_file ( ) error {
  f, err := os.Create ( r.config_file_path )
  if err != nil {
    return err
  }
  defer f.Close ( )

  fp ( f, "router {\n" )
  fp ( f, "  workerThreads : %d\n", r.worker_threads )
  fp ( f, "  mode          : %s\n", r.router_type )
  fp ( f, "  id            : %s\n", r.name )
  fp ( f, "}\n" )

  fp ( f, "address {\n" );
  fp ( f, "  prefix       : closest\n" );
  fp ( f, "  distribution : closest\n" );
  fp ( f, "}\n" )

  fp ( f, "address {\n" );
  fp ( f, "  prefix       : balanced\n" );
  fp ( f, "  distribution : balanced\n" );
  fp ( f, "}\n" )

  fp ( f, "address {\n" );
  fp ( f, "  prefix       : multicast\n" );
  fp ( f, "  distribution : multicast\n" );
  fp ( f, "}\n" )

  fp ( f, "log {\n" )
  fp ( f, "  outputFile    : %s.log\n", r.log_path + "/" +r.name )
  // Use this if you want no output.
  //fp ( f, "  enable        : none\n" )
  fp ( f, "  includeSource : true\n" )
  fp ( f, "  module        : DEFAULT\n" )
  fp ( f, "}\n" )

  // The Client Listener -----------------
  fp ( f, "listener {\n" )
  fp ( f, "  role               : normal\n")
  fp ( f, "  host               : 0.0.0.0\n")
  fp ( f, "  port               : %s\n", r.client_port )
  fp ( f, "  stripAnnotations   : no\n")
  fp ( f, "  idleTimeoutSeconds : 120\n")
  fp ( f, "  saslMechanisms     : ANONYMOUS\n")
  fp ( f, "  authenticatePeer   : no\n")
  fp ( f, "}\n")

  // The Console Listener -----------------
  console_dir := r.dispatch_install_root + "/share/qpid-dispatch/console/stand-alone"

  fp ( f, "listener {\n" )
  fp ( f, "  role               : normal\n")
  fp ( f, "  host               : 0.0.0.0\n")
  fp ( f, "  port               : %s\n", r.console_port )
  fp ( f, "  stripAnnotations   : no\n")
  fp ( f, "  idleTimeoutSeconds : 120\n")
  fp ( f, "  saslMechanisms     : ANONYMOUS\n")
  fp ( f, "  authenticatePeer   : no\n")
  fp ( f, "  http               : true\n")
  fp ( f, "  httpRoot           : %s\n", console_dir)
  fp ( f, "}\n")

  // The Router Listener -----------------
  if r.router_type != "edge" {
    fp ( f, "listener {\n" )
    fp ( f, "  role               : inter-router\n")
    fp ( f, "  host               : 0.0.0.0\n")
    fp ( f, "  port               : %s\n", r.router_port )
    fp ( f, "  stripAnnotations   : no\n")
    fp ( f, "  idleTimeoutSeconds : 120\n")
    fp ( f, "  saslMechanisms     : ANONYMOUS\n")
    fp ( f, "  authenticatePeer   : no\n")
    fp ( f, "}\n")
  }

  /*
  */
  // The Edge Listener -----------------
  if r.router_type != "edge" {
    // Edge routers do not get an edge listener.
    fp ( f, "listener {\n" )
    fp ( f, "  role               : edge\n")
    fp ( f, "  host               : 0.0.0.0\n")
    fp ( f, "  port               : %s\n", r.edge_port )
    fp ( f, "  stripAnnotations   : no\n")
    fp ( f, "  idleTimeoutSeconds : 120\n")
    fp ( f, "  saslMechanisms     : ANONYMOUS\n")
    fp ( f, "  authenticatePeer   : no\n")
    fp ( f, "}\n")
  }

  // The Connectors --------------------
  for _, port := range r.i_connect_to_ports {
    // fp ( os.Stderr, "router %s connect to %s\n", r.name, port )
    fp ( f, "connector {\n" )
    //fp ( f, "  verifyHostname     : no\n")
    fp ( f, "  name               : %s_connector_to_%s\n", r.name, port)
    fp ( f, "  idleTimeoutSeconds : 120\n")
    fp ( f, "  saslMechanisms     : ANONYMOUS\n")
    fp ( f, "  host               : 127.0.0.1\n")
    fp ( f, "  port               : %s\n", port)
    if r.router_type == "edge" {
      fp ( f, "  role               : edge\n")
    } else {
      fp ( f, "  role               : inter-router\n")
    }
    fp ( f, "}\n")
  }

  if r.verbose {
    upl ( "config file written to |%s|", module_name, r.config_file_path )
  }

  return nil
}





func ( r * Router ) Check_memory ( ) {

  qdstat_path := r.dispatch_install_root + "/bin/qdstat"

  DISPATCH_LIBRARY_PATH := r.dispatch_install_root + "/lib"
  PROTON_LIBRARY_PATH   := r.proton_install_root   + "/lib64"
  LD_LIBRARY_PATH       := DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH

  DISPATCH_PYTHONPATH   := DISPATCH_LIBRARY_PATH + "/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := DISPATCH_LIBRARY_PATH + "/python2.7/site-packages"

  PROTON_PYTHONPATH     := PROTON_LIBRARY_PATH + "/proton/bindings/python"

  PYTHONPATH            := DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2 +":"+ PROTON_LIBRARY_PATH +":"+ PROTON_PYTHONPATH

  // Set up the environment for the router process.
  os.Setenv ( "LD_LIBRARY_PATH", LD_LIBRARY_PATH )
  os.Setenv ( "PYTHONPATH"     , PYTHONPATH )
  // done set up env -----------------------------------------

  args := "-m -b 0.0.0.0:" + r.Client_port ( )
  args_list := strings.Fields ( args )

  cmd := exec.Command ( qdstat_path,  args_list... )
  out, _ := cmd.Output()

  // At this point the router has run and stopped,
  // and all qdstat outputs that happened during the 
  // run are stored in an internal router variable:
  // "r.qdstat_outputs".
  r.qdstat_output_filename = r.result_path + "/qdstat_output_" + r.name
  f, _ := os.OpenFile ( r.qdstat_output_filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660);
  defer f.Close()
  fmt.Fprintf ( f, "output %d -----------------------\n%s\n\n", len(r.qdstat_outputs), out )
  r.qdstat_outputs = append ( r.qdstat_outputs, qdo.New_qdstat_output ( string(out) ) )
}





/*
  Call this only after calling Init() on the router.
  This fn sets up the router's environment variables, 
  and runs the router as a separate process.
*/
func ( r * Router ) Run ( ) error {

  if r.state > initialized {
    return nil
  }

  var do_resource_measurement bool
  if r.resource_measurement_frequency > 0 {
    do_resource_measurement = true
  } else {
    do_resource_measurement = false
  }

  if r.verbose {
    fp ( os.Stderr, "router.Run router %s\n", r.name )
  }

  DISPATCH_LIBRARY_PATH := r.dispatch_install_root + "/lib"
  PROTON_LIBRARY_PATH   := r.proton_install_root   + "/lib64"
  LD_LIBRARY_PATH       := DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH

  DISPATCH_PYTHONPATH   := r.dispatch_install_root + "/lib/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := r.dispatch_install_root + "/lib/python2.7/site-packages"
  PYTHONPATH            := DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2

  // Set up the environment for the router process.
  os.Setenv ( "LD_LIBRARY_PATH", LD_LIBRARY_PATH )
  os.Setenv ( "PYTHONPATH"     , PYTHONPATH )
  include := " -I " + r.dispatch_install_root + "/lib/qpid-dispatch/python"


  router_args     := " --config " + r.config_file_path + include
  args            := router_args

  args_list := strings.Fields ( args )

  // Start the router process and get its pid for the result directory name.
  // After the Start() call, the router process is running detached.
  r.cmd = exec.Command ( r.executable_path,  args_list... )
  if r.cmd == nil {
    fp ( os.Stdout, "   router.Run error: can't execute |%s|\n", r.executable_path )
    return errors.New ( "Can't execute router executable." )
  }
  r.cmd.Start ( )
  r.state = running

  if r.cmd.Process == nil {
    fp ( os.Stdout, "   router.Run error: can't execute |%s|\n", r.executable_path )
    return errors.New ( "Can't execute router executable." )
  }

  r.pid = r.cmd.Process.Pid
  setup_dir := r.result_path + "/setup/" + r.name
  utils.Find_or_create_dir ( setup_dir )

  if do_resource_measurement {
    r.resource_usage_dir = r.result_path + "/resources/" + strconv.Itoa(r.cmd.Process.Pid)
    utils.Find_or_create_dir ( r.resource_usage_dir )
    r.mem_usage_file_name = r.resource_usage_dir + "/mem"
    r.cpu_usage_file_name = r.resource_usage_dir + "/cpu"
  }

  // Write the environment variables to the setup directory.
  // This helps the user to reproduce this test, if desired.
  env_file_name := setup_dir + "/environment_variables"
  env_file, err := os.Create ( env_file_name )
  utils.Check ( err )
  defer env_file.Close ( )
  env_string := "export LD_LIBRARY_PATH=" + LD_LIBRARY_PATH + "\n"
  env_file.WriteString ( env_string )
  env_string  = "export PYTHONPATH=" + PYTHONPATH + "\n"
  env_file.WriteString ( env_string )

  // Write the command line to the results directory.
  // This helps the user to reproduce this test, if desired.
  command_file_name := setup_dir + "/command_line"
  command_file, err := os.Create ( command_file_name )
  utils.Check ( err )
  defer command_file.Close ( )
  command_string := r.executable_path + " " + args
  command_file.WriteString ( command_string + "\n" )

  // Start the resource usage ticker.
  if do_resource_measurement {
    ticker_time := time.Second * time.Duration ( r.resource_measurement_frequency )
    r.resource_ticker = time.NewTicker ( ticker_time )
    go r.resource_measurement_ticker ( )
  }

  return nil
}





func ( r * Router ) Is_not_halted ( ) ( bool ) {
  return "halted" != r.State()
}





func ( r * Router ) State ( ) ( string ) {

  switch r.state {

    case none: 
      return "none"

    case initialized:
      return "initialized"

    case running:
      return "running"

    case halted:
      return "halted"
  }

  return "error"
}





func ( r * Router ) resource_measurement_ticker ( ) {
  for range r.resource_ticker.C {
    r.record_resource_usage ( )
    r.Check_memory ( )
  }
}





/*
  Halt the router.
  If it has already halted on its own, that is returned
  as an error. If the process returned an error code, 
  return that to the caller -- but early termination is
  considered an error even if the process did not return 
  an error code.
*/
func ( r * Router ) Halt ( ) error {

  if r.verbose {
    fp ( os.Stdout, "halting router |%s|\n", r.name )
  }

  if r.state == halted {
    if r.verbose {
      upl ( "Attempt to re-halt router %s", module_name, r.name )
    }
    return nil
  }

  if r.resource_ticker != nil {
    r.resource_ticker.Stop()
  }

  // Set up a channel that will return a 
  // message immediately if the process has
  // already terminated. Then set up a half-second 
  // timer. If the timer expires before the Wait 
  // returns a 'done' message, we judge that the 
  // process was still running when we came along
  // and killed it. Which is good.
  done := make ( chan error, 1 )
  go func ( ) {
      done <- r.cmd.Wait ( )
  } ( )

  select {
    /*
      This is the expected case.
      Our timer times out while the above Wait() is still waiting.
      This means that the process is still running normally when we kill it.
    */
    case <-time.After ( 250 * time.Millisecond ) :
      r.state = halted
      if err := r.cmd.Process.Kill(); err != nil {
        return errors.New ( "failed to kill process: " + err.Error() )
      }
      if len(r.qdstat_outputs) > 0 {
        last_index := len(r.qdstat_outputs) - 1  
        diffed_output := r.qdstat_outputs[last_index].Diff ( r.qdstat_outputs[0] )
        f, _ := os.OpenFile ( r.qdstat_output_filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660);
        defer f.Close()
        fp ( f, "\n\n\nDIFFED QDSTAT OUTPUT for router %s --------------------- \n", r.name )
        diffed_output.Print_nonzero ( f )
      }
      return nil

    case err := <-done:
      if err != nil {
        return errors.New ( "process terminated early with error: " + err.Error() )
      }

      // Even though there was no error reported -- the process 
      // mevertheless stopped early, which is an error in the 
      // context of this test.
      return errors.New ( "process self-terminated." )
  }

  return nil
}





func ( r * Router ) Pid ( ) ( int )  {
  return int ( r.cmd.Process.Pid )
}





func ( r * Router ) record_resource_usage ( ) {
  // memory ----------------------------
  rss := utils.Memory_usage ( r.cmd.Process.Pid )
  mem_usage_file, err := os.OpenFile ( r.mem_usage_file_name, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0600 )
  utils.Check ( err )
  defer mem_usage_file.Close ( )
  mem_usage_file.WriteString ( utils.Timestamp() + " " + strconv.Itoa ( rss ) + "\n" )

  // cpu ----------------------------
  cpu_usage_file, err := os.OpenFile ( r.cpu_usage_file_name, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0600 )
  utils.Check ( err )
  defer cpu_usage_file.Close ( )
  cpu_usage := utils.Cpu_usage ( r.cmd.Process.Pid )
  cpu_usage_file.WriteString ( utils.Timestamp() + " " + strconv.Itoa ( cpu_usage ) + "\n" )
}





