# Target

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

## Target mapping

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
    - user@contoso.com | $(cat secrets/example_password)

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
      rsync -avzh ~/git/yatgbotrs/target/x86_64-unknown-linux-gnu/release/yatgbotrs user@contoso.com:yatgbotrs

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
