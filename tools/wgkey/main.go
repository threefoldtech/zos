package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
)

func main() {
	var (
		sk         string
		nodeID     string
		privateKey wgtypes.Key
		err        error
	)

	flag.StringVar(&sk, "sk", "", "private key")
	flag.StringVar(&nodeID, "n", "", "node id")
	flag.Parse()

	if sk == "" {
		privateKey, err = wgtypes.GeneratePrivateKey()
	} else {
		privateKey, err = wgtypes.ParseKey(sk)
	}
	if err != nil {
		log.Fatalln(err)
	}
	sk = privateKey.String()

	pk, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		log.Fatalln(err)
	}

	encrypted, err := crypto.Encrypt([]byte(sk), pk)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("private key ", privateKey.String())
	fmt.Println("private key encrypted", hex.EncodeToString(encrypted))
	fmt.Println("public key", privateKey.PublicKey())
}
