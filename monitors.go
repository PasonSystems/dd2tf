//go:generate go-bindata -o tpl.go tmpl

package main

import (
	"github.com/zorkian/go-datadog-api"
)

type Monitor struct {
}

func (m Monitor) getElementById(client datadog.Client, id int) (interface{}, error) {
	mon, err := client.GetMonitor(id)
	return mon, err
}

func (m Monitor) getElementByTags(client datadog.Client, tags []string) ([]Item, error) {
	var ret []Item
	monitors, err := client.GetMonitorsByTags(tags)
	if err != nil {
		return nil, err
	}
	for _, elem := range monitors {
		ret = append(ret, Item{id: *elem.Id, d: Monitor{}})
	}
	return ret, nil
}

func (m Monitor) deleteElement(client datadog.Client, id int) error {
	return client.DeleteMonitor(id)
}

func (m Monitor) getAsset() string {
	return "tmpl/monitor.tmpl"
}

func (m Monitor) getName() string {
	return "monitors"
}

func (m Monitor) String() string {
	return m.getName()
}

func (m Monitor) getAllElements(client datadog.Client) ([]Item, error) {
	var ids []Item
	monitors, err := client.GetMonitors()
	if err != nil {
		return nil, err
	}
	for _, elem := range monitors {
		ids = append(ids, Item{id: *elem.Id, d: Monitor{}})
	}
	return ids, nil
}
