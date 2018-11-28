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
  test_01 just starts a router, waits for a few seconds, and
  shuts it down again. If the router has not already died on
  its own, then the test succeeds.
*/
package main
  
import ( "fmt"
         "os"
         "router"
         "time"
         "utils"
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





/*
  Create the paths that are derivable from the Mercury root 
  and test name/ID and create the necessary directories.
*/
func do_paths ( mercury_root, test_name, test_id string ) ( router_path, result_path, config_path, log_path string ) {

  router_path,
  result_path,
  config_path,
  log_path =
  utils.Make_paths ( mercury_root, test_id, test_name )

  utils.Find_or_create_dir ( result_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( log_path )
  
  return router_path, result_path, config_path, log_path
}





func
main ( ) {
  mercury_root          := getenv ( "MERCURY_ROOT" )
  dispatch_install_root := getenv ( "DISPATCH_INSTALL_ROOT" )
  proton_install_root   := getenv ( "PROTON_INSTALL_ROOT" )

  test_name_p := flag.String ( "name",    "test_01", "the name shared by all runs of this test." )
  test_id_p   := flag.String ( "id",      "example", "the unique name for this run of the test." )
  verbose_p   := flag.Bool   ( "verbose", false,     "if true, print out debugging aids."        )
  flag.Parse ( )

  router_path, 
  result_path, 
  config_path, 
  log_path := 
  do_paths ( mercury_root, * test_name_p, * test_id_p )

  router_name           := "A"
  n_worker_threads      := 4
  router_type           := "interior"

  client_port, _ := utils.Available_port ( )
  router_port, _ := utils.Available_port ( )
  edge_port,   _ := utils.Available_port ( )

  if * verbose_p {
    fp ( os.Stdout, "%s %s : creating router %s.\n", * test_name_p, * test_id_p, router_name )
  }

  usage_measurement_frequency := 10

  router := router.New_Router ( router_name,
                                router_type,
                                n_worker_threads, 
                                result_path,
                                router_path,
                                config_path,
                                log_path,
                                dispatch_install_root,
                                proton_install_root,
                                client_port,
                                router_port,
                                edge_port,
                                * verbose_p,
                                usage_measurement_frequency )
  err := router.Init ( )
  if err != nil {
    utils.End_test_and_exit ( result_path, "error on init: " + err.Error() )
  }

  fp ( os.Stdout, "%s %s : running router %s.\n", * test_name_p, * test_id_p, router_name )
  err = router.Run ( )
  if err != nil {
    utils.End_test_and_exit ( result_path, "error on startup: " + err.Error() )
  }

  time.Sleep ( 180 * time.Second )

  err = router.Halt ( )
  if err != nil {
    fp ( os.Stdout, "%s %s : test failed.\n",   * test_name_p, * test_id_p )
    fp ( os.Stdout, "%s %s : results in %s .\n", * test_name_p, * test_id_p, result_path )
    utils.End_test_and_exit ( result_path, "error on shutdown: " + err.Error() )
  }
  fp ( os.Stdout, "%s %s : test successful.\n", * test_name_p, * test_id_p )
  fp ( os.Stdout, "%s %s : results in %s .\n",   * test_name_p, * test_id_p, result_path )

  utils.End_test_and_exit ( result_path, "" )
}





