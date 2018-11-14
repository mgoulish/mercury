package main
  
import ( "fmt"
         "os"
         "router"
         "time"
         "utils"
       )


var fp = fmt.Fprintf





func
main ( ) {
  mercury_root          := os.Args [ 1 ]
  test_id               := os.Args [ 2 ]
  test_name             := "test_01"

  dispatch_install_root, 
  proton_install_root, 
  router_path, 
  result_path, 
  config_path, 
  log_path := 
  utils.Make_paths ( mercury_root, test_id, test_name )

  router_name           := "A"
  n_worker_threads      := 4
  router_type           := "interior"

  utils.Find_or_create_dir ( result_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( log_path )

  client_port, _ := utils.Available_port ( )
  router_port, _ := utils.Available_port ( )
  edge_port,   _ := utils.Available_port ( )

  fp ( os.Stdout, "%s: creating router %s.\n", test_name, router_name )
  router := router.New_Router ( router_name,
                                router_type,
                                n_worker_threads, 
                                router_path,
                                config_path,
                                log_path,
                                dispatch_install_root,
                                proton_install_root,
                                client_port,
                                router_port,
                                edge_port )
  err := router.Init ( )
  if err != nil {
    utils.End_test_and_exit ( result_path, "error on init: " + err.Error() )
  }

  fp ( os.Stdout, "%s: running router %s.\n", test_name, router_name )
  err = router.Run ( )
  if err != nil {
    utils.End_test_and_exit ( result_path, "error on startup: " + err.Error() )
  }

  time.Sleep ( 10 * time.Second )

  err = router.Halt ( )
  if err != nil {
    fp ( os.Stdout, "%s: test failed.\n", test_name )
    utils.End_test_and_exit ( result_path, "error on shutdown: " + err.Error() )
  }
  fp ( os.Stdout, "%s: test successful.\n", test_name )
  fp ( os.Stdout, "%s: results in %s.\n", test_name, result_path )

  utils.End_test_and_exit ( result_path, "" )
}





