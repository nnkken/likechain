package types

const (
	ModuleName   = "iscn"
	StoreKey     = ModuleName
	QuerierRoute = ModuleName
	RouterKey    = ModuleName
)

var (
	CidBlockKey    = []byte{0x01}
	IscnKernelKey  = []byte{0x02}
	IscnOwnerKey   = []byte{0x03}
	IscnCountKey   = []byte{0x04}
	CidToIscnIDKey = []byte{0x05}
)

func GetIscnKernelKey(iscnID []byte) []byte {
	return append(IscnKernelKey, iscnID...)
}

func GetIscnOwnerKey(iscnID []byte) []byte {
	return append(IscnOwnerKey, iscnID...)
}

func GetCidBlockKey(cid []byte) []byte {
	return append(CidBlockKey, cid...)
}

func GetCidToIscnIDKey(cid []byte) []byte {
	return append(CidToIscnIDKey, cid...)
}
