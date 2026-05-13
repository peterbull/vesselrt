package network

import (
	"fmt"
	"log"

	"github.com/vishvananda/netlink"
)

func CreateVethPairs() (netlink.Link, netlink.Link) {
	localName := "veth-local"
	peerName := "veth-peer"
	la := netlink.NewLinkAttrs()
	la.Name = localName
	veth := &netlink.Veth{
		LinkAttrs: la,
		PeerName:  peerName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		log.Fatalf("couldn't add veth pair: %v", err)
	}

	// bring up interfaces
	local, _ := netlink.LinkByName(localName)
	peer, _ := netlink.LinkByName(peerName)

	netlink.LinkSetUp(local)
	netlink.LinkSetUp(peer)
	fmt.Println("break")
	return local, peer
}
