package drandr

import (
	"fmt"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
	"regexp"
)

type randrIdList []uint32

type OutputInfo struct {
	Name       string
	Id         uint32
	MmWidth    uint32
	MmHeight   uint32
	Crtc       CrtcInfo
	Connection bool
	Invalid    bool
	Timestamp  xproto.Timestamp

	Clones randrIdList
	Crtcs  randrIdList
	Modes  randrIdList
}
type OutputInfos []OutputInfo

var badOutputReg = regexp.MustCompile(`.+-\d-\d$`)

func (infos OutputInfos) Query(id uint32) OutputInfo {
	return infos.query("id", fmt.Sprintf("%v", id))
}

func (infos OutputInfos) QueryByName(name string) OutputInfo {
	return infos.query("name", name)
}

func (infos OutputInfos) ListNames() []string {
	var names []string
	for _, info := range infos {
		names = append(names, info.Name)
	}
	return names
}

func (infos OutputInfos) ListConnectionOutputs() OutputInfos {
	var ret OutputInfos
	for _, info := range infos {
		if !info.Connection {
			continue
		}
		ret = append(ret, info)
	}
	return ret
}

func (infos OutputInfos) ListValidOutputs() OutputInfos {
	var ret OutputInfos
	for _, info := range infos {
		if info.Invalid {
			continue
		}
		ret = append(ret, info)
	}
	return ret
}

func (infos OutputInfos) query(key, value string) OutputInfo {
	for _, info := range infos {
		if key == "id" {
			if fmt.Sprintf("%d", info.Id) == value {
				return info
			}
		} else if key == "name" {
			if info.Name == value {
				return info
			}
		}
	}
	return OutputInfo{}
}

func toOuputInfo(conn *xgb.Conn, output randr.Output) OutputInfo {
	reply, err := randr.GetOutputInfo(conn, output, lastConfigTimestamp).Reply()
	if err != nil {
		return OutputInfo{}
	}
	var info = OutputInfo{
		Name:       string(reply.Name),
		Id:         uint32(output),
		MmWidth:    reply.MmWidth,
		MmHeight:   reply.MmHeight,
		Connection: (reply.Connection == randr.ConnectionConnected),
		Timestamp:  reply.Timestamp,
		Clones:     outputsToRandrIdList(reply.Clones),
		Crtcs:      crtcsToRandrIdList(reply.Crtcs),
		Modes:      modesToRandrIdList(reply.Modes),
	}
	info.Invalid = isBadOutput(conn, info.Name, reply.Crtc)
	info.Crtc = toCrtcInfo(conn, reply.Crtc)

	if reply.Crtc == 0 {
		if len(info.Modes) != 0 {
			info.Invalid = false
		}
	} else {
		if info.Crtc.Width == 0 || info.Crtc.Height == 0 {
			info.Invalid = true
		}
	}
	return info
}

func outputsToRandrIdList(outputs []randr.Output) randrIdList {
	var list randrIdList
	for _, output := range outputs {
		list = append(list, uint32(output))
	}
	return list
}

func crtcsToRandrIdList(crtcs []randr.Crtc) randrIdList {
	var list randrIdList
	for _, crtc := range crtcs {
		list = append(list, uint32(crtc))
	}
	return list
}

func modesToRandrIdList(modes []randr.Mode) randrIdList {
	var list randrIdList
	for _, mode := range modes {
		list = append(list, uint32(mode))
	}
	return list
}

func isBadOutput(conn *xgb.Conn, output string, crtc randr.Crtc) bool {
	if !badOutputReg.MatchString(output) {
		return false
	}

	if crtc == 0 {
		return true
	}

	cinfo, err := randr.GetCrtcInfo(conn, crtc,
		lastConfigTimestamp).Reply()
	if err != nil {
		return true
	}

	hasOnlyOneRotation := (cinfo.Rotations == 1)
	if !hasOnlyOneRotation {
		return false
	}
	if cinfo.Mode != 0 {
		randr.SetCrtcConfig(conn, crtc, 0,
			lastConfigTimestamp, 0, 0, 0, 1, nil)
	}
	return true
}
