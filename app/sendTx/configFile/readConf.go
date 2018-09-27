package configfile

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/naoina/toml"
)

//URLStructs url struct
// type URLStructs struct {
// 	// DataDir          string
// 	Http_port int
// 	// HTTPModules      []string
// 	Http_virtual_hosts string
// 	// HTTPTimeouts       *rpc.HTTPTimeouts
// 	W_s_port int
// 	// WSModules          []string
// 	// P2P                *p2p.Config
// }

// type URLStructs struct {
// 	HOST string
// 	PORT string
// }

//URLStructs url struct
type URLStructs struct {
	Urlstring string
}

//ReadConf read from config file
func ReadConf(fname string) (url *URLStructs, err error) {
	var (
		fp       *os.File
		fcontent []byte
	)

	url = new(URLStructs)

	if fp, err = os.Open(fname); err != nil {
		fmt.Println("open error", err)
		return
	}

	if fcontent, err = ioutil.ReadAll(fp); err != nil {
		fmt.Println("ReadAll error ", err)
		return
	}

	if err = toml.Unmarshal(fcontent, url); err != nil {
		fmt.Println("toml.Unmarshal error ", err)
		return
	}
	return
}
