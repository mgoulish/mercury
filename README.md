# mercury
##interactive golang testing system for qpid dispatch router.

###Requirements

  1. the go language on your machine
  2. installed proton and dispatch router software


###Running the Demos

  1. Go into the mercury/mercury directory and look at the ./r script.
     Modify it to point to your mercury root directory.
     ( Although if you have it installed in your home directory, the 
     current contents of mercury/mercury/r should work. )

  2. Look at the file mercury/mercury/paths .  Modify it to point to 
     your dispatch install directory, your proton install directory,
     and your mercury directory (again.)  (Hmm.)

  3. Go to mercury/clients and type ./m <enter>
     This should build the C Proactor-based client that we will be
     using. You should see a new executable file called c_proactor_client .

  4. Now go to the mercury/mercury directory. In there, ./r should start 
     the interactive mercury program. You should see a little mercury
     prompt. ( Similar to the astrological symbol for Venus, except 
     with a hat on it. )  Like this:  ☿ 

  5. At the ☿ prompt, type inc demo_1 <enter>.  This should start 
     running a script, one line at a time. This demo will make a 
     simple network and run it, and make a couple clients and run 
     them.  
