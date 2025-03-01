$Env:GOOS = "linux"
$Env:GOARCH = "arm64"
go build -ldflags="-s -w" -trimpath
ssh bamchoh@192.168.2.1 killall kuda_server
scp kuda_server bamchoh@192.168.2.1:./