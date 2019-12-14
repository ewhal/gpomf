// +build postgres

package main

import (
	"fmt"
	_ "github.com/lib/pq"
)

const dbDriver = "postgres"

func makeURL(conf Configuration) string {
	return fmt.Sprintf("user=%s dbname=%s password=%s", conf.Username, conf.Name, conf.Pass)
}
