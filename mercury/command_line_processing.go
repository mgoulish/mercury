package main

import (
  "bufio"
  "fmt"
  "os"
  "strings"

  "lisp"
)





func process_line ( context * Context, line string ) {

  first_nonwhitespace := context.first_nonwhitespace_rgx.FindString ( line )
  if first_nonwhitespace == "" {
    // If the line is just empty, don't even echo it to the log.
    // The user just hit 'enter'.
    return
  }

  // Except for empty lines, echo everything else, 
  // including comments, to the log.
  fmt.Fprintf ( context.mercury_log_file, "%s\n", line )

  if first_nonwhitespace == "#" {
    // This line is a comment.
    return
  }

  // Clean up the line
  line = strings.Replace ( line, "\n", "", -1 )
  line = context.line_rgx.ReplaceAllString ( line, " " )
  fields := lisp.Listify ( line )
  _, list := lisp.Parse_from_string ( fields )

  call_command ( context, list )
}





func read_file ( context * Context, file_name string ) {

  file, err := os.Open ( file_name )
  if err != nil {
    panic ( err )
  }
  defer file.Close()

  scanner := bufio.NewScanner ( file )
  for scanner.Scan() {
    process_line ( context, scanner.Text() )
  }

  if err := scanner.Err(); err != nil {
    panic ( err )
  }
}





