// +build dev

package data

import (
	"net/http"

	"github.com/shurcooL/httpfs/union"
)

var Assets = union.New(map[string]http.FileSystem{
	"/static":    http.Dir("static"),
	"/templates": http.Dir("templates"),
	"/wiki":      http.Dir("wiki"),
})
