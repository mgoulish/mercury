package main
  
import ( "fmt"
         "os"
         "utils"
         rn "router_network"
         "time"
       )



var fp = fmt.Fprintf





func
main ( ) {
  test_name             := os.Args [ 1 ]
  mercury_root          := os.Args [ 2 ]
  test_id               := os.Args [ 3 ]

  fp ( os.Stdout, "\n\ntest %s starting.\n", test_name )
  n_edges := 5

  dispatch_install_root, 
  proton_install_root, 
  router_path, 
  result_path, 
  config_path, 
  log_path := 
  utils.Make_paths ( mercury_root, test_id, test_name )

  utils.Find_or_create_dir ( result_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( log_path )

  n_worker_threads := 4

  network := rn.New_Router_Network ( test_name + "_router_network",
                                     n_worker_threads,
                                     router_path,
                                     config_path,
                                     log_path,
                                     dispatch_install_root,
                                     proton_install_root )

  fp ( os.Stdout, "  Making interior routers.\n" )
  network.Add_Router ( "A" )
  network.Add_Router ( "B" )
  network.Add_Router ( "C" )

  network.Connect ( "A",  "B" )
  network.Connect ( "B",  "C" )

  network.Init ( )
  network.Run ( )
  fp ( os.Stdout, "  Interior router network is running.\n" )

  time.Sleep ( 3 * time.Second )

  // Make edges -----------------------------------
  //start_time := time.Now ( )
  fp ( os.Stdout, "  Making edge routers.\n" )
  for edge_count := 1; edge_count <= n_edges; edge_count ++ {
    time.Sleep ( 500 * time.Millisecond )
    edge_name := fmt.Sprintf ( "e%d", edge_count )
    network.Add_Edge   ( edge_name )
    network.Connect ( edge_name, "A" )
    network.Init ( )
    network.Run ( )
    fp ( os.Stdout, "    %s\n", edge_name )
  }
  //stop_time := time.Now ( )
  //elapsed   := stop_time.Sub ( start_time )

  time.Sleep ( 3 * time.Second )

  // Check all the edges -----------------------------------
  fp ( os.Stdout, "  Checking edges.\n" )
  for edge_count := 1; edge_count <= n_edges; edge_count ++ {
    edge_name := fmt.Sprintf ( "e%d", edge_count )
    fp ( os.Stdout, "    %s\n", edge_name )
    err := network.Check_Links ( edge_name )

    if err != nil {
      fp ( os.Stdout, "  Error on router |%s| : |%s|\n", edge_name, err.Error() )
    }
  }

  fp ( os.Stdout, "  Halting.\n" )
  err := network.Halt ( )
  if err != nil {
    fp ( os.Stdout, "  Error.\n" )
    fp ( os.Stdout, "  Results are in |%s|\n", result_path )
    fp ( os.Stdout, "test %s complete.\n", test_name )
    utils.End_test_and_exit ( result_path, "error on halt: " + err.Error() )
  } 

  fp ( os.Stdout, "  Success.\n" );
  fp ( os.Stdout, "  Results are in |%s|\n", result_path )
  fp ( os.Stdout, "test %s complete.\n\n\n", test_name )
  utils.End_test_and_exit ( result_path, "" )

}





