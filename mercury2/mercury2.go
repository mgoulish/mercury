package main

import (
         "fmt"
         "os"

         rn "router_network"
       )


var fp=fmt.Fprintf


func main ( ) {

  mercury_root := os.Getenv ( "MERCURY_ROOT" )

  network := rn.New_router_network ( "network",
                                     mercury_root,
                                     "." )
  fp ( os.Stdout, "MDEBUG network: |%#v|\n", network )
}





