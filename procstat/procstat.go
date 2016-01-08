// +build linux

/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2015 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package procstat

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	procPath   = "/proc"
	procStat   = "stat"
	procStatus = "status"
	procCmd    = "cmdline"
	procIO     = "io"
)

var (
	States = map[string]string{
		"R": "running",
		"S": "sleeping",
		"D": "waiting",
		"Z": "zombie",
		"T": "stopped",
		"t": "tracing",
		"X": "dead",
		"K": "wakekill",
		"W": "waking",
		"P": "parked",
	}
)

type proc struct {
	Pid     int
	State   string
	CmdLine string
	Stat    []string
	Io      map[string]uint64
	VmData  uint64
	VmCode  uint64
}

func GetStats() (map[string][]proc, error) {
	files, err := ioutil.ReadDir(procPath)
	if err != nil {
		return nil, err
	}
	procs := map[string][]proc{}
	for _, file := range files {
		// process only PID sub dirs
		if pid, err := strconv.Atoi(file.Name()); err == nil {
			// get proc/<pid>/stat data
			fstat := filepath.Join(procPath, file.Name(), procStat)
			procStat, err := ioutil.ReadFile(fstat)
			if err != nil {
				return nil, err
			}
			// get proc/<pid>/cmdline data
			fcmd := filepath.Join(procPath, file.Name(), procCmd)
			procCmdLine, err := ioutil.ReadFile(fcmd)
			if err != nil {
				return nil, err
			}
			// get proc/<pid>/io data
			procIo, err := read2Map(filepath.Join(procPath, file.Name(), procIO))
			if err != nil {
				return nil, err
			}
			// get proc/<pid>/status data
			var pStatus map[string]uint64
			var vmData, vmCode uint64
			// special case for zombie
			state := strings.Fields(string(procStat))[2]
			if state == "Z" {
				vmData = 0
				vmCode = 0
			} else {
				pStatus, err = read2Map(filepath.Join(procPath, file.Name(), procStatus))
				if err != nil {
					return nil, err
				}
				vmData = pStatus["VmData"] * 1024
				vmCode = (pStatus["VmExe"] + pStatus["VmLib"]) * 1024
			}
			// TODO - if ( report_ctx_switch ) ps_read_tasks_status(pid, ps)
			pc := proc{
				Pid:     pid,
				State:   strings.Fields(string(procStat))[2],
				Stat:    strings.Fields(string(procStat)),
				CmdLine: strings.Replace(string(procCmdLine), "\x00", " ", -1),
				Io:      procIo,
				VmData:  vmData,
				VmCode:  vmCode,
			}
			// tmpName begins and end with brackets, removing them
			tmpName := strings.Fields(string(procStat))[1]
			procName := tmpName[1 : len(tmpName)-1]
			instances, _ := procs[procName]
			procs[procName] = append(instances, pc)
		}
	}
	return procs, nil
}

func read2Map(fileName string) (map[string]uint64, error) {
	stats := map[string]uint64{}
	status, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(status), "\n") {
		if line == "" {
			continue
		}

		data := strings.Fields(line)
		if len(data) < 2 {
			continue
		}

		name := data[0]
		last := len(name) - 1
		if string(name[last]) == ":" {
			name = name[:last]
		}

		value, err := strconv.ParseUint(data[1], 10, 64)
		if err != nil {
			continue
		}

		stats[name] = value
	}
	return stats, nil
}
