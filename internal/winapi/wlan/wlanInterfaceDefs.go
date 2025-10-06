package wlan

import "golang.org/x/sys/windows"

const (
	wlanAPIVersion                uint32 = 2
	wlanIFOpcodeCurrentConnection uint32 = 7
)

type wlanInterface struct {
	InterfaceGuid        windows.GUID
	InterfaceDescription [256]uint16
	IsState              uint32
}

type wlanInterfaceInfoList struct {
	NumberOfItems uint32
	Index         uint32
	InterfaceInfo [1]wlanInterface
}

type DOT11_SSID struct {
	ElementLength uint32
	UCSSID        [32]byte // SSID 最长 32 字节
}

func (ssid DOT11_SSID) String() string {
	return string(ssid.UCSSID[:ssid.ElementLength])
}

type wlanAssociationAttributes struct {
	Dot11Ssid         DOT11_SSID
	Dot11BssType      uint32
	Dot11Bssid        [6]byte // MAC 地址
	Dot11PhyType      uint32
	UPhyIndex         uint32
	WlanSignalQuality uint32 // 信号质量 0-100
	UlRxRate          uint32
	UlTxRate          uint32
}

type wlanConnectionAttributes struct {
	IsState                   uint32
	WlanConnectionMode        uint32
	StrProfileName            [256]uint16
	WlanAssociationAttributes wlanAssociationAttributes
	WlanSecurityAttributes    [100]byte // 简化处理，实际结构更复杂，但这里不需要
}
