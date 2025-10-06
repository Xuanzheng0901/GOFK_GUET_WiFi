package wlan

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Handle struct {
	handle        uintptr
	interfaceList *wlanInterfaceInfoList
	SSID          string
}

var wlanapi = windows.NewLazySystemDLL("wlanapi.dll")
var (
	procWlanOpenHandle     = wlanapi.NewProc("WlanOpenHandle")
	procWlanCloseHandle    = wlanapi.NewProc("WlanCloseHandle")
	procWlanEnumInterfaces = wlanapi.NewProc("WlanEnumInterfaces")
	procWlanQueryInterface = wlanapi.NewProc("WlanQueryInterface")
	procWlanFreeMemory     = wlanapi.NewProc("WlanFreeMemory")
)

func NewWlanHandle() *Handle {
	return &Handle{}
}

func (wh *Handle) Init() bool {
	var negotiatedVersion uint32
	ret, _, _ := procWlanOpenHandle.Call(uintptr(wlanAPIVersion), 0, uintptr(unsafe.Pointer(&negotiatedVersion)), uintptr(unsafe.Pointer(&wh.handle)))
	if ret != 0 {
		return false
	}
	wh.getInterfaces()
	return true
}

func (wh *Handle) getInterfaces() {
	var interfaceList *wlanInterfaceInfoList
	ret, _, _ := procWlanEnumInterfaces.Call(wh.handle, 0, uintptr(unsafe.Pointer(&interfaceList)))
	if ret != 0 || interfaceList == nil {
		fmt.Printf("WlanEnumInterfaces failed: %d\n", ret)
		return
	}

	wh.interfaceList = interfaceList
	IFCount := interfaceList.NumberOfItems
	interfaces := (*[1 << 10]wlanInterface)(unsafe.Pointer(&interfaceList.InterfaceInfo[0]))[:IFCount:IFCount]

	for _, iface := range interfaces {
		if iface.IsState != 0 {
			wh.getDetail(&iface)
			return
		}
	}
}

func (wh *Handle) getDetail(iface *wlanInterface) {
	var dataSize uint32
	var dataPtr uintptr
	var opcode = wlanIFOpcodeCurrentConnection
	ret, _, _ := procWlanQueryInterface.Call(wh.handle, uintptr(unsafe.Pointer(&iface.InterfaceGuid)), uintptr(opcode), 0, uintptr(unsafe.Pointer(&dataSize)), uintptr(unsafe.Pointer(&dataPtr)), 0)
	if dataPtr != 0 {
		defer func(procWlanFreeMemory *windows.LazyProc, a ...uintptr) {
			_, _, _ = procWlanFreeMemory.Call(a...)
		}(procWlanFreeMemory, dataPtr)
	}

	if ret == 0 && dataPtr != 0 {
		connAttr := (*wlanConnectionAttributes)(unsafe.Pointer(dataPtr))
		wh.SSID = connAttr.WlanAssociationAttributes.Dot11Ssid.String()
	}
}

func (wh *Handle) CheckConnetionState() (SSID string, isConnected bool) {
	var dataSize uint32
	var connetionAttr *wlanConnectionAttributes
	ret, _, _ := procWlanQueryInterface.Call(
		wh.handle,
		uintptr(unsafe.Pointer(&wh.interfaceList.InterfaceInfo[0].InterfaceGuid)),
		7,
		0,
		uintptr(unsafe.Pointer(&dataSize)),
		uintptr(unsafe.Pointer(&connetionAttr)),
		0,
	)
	if ret != 0 || connetionAttr == nil {
		return "", false
	}
	if connetionAttr.IsState == 1 { // wlan_interface_state_connected
		SSID = string(connetionAttr.WlanAssociationAttributes.Dot11Ssid.UCSSID[:connetionAttr.WlanAssociationAttributes.Dot11Ssid.ElementLength])
		return SSID, true
	}
	return "", false
}

func (wh *Handle) Close() {
	if wh.interfaceList != nil {
		_, _, _ = procWlanFreeMemory.Call(uintptr(unsafe.Pointer(wh.interfaceList)))
		wh.interfaceList = nil
	}
	if wh.handle != 0 {
		_, _, _ = procWlanCloseHandle.Call(wh.handle)
		wh.handle = 0
	}
}

func ConnectBySSID(ssid string) {
	cmd := exec.Command("netsh", "WLAN", "connect", ssid)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("命令执行失败: %v", err)
	}
}

func Disconnect() {
	cmd := exec.Command("netsh", "WLAN", "disconnect")
	err := cmd.Run()
	if err != nil {
		log.Fatalf("命令执行失败: %v", err)
	}
}

func GetLocalMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	var fallback string

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || len(iface.HardwareAddr) == 0 {
			continue
		}

		if strings.Contains(strings.ToLower(iface.Name), "wi-fi") {
			return iface.HardwareAddr.String()
		}

		if fallback == "" {
			fallback = iface.HardwareAddr.String()
		}
	}

	if fallback != "" {
		return fallback
	}
	log.Fatalln("未找到有效的无线网卡 MAC 地址")
	return ""
}
