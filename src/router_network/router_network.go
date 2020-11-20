package router_network

import ( "bufio"
         "errors"
         "fmt"
         "os"
         "os/exec"
         "strings"
         "math/rand"
         "strconv"
         "sync"
         "time"

         "client"
         "router"
         "utils"
       )


var fp          = fmt.Fprintf
var module_name = "router_network"
var ume         = utils.M_error
var umi         = utils.M_info





type Version struct {
  Name            string

  dispatch_root   string
  proton_root     string

  Router_path     string
  Pythonpath      string
  Ld_library_path string
  Console_path    string
  Include_path    string
}




func ( v * Version ) Print_version ( ) {
  fp ( os.Stdout, "Version -----------------------------\n" )
  fp ( os.Stdout, "  Name             |%s|\n", v.Name             )
  fp ( os.Stdout, "  dispatch_root    |%s|\n", v.dispatch_root    )
  fp ( os.Stdout, "  proton_root      |%s|\n", v.proton_root      )
  fp ( os.Stdout, "  Router_path      |%s|\n", v.Router_path      )
  fp ( os.Stdout, "  Pythonpath       |%s|\n", v.Pythonpath       )
  fp ( os.Stdout, "  Ld_library_path  |%s|\n", v.Ld_library_path  )
  fp ( os.Stdout, "  Console_path     |%s|\n", v.Console_path     )
  fp ( os.Stdout, "  Include_path     |%s|\n", v.Include_path     )
  fp ( os.Stdout, "----------------------------- end Version\n" )
}




func check_path ( name string, path string, must_exist bool ) {
  if ! utils.Path_exists ( path ) {
    if must_exist {
      ume ( "rn.check_path: Path |%s| does not exist at |%s|", name, path )
      // os.Exit ( 1 )
      ume ( "rn.check_path: But I'm not exiting...\n" )
    } else {
      //umi ( true, "rn.check_path: Path |%s| does not exist at |%s|", name, path )
    }
  }
}





/*===================================================================

  The purpose of the two Version constructors is to allow you to 
  construct a Version the easy way -- with just two install paths --
  or the hard way, by supplying all paths yourself. 

  This is because some environments are not set up as my code 
  expects them to be, and you may need the flexibility of the 
  explicit paths method.

  You call these indirectly, by calling one of 
    Add_version_with_roots(), or
    Add_version_with_paths().
    
-===================================================================*/

func new_version_with_roots ( name          string,
                              dispatch_root string,
                              proton_root   string,
                              verbose       bool ) * Version {

  // First make sure that what the caller gave us is real.
  check_path ( "dispatch root", dispatch_root, true )
  check_path ( "proton root",   proton_root,   true )

  v := & Version { Name          : name,
                   dispatch_root : dispatch_root,
                   proton_root   : proton_root } 
  v.Router_path = dispatch_root + "/sbin/qdrouterd"
  check_path ( "router path", v.Router_path, true )

  // Calculate LD_LIBRARY_PATH for this version.
  DISPATCH_LIBRARY_PATH := v.dispatch_root + "/lib"
  PROTON_LIBRARY_PATH   := v.proton_root   + "/lib64"
  v.Ld_library_path      = DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH
  check_path ( "dispatch library path", DISPATCH_LIBRARY_PATH, true )
  check_path (   "proton library path",   PROTON_LIBRARY_PATH, true )


  // TODO -- fix this Python path.
  // Calculate PYTHONPATH for this version.
  DISPATCH_PYTHONPATH   := DISPATCH_LIBRARY_PATH + "/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := DISPATCH_LIBRARY_PATH + "/python2.7/site-packages"
  PROTON_PYTHONPATH     := PROTON_LIBRARY_PATH   + "/proton/bindings/python"
  check_path ( "dispatch python path",  DISPATCH_PYTHONPATH,  true )
  check_path ( "dispatch pythonpath 2", DISPATCH_PYTHONPATH2, true )
  check_path ( "proton python path",    PROTON_PYTHONPATH,    true )

  v.Pythonpath          =  DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2 +":"+ PROTON_LIBRARY_PATH +":"+ PROTON_PYTHONPATH
  v.Console_path        =  v.dispatch_root + "/share/qpid-dispatch/console"
  v.Include_path        =  v.dispatch_root + "/lib/qpid-dispatch/python"
  check_path ( "include path", v.Include_path, true )
  check_path ( "console path", v.Console_path, false ) // The console is an optional install.

  umi ( verbose, "router path     |%s|", v.Router_path )
  umi ( verbose, "ld library path |%s|", v.Ld_library_path )
  umi ( verbose, "python path     |%s|", v.Pythonpath )
  umi ( verbose, "include path    |%s|", v.Include_path )
  umi ( verbose, "console path    |%s|", v.Console_path )

  return v
}





type Router_network struct {
  Name                        string
  Running                     bool

  results_path                string
  events_path                 string
  log_path                    string

  mercury_root                string

  /*
    The Network, rather than the Version has
    the client path, because the client comes
    from the Mercury install, not from the Dispatch
    or Proton installs, which are contained in Version.
  */
  client_names           []   string

  client_dir                  string

  Versions               [] * Version
  Default_version           * Version

  verbose                     bool

  routers                [] * router.Router
  clients                [] * client.Client

  ticker_frequency            int
  client_ticker             * time.Ticker
  router_ticker             * time.Ticker
  completed_clients           int

  n_senders                   int

  init_only                   bool

  Router_PIDs            []   int
  previous_idle_time, previous_total_time uint64
}





func ( rn * Router_network ) Kill_and_restart_random_client ( ) {
  n_clients := len ( rn.clients )
  if n_clients <= 0 {
    ume ( "router_network.Kill_and_restart_random_client error: no clients.\n" )
    return
  }

  client_number := rand.Intn ( n_clients )
  fp ( os.Stdout,  "Kill_and_restart_random_client: %d\n", client_number )

  client := rn.clients [ client_number ]
  client.Kill_and_restart ( 15 )
}





func ( rn * Router_network ) First_router_name ( ) ( string ) {
   fp ( os.Stdout, "n_routers: %d\n", len ( rn.routers ) )
  return rn.routers[0].Name()
}





func ( rn * Router_network ) Last_router_name ( ) ( string ) {
  n_routers := len(rn.routers)-1
  return rn.routers [ n_routers ].Name()
}





func ( rn * Router_network ) Init_only ( val bool ) {
  rn.init_only = val
}





func ( rn * Router_network ) Reset ( ) {
  rn.Running         = false
  rn.Versions        = nil
  rn.Default_version = nil
  rn.verbose         = false
  rn.routers         = nil
  rn.clients         = nil
  rn.n_senders       = 0
  rn.init_only       = false
}





func ( rn * Router_network ) get_client_by_name ( target_name string ) ( * client.Client ) {
  for _, c := range rn.clients {
    if target_name == c.Name {
      return c
    }
  }
  return nil
}





func ( rn * Router_network )  Build_clients ( ) {

  rn.client_dir = rn.mercury_root + "/clients"

  // TODO -- make it discover the client source files -- not hallucinate them.
  rn.client_names = append ( rn.client_names, "c_proactor_client" )
  // rn.client_names = append ( rn.client_names, "chaos_links_1" )


  for _, client_name := range ( rn.client_names ) {

    client_path := rn.client_dir + "/" + client_name

    if ! utils.Path_exists ( client_path  ) {
      fp ( os.Stdout, "\nClient |%s| does not exist. Building...\n", client_name )
      proton_include_path := rn.Default_version.proton_root + "/include"
      proton_link_path    := rn.Default_version.proton_root + "/lib64"

      // Compile --------------------------------------------------------
      command_name := "g++"
      args         := "-fpermissive -O3 -I" + proton_include_path + " -c " + client_name + ".c"
      args_list    := strings.Fields ( args )
      cmd          := exec.Command ( command_name, args_list... )
      cmd.Dir       = rn.client_dir
      out, err     := cmd.Output ( )
      if err != nil {
        fp ( os.Stderr, "Build_clients error : Can't compile c_proactor_client. |%s|\n", err.Error() )
        fp ( os.Stderr, "  command output: |%s|\n", out )
        os.Exit ( 1 )
      }

      // Link --------------------------------------------------------
      command_name = "g++"
      // args         = "-o c_proactor_client -L" + proton_link_path + " c_proactor_client.o -lqpid-proton -lpthread"
      args         = "-o " + client_name + " -L" + proton_link_path + " " + client_name + ".o -lqpid-proton -lpthread"
      args_list    = strings.Fields ( args )
      cmd          = exec.Command ( command_name, args_list... )
      cmd.Dir      = rn.client_dir
      out, err     = cmd.Output ( )
      if err != nil {
        fp ( os.Stderr, "Build_clients error : Can't link c_proactor_client. |%s|\n", err.Error() )
        fp ( os.Stderr, "  command output: |%s|\n", out )
        os.Exit ( 1 )
      }

      fp ( os.Stdout, "Build_clients: built client at path |%s|.\n\n", client_path )
    }
  }

}





// Create a new router network.
// Tell it how many worker threads each router should have,
// and provide lots of paths.
func New_router_network ( name         string,
                          mercury_root string,
                          log_path     string ) * Router_network {

  rn := & Router_network { Name         : name,
                           log_path     : log_path,
                           mercury_root : mercury_root }
  rn.ticker_frequency = 10

  return rn
}





func ( rn * Router_network ) Set_results_path ( path string ) {
  rn.results_path = path
  utils.Find_or_create_dir ( rn.results_path )
}





func ( rn * Router_network ) Set_events_path ( path string ) {
  rn.events_path = path
  utils.Find_or_create_dir ( rn.events_path )
}





func ( rn * Router_network ) Add_version_with_roots ( name          string,
                                                      proton_root   string,
                                                      dispatch_root string ) {

  version := new_version_with_roots ( name, dispatch_root, proton_root, rn.verbose )
  rn.Versions = append ( rn.Versions, version )

  // fp ( os.Stdout,  "Added version |%s|.\n", name )
  // version.Print_version ( )

  // If this is the first one, make it the default.
  // And build the clients using this version.
  if 1 == len ( rn.Versions ) {
    rn.Default_version = version
    rn.Build_clients ( )
  }
}





func ( rn * Router_network ) Get_n_routers ( ) ( int ) {
  return len ( rn.routers )
}





func ( rn * Router_network ) Get_n_interior_routers ( ) ( int ) {
  count := 0
  for _, r := range rn.routers {
    if r.Type() == "interior" {
      count ++
    }
  }
  return count
}


func ( rn * Router_network ) Get_interior_routers_names ( ) ( [] string ) {
  var interior_router_names [] string
  for _, r := range rn.routers {
    if r.Type() == "interior" { 
      interior_router_names = append ( interior_router_names, r.Name() )
    }
  }

  return interior_router_names
}





func ( rn * Router_network ) Get_router_log_file_paths ( router_names [] string )  ( [] string ) {
  var log_file_names [] string
  for _, r_name := range router_names {
    r := rn.get_router_by_name ( r_name )
    log_file_names = append ( log_file_names, r.Log_file_path )
  }

  return log_file_names
}





func ( rn * Router_network ) Get_version_from_name ( target_name string ) ( * Version ) {
  for _, v := range rn.Versions {
    if v.Name == target_name {
      return v
    }
  }
  return nil
}





func ( rn * Router_network ) Get_router_edges ( router_name string ) ( [] string ) {
  rtr := rn.get_router_by_name ( router_name )
  if rtr == nil {
    fp ( os.Stdout, "    network.Get_router_edges error: can't find router |%s|\n", router_name )
    return nil
  }

  return rtr.Edges ( )
}





func ( rn * Router_network ) Print_console_ports ( ) {
  for _, r := range rn.routers {
    r.Print_console_port ( )
  }
}





func ( rn * Router_network ) add_router ( name         string, 
                                          router_type  string, 
                                          version_name string,
                                          config_path  string,
                                          log_path     string ) {
/*
  Some paths are related to the current session. They get passed
  in here directly as args. The others are related to the version
  of the router code we are using, and they come in as part of the
  version structure.
*/
  var console_port string
  if name == "A" {
    console_port = "5673"
  } else {
    console_port, _ = utils.Available_port ( )
  }

  client_port, _  := utils.Available_port ( )
  router_port, _  := utils.Available_port ( )
  edge_port, _    := utils.Available_port ( )

  version := rn.Get_version_from_name ( version_name )

  // TODO -- pass this down from on high
  worker_threads := 30

  r := router.New_Router ( name,
                           version.Name,
                           router_type,
                           worker_threads,
                           version.Router_path,
                           config_path,
                           log_path,
                           version.Include_path,
                           version.Console_path,
                           version.Ld_library_path,
                           version.Pythonpath,
                           client_port,
                           console_port,
                           router_port,
                           edge_port,
                           rn.verbose )
  rn.routers = append ( rn.routers, r )
}





func ( rn * Router_network ) Verbose ( val bool ) {
  rn.verbose = val
  for _, r := range rn.routers {
    r.Verbose ( val )
  }
}





// Add a new router to the network. You can add all routers before
// calling Init, but it's also OK to add more after the network has 
// started. In that case, you must call Init() and Run() again.
// Routers that have already been initialized and started will not 
// be affected.
func ( rn * Router_network ) Add_router ( name         string, 
                                          version_name string,
                                          config_path  string,
                                          log_path     string ) {

  rn.add_router ( name, 
                  "interior", 
                  version_name, 
                  config_path, 
                  log_path )
}





/*
  Similar to Add_Router(), but adds an edge instead of an interior
  router.
*/
func ( rn * Router_network ) Add_edge ( name         string, 
                                        version_name string,
                                        config_path  string,
                                        log_path     string ) {
  rn.add_router ( name, 
                  "edge", 
                  version_name,
                  config_path, 
                  log_path )
}




func ( rn * Router_network ) Add_receiver ( name               string, 
                                            config_path        string,
                                            n_messages         int, 
                                            max_message_length int, 
                                            router_name        string,
                                            delay              string,
                                            soak               string ) {

  throttle := "0" // Receivers do not get throttled.

  rn.add_client ( name, 
                  config_path,
                  rn.results_path,
                  rn.events_path,
                  false, 
                  n_messages, 
                  max_message_length, 
                  router_name, 
                  throttle,
                  delay,
                  soak )
}





func ( rn * Router_network ) Add_sender ( name               string, 
                                          config_path        string,
                                          n_messages         int, 
                                          max_message_length int, 
                                          router_name        string, 
                                          throttle           string,
                                          delay              string,
                                          soak               string ) {
  rn.add_client ( name, 
                  config_path,
                  rn.results_path,
                  rn.events_path,
                  true, 
                  n_messages, 
                  max_message_length, 
                  router_name, 
                  throttle,
                  delay,
                  soak )
}





func ( rn * Router_network ) add_client ( name               string, 
                                          config_path        string,
                                          results_path       string,
                                          events_path        string,
                                          sender             bool, 
                                          n_messages         int, 
                                          max_message_length int, 
                                          router_name        string, 
                                          throttle           string,
                                          delay              string,
                                          soak               string ) {


  var operation string
  if sender {
    operation = "send"
    rn.n_senders ++
  } else {
    operation = "receive"
    throttle = "0" // Receivers do not get throttled.
  }

  r := rn.get_router_by_name ( router_name )

  if r == nil {
    ume ( "Network: add_client: no such router: |%s|", router_name )
    return
  }

  // Clients just use the default version.
  ld_library_path := rn.Default_version.Ld_library_path
  pythonpath      := rn.Default_version.Pythonpath

  status_file := rn.log_path + "/" + name

  c := client.New_client ( name,
                           config_path,
                           results_path,
                           events_path,
                           operation,
                           r.Client_port ( ),
                           rn.client_dir + "/" + rn.client_names[0],
                           ld_library_path,
                           pythonpath,
                           status_file,
                           n_messages,
                           max_message_length,
                           throttle,
                           rn.verbose,
                           delay,
                           soak )

  rn.clients = append ( rn.clients, c )
}





func ( rn * Router_network ) Get_Client_By_Name ( target_name string ) ( * client.Client )  {
  for _, c := range rn.clients {
    if target_name == c.Name {
      return c
    }
  }
  return nil
}





func ( rn * Router_network ) Add_Address_To_Client ( client_name string,
                                                     addr        string ) {
  c := rn.Get_Client_By_Name ( client_name )
  if c == nil {
    ume ( "router_network: can't find client |%s|", client_name )
    return
  }

  c.Add_Address ( addr )
}





/*
  Connect the first router to the second. I.e. the first router
  will have a connector created in its config file that will 
  connect to the appropriate port of the second router.
  You cannot connect to an edge router.
*/
func ( rn * Router_network ) Connect_router ( router_1_name string, router_2_name string ) {
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


func ( rn * Router_network ) Are_connected ( router_1_name string, router_2_name string ) ( bool ) {

  router_1 := rn.get_router_by_name ( router_1_name )
  return router_1.Is_connected_to ( router_2_name )
}





/*
  Initialize the network. This is usually called once just before
  starting the network, but can also be called when the network is 
  running, after new routers have been added.
  And uninitialized routers will be initialized, i.e. their config
  files will be created, so they will be ready to start.
*/
func ( rn * Router_network ) Init ( ) {
  for _, router := range rn.routers {
    router.Init ( )
  }
  
  umi ( rn.verbose, "Network is initialized." )
  if rn.init_only {
    umi ( rn.verbose, "Init only is set : halting." )
    os.Exit ( 0 )
  }
}





/*
  Start all routers in the network that are not already started.
*/
func ( rn * Router_network ) Run ( ) {

  fp ( os.Stdout, "MDEBUG starting network !\n" )
  router_run_count := 0

  for _, r := range rn.routers {
    if r.State() == "initialized" {
      pid, _ := r.Run ( )
      router_run_count ++
      rn.Router_PIDs = append ( rn.Router_PIDs, pid )
    }
  }

  // TODO -- figure ut how to read cpu
  // go rn.Router_status_check ( )

  if len(rn.clients) > 0 {
    if router_run_count > 0 {
      nap_time := 5
      if rn.verbose {
        fp ( os.Stdout, 
             "    network info: sleeping %d seconds to wait for network stabilization.\n", 
             nap_time )
      }
      time.Sleep ( time.Duration(nap_time) * time.Second )
    }

    count := 0
    for _, c := range rn.clients {
      c.Run ( )

      // TODO replace this with more intelligence in senders.
      // Inter-client sleep 
      time.Sleep ( 50 * time.Millisecond )
      count ++
      if 0 == (count % 20) {
        fp ( os.Stdout, "MDEBUG started %d clients.\n", count )
      }
    }
  }

  rn.Running = true
}





func ( rn * Router_network ) Router_status_check ( ) ( ) {

  fp ( os.Stdout, "MDEBUG in Router_status_check!\n" )
  for {
    time.Sleep ( 5 * time.Second )
    fp ( os.Stdout, "MDEBUG check %d routers.\n", len(rn.Router_PIDs) )
    for _, pid := range rn.Router_PIDs {
      fp ( os.Stdout, "MDEBUG Router status check for PID %d\n", pid )
    }
    fp ( os.Stdout, "MDEBUG Read CPU: \n" )
    rn.read_CPU ( )
  }
}





func ( rn * Router_network ) Client_port ( target_router_name string ) ( client_port string ) {
  r := rn.get_router_by_name ( target_router_name )
  return r.Client_port ( )
}





func halt_router ( wg * sync.WaitGroup, r * router.Router ) {
  defer wg.Done()
  err := r.Halt ( )
  if err != nil {
    ume ( "Router %s halting error: %s", r.Name(), err.Error() )
  }
}





func (rn * Router_network) Halt_router ( router_name string ) ( error ) {
  r := rn.get_router_by_name ( router_name )
  if r == nil {
    return errors.New ( "No such router." )
  }

  go r.Halt()
  return nil
}





func (rn * Router_network) Halt_and_restart_router ( router_name string, pause int ) ( error ) {
  r := rn.get_router_by_name ( router_name )
  if r == nil {
    return errors.New ( "No such router." )
  }

  r.Halt()
  if rn.verbose {
    umi ( rn.verbose, "Halt_and_restart_router: Pausing %d seconds.", pause )
  }
  time.Sleep ( time.Duration(pause) * time.Second )

  r.Run ( )

  return nil
}





func (rn * Router_network) Get_edge_list ( ) ( edge_list [] string) {
  for _, r := range rn.routers {
    if r.Type() == "edge" {
      edge_list = append ( edge_list, r.Name() )
    }
  }

  return edge_list
}





func halt_client ( wg * sync.WaitGroup, c * client.Client ) {
  defer wg.Done()

  /*
   This looks like it is not actually an error.
  err := c.Halt ( )
  if err != nil && err.Error() != "process self-terminated." {
    ume ( "Client |%s| halting error: %s", c.Name, err.Error() )
  }
  */
  c.Halt ( )
}





/*
  It takes a while to halt each router, so use a workgroup of
  goroutines to do them all in parallel.
*/
func ( rn * Router_network ) Halt ( ) {
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
  rn.Running = false
}




func ( rn * Router_network ) Display_routers ( ) {
  for index, r := range rn.routers {
    umi ( rn.verbose, "router %d: %s %d %s", index, r.Name(), r.Pid, r.State() )
  }
}



func ( rn * Router_network ) Halt_first_edge ( ) error {
  
  for _, r := range rn.routers {
    if "edge" == r.Type() {
      if r.State() == "running" {
        if rn.verbose {
          umi ( rn.verbose, "halting router |%s|", r.Name() )
        }
        err := r.Halt ( )
        if err != nil {
          umi ( rn.verbose, "error halting router |%s| : |%s|\n", r.Name(), err.Error() )
        }
        return err
      }
    }
  }

  return errors.New ( "Router_network.Halt_first_edge error : Could not find an edge router to halt." )
}





func ( rn * Router_network ) get_router_by_name ( target_name string ) * router.Router {
  for _, router := range rn.routers {
    if router.Name() == target_name {
      return router
    }
  }

  ume ( "router_network: get_router_by_name: no such router |%s|", target_name )
  fp ( os.Stdout, "    routers are:\n" )
  for _, router := range rn.routers {
    fp ( os.Stdout, "      %s\n", router.Name() )
  }
  os.Exit ( 1 )
  return nil
}





func ( rn * Router_network ) How_many_interior_routers ( ) ( int ) {
  count := 0

  for _, r := range rn.routers {
    if r.Is_interior() {
      count ++
    }
  }

  return count
}





func ( rn * Router_network ) Get_nth_interior_router_name ( index int ) ( string ) {
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





func element_of ( target_str string, strings [] string ) ( bool ) {
  for _, str := range strings {
    if target_str == str {
      return true
    }
  }

  return false
}





func union ( list_1, list_2 [] string ) ( [] string ) {

  var union_list [] string

  for _, str := range list_1 {
    if ! element_of ( str, union_list ) {
      union_list = append ( union_list, str )
    }
  }

  for _, str := range list_2 {
    if ! element_of ( str, union_list ) {
      union_list = append ( union_list, str )
    }
  }

  return union_list 
}





func print_list ( label string, list [] string ) {
  fp ( os.Stdout, "%s\n", label )
  for _, x := range list {
    fp ( os.Stdout, "    %s\n", x )
  }
}





func ( rn * Router_network ) Is_the_network_connected ( ) ( bool ) {

  if len ( rn.routers ) <= 0  {
    return false
  }

  // reachable_nodes is the Big Kahuna. 
  // This is the list we are building up such that, when it is 
  // the same size as the set of all nodes -- that means that the
  // network is connected.
  reachable_nodes := [ ] string { "A" }
  var next_generation, neighbors_of_this_node [ ] string

  if len(reachable_nodes) >= len(rn.routers) {
    return true
  }

  for {
    next_generation = nil

    // Make the next generation.
    // For each node in the reachables, add all the nodes that
    // are connected to them into the new set of reachables.
    for _, reachable_node_name := range reachable_nodes {
      r := rn.get_router_by_name ( reachable_node_name )
      neighbors_of_this_node = union ( r.I_connect_to_names, r.Connect_to_me_interior )

      // For each neighbor of this node, put it into the next gen if
      //  1. it's not already there, and
      //  2. it's also not in the grand list of reachables already.
      for _, neighbor_of_this_node := range neighbors_of_this_node {
        if element_of ( neighbor_of_this_node, next_generation ) {
          continue
        }

        if element_of ( neighbor_of_this_node, reachable_nodes ) {
          continue
        }

        next_generation = append ( next_generation, neighbor_of_this_node )
      }
    }

    // Now we have built the next generation: the list of all nodes 
    // that are reachable from the set of 'reachable nodes' but were
    // not already contained therein.

    // If at this point the next generation is empty, the set 
    // of reachable nodes will not grow anymore. Fail.
    if len(next_generation) <= 0 {
      return false
    }
    
    // OK, so we *do* have some new nodes in the next generation.
    // Union them in with the set of reachables.
    reachable_nodes = union ( reachable_nodes, next_generation )

    // Do we have a Full House?
    // If so, then the network is connected!
    if len ( reachable_nodes ) >= len ( rn.routers ) {
      return true
    }
  }
}





func ( rn * Router_network ) read_CPU ( ) {

  file, err := os.Open ( "/proc/stat" ) 
  if err != nil {
    fp ( os.Stdout, "Router_network.read_CPU error: |%s|\n", err.Error() )
  }

  scanner := bufio.NewScanner ( file ) 
  scanner.Scan (  ) 
  first_line := scanner.Text (  ) [5:] // get rid of cpu plus 2 spaces
  file.Close (  ) 
  err = scanner.Err (  ) 
  if err != nil {
    fp ( os.Stdout, "Router_network.read_CPU error: |%s|\n", err.Error() )
  }

  split := strings.Fields ( first_line ) 
  idle_time, _ := strconv.ParseUint ( split[3], 10, 64 ) 
  total_time := uint64 ( 0 ) 
  for _, s := range split {
    u, _ := strconv.ParseUint ( s, 10, 64 ) 
    total_time += u
  }

  delta_idle_time := idle_time - rn.previous_idle_time
  delta_total_time := total_time - rn.previous_total_time
  cpuUsage :=  ( 1.0 - float64 ( delta_idle_time ) /float64 ( delta_total_time )  )  * 100.0
  fp ( os.Stdout, "MDEBUG %6.3f\n", cpuUsage )

  rn.previous_idle_time  = idle_time
  rn.previous_total_time = total_time
}





