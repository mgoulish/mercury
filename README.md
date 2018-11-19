# mercury
##golang testing system for qpid dispatch router.

###Requirements

  1. the go language on your machine
  2. installed proton and dispatch router software



###Mercury Directory Structure

        ├── NOTICE
        ├── README.md
        ├── results
        │   └── test_name
        │       └── 2018_11_15
        │           ├── config
        │           │   ├── ...all router config files ...
        │           ├── log
        │           │   ├── ...all router log files ...
        │           └── result
        ├── src
        │   ├── router
        │   │   └── router.go
        │   ├── router_network
        │   │   └── router_network.go
        │   └── utils
        │       └── utils.go
        └── tests
            ├── test_01
            │   ├── r
            │   └── test_01.go
            └── test_02
                ├── r
                └── test_02.go



####The tests directory

  Each subdirectory under directory *tests* contains the test program,
  and a small run-script. The run-script shows examples for setting the 
  required environment variables.



####The results directory

  The results directory is created when tests are run. 
  Individual test result directories are grouped under the test name,
  and then the individual result directory is named with the test ID.
  So if you have a test called test_01, and you run it 

  There is a description of each test at the top of its go file.



###To run the tests:

  1. The tests need the following environment variables defined:
        
        MERCURY_ROOT
        DISPATCH_INSTALL_ROOT
        PROTON_INSTALL_ROOT

     There are examples of how I set these variables in each of the
     run-scripts, i.e. test_XX/r .


  2. After setting the environment variables, go into the directory 
     of the test you want to run, and use the run-script by typing:
       
       ./r
 

###Results

  The results directory will be created by the first test that runs,
  and each test will create its own subdirectories.
  The results directory structure will look like this:

            results/
            ├── test_01
            │   └── 2018_11_16
            │       ├── config
            │       │   └── A.conf
            │       ├── log
            │       │   └── A.log
            │       └── result
            └── test_02
                └── 2018_11_16
                    ├── config
                    │   ├── A.conf
                    │   ├── B.conf
                    │   ├── C.conf
                    │   ├── e1.conf
                    │   ├── e2.conf
                    │   ├── e3.conf
                    │   ├── e4.conf
                    │   └── e5.conf
                    ├── log
                    │   ├── A.log
                    │   ├── B.log
                    │   ├── C.log
                    │   ├── e1.log
                    │   ├── e2.log
                    │   ├── e3.log
                    │   ├── e4.log
                    │   └── e5.log
                    └── result

  
  Each test will have its own subdirectory under *results*, and each instance
  of the test will have its own subdirectory under *test_name*. The result will
  written to the file *result*, and the environment variables and executable
  command line will be captured to their respective files.  All the configuration
  files of the routers are written to the config subdirectory.
