//go:generate go-bindata -o tpl.go tmpl

package main

import (
	"github.com/zorkian/go-datadog-api"
)

type ScreenBoard struct {
}

func (s ScreenBoard) getElement(client datadog.Client, id int) (interface{}, error) {
	elem, err := client.GetScreenboard(*datadog.Int(id))
	return elem, err
}

func (s ScreenBoard) getAsset() string {
	return "tmpl/screenboard.tmpl"
}

func (s ScreenBoard) getName() string {
	return "screenboards"
}

func (s ScreenBoard) String() string {
	return s.getName()
}
