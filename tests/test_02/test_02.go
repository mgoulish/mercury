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



var fp = fmt.Fprintf



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

  mercury_root          := getenv ( "MERCURY_ROOT" )
  dispatch_install_root := getenv ( "DISPATCH_INSTALL_ROOT" )
  proton_install_root   := getenv ( "PROTON_INSTALL_ROOT" )

  test_name_p := flag.String ( "name",    "test_02", "the name shared by all runs of this test." )
  test_id_p   := flag.String ( "id",      "example", "the unique name for this run of the test." )
  verbose_p   := flag.Bool   ( "verbose", false,     "if true, print out debugging aids."        )
  flag.Parse ( )

  if * verbose_p {
    fp ( os.Stdout, "\n\ntest %s starting.\n", * test_name_p )
  }
  n_edges := 5

  router_path, 
  result_path, 
  config_path, 
  log_path := 
  utils.Make_paths ( mercury_root, * test_id_p, * test_name_p )

  utils.Find_or_create_dir ( result_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( log_path )

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
    fp ( os.Stdout, "  Making interior routers.\n" )
  }
  network.Add_Router ( "A" )
  network.Add_Router ( "B" )
  network.Add_Router ( "C" )

  network.Connect ( "A",  "B" )
  network.Connect ( "B",  "C" )

  network.Init ( )
  network.Run ( )
  if * verbose_p {
    fp ( os.Stdout, "  Interior router network is running.\n" )
  }

  time.Sleep ( 3 * time.Second )

  // Make edges -----------------------------------
  if * verbose_p {
    fp ( os.Stdout, "  Making edge routers.\n" )
  }
  //start_time := time.Now ( )
  for edge_count := 1; edge_count <= n_edges; edge_count ++ {
    time.Sleep ( 500 * time.Millisecond )
    edge_name := fmt.Sprintf ( "e%d", edge_count )
    network.Add_Edge   ( edge_name )
    network.Connect ( edge_name, "A" )
    network.Init ( )
    network.Run ( )
    if * verbose_p {
      fp ( os.Stdout, "    %s\n", edge_name )
    }
  }
  //stop_time := time.Now ( )
  //elapsed   := stop_time.Sub ( start_time )

  time.Sleep ( 3 * time.Second )

  // Check all the edges -----------------------------------
  if * verbose_p {
    fp ( os.Stdout, "  Checking edges.\n" )
  }
  for edge_count := 1; edge_count <= n_edges; edge_count ++ {
    edge_name := fmt.Sprintf ( "e%d", edge_count )
    if * verbose_p {
      fp ( os.Stdout, "    %s\n", edge_name )
    }
    err := network.Check_links ( edge_name )

    if err != nil {
      fp ( os.Stdout, "  Error on router |%s| : |%s|\n", edge_name, err.Error() )
    }
  }

  if * verbose_p {
    fp ( os.Stderr, "main is sleeping for 30 seconds.\n" )
  }
  time.Sleep ( 30 * time.Second )

  if * verbose_p {
    fp ( os.Stdout, "  Halting.\n" )
  }

  err := network.Halt ( )

  if err != nil {
    if * verbose_p {
      fp ( os.Stdout, "  Error.\n" )
      fp ( os.Stdout, "  Results are in |%s|\n", result_path )
      fp ( os.Stdout, "test %s complete.\n", * test_name_p )
    }
    utils.End_test_and_exit ( result_path, "error on halt: " + err.Error() )
  } 

  if * verbose_p {
    fp ( os.Stdout, "  Success.\n" );
    fp ( os.Stdout, "  Results are in |%s|\n", result_path )
    fp ( os.Stdout, "test %s complete.\n\n\n", * test_name_p )
  }

  utils.End_test_and_exit ( result_path, "" )
}





