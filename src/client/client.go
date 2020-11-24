package client

import ( "fmt"
         "errors"
         "os"
         "os/exec"
         "strings"
         "time"
         "strconv"

         "utils"
       )





var fp          = fmt.Fprintf
var module_name = "client"
var ume         = utils.M_error
var umi         = utils.M_info




type Client_state int

const (
  none         Client_state = iota
  initialized
  running
  halted
)




type Client struct {
  Name                 string
  config_path          string
  results_path         string
  events_path          string
  Operation            string
  Port                 string

  Path                 string
  ld_library_path      string
  pythonpath           string
  log_file             string

  N_messages           int

  cmd                * exec.Cmd
  State                Client_state
  message_length       int
  addrs             [] string

  throttle             string

  verbose              bool

  delay                string

  soak                 string

  status_file_name     string


  // This gets set by the network, when it is 
  // checking on this client's status file.
  Completed            bool

  Received             int
  Accepted             int
  Rejected             int
  Released             int
  Modified             int
}





func New_client ( name                  string,
                  config_path           string,
                  results_path          string,
                  events_path           string,
                  operation             string,
                  port                  string,
                  path                  string,
                  ld_library_path       string,
                  pythonpath            string,
                  log_file              string,
                  n_messages            int,
                  message_length        int, 
                  throttle              string,
                  verbose               bool,
                  delay                 string,
                  soak                  string ) ( * Client )  { 
  var c * Client

  full_config_path := config_path + "/" + name
  utils.Find_or_create_dir ( full_config_path )

  c = & Client { Name                  : name,
                 config_path           : full_config_path,
                 results_path          : results_path,
                 events_path           : events_path,
                 Operation             : operation,
                 Port                  : port,
                 Path                  : path,
                 ld_library_path       : ld_library_path,
                 pythonpath            : pythonpath,
                 log_file              : log_file,
                 State                 : initialized,
                 N_messages            : n_messages,
                 message_length        : message_length,
                 throttle              : throttle,
                 verbose               : verbose,
                 delay                 : delay,
                 soak                  : soak }

  if ! utils.Path_exists ( path ) {
    ume ( "client: executable path |%s| isn't there.", c.Path )
    return nil
  }

  utils.Find_or_create_dir ( config_path )

  return c
}





func ( c * Client ) Add_Address ( addr string ) {
  c.addrs = append ( c.addrs, addr ) 
}





func ( c * Client ) Run ( ) {

  // Don't warn in this case. It's normal behavior
  // to tell everything to run -- even those clients
  // that are already running.
  if c.State >= running {
    return
  }

  // Set up the environment for the router process.
  os.Setenv ( "LD_LIBRARY_PATH", c.ld_library_path )
  os.Setenv ( "PYTHONPATH"     , c.pythonpath )

  // Name should always be first, because it may be used 
  // in the course of other argv processing.
  if c.results_path == "" {
    fp ( os.Stdout, "client.Run error: empty result path.\n" )
    utils.Print_Callstack ( )
    return
  }

  args := " --name " + c.Name + 
          " --flight_times_file_name " + c.results_path + 
          " --events_path " + c.events_path +
          " --operation " + c.Operation + 
          " --port " + c.Port + 
          " --log " + c.log_file + 
          " --messages " + strconv.Itoa(c.N_messages) + 
          " --message_length " + strconv.Itoa(c.message_length) + 
          " --throttle " + c.throttle +
          " --delay " + c.delay

  if c.soak == "true" {
    args += " --soak"
  }

  for _, addr := range c.addrs {
    args += " --address " + addr
  }
  args_list := strings.Fields ( args )
  c.cmd = exec.Command ( c.Path,  args_list... )

  // Write the command line. -------------------------------
  command_file_name := c.config_path + "/" + "command_line"
  command_file, err := os.Create ( command_file_name )
  utils.Check ( err )
  defer command_file.Close ( )
  command_string := c.Path + " " + args
  command_file.WriteString ( command_string + "\n" )

  // Write the environment variables. ----------------------
  environment_file_name := c.config_path + "/" + "environment_variables"
  environment_file, err := os.Create ( environment_file_name )
  utils.Check ( err )
  defer environment_file.Close ( )
  environment_string := "export LD_LIBRARY_PATH=" + c.ld_library_path + "\n" +
                        "export PYTHONPATH="      + c.pythonpath + "\n"
  environment_file.WriteString ( environment_string )


  // Start the client command. After the call to Start(),
  // the client is running detached.
  //fp ( os.Stderr, "running client |%s|\n", c.Name )
  err = c.cmd.Start()
  if err != nil {
    ume ( "client |%s| start-up error: |%s|", c.Name, err.Error() )
    return
  }

  c.State = running
  umi ( c.verbose, "client |%s| is running with pid %d.", c.Name, c.cmd.Process.Pid )
}





func ( c * Client ) Is_running ( ) ( bool ) {
  return c.State == running
}





func ( c * Client ) Halt ( ) error {

  // Let's not treat this as an error. Just as the user 
  // can freely "run" the network even if parts of it are
  // already running -- let's allow them to halt it even 
  // if parts are already halted.
  if c.State == halted {
    return nil
  }

  // Set up a channel that will return a
  // message immediately if the process has
  // already terminated. Then set up a half-second
  // timer. If the timer expires before the Wait
  // returns a 'done' message, we judge that the
  // process was still running when we came along
  // and killed it. Which is good.
  done := make ( chan error, 1 )
  go func ( ) {
      done <- c.cmd.Wait ( )
  } ( )

  select {
    /*
      This is the expected case.
      Our timer times out while the above Wait() is still waiting.
      This means that the process is still running normally when we kill it.
    */
    case <-time.After ( 250 * time.Millisecond ) :
      c.State = halted
      if err := c.cmd.Process.Kill(); err != nil {
        return errors.New ( "failed to kill process: " + err.Error() )
      }
      return nil

    case err := <-done:
      c.State = halted
      if err != nil {
        return errors.New ( "process terminated early with error: " + err.Error() )
      }

      // Even though there was no error reported -- the process
      // mevertheless stopped early, which is an error in the
      // context of this test.
      return errors.New ( "process self-terminated." )
  }

  // I think this is unreachable.
  c.State = halted
  return nil
}





func ( c * Client ) Kill_and_restart ( pause int ) {
  umi ( c.verbose, "client |%s| going down for kill-and-restart.", c.Name )
  c.Halt ( )
  c.State = initialized
  time.Sleep ( time.Duration(pause) * time.Second )
  c.Run ( )
  umi ( c.verbose, "client |%s| restarted.", c.Name )
}





