# Mercury
##interactive Golang Testing System for Qpid Dispatch Router



<br/>


###Requirements

1. the go language on your machine
2. installed proton and dispatch router software



<br/>

###Mercury Audience

I see Mercury as a tool for developers. The user has one or more installed versions of the Dispatch Router + Proton code and wantsto easily set up a complex network including Dispatch routers, edge routers, and clients with nontrivial addressing patterns. Currently I think that creation of nontrivial networks is difficult enough that it discourages extensive testing during development.

Especially I would like to see a more interactive form of development testing, in which a developer easily creates a network, and easily iterates through a cycle of altering the network, running it, and seeing results from the run. At the end of such a 'session' all of the developer's actions have been captured and will be reproducible later if desired. The captured session is a runnable script, and can be edited and used as a standardized test.


<br/>



###The C Client

The client that I use is written in C to the proactor interface and has been heavily adapted from an original by Alan Conway. It needs to be built before Mercury will be able to do anything useful. Look at the directory mercury/clients, look at the file in there called "m", adapt it for your system, and run it so you get an executable.

Having my own client allows me to do things like:

1. throttle send-speed with the "--throttle" argument
2. Tell the client to form multiple links with multiple "--address" arguments.
3. Tell it where to send its log files and so on.


<br/>


###Starting Mercury

The directory from which to run Mercury is also called mercury.  So it's mercury/mercury.  And the run-script that I use is 'r'.
In that script you will see that it sets an environment variable MERCURY\_ROOT to ${HOME}/mercury. If you install mercury someplace other than your home directory, change this variable as appropriate.

Here, as an example, is the startup script that I use:

    #! /usr/bin/bash

    export MERCURY_ROOT=${HOME}/mercury
    export GOPATH=${MERCURY_ROOT}

    # go run ./*.go  ./test_3

The last arg on the command line in the above startup script is the filename of the script for Mercury to run.



<br/>


###Getting Help

When Mercury is running, type 'help', and you will see a list of commands with brief descriptions. If you then type "help COMMAND\_NAME" you will get detailed help for that command, plus its arguments.



<br/>


###Running the Test Files


There is a growing collection of tests scripts in the directory  mercury/mercury/tests.  They are designed to illustrate different aspects of Mercury. You can run them by using the 'r' script in mercury/mercury and editing it to have the test script you want on the command line, or you can just start Mercury and type "inc tests/05\_addresses" or whatever.

The test scripts includes one other file with the 'inc' command -- a file called 'versions' which defines two different versions of the router code.

You will also need to change that 'versions' file to point to one or more versions that you have installed on your system, and then change the 'test' file to only use your versions.  (If you only define one version, then it will be the default and will get used whenever you create a new router if you just don't use the 'version' arg in the 'routers' command.




<br/>


###Debugging Startup

When Mercury starts up a router, it saves all the information you need to reproduce the same startup by hand. The router config file, the environment variables that are set, and the command line that is used are all saved in MERCURY\_ROOT/mercury/sessions/session\_TIMESTAMP/config. Router config files have the same names as their routers.

Here is an example:

    /home/mick/mercury/mercury/sessions/session\_2019\_03\_05\_2115/config/
        |-- A.conf
        |-- B.conf
        |-- command_line
        |-- environment_variables


If you have a router fail to start, or it starts up and is immediately defunct, use this information to reproduce the same startup by hand, and see what's happening.



<br/>


###Running a test 'By Hand'

One nice way to use Mercury is to use it to run a test for you and then see how it did that, so you can run the same setup 'by hand'.  You can look in the session/config directory and see all the environment variables it set and the command lines it used for the routers and the clients. 

The command lines for the routers will point to the config files that it created, and those config files have ports that were chosen because they were free at that moment. It is possible that they will *no longer* be free when you run the test 'by hand' if you have other stuff running on your system. But unlikely.



<br/>


###Versions

A 'version' represents a version of the dispatch router + proton code. The idea is that you have as many dispatch+proton installations as you like on your system, you define a Mercury version for each one of them, and then when you create a new router in Mercury you can tell it which version you want that router to use. Mercury will use the executable that corresponds to that version, and make it point to the correct libraries.

You can define a version in one of two different ways

1. You can provide the root directories for the proton and dispatch installation, and let Mercury calculate from them all the paths it needs, or

2. You can directly provide all the paths. This second option is meant for situations where your installation is different somehow from what Mercury expects.

To define a version with roots, use the 'version\_roots' command something like this:  (Here I am defining two different versions.)


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



<br/>


### Sessions

Each time Mercury starts up, it defines a new session. The name of the session is  "session\_YEAR\_MONTH\_DAY\_HOURMINUTE", for example: session\_2019\_03\_08\_0336".  A directory is made with that name as a subdirectory of mercury/mercury/sessions, and all information from that session is stored in there.

To replay a session, you just use the mercury log file name on the command line as the script for Mercury to run.
For example (see example of whole startup script, above) :

    go run ./*.go  ~/mercury/mercury/sessions/session_2019_03_08_0659/mercury_log

And it will replay your session.  The only thing is, that sessin-recording will have a 'quit' command at the end that you might want to delete first.



<br/>


###Client Status Reporting

When the network creates clients it gives each one of them their own individual log file in the directory SESSION/logs .  When the network starts running, a ticker is started that expires every 10 seconds. Every time it expires, a goroutine in the network code checks each clients status as written in the log files. 

Right now the only notification you get in Mercury is when the client 'completes' -- i.e. it has sent or received all the messages it was expecting to send or receive.



<br/>


###Router Status Reporting

TODO


<br/>


###Creating many clients with different addresses

You can quickly create a large number of clients, each with its own address, with a command like this:

send A count 100 address my\_address\_%d

That will add 100 sender clients to router A, and each one will have its own address. Mercury will notice the "%d" in the address, and it will replace that string with numbers that start at 1 and count up. So you will get  "my\_address\_1, my\_address\_2, ... my\_address\_100".

Then you can do the same thing with receivers:

recv C count 100 address my\_address\_%d

So now you will have 100 sender-receiver pairs, each connected by a unique address.


If you want the address-numbers to start from some number other than 1, you can control that also:

send B count 100 address my\_address\_%d  start\_at 101



<br/>


###Distributing Clients over Routers

TODO



<br/>


###One-Command Networks

There are some network topologies that we tend to use a lot, and Mercury gives you a way to create each of these with a single command. Here are some examples to try:

    linear 3
    mesh   4
    teds_diamond

<br/>

###Repeatable Randomness

TODO

<br/>
<br/>
