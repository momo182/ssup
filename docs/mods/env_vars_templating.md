# env vars templating

env vars are templated in description when running ssup without args.  
this is to reflect key vars inside description at a glance.  

## how to use

templating is done by running envsubst package over desc text.
https://pkg.go.dev/gomodules.xyz/envsubst  
the format is `${SOME_VAR}` not `$SOME_VAR`  

## download dest templating

envsubst is also applied to download dest links  
to support downloads from multiple sources by adding `${SUP_HOST}` var  
to filename.

```yaml
  fetch_requirements:
    desc: fetch requirements.txt from remote host packed as tar
    fetch:
    - src: ~/foo.tar
      dst: foo.tar
```

could be written as

```yaml
  fetch_requirements:
    desc: fetch requirements.txt from remote host packed as tar
    fetch:
    - src: ~/foo.tar
      dst: foo_${SUP_HOST}.tar
```