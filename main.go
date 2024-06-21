/*
Copyright © 2024 Brad Soper BRADLEY.SOPER@CNVRG.IO
*/
package main

import (
	"github.com/dilerous/cnvrgctl/cmd"
	_ "github.com/dilerous/cnvrgctl/cmd/backup"
)

func main() {
	cmd.Execute()
}
