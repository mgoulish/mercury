package main

import (
            "fmt"
            "os"
            "time"

         rn "router_network"
            "utils"
       )


var fp=fmt.Fprintf





func run_test ( test_name string, mercury_root string, n_routers int ) ( string )  {

  log_path    := test_name + "/" + test_name + "/log"
  config_path := test_name + "/" + test_name + "/config"
  event_path  := test_name + "/" + test_name + "/event"
  result_path := test_name + "/" + test_name + "/result"

  utils.Find_or_create_dir ( log_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( event_path )
  utils.Find_or_create_dir ( result_path )

  network := rn.New_router_network ( test_name,
                                     mercury_root,
                                     log_path )

  // TODO fix this!
  network.Add_version_with_roots ( "latest",
                                   "/home/mick/latest/install/proton",
                                   "/home/mick/latest/install/dispatch" )

  // Make the routers -------------------------------
  for i := 0; i < n_routers; i ++ {
    router_name := fmt.Sprintf ( "%X", i )
    network.Add_router ( router_name,
                         "latest",
                         config_path,
                         log_path )
    fp ( os.Stdout, "Add router |%s|\n", router_name )
  }

  // Connect them  ----------------------------------
  for i := 0; i < n_routers; i ++ {
    current_router_name := fmt.Sprintf ( "%X", i )
    for j := i + 1; j < n_routers; j ++ {
      connect_to_router_name := fmt.Sprintf ( "%X", j )
      network.Connect_router ( current_router_name, connect_to_router_name )
      fp ( os.Stdout, "Connect |%s| to |%s|\n", current_router_name, connect_to_router_name )
    }
  }


  network.Init ( )
  network.Set_results_path ( result_path )
  network.Set_events_path  ( event_path )

  network.Run  ( )
  fp ( os.Stdout, "network |%s| is running.\n", test_name )

  for {
    time.Sleep ( 5 * time.Second )
  }

  return result_path
}





func main ( ) {

  mercury_root := os.Getenv ( "MERCURY_ROOT" )
  test_name := "mesh" + "_" + time.Now().Format ( "2006_01_02_1504" )

  run_test ( test_name, mercury_root, 3 )

  fp ( os.Stdout, "Test %s done at %s\n", test_name, time.Now().Format ( "2006_01_02_1504" ) )
}





