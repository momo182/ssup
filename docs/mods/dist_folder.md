# ./dist folder support

If you create a `dist` folder in your project root,  
any binaries created in that folder will be available for user on remote machine.  
Contents of the folder are copied on every launch,  
so if you updated a binary in that folder, it will be updated on next launch too (if size differs).

To actually do the copying of `./dist` folder to remote for a given command,  
set `dist: true`

## Example

Suppose cert_info.sh is inside ./dist folder:

```yaml
commands:
    cert_info:
        desc: list hashes of critical certs
        dist: true
        sudo: true
        run: |
            echo "// selfsigned root cert"
            cert_info.sh /some/path/selfsigned.root.crt
            echo ""
            echo "// server certs"
            cert_info.sh /some/path/server.crt
            cert_info.sh /other/path/server.crt
            echo ""
            echo "// client certs"
            cert_info.sh /some/path/client.crt
            cert_info.sh /other/path/client.key
```

> I tend to use `dist` folder for distributing nushell binaries and later  
> utilizing shebang support by adding shebang `#!/usr/bin/env nu`
