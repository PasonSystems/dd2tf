//go:generate go-bindata -o tpl.go tmpl

package main

import (
	"github.com/zorkian/go-datadog-api"
)

type Dashboard struct {
}

func (d Dashboard) getElement(client datadog.Client, id int) (interface{}, error) {
	dash, err := client.GetDashboard(*datadog.Int(id))
	return dash, err
}

func (d Dashboard) getAsset() string {
	return "tmpl/timeboard.tmpl"
}

func (d Dashboard) getName() string {
	return "dashboards"
}

func (d Dashboard) String() string {
	return d.getName()
}
