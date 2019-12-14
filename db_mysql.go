// +build !postgres

package main

import _ "github.com/go-sql-driver/mysql"

const dbDriver = "mysql"

func makeQuery(q string) string {
	return q
}

func makeURL(conf Configuration) string {
	return conf.Username + ":" + conf.Pass + "@/" + conf.Name + "?charset=utf8"
}
