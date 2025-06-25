Note
========

This a rewrite of https://github.com/pressly/sup in a more "clean acrh" fashion so i could understand and tinker.    
Made to experiment a bit, so it will diverge for sure.

exact commit that was forked:  
- https://github.com/pressly/sup/commit/17c751e8ca547e2ef7fb5b6b2017543cd7172a05 
to be more specific.

this repo is mostly that code, rearranged based on the ideas from:
https://www.youtube.com/watch?v=C7MRkqP5NRI  
hense the name change, to keep projects visually different

Super Stack Up
========

Super Stack Up is a simple deployment tool that performs given set of commands on multiple hosts in parallel. It reads Supfile, a YAML configuration file, which defines networks (groups of hosts), commands and targets.

Extensions to original sup
=========

- added support for `sudo: true` for task
- added support for SSH password auth
- added support for short and long form network definitions
- env vars can use subshell syntax to grab a value
- password fields can use subshell syntax to grab a value and use plain text value
- added automatic shellcheck support if you have it in PATH
- added `#source://` directive for `run:` and `local:`
- ssup now changes dir to Supfile location to accomodate use of relative links with `#source://`
- skip all networks definitions to use implicit localhost mode (makefile mode)
- namespaces to collect vars from one stage and pass those to next stages

# Demo

[![Sup](https://github.com/pressly/sup/blob/gif/asciinema.gif?raw=true)](https://asciinema.org/a/19742?autoplay=1)

*Note: Demo is based on [this example Supfile](./example/Supfile).*

# Installation

    $ go get -u github.com/momo182/ssup/cmd/ssup

# Usage

    $ sup [OPTIONS] NETWORK COMMAND [...]

### Options

| Option             | Description                      |
| ------------------ | -------------------------------- |
| `-f Supfile`       | Custom path to Supfile           |
| `-e`, `--env=[]`   | Set environment variables        |
| `--only REGEXP`    | Filter hosts matching regexp     |
| `--except REGEXP`  | Filter out hosts matching regexp |
| `--debug`, `-D`    | Enable debug/verbose mode        |
| `--no-color`, `-c` | Disable colors                   |
| `--disable-prefix` | Disable hostname prefix          |
| `--help`, `-h`     | Show help/usage                  |
| `--version`, `-v`  | Print version                    |

## Network

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

### Network mods

Now two forms supported, short and long.

#### Short form

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

#### Long form

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

#### Local/MAKEFILE mode

now you can skip defining `networks:` section at all. 
If you run ssup with args that contain only commands and/or targets, ssup  
will run those commands on localhost:

```yaml
# example Makefile i use to build some tool
---
version: 0.5

commands:
  build:
    desc: builds export app
    run: |
      go mod tidy
      go build -o logseq-export ./cmd/logseq-export/main.go
      rclone tree .

  run:
    desc: runs conversion process
    run: |
      DEBUG='*' ./logseq-export ~/Documents/mm_wiki_copy2/pages  ~/Documents/mm_wiki_copy2_converted

  clean:
    desc: cleans converted dir
    run: |
      rm -rfv ~/Documents/mm_wiki_copy2_converted/*
```
this is the actual run:

```text
~/git/logseq-export> ssup build
local_user@localhost | /
local_user@localhost | ├── Supfile
local_user@localhost | ├── cmd
local_user@localhost | │   └── logseq-export
local_user@localhost | │       └── main.go
local_user@localhost | ├── go.mod
local_user@localhost | ├── go.sum
local_user@localhost | ├── internal
local_user@localhost | │   ├── file
local_user@localhost | │   │   ├── copier.go
local_user@localhost | │   │   └── transformer.go
local_user@localhost | │   └── usecase
local_user@localhost | │       └── export.go
local_user@localhost | └── logseq-export # <<< this is the file that was built
local_user@localhost |
local_user@localhost | 5 directories, 8 files
```

## Command

A shell command(s) to be run remotely.

```yaml
# Supfile

commands:
    restart:
        desc: Restart example Docker container
        run: sudo docker restart example
    tail-logs:
        desc: Watch tail of Docker logs from all hosts
        run: sudo docker logs --tail=20 -f example
```

`$ sup staging restart` will restart all staging Docker containers in parallel.

`$ sup production tail-logs` will tail Docker logs from all production containers in parallel.

### Serial command (a.k.a. Rolling Update)

`serial: N` constraints a command to be run on `N` hosts at a time at maximum. Rolling Update for free!

```yaml
# Supfile

commands:
    restart:
        desc: Restart example Docker container
        run: sudo docker restart example
        serial: 2
```

`$ sup production restart` will restart all Docker containers, two at a time at maximum.

### Once command (one host only)

`once: true` constraints a command to be run only on one host. Useful for one-time tasks.

```yaml
# Supfile

commands:
    build:
        desc: Build Docker image and push to registry
        run: sudo docker build -t image:latest . && sudo docker push image:latest
        once: true # one host only
    pull:
        desc: Pull latest Docker image from registry
        run: sudo docker pull image:latest
```

`$ sup production build pull` will build Docker image on one production host only and spread it to all hosts.

### Local command

Runs command always on localhost.

```yaml
# Supfile

commands:
    prepare:
        desc: Prepare to upload
        local: npm run build
```

### Upload command

Uploads files/directories to all remote hosts. Uses `tar` under the hood.

```yaml
# Supfile

commands:
    upload:
        desc: Upload dist files to all hosts
        upload:
          - src: ./dist
            dst: /tmp/
```

### Interactive Bash on all hosts

Do you want to interact with multiple hosts at once? Sure!

```yaml
# Supfile

commands:
    bash:
        desc: Interactive Bash on all hosts
        stdin: true
        run: bash
```

```bash
$ sup production bash
#
# type in commands and see output from all hosts!
# ^C
```

Passing prepared commands to all hosts:
```bash
$ echo 'sudo apt-get update -y' | sup production bash

# or:
$ sup production bash <<< 'sudo apt-get update -y'

# or:
$ cat <<EOF | sup production bash
sudo apt-get update -y
date
uname -a
EOF
```

### Interactive Docker Exec on all hosts

```yaml
# Supfile

commands:
    exec:
        desc: Exec into Docker container on all hosts
        stdin: true
        run: sudo docker exec -i $CONTAINER bash
```

```bash
$ sup production exec
ps aux
strace -p 1 # trace system calls and signals on all your production hosts
```

## Target

Target is an alias for multiple commands. Each command will be run on all hosts in parallel,
`sup` will check return status from all hosts, and run subsequent commands on success only
(thus any error on any host will interrupt the process).

```yaml
# Supfile

targets:
    deploy:
        - build
        - pull
        - migrate-db-up
        - stop-rm-run
        - health
        - slack-notify
        - airbrake-notify
```

`$ sup production deploy`

is equivalent to

`$ sup production build pull migrate-db-up stop-rm-run health slack-notify airbrake-notify`

### Target mapping

now it's possible to map target names to networks.
This way you can map the whole journey of your code to a single target.
To illustrate this, here is an example of a Supfile that builds and deploys
a telegram bot:

```yaml
---
version: 0.5
env:
  TG_BOT_TOKEN: $(cat secrets/token.txt)

networks:
  local:
    hosts:
    - 127.0.0.1
  remote:
    hosts:
    - user@example.com | $(cat secrets/example_password)

commands:
  run:
    desc: run the bot locally
    local: |
      cargo run

  build-release:
    desc: release build for x86_64-unknown-linux-gnu
    local: |
      cargo zigbuild --target x86_64-unknown-linux-gnu --release
      file ~/git/yatgbotrs/target/x86_64-unknown-linux-gnu/release/yatgbotrs

  upload:
    local: |
      rsync -avzh ~/git/yatgbotrs/target/x86_64-unknown-linux-gnu/release/yatgbotrs user@example.com:yatgbotrs

  setup:
    desc: move yatgbotrs to right place on remote
    run: |
      rm -rfv ~/yatgbot/yatgbot
      mv -v ~/yatgbotrs ~/yatgbot/yatgbot
      sudo systemctl restart yatgbot.service && echo "yatgbot restarted" || echo "yatgbot not restarted"

targets:
  do_remote:
  - build-release local
  - upload local
  - setup remote
```
^^^^ here comes the fun part, notice that **build-release** and **upload**  
are run locally and **setup** is run remotely.  


# Supfile

See [example Supfile](./example/Supfile).

### Basic structure

```yaml
# Supfile
---
version: 0.4
desc: supfile description goes here...
# Global environment variables
env:
  NAME: api
  IMAGE: example/api

networks:
  local:
    hosts:
      - localhost
  staging:
    hosts:
      - stg1.example.com
  production:
    hosts:
      - api1.example.com
      - api2.example.com

commands:
  echo:
    desc: Print some env vars
    run: echo $NAME $IMAGE $SUP_NETWORK
  date:
    desc: Print OS name and current date/time
    run: uname -a; date

targets:
  all:
    - echo
    - date
```

### Default environment variables available in Supfile

- `$SUP_HOST` - Current host.
- `$SUP_NETWORK` - Current network.
- `$SUP_USER` - User who invoked sup command.
- `$SUP_TIME` - Date/time of sup command invocation.
- `$SUP_ENV` - Environment variables provided on sup command invocation. You can pass `$SUP_ENV` to another `sup` or `docker` commands in your Supfile.


### Shellcheck support

if you happen to have shellcheck installed on your machine
ssup will be able to check your shell scripts for errors.
Shellckeck will run before any connections are done,
so any errors will halt the execution of your Supfile.

SHELLCHECK WILL NOT RUN IF SET UP SHEBANG FOR YOUR COMMAND  
more on this below...

### shebang support

it is now possible to use shebang in the first line of your script.
Due to the nature of how ssup executes commands, it will create a script on the remote machine
first, and launch that script second, to use the shebang.


### Namespaces

namespaces allow to pass envs from one command to another command.
To pass any env to the next command use `register` bash function with two or three params:

```shell
register foo22 bar33
# ^      ^     ^  
# |      |     |
# |      |     value
# |      key name
# register function

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

second example sets env into the named namespace and you specify the third param to
`register` to set exact namespace name to push envs to.
See example Supfile below, note how network `remote` has namespace `main_tube` attached to it.
In the body of the first command, commands `register foo22 bar33` and `register foo33 bar22`
will register envs that will stay in host namespace, but `register ENV_WE_PASS_FURTHER super_secret main_tube`
uses namespace `main_tube` to pass env to the `test2` command run on `remote` and as you can see via export + grep,
it's there.

> register name was borrowed from ansible to be natural to anyone who used ansible before

now it might not be obvious from the first glance how come, fisrt command runs on localhost and the next  
is on remote, but just look closely at definitions:
```yaml
commands:
  test:
    desc: demostrates usage of register func
    env:
      CMD_ENV_VAR: SUPOER_VAR_FOOBAR
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
      CMD_ENV_VAR: SUPOER_VAR_FOOBAR
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
remote_user@remote | declare -x CMD_ENV_VAR="SUPOER_VAR_FOOBAR"
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

# Running sup from Supfile

Supfile doesn't let you import another Supfile. Instead, it lets you run `sup` sub-process from inside your Supfile. This is how you can structure larger projects:

```
./Supfile
./database/Supfile
./services/scheduler/Supfile
```

Top-level Supfile calls `sup` with Supfiles from sub-projects:
```yaml
 restart-scheduler:
    desc: Restart scheduler
    local: >
      sup -f ./services/scheduler/Supfile $SUP_ENV $SUP_NETWORK restart
 db-up:
    desc: Migrate database
    local: >
      sup -f ./database/Supfile $SUP_ENV $SUP_NETWORK up
```

# Common SSH Problem

if for some reason sup doesn't connect and you get the following error,

```bash
connecting to clients failed: connecting to remote host failed: Connect("myserver@xxx.xxx.xxx.xxx"): ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain
```

it means that your `ssh-agent` dosen't have access to your public and private keys. in order to fix this issue, follow the below instructions:

- run the following command and make sure you have a key register with `ssh-agent`

```bash
ssh-add -l
```

if you see something like `The agent has no identities.` it means that you need to manually add your key to `ssh-agent`.
in order to do that, run the following command

```bash
ssh-add ~/.ssh/id_rsa
```

you should now be able to use sup with your ssh key.


# Development

    fork it, hack it..

    $ make build

    create new Pull Request

We'll be happy to review & accept new Pull Requests!

# License

Licensed under the [MIT License](./LICENSE).
