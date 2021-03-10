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





func run_test ( test_name    string,
                run_name     string, 
                mercury_root string,
                n_pairs      int,
                msec_pause   int,
                n_messages   int,
                message_size int,
                client_events_channel chan string ) ( string )  {

  fp ( os.Stdout, "Running message size %d\n", message_size )
  log_path    := test_name + "/" + run_name + "/log"
  config_path := test_name + "/" + run_name + "/config"
  event_path  := test_name + "/" + run_name + "/event"
  result_path := test_name + "/" + run_name + "/result"

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

  // One-router linear network in which each connects to the previous.
  router_name   := 'A'
  network.Add_router ( string(rune(router_name)),
                       "latest",
                       config_path,
                       log_path )

  network.Init ( )
  network.Set_results_path ( result_path )
  network.Set_events_path  ( event_path )

  msec_pause_str := fmt.Sprintf ( "%d", msec_pause )

  for i := 0; i < n_pairs; i ++ {

    sender_name := fmt.Sprintf ( "sender_%05d", i )

    network.Add_sender ( sender_name,
                         config_path,
                         "0.0.0.0",
                         n_messages,
                         message_size,
                         string(rune(router_name)),
                         msec_pause_str, // throttle (msec)
                         "0",        // delay               -- and this
                         "0" )       // soak                -- and this

    receiver_name := fmt.Sprintf ( "receiver_%05d", i )
    network.Add_receiver ( receiver_name,
                           config_path,
                           "0.0.0.0",
                           n_messages,
                           message_size,
                           string(rune(router_name)),
                           "0",
                           "0" )
    
    address := fmt.Sprintf ( "addr_%05d", i )
    network.Add_Address_To_Client ( sender_name,   address )
    network.Add_Address_To_Client ( receiver_name, address )
  }
                        

  network.Run  ( )
  fp ( os.Stdout, "network |%s| is running.\n", run_name )

  // TODO fix this with communication!
  time.Sleep ( 10 * time.Second )

  fp ( os.Stdout, "start_sending at %f\n", utils.Timestamp() )
  os.Create ( event_path + "/start_sending" )

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

  os.Create ( event_path + "/dump_data" )

  // TODO fix this! -- with communication
  time.Sleep ( 30 * time.Second )
  
  network.Halt ( );

  return result_path
}





func main ( ) {

  mercury_root := os.Getenv ( "MERCURY_ROOT" )
  client_events_channel := make ( chan string, 5 )
  test_name := "message-size" + "_" + time.Now().Format ( "2006_01_02_1504" )

  n_messages     := 1000
  n_client_pairs := 10
  msec_pause     := 6
  message_size   := 200000

  // This is how you would make a run name if you were 
  // running a message-size test.
  run_name := fmt.Sprintf ( "message-size_%d", message_size )
  fp ( os.Stdout, "Running: %s at %v\n", run_name, time.Now() )
  run_test ( test_name,
             run_name, 
             mercury_root, 
             n_client_pairs,
             int(msec_pause),
             int(n_messages),
             message_size,
             client_events_channel )

  fp ( os.Stdout, "Test %s done at %s\n", test_name, time.Now().Format ( "2006_01_02_1504" ) )
}





