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
  test_02 starts a linear network of 3 routers, adds several edge 
  routers to 1 of them, and then checks those edge routers to make
  sure that all of their links are up and running. If they are,
  the test succeeds.
*/

package main
  
import ( "fmt"
         "os"
         "time"
         "flag"
         "utils"
         rn "router_network"
       )



var upl = utils.Print_log
var fp  = fmt.Fprintf





func getenv ( key string ) string {
  val := os.Getenv ( key )
  if val == "" {
    fp ( os.Stderr, "need environment variable |%s|.\n", key )
    os.Exit ( 1 )
  }
  return val
}





func
main ( ) {

  test_start_time := time.Now()

  // Get environment variables ----------------------------------
  mercury_root          := getenv ( "MERCURY_ROOT" )
  dispatch_install_root := getenv ( "DISPATCH_INSTALL_ROOT" )
  proton_install_root   := getenv ( "PROTON_INSTALL_ROOT" )

  // Get command line flags ----------------------------------
  test_name_p := flag.String ( "name",    "test_03", "the name shared by all runs of this test." )
  test_id_p   := flag.String ( "id",      "example", "the unique name for this run of the test." )
  verbose_p   := flag.Bool   ( "verbose", false,     "if true, print out debugging aids."        )
  flag.Parse ( )

  if * verbose_p {
    //fp ( os.Stdout, "\n\ntest %s starting.\n", * test_name_p )
    upl ( "starting", * test_name_p )    
  }

  // Make paths and create directories ----------------------
  router_path, 
  result_path, 
  config_path, 
  log_path := 
  utils.Make_paths ( mercury_root, * test_id_p, * test_name_p )

  utils.Find_or_create_dir ( result_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( log_path )

  client_path := mercury_root + "/clients/c_proactor_client"

  /*-------------------------------------------
     Make the network 
      A --- B --- C 
  -------------------------------------------*/
  n_worker_threads := 4
  resource_measurement_frequency := 0   // turn it off

  network := rn.New_Router_Network ( * test_name_p + "_router_network",
                                     n_worker_threads,
                                     result_path,
                                     router_path,
                                     config_path,
                                     log_path,
                                     client_path,
                                     dispatch_install_root,
                                     proton_install_root,
                                     * verbose_p,
                                     resource_measurement_frequency )

  if * verbose_p {
    upl ( "Making interior routers", * test_name_p )
  }
  network.Add_router ( "A" )
  network.Add_router ( "B" )
  network.Add_router ( "C" )
  network.Add_router ( "D" )

  network.Connect_router ( "A",  "B" )
  network.Connect_router ( "B",  "C" )
  network.Connect_router ( "C",  "D" )

  use_edge := false


  var edge_name          string
  var sender_router_name string

  if use_edge {
    edge_name = "edge_01"
    network.Add_edge ( edge_name )
    network.Connect_router ( edge_name, "A" )
    sender_router_name = edge_name
  } else {
    sender_router_name = "A"
  }



  n_senders           := 1000
  messages_per_sender := 100
  max_message_length  := 100000

  network.Add_client ( "receiver_1", false, n_senders * messages_per_sender, max_message_length, "D" )

  network.Add_n_senders ( n_senders, messages_per_sender, max_message_length, sender_router_name )

  network.Init ( )
  network.Run ( )


  /*
  nap_time := 200
  upl ( "sleeping %d seconds.", * test_name_p, nap_time )
  time.Sleep ( time.Duration(nap_time) * time.Second )

  upl ( "sleeping 100 seconds.", * test_name_p )
  time.Sleep ( 100 * time.Second )

  n_sleeps := 50
  nap_size := 2
  for i := 0; i < n_sleeps; i ++ {
    time.Sleep ( time.Duration(nap_size) * time.Second )
    upl ( "%d : halting a sender.", * test_name_p, n_sleeps - i )
    network.Halt_a_sender ( )
  }
  */

  upl ( "sleeping 500 seconds.", * test_name_p )
  time.Sleep ( 500 * time.Second )

  upl ( "Shutting down network.", * test_name_p )
  network.Halt ( )

  test_stop_time := time.Now()

  test_duration := test_stop_time.Sub ( test_start_time )
  if * verbose_p {
    upl ( "total test time: %.3f", * test_name_p, test_duration.Seconds() )
    upl ( "Results are in |%s|", * test_name_p, result_path )
    upl ( "test complete", * test_name_p )
  }

  utils.End_test_and_exit ( result_path, "" )
}





