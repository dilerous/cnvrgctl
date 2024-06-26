/*
Copyright © 2024 Brad Soper BRADLEY.SOPER@CNVRG.IO
*/
package main

import (
	"github.com/dilerous/cnvrgctl/cmd"
	_ "github.com/dilerous/cnvrgctl/cmd/backup"
	_ "github.com/dilerous/cnvrgctl/cmd/install"
	_ "github.com/dilerous/cnvrgctl/cmd/logs"
	_ "github.com/dilerous/cnvrgctl/cmd/restore"
)

func main() {
	cmd.Execute()
}
