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

import ( "fmt"
         "os"
         "strings"
         "os/exec"
         "time"
         "errors"
       )


var fp = fmt.Fprintf



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
  name                    string
  router_type             string
  worker_threads          int
  executable_path         string
  config_path             string
  log_path                string
  config_file_path        string
  dispatch_install_root   string
  proton_install_root     string
  client_port             string
  router_port             string
  edge_port               string
  verbose                 bool

  state                   router_state            
  cmd                   * exec.Cmd
  connect_to_ports        [] string
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
func New_Router ( name                  string, 
                  router_type           string,
                  worker_threads        int,
                  executable_path       string,
                  config_path           string,
                  log_path              string,
                  dispatch_install_root string,
                  proton_install_root   string,
                  client_port           string,
                  router_port           string,
                  edge_port             string,
                  verbose               bool ) * Router {
  var r * Router

  r = & Router { name                  : name, 
                 router_type           : router_type,
                 worker_threads        : worker_threads,
                 executable_path       : executable_path,
                 config_path           : config_path,
                 log_path              : log_path,
                 dispatch_install_root : dispatch_install_root,
                 proton_install_root   : proton_install_root,
                 client_port           : client_port,
                 router_port           : router_port,
                 edge_port             : edge_port,
                 verbose               : verbose }
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
func ( r * Router ) Router_Type ( ) string  {
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
    fp ( os.Stdout, "router : config file written to |%s|\n", r.config_file_path )
  }

  return nil
}





/*
  Call this only after calling Init() on the router.
  This fn sets up the router's environment variables, 
  and runs the router as a separate process.
*/
func ( r * Router ) Run ( ) error {

  if r.state == running {
    return nil
  }

  DISPATCH_LIBRARY_PATH := r.dispatch_install_root + "/lib64"
  PROTON_LIBRARY_PATH   := r.proton_install_root   + "/lib64"
  LD_LIBRARY_PATH       := DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH

  DISPATCH_PYTHONPATH   := r.dispatch_install_root + "/lib/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := r.dispatch_install_root + "/lib/python2.7/site-packages"
  PYTHONPATH            := DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2

  os.Setenv ( "LD_LIBRARY_PATH", LD_LIBRARY_PATH )
  os.Setenv ( "PYTHONPATH"     , PYTHONPATH )

  if r.verbose {
    fp ( os.Stdout, "router : LD_LIBRARY_PATH is |%s|\n", LD_LIBRARY_PATH )
    fp ( os.Stdout, "router : PYTHONPATH is |%s|\n", PYTHONPATH )
  }

  include := " -I " + r.dispatch_install_root + "/lib/qpid-dispatch/python"
  args := " --config " + r.config_file_path + include
  args_list := strings.Fields ( args )

  if r.verbose {
    fp ( os.Stdout, "router : command is |%s %s|\n", r.executable_path, args )}

  r.cmd = exec.Command ( r.executable_path,  args_list... )

  r.cmd.Start ( )
  r.state = running

  return nil
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

    case <-time.After ( 500 * time.Millisecond ) :
      if err := r.cmd.Process.Kill(); err != nil {
        return errors.New ( "failed to kill process: " + err.Error() )
      }

      // We killed the process: no error occurred.
      // fp ( os.Stderr, "Router %s halted normally.\n", r.name )
      r.state = halted
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





