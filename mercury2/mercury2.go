package main

import (
         "fmt"
         "os"
         "time"

         rn "router_network"
       )


var fp=fmt.Fprintf


func main ( ) {

  mercury_root := os.Getenv ( "MERCURY_ROOT" )

  network := rn.New_router_network ( "network",
                                     mercury_root,
                                     "." )
  network.Add_version_with_roots ( "latest",
                                   "/home/mick/latest/install/proton",
                                   "/home/mick/latest/install/dispatch" )

  network.Add_router ( "A",
                       "latest",
                       ".",    // config_path
                       "." )   // log_path
  network.Init ( )
  network.Run  ( )

  for {
    fp ( os.Stdout, "MDEBUG The network is running.\n" )
    time.Sleep ( 5 * time.Second )
  }
}





