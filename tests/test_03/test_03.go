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
         "utils"
         rn "router_network"
         "time"
         "flag"
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
  n_edges := 100

  // Make paths and create directories ----------------------
  router_path, 
  result_path, 
  config_path, 
  log_path := 
  utils.Make_paths ( mercury_root, * test_id_p, * test_name_p )

  utils.Find_or_create_dir ( result_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( log_path )


  // Make the network ----------------------------------------
  n_worker_threads := 4
  resource_measurement_frequency := 30

  network := rn.New_Router_Network ( * test_name_p + "_router_network",
                                     n_worker_threads,
                                     result_path,
                                     router_path,
                                     config_path,
                                     log_path,
                                     dispatch_install_root,
                                     proton_install_root,
                                     * verbose_p,
                                     resource_measurement_frequency )

  if * verbose_p {
    upl ( "Making interior routers", * test_name_p )
  }
  network.Add_Router ( "A" )
  network.Add_Router ( "B" )
  network.Add_Router ( "C" )

  network.Connect ( "A",  "B" )
  network.Connect ( "B",  "C" )

  network.Init ( )
  network.Run ( )
  if * verbose_p {
    upl ( "Interior router network is running", * test_name_p )
  }

  time.Sleep ( 3 * time.Second )




  if * verbose_p {
    upl ( "Making edge routers", * test_name_p )
  }
  total_edges_so_far := 0
  make_edges ( total_edges_so_far, n_edges, network, * verbose_p, * test_name_p )
  total_edges_so_far += n_edges


  /*
    In each iteration, kill one edge router and add a 
    new one, with small polite pauses after each evennt.
  */
  for iteration := 1; iteration < 25; iteration ++ {

    network.Halt_first_edge ( )

    time.Sleep ( 3 * time.Second )

    /*
      In starting the replacement router, we use the total 
      count of edged so far as the starting number. This means 
      that the name of a given edge will never repeat, but we 
      should always have n_edges alive after each pass through 
      this loop.
    */
    make_edges ( total_edges_so_far, 1, network, * verbose_p, * test_name_p )
    time.Sleep ( 3 * time.Second )

    total_edges_so_far ++
  }


  if * verbose_p {
    upl ( "main is sleeping for 15 seconds", * test_name_p )
  }
  time.Sleep ( 15 * time.Second )

  if * verbose_p {
    upl ( "halting", * test_name_p )
  }
  network.Halt ( )

  test_stop_time := time.Now()

  test_duration := test_stop_time.Sub ( test_start_time )
  if * verbose_p {
    upl ( "total test time: %.3f", * test_name_p, test_duration.Seconds() )
  }

  if * verbose_p {
    upl ( "Results are in |%s|", * test_name_p, result_path )
    upl ( "test complete", * test_name_p )
  }

  utils.End_test_and_exit ( result_path, "" )
}





func make_edges ( start_edge_number, n_edges int, network * rn.Router_Network, verbose bool, test_name string ) {
  last_edge_number := start_edge_number + n_edges
  for edge_count := start_edge_number; edge_count < last_edge_number; edge_count ++ {
    time.Sleep ( 100 * time.Millisecond )
    edge_name := fmt.Sprintf ( "e%d", edge_count )
    if verbose {
      upl ( "make_edges making new router with name |%s|", test_name, edge_name )
    }
    network.Add_Edge   ( edge_name )
    network.Connect ( edge_name, "A" )
    network.Init ( )
    network.Run ( )
  }
}





