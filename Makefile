mac: buildmac move
linux: buildlinux move

buildmac:
				GOARCH=amd64 GOOS=darwin go build -o ./bin/cnvrgctl
buildlinux:
				GOARCH=amd64 GOOS=linux go build -o ./bin/cnvrgctl
move:
				chmod +x ./bin/cnvrgctl && sudo cp ./bin/cnvrgctl /usr/local/bin
tag:
				git commit -am "testing gorelease"
				git tag -d v0.0.4
				git push origin --delete v0.0.4
				git tag -a v0.0.4 -m "gorelease support"
				git push origin v0.0.4
