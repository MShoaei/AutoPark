package main

import (
	"github.com/alexedwards/argon2id"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kataras/iris"
	"strconv"
	"time"
	"log"
)

var argon2 *argon2id.Params

func init() {
	argon2 = &argon2id.Params{
		Memory:      1 << 10,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}
func App() *iris.Application {
	app := iris.Default()
	app.Post("/register", register)
	app.Post("/login", login)
	app.Get("/forgot/{phone:string}", forgotPassword)
	app.Post("/forgot", forgotPassword)
	app.Get("/parking/list", listParking)
	app.Get("/parking/{id:int}", parkingDetails)
	app.Get("/parking/{id:int}/{floor:int}/{startTime:string}", parkingFloorSpots)
	app.Get("/reserve/{id:int}", parkingPrice)
	app.Post("/reserve", reserveSpot)
	app.Get("/user/{id:int}/credit", getCredit)
	app.Post("/user/{id:int}/credit", updateCredit)
	app.Get("/user/{id:int}/car", getCar)
	app.Patch("/user/{id:int}/car", updateCar)
	app.Patch("/user/{id:int}", updateUser)
	app.Get("/user/{id:int}/reserves", allReserves)

	return app
}

func register(context iris.Context) {
	var err error
	phone := context.PostValue("phone")
	name := context.PostValue("name")
	password := context.PostValue("password")
	credit := context.PostValueFloat64Default("credit", 0)
	hash, _ := argon2id.CreateHash(password, argon2)
	tx, _ := DB.Beginx()
	_, err = tx.Exec("INSERT INTO accounts(phone_number, full_name, password) VALUES (?, ?, ?)", phone, name, hash)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		tx.Rollback()
		return
	}
	var userID int
	err = tx.Get(&userID, "SELECT LAST_INSERT_ID()")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		tx.Rollback()
		return
	}
	_, err = tx.Exec("INSERT INTO wallets(user_id, credit) VALUE (?, ?)", userID, credit)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		tx.Rollback()
		return
	}
	_, err = tx.Exec("INSERT INTO cars(user_id) VALUE (?)", userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		return
	}
	_, _ = context.JSON(iris.Map{"success": true})
}

func login(context iris.Context) {
	var err error
	var user struct {
		ID       int
		Password string
	}
	phone := context.PostValue("phone")
	password := context.PostValue("password")
	err = DB.Get(&user, "SELECT id, password FROM accounts WHERE phone_number=?", phone)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		return
	}
	if match, _ := argon2id.ComparePasswordAndHash(password, user.Password); !match {
		context.StatusCode(iris.StatusUnauthorized)
		_, _ = context.JSON(iris.Map{"success": false})
		return
	}
	_, _ = context.JSON(iris.Map{"id": user.ID})
}

func forgotPassword(context iris.Context) {
	var err error
	var id int

	switch context.Method() {
	case iris.MethodGet:
		phone := context.Params().GetString("phone")
		err = DB.Get(&id, "SELECT id FROM accounts WHERE phone_number=?", phone)
		if err != nil {
			context.StatusCode(iris.StatusBadRequest)
			_, _ = context.JSON(iris.Map{"exists": false})
			return
		}
		_, _ = context.JSON(iris.Map{"exists": true, "id": id})
	case iris.MethodPost:
		id, _ = strconv.Atoi(context.PostValue("id"))
		newPassword, _ := argon2id.CreateHash(context.PostValue("password"), argon2)
		_, err = DB.Query("UPDATE accounts SET password=? WHERE id=?", newPassword, id)
		if err != nil {
			context.StatusCode(iris.StatusBadRequest)
			_, _ = context.JSON(iris.Map{"success": false})
			return
		}
		_, _ = context.JSON(iris.Map{"success": true})
	}
}

func listParking(context iris.Context) {
	var err error
	p := make([]struct {
		ID   int
		Name string
	}, 0)
	err = DB.Select(&p, "SELECT id, name FROM parking")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, _ = context.JSON(iris.Map{"list": p})
}

func parkingDetails(context iris.Context) {
	var err error
	var detail struct {
		Name      string
		Capacity  int
		StartTime string `db:"start_time"`
		EndTime   string `db:"end_time"`
		Node1     string
		Node2     string
	}
	id := context.Params().Get("id")
	err = DB.Get(&detail, "SELECT name, capacity, start_time, end_time, node1, node2 FROM parking WHERE id=?", id)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, _ = context.JSON(iris.Map{"detail": detail})
}

func parkingFloorSpots(context iris.Context) {
	var err error
	parkingID, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	floorID, err := context.Params().GetInt("floor")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	//TODO: why do we need time nad how to use?
	startTime, err := time.Parse("15:04", context.Params().GetString("startTime"))
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	spots := make([]struct {
		ID     int
		Number int
		Price  float64
	}, 0)
	// selects reserved spots
	date := time.Now().Format("2006-01-02")
	err = DB.Select(&spots, "SELECT s.id, s.number, r.price FROM spots s JOIN reserves r on s.id = r.spot_id WHERE s.parking_id=? AND s.floor_id=? AND r.start_time=? AND r.end_time=? AND r.date=?", parkingID, floorID, startTime, startTime.Add(1*time.Hour), date)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	var capacity int
	err = DB.Get(&capacity, "SELECT count(*) FROM spots WHERE floor_id=?", floorID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		log.Println(err)
		return
	}
	_, _ = context.JSON(iris.Map{"spots": spots,
		"capacity": capacity})
}

func parkingPrice(context iris.Context) {
	var err error
	var price float64
	id, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	err = DB.Get(&price, "SELECT price FROM parking WHERE id=?", id)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, _ = context.JSON(iris.Map{"price": price})
}

func reserveSpot(context iris.Context) {
	var err error
	userID := context.PostValue("user_id")
	carID := context.PostValue("car_id")
	parkingID := context.PostValue("parking_id")
	floorID := context.PostValue("floor_id")
	spotNumber := context.PostValue("park_place_number")
	startTime := context.PostValue("start_time")
	endTime := context.PostValue("end_time")
	date := time.Now().Format("2006-01-02")
	paidOnline := context.PostValue("paid_online")
	price := context.PostValue("price")
	var plate string
	err = DB.Get(&plate, "SELECT plate FROM cars WHERE id=?", carID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	var spotID int
	err = DB.Get(&spotID, "SELECT id FROM spots WHERE parking_id=? AND floor_id=? AND number=?", parkingID, floorID, spotNumber)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, err = DB.Exec("INSERT INTO reserves(user_id, car_id, spot_id, start_time, end_time, date, paid_online, price, plate) VALUE (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, carID, spotID, startTime, endTime, date, paidOnline, price, plate)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		return
	}
	_, _ = context.JSON(iris.Map{"success": true})
}

func getCredit(context iris.Context) {
	var err error
	var credit float64
	id, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	err = DB.Get(&credit, "SELECT credit FROM wallets WHERE user_id=?", id)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, _ = context.JSON(iris.Map{"credit": credit})
}

func updateCredit(context iris.Context) {
	var err error
	userID, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	newCredit := context.PostValue("credit")
	_, err = DB.Query("UPDATE wallets SET credit=? WHERE user_id=?", newCredit, userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		return
	}
	_, _ = context.JSON(iris.Map{"success": true})
}

func getCar(context iris.Context) {
	var err error
	userID, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	var car = struct {
		ID    int
		Model string
		Plate string
		Color string
	}{}
	err = DB.Get(&car, "SELECT id, model, plate, color FROM cars WHERE user_id=?", userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, _ = context.JSON(iris.Map{"car": car})
}

func updateCar(context iris.Context) {
	var err error
	model := context.PostValue("model")
	plate := context.PostValue("plate")
	color := context.PostValue("color")
	userID, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, err = DB.Query("UPDATE cars SET model=?, plate=?, color=? WHERE user_id=?", model, plate, color, userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		return
	}
	_, _ = context.JSON(iris.Map{"success": true})
}

func updateUser(context iris.Context) {
	var err error
	name := context.PostValue("name")
	userID, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	password, _ := argon2id.CreateHash(context.PostValue("password"), argon2)
	_, err = DB.Query("UPDATE accounts SET full_name=?, password=? WHERE id=?", name, password, userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false})
		return
	}
	_, _ = context.JSON(iris.Map{"success": true})
}

func allReserves(context iris.Context) {
	var err error
	userID, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		log.Println(err)
		return
	}
	all := make([]struct {
		Name      string
		Number    int
		Date      string
		StartTime string `db:"start_time"`
		Plate     string
	}, 0)
	err = DB.Select(&all, "SELECT p.name, f.number, reserves.date, reserves.start_time, plate FROM reserves JOIN spots s on reserves.spot_id = s.id JOIN parking p on s.parking_id = p.id JOIN floors f on s.floor_id = f.id WHERE user_id=?", userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		log.Println(err)
		return
	}
	_, _ = context.JSON(iris.Map{"spots": all})
}
