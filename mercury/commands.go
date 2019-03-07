package main

import (
  "fmt"
  "os"
  "sort"
  "strings"
  "math/rand"

  "lisp"
)




/*=====================================================================
  Command Functions
======================================================================*/


func usage ( merc * Merc, command_name string ) {
  fp ( os.Stdout, "    usage for command |%s|\n", command_name )
}





func verbose ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "verbose" ]
  parse_command_line ( merc, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    merc.verbose = true
  } else if val == "off" {
    merc.verbose = false
  } else {
    ume ( "verbose: unknown value: |%s|", val )
  }

  merc.network.Verbose ( merc.verbose )
  umi ( merc.verbose, "verbose: set to |%t|", merc.verbose )
}





func echo ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "echo" ]
  parse_command_line ( merc, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    merc.echo = true
  } else if val == "off" {
    merc.echo = false
  } else {
    ume ( "echo: unknown value: |%s|", val )
    return
  }

  umi ( merc.verbose, "echo: set to |%s|", val )
}





func prompt ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "prompt" ]
  parse_command_line ( merc, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    merc.prompt = true
  } else if val == "off" {
    merc.prompt = false
  } else {
    ume ( "prompt: unknown value: |%s|", val )
  }

  umi ( merc.verbose, "prompt: set to |%s|", val )
}





func edges ( merc * Merc, command_line * lisp.List ) {
  /*
  cmd := merc.commands [ "edges" ]
  parse_command_line ( merc, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value
  count       := cmd.unlabelable_int.int_value

  version_name := merc.default_dispatch_version
  version_arg  := cmd.argmap [ "version" ]
  if version_arg.explicit {
    // The user entered a value.
    version_name = version_arg.string_value
  }

  // Make the edges.
  var edge_name string
  for i := 0; i < count; i ++ {
    merc.edge_count ++
    edge_name = fmt.Sprintf ( "edge_%04d", merc.edge_count )
    merc.network.Add_edge ( edge_name, version_name )
    merc.network.Connect_router ( edge_name, router_name )
    umi ( merc.verbose, 
          "edges: added edge %s with version %s to router %s", 
          edge_name, 
          version_name, 
          router_name )
  }
  */
}





func seed ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "seed" ]
  parse_command_line ( merc, cmd, command_line )

  value := cmd.unlabelable_int.int_value
  rand.Seed ( int64 ( value ) )
}





func version_roots ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "version_roots" ]
  parse_command_line ( merc, cmd, command_line )

  name     := cmd.argmap [ "name" ]     . string_value
  proton   := cmd.argmap [ "proton" ]   . string_value
  dispatch := cmd.argmap [ "dispatch" ] . string_value

  if name == "" || proton == "" || dispatch == "" {
    help_for_command ( merc, cmd )
    return
  }

  merc.network.Add_version_with_roots ( name, proton, dispatch )
}





func send ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "send" ]
  parse_command_line ( merc, cmd, command_line )

  // The user may specify target routers two different ways:
  // with the 'router' arg, which specifies a single interior
  // node, or with the 'edges' arg, which specifies all the 
  // edge-routers attached to an interior router.
  // This list will accumulate all targets.
  target_router_list := make ( [] string, 0 )

  router_name := cmd.unlabelable_string.string_value
  count       := cmd.unlabelable_int.int_value

  if router_name != "" {
    target_router_list = append ( target_router_list, router_name )
  }
  router_with_edges := cmd.argmap [ "edges" ] . string_value

  var edge_list [] string
  if router_with_edges != "" {
    edge_list = merc.network.Get_router_edges ( router_with_edges )
  }
  target_router_list = append ( target_router_list, edge_list ... )

  // If it turns out that this address is not variable, 
  // then this 'final' address is the only one we will
  // use. But if address is variable this value will get
  // replaced every time through the loop with the changing
  // value of the variable address: add_r, addr_2, etc.
  address    := cmd.argmap [ "address" ] . string_value
  final_addr := address

  // Is this address variable? It is if it contains a "%d" somewhere.
  // I have to test for this because fmt.Sprintf treats it as an 
  // error if I have a format string that contains no "%d" and I try
  // to print it with what would be an unused int arg.
  variable_address := false
  if strings.Contains ( address, "%d" ) {
    variable_address = true
  }

  start_at           := cmd.argmap [ "start_at"           ] . int_value
  n_messages         := cmd.argmap [ "n_messages"         ] . int_value
  max_message_length := cmd.argmap [ "max_message_length" ] . int_value
  throttle           := cmd.argmap [ "throttle"           ] . string_value

  router_index := 0
  for i := 0; i < count; i ++ {
    merc.sender_count ++
    sender_name := fmt.Sprintf ( "send_%04d", merc.sender_count )

    if variable_address {
      final_addr = fmt.Sprintf ( address, start_at )
      start_at ++
    }

    router_name = target_router_list[router_index]

    merc.network.Add_sender ( sender_name,
                                 n_messages,
                                 max_message_length,
                                 router_name,
                                 final_addr,
                                 throttle )

    umi ( merc.verbose,
          "send: added sender |%s| with addr |%s| to router |%s|.", 
          sender_name,
          final_addr,
          router_name )

    router_index ++
    if router_index >= len(target_router_list) {
      router_index = 0
    }
  }
}





func recv ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "recv" ]
  parse_command_line ( merc, cmd, command_line )

  // The user may specify target routers two different ways:
  // with the 'router' arg, which specifies a single interior
  // node, or with the 'edges' arg, which specifies all the 
  // edge-routers attached to an interior router.
  // This list will accumulate all targets.
  target_router_list := make ( [] string, 0 )

  router_name := cmd.unlabelable_string.string_value
  count       := cmd.unlabelable_int.int_value

  if router_name != "" {
    target_router_list = append ( target_router_list, router_name )
  }
  router_with_edges := cmd.argmap [ "edges" ] . string_value

  var edge_list [] string
  if router_with_edges != "" {
    edge_list = merc.network.Get_router_edges ( router_with_edges )
  }
  target_router_list = append ( target_router_list, edge_list ... )

  // If it turns out that this address is not variable, 
  // then this 'final' address is the only one we will
  // use. But if address is variable this value will get
  // replaced every time through the loop with the changing
  // value of the variable address: add_r, addr_2, etc.
  address    := cmd.argmap [ "address" ] . string_value
  final_addr := address

  // Is this address variable? It is if it contains a "%d" somewhere.
  // I have to test for this because fmt.Sprintf treats it as an 
  // error if I have a format string that contains no "%d" and I try
  // to print it with what would be an unused int arg.
  variable_address := false
  if strings.Contains ( address, "%d" ) {
    variable_address = true
  }

  start_at           := cmd.argmap [ "start_at"           ] . int_value
  n_messages         := cmd.argmap [ "n_messages"         ] . int_value
  max_message_length := cmd.argmap [ "max_message_length" ] . int_value

  router_index := 0
  for i := 0; i < count; i ++ {
    merc.receiver_count ++
    recv_name := fmt.Sprintf ( "recv_%04d", merc.receiver_count )

    if variable_address {
      final_addr = fmt.Sprintf ( address, start_at )
      start_at ++
    }

    router_name := target_router_list[router_index]

    merc.network.Add_receiver ( recv_name,
                                n_messages,
                                max_message_length,
                                router_name,
                                final_addr )

    umi ( merc.verbose,
          "recv: added |%s| with addr |%s| to router |%s|.", 
          recv_name,
          final_addr,
          router_name )

    router_index ++
    if router_index >= len(target_router_list) {
      router_index = 0
    }
  }
}





// This command has its own special magic syntax, so it 
// parses the command lline its own way.
func dispatch_version ( merc * Merc, command_line * lisp.List ) {

  umi ( merc.verbose, "version command is under construction." )
  return

  version_name, err := command_line.Get_atom ( 1 )
  if err != nil {
    ume ( "dispatch_version: error on version name: %s", err.Error() )
    return
  }

  path, err := command_line.Get_atom ( 2 )
  if err != nil {
    ume ( "dispatch_version: error on path: %s", err.Error() )
    return
  }

  if _, err := os.Stat ( path ); os.IsNotExist ( err ) {
    ume ( "dispatch_version: %s version path does not exist: |%s|.", version_name, path )
    return
  }

  umi ( merc.verbose, "dispatch_version: added version %s with path %s", version_name, path )
}





func routers ( merc  * Merc, command_line * lisp.List ) {
  if len(merc.network.Versions) < 1 {
    ume ( "routers: You must define at least one version before creating routers." )
    return
  }

  cmd := merc.commands [ "routers" ]
  parse_command_line ( merc, cmd, command_line )

  count   := cmd.unlabelable_int.int_value
  version_name := cmd.unlabelable_string.string_value

  // If no version name was supplied, use default.
  if version_name == "" {
    version_name = merc.network.Default_version.Name
  }

  // Make the requested routers.
  var router_name string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( merc )
    merc.network.Add_router ( router_name, 
                              version_name, 
                              merc.session.config_path,
                              merc.session.log_path )
    umi ( merc.verbose, 
          "routers: added router |%s| with version |%s|.", 
          router_name, 
          version_name )
  }
}





func connect ( merc  * Merc, command_line * lisp.List ) {
  from_router, _ := command_line.Get_atom ( 1 )
  to_router, _   := command_line.Get_atom ( 2 )

  if from_router == "" || to_router == "" {
    ume ( "connect: from and to routers must both be specified." )
    return
  }

  merc.network.Connect_router ( from_router, to_router )
  
  umi ( merc.verbose, 
        "connect: connected router |%s| to router |%s|.", 
        from_router, 
        to_router )
}





func inc ( merc  * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "inc" ]
  parse_command_line ( merc, cmd, command_line )

  file_name := cmd.unlabelable_string.string_value

  if file_name == "" {
    ume ( "inc: I need a file name." )
    return
  }

  if merc.verbose {
    umi ( merc.verbose,
          "inc: |%s|", 
          file_name )
  }

  read_file ( merc, file_name )
}





func linear ( merc  * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "linear" ]
  parse_command_line ( merc, cmd, command_line )

  count             := cmd.unlabelable_int.int_value
  requested_version := cmd.unlabelable_string.string_value

  var version string

  if requested_version == "" {
    version = merc.network.Default_version.Name
  } 


  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( merc )

    if requested_version == "random" {
      n_versions   := len(merc.network.Versions)
      random_index := rand.Intn ( n_versions )
      version = merc.network.Versions[random_index].Name
    }

    merc.network.Add_router ( router_name, 
                              version,
                              merc.session.config_path,
                              merc.session.log_path )
    temp_names = append ( temp_names, router_name )
    umi ( merc.verbose, "linear: added router |%s| with version |%s|.", router_name, version )
  }

  // And connect them.
  for index, name := range temp_names {
    if index < len(temp_names) - 1 {
      pitcher := name
      catcher := temp_names [ index + 1 ]
      merc.network.Connect_router ( pitcher, catcher )
      umi ( merc.verbose, "linear: connected router |%s| to router |%s|", pitcher, catcher )
    }
  }
}





func mesh ( merc  * Merc, command_line * lisp.List ) {
  /*
  cmd := merc.commands [ "mesh" ]
  parse_command_line ( merc, cmd, command_line )

  count   := cmd.unlabelable_int.int_value
  version := cmd.unlabelable_string.string_value

  if version == "" {
    version = merc.default_dispatch_version
  }

  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( merc )
    merc.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    umi ( merc.verbose, "mesh: added router |%s| with version |%s|.", router_name, version )
  }

  // And connect them.
    var catcher string
  for index, pitcher := range temp_names {
    if index < len(temp_names) - 1 {
      for j := index + 1; j < len(temp_names); j ++ {
        catcher = temp_names[j]
        merc.network.Connect_router ( pitcher, catcher )
        if merc.verbose {
          umi ( merc.verbose, "mesh: connected router |%s| to router |%s|", pitcher, catcher )
        }
      }
    }
  }
  */
}





func teds_diamond ( merc  * Merc, command_line * lisp.List ) {
  /*
  cmd := merc.commands [ "teds_diamond" ]
  parse_command_line ( merc, cmd, command_line )

  count   := 4
  version := cmd.unlabelable_string.string_value

  if version == "" {
    version = merc.default_dispatch_version
  }

  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( merc )
    merc.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    umi ( merc.verbose, "teds_diamond: added router |%s| with version |%s|.", router_name, version )
  }

  // And connect them.
    var catcher string
  for index, pitcher := range temp_names {
    if index < len(temp_names) - 1 {
      for j := index + 1; j < len(temp_names); j ++ {
        catcher = temp_names[j]
        merc.network.Connect_router ( pitcher, catcher )
        if merc.verbose {
          umi ( merc.verbose, "teds_diamond: connected router |%s| to router |%s|", pitcher, catcher )
        }
      }
    }
  }

  // Now make the two outliers.
  outlier := get_next_interior_router_name ( merc )
  merc.network.Add_router ( outlier, version )
  umi ( merc.verbose, "teds_diamond: added router |%s| with version |%s|.", outlier, version )
  catcher = temp_names[0]
  merc.network.Connect_router ( outlier, catcher )
  umi ( merc.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )
  catcher = temp_names[1]
  merc.network.Connect_router ( outlier, catcher )
  umi ( merc.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )


  outlier = get_next_interior_router_name ( merc )
  merc.network.Add_router ( outlier, version )
  umi ( merc.verbose, "teds_diamond: added router |%s| with version |%s|.", outlier, version )
  catcher = temp_names[2]
  merc.network.Connect_router ( outlier, catcher )
  umi ( merc.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )
  catcher = temp_names[3]
  merc.network.Connect_router ( outlier, catcher )
  umi ( merc.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )
  */
}





func run ( merc  * Merc, command_line * lisp.List ) {
  merc.network.Init ( )
  merc.network.Run  ( )

  merc.network_running = true
  umi ( merc.verbose, "run: network is running." )
}





func quit ( merc * Merc, command_line * lisp.List ) {
  if merc.network_running {
    merc.network.Halt ( )
  }
  umi ( merc.verbose, "Mercury quitting." )
  os.Exit ( 0 )
}





func console_ports ( merc * Merc, command_line * lisp.List ) {
  merc.network.Print_console_ports ( )
}





func is_a_command_name ( merc * Merc, name string ) (bool) {
  for _, cmd := range ( merc.commands ) {
    if name == cmd.name {
      return true
    }
  }
  return false
}





func help_for_command ( merc * Merc, cmd * command ) {
  fp ( os.Stdout, "\n    %s : %s\n", cmd.name, cmd.help )

  longest_arg_name := 0
  var temp_names [] string
  for _, arg := range cmd.argmap {
    temp_names = append ( temp_names, arg.name )
    if len(arg.name) > longest_arg_name {
      longest_arg_name = len(arg.name)
    }
  }

  sort.Strings ( temp_names )

  for _, arg_name := range temp_names {
    pad_size := longest_arg_name - len(arg_name)
    pad      := strings.Repeat(" ", pad_size)
    arg      := cmd.argmap [ arg_name ]

    str := fmt.Sprintf ( "      %s%s : ", arg_name, pad )
    if arg.unlabelable {
      str = fmt.Sprintf ( "%sUNLABELABLE ", str )
    }
    if arg.help != "" {
      str = fmt.Sprintf ( "%s%s", str, arg.help )
    }
    if arg.default_value != "" {
      str = fmt.Sprintf ( "%s -- default: |%s|", str, arg.default_value )
    }
    fp ( os.Stdout, "%s\n", str )
  }
  fp ( os.Stdout, "\n\n" )
}





func kill ( merc * Merc, command_line * lisp.List ) {
  cmd := merc.commands [ "kill" ]
  parse_command_line ( merc, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value

  if err := merc.network.Halt_router ( router_name ); err != nil {
    ume ( "kill: no such router |%s|", router_name )
    return
  }

  umi ( merc.verbose, "kill: killing router |%s|", router_name )
}





func kill_and_restart ( merc * Merc, command_line * lisp.List ) {

  if ! merc.network.Running {
    ume ( "kill_and_restart: the network is not running." )
    return
  }

  cmd := merc.commands [ "kill_and_restart" ]
  parse_command_line ( merc, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value
  pause       := cmd.unlabelable_int.int_value

  if err := merc.network.Halt_and_restart_router ( router_name, pause ); err != nil {
    ume ( "kill_and_restart: no such router |%s|", router_name )
    return
  }
}





func help ( merc * Merc, command_line * lisp.List ) {
  // Get a sorted list of command names, 
  // and find the longest one.
  longest_command_name := 0
  cmd_names := make ( []string, 0 )
  for _, cmd := range merc.commands {
    cmd_names = append ( cmd_names, cmd.name )
    if len(cmd.name) > longest_command_name {
      longest_command_name = len(cmd.name)
    }
  }
  sort.Strings ( cmd_names )

  // If there is an arg on the command line, the 
  // user is asking for help with a specific command
  if command_line != nil && len(command_line.Elements) > 1 {
    requested_command, _ := command_line.Get_atom ( 1 )
    if is_a_command_name ( merc, requested_command ) {
      cmd := merc.commands [ requested_command ]
      help_for_command ( merc, cmd )
    } else {
      // The user did not enter a command name.
      // Maybe it is the first few letters of a command?
      // Give him the first one that matches.
      for _, cmd_name := range cmd_names {
        if strings.HasPrefix ( cmd_name, requested_command ) {
          cmd := merc.commands [ cmd_name ]
          help_for_command ( merc, cmd ) 
        }
      }
    }
  } else {
    // No arg on command line. The user 
    // wants all commands. Get the names.
    for _, name := range cmd_names {
      cmd      := merc.commands [ name ]
      pad_size := longest_command_name - len(name)
      pad      := strings.Repeat(" ", pad_size)
      fp ( os.Stdout, "    %s%s : %s\n", name, pad, cmd.help )
    }
  }
}





