// Copyright 2018 The Container File System Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package metanode

import (
	"io"
	"net"

	"github.com/tiglabs/containerfs/proto"
	"github.com/tiglabs/containerfs/util/log"
)

// StartTcpService binds and listens to the specified port.
func (m *MetaNode) startServer() (err error) {
	// initialize and start the server.
	m.httpStopC = make(chan uint8)
	ln, err := net.Listen("tcp", ":"+m.listen)
	if err != nil {
		return
	}
	go func(stopC chan uint8) {
		// TODO Unhandled errors
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			select {
			case <-stopC:
				return
			default:
			}
			if err != nil {
				continue
			}
			go m.serveConn(conn, stopC)
		}
	}(m.httpStopC)
	log.LogDebugf("start Server over...")
	return
}

func (m *MetaNode) stopServer() {
	if m.httpStopC != nil {
		defer func() {
			if r := recover(); r != nil {
				log.LogErrorf("action[StopTcpServer],err:%v", r)
			}
		}()
		close(m.httpStopC)
	}
}

// Read data from the specified tcp connection until the connection is closed by the remote or the tcp service is down.
func (m *MetaNode) serveConn(conn net.Conn, stopC chan uint8) {
	// TODO Unhandled errors
	defer conn.Close()
	c := conn.(*net.TCPConn)
	// TODO Unhandled errors
	c.SetKeepAlive(true)
	c.SetNoDelay(true)
	for {
		select {
		case <-stopC:
			return
		default:
		}
		p := &Packet{}
		if err := p.ReadFromConn(conn, proto.NoReadDeadlineTime); err != nil {
			if err != io.EOF {
				log.LogError("serve MetaNode: ", err.Error())
			}
			return
		}
		// Start a goroutine for packet handling. Do not block connection read goroutine.
		go func() {
			if err := m.handlePacket(conn, p); err != nil {
				log.LogError("serve operatorPkg: ", err.Error())
				return
			}
		}()
	}
}

func (m *MetaNode) handlePacket(conn net.Conn, p *Packet) (err error) {
	// Handle request
	err = m.metadataManager.HandleMetadataOperation(conn, p)
	return
}