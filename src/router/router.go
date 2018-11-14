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
  state                   router_state            

  cmd                   * exec.Cmd
  connect_to_ports        [] string
}





/*
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
                  edge_port             string ) * Router {
  var r * Router

  r = & Router { name            : name, 
                 router_type     : router_type,
                 worker_threads  : worker_threads,
                 executable_path : executable_path,
                 config_path     : config_path,
                 log_path        : log_path,
                 client_port     : client_port,
                 router_port     : router_port,
                 edge_port       : edge_port }
  return r
}





func ( r * Router ) Connect_To ( port string ) {
  r.connect_to_ports = append ( r.connect_to_ports, port )
}





func ( r * Router ) Name ( ) string  {
  return r.name
}





func ( r * Router ) Router_Type ( ) string  {
  return r.router_type
}





func ( r * Router ) Client_Port ( ) string {
  return r.client_port
}





func ( r * Router ) Router_Port ( ) string {
  return r.router_port
}





func ( r * Router ) Edge_Port ( ) string {
  return r.edge_port
}





func make_indent ( indent_size int ) * strings.Builder {
  var indent strings.Builder
  indent.Grow ( indent_size )
  for i := 0; i < indent_size; i ++ {
    indent.WriteByte ( ' ' )
  }
  return & indent
}





func ( r * Router ) Print ( indent_size int ) {
  indent := make_indent ( indent_size )

  fp ( os.Stdout, "%srouter\n"                , indent.String() )
  fp ( os.Stdout, "%s{\n"                     , indent.String() )
  fp ( os.Stdout, "%s  name            : %s\n", indent.String(), r.name )
  fp ( os.Stdout, "%s  worker_threads  : %d\n", indent.String(), r.worker_threads )
  fp ( os.Stdout, "%s  executable_path : %s\n", indent.String(), r.executable_path )
  fp ( os.Stdout, "%s}\n"                     , indent.String() )
}





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

  return nil
}





func ( r * Router ) Run ( ) error {

  if r.state == running {
    return nil
  }

  INSTALL_ROOT          := "/home/mick/mercury/system_code/install"

  PROTON_INSTALL_DIR    := INSTALL_ROOT + "/proton"
  DISPATCH_INSTALL_DIR  := INSTALL_ROOT + "/dispatch"

  DISPATCH_LIBRARY_PATH := DISPATCH_INSTALL_DIR + "/lib64"
  PROTON_LIBRARY_PATH   := PROTON_INSTALL_DIR   + "/lib64"
  LD_LIBRARY_PATH       := DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH

  DISPATCH_PYTHONPATH   := DISPATCH_INSTALL_DIR + "/lib/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := DISPATCH_INSTALL_DIR + "/lib/python2.7/site-packages"
  PYTHONPATH            := DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2

  os.Setenv ( "LD_LIBRARY_PATH", LD_LIBRARY_PATH )
  os.Setenv ( "PYTHONPATH"     , PYTHONPATH )

  include := " -I /home/mick/mercury/system_code/qpid-dispatch/python"
  args := " --config " + r.config_file_path + include
  args_list := strings.Fields ( args )
  r.cmd = exec.Command ( r.executable_path,  args_list... )

  r.cmd.Start ( )
  r.state = running

  return nil
}





func ( r * Router ) Halt ( ) error {

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





