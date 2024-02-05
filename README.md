# contaiman
GUI designed for Podman the container manager.

### Build and Run
```shell
$ go build
$ ./contaiman
```
That's all you need.

### Work with Podman
```shell
$ systemctl start --user podman
```
For normal users, the socket path is ```unix://$XDG_RUNTIME_DIR/podman/podman.sock```.
