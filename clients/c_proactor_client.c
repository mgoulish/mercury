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

#include <memory.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <time.h>
#include <inttypes.h>
#include <sys/types.h>
#include <unistd.h>



#define MAX_BUF    10000000
#define MAX_NAME   1000
#define MAX_LINKS  1000


typedef 
struct context_s 
{
  pn_link_t * links [ MAX_LINKS ];
  int         link_count;
  char        name [ MAX_NAME ];
  int         sending;
  char        id [ MAX_NAME ];
  char        host [ MAX_NAME ];
  int         max_message_length,
              message_length;
  char      * outgoing_message_body;
  char      * port;
}
context_t,
* context_p;






int operation = 0;
char const * path = "speedy/my_path";
int messages  = 1000;
int body_size = 100;
size_t credit_window = 100;

pn_proactor_t *proactor = 0;
pn_listener_t *listener = 0;
pn_connection_t *connection = 0;
pn_message_t *message = 0;
pn_rwbytes_t buffer;        /* Encoded message buffer */

size_t sent = 0;
size_t received = 0;
size_t accepted = 0;




int
rand_int ( int one_past_max )
{
  double zero_to_one = (double) rand() / (double) RAND_MAX;
  return (int) (zero_to_one * (double) one_past_max);
}





int64_t
nanoseconds ( ) 
{
  struct timespec t;
  clock_gettime ( CLOCK_REALTIME, & t );
  return t.tv_sec * 1000 * 1000 * 1000 + t.tv_nsec;
}





void
make_random_message ( context_p context )
{
  context->message_length = rand_int ( context->max_message_length );
  for ( int i = 0; i < context->message_length; ++ i )
    context->outgoing_message_body [ i ] = uint8_t ( rand_int ( 256 ) );

  fprintf ( stderr, "random message!: " );
  for ( int i = 0; i < context->message_length ; i ++ )
  {
    fprintf ( stderr, "%d ", context->outgoing_message_body [ i ] );
  }
  fprintf ( stderr, "\n" );
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
decode_message ( pn_message_t  * msg, 
                 pn_delivery_t * delivery, 
                 pn_rwbytes_t  * buffer 
               ) 
{
  pn_link_t * link = pn_delivery_link    ( delivery );
  ssize_t     size = pn_delivery_pending ( delivery );

  pn_link_recv ( link, buffer->start, size);
  pn_message_clear ( msg );
  if ( pn_message_decode ( msg, buffer->start, size ) ) 
  {
    fprintf ( stderr, 
              "error from pn_message_decode: |%s|\n", 
              pn_error_text ( pn_message_error ( msg ) ) 
            );
    exit ( 2 );
  }
}





void 
send_message ( context_p context, pn_link_t * link ) 
{
  ++sent;
  int64_t stime = nanoseconds ( );
  pn_atom_t id_atom;
  int id_len = snprintf(NULL, 0, "%zu", sent);
  char id_str[id_len + 1];
  snprintf(id_str, id_len + 1, "%zu", sent);
  id_atom.type = PN_STRING;
  id_atom.u.as_bytes = pn_bytes(id_len + 1, id_str);
  pn_message_set_id(message, id_atom);

  // TEST
  make_random_message ( context );
  // END TEST

  pn_data_t *body = pn_message_body(message);
  pn_data_clear(body);
  pn_data_put_map(body);
  pn_data_enter(body);
  pn_data_put_string(body, pn_bytes ( 12, "Hello, World" ) );
  pn_data_put_long ( body, stime);
  pn_data_exit(body);

  size_t size = encode_message ( message, & buffer);

  // CHANGE THIS
  /* Use id as unique delivery tag. */
  pn_delivery ( link, pn_dtag((const char *)&sent, sizeof(sent)) );
  pn_link_send ( link, buffer.start, size );
  pn_link_advance ( link );
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
      connection = pn_connection ( );
      pn_listener_accept ( pn_event_listener ( event ), connection );
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
        pn_link_flow ( event_link, credit_window );
    break;


    case PN_LINK_FLOW : 
      event_link = pn_event_link ( event );
      fprintf ( stderr, "PN_LINK_FLOW on link |%s|\n", pn_link_name ( event_link ));
      if ( pn_link_is_sender ( event_link ) )
      {
        while ( pn_link_credit ( event_link ) > 0 && sent < messages )
          send_message ( context, event_link );
      }
    break;


    case PN_DELIVERY: 
      event_delivery = pn_event_delivery( event );
      event_link = pn_delivery_link ( event_delivery );
      if ( pn_link_is_sender ( event_link ) ) 
      {
        pn_delivery_settle ( event_delivery );
        ++ accepted;
        fprintf ( stdout, "sender: %d accepted.\n", accepted );

        if (accepted >= messages) 
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

        decode_message ( message, event_delivery, & buffer );
        pn_delivery_update ( event_delivery, PN_ACCEPTED );
        pn_delivery_settle ( event_delivery );
        ++ received;

        if (received >= messages) 
        {
          fprintf ( stderr, "receiver: got %d messages. Stopping.\n", received );
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
      pn_connection_close ( pn_event_connection( event ) );
    break;

    case PN_SESSION_REMOTE_CLOSE :
      pn_session_close ( pn_event_session( event ) );
    break;


    case PN_LINK_REMOTE_CLOSE :
      pn_link_close ( pn_event_link( event ) );
    break;


    case PN_PROACTOR_INACTIVE:
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

  strcpy ( context->name, "default_name" );
  strcpy ( context->host, "0.0.0.0" );


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
    // unknown ----------------------------------------------
    else
    {
      fprintf ( stderr, "Unknown option: |%s|\n", argv[i] );
      exit ( 1 );
    }
  }

  fprintf ( stdout, "context.sending            %d\n", context->sending );
  fprintf ( stdout, "context.name               %s\n", context->name );
  fprintf ( stdout, "context.id                 %s\n", context->id );
  fprintf ( stdout, "context.max_message_length %d\n", context->max_message_length );
}





int 
main ( int argc, char ** argv ) 
{
  context_t context;
  init_context ( & context, argc, argv );

  if ( context.max_message_length <= 0 )
  {
    fprintf ( stderr, "no max message length.\n" );
    exit ( 1 );
  }
  context.outgoing_message_body = (char *) malloc ( context.max_message_length );

  // CHANGE THIS
  if ( ! (buffer.start = (char *) malloc ( MAX_BUF )) )
  {
    fprintf ( stderr, "Can't get buffer memory.\n" );
    exit ( 1 );
  }
  buffer.size = MAX_BUF;

  message = pn_message();
  // CHANGE THIS
  char * body = (char *) malloc ( body_size );
  memset ( body, 'x', body_size );
  pn_data_put_string ( pn_message_body(message), pn_bytes(body_size, body) );

  char addr[PN_MAX_ADDR];
  pn_proactor_addr ( addr, sizeof(addr), context.host, context.port );
  proactor   = pn_proactor();
  connection = pn_connection();
  pn_proactor_connect ( proactor, connection, addr );

  int batch_done = 0;
  while ( ! batch_done ) 
  {
    fprintf ( stderr, "pn_proactor_wait...\n" );
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

  return 0;
}





