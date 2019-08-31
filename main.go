package main

import (
	"github.com/kataras/iris"
	"log"
)

func main() {
	app := App()
	defer DB.Close()
	log.Panic(app.Run(iris.Addr(":9090")))
}
