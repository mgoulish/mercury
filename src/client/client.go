/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */


/*
  Package router implements a dispatch router.
  The normal order for operations on a router is:
    * create
    * connect to other routers
    * init
    * run
    * halt
  It is during the initialization step that the 
  configuration file is written that will be read at 
  router startup. So any connecting that you want to do
  should be done before you call Init.
  Every router is created with a normal-mode listener, so 
  you can always attach a client to it, or send it 
  management commands.
*/
package client

import ( "fmt"
         "errors"
         "os"
         "os/exec"
         "strings"
         "time"
         "strconv"
       )





var fp          = fmt.Fprintf
var module_name = "client"



type Client_state int

const (
  none         Client_state = iota
  initialized
  running
  halted
)




type Client struct {
  Name                  string
  Operation             string
  Id                    string
  Port                  string
  Path                  string
  log_path              string

  dispatch_install_root string
  proton_install_root   string

  n_messages            int

  cmd                 * exec.Cmd
  State                 Client_state
  max_message_length    int
  address               string

  throttle              string
}





func New_client ( name                  string,
                  operation             string,
                  id                    string,
                  port                  string,
                  path                  string,
                  log_path              string,
                  dispatch_install_root string,
                  proton_install_root   string,
                  n_messages            int,
                  max_message_length    int, 
                  address               string,
                  throttle              string ) ( * Client )  { 
  var c * Client

  c = & Client { Name                  : name,
                 Operation             : operation,
                 Id                    : id,
                 Port                  : port,
                 Path                  : path,
                 log_path              : log_path,
                 State                 : initialized,
                 dispatch_install_root : dispatch_install_root,
                 proton_install_root   : proton_install_root,
                 n_messages            : n_messages,
                 max_message_length    : max_message_length,
                 address               : address,
                 throttle              : throttle }

  return c
}





func ( c * Client ) Run ( ) {

  if c.State >= running {
    return
  }

  DISPATCH_LIBRARY_PATH := c.dispatch_install_root + "/lib"
  PROTON_LIBRARY_PATH   := c.proton_install_root   + "/lib64"
  LD_LIBRARY_PATH       := DISPATCH_LIBRARY_PATH +":"+ PROTON_LIBRARY_PATH

  DISPATCH_PYTHONPATH   := DISPATCH_LIBRARY_PATH + "/qpid-dispatch/python"
  DISPATCH_PYTHONPATH2  := DISPATCH_LIBRARY_PATH + "/python2.7/site-packages"

  PROTON_PYTHONPATH     := PROTON_LIBRARY_PATH + "/proton/bindings/python"

  PYTHONPATH            := DISPATCH_PYTHONPATH +":"+ DISPATCH_PYTHONPATH2 +":"+ PROTON_LIBRARY_PATH +":"+ PROTON_PYTHONPATH


  // Set up the environment for the router process.
  os.Setenv ( "LD_LIBRARY_PATH", LD_LIBRARY_PATH )
  os.Setenv ( "PYTHONPATH"     , PYTHONPATH )

  fp ( os.Stderr, "MDEBUG CLIENT export LD_LIBRARY_PATH=" + LD_LIBRARY_PATH + "\n" )
  fp ( os.Stderr, "MDEBUG CLIENT export PYTHONPATH=" + PYTHONPATH + "\n" )

  args := " --name " + c.Name + " --operation " + c.Operation + " --id " + c.Id + " --port " + c.Port + " --log " + c.log_path +"/"+ c.Name + " --messages " + strconv.Itoa(c.n_messages) + " --max_message_length " + strconv.Itoa(c.max_message_length) + " --address " + c.address + " --throttle " + c.throttle
  args_list := strings.Fields ( args )
  c.cmd = exec.Command ( c.Path,  args_list... )

  fp ( os.Stderr, "MDEBUG CLIENT command: |%s %s|\n", c.Path, args )

  // Start the client command. After the call to Start(),
  // the client is running detached.
  //fp ( os.Stderr, "running client |%s|\n", c.Name )
  c.cmd.Start()

  c.State = running
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





