package main

import (
  "bufio"
  "fmt"
  "os"
  "strings"
  "regexp"

  "utils"
  rn "router_network"
)



var fp = fmt.Fprintf


type command func ( string, * Context, []string ) 



type Context struct {
  line_rx                      * regexp.Regexp
  functions                      map [ string ] command

  dispatch_install_root          string
  proton_install_root            string
  mercury_root                   string

  result_path                    string
  router_path                    string
  config_path                    string
  log_path                       string
  client_path                    string

  verbose                        bool

  n_worker_threads               int
  resource_measurement_frequency int

  network                        * rn.Router_Network
  network_running                bool
}





/*=====================================================================
  Command Functions
======================================================================*/


func quit ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" {
    fp ( os.Stdout, "    %s\n", command_name  )
    fp ( os.Stdout, "        Gracefully shut down.\n\n"  )
    return
  }

  if context.network_running {
    context.network.Halt ( )
  }

  os.Exit ( 0 )
}





func verbose ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" {
    fp ( os.Stdout, "    %s\n", command_name  )
    fp ( os.Stdout, "        Explain everything that's happening.\n\n"  )
    return
  }

  context.verbose = true
}





func set_paths ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" || len(argv) < 4 {
    fp ( os.Stdout, "    %s dispatch_install_root proton_install_root mercury_root\n", command_name  )
    fp ( os.Stdout, "        Set the paths that Mercury needs.\n\n",  )
    return
  }

  context.dispatch_install_root = argv[1]
  context.proton_install_root   = argv[2]
  context.mercury_root          = argv[3]

  context.router_path = context.dispatch_install_root + "/sbin/qdrouterd"

  if context.verbose {
    fp ( os.Stdout, "  %s command: dispatch install root set to |%s|\n", command_name, context.dispatch_install_root  )
    fp ( os.Stdout, "  %s command: proton   install root set to |%s|\n", command_name, context.proton_install_root  )
    fp ( os.Stdout, "  %s command: mercury path set to |%s|\n",          command_name, context.mercury_root )
    fp ( os.Stdout, "\n" )
  }
}





func help ( command_name string, context * Context, argv [] string ) {
  for key, fn := range context.functions {
    if key != "help" && key != "?" {
      fn ( key, context, []string{"help"} )
    }
  }
}





func read_file ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" || len(argv) < 2 {
    fp ( os.Stdout, "    %s file_path\n", command_name )
    fp ( os.Stdout, "        Open the given file and process its lines just as\n" )
    fp ( os.Stdout, "        lines typed from the console would be processed.\n\n"  )
    fp ( os.Stdout, "\n" )
    return
  }

  file, err := os.Open ( argv[1] )
  if err != nil {
    panic ( err )
  }
  defer file.Close()

  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    process_line ( context, scanner.Text() )
  }

  if err := scanner.Err(); err != nil {
    panic ( err )
  }

}





func add_router ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" || len(argv) < 2 {
    fp ( os.Stdout, "    %s router_name\n", command_name )
    fp ( os.Stdout, "        Add routers of the given names to the network.\n" )
    fp ( os.Stdout, "        ( You can have multiple names in one command. )\n\n"  )
    return
  }

  for i := 1; i < len(argv); i ++ {
    if argv[i] != "" {
      context.network.Add_router ( argv[i] )
      if context.verbose {
        fp ( os.Stdout, "  added router |%s|\n", argv[i] )
      }
    }
  }
}





func connect ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" || len(argv) != 3 {
    fp ( os.Stdout, "    %s router_1 router_2\n", command_name )
    fp ( os.Stdout, "        Connect router_1 to router_2. I.e. router_1 will\n" )
    fp ( os.Stdout, "        connect to router_2's listener.\n\n"  )
    fp ( os.Stderr, " len argv was %d\n", len(argv) )
    return
  }

  context.network.Connect_router ( argv[1], argv[2] )

  if context.verbose {
    fp ( os.Stdout, "  connected %s to %s\n\n", argv[1], argv[2] )
  }
}





func network ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" {
    fp ( os.Stdout, "    %s\n", command_name )
    fp ( os.Stdout, "        Create the router network.\n" )
    return
  }

  create_network ( context )

  if context.verbose {
    fp ( os.Stdout, "  network created.\n\n" )
  }
}





func run ( command_name string, context * Context, argv [] string ) {

  if argv[0] == "help" {
    fp ( os.Stdout, "    %s\n", command_name )
    fp ( os.Stdout, "        Run the router network.\n" )
    return
  }

  context.network.Init ( )
  context.network.Run ( )

  if context.verbose {
    fp ( os.Stdout, "  network is running.\n\n" )
  }

  context.network_running = true
}



//^^^^^^^^^^^^^^^  End Command Functions ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^



func process_line ( context * Context, line string ) {
  /*----------------------------------------
    Clean up the line
  -----------------------------------------*/
  line = strings.Replace ( line, "\n", "", -1 )
  line = context.line_rx.ReplaceAllString ( line, " " )
  words := strings.Split ( line, " " )

  /*----------------------------------------
    The first word should be the name 
    of a function. Call it.
  -----------------------------------------*/
  found_it := false
  for key, fn := range context.functions {
    if key == words[0] {
      found_it = true
      fn ( key, context, words )
    }
  }

  if ! found_it {
    help ( "help", context, []string{"help"} )
  }
}





func init_context ( context * Context ) {
  context.verbose                        = false
  context.network_running                = false
  context.n_worker_threads               = 4
  context.resource_measurement_frequency = 0
}





func create_network (context * Context ) {

  context.result_path             = context.mercury_root + "/mercury/results"
  context.config_path             = context.mercury_root + "/mercury/config"
  context.log_path                = context.mercury_root + "/mercury/log"
  context.client_path             = context.mercury_root + "/clients/c_proactor_client"

  utils.Find_or_create_dir ( context.result_path )
  utils.Find_or_create_dir ( context.config_path )
  utils.Find_or_create_dir ( context.log_path )
  // Don't try to create the client path. That's an executable, not a directory.

  if context.verbose {
    fp ( os.Stdout, "  create_network: result_path == |%s|\n", context.result_path )
    fp ( os.Stdout, "  create_network: config_path == |%s|\n", context.config_path )
    fp ( os.Stdout, "  create_network: log_path    == |%s|\n", context.log_path )
    fp ( os.Stdout, "  create_network: client_path == |%s|\n", context.client_path )
    fp ( os.Stdout, "\n" )
  }

  context.network = rn.New_Router_Network ( "mercury_router_network",
                                            context.n_worker_threads,
                                            context.result_path,
                                            context.router_path,
                                            context.config_path,
                                            context.log_path,
                                            context.client_path,
                                            context.dispatch_install_root,
                                            context.proton_install_root,
                                            context.verbose,
                                            context.resource_measurement_frequency )
}





func main() {
  
  var context Context
  init_context   ( & context )

  functions := make ( map [string] command )
  functions [ "help"       ] = help
  functions [ "?"          ] = help
  functions [ "quit"       ] = quit
  functions [ "q"          ] = quit
  functions [ "paths"      ] = set_paths
  functions [ "read_file"  ] = read_file
  functions [ "rf"         ] = read_file
  functions [ "verbose"    ] = verbose
  functions [ "add_router" ] = add_router
  functions [ "ar"         ] = add_router
  functions [ "connect"    ] = connect
  functions [ "c"          ] = connect
  functions [ "run"        ] = run
  functions [ "network"    ] = network

  context.functions = functions
  context.line_rx   = regexp.MustCompile(`\s+`)

  /*--------------------------------------------
    Process files named on command line.
  --------------------------------------------*/
  for i := 1; i < len(os.Args); i ++ {
    read_file ( "read_file", & context, [] string { "read_file", os.Args[i] } )
  }

  /*--------------------------------------------
    Prompt for and read the next line of input.
  --------------------------------------------*/
  reader := bufio.NewReader ( os.Stdin )
  for {
    fp ( os.Stdout, "%c ", '\u263F' )
    line, _ := reader.ReadString ( '\n' )

    process_line ( & context, line )
  }
}





