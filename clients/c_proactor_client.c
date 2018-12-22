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





#define MAX_BUF    10000000
#define MAX_NAME   1000
#define MAX_LINKS  1000


typedef 
struct context_s 
{
  pn_link_t    * links [ MAX_LINKS ];
  int            link_count;
  char           name [ MAX_NAME ];
  int            sending;
  char           id [ MAX_NAME ];
  char           host [ MAX_NAME ];
  size_t         max_message_length,
                 message_length;
  char         * outgoing_message_body;
  char         * port;
  char         * log_file_name;
  FILE         * log_file;
  int            sent;
  int            received;
  pn_message_t * message;
  uint64_t       total_bytes_sent,
                 total_bytes_received;
  int            messages;
  int            next_received_report;
}
context_t,
* context_p;






int operation = 0;
char const * path = "speedy/my_path";
int body_size = 100;
size_t credit_window = 100;

pn_proactor_t *proactor = 0;
pn_listener_t *listener = 0;
pn_connection_t *connection = 0;
pn_rwbytes_t buffer;        /* Encoded message buffer */

size_t accepted = 0;




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
  context->message_length = rand_int ( context->max_message_length );
  for ( int i = 0; i < context->message_length; ++ i )
    context->outgoing_message_body [ i ] = uint8_t ( rand_int ( 256 ) );
}





/* Encode message m into buffer buf, return the size.
 * The buffer is expanded using realloc() if needed.
 */
size_t 
encode_message ( pn_message_t * m, pn_rwbytes_t * buffer ) 
{
  int err = 0;
  size_t size = buffer->size;
  while ((err = pn_message_encode ( m, buffer->start, & size) ) != 0) 
  {
    if (err == PN_OVERFLOW) 
    {
      fprintf ( stderr, "error: overflowed MAX_BUF == %d\n", MAX_BUF );
      exit ( 1 );
    } 
    else 
    if (err != 0) 
    {
      fprintf ( stderr, 
                "error encoding message: %s |%s|\n", 
                pn_code(err), 
                pn_error_text(pn_message_error(m)) 
              );
      exit ( 1 );
    }
  }
  return size;
}





// CHANGE THIS
/* Decode message from delivery d into message m.
 * Use buf to hold the message data, expand with realloc() if needed.
 */
void 
decode_message ( context_p context, 
                 pn_delivery_t * delivery, 
                 pn_rwbytes_t  * buffer 
               ) 
{
  pn_message_t * msg  = context->message;
  pn_link_t    * link = pn_delivery_link    ( delivery );
  ssize_t        size = pn_delivery_pending ( delivery );

  pn_link_recv ( link, buffer->start, size);
  pn_message_clear ( msg );
  if ( pn_message_decode ( msg, buffer->start, size ) ) 
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
    // log ( context, "%s\n", pn_string_get(s));
    context->total_bytes_received += strlen ( pn_string_get(s) );
    pn_free ( s );
  }
}





void 
send_message ( context_p context, pn_link_t * link ) 
{
  pn_atom_t id_atom;
  int id_len = snprintf(NULL, 0, "%zu", context->sent);
  char id_str[id_len + 1];
  snprintf(id_str, id_len + 1, "%zu", context->sent);
  id_atom.type = PN_STRING;
  id_atom.u.as_bytes = pn_bytes(id_len + 1, id_str);
  pn_message_set_id ( context->message, id_atom );

  make_random_message ( context );
  // log ( context, "message: |%s|\n", context->outgoing_message_body );

  // pn_data_put_string ( pn_message_body(context->outgoing_message_body), pn_bytes(context->message_length, context->outgoing_message_body) );
  pn_data_t *body = pn_message_body ( context->message );
  pn_data_clear ( body );
  pn_data_enter ( body );
  pn_bytes_t bytes = { context->message_length, context->outgoing_message_body };
  pn_data_put_string ( body, bytes );
  pn_data_exit ( body );

  size_t size = encode_message ( context->message, & buffer);

  // CHANGE THIS
  /* Use id as unique delivery tag. */
  pn_delivery ( link, pn_dtag((const char *)&context->sent, sizeof(context->sent)) );
  pn_link_send ( link, buffer.start, size );
  pn_link_advance ( link );
  context->sent ++;
  context->total_bytes_sent += context->message_length;
  if ( ! (context->sent % 10) )
  {
    log ( context, "%d messages sent\n", context->sent );
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
      //log ( context, "PN_LISTENER_ACCEPT\n" );
      connection = pn_connection ( );
      pn_listener_accept ( pn_event_listener ( event ), connection );
    break;


    case PN_CONNECTION_INIT:
      //log ( context, "PN_CONNECTION_INIT:\n" );
      pn_connection_set_container ( pn_event_connection( event ), context->id );

      event_session = pn_session ( pn_event_connection( event ) );
      pn_session_open ( event_session );
      if ( context->sending )
      {
        sprintf ( link_name, "%d_send_%05d", getpid(), context->link_count );
        context->link_count ++;
        context->links[0] = pn_sender (  event_session, link_name );
        pn_terminus_set_address ( pn_link_target(context->links[0]), path );
        pn_link_set_snd_settle_mode ( context->links[0], PN_SND_UNSETTLED );
        pn_link_set_rcv_settle_mode ( context->links[0], PN_RCV_FIRST );
      }
      else
      {
        sprintf ( link_name, "%d_recv_%05d", getpid(), context->link_count );
        context->link_count ++;
        context->links[0] = pn_receiver( event_session, link_name );
        pn_terminus_set_address ( pn_link_source(context->links[0]), path );
      }

      pn_link_open ( context->links[0] );
    break;


    case PN_CONNECTION_BOUND: 
      //log ( context, "PN_CONNECTION_BOUND\n" );
      event_transport = pn_event_transport ( event );
      pn_transport_require_auth ( event_transport, false );
      pn_sasl_allowed_mechs ( pn_sasl(event_transport), "ANONYMOUS" );
    break;


    case PN_CONNECTION_REMOTE_OPEN : 
      //log ( context, "PN_CONNECTION_REMOTE_OPEN\n" );
      pn_connection_open ( pn_event_connection( event ) ); 
    break;


    case PN_SESSION_REMOTE_OPEN:
      //log ( context, "PN_SESSION_REMOTE_OPEN\n" );
      pn_session_open ( pn_event_session( event ) );
    break;


    case PN_LINK_REMOTE_OPEN: 
      //log ( context, "PN_LINK_REMOTE_OPEN:\n" );
      event_link = pn_event_link( event );
      pn_link_open ( event_link );
      if ( pn_link_is_receiver ( event_link ) )
        pn_link_flow ( event_link, credit_window );
    break;


    case PN_LINK_FLOW : 
      //log ( context, "PN_LINK_FLOW\n" );
      event_link = pn_event_link ( event );
      if ( pn_link_is_sender ( event_link ) )
      {
        while ( pn_link_credit ( event_link ) > 0 && context->sent < context->messages )
          send_message ( context, event_link );
      }
    break;


    case PN_DELIVERY: 
      //log ( context, "PN_DELIVERY\n" );
      event_delivery = pn_event_delivery( event );
      event_link = pn_delivery_link ( event_delivery );
      if ( pn_link_is_sender ( event_link ) ) 
      {
        pn_delivery_settle ( event_delivery );
        ++ accepted;

        if (accepted >= context->messages) 
        {
          if ( connection )
            pn_connection_close(connection);
          if ( listener )
            pn_listener_close(listener);
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

        decode_message ( context, event_delivery, & buffer );
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
          if ( connection )
            pn_connection_close(connection);
          if ( listener )
            pn_listener_close(listener);
          break;
        }
        pn_link_flow ( event_link, credit_window - pn_link_credit(event_link) );
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

  context->sending            = 0;
  context->link_count         = 0;
  context->max_message_length = 100;
  context->sent               = 0;
  context->received           = 0;

  strcpy ( context->name, "default_name" );
  strcpy ( context->host, "0.0.0.0" );

  context->log_file_name = 0;
  context->log_file      = 0;
  context->message       = 0;

  context->total_bytes_sent     = 0;
  context->total_bytes_received = 0;

  context->messages = 1000;
  context->next_received_report = 100;


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
      context->max_message_length = atoi ( NEXT_ARG );
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

  log ( & context, "info max_message_length %d \n", context.max_message_length );

  if ( context.max_message_length <= 0 )
  {
    fprintf ( stderr, "no max message length.\n" );
    exit ( 1 );
  }
  context.outgoing_message_body = (char *) malloc ( context.max_message_length );

  // CHANGE THIS
  if ( ! (buffer.start = (char *) malloc ( MAX_BUF )) )
  {
    fprintf ( stdout, "Can't get buffer memory.\n" );
    exit ( 1 );
  }
  buffer.size = MAX_BUF;

  context.message = pn_message();


  char addr[PN_MAX_ADDR];
  pn_proactor_addr ( addr, sizeof(addr), context.host, context.port );
  proactor   = pn_proactor();
  connection = pn_connection();
  pn_proactor_connect ( proactor, connection, addr );

  int batch_done = 0;
  while ( ! batch_done ) 
  {
    pn_event_batch_t *events = pn_proactor_wait ( proactor );
    pn_event_t * event;
    for ( event = pn_event_batch_next(events); event; event = pn_event_batch_next(events)) 
    {
      if (! process_event( & context, event ))
      {
        batch_done = 1;
        break;
       }
    }
    pn_proactor_done(proactor, events);
  }

  if ( context.sending ) 
  {
    log ( & context, 
          "info sent %d messages %ld bytes\n", 
          context.sent, 
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





