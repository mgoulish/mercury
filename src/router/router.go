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
  router_port                    string
  edge_port                      string
  verbose                        bool

  state                          router_state            
  cmd                          * exec.Cmd
  connect_to_ports               [] string
  resource_usage_dir             string
  mem_usage_file_name            string
  cpu_usage_file_name            string
  resource_measurement_frequency int
  resource_ticker              * time.Ticker
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
                  router_type                 string,
                  worker_threads              int,
                  result_path                 string,
                  executable_path             string,
                  config_path                 string,
                  log_path                    string,
                  dispatch_install_root       string,
                  proton_install_root         string,
                  client_port                 string,
                  router_port                 string,
                  edge_port                   string,
                  verbose                     bool,
                  usage_measurement_frequency int ) * Router {
  var r * Router

  r = & Router { name                           : name, 
                 router_type                    : router_type,
                 worker_threads                 : worker_threads,
                 result_path                    : result_path,
                 executable_path                : executable_path,
                 config_path                    : config_path,
                 log_path                       : log_path,
                 dispatch_install_root          : dispatch_install_root,
                 proton_install_root            : proton_install_root,
                 client_port                    : client_port,
                 router_port                    : router_port,
                 edge_port                      : edge_port,
                 verbose                        : verbose,
                 resource_measurement_frequency : usage_measurement_frequency }
  return r
}




/*
  Tell the router a port number (represented as a string)
  that it should attach to.
*/
func ( r * Router ) Connect_To ( port string ) {
  r.connect_to_ports = append ( r.connect_to_ports, port )
}





/*
  Get the router's name.
*/
func ( r * Router ) Name ( ) string  {
  return r.name
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
func ( r * Router ) Client_Port ( ) string {
  return r.client_port
}





/*
  Get the router's router port number (as a string).
  Or nil if this is an edge router.
*/
func ( r * Router ) Router_Port ( ) string {
  return r.router_port
}





/*
  Get the router's edge port number (as a string).
  Or nil if this is an edge router.
*/
func ( r * Router ) Edge_Port ( ) string {
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
  for _, port := range r.connect_to_ports {
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





func ( r * Router ) check_memory ( ) {
  // /home/mick/latest/install/dispatch
  // fp ( os.Stderr, "r.dispatch_install_root == |%s|\n", r.dispatch_install_root )

  // fp ( os.Stderr, "r.proton_install_root == |%s|\n", r.proton_install_root )

  // set up env ----------------------------------------------
  // 
  // 
  // export LD_LIBRARY_PATH=/home/mick/latest/install/dispatch/lib:/home/mick/latest/install/proton/lib64
// export PYTHONPATH=/home/mick/latest/install/dispatch/lib/qpid-dispatch/python:/home/mick/latest/install/dispatch/lib/python2.7/site-packages:/home/mick/latest/install/proton/lib64:/home/mick/latest/install/proton/lib64/proton/bindings/python

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

  args := "-m -b 0.0.0.0:" + r.Client_Port ( )
  args_list := strings.Fields ( args )

  fp ( os.Stderr, "\n\n\ncheck_mem: LD_LIBRARY_PATH |%s|\n", LD_LIBRARY_PATH )
  fp ( os.Stderr, "check_mem: PYTHONPATH |%s|\n", LD_LIBRARY_PATH )
  fp ( os.Stderr, "check_mem: command: |%s %s|\n\n\n", qdstat_path, args )

  cmd := exec.Command ( qdstat_path,  args_list... )
  out, _ := cmd.Output()

  fp ( os.Stderr, "router %s ---------------------------------------------------------\n", r.name )
  fp ( os.Stderr, "check mem: here's the output: |%s|\n", out )
}





/*
  Call this only after calling Init() on the router.
  This fn sets up the router's environment variables, 
  and runs the router as a separate process.
*/
func ( r * Router ) Run ( do_resource_measurement bool ) error {

  if r.state > initialized {
    return nil
  }

  DISPATCH_LIBRARY_PATH := r.dispatch_install_root + "/lib64"
  PROTON_LIBRARY_PATH   := r.proton_install_root   + "/lib64"
  LD_LIBRARY_PATH       := DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH

  DISPATCH_PYTHONPATH   := r.dispatch_install_root + "/lib/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := r.dispatch_install_root + "/lib/python2.7/site-packages"
  PYTHONPATH            := DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2

  // Set up the environment for the router process.
  os.Setenv ( "LD_LIBRARY_PATH", LD_LIBRARY_PATH )
  os.Setenv ( "PYTHONPATH"     , PYTHONPATH )
  include := " -I " + r.dispatch_install_root + "/lib/qpid-dispatch/python"
  args := " --config " + r.config_file_path + include
  args_list := strings.Fields ( args )

  // Start the router process and gets its pid for the result directory name.
  r.cmd = exec.Command ( r.executable_path,  args_list... )
  r.cmd.Start ( )
  r.state = running
  env_dir := r.result_path + "/env/" + strconv.Itoa(r.cmd.Process.Pid) 
  utils.Find_or_create_dir ( env_dir )

  if do_resource_measurement {
    r.resource_usage_dir = r.result_path + "/resources/" + strconv.Itoa(r.cmd.Process.Pid)
    utils.Find_or_create_dir ( r.resource_usage_dir )
    r.mem_usage_file_name = r.resource_usage_dir + "/mem"
    r.cpu_usage_file_name = r.resource_usage_dir + "/cpu"
  }

  // Write the environment variables to the results directory.
  // This helps the user to reproduce this test, if desired.
  env_file_name := env_dir + "/environment_variables"
  env_file, err := os.Create ( env_file_name )
  utils.Check ( err )
  defer env_file.Close ( )
  env_string := "export LD_LIBRARY_PATH=" + LD_LIBRARY_PATH + "\n"
  env_file.WriteString ( env_string )
  env_string  = "export PYTHONPATH=" + PYTHONPATH + "\n"
  env_file.WriteString ( env_string )

  // Write the command line to the results directory.
  // This helps the user to reproduce this test, if desired.
  command_file_name := env_dir + "/command_line"
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
    r.check_memory ( )
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





