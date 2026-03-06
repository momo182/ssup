# Local/MAKEFILE mode

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
