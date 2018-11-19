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



package utils

import ( "fmt"
         "os"
         "net"
)


var fp = fmt.Fprintf





func Available_port () ( port string, err error ) {

  server, err := net.Listen ( "tcp", ":0" )
  if err != nil {
    return "", err
  }
  defer server.Close()

  hostString := server.Addr().String()

  _, portString, err := net.SplitHostPort(hostString)
  if err != nil {
    return "", err
  }

  return portString, nil
}




func Find_or_create_dir ( path string ) {
  _, err := os.Stat ( path )
  if os.IsNotExist ( err ) {
    err = os.MkdirAll ( path, os.ModePerm )
    if err != nil {
        fp ( os.Stderr, "error creating network_root |%s| : %v\n", path, err )
        os.Exit ( 1 )
    }
  }
}



func End_test_and_exit ( result_path string, test_error string ) {
  f, err := os.Create ( result_path + "/result" )
  if err != nil {
    fp ( os.Stderr, "Can't write results file!\n" )
    os.Exit ( 1 )
  }
  defer f.Close ( )

  if test_error == "" {
    fp ( f, "success\n" )
  } else {
    fp ( f, "failure : %s\n", test_error )
  }

  os.Exit ( 0 )
}


func Make_paths ( mercury_root, test_id, test_name string ) ( router_path, result_path, config_path, log_path string ) {
  dispatch_install_root := os.Getenv ( "DISPATCH_INSTALL_ROOT" )
  router_path            = dispatch_install_root + "/sbin/qdrouterd"
  result_path            = mercury_root + "/results/" + test_name + "/" + test_id
  config_path            = result_path + "/config"
  log_path               = result_path + "/log"

  return router_path, result_path, config_path, log_path
}





func Memory_usage ( pid int ) ( usage int ) {
  
  proc_file_name := "/proc/" + string(pid) + "/statm"
  fp ( os.Stderr, "My file name is |%s|\n", proc_file_name );

  return 0
}
