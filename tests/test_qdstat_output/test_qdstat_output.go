package main

import ( "fmt"
         qdo "qdstat_output"
         "os"
         "utils"
       )



var upl = utils.Print_log
var fp  = fmt.Fprintf





func
main ( ) {
  sample_output_1 := `Types
type                        size   batch  thread-max  total  in-threads  rebal-in  rebal-out
==============================================================================================
qd_bitmask_t                24     64     128         512    512         0         0
  qd_buffer_t                 536    64     128         320    256         0         1
  qd_composed_field_t         64     64     128         256    256         0         0
  qd_composite_t              112    64     128         256    256         0         0
  qd_connection_t             2,360  16     32          128    128         0         0
  qd_connector_t              488    64     128         64     64          0         0
  qd_hash_handle_t            16     64     128         384    384         0         0
  qd_hash_item_t              32     64     128         384    384         0         0
  qd_iterator_t               160    64     128         320    192         1         3
  qd_link_ref_t               24     64     128         64     64          0         0
  qd_link_t                   104    64     128         704    704         0         0
  qd_listener_t               432    64     128         64     64          0         0
  qd_log_entry_t              2,112  16     32          176    176         0         0
  qd_message_content_t        1,064  64     128         256    256         0         0
  qd_message_t                160    64     128         320    320         0         0
  qd_node_t                   56     64     128         64     64          0         0
  qd_parse_node_t             104    64     128         128    128         0         0
  qd_parsed_field_t           88     64     128         320    320         0         0
  qd_parsed_turbo_t           64     64     128         256    256         0         0
  qd_timer_t                  56     64     128         128    128         0         0
  qdr_action_t                160    64     128         320    256         17        18
  qdr_addr_endpoint_state_t   40     64     128         128    128         0         0
  qdr_address_config_t        64     64     128         64     64          0         0
  qdr_address_t               344    64     128         256    256         0         0
  qdr_connection_info_t       88     64     128         256    256         0         0
  qdr_connection_t            544    64     128         256    256         0         0
  qdr_connection_work_t       48     64     128         512    384         1         3
  qdr_delivery_ref_t          24     64     128         64     64          0         0
  qdr_delivery_t              256    64     128         320    320         0         0
  qdr_error_t                 24     64     128         128    128         0         0
  qdr_field_t                 40     64     128         256    256         1         1
  qdr_forward_deliver_info_t  32     64     128         64     64          0         0
  qdr_general_work_t          80     64     128         256    192         0         1
  qdr_link_ref_t              24     64     128         1,152  1,152       0         0
  qdr_link_t                  440    64     128         768    768         0         0
  qdr_link_work_t             48     64     128         320    320         0         0
  qdr_node_t                  64     64     128         64     64          0         0
  qdr_query_t                 344    64     128         64     64          0         0
  qdr_terminus_t              64     64     128         384    256         0         2
  qdrc_endpoint_t             24     64     128         128    128         0         0
  qdtm_router_t               16     64     128         128    128         0         0
`

  sample_output_2 := `Types
type                        size   batch  thread-max  total  in-threads  rebal-in  rebal-out
==============================================================================================
qd_bitmask_t                24     64     128         512    512         0         0
  qd_buffer_t                 536    64     128         320    256         0         1
  qd_composed_field_t         64     64     128         256    256         0         0
  qd_composite_t              112    64     128         256    256         0         0
  qd_connection_t             2,360  16     32          128    128         0         0
  qd_connector_t              488    64     128         64     64          0         0
  qd_hash_handle_t            16     64     128         384    384         0         0
  qd_hash_item_t              32     64     128         384    384         0         0
  qd_iterator_t               160    64     128         320    192         1         3
  qd_link_ref_t               24     64     128         64     64          0         0
  qd_link_t                   104    64     128         704    704         0         0
  qd_listener_t               432    64     128         64     64          0         0
  qd_log_entry_t              2,112  16     32          176    176         0         0
  qd_message_content_t        1,064  64     128         256    256         0         0
  qd_message_t                160    64     128         320    320         0         0
  qd_node_t                   56     64     128         64     64          0         0
  qd_parse_node_t             104    64     128         128    192         0         0
  qd_parsed_field_t           88     64     128         320    320         0         0
  qd_parsed_turbo_t           64     64     128         256    256         0         0
  qd_timer_t                  56     64     128         128    128         0         0
  qdr_action_t                160    64     128         320    256         17        18
  qdr_addr_endpoint_state_t   40     64     128         128    128         0         0
  qdr_address_config_t        64     64     128         64     64          0         0
  qdr_address_t               344    64     128         256    256         0         0
  qdr_connection_info_t       88     64     128         256    256         0         0
  qdr_connection_t            544    64     128         256    256         0         0
  qdr_connection_work_t       48     64     128         512    384         1         3
  qdr_delivery_ref_t          24     64     128         64     64          0         0
  qdr_delivery_t              256    64     128         320    320         0         0
  qdr_error_t                 24     64     128         128    128         0         0
  qdr_field_t                 40     64     128         256    256         1         1
  qdr_forward_deliver_info_t  32     64     128         64     64          0         0
  qdr_general_work_t          80     64     128         256    192         0         1
  qdr_link_ref_t              24     64     128         1,152  1,152       0         0
  qdr_link_t                  440    64     128         768    768         0         0
  qdr_link_work_t             48     64     128         320    320         0         0
  qdr_node_t                  64     64     128         64     64          0         0
  qdr_query_t                 344    64     128         64     64          0         0
  qdr_terminus_t              64     64     128         384    256         0         2
  qdrc_endpoint_t             24     64     128         128    128         0         0
  qdtm_router_t               16     64     128         128    128         0         0
`
  
  qdstat_result_1 := qdo.New_qdstat_output ( sample_output_1 )
  qdstat_result_2 := qdo.New_qdstat_output ( sample_output_2 )

  /*
  for _, s := range qdstat_result_1.Lines {
    s.Print ( )
  }

  for _, s := range qdstat_result_2.Lines {
    s.Print ( )
  }
  */

  // qdstat_diff := qdo.Diff ( sample_output_2, sample_output_1 )
  diffed_output := qdstat_result_2.Diff ( qdstat_result_1 )

  fp ( os.Stderr, "DIFFED -----------------------------------------\n" )
  diffed_output.Print_nonzero ( )

}





