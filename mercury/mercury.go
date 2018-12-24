package main

import (
  "bufio"
  "fmt"
  "os"
  "strings"
  "regexp"
)



var fp = fmt.Fprintf


type action func ( string, * Context, []string ) 



type Context struct {
  line_rx                      * regexp.Regexp
  functions                      map [ string ] action

  router_path                    string
  mercury_path                   string

  verbose                        bool

  n_worker_threads               int
  resource_measurement_frequency int
}





/*=====================================================================
  Action Functions
======================================================================*/


func quit ( name string, context * Context, argv [] string ) {

  if argv[0] == "help" {
    fp ( os.Stdout, "    %s\n", name  )
    fp ( os.Stdout, "        Gracefully shut down.\n\n"  )
    return
  }

  os.Exit ( 0 )
}





func verbose ( name string, context * Context, argv [] string ) {

  if argv[0] == "help" {
    fp ( os.Stdout, "    %s\n", name  )
    fp ( os.Stdout, "        Explain everything that's happening.\n\n"  )
    return
  }

  context.verbose = true
}





func set_paths ( name string, context * Context, argv [] string ) {

  if argv[0] == "help" || len(argv) < 3 {
    fp ( os.Stdout, "    %s router_path mercury_path\n", name  )
    fp ( os.Stdout, "        Set the paths tht Mercury needs.\n\n",  )
    return
  }

  context.router_path  = argv[1]
  context.mercury_path = argv[2]

  if context.verbose {
    fp ( os.Stderr, "  router  path set to |%s|\n", context.router_path  )
    fp ( os.Stderr, "  mercury path set to |%s|\n", context.mercury_path )
  }
}





func help ( name string, context * Context, argv [] string ) {
  for key, fn := range context.functions {
    if key != "help" && key != "?" {
      fn ( key, context, []string{"help"} )
    }
  }
}





func read_file ( name string, context * Context, argv [] string ) {

  if argv[0] == "help" || len(argv) < 2 {
    fp ( os.Stdout, "    %s file_path\n", name )
    fp ( os.Stdout, "        Open the given file and process its lines just as\n" )
    fp ( os.Stdout, "        lines typed from the console would be processed.\n\n"  )
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


//^^^^^^^^^^^^^^^  End Action Functions ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^





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
  context.n_worker_threads               = 4
  context.resource_measurement_frequency = 0
}





func main() {
  
  var context Context
  init_context ( & context )

  functions := make ( map [string] action )
  functions [ "help"     ] = help
  functions [ "?"        ] = help
  functions [ "quit"     ] = quit
  functions [ "q"        ] = quit
  functions [ "paths"    ] = set_paths
  functions [ "read_file"] = read_file
  functions [ "rf"       ] = read_file
  functions [ "verbose"  ] = verbose

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





