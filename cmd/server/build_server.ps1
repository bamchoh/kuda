$Env:GOOS = "linux"
$Env:GOARCH = "arm64"
go build -ldflags="-s -w" -trimpath
scp kuda_server bamchoh@192.168.211.1:./