// Network watcher: which connected networks Windows calls Private (or Domain), and the laptop's
// IPv4 addresses on them. PhoneDrop listens only on those addresses, so on a Public network
// (café, hotel, airport) nothing is listening at all.
//
// Categories come from the Network List Manager (COM, called through raw vtables). Changes wake
// two blocked threads, NotifyAddrChange (addresses) and RegNotifyChangeKeyValue on the network
// profiles key (a user switching Public/Private), so watching costs no CPU while nothing changes.
package main

import (
	"net/netip"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	catPublic  = 0
	catPrivate = 1
	catDomain  = 2
)

type network struct {
	id       guid // INetwork::GetNetworkId: two networks can share a display name
	name     string
	category int
	trusted  bool // Public, but the user chose "Use here" on it
	addrs    []netip.Addr
}

// netState: the connected networks, and the addresses PhoneDrop may listen on.
type netState struct {
	networks []network
	listen   []netip.Addr
	err      error
}

type guid struct {
	d1     uint32
	d2, d3 uint16
	d4     [8]byte
}

func (g guid) String() string {
	b := make([]byte, 0, 38)
	hex := func(v uint64, digits int) {
		for i := digits - 1; i >= 0; i-- {
			b = append(b, "0123456789ABCDEF"[v>>(4*uint(i))&0xF])
		}
	}
	b = append(b, '{')
	hex(uint64(g.d1), 8)
	b = append(b, '-')
	hex(uint64(g.d2), 4)
	b = append(b, '-')
	hex(uint64(g.d3), 4)
	b = append(b, '-')
	hex(uint64(g.d4[0])<<8|uint64(g.d4[1]), 4)
	b = append(b, '-')
	for _, x := range g.d4[2:] {
		hex(uint64(x), 2)
	}
	return string(append(b, '}'))
}

var (
	clsidNetworkListManager = guid{0xDCB00C01, 0x570F, 0x4A9B, [8]byte{0x8D, 0x69, 0x19, 0x9F, 0xDB, 0xA5, 0x72, 0x3B}}
	iidINetworkListManager  = guid{0xDCB00000, 0x570F, 0x4A9B, [8]byte{0x8D, 0x69, 0x19, 0x9F, 0xDB, 0xA5, 0x72, 0x3B}}
)

// comCall calls method index of a COM object's vtable (0-2 IUnknown, 3-6 IDispatch). Pointer
// arguments must be converted inline at the call, as for win.
//
//go:uintptrescapes
func comCall(obj unsafe.Pointer, index int, args ...uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, index*int(unsafe.Sizeof(uintptr(0)))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{uintptr(obj)}, args...)...)
	return r
}

func comRelease(obj unsafe.Pointer) {
	if obj != nil {
		comCall(obj, 2)
	}
}

// Vtable slots (netlistmgr.h).
const (
	nlmGetNetworkConnections = 9  // INetworkListManager
	enumNext                 = 8  // IEnumNetworkConnections
	connGetNetwork           = 7  // INetworkConnection
	connGetAdapterID         = 12 // INetworkConnection
	netGetName               = 7  // INetwork
	netGetNetworkID          = 11 // INetwork
	netGetCategory           = 18 // INetwork
)

// readNetworks asks Windows for the connected networks and their categories, and matches each
// connection's adapter to its IPv4 addresses.
func readNetworks() netState {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if r := win(ole32, "CoInitializeEx", 0, 0 /*COINIT_MULTITHREADED*/); int32(r) < 0 {
		return netState{err: syscall.Errno(r)}
	}
	defer win(ole32, "CoUninitialize")

	var nlm unsafe.Pointer
	if r := win(ole32, "CoCreateInstance", uintptr(unsafe.Pointer(&clsidNetworkListManager)), 0, 0x17, /*CLSCTX_ALL*/
		uintptr(unsafe.Pointer(&iidINetworkListManager)), uintptr(unsafe.Pointer(&nlm))); int32(r) < 0 {
		return netState{err: syscall.Errno(r)}
	}
	defer comRelease(nlm)
	var enum unsafe.Pointer
	if r := comCall(nlm, nlmGetNetworkConnections, uintptr(unsafe.Pointer(&enum))); int32(r) < 0 {
		return netState{err: syscall.Errno(r)}
	}
	defer comRelease(enum)

	adapters := adapterAddrs()
	byID := map[guid]*network{}
	var order []guid
	for {
		var conn unsafe.Pointer
		var fetched uint32
		if r := comCall(enum, enumNext, 1, uintptr(unsafe.Pointer(&conn)), uintptr(unsafe.Pointer(&fetched))); r != 0 || fetched == 0 {
			break
		}
		var adapter guid
		comCall(conn, connGetAdapterID, uintptr(unsafe.Pointer(&adapter)))
		var nw unsafe.Pointer
		comCall(conn, connGetNetwork, uintptr(unsafe.Pointer(&nw)))
		comRelease(conn)
		if nw == nil {
			continue
		}
		var bstr *uint16
		var cat int32
		var id guid
		comCall(nw, netGetName, uintptr(unsafe.Pointer(&bstr)))
		comCall(nw, netGetNetworkID, uintptr(unsafe.Pointer(&id)))
		comCall(nw, netGetCategory, uintptr(unsafe.Pointer(&cat)))
		comRelease(nw)
		name := bstrString(bstr)
		n := byID[id]
		if n == nil {
			n = &network{id: id, name: name, category: int(cat)}
			byID[id] = n
			order = append(order, id)
		}
		n.addrs = append(n.addrs, adapters[strings.ToUpper(adapter.String())]...)
	}

	var st netState
	for _, id := range order {
		n := byID[id]
		n.trusted = n.category == catPublic && isTrusted(n.id)
		st.networks = append(st.networks, *n)
		if n.category == catPublic && !n.trusted {
			continue
		}
		for _, a := range n.addrs {
			// Only LAN addresses: a Tailscale/CGNAT or public address on a Private adapter must
			// not be bound (nor end up in the TLS certificate).
			if privatePeer(a) {
				st.listen = append(st.listen, a)
			}
		}
	}
	sort.Slice(st.listen, func(i, j int) bool { return st.listen[i].Less(st.listen[j]) })
	return st
}

func bstrString(b *uint16) string {
	if b == nil {
		return ""
	}
	n := win(oleaut32, "SysStringLen", uintptr(unsafe.Pointer(b)))
	s := syscall.UTF16ToString(unsafe.Slice(b, n))
	win(oleaut32, "SysFreeString", uintptr(unsafe.Pointer(b)))
	return s
}

// adapterAddrs: adapter GUID ("{...}", upper case) -> its IPv4 addresses.
func adapterAddrs() map[string][]netip.Addr {
	out := map[string][]netip.Addr{}
	size := uint32(16 * unsafe.Sizeof(syscall.IpAdapterInfo{}))
	for range 3 {
		buf := make([]syscall.IpAdapterInfo, size/uint32(unsafe.Sizeof(syscall.IpAdapterInfo{}))+1)
		err := syscall.GetAdaptersInfo(&buf[0], &size)
		if err == syscall.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return out
		}
		for ai := &buf[0]; ai != nil; ai = ai.Next {
			name := strings.ToUpper(cString(ai.AdapterName[:]))
			for ip := &ai.IpAddressList; ip != nil; ip = ip.Next {
				if a, err := netip.ParseAddr(cString(ip.IpAddress.String[:])); err == nil && !a.IsUnspecified() {
					out[name] = append(out[name], a)
				}
			}
		}
		return out
	}
	return out
}

func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// watchNetworks calls changed with a fresh netState now and after every network change.
// After a change it looks again a few times, because Windows takes a few seconds to identify a
// network it has just joined (it reports it as Public until then).
func watchNetworks(changed func(netState), recheck <-chan struct{}) {
	kick := make(chan struct{}, 1)
	notify := func() {
		select {
		case kick <- struct{}{}:
		default:
		}
	}
	go func() { // IPv4 address changes: joining, leaving or switching networks
		runtime.LockOSThread()
		for {
			if r, _, _ := iphlpapi.NewProc("NotifyAddrChange").Call(0, 0); r != 0 {
				return
			}
			notify()
		}
	}()
	go func() { // a profile's category changed (Settings > Network > Public/Private)
		runtime.LockOSThread()
		var key uintptr
		profiles := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\NetworkList\Profiles`
		if r, _, _ := advapi32.NewProc("RegOpenKeyExW").Call(hklm, uintptr(unsafe.Pointer(wstr(profiles))), 0,
			0x0010 /*KEY_NOTIFY*/, uintptr(unsafe.Pointer(&key))); r != 0 {
			return
		}
		for {
			if r, _, _ := advapi32.NewProc("RegNotifyChangeKeyValue").Call(key, 1,
				0x1|0x4 /*NAME | LAST_SET*/, 0, 0); r != 0 {
				return
			}
			notify()
		}
	}()
	go func() {
		for range recheck {
			notify()
		}
	}()

	last := readNetworks()
	changed(last)
	for range kick {
		for _, wait := range []time.Duration{time.Second, 4 * time.Second, 10 * time.Second} {
			time.Sleep(wait)
			if st := readNetworks(); !sameNet(st, last) {
				last = st
				changed(st)
			}
		}
	}
}

func sameNet(a, b netState) bool {
	if len(a.networks) != len(b.networks) || len(a.listen) != len(b.listen) || (a.err == nil) != (b.err == nil) {
		return false
	}
	for i := range a.networks {
		if a.networks[i].id != b.networks[i].id || a.networks[i].name != b.networks[i].name ||
			a.networks[i].category != b.networks[i].category || a.networks[i].trusted != b.networks[i].trusted {
			return false
		}
	}
	for i := range a.listen {
		if a.listen[i] != b.listen[i] {
			return false
		}
	}
	return true
}
