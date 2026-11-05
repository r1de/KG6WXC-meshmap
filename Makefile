.PHONY: build release clean
build:
	CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o meshmap .
release:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/meshmap-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/meshmap-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/meshmap-linux-armv7 .

	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/meshmap-macos-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/meshmap-macos-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/meshmap-windows-amd64.exe .
clean:
	rm -f -- meshmap
	rm -rf -- dist/
