# Namespaces

Namespaces allow to pass envs from one command to another command.
To pass any env to the next command use `register` bash function with two or three params:

```sh
# First example
commands:
    example_one:
        run: |
            register foo22 bar33
            # ^      ^     ^  
            # |      |     |
            # |      |     value
            # |      key name
            # register function

# Second example
commands:
    example_two:
        run: |
            register ENV_WE_PASS_FURTHER super_secret main_tube
            # ^      ^                   ^            ^
            # |      |                   |            |
            # |      |                   |            namespace name
            # |      |                   value
            # |      key name
            # register function
```

as seen above, example #1 sets env into the host namespace and those envs
are automatically injected into all consecutive commands run on that exact host.

Second example sets env into the named namespace and you specify the third param to
`register` to set exact namespace name to push envs to.

Sadly, register is available **ONLY** in case you're not using [shebang functionality](docs/mods/shellcheck_shebang.md).  

See example Supfile below, note how network `remote` has namespace `main_tube` attached to it.
In the body of the first command, commands `register foo22 bar33` and `register foo33 bar22`
will register envs that will stay in host namespace, but `register ENV_WE_PASS_FURTHER super_secret main_tube`
uses namespace `main_tube` to pass env to the `test2` command run on `remote` and as you can see via export + grep,
it's there.

> register name was borrowed from ansible to be natural to anyone who used ansible before

example supfile to demonstrate namespaces usage:

```yaml
---
version: 0.5
desc: example to demonstrate namespaces usage
networks:
  l:
    hosts:
      - localhost
  all:
    hosts:
      - momo182@1.2.3.4 | some_password
  win:
    env:
      foo22: bar414
    hosts:
      - host: ssh://win_user@4.3.2.1
        # user: Administrator
        pass: user_pasword
        tube: ssup_was_here
        env:
          HOST_FOO: hello_FOOBAR-44
  remote:
    hosts:
      - remote_user@remote | $(cat ../secrets/remote_password.txt) << main_tube
commands:
  test:
    desc: demostrates usage of register func
    env:
      CMD_ENV_VAR: SUPER_VAR_FOOBAR
    local: |
      echo "==================================="
      echo "part1, where we define stuff
      "
      register foo22 bar33
      register foo33 bar22
      register FOO_BAR momowashere123 main_tube
      register ENV_WE_PASS_FURTHER super_secret main_tube
      
      echo "done with definitions"

  test2:
    desc: |
      notice how foo22 and foo33 are not passed over, the stayed in the host namespace
    env:
      NEW_VAR: sFOOBAR222222
    run: |
      echo "==================================="
      echo "part 2, where we dont find what we want
      "
      export | grep -i foo
      echo "passed var: \$FOO_BAR = $FOO_BAR"
      echo "new var: \$NEW_VAR = $NEW_VAR"
      # this one won't override the one from part1 as tube values will win over
      register FOO_BAR momowashere321 

  test3:
    run: |
      echo "==================================="
      echo "part 3, we tried to register \$FOO_BAR but alas
      "
      echo "passed var: \$FOO_BAR = $FOO_BAR"
      echo "passed var2: \$ENV_WE_PASS_FURTHER = $ENV_WE_PASS_FURTHER"
      register FOO_BAR momowashere321 main_tube

  test4:
    run: |
      echo "==================================="
      echo "part 4, where key = value with tube name wins
      "
      echo "passed var: \$FOO_BAR = $FOO_BAR"
      echo "passed var2: \$ENV_WE_PASS_FURTHER = $ENV_WE_PASS_FURTHER"
```

output will be:

```
~/sup_files> ssup remote test test2 test3 test4
local_user@localhost | ===================================
local_user@localhost | part1, where we define stuff
local_user@localhost |
local_user@localhost | done with definitions
remote_user@remote | ===================================
remote_user@remote | part 2, where we dont find what we want
remote_user@remote |
remote_user@remote | declare -x CMD_ENV_VAR="SUPER_VAR_FOOBAR"
remote_user@remote | declare -x FOO_BAR="momowashere123"
remote_user@remote | declare -x NEW_VAR="sFOOBAR222222"
remote_user@remote | passed var: $FOO_BAR = momowashere123
remote_user@remote | new var: $NEW_VAR = sFOOBAR222222
remote_user@remote | ===================================
remote_user@remote | part 3, we tried to register $FOO_BAR but alas
remote_user@remote |
remote_user@remote | passed var: $FOO_BAR = momowashere123
remote_user@remote | passed var2: $ENV_WE_PASS_FURTHER = super_secret
remote_user@remote | ===================================
remote_user@remote | part 4, where key = value with tube name wins
remote_user@remote |
remote_user@remote | passed var: $FOO_BAR = momowashere321
remote_user@remote | passed var2: $ENV_WE_PASS_FURTHER = super_secret
```

> now it might not be obvious from the first glance how come, first command runs on localhost and the next  
> is on remote, but just look closely at definitions:

```yaml
commands:
  test:
    desc: demostrates usage of register func
    env:
      CMD_ENV_VAR: SUPER_VAR_FOOBAR
    local: |
```

the `local` keyword forces to run this command on localhost, analogous to  
ansible's `delegate_to: 127.0.0.1`.
The `run` keyword will run commands on any `networks:` hosts you asked

```yaml
  test2:
    desc: |
      notice how foo22 and foo33 are not passed over, the stayed in the host namespace
    env:
      NEW_VAR: sFOOBAR222222
    run: |
```
