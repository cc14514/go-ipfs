package config

import (
	"errors"
	"fmt"

	iaddr "github.com/cc14514/go-ipfs/thirdparty/ipfsaddr"
)

// DefaultBootstrapAddresses are the hardcoded bootstrap addresses
// for IPFS. they are nodes run by the IPFS team. docs on these later.
// As with all p2p networks, bootstrap is an important security concern.
//
// NOTE: This is here -- and not inside cmd/ipfs/init.go -- because of an
// import dependency issue. TODO: move this into a config/default/ package.
var DefaultBootstrapAddresses = []string{
	"/dnsaddr/bootstrap.libp2p.io/ipfs/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/ipfs/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/ipfs/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/ipfs/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	//YBJ
	"/ip4/101.251.230.212/tcp/4001/ipfs/QmSkDigBiYhdRQjNZo6UQzEwEDqTCgEG5zpo5rjLXmTvmX",
	"/ip4/101.251.230.213/tcp/4001/ipfs/QmXPLT2xKRLHH2i9TZPiTxesFR6n74RZugXsDWDtsyHREW",
	"/ip4/101.251.230.215/tcp/4001/ipfs/Qmape1ua1c8ioWfyaeWEFHgFzcd8MYdUuzqZ2mqefEsk5R",
	"/ip4/101.251.230.219/tcp/50002/ipfs/Qmcc6SdJaTavk4b76U3WDQDfgXzx8xLaS12VkQXFPuUbXt",
	//QYQ
	"/ip4/180.76.148.252/tcp/50002/ipfs/QmeEQGQHXDFndN1SrDB6wusL3CuYhdzeMXmhZ6Vi85H8pL",
	"/ip4/180.76.149.1/tcp/50002/ipfs/Qme5zmBKHck6pN9J6w4CCAyjmVhGUfZ8uHGYWtZ5VEGuN3",
}

// BootstrapPeer is a peer used to bootstrap the network.
type BootstrapPeer iaddr.IPFSAddr

// ErrInvalidPeerAddr signals an address is not a valid peer address.
var ErrInvalidPeerAddr = errors.New("invalid peer address")

func (c *Config) BootstrapPeers() ([]BootstrapPeer, error) {
	return ParseBootstrapPeers(c.Bootstrap)
}

// DefaultBootstrapPeers returns the (parsed) set of default bootstrap peers.
// if it fails, it returns a meaningful error for the user.
// This is here (and not inside cmd/ipfs/init) because of module dependency problems.
func DefaultBootstrapPeers() ([]BootstrapPeer, error) {
	ps, err := ParseBootstrapPeers(DefaultBootstrapAddresses)
	if err != nil {
		return nil, fmt.Errorf(`failed to parse hardcoded bootstrap peers: %s
This is a problem with the ipfs codebase. Please report it to the dev team.`, err)
	}
	return ps, nil
}

func (c *Config) SetBootstrapPeers(bps []BootstrapPeer) {
	c.Bootstrap = BootstrapPeerStrings(bps)
}

func ParseBootstrapPeer(addr string) (BootstrapPeer, error) {
	ia, err := iaddr.ParseString(addr)
	if err != nil {
		return nil, err
	}
	return BootstrapPeer(ia), err
}

func ParseBootstrapPeers(addrs []string) ([]BootstrapPeer, error) {
	peers := make([]BootstrapPeer, len(addrs))
	var err error
	for i, addr := range addrs {
		peers[i], err = ParseBootstrapPeer(addr)
		if err != nil {
			return nil, err
		}
	}
	return peers, nil
}

func BootstrapPeerStrings(bps []BootstrapPeer) []string {
	bpss := make([]string, len(bps))
	for i, p := range bps {
		bpss[i] = p.String()
	}
	return bpss
}
