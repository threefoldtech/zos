package yggdrasil

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zos/pkg/crypto"

	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	zinitService = "yggdrasil"
	confPath     = "/var/cache/modules/networkd/yggdrasil.json"
)

type config struct {
	Peers          []string `json:"Peers"`
	InterfacePeers struct {
	} `json:"InterfacePeers"`
	Listen                      []string `json:"Listen"`
	AdminListen                 string   `json:"AdminListen"`
	MulticastInterfaces         []string `json:"MulticastInterfaces"`
	AllowedEncryptionPublicKeys []string `json:"AllowedEncryptionPublicKeys"`
	EncryptionPublicKey         string   `json:"EncryptionPublicKey"`
	EncryptionPrivateKey        string   `json:"EncryptionPrivateKey"`
	SigningPublicKey            string   `json:"SigningPublicKey"`
	SigningPrivateKey           string   `json:"SigningPrivateKey"`
	LinkLocalTCPPort            int      `json:"LinkLocalTCPPort"`
	IfName                      string   `json:"IfName"`
	IfMTU                       int      `json:"IfMTU"`
	SessionFirewall             struct {
		Enable                        bool     `json:"Enable"`
		AllowFromDirect               bool     `json:"AllowFromDirect"`
		AllowFromRemote               bool     `json:"AllowFromRemote"`
		AlwaysAllowOutbound           bool     `json:"AlwaysAllowOutbound"`
		WhitelistEncryptionPublicKeys []string `json:"WhitelistEncryptionPublicKeys"`
		BlacklistEncryptionPublicKeys []string `json:"BlacklistEncryptionPublicKeys"`
	} `json:"SessionFirewall"`
	TunnelRouting struct {
		Enable bool `json:"Enable"`
		// 	IPv6RemoteSubnets interface{} `json:"IPv6RemoteSubnets"`
		// 	IPv6LocalSubnets  interface{} `json:"IPv6LocalSubnets"`
		// 	IPv4RemoteSubnets interface{} `json:"IPv4RemoteSubnets"`
		// 	IPv4LocalSubnets  interface{} `json:"IPv4LocalSubnets"`
	} `json:"TunnelRouting"`
	SwitchOptions struct {
		MaxTotalQueueSize int `json:"MaxTotalQueueSize"`
	} `json:"SwitchOptions"`
	NodeInfoPrivacy bool `json:"NodeInfoPrivacy"`
	NodeInfo        struct {
		Name string `json:"name"`
	} `json:"NodeInfo"`
}

// Server represent a yggdrasil server
type Server struct {
	signingKey ed25519.PrivateKey
	zinit      *zinit.Client
}

// NewServer create a new yggdrasil Server
// the privateKey is used to generate all the signing and encryption key of the yggdrasil node
func NewServer(zinit *zinit.Client, privateKey ed25519.PrivateKey) *Server {
	return &Server{
		signingKey: privateKey,
		zinit:      zinit,
	}
}

func (s *Server) generateConfig(w io.Writer) error {
	bin, err := exec.LookPath("yggdrasil")
	if err != nil {
		return err
	}

	cmd := exec.Command(bin, "-genconf", "-json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	config := config{}
	if err := json.NewDecoder(stdout).Decode(&config); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	config.SigningPrivateKey = hex.EncodeToString(s.signingKey)

	signingPublicKey := s.signingKey.Public().(ed25519.PublicKey)
	config.SigningPublicKey = hex.EncodeToString(signingPublicKey)

	encryptionPrivateKey := crypto.PrivateKeyToCurve25519(s.signingKey)
	config.EncryptionPrivateKey = hex.EncodeToString(encryptionPrivateKey[:])

	encryptionPublicKey := crypto.PublicKeyToCurve25519(signingPublicKey)
	config.EncryptionPublicKey = hex.EncodeToString(encryptionPublicKey[:])

	config.NodeInfo.Name = base58.Encode(signingPublicKey)
	config.MulticastInterfaces = []string{"zos"}

	config.IfName = "ygg0"
	config.TunnelRouting.Enable = false
	config.SessionFirewall.Enable = false

	config.Listen = []string{
		"tcp://0.0.0.0:4434",
		"tls://0.0.0.0:4444",
	}
	//TODO: any other config

	return json.NewEncoder(w).Encode(config)
}

// Start creates an yggdrasil zinit service and starts it
func (s *Server) Start() error {
	status, err := s.zinit.Status(zinitService)
	if err == nil && status.State.Is(zinit.ServiceStateRunning) {
		return nil
	}

	f, err := os.Create(confPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := s.generateConfig(f); err != nil {
		return err
	}

	bin, err := exec.LookPath("yggdrasil")
	if err != nil {
		return err
	}

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: fmt.Sprintf("%s -useconffile %s", bin, confPath),
		After: []string{
			"node-ready",
			"networkd",
		},
	})
	if err != nil {
		return err
	}

	if err := s.zinit.Monitor(zinitService); err != nil {
		return err
	}

	return s.zinit.Start(zinitService)
}

// Stop stop the yggdrasil zinit service
func (s *Server) Stop() error {
	status, err := s.zinit.Status(zinitService)
	if err != nil {
		return err
	}

	if !status.State.Is(zinit.ServiceStateRunning) {
		return nil
	}

	return s.zinit.Stop(zinitService)
}
