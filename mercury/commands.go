package main

import (
  "fmt"
  "os"
  "strings"

  "lisp"
)



/*=====================================================================
  Command Functions
======================================================================*/


func verbose ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "verbose" ]
  parse_command_line ( context, cmd, command_line )

  val := cmd.unlabelable_string.string_value
  if val == "on" {
    context.verbose = true
    m_info ( context, "verbose: on" )
  } else if val == "off" {
    context.verbose = false
  } else {
    fp ( os.Stdout, " ERROR do something here.\n" )
  }
}





func edges ( context * Context, command_line * lisp.List ) {
  cmd := context.commands [ "edges" ]
  parse_command_line ( context, cmd, command_line )

  router_name := cmd.unlabelable_string.string_value
  count       := cmd.unlabelable_int.int_value

  version_name := context.first_version_name  // This will be the default.
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
    m_info ( context, 
             "edges: added edge %s with version %s to router %s", 
             edge_name, 
             version_name, 
             router_name )
  }
}





func paths ( context * Context, arglist * lisp.List ) {

  dispatch_path := arglist.Match_atom ( "dispatch" )
  proton_path   := arglist.Match_atom ( "proton" )
  mercury_path  := arglist.Match_atom ( "mercury" )

  trouble := 0

  if dispatch_path == "" {
    m_error ( "paths: dispatch path missing." )
    trouble ++
  }
  if _, err := os.Stat ( dispatch_path ); os.IsNotExist ( err ) {
    m_error ( "paths: dispatch path does not exist: |%s|.", dispatch_path )
    trouble ++
  }

  if proton_path == "" {
    m_error ( "paths: proton path missing." )
    trouble ++
  }
  if _, err := os.Stat ( mercury_path ); os.IsNotExist ( err ) {
    m_error ( "paths: mercury path does not exist: |%s|.", mercury_path )
    trouble ++
  }

  if mercury_path == "" {
    m_error ( "paths: mercury path missing." )
    trouble ++
  }
  if _, err := os.Stat ( proton_path ); os.IsNotExist ( err ) {
    m_error ( "paths: proton path does not exist: |%s|.", proton_path )
    trouble ++
  }

  if trouble > 0 {
    os.Exit ( 1 )
  }

  context.dispatch_install_root = dispatch_path
  context.proton_install_root   = proton_path
  context.mercury_root          = mercury_path

  m_info ( context, "paths: dispatch_path : |%s|", dispatch_path )
  m_info ( context, "paths: proton_path   : |%s|", proton_path   )
  m_info ( context, "paths: mercury_path  : |%s|", mercury_path  )

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

  target_router_list = append ( target_router_list, router_name )
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
  fp ( os.Stdout, " address init to |%s|\n", address )

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

    router_name := target_router_list[router_index]

    context.network.Add_sender ( sender_name,
                                 n_messages,
                                 max_message_length,
                                 router_name,
                                 final_addr,
                                 throttle )

    m_info ( context,
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

  target_router_list = append ( target_router_list, router_name )
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

    m_info ( context,
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
  version_name, err := command_line.Get_atom ( 1 )
  if err != nil {
    m_error ( "dispatch_version: error on version name: %s", err.Error() )
    return
  }

  path, err := command_line.Get_atom ( 2 )
  if err != nil {
    m_error ( "dispatch_version: error on path: %s", err.Error() )
    return
  }

  if _, err := os.Stat ( path ); os.IsNotExist ( err ) {
    m_error ( "dispatch_version: %s version path does not exist: |%s|.", version_name, path )
    return
  }

  context.network.Add_dispatch_version ( version_name, path )
  m_info ( context, "dispatch_version: added version %s with path %s", version_name, path )

  // If this one is first, store it.
  // It will become the default.
  if context.first_version_name == "" {
    context.first_version_name = version_name
    m_info ( context, "dispatch_version: version %s is default.", context.first_version_name )
  }
}





func linear ( context  * Context, command_line * lisp.List ) {
  cmd := context.commands [ "linear" ]
  parse_command_line ( context, cmd, command_line )

  count   := cmd.unlabelable_int.int_value
  version := cmd.unlabelable_string.string_value

  if version == "" {
    version = context.first_version_name
  }

  // Make the requested routers.
  var router_name string
  var temp_names [] string
  for i := 0; i < count; i ++ {
    router_name = get_next_interior_router_name ( context )
    context.network.Add_router ( router_name, version )
    temp_names = append ( temp_names, router_name )
    m_info ( context, "linear: added router |%s| with version |%s|.", router_name, version )
  }

  // And connect them.
  for index, name := range temp_names {
    if index < len(temp_names) - 1 {
      pitcher := name
      catcher := temp_names [ index + 1 ]
      context.network.Connect_router ( pitcher, catcher )
      m_info ( context, "linear: connected router |%s| to router |%s|", pitcher, catcher )
    }
  }
}





func run ( context  * Context, command_line * lisp.List ) {
  context.network.Init ( )
  context.network.Run  ( )

  context.network_running = true
  m_info ( context, "run: network is running." )
}





func quit ( context * Context, command_line * lisp.List ) {
  if context.network_running {
    context.network.Halt ( )
  }
  m_info ( context, "Mercury quitting." )
  os.Exit ( 0 )
}





func console_ports ( context  * Context, command_line * lisp.List ) {
  context.network.Print_console_ports ( )
}





