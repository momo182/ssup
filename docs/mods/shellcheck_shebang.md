# Shellcheck support

If you happen to have shellcheck installed on your machine  
ssup will be able to check your shell scripts for errors.  
Shellckeck will run before any connections are made to fail early,  
so any errors will halt the execution of your Supfile.  

> SHELLCHECK WILL NOT RUN IF SET UP SHEBANG FOR YOUR COMMAND  
> more on this below...  

# Shebang support

It is now possible to use shebang in the first line of your script.
Due to the nature of how ssup executes commands,  
it will create a script on the remote machine first,  
and launch that script second, to use the shebang naturally.

Paired with [`./dist` folder support](docs/mods/dist_folder.md),  
you can bring your own shell anywhere you want.
