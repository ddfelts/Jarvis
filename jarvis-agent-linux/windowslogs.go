package main

import (
	"encoding/json"
	"fmt"
	"jarvis-agent/configs/config"

	"github.com/0xrawsec/golang-evtx/evtx"
	"github.com/0xrawsec/golang-win32/win32/wevtapi"
)

// XMLEventToGoEvtxMap converts an XML event to a GoEvtxMap

func XMLEventToGoEvtxMap(xe *wevtapi.XMLEvent) (*evtx.GoEvtxMap, error) {
	ge := make(evtx.GoEvtxMap)
	bytes, err := json.Marshal(xe.ToJSONEvent())
	if err != nil {
		return &ge, err
	}
	err = json.Unmarshal(bytes, &ge)
	if err != nil {
		return &ge, err
	}
	return &ge, nil
}

func windowslogs(config config.Config) {
	xmlEvents := eventProvider.FetchEvents(config.Windowslogs.Channels, wevtapi.EvtSubscribeToFutureEvents)
	for xe := range xmlEvents {
		e, err := XMLEventToGoEvtxMap(xe)
		if err != nil {
			logger.Log("WindowsLogs", "ERROR", fmt.Sprintf("Failed to convert event: %s", err))
			logger.Log("WindowsLogs", "DEBUG", fmt.Sprintf("Error data: %v", xe))
			continue
		}
		logger.Log("WindowsLogs", "INFO", string(evtx.ToJSON(e)))
	}
}
