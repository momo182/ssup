package entity

// ExampleSupfile is an example supfile.
var ExampleSupfile = `
---
version: 0.5
desc: |
  example supfile example
  for testing purposes only
env:
  TOKEN: $(cat ./secrets/token.txt)

# you can safely remove the networks section, if you dont need remote hosts
# at all, making it effectively a makefile
networks:
  local:
    hosts:
    - 127.0.0.1
  remote:
    hosts:
    - sudo_user@example.pvt | $(cat ./secrets/login_creds.json | jq -r .password)

commands:
  run:
    desc: do a test run via cargo
    local: |
      cargo run

  build-release:
    desc: release build for x86_64-unknown-linux-gnu
    local: |
      cargo zigbuild --target x86_64-unknown-linux-gnu --release
      file ~/git/footgbot/target/x86_64-unknown-linux-gnu/release/footgbot

  upload:
    local: |
      rsync -avzh ~/git/footgbot/target/x86_64-unknown-linux-gnu/release/footgbot sudo_user@example.pvt:footgbot

  setup:
    desc: move footgbot to right place on remote
	sudo: true
    run: |
      rm -rfv ~/footgbot/footgbot
      mv -v ~/footgbot ~/footgbot/footgbot
      sudo systemctl restart footgbot.service && echo "footgbot restarted" || echo "footgbot not restarted"

targets:
  do_remote:
  - build-release local
  - upload local
  - setup remote
`
