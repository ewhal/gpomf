// +build postgres

package main

import (
	"fmt"
	_ "github.com/lib/pq"
	"strings"
)

const dbDriver = "postgres"

func makeQuery(q string) (r string) {
	num := 1
	count := strings.Count(q, "?")
	for idx, part := range strings.Split(q, "?") {
		if count > idx {
			r += fmt.Sprintf("%s$%d", part, num)
			num++
		} else {
			r += part
		}
	}
	return
}

func makeURL(conf Configuration) string {
	if conf.Username == "" && conf.Name == "" && conf.Pass == "" {
		return "host=/var/run/postgresql"
	}
	return fmt.Sprintf("user=%s dbname=%s password=%s", conf.Username, conf.Name, conf.Pass)
}
