package main

import (
            "fmt"
            "io/ioutil"
            "os"
            "strings"
            "time"

         rn "router_network"
            "utils"
       )


var fp=fmt.Fprintf



func listen_for_messages_from_clients ( event_path string, 
                                        receiver_count int,
                                        client_events_channel chan string ) {
  previous_count := 0
  same_count     := 0

  for {
    time.Sleep ( 5 * time.Second )

    done_receiving_count := 0
    files, _ := ioutil.ReadDir ( event_path )
    for _, f := range files {
      if strings.HasPrefix ( f.Name(), "done_receiving" ) {
        done_receiving_count ++
      }
    }

    if done_receiving_count > 0 {
      if done_receiving_count == previous_count {
        same_count ++
      }

      if same_count > 5 {
        client_events_channel <- "not changing"
        break
      }

      previous_count = done_receiving_count
    }

    if done_receiving_count >= receiver_count {
      client_events_channel <- "done receiving"
      break
    }
  }
}





func run_linear_network ( run_name     string, 
                          mercury_root string,
                          n_routers    int,
                          n_pairs      int,
                          client_events_channel chan string ) {

  log_path    := run_name + "/log"
  config_path := run_name + "/config"
  event_path  := run_name + "/event"
  result_path := run_name + "/result"

  utils.Find_or_create_dir ( log_path )
  utils.Find_or_create_dir ( config_path )
  utils.Find_or_create_dir ( event_path )
  utils.Find_or_create_dir ( result_path )

  network := rn.New_router_network ( run_name,
                                     mercury_root,
                                     log_path )

  // TODO fix this!
  network.Add_version_with_roots ( "latest",
                                   "/home/mick/latest/install/proton",
                                   "/home/mick/latest/install/dispatch" )

  // N router linear network in which each connects to the previous.
  first_router_name := 'A'
  last_router_name  := first_router_name

  for i := 0; i < n_routers; i ++ {
    current_router   := 'A' + i
    last_router_name = rune(current_router)
    network.Add_router ( string(rune(current_router)),
                         "latest",
                         config_path,
                         log_path )
    if i > 0 {
      network.Connect_router ( string(rune(current_router)), 
                               string(rune(current_router - 1)) )
    }
  }

  network.Init ( )
  network.Set_results_path ( result_path )
  network.Set_events_path  ( event_path )

  // Send both signals right now.
  os.Create ( event_path + "/start_sending" )
  os.Create ( event_path + "/dump_data" )

  for i := 0; i < n_pairs; i ++ {

    sender_name := fmt.Sprintf ( "sender_%05d", i )

    network.Add_sender ( sender_name,
                         ".",        // config_path
                         100,        // n_messages
                         100,        // max_message_length  -- TODO get rid of this.
                         string(rune(first_router_name)),
                         "100",      // throttle (msec)
                         "0",        // delay               -- and this
                         "0" )       // soak                -- and this

    receiver_name := fmt.Sprintf ( "receiver_%05d", i )
    network.Add_receiver ( receiver_name,
                           ".",
                           100,
                           100,
                           string(rune(last_router_name)),
                           "0",
                           "0" )
    
    address := fmt.Sprintf ( "addr_%05d", i )
    network.Add_Address_To_Client ( sender_name,   address )
    network.Add_Address_To_Client ( receiver_name, address )
  }
                        

  network.Run  ( )

  fp ( os.Stdout, "MDEBUG network |%s| is running.\n", run_name )

  // TODO NEXT ! -- make it get signals!
  // TODO -- replace this with client-to-router comms
  go listen_for_messages_from_clients ( event_path, 
                                        n_pairs,
                                        client_events_channel ) 

  for {
    msg := <- client_events_channel

    switch msg {
      case "done receiving" :
        fp ( os.Stdout, "test ran successfully.\n" )

      default :
        fp ( os.Stdout, "test failed.\n" )
    }

    break
  }
  
  network.Halt ( );
}





func main ( ) {

  mercury_root := os.Getenv ( "MERCURY_ROOT" )
  client_events_channel := make ( chan string, 5 )

  for n_routers := 1; n_routers <= 3; n_routers ++ {
    for n_client_pairs := 100; n_client_pairs <= 4000; n_client_pairs += 100 {
      run_name := fmt.Sprintf ( "n-routers_%d_n-clients_%d", n_routers, n_client_pairs )
      fp ( os.Stdout, "Running: %s at %v\n", run_name, time.Now() )
      run_linear_network ( run_name, 
                           mercury_root, 
                           n_routers, 
                           n_client_pairs,
                           client_events_channel )
      // A little pause before starting next one.
      time.Sleep ( 10 * time.Second )
    }
  }
}





