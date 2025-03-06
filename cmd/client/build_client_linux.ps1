$Env:GOOS = "linux"
$Env:GOARCH = "arm64"
go build -ldflags="-s -w" -trimpath
scp kuda_client bamchoh@192.168.2.1:./