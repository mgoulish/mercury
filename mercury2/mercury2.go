package main

import (
         "fmt"
         "os"
         "time"

         rn "router_network"
       )


var fp=fmt.Fprintf




func run_network ( network_name string, mercury_root string ) {

  network := rn.New_router_network ( network_name,
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

  sender_name   := network_name + "_" + "sender"
  receiver_name := network_name + "_" + "receiver"

  network.Add_sender ( sender_name,
                       ".",        // config_path
                       100,        // n_messages
                       100,        // max_message_length  -- TODO get rid of this.
                       "A",        // router name
                       "100",      // throttle (msec)
                       "0",          // delay               -- and this
                       "0" )         // soak                -- and this

  network.Add_receiver ( receiver_name,
                         ".",
                         100,
                         100,
                         "A",
                         "0",
                         "0" )
                        
  network.Add_Address_To_Client ( sender_name,   "addr" )
  network.Add_Address_To_Client ( receiver_name, "addr" )

  network.Run  ( )

  fp ( os.Stdout, "MDEBUG network |%s| is running.\n", network_name )
  time.Sleep ( 20 * time.Second )

  
  // CAN WE HALT THIS ONE AND MAKE A NEW ONE ???
  network.Halt ( );

  // Clean up the signals
  os.Remove ( "./start_sending" )
  os.Remove ( "./dump_data" )
}





func main ( ) {

  mercury_root := os.Getenv ( "MERCURY_ROOT" )

  run_network ( "network_one", mercury_root )

  // A little pause before starting next one.
  time.Sleep ( 10 * time.Second )

  run_network ( "network_two", mercury_root )
}





