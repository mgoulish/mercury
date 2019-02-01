package main

import (
  "fmt"
  "os"
  "sort"
  "strings"

  "utils"
  "lisp"
)


var ume     = utils.M_error
var umi     = utils.M_info



/*=====================================================================
  Command Functions
======================================================================*/


func verbose ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "verbose" ]
  parse_command_line ( context, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    context.verbose = true
  } else if val == "off" {
    context.verbose = false
  } else {
    ume ( "verbose: unknown value: |%s|", val )
  }

  context.network.Verbose ( context.verbose )
  umi ( context.verbose, "verbose: set to |%t|", context.verbose )
}





func echo ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "echo" ]
  parse_command_line ( context, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    context.echo = true
  } else if val == "off" {
    context.echo = false
  } else {
    ume ( "echo: unknown value: |%s|", val )
    return
  }

  umi ( context.verbose, "echo: set to |%s|", val )
}





func prompt ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "prompt" ]
  parse_command_line ( context, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    context.prompt = true
  } else if val == "off" {
    context.prompt = false
  } else {
    ume ( "prompt: unknown value: |%s|", val )
  }

  umi ( context.verbose, "prompt: set to |%s|", val )
}





func edges ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "edges" ]
  parse_command_line ( context, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value
  count       := cmd.unlabelable_int.int_value

  version_name := context.default_dispatch_version
  version_arg  := cmd.argmap [ "version" ]
  if version_arg.explicit {
    // The user entered a value.
    version_name = version_arg.string_value
  }

  // Make the edges.
  var edge_name string
  for i := 0; i < count; i ++ {
    context.edge_count ++
    edge_name = fmt.Sprintf ( "edge_%04d", context.edge_count )
    context.network.Add_edge ( edge_name, version_name )
    context.network.Connect_router ( edge_name, router_name )
    umi ( context.verbose, 
          "edges: added edge %s with version %s to router %s", 
          edge_name, 
          version_name, 
          router_name )
  }
}





func paths ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "paths" ]
  parse_command_line ( context, cmd, command_line )

  dispatch_path := cmd.argmap [ "dispatch" ]
  proton_path   := cmd.argmap [ "proton" ]
  mercury_path  := cmd.argmap [ "mercury" ]


  trouble := 0

  if dispatch_path.string_value == "" {
    ume ( "paths: dispatch path missing." )
    trouble ++
  }
  if _, err := os.Stat ( dispatch_path.string_value ); os.IsNotExist ( err ) {
    ume ( "paths: dispatch path does not exist: |%s|.", dispatch_path.string_value )
    trouble ++
  }


  if proton_path.string_value == "" {
    ume ( "paths: proton path missing." )
    trouble ++
  }
  if _, err := os.Stat ( proton_path.string_value ); os.IsNotExist ( err ) {
    ume ( "paths: proton path does not exist: |%s|.", proton_path.string_value )
    trouble ++
  }


  if mercury_path.string_value == "" {
    ume ( "paths: mercury path missing." )
    trouble ++
  }
  if _, err := os.Stat ( mercury_path.string_value ); os.IsNotExist ( err ) {
    ume ( "paths: mercury path does not exist: |%s|.", mercury_path.string_value )
    trouble ++
  }

  // If this is an interactive session, allow the user
  // to try again. If it's a script, it will die in a
  // little bit, but at least they'll know why.
  if trouble > 0 {
    return
  }


  context.dispatch_install_root = dispatch_path.string_value
  context.proton_install_root   = proton_path.string_value
  context.mercury_root          = mercury_path.string_value

  // The dispatch path defined in this command will be the default version.
  // If the user wants to define other versions they may do so with the 
  //'dispatch_version' command, but they are not forced to do so.
  context.default_dispatch_version = context.dispatch_install_root
  umi ( context.verbose,
        "paths: default dispatch version set to |%s|.", 
        context.default_dispatch_version )

  umi ( context.verbose, "paths: dispatch_path : |%s|", context.dispatch_install_root )
  umi ( context.verbose, "paths: proton_path   : |%s|", context.proton_install_root   )
  umi ( context.verbose, "paths: mercury_path  : |%s|", context.mercury_root  )

  // Now that paths are set, the network can be created.
  create_network ( context )
}





func send ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "send" ]
  parse_command_line ( context, cmd, command_line )

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
    edge_list = context.network.Get_router_edges ( router_with_edges )
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
    context.sender_count ++
    sender_name := fmt.Sprintf ( "send_%04d", context.sender_count )

    if variable_address {
      final_addr = fmt.Sprintf ( address, start_at )
      start_at ++
    }

    router_name = target_router_list[router_index]

    context.network.Add_sender ( sender_name,
                                 n_messages,
                                 max_message_length,
                                 router_name,
                                 final_addr,
                                 throttle )

    umi ( context.verbose,
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





func recv ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "recv" ]
  parse_command_line ( context, cmd, command_line )

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
    edge_list = context.network.Get_router_edges ( router_with_edges )
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
    context.receiver_count ++
    recv_name := fmt.Sprintf ( "recv_%04d", context.receiver_count )

    if variable_address {
      final_addr = fmt.Sprintf ( address, start_at )
      start_at ++
    }

    router_name := target_router_list[router_index]

    context.network.Add_receiver ( recv_name,
                                   n_messages,
                                   max_message_length,
                                   router_name,
                                   final_addr )

    umi ( context.verbose,
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
func dispatch_version ( context * Context, command_line * lisp.List ) {

  umi ( context.verbose, "version command is under construction." )
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

  // TODO store these at app level -- not in network.  context.network.Add_dispatch_version ( version_name, path )
  umi ( context.verbose, "dispatch_version: added version %s with path %s", version_name, path )
}





func routers ( context  * Context, command_line * lisp.List ) {
  cmd := context.commands [ "routers" ]
  parse_command_line ( context, cmd, command_line )

  count   := cmd.unlabelable_int.int_value
  version := cmd.unlabelable_string.string_value

  if version == "" {
    version = context.default_dispatch_version
  }

  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( context )
    context.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    umi ( context.verbose, "routers: added router |%s| with version |%s|.", router_name, version )
  }
}





func connect ( context  * Context, command_line * lisp.List ) {
  from_router, _ := command_line.Get_atom ( 1 )
  to_router, _   := command_line.Get_atom ( 2 )

  if from_router == "" || to_router == "" {
    ume ( "connect: from and to routers must both be specified." )
    return
  }

  context.network.Connect_router ( from_router, to_router )
  
  umi ( context.verbose, 
        "connect: connected router |%s| to router |%s|.", 
        from_router, 
        to_router )
}





func inc ( context  * Context, command_line * lisp.List ) {
  cmd := context.commands [ "inc" ]
  parse_command_line ( context, cmd, command_line )

  file_name := cmd.unlabelable_string.string_value

  if file_name == "" {
    ume ( "inc: I need a file name." )
    return
  }

  read_file ( context, file_name )
}





func linear ( context  * Context, command_line * lisp.List ) {
  cmd := context.commands [ "linear" ]
  parse_command_line ( context, cmd, command_line )

  count   := cmd.unlabelable_int.int_value
  version := cmd.unlabelable_string.string_value

  if version == "" {
    version = context.default_dispatch_version
  }


  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( context )
    context.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    umi ( context.verbose, "linear: added router |%s| with version |%s|.", router_name, version )
  }

  // And connect them.
  for index, name := range temp_names {
    if index < len(temp_names) - 1 {
      pitcher := name
      catcher := temp_names [ index + 1 ]
      context.network.Connect_router ( pitcher, catcher )
      umi ( context.verbose, "linear: connected router |%s| to router |%s|", pitcher, catcher )
    }
  }
}





func mesh ( context  * Context, command_line * lisp.List ) {
  cmd := context.commands [ "mesh" ]
  parse_command_line ( context, cmd, command_line )

  count   := cmd.unlabelable_int.int_value
  version := cmd.unlabelable_string.string_value

  if version == "" {
    version = context.default_dispatch_version
  }

  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( context )
    context.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    umi ( context.verbose, "mesh: added router |%s| with version |%s|.", router_name, version )
  }

  // And connect them.
    var catcher string
  for index, pitcher := range temp_names {
    if index < len(temp_names) - 1 {
      for j := index + 1; j < len(temp_names); j ++ {
        catcher = temp_names[j]
        context.network.Connect_router ( pitcher, catcher )
        if context.verbose {
          umi ( context.verbose, "mesh: connected router |%s| to router |%s|", pitcher, catcher )
        }
      }
    }
  }
}





func teds_diamond ( context  * Context, command_line * lisp.List ) {
  cmd := context.commands [ "teds_diamond" ]
  parse_command_line ( context, cmd, command_line )

  count   := 4
  version := cmd.unlabelable_string.string_value

  if version == "" {
    version = context.default_dispatch_version
  }

  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( context )
    context.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    umi ( context.verbose, "teds_diamond: added router |%s| with version |%s|.", router_name, version )
  }

  // And connect them.
    var catcher string
  for index, pitcher := range temp_names {
    if index < len(temp_names) - 1 {
      for j := index + 1; j < len(temp_names); j ++ {
        catcher = temp_names[j]
        context.network.Connect_router ( pitcher, catcher )
        if context.verbose {
          umi ( context.verbose, "teds_diamond: connected router |%s| to router |%s|", pitcher, catcher )
        }
      }
    }
  }

  // Now make the two outliers.
  outlier := get_next_interior_router_name ( context )
  context.network.Add_router ( outlier, version )
  umi ( context.verbose, "teds_diamond: added router |%s| with version |%s|.", outlier, version )
  catcher = temp_names[0]
  context.network.Connect_router ( outlier, catcher )
  umi ( context.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )
  catcher = temp_names[1]
  context.network.Connect_router ( outlier, catcher )
  umi ( context.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )


  outlier = get_next_interior_router_name ( context )
  context.network.Add_router ( outlier, version )
  umi ( context.verbose, "teds_diamond: added router |%s| with version |%s|.", outlier, version )
  catcher = temp_names[2]
  context.network.Connect_router ( outlier, catcher )
  umi ( context.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )
  catcher = temp_names[3]
  context.network.Connect_router ( outlier, catcher )
  umi ( context.verbose, "teds_diamond: connected router |%s| to router |%s|", outlier, catcher )
}





func run ( context  * Context, command_line * lisp.List ) {
  context.network.Init ( )
  context.network.Run  ( )

  context.network_running = true
  umi ( context.verbose, "run: network is running." )
}





func quit ( context * Context, command_line * lisp.List ) {
  if context.network_running {
    context.network.Halt ( )
  }
  umi ( context.verbose, "Mercury quitting." )
  os.Exit ( 0 )
}





func console_ports ( context * Context, command_line * lisp.List ) {
  context.network.Print_console_ports ( )
}





func is_a_command_name ( context * Context, name string ) (bool) {
  for _, cmd := range ( context.commands ) {
    if name == cmd.name {
      return true
    }
  }
  return false
}





func help_for_cmd ( context * Context, cmd * command ) {
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





func kill ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "kill" ]
  parse_command_line ( context, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value

  if err := context.network.Halt_router ( router_name ); err != nil {
    ume ( "kill: no such router |%s|", router_name )
    return
  }

  umi ( context.verbose, "kill: killing router |%s|", router_name )
}





func kill_and_restart ( context * Context, command_line * lisp.List ) {

  if ! context.network.Running {
    ume ( "kill_and_restart: the network is not running." )
    return
  }

  cmd := context.commands [ "kill_and_restart" ]
  parse_command_line ( context, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value
  pause       := cmd.unlabelable_int.int_value

  if err := context.network.Halt_and_restart_router ( router_name, pause ); err != nil {
    ume ( "kill_and_restart: no such router |%s|", router_name )
    return
  }
}





func help ( context * Context, command_line * lisp.List ) {
  // Get a sorted list of command names, 
  // and find the longest one.
  longest_command_name := 0
  cmd_names := make ( []string, 0 )
  for _, cmd := range context.commands {
    cmd_names = append ( cmd_names, cmd.name )
    if len(cmd.name) > longest_command_name {
      longest_command_name = len(cmd.name)
    }
  }
  sort.Strings ( cmd_names )

  // If there is an arg on the command line, the 
  // user is asking for help with a specific command
  if len(command_line.Elements) > 1 {
    requested_command, _ := command_line.Get_atom ( 1 )
    if is_a_command_name ( context, requested_command ) {
      cmd := context.commands [ requested_command ]
      help_for_cmd ( context, cmd )
    } else {
      // The user did not enter a command name.
      // Maybe it is the first few letters of a command?
      // Give him the first one that matches.
      for _, cmd_name := range cmd_names {
        if strings.HasPrefix ( cmd_name, requested_command ) {
          cmd := context.commands [ cmd_name ]
          help_for_cmd ( context, cmd ) 
        }
      }
    }
  } else {
    // No arg on command line. The user 
    // wants all commands. Get the names.
    for _, name := range cmd_names {
      cmd      := context.commands [ name ]
      pad_size := longest_command_name - len(name)
      pad      := strings.Repeat(" ", pad_size)
      fp ( os.Stdout, "    %s%s : %s\n", name, pad, cmd.help )
    }
  }
}





