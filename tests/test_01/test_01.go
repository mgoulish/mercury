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
    fp ( os.Stderr, "need environment variable |%s|.\n" )
    os.Exit ( 1 )
  }
  return val
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

  // TEMP
  fp ( os.Stderr, "verbose == %v\n", * verbose_p )

  router_path, 
  result_path, 
  config_path, 
  log_path := 
  utils.Make_paths ( mercury_root, * test_id_p, * test_name_p )

  router_name           := "A"
  n_worker_threads      := 4
  router_type           := "interior"

  utils.Find_or_create_dir ( result_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( log_path )

  client_port, _ := utils.Available_port ( )
  router_port, _ := utils.Available_port ( )
  edge_port,   _ := utils.Available_port ( )

  fp ( os.Stdout, "%s %s : creating router %s.\n", * test_name_p, * test_id_p, router_name )
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
                                edge_port,
                                * verbose_p )
  err := router.Init ( )
  if err != nil {
    utils.End_test_and_exit ( result_path, "error on init: " + err.Error() )
  }

  fp ( os.Stdout, "%s %s : running router %s.\n", * test_name_p, * test_id_p, router_name )
  err = router.Run ( )
  if err != nil {
    utils.End_test_and_exit ( result_path, "error on startup: " + err.Error() )
  }

  time.Sleep ( 10 * time.Second )

  err = router.Halt ( )
  if err != nil {
    fp ( os.Stdout, "%s %s : test failed.\n",   * test_name_p, * test_id_p )
    fp ( os.Stdout, "%s %s : results in %s.\n", * test_name_p, * test_id_p, result_path )
    utils.End_test_and_exit ( result_path, "error on shutdown: " + err.Error() )
  }
  fp ( os.Stdout, "%s %s : test successful.\n", * test_name_p, * test_id_p )
  fp ( os.Stdout, "%s %s : results in %s.\n",   * test_name_p, * test_id_p, result_path )

  utils.End_test_and_exit ( result_path, "" )
}





