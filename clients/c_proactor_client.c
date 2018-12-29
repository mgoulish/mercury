#include <proton/codec.h>
#include <proton/delivery.h>
#include <proton/engine.h>
#include <proton/event.h>
#include <proton/listener.h>
#include <proton/message.h>
#include <proton/proactor.h>
#include <proton/sasl.h>
#include <proton/types.h>
#include <proton/version.h>

#include <inttypes.h>
#include <memory.h>
#include <pthread.h>
#include <stdarg.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/types.h>
#include <time.h>
#include <unistd.h>





#define MAX_NAME   1000
#define MAX_LINKS  1000


typedef 
struct context_s 
{
  pn_link_t       * links [ MAX_LINKS ];
  int               link_count;
  char              name [ MAX_NAME ];
  int               sending;
  char              id [ MAX_NAME ];
  char              host [ MAX_NAME ];
  size_t            max_send_length,
                    max_receive_length,
                    message_length,
                    outgoing_buffer_size;
  char            * outgoing_buffer;
  char              incoming_message [ 10000 ];
  char            * port;
  char            * log_file_name;
  FILE            * log_file;
  int               messages_sent;
  int               received;
  pn_message_t    * message;
  uint64_t          total_bytes_sent,
                    total_bytes_received;
  int               messages;
  int               next_received_report;
  size_t            credit_window;
  char              path [ MAX_NAME ];
  pn_proactor_t   * proactor;
  pn_listener_t   * listener;
  pn_connection_t * connection;
  size_t            accepted;
}
context_t,
* context_p;












int
rand_int ( int one_past_max )
{
  double zero_to_one = (double) rand() / (double) RAND_MAX;
  return (int) (zero_to_one * (double) one_past_max);
}





double
get_timestamp ( void )
{
  struct timeval t;
  gettimeofday ( & t, 0 );
  return t.tv_sec + ((double) t.tv_usec) / 1000000.0;
}





void
log ( context_p context, char const * format, ... )
{
  if ( ! context->log_file )
    return;

  fprintf ( context->log_file, "%.6f : ", get_timestamp() );
  va_list ap;
  va_start ( ap, format );
  vfprintf ( context->log_file, format, ap );
  va_end ( ap );
  fflush ( context->log_file );
}





void
make_random_message ( context_p context )
{
  context->message_length = rand_int ( context->max_send_length );
  for ( int i = 0; i < context->message_length; ++ i )
    context->outgoing_buffer [ i ] = uint8_t ( rand_int ( 256 ) );
}





size_t 
encode_outgoing_message ( context_p context ) 
{
  int err = 0;
  size_t size = context->outgoing_buffer_size;

  if ( 0 == (err = pn_message_encode ( context->message, context->outgoing_buffer, & size) ) )
    return size;

  if ( err == PN_OVERFLOW ) 
  {
    log ( context, "error: overflowed outgoing_buffer_size == %d\n", context->outgoing_buffer_size );
    exit ( 1 );
  } 
  else
  if ( err != 0 ) 
  {
    log ( context, 
          "error encoding message: %s |%s|\n", 
          pn_code ( err ), 
          pn_error_text(pn_message_error ( context->message ) ) 
        );
    exit ( 1 );
  }

  return 0; // unreachable
}





void 
decode_message ( context_p context, pn_delivery_t * delivery ) 
{
  pn_message_t * msg  = context->message;
  pn_link_t    * link = pn_delivery_link ( delivery );
  ssize_t        incoming_size = pn_delivery_pending ( delivery );

  if ( incoming_size >= context->max_receive_length )
  {
    log ( context, "incoming message too big: %d.\n", incoming_size );
    exit ( 1 );
  }

  pn_link_recv ( link, context->incoming_message, incoming_size);
  pn_message_clear ( msg );

  if ( pn_message_decode ( msg, context->incoming_message, incoming_size ) ) 
  {
    log ( context, 
          "error from pn_message_decode: |%s|\n",
          pn_error_text ( pn_message_error ( msg ) )
        );
    exit ( 2 );
  }
  else
  {
    pn_string_t *s = pn_string ( NULL );
    pn_inspect ( pn_message_body(msg), s );
    log ( context, "%s\n", pn_string_get(s));
    context->total_bytes_received += strlen ( pn_string_get(s) );
    pn_free ( s );
  }
}





void 
send_message ( context_p context, pn_link_t * link ) 
{
  /*
   *   Set messages ID from sent count.
   */
  pn_atom_t id_atom;
  char id_string [ 20 ];
  sprintf ( id_string, "%d", context->messages_sent );
  id_atom.type = PN_STRING;
  id_atom.u.as_bytes = pn_bytes ( strlen(id_string), id_string );
  pn_message_set_id ( context->message, id_atom );

  /*
   * Make a random-length messages body, filled with random bytes.
   */
  make_random_message ( context );
  pn_data_t * body = pn_message_body ( context->message );
  pn_data_clear ( body );
  pn_data_enter ( body );
  pn_bytes_t bytes = { context->message_length, context->outgoing_buffer };
  pn_data_put_string ( body, bytes );
  pn_data_exit ( body );
  size_t outgoing_size = encode_outgoing_message ( context );

  pn_delivery ( link, 
                pn_dtag ( (const char *) & context->messages_sent, sizeof(context->messages_sent) ) 
              );
  pn_link_send ( link, 
                 context->outgoing_buffer, 
                 outgoing_size 
               );
  context->messages_sent ++;
  pn_link_advance ( link );
  context->total_bytes_sent += outgoing_size;
  if ( ! (context->messages_sent % 10) )
  {
    log ( context, "sent %d messages %d bytes\n", context->messages_sent, context->total_bytes_sent );
  }
}





bool 
process_event ( context_p context, pn_event_t * event ) 
{
  pn_session_t   * event_session;
  pn_transport_t * event_transport;
  pn_link_t      * event_link;
  pn_delivery_t  * event_delivery;

  char link_name [ 1000 ];


  switch ( pn_event_type( event ) ) 
  {
    case PN_LISTENER_ACCEPT:
      context->connection = pn_connection ( );
      pn_listener_accept ( pn_event_listener ( event ), context->connection );
    break;


    case PN_CONNECTION_INIT:
      pn_connection_set_container ( pn_event_connection( event ), context->id );

      event_session = pn_session ( pn_event_connection( event ) );
      pn_session_open ( event_session );
      if ( context->sending )
      {
        sprintf ( link_name, "%d_send_%05d", getpid(), context->link_count );
        context->link_count ++;
        context->links[0] = pn_sender (  event_session, link_name );
        pn_terminus_set_address ( pn_link_target(context->links[0]), context->path );
        pn_link_set_snd_settle_mode ( context->links[0], PN_SND_UNSETTLED );
        pn_link_set_rcv_settle_mode ( context->links[0], PN_RCV_FIRST );
      }
      else
      {
        sprintf ( link_name, "%d_recv_%05d", getpid(), context->link_count );
        context->link_count ++;
        context->links[0] = pn_receiver( event_session, link_name );
        pn_terminus_set_address ( pn_link_source(context->links[0]), context->path );
      }

      pn_link_open ( context->links[0] );
    break;


    case PN_CONNECTION_BOUND: 
      event_transport = pn_event_transport ( event );
      pn_transport_require_auth ( event_transport, false );
      pn_sasl_allowed_mechs ( pn_sasl(event_transport), "ANONYMOUS" );
    break;


    case PN_CONNECTION_REMOTE_OPEN : 
      pn_connection_open ( pn_event_connection( event ) ); 
    break;


    case PN_SESSION_REMOTE_OPEN:
      pn_session_open ( pn_event_session( event ) );
    break;


    case PN_LINK_REMOTE_OPEN: 
      event_link = pn_event_link( event );
      pn_link_open ( event_link );
      if ( pn_link_is_receiver ( event_link ) )
        pn_link_flow ( event_link, context->credit_window );
    break;


    case PN_LINK_FLOW : 
      event_link = pn_event_link ( event );
      if ( pn_link_is_sender ( event_link ) )
      {
        while ( pn_link_credit ( event_link ) > 0 && context->messages_sent < context->messages )
          send_message ( context, event_link );
      }
    break;


    case PN_DELIVERY: 
      event_delivery = pn_event_delivery( event );
      event_link = pn_delivery_link ( event_delivery );
      if ( pn_link_is_sender ( event_link ) ) 
      {
        pn_delivery_settle ( event_delivery );
        ++ context->accepted;

        if (context->accepted >= context->messages) 
        {
          if ( context->connection )
            pn_connection_close(context->connection);
          if ( context->listener )
            pn_listener_close(context->listener);
          break;
        }
      }
      else 
      if ( pn_link_is_receiver ( event_link ) )
      {

        if ( ! pn_delivery_readable  ( event_delivery ) )
          break;

        if ( pn_delivery_partial ( event_delivery ) ) 
          break;

        decode_message ( context, event_delivery );
        pn_delivery_update ( event_delivery, PN_ACCEPTED );
        pn_delivery_settle ( event_delivery );
        context->received ++;

        if ( context->received >= context->next_received_report )
        {
          log ( context,
                "info received %d messages %ld bytes\n",
                context->received,
                context->total_bytes_received
              );
          context->next_received_report += 100;
        }


        if ( context->received >= context->messages) 
        {
          fprintf ( stderr, "receiver: got %d messages. Stopping.\n", context->received );
          if ( context->connection )
            pn_connection_close(context->connection);
          if ( context->listener )
            pn_listener_close(context->listener);
          break;
        }
        pn_link_flow ( event_link, context->credit_window - pn_link_credit(event_link) );
      }
      else
      {
        fprintf ( stderr, 
                  "A delivery came to a link that is not a sender or receiver.\n" 
                );
        exit ( 1 );
      }
    break;


    case PN_CONNECTION_REMOTE_CLOSE :
      log ( context, "PN_CONNECTION_REMOTE_CLOSE\n" );
      pn_connection_close ( pn_event_connection( event ) );
    break;

    case PN_SESSION_REMOTE_CLOSE :
      log ( context, "PN_SESSION_REMOTE_CLOSE\n" );
      pn_session_close ( pn_event_session( event ) );
    break;


    case PN_LINK_REMOTE_CLOSE :
      log ( context, "PN_LINK_REMOTE_CLOSE\n" );
      pn_link_close ( pn_event_link( event ) );
    break;


    case PN_PROACTOR_INACTIVE:
      log ( context, "PN_PROACTOR_INACTIVE\n" );
      return false;

    default:
      break;
  }

  return true;
}





void
init_context ( context_p context, int argc, char ** argv )
{
  #define NEXT_ARG      argv[i+1]

  strcpy ( context->name, "default_name" );
  strcpy ( context->host, "0.0.0.0" );
  strcpy ( context->path, "speedy/my_path" );

  context->listener             = 0;
  context->connection           = 0;
  context->proactor             = 0;

  context->sending              = 0;
  context->link_count           = 0;
  context->messages_sent        = 0;
  context->received             = 0;
  context->accepted             = 0;
  context->log_file_name        = 0;
  context->log_file             = 0;
  context->message              = 0;
  context->total_bytes_sent     = 0;
  context->total_bytes_received = 0;

  context->messages             = 1000;
  context->next_received_report = 100;
  context->credit_window        = 100;
  context->max_send_length      = 100;



  for ( int i = 1; i < argc; ++ i )
  {
    // operation ----------------------------------------------
    if ( ! strcmp ( "--operation", argv[i] ) )
    {
      if ( ! strcmp ( "send", argv[i+1] ) )
      {
        context->sending = 1;
      }
      else
      if ( ! strcmp ( "receive", NEXT_ARG ) )
      {
        context->sending = 0;
      }
      else
      {
        fprintf ( stderr, "value for --operation should be 'send' or 'receive'.\n" );
        exit ( 1 );
      }
      
      i ++;
    }
    // name ----------------------------------------------
    else
    if ( ! strcmp ( "--name", argv[i] ) )
    {
      if ( ! strcmp ( NEXT_ARG, "PID" ) )
      {
        sprintf ( context->name, "client_%d", getpid() );
      }
      else
      {
        memset  ( context->name, 0, MAX_NAME );
        strncpy ( context->name, NEXT_ARG, MAX_NAME );
      }

      i ++;
    }
    // id ----------------------------------------------
    else
    if ( ! strcmp ( "--id", argv[i] ) )
    {
      if ( ! strcmp ( NEXT_ARG, "PID" ) )
      {
        sprintf ( context->id, "client_%d", getpid() );
      }
      else
      {
        memset  ( context->id, 0, MAX_NAME );
        strncpy ( context->id, NEXT_ARG, MAX_NAME );
      }

      i ++;
    }
    // max_message_length ----------------------------------------------
    else
    if ( ! strcmp ( "--max_message_length", argv[i] ) )
    {
      context->max_send_length = atoi ( NEXT_ARG );
      i ++;
    }
    // port ----------------------------------------------
    else
    if ( ! strcmp ( "--port", argv[i] ) )
    {
      context->port = strdup ( NEXT_ARG );
      i ++;
    }
    // log ----------------------------------------------
    else
    if ( ! strcmp ( "--log", argv[i] ) )
    {
      context->log_file_name = strdup ( NEXT_ARG );
      i ++;
    }
    // messages ----------------------------------------------
    else
    if ( ! strcmp ( "--messages", argv[i] ) )
    {
      context->messages = atoi ( NEXT_ARG );
      i ++;
    }
    // unknown ----------------------------------------------
    else
    {
      fprintf ( stderr, "Unknown option: |%s|\n", argv[i] );
      exit ( 1 );
    }
  }
}





int 
main ( int argc, char ** argv ) 
{
  context_t context;
  init_context ( & context, argc, argv );

  if ( context.log_file_name ) 
  {
    context.log_file = fopen ( context.log_file_name, "w" );
  }

  log ( & context, "max_message_length %d \n", context.max_send_length );

  if ( context.max_send_length <= 0 )
  {
    fprintf ( stderr, "no max message length.\n" );
    exit ( 1 );
  }

  // Make the max send length larger than the max receive length 
  // to account for the extra header bytes.
  context.max_receive_length  = context.max_send_length + 200;
  context.outgoing_buffer_size = context.max_send_length * 3;
  context.outgoing_buffer = (char *) malloc ( context.outgoing_buffer_size );

  context.message = pn_message();


  char addr[PN_MAX_ADDR];
  pn_proactor_addr ( addr, sizeof(addr), context.host, context.port );
  context.proactor   = pn_proactor();
  context.connection = pn_connection();
  pn_proactor_connect ( context.proactor, context.connection, addr );

  int batch_done = 0;
  while ( ! batch_done ) 
  {
    pn_event_batch_t *events = pn_proactor_wait ( context.proactor );
    pn_event_t * event;
    for ( event = pn_event_batch_next(events); event; event = pn_event_batch_next(events)) 
    {
      if (! process_event( & context, event ))
      {
        batch_done = 1;
        break;
       }
    }
    pn_proactor_done ( context.proactor, events );
  }

  if ( context.sending ) 
  {
    log ( & context, 
          "info sent %d messages %ld bytes\n", 
          context.messages_sent, 
          context.total_bytes_sent 
        );
  }
  else
  {
    log ( & context, 
          "info received %d messages %ld bytes\n", 
          context.received,
          context.total_bytes_received
        );
  }
  return 0;
}





