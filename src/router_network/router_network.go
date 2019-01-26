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

import ( "errors"
         "fmt"
         "os"
         "os/exec"
         "strings"
         "sync"
         "time"
         "utils"
         "router"
         "client"
       )


var fp          = fmt.Fprintf
var upl         = utils.Print_log
var module_name = "router_network"



/*
  Represents Dispatch Router network state and
  conatins the vector of routers.
*/
type Router_Network struct {
  Name                        string

  worker_threads                 int
  result_path                    string
  executable_path                string
  config_path                    string
  log_path                       string
  client_path                    string
  dispatch_root                  string
  proton_root                    string
  qdstat_path                    string
  verbose                        bool
  resource_measurement_frequency int

  routers                        [] * router.Router
  clients                        [] * client.Client

  Dispatch_versions              map [string] string
}





/*
  Create a new router network.
  Tell it how many worker threads each router should have,
  and provide lots of paths.

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
func New_Router_Network ( name                           string,
                          worker_threads                 int,
                          result_path                    string,
                          executable_path                string,
                          config_path                    string,
                          log_path                       string,
                          client_path                    string,
                          dispatch_root                  string,
                          proton_root                    string,
                          verbose                        bool,
                          resource_measurement_frequency int ) * Router_Network {
  var rn * Router_Network

  rn = & Router_Network { Name                           : name,
                          worker_threads                 : worker_threads,
                          result_path                    : result_path,
                          executable_path                : executable_path,
                          config_path                    : config_path,
                          log_path                       : log_path,
                          client_path                    : client_path,
                          dispatch_root                  : dispatch_root,
                          proton_root                    : proton_root,
                          qdstat_path                    : dispatch_root + "/bin/qdstat",
                          verbose                        : verbose,
                          resource_measurement_frequency : resource_measurement_frequency }

  rn.Dispatch_versions = make ( map [string] string )
  return rn
}



func ( rn * Router_Network ) N_routers ( ) ( int ) {
  return len ( rn.routers )
}





func ( rn * Router_Network ) N_interior_routers ( ) ( int ) {
  count := 0
  for _, r := range rn.routers {
    if r.Type() == "interior" {
      count ++
    }
  }
  return count
}





func ( rn * Router_Network ) Get_router_edges ( router_name string ) ( [] string ) {
  rtr := rn.get_router_by_name ( router_name )
  if rtr == nil {
    fp ( os.Stdout, "    network.Get_router_edges error: can't find router |%s|\n", router_name )
    return nil
  }

  return rtr.Edges ( )
}





func ( rn * Router_Network ) Print ( ) {
  fp ( os.Stdout, "network                          |%s|\n", rn.Name )
  fp ( os.Stdout, "  worker_threads                  %d\n",  rn.worker_threads )
  fp ( os.Stdout, "  result_path                    |%s|\n", rn.result_path )
  fp ( os.Stdout, "  executable_path                |%s|\n", rn.executable_path )
  fp ( os.Stdout, "  config_path                    |%s|\n", rn.config_path )
  fp ( os.Stdout, "  log_path                       |%s|\n", rn.log_path )
  fp ( os.Stdout, "  client_path                    |%s|\n", rn.client_path )
  fp ( os.Stdout, "  dispatch_root                  |%s|\n", rn.dispatch_root )
  fp ( os.Stdout, "  proton_root                    |%s|\n", rn.proton_root )
  fp ( os.Stdout, "  verbose                         %t\n",  rn.verbose )
  fp ( os.Stdout, "  resource_measurement_frequency  %d\n",  rn.resource_measurement_frequency )
  fp ( os.Stdout, "\n" )

  for _, r := range rn.routers {
    r.Print ( )
  }
}





func ( rn * Router_Network ) Print_console_ports ( ) {
  for _, r := range rn.routers {
    r.Print_console_port ( )
  }
}






func ( rn * Router_Network ) Check_memory_all () {
  for _, r := range rn.routers {
    rn.Check_memory ( r.Name() )
  }
}





func ( rn * Router_Network ) add_router ( name string, router_type string, version string ) {
  console_port, _ := utils.Available_port ( )
  client_port, _ := utils.Available_port ( )
  router_port, _ := utils.Available_port ( )
  edge_port, _   := utils.Available_port ( )

  version_path := rn.Dispatch_versions [ version ]
  executable_path := version_path + "/sbin/qdrouterd"

  r := router.New_Router ( name,
                           version,
                           router_type,
                           rn.worker_threads,
                           rn.result_path,
                           executable_path,
                           rn.config_path,
                           rn.log_path,
                           rn.dispatch_root,
                           rn.proton_root,
                           client_port,
                           console_port,
                           router_port,
                           edge_port,
                           rn.verbose,
                           rn.resource_measurement_frequency )
  rn.routers = append ( rn.routers, r )
}





/*
  Add a new router to the network. You can add all routers before
  calling Init, but it's also OK to add more after the network has 
  started. In that case, you must call Init() and Run() again.
  Routers that have already been initialized and started will not 
  be affected.
*/
func ( rn * Router_Network ) Add_router ( name string, version string ) {
  rn.add_router ( name, "interior", version )
}





/*
  Similar to Add_Router(), but adds an edge instead of an interior
  router.
*/
func ( rn * Router_Network ) Add_edge ( name string, version string ) {
  rn.add_router ( name, "edge", version )
}




func ( rn * Router_Network ) Add_receiver ( name string, n_messages int, max_message_length int, router_name string, address string ) {

  throttle := "0" // Receivers do not get throttled.
  r := rn.get_router_by_name ( router_name )

  client := client.New_client ( name,
                                "receive",
                                name,
                                r.Client_port ( ),
                                rn.client_path,
                                rn.log_path,
                                rn.dispatch_root,
                                rn.proton_root,
                                n_messages,
                                max_message_length,
                                address,
                                throttle )
  rn.clients = append ( rn.clients, client )
}





func ( rn * Router_Network ) Add_sender ( name string, n_messages int, max_message_length int, router_name string, address string, throttle string ) {
  rn.Add_client ( name, true, n_messages, max_message_length, router_name, address, throttle )
}





func ( rn * Router_Network ) Add_client ( name string, sender bool, n_messages int, max_message_length int, router_name string, address string, throttle string ) {

  var operation string
  if sender {
    operation = "send"
  } else {
    operation = "receive"
    throttle = "0" // Receivers do not get throttled.
  }

  r := rn.get_router_by_name ( router_name )

  client := client.New_client ( name,
                                operation,
                                name,
                                r.Client_port ( ),
                                rn.client_path,
                                rn.log_path,
                                rn.dispatch_root,
                                rn.proton_root,
                                n_messages,
                                max_message_length,
                                address,
                                throttle )
  rn.clients = append ( rn.clients, client )
}





func ( rn * Router_Network ) Add_n_senders ( n int, n_messages int, max_message_length int, router_name string, address string, throttle string ) {
  for i := 1; i <= n; i ++ {
    name := fmt.Sprintf ( "sender_%03d", i )
    rn.Add_client ( name, true, n_messages, max_message_length, router_name, address, throttle )
  }
}





func ( rn * Router_Network ) Halt_a_sender ( ) {
  for _, c := range rn.clients {
    if c.Is_running() && c.Operation == "send" {
      c.Halt ( )
      return
    }
  }
}





/*
  Connect the first router to the second. I.e. the first router
  will have a connector created in its config file that will 
  connect to the appropriate port of the second router.
  You cannot connect to an edge router.
*/
func ( rn * Router_Network ) Connect_router ( router_1_name string, router_2_name string ) {
  router_1 := rn.get_router_by_name ( router_1_name )
  router_2 := rn.get_router_by_name ( router_2_name )

  if router_2.Type() == "edge" {
    // A router can't connect to an edge router.
    return
  }

  connect_to_port := router_2.Router_port()
  if router_1.Type() == "edge" {
    connect_to_port = router_2.Edge_port()
  }

  // Tell router_1 whom to connect to.  ( To whom to connect? To? )
  router_1.Connect_to ( router_2_name, connect_to_port )
  // And tell router_2 who just connected to it.
  router_2.Connected_to_you ( router_1_name, "edge" == router_1.Type() )
}





/*
  Initialize the network. This is usually called once just before
  starting the network, but can also be called when the network is 
  running, after new routers have been added.
  And uninitialized routers will be initialized, i.e. their config
  files will be created, so they will be ready to start.
*/
func ( rn * Router_Network ) Init ( ) {
  for _, router := range rn.routers {
    router.Init ( )
  }
}





/*
  Start all routers in the network that are not already started.
*/
func ( rn * Router_Network ) Run ( ) {

  router_run_count := 0

  for _, r := range rn.routers {
    if r.State() == "initialized" {
      r.Run ( )
      router_run_count ++
    }
  }

  if len(rn.clients) > 0 {

    if router_run_count > 0 {
      nap_time := 10
      if rn.verbose {
        fp ( os.Stdout, 
             "network info: sleeping %d seconds to wait for network stabilization.\n", 
             nap_time )
      }
      time.Sleep ( time.Duration(nap_time) * time.Second )
    }

    upl ( "starting clients.", module_name )
    for _, c := range rn.clients {
      c.Run ( )
    }
    upl ( "Done starting clients.", module_name )
  }
}





func ( rn * Router_Network ) Client_port ( target_router_name string ) ( client_port string ) {
  r := rn.get_router_by_name ( target_router_name )
  return r.Client_port ( )
}





func ( rn * Router_Network ) Add_dispatch_version ( name, path string ) {
  rn.Dispatch_versions [ name ] = path
}





func ( rn * Router_Network ) Check_memory ( router_name string ) error {
  // set up env ----------------------------------------------
  PROTON_INSTALL_DIR    := rn.proton_root
  DISPATCH_INSTALL_DIR  := rn.dispatch_root

  DISPATCH_LIBRARY_PATH := DISPATCH_INSTALL_DIR + "/lib"
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
  args := "-m -b 0.0.0.0:" + router.Client_port ( )
  args_list := strings.Fields ( args )
  cmd := exec.Command ( rn.qdstat_path,  args_list... )
  out, _ := cmd.Output()

  fp ( os.Stderr, "\nMemory Report for router |%s| -------------\n%s\n\n\n", router_name, out )

  return nil
}





/*
  Call the qdstat tool on the named router, and confirm that all 
  of its endpoint links are up and running. 
  This is meant to let you check on an interior router to confirm 
  that all its attached edge routers are still connected.
*/
func ( rn * Router_Network ) Check_links ( router_name string ) error {
  // set up env ----------------------------------------------
  PROTON_INSTALL_DIR    := rn.proton_root
  DISPATCH_INSTALL_DIR  := rn.dispatch_root

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
  args := "-l -b 0.0.0.0:" + router.Client_port ( )
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




func halt_router ( wg * sync.WaitGroup, r * router.Router ) {
  defer wg.Done()
  err := r.Halt ( )
  if err != nil {
    upl ( "Router %s halting error: %s", module_name, r.Name(), err.Error() )
  }
}





func (rn * Router_Network) Halt_router ( router_name string ) {
  r := rn.get_router_by_name ( router_name )
  if r == nil {
    return
  }

  go r.Halt()
}





func (rn * Router_Network) Get_edge_list ( ) ( edge_list [] string){
  for _, r := range rn.routers {
    if r.Type() == "edge" {
      edge_list = append ( edge_list, r.Name() )
    }
  }

  return edge_list
}





func halt_client ( wg * sync.WaitGroup, c * client.Client ) {
  defer wg.Done()
  err := c.Halt ( )
  if err != nil && err.Error() != "process self-terminated." {
    upl ( "Client %s halting error: %s", module_name, c.Name, err.Error() )
  }
}





/*
  It takes a while to halt each router, so use a workgroup of
  goroutines to do them all in parallel.
*/
func ( rn * Router_Network ) Halt ( ) {
  var wg sync.WaitGroup

  for _, c := range rn.clients {
    wg.Add ( 1 )
    go halt_client ( & wg, c )
  }

  for _, r := range rn.routers {
    if r.Is_not_halted() {
      wg.Add ( 1 )
      go halt_router ( & wg, r )
    }
  }

  wg.Wait()
}




func ( rn * Router_Network ) Display_routers ( ) {
  for index, r := range rn.routers {
    upl ( "router %d: %s %d %s", module_name, index, r.Name(), r.Pid(), r.State() )
  }
}



func ( rn * Router_Network ) Halt_first_edge ( ) error {
  
  for _, r := range rn.routers {
    if "edge" == r.Type() {
      if r.State() == "running" {
        if rn.verbose {
          upl ( "Halt : halting router |%s|", module_name, r.Name() )
        }
        err := r.Halt ( )
        if err != nil {
          upl ( "Halt_first_edge : error halting router %s : %s", module_name, r.Name(), err.Error() )
        }
        return err
      }
    }
  }

  return errors.New ( "Router_Network.Halt_first_edge error : Could not find an edge router to halt." )
}





func ( rn * Router_Network ) get_router_by_name ( target_name string ) * router.Router {
  for _, router := range rn.routers {
    if router.Name() == target_name {
      return router
    }
  }

  return nil
}





func ( rn * Router_Network ) How_many_interior_routers ( ) ( int ) {
  count := 0

  for _, r := range rn.routers {
    if r.Is_interior() {
      count ++
    }
  }

  return count
}





func ( rn * Router_Network ) Get_nth_interior_router_name ( index int ) ( string ) {
  count := 0

  for _, r := range rn.routers {
    if r.Is_interior() {
      if count == index {
        return r.Name()
      }
      count ++
    }
  }
  return ""
}





