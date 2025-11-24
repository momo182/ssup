# env vars templating

env vars are templated in description when running ssup without args.  
this is to reflect key vars inside description at a glance.  

## how to use

templating is done by running envsubst package over desc text.
https://pkg.go.dev/gomodules.xyz/envsubst  
the format is `${SOME_VAR}` not `$SOME_VAR`  
