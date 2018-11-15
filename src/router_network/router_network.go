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
  Package router_network controls a network of dispatch routers.

  Theory of Operation
  --------------------------------

  1. create the network.

  2. add routers to it. There are separate commands for 
     adding interior or edge routers. You just make up a 
     name.

  3. connect some routers to other routers. You just 
     give the names of the routers to be connected.

  4. Initialize the network. This causes all the config
     files to be written.

  5. Start the network.

  6. Talk to the routers in various ways. Right now, the
     only way is by calling Check_Links().

  7/ Halt the network.
*/
package router_network

import ( "fmt"
         "router"
         "os"
         "os/exec"
         "strings"
         "errors"
         "utils"
       )


var fp = fmt.Fprintf




/*
  Represents Dispatch Router network state and
  conatins the vector of routers.
*/
type Router_Network struct {
  Name             string
  worker_threads   int
  executable_path  string
  config_path      string
  log_path         string
  dispatch_root    string
  proton_root      string
  qdstat_path      string
  verbose          bool

  routers          [] * router.Router
}





/*
  Create a new router network.
  Tell it how many worker threads each router should have,
  and provide lots of paths:

    1. executable_path : router executable
    2. config_path     : where to put all the router config files
    3. log_path        : where to put all the log files
    4. dispatch_root   : the root of the installed dispatch code.
                         i.e. the directory that contains 
                           bin
                           etc
                           include
                           lib
                           sbin
                           share
    5. proton_root     : the root of the installed proton code.
                         i.e. the directory that contains
                           include
                           lib64
                           share
*/
func New_Router_Network ( name            string,
                          worker_threads  int,
                          executable_path string,
                          config_path     string,
                          log_path        string,
                          dispatch_root   string,
                          proton_root     string,
                          verbose         bool ) * Router_Network {
  var rn * Router_Network
  rn = & Router_Network { Name            : name,
                          worker_threads  : worker_threads,
                          executable_path : executable_path,
                          config_path     : config_path,
                          log_path        : log_path,
                          dispatch_root   : dispatch_root,
                          proton_root     : proton_root,
                          qdstat_path     : dispatch_root + "/bin/qdstat",
                          verbose         : verbose }

  return rn
}





func ( rn * Router_Network ) add_router ( name string, router_type string ) {
  client_port, _ := utils.Available_port ( )
  router_port, _ := utils.Available_port ( )
  edge_port, _   := utils.Available_port ( )

  router := router.New_Router ( name,
                                router_type,
                                rn.worker_threads,
                                rn.executable_path,
                                rn.config_path,
                                rn.log_path,
                                rn.dispatch_root,
                                rn.proton_root,
                                client_port,
                                router_port,
                                edge_port,
                                rn.verbose )
  rn.routers = append ( rn.routers, router )
}





/*
  Add a new router to the network. You can add all routers before
  calling Init, but it's also OK to 
*/
func ( rn * Router_Network ) Add_Router ( name string ) {
  rn.add_router ( name, "interior" )
}





func ( rn * Router_Network ) Add_Edge ( name string ) {
  rn.add_router ( name, "edge" )
}





func ( rn * Router_Network ) Connect ( router_1_name string, router_2_name string ) {
  router_1 := rn.get_router_by_name ( router_1_name )
  router_2 := rn.get_router_by_name ( router_2_name )

  if router_1.Router_Type() == "edge" {
    router_1.Connect_To ( router_2.Edge_Port() )
  } else {
    router_1.Connect_To ( router_2.Router_Port() )
  }
}




func ( rn * Router_Network ) Init ( ) {
  for _, router := range rn.routers {
    router.Init ( )
  }
}





func ( rn * Router_Network ) Run ( ) {
  for _, router := range rn.routers {
    router.Run ( )
  }
}





func ( rn * Router_Network ) Check_Links ( router_name string ) error {
  // set up env ----------------------------------------------
  INSTALL_ROOT          := "/home/mick/mercury/system_code/install"

  PROTON_INSTALL_DIR    := INSTALL_ROOT + "/proton"
  DISPATCH_INSTALL_DIR  := INSTALL_ROOT + "/dispatch"

  DISPATCH_LIBRARY_PATH := DISPATCH_INSTALL_DIR + "/lib64"
  PROTON_LIBRARY_PATH   := PROTON_INSTALL_DIR   + "/lib64"
  LD_LIBRARY_PATH       := DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH

  DISPATCH_PYTHONPATH   := DISPATCH_INSTALL_DIR + "/lib/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := DISPATCH_INSTALL_DIR + "/lib/python2.7/site-packages"
  PROTON_PYTHON_PATH    := PROTON_INSTALL_DIR   + "/lib64/proton/bindings/python"
  PYTHONPATH            := DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2 +":"+ PROTON_PYTHON_PATH

  os.Setenv ( "LD_LIBRARY_PATH", LD_LIBRARY_PATH )
  os.Setenv ( "PYTHONPATH"     , PYTHONPATH )
  // done set up env -----------------------------------------

  router := rn.get_router_by_name ( router_name )
  args := "-l -b 0.0.0.0:" + router.Client_Port ( )
  args_list := strings.Fields ( args )
  cmd := exec.Command ( rn.qdstat_path,  args_list... )
  out, _ := cmd.Output()
  lines := strings.Split ( string(out), "\n" )
  bad_links := 0

  for _, line := range lines {
    fields := strings.Fields ( line )
    if ( len(fields) >= 1 ) {
      if fields[0] == "endpoint" {
        enabled := 0
        up      := 0
        for _, field := range fields {
          if field == "enabled" {
            enabled = 1
          }
          if field == "up" {
            up = 1
          }
        }
        if enabled + up < 2 {
          bad_links ++
        }
      }
    }
  }

  if bad_links > 0 {
    return errors.New ( "endpoint link down" )
  }

  return nil
}





func ( rn * Router_Network ) Halt ( ) error {
  var first_err error
  for _, router := range rn.routers {
    err := router.Halt ( )
    if err != nil && first_err == nil {
      first_err = err
    }
  }

  return first_err
}





func ( rn * Router_Network ) get_router_by_name ( target_name string ) * router.Router {
  for _, router := range rn.routers {
    if router.Name() == target_name {
      return router
    }
  }

  return nil
}





