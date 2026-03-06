
# Command

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

## Serial command (a.k.a. Rolling Update)

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

## Once command (one host only)

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

## Local command

Runs command always on localhost.

```yaml
# Supfile

commands:
    prepare:
        desc: Prepare to upload
        local: npm run build
```

## Upload command

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

## Interactive Bash on all hosts

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

## Interactive Docker Exec on all hosts

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
