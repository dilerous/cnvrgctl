mac: buildmac move
linux: buildlinux move

buildmac:
				GOARCH=amd64 GOOS=darwin go build -o ./bin/cnvrgctl
buildlinux:
				GOARCH=amd64 GOOS=linux go build -o ./bin/cnvrgctl
move:
				chmod +x ./bin/cnvrgctl && sudo cp ./bin/cnvrgctl /usr/local/bin
