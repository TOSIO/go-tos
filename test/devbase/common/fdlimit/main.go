package main

import (
	"fmt"

	"github.com/TOSIO/go-tos/devbase/common/fdlimit"
)

func main() {
	//fdlimit
	fmt.Println(fdlimit.Current())
	fmt.Println(fdlimit.Maximum())
	fmt.Println(fdlimit.Raise(16385))
}
