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
  network.Set_results_path ( "." )
  network.Set_events_path  ( "." )

  // Send both signals right now.
  os.Create ( "./start_sending" )
  os.Create ( "./dump_data" )

  network.Add_sender ( "sender",   // name
                       ".",        // config_path
                       100,        // n_messages
                       100,        // max_message_length  -- TODO get rid of this.
                       "A",        // router name
                       "100",      // throttle (msec)
                       "0",          // delay               -- and this
                       "0" )         // soak                -- and this

  network.Add_receiver ( "receiver",
                         ".",
                         100,
                         100,
                         "A",
                         "0",
                         "0" )
                        
  network.Add_Address_To_Client ( "sender",   "addr" )
  network.Add_Address_To_Client ( "receiver", "addr" )

  network.Run  ( )

  for {
    fp ( os.Stdout, "MDEBUG The network is running.\n" )
    time.Sleep ( 5 * time.Second )
  }
}





