package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB

//docker run --name db -p 3306:3306 -v $HOME/AutoPark/database:/docker-entrypoint-initdb.d -e MYSQL_ROOT_PASSWORD=abc123 -e MYSQL_DATABASE=AutoPark -e MYSQL_USER=94213020 -e MYSQL_PASSWORD=845566321 -d mysql

func init() {
	DB = sqlx.MustConnect("mysql", "94213020:845566321@(localhost)/AutoPark?interpolateParams=true")
}
