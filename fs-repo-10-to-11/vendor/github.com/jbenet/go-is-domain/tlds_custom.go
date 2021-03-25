package isdomain

// ExtendedTLDs is a set of additional "TLDs", allowing decentralized name
// systems, like TOR and Namecoin.
var ExtendedTLDs = map[string]bool{
	"BIT":    true, // namecoin.org
	"ONION":  true, // torproject.org
	"ETH":    true, // ens.domains
	"CRYPTO": true, // unstoppabledomains.com
	"ZIL":    true, // unstoppabledomains.com
	"BBS":    true, // opennic.org
	"CHAN":   true, // opennic.org
	"CYB":    true, // opennic.org
	"DYN":    true, // opennic.org
	"EPIC":   true, // opennic.org
	"GEEK":   true, // opennic.org
	"GOPHER": true, // opennic.org
	"INDY":   true, // opennic.org
	"LIBRE":  true, // opennic.org
	"NEO":    true, // opennic.org
	"NULL":   true, // opennic.org
	"O":      true, // opennic.org
	"OSS":    true, // opennic.org
	"OZ":     true, // opennic.org
	"PARODY": true, // opennic.org
	"PIRATE": true, // opennic.org
}
