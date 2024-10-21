# systemd-age-creds

Load [age](https://github.com/FiloSottile/age) encrypted credentials in [systemd units](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html).

At the moment, [systemd-creds](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html) only support symmetric encryption requiring secrets to be encrypted on the machine with the TPM itself. Though, it's on the [systemd TODO](https://github.com/systemd/systemd/blob/e8fb0643c1bea626d5f5e880c3338f32705fd46d/TODO#L990-L1000) to add one day.

Solutions like [SOPS](https://github.com/getsops/sops) allow secrets to be encrypted elsewhere, checked into git and then only decrypted on the deployment host. It would be nice if a similar pattern could be applied to [systemd credentials](https://systemd.io/CREDENTIALS/).

`systemd-age-creds` provides a service credential server over `AF_UNIX` socket to provide [age](https://github.com/FiloSottile/age) encrypted credentials to [systemd units](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html) using `LoadCredential`.

## Usage

**systemd-age-creds.socket**

```ini
[Unit]
Description=age credential socket

[Socket]
ListenStream=%t/systemd-age-creds.sock
SocketMode=0600
Service=systemd-age-creds.service

[Install]
WantedBy=sockets.target
```

**systemd-age-creds.service**

```ini
[Unit]
Description=age credential server
Requires=systemd-age-creds.socket
# After=tpm

[Service]
Type=simple
ExecStart=/path/to/bin/systemd-age-creds -i /path/to/age-key.txt /path/to/secrets
```

**foo.service**

```ini
[Service]
ExecStart=/usr/bin/myservice.sh
# Instead of loading a symmetrically encrypted systemd cred from a file,
# LoadCredentialEncrypted=foobar:/etc/credstore/myfoobarcredential.txt
#
# You can reference the credential id loading from the systemd-age-creds socket.
LoadCredential=foobar:%t/systemd-age-creds.sock
```

### Nix

TK explain Nix usage

## See Also

[systemd Credentials](https://systemd.io/CREDENTIALS/), [systemd-creds](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html), [age](https://github.com/FiloSottile/age), [age-plugin-tpm](https://github.com/Foxboron/age-plugin-tpm)
