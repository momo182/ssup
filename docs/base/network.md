# Network

A group of hosts.

```yaml
# Supfile

networks:
    production:
        hosts:
            - api1.example.com
            - api2.example.com
            - api3.example.com
    staging:
        # fetch dynamic list of hosts
        inventory: curl http://example.com/latest/meta-data/hostname
```

`$ sup production COMMAND` will run COMMAND on `api1`, `api2` and `api3` hosts in parallel.  
__^^^ plain old way from original sup, at it's minimal form.__

## Network mods

Now two forms supported, short and long.

### Short form

First, short form:

```yaml
networks:
  remote:
    hosts:
      - remote_user@remote | P@ssw0rd << namespace_foo22
      # ^       ^          ^ ^        ^  ^
      # |       |          | |        |  |
      # user    |          | |        |  |
      #         host       | password |  |
      #                    |          |  namespace
      #                    |          namespace separator
      #                    password separator
```

as stated in extensions section, you can swap plain password with
shell command, like this:

```yaml
networks:
  remote1:
    hosts:
      - jim@example.com | $(echo "P@ssw0rd") << tube_foo22
```

### Long form

next here comes the long form:

```yaml
networks:
  remote2:
    env:
      foo22: bar414
    hosts:
      - host: ssh://jim@example.com
        # user: root
        pass: P@ssw0rd
        tube: ssup_was_here
        env:
          HOST_FOO: hello_FOOBAR-44
```
