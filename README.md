# mercury
##interactive golang testing system for qpid dispatch router.

###Requirements

  1. the go language on your machine
  2. installed proton and dispatch router software



###The C Client

  The client that I use is written in C to the proactor interface and has been heavily adapted from an original by Alan Conway. It needs to be built before Mercury will be able to do anything useful. Look at the directory mercury/clients, look at the file in there called "m", adapt it for your system, and run it so you get an executable.

  Having my own client allows me to do things like:
    
    1. throttle send-speed with the "--throttle" argument
    2. Tell the client to form multiple links with multiple "--address" arguments.
    3. Tell it where to send its log files and so on.



###Setting up Versions

  By 'version', I mean installed source trees for the dispatch router and proton. Mercury is a development tool, so its 'versions' capability is meant for multiple source code trees, all in nonstandard places -- for example $HOME/version_1/install/proton   $HOME/version_1/install/dispatch,  $HOME/version_2/install/proton, $HOME/version_2/install/dispatch, and so on.

  Of course you do not *need* multiple verions installed. You can have just one, but you will still need to define it as a version with a command like:

  version_roots name latest dispatch /home/mick/latest/install/dispatch proton /home/mick/latest/install/proton

  And if that's the only version you have, you can just forget about this feature. After you define that one, it will be used as the default for all routers.





###Starting Mercury

  The directory from which to run Mercury is also called mercury.  So it's mercury/mercury.  And the run-script that I use is 'r'.
  In that script you will see that it sets an environment variable MERCURY_ROOT to ${HPME}/mercury. If you install mercury someplace other than your home directory, change this variable as appropriate.

  Here, as an example, is the startup script that I use:

            #! /usr/bin/bash

            export MERCURY_ROOT=${HOME}/mercury
            export GOPATH=${MERCURY_ROOT}

            # go run ./*.go  ./test_3

  The last arg on the command line in the above startup script is the filename of the script for Mercury to run.




###Getting Help

  When Mercury is running, type 'help', and you will see a list of commands with brief descriptions. If you then type "help COMMAND_NAME" you will get detailed help for that command, plus its arguments.



###Running the Test Files


  There is a growing collection of tests scripts in the directory  mercury/mercury/tests.  They are designed to illustrate different aspects of Mercury. You can run them by using the 'r' script in mercury/mercury and editing it to have the test script you want on the command line, or you can just start Mercury and type "inc tests/05_addresses" or whatever.

  The test scripts includes one other file with the 'inc' command -- a file called 'versions' which defines two different versions of the router code.

  You will also need to change that 'versions' file to point to one or more versions that you have installed on your system, and then change the 'test' file to only use your versions.  (If you only define one version, then it will be the default and will get used whenever you create a new router if you just don't use the 'version' arg in the 'routers' command.




###Debugging Startup

  When Mercury starts up a router, it saves all the information you need to reproduce the same startup by hand. The router config file, the environment variables that are set, and the command line that is used are all saved in MERCURY_ROOT/mercury/sessions/session_TIMESTAMP/config. Router config files have the same names as their routers.

  Here is an example:

    /home/mick/mercury/mercury/sessions/session_2019_03_05_2115/config/
    ├── A.conf
    ├── B.conf
    ├── command_line
    └── environment_variables

  If you have a router fail to start, or it starts up and is immediately defunct, use this information to reproduce the same startup by hand, and see what's happening.





###Running a test 'By Hand'
  
  One nice way to use Mercury is to use it to run a test for you and then see how it did that, so you can run the same setup 'by hand'.  You can look in the session/config directory and see all the environment variables it set and the command lines it used for the routers and the clients. 
  
  The command lines for the routers will point to the config files that it created, and those config files have ports that were chosen because they were free at that moment. It is possible that they will *no longer* be free when you run the test 'by hand' if you have other stuff running on your system. But unlikely.





###Versions

  A 'version' represents a version of the dispatch router + proton code. The idea is that you have as many dispatch+proton installations as you like on your system, you define a Mercury version for each one of them, and then when you create a new router in Mercury you can tell it which version you want that router to use. Mercury will use the executable that corresponds to that version, and make it point to the correct libraries.

  You can define a version in one of two different ways
     
     1. You can provide the root directories for the proton and dispatch installation, and let Mercury calculate from them all the paths it needs, or

     2. You can directly provide all the paths. This second option is meant for situations where your installation is different somehow from what Mercury expects.

 To define a version with roots, use the 'version_roots' command something like this:  (Here I am defining two different versions.)


      version_roots name latest dispatch /home/mick/latest/install/dispatch proton /home/mick/latest/install/proton
      version_roots name later  dispatch /home/mick/later/install/dispatch  proton /home/mick/later/install/proton


  After defining those two versions, you can define a two-router network using those two different versions this way:

      routers 1
      routers 1 version later

  The first 'routers' command does not specify a version, so it will get the default version, which is the first one that you defined.  (In this example, 'latest'. )

  The second 'routers' command does specify a version 'later'.
  Both of these commands created 1 router apiece.

  Now connect them with the command 

      connect A B

  And you have a heterogeneous network!




### Sessions

  Each time Mercury starts up, it defines a new session. The name of the session is  "session_YEAR_MONTH_DAY_HOURMINUTE", for example: session_2019_03_08_0336".  A directory is made with that name as a subdirectory of mercury/mercury/sessions, and all information from that session is stored in there.

  To replay a session, you just use the mercury log file name on the command line as the script for Mercury to run.
  For example (see example of whole startup script, above) :

    go run ./*.go  ~/mercury/mercury/sessions/session_2019_03_08_0659/mercury_log

  And it will replay your session.  The only thing is, that sessin-recording will have a 'quit' command at the end that you might want to delete first.




###Client Status Reporting

  When the network creates clients it gives each one of them their own individual log file in the directory SESSION/logs .  When the network starts running, a ticker is started that expires every 10 seconds. Every time it expires, a goroutine in the network code checks each clients status as written in the log files. 

  Right now the only notification you get in Mercury is when the client 'completes' -- i.e. it has sent or received all the messages it was expecting to send or receive.



