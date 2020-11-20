package main

import (
            "bufio"
            "fmt"
            "io/ioutil"
            "os"
            "os/exec"
            "path/filepath"
            "sort"
            "strconv"
            "strings"
            "time"

         rn "router_network"
            "utils"
       )


var fp=fmt.Fprintf




type message_result struct {
  arrival_time float64 
  latency      float64
}



/*---------------------------------------------------
  Result list that will be sorted for arrival time.
---------------------------------------------------*/
type arrival_sorted_results [] *message_result


// Functions to satisfy the Sort interface.
func ( asr arrival_sorted_results ) Len ( ) ( int ) {
  return len ( asr )
}

func ( asr arrival_sorted_results ) Less ( i, j int ) bool {
  return asr[i].arrival_time < asr[j].arrival_time
}

func ( asr arrival_sorted_results ) Swap ( i, j int ) {
  asr[i] , asr[j] = asr[j], asr[i]
}



/*---------------------------------------------------
  Result list that will be sorted for latency.
---------------------------------------------------*/
type latency_sorted_results [] *message_result


// Functions to satisfy the Sort interface.
func ( lsr latency_sorted_results ) Len ( ) ( int ) {
  return len ( lsr )
}

func ( lsr latency_sorted_results ) Less ( i, j int ) bool {
  return lsr[i].latency < lsr[j].latency
}

func ( lsr latency_sorted_results ) Swap ( i, j int ) {
  lsr[i] , lsr[j] = lsr[j], lsr[i]
}









type test_result struct {
  test_time      time.Time
  n_routers      int
  n_client_pairs int
  a_results      arrival_sorted_results
  l_results      latency_sorted_results
  mean           float64
  ninety_nine    float64
}





func new_test_result ( test_time time.Time, n_routers, n_pairs int ) ( * test_result ) {
  return & test_result { test_time      : test_time,
                         n_routers      : n_routers,
                         n_client_pairs : n_pairs }
                         
}





func ( t * test_result ) read ( dir string, signifier string ) ( error ) {
  // Get a list of all the file names in 'dir' 
  // whose names contain 'signifier'.
  var file_names [] string
  _ = filepath.Walk ( dir,
                      func ( path string, info os.FileInfo, err error) error {
                        if ! info.IsDir ( ) {
                          if strings.Contains ( path, signifier ) {
                            file_names = append ( file_names, path )
                          }
                        }
                        return nil
                      } )

  for _, file_name := range file_names {
    // Get the message results out of the file.
    // Two floats per line.
    content, err := ioutil.ReadFile ( file_name )
    if err != nil {
      return err
    }

    lines := strings.Split ( string(content), "\n" )

    for _, line := range lines {

      if line == "" {
        // This file is finished.
        break
      }

      numbers := strings.Split ( line, " " )
      result := message_result{}
      result.arrival_time, err = strconv.ParseFloat ( numbers[0], 64 )
      if err != nil {
        return err
      }
      result.latency, err      = strconv.ParseFloat ( numbers[1], 64 )
      if err != nil {
        return err
      }
      t.a_results = append ( t.a_results, &result )
    }
  }

  return nil
}





func process_all_runs ( results [] * test_result, test_path string ) {

  // Print out means for all runs.
  mean_file_name := test_path + "/mean.data"
  f, _ := os.Create ( mean_file_name )

  for _, r := range results {
    fmt.Fprintf ( f, "%d    %.6f\n", r.n_client_pairs, r.mean )
  }
  f.Close()  // Can't defer!


  // Print out 99s for all runs.
  nn_file_name := test_path + "/99.data"
  f, _ = os.Create ( nn_file_name )

  for _, r := range results {
    fmt.Fprintf ( f, "%d    %.6f\n", r.n_client_pairs, r.ninety_nine )
  }
  f.Close()  // Can't defer!


  n_routers := results[0].n_routers // Same number of routers for all runs.
  least_n_clients := results[0].n_client_pairs
  most_n_clients  := results[len(results)-1].n_client_pairs

  // Make the gnuplot file for mean latency.
  gnuplot_script := "set autoscale\n"
  gnuplot_script += "unset key\n"
  gnuplot_script += "set ylabel \"latency (msec)\"\n"
  gnuplot_script += "set xlabel \"N-Clients  (Sender-Receiver Pairs)\"\n"
  gnuplot_script += "set terminal jpeg size 2000,600\n"
  gnuplot_script += fmt.Sprintf ( "set output \"latency_n-routers_%d_n-clients_%d_%d.jpg\"\n", n_routers, least_n_clients, most_n_clients )
  gnuplot_script += fmt.Sprintf ( "set title \"Dispatch Router Latency -- N-Routers : %d\"\n", n_routers )
  gnuplot_script += "set mytics 2\n"
  gnuplot_script += fmt.Sprintf ( "plot \"%s\" with linespoints lt rgb \"red\" lw 3, \"%s\" with linespoints lt rgb \"gold\" lw 3\n", "mean.data", "99.data" )

  gnuplot_script_name := test_path + "/mean.gplot"
  file, err := os.Create ( gnuplot_script_name )
  if err != nil {
    fp ( os.Stdout, "Can't create gplot script file: |%s|\n", err.Error() )
    os.Exit ( 1 )
  }
  fmt.Fprintf ( file, "%s", gnuplot_script )
  file.Close()

  // Run gnuplot.
  args_list := [] string { "mean.gplot" }
  command := exec.Command ( "/usr/bin/gnuplot", args_list ... )
  command.Dir = test_path
  output, err := command.CombinedOutput ( )
  if err != nil {
    fp ( os.Stdout, "process_all_runs error running gnuplot: stdout: |%s|  error: |%s|\n", output, err.Error() )
  }
}





// We get the collection of all results from all clients.
// Sort them all by time of occurrence, then subtract the first
// time from all, delete the first and last second, and print
// the gnuplot file.
func ( t * test_result ) process_run ( graphics_path string ) {

  sort.Sort ( arrival_sorted_results ( t.a_results ) )

  first_time := t.a_results[0].arrival_time

  for _, result := range ( t.a_results ) {
    result.arrival_time -= first_time
  }

  // Delete first and last 1-second intervals.
  last_time  := t.a_results[len(t.a_results)-1].arrival_time

  for _, result := range ( t.a_results ) {

    // first 1-second
    if result.arrival_time < 1.0 {
      result.arrival_time = -1.0
    }

    // last 1-second
    if last_time - result.arrival_time <= 1.0 {
      result.arrival_time = -1.0
    }
  }

  // Use these to calculate the average.
  sum   := float64(0)
  count := 0

  // Make the timeline data file.
  f, _ := os.Create ( graphics_path + "/timeline.data" )
  defer f.Close()
  w := bufio.NewWriter ( f )

  for _, result := range t.a_results {
    if result.arrival_time >= 0 {
      fp ( w, "%.6f %.6f\n", result.arrival_time, result.latency )
      sum += result.latency
      count ++
    }
  }
  w.Flush ( )

  // Store the average.
  t.mean = sum / float64(count)


  // The a_results are results sorted for arrival time.
  // Now copy all the ones we have not deleted into the array of 
  // l_results: results that will be sorted for latency.
  for _, result := range t.a_results {
    if result.arrival_time >= 0 {
      t.l_results = append ( t.l_results, result )
    }
  }
  sort.Sort ( latency_sorted_results ( t.l_results ) )
  /*
  f, err := os.Create("./latency_sorted_results")
  defer f.Close()
  for _, result := range t.l_results {
    fmt.Fprintf ( f, "%f %f\n", result.arrival_time, result.latency )
  }
  */

  // Find the ninety-nine percent mark.
  ninety_ninth_percent_index := len(t.l_results) - len(t.l_results)/100
  t.ninety_nine = t.l_results [ ninety_ninth_percent_index ].latency

  //fp ( os.Stdout, "MDEBUG ninety_ninth_percent_value: %f\n", ninety_ninth_percent_value )


  // Make the timeline gnuplot file.
  gnuplot_script := "set autoscale\n"
  gnuplot_script += "unset key\n"
  gnuplot_script += "set ylabel \"latency (msec)\"\n"
  gnuplot_script += "set xlabel \"time\"\n"
  gnuplot_script += "set terminal jpeg size 2000,500\n"
  gnuplot_script += fmt.Sprintf ( "set output \"timeline_n-routers_%d_n-clients_%d.jpg\"\n", t.n_routers, t.n_client_pairs )
  gnuplot_script += fmt.Sprintf ( "set title \"Timeline View of Trimmed Data -- %d Client-Pairs\"\n", t.n_client_pairs )
  gnuplot_script += "plot \"timeline.data\" with points\n"

  // Write out the gnuplot script.
  gnuplot_script_name := graphics_path + "/timeline.gplot"
  file, err := os.Create ( gnuplot_script_name )
  if err != nil {
    fp ( os.Stdout, "Can't create gplot script file: |%s|\n", err.Error() )
    os.Exit ( 1 )
  }
  fmt.Fprintf ( file, "%s", gnuplot_script )
  file.Close()

  args_list := [] string { "timeline.gplot" }
  command := exec.Command ( "/usr/bin/gnuplot", args_list ... )
  command.Dir = graphics_path
  //err = command.Run ( )
  output, err := command.CombinedOutput ( )
  if err != nil {
    fp ( os.Stdout, "test_result.process_run : error running gnuplot: stdout: |%s|  error: |%s|\n", output, err.Error() )
    return
  }
}





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





func run_linear_network ( test_name    string,
                          run_name     string, 
                          mercury_root string,
                          n_routers    int,
                          n_pairs      int,
                          msec_pause   int,
                          n_messages   int,
                          client_events_channel chan string ) ( string )  {

  fp ( os.Stdout, "Running linear network with pairs: %d, msec: %d, messages: %d\n", n_pairs, msec_pause, n_messages )
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

  //mpsps := 1000.0 / float64(msec_pause)
  msec_pause_str := fmt.Sprintf ( "%d", msec_pause )

  for i := 0; i < n_pairs; i ++ {

    sender_name := fmt.Sprintf ( "sender_%05d", i )

    network.Add_sender ( sender_name,
                         ".",        // config_path
                         n_messages,
                         100,        // max_message_length  -- TODO get rid of this.
                         string(rune(first_router_name)),
                         msec_pause_str, // throttle (msec)
                         "0",        // delay               -- and this
                         "0" )       // soak                -- and this

    receiver_name := fmt.Sprintf ( "receiver_%05d", i )
    network.Add_receiver ( receiver_name,
                           ".",
                           n_messages,
                           100,
                           string(rune(last_router_name)),
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

  fp ( os.Stdout, "MDEBUG start_sending at %f\n", utils.Timestamp() )
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
  test_name := "latency" + "_" + time.Now().Format ( "2006_01_02_1504" )

  // Hold MPS constant.
  messages_per_second := 10000.0
  n_seconds           := 200000.0

  // Each time through this loop is one 'run' of the test.
  for n_routers := 1; n_routers <= 1; n_routers ++ { 
    
    // This holds the results for one run.
    var test_results [] * test_result
    for n_client_pairs := 500; n_client_pairs <= 12000; n_client_pairs += 500 { 

      messages_per_second_per_sender := messages_per_second / float64(n_client_pairs)
      msec_pause := 1000 / messages_per_second_per_sender
      n_messages := n_seconds / msec_pause

      // This inner loop performs a single run.
      run_name := fmt.Sprintf ( "n-routers_%d_n-clients_%d", n_routers, n_client_pairs )
      fp ( os.Stdout, "Running: %s at %v\n", run_name, time.Now() )

      graphics_path := test_name + "/" + run_name + "/graphics"
      utils.Find_or_create_dir ( graphics_path )

      results_dir := run_linear_network ( test_name,
                                          run_name, 
                                          mercury_root, 
                                          n_routers, 
                                          n_client_pairs,
                                          int(msec_pause),
                                          int(n_messages),
                                          client_events_channel )

      result := new_test_result ( time.Now(), n_routers, n_client_pairs )
      result.read ( results_dir, "flight_times" )
      result.process_run ( graphics_path )
      test_results = append ( test_results, result )

      // A little pause before starting next one.
      time.Sleep ( 10 * time.Second )
    }

    // Process all runs for the current number of routers.
    process_all_runs ( test_results, test_name )
  }

  fp ( os.Stdout, "Test %s done at %s\n", test_name, time.Now().Format ( "2006_01_02_1504" ) )
}





