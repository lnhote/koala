package inbound

import (
	"net/http"
	"io/ioutil"
	"net"
	"time"
	"github.com/v2pro/koala/replaying"
	"github.com/v2pro/koala/envarg"
	"encoding/json"
	"github.com/v2pro/koala/internal"
	"github.com/v2pro/plz/countlog"
)

func Start() {
	go server()
}

func server() {
	internal.SetCurrentGoRoutineIsKoala()
	defer func() {
		recovered := recover()
		if recovered != nil {
			countlog.Fatal("event!inbound.panic", "err", recovered,
				"stacktrace", countlog.ProvideStacktrace)
		}
	}()
	http.HandleFunc("/json", handleInbound)
	countlog.Info("event!inbound.started",
		"inboundAddr", envarg.InboundAddr(),
		"sutAddr", envarg.SutAddr())
	err := http.ListenAndServe(envarg.InboundAddr().String(), nil)
	countlog.Info("event!inbound.exited", "err", err)
}

func handleInbound(respWriter http.ResponseWriter, req *http.Request) {
	internal.SetCurrentGoRoutineIsKoala()
	defer func() {
		recovered := recover()
		if recovered != nil {
			countlog.Fatal("event!inbound.panic", "err", recovered,
				"stacktrace", countlog.ProvideStacktrace)
		}
	}()
	countlog.Debug("event!inbound.received_request", "remoteAddr", req.RemoteAddr)
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		countlog.Error("event!inbound.failed to read request", "err", err)
		return
	}
	defer req.Body.Close()
	replayingSession := replaying.NewReplayingSession()
	err = json.Unmarshal(reqBody, replayingSession)
	if err != nil {
		countlog.Error("event!inbound.failed to unmarshal session", "err", err)
		return
	}
	localAddr, err := replaying.AssignLocalAddr()
	if err != nil {
		countlog.Error("event!inbound.failed to assign local addresses", "err", err)
		return
	}
	replaying.StoreTmp(*localAddr, replayingSession)
	conn, err := net.DialTCP("tcp4", localAddr, envarg.SutAddr())
	if err != nil {
		countlog.Error("event!inbound.failed to connect sut", "err", err)
		return
	}
	_, err = conn.Write(replayingSession.CallFromInbound.Request)
	if err != nil {
		countlog.Error("event!inbound.failed to write sut", "err", err)
		return
	}
	response, err := readResponse(conn)
	if err != nil {
		return
	}
	replayedSession := replayingSession.Finish(response)
	marshaledReplayedSession, err := json.Marshal(replayedSession)
	if err != nil {
		countlog.Error("event!inbound.marshal replaying session failed", "err", err)
		return
	}
	_, err = respWriter.Write(marshaledReplayedSession)
	if err != nil {
		countlog.Error("event!inbound.failed to write response", "err", err)
		return
	}
}

func readResponse(conn *net.TCPConn) ([]byte, error) {
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	bytesRead, err := conn.Read(buf)
	if err != nil {
		countlog.Error("event!inbound.failed to read first packet from sut", "err", err)
		return nil, err
	}
	response := []byte{}
	response = append(response, buf[:bytesRead]...)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		bytesRead, err = conn.Read(buf)
		if err != nil {
			break
		}
		response = append(response, buf[:bytesRead]...)
	}
	return response, nil
}
