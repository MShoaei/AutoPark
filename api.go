package main

import (
	"github.com/alexedwards/argon2id"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kataras/iris"
	"strconv"
	"time"
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
	hash, _ := argon2id.CreateHash(password, argon2)
	//fmt.Println(string(hash[:]))
	//fmt.Println(string(hash[:32]))
	_, err = DB.Exec("INSERT INTO accounts(phone_number, full_name, password) VALUES (?, ?, ?)", phone, name, hash)
	if err != nil {
		//log.Fatal(err)
		context.StatusCode(iris.StatusBadRequest)
		_, _ = context.JSON(iris.Map{"success": false, "error": err})
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
		return
	}
	if match, _ := argon2id.ComparePasswordAndHash(password, user.Password); !match {
		context.StatusCode(iris.StatusUnauthorized)
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
		_, _ = context.JSON(iris.Map{"exists": true})
	case iris.MethodPost:
		id, _ = strconv.Atoi(context.PostValue("id"))
		newPassword, _ := argon2id.CreateHash(context.PostValue("pass"), argon2)
		_, err = DB.Query("UPDATE accounts SET password=? WHERE id=?", newPassword, id)
		if err != nil {
			context.StatusCode(iris.StatusBadRequest)
			return
		}
		_, _ = context.JSON(iris.Map{})
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
		StartTime time.Time `db:"start_time"`
		EndTime   time.Time `db:"end_time"`
		Node1     string
		Node2     string
	}
	id := context.Params().Get("id")
	err = DB.Select(&detail, "SELECT name, capacity, start_time, end_time, node1, node2 FROM parking WHERE id=?", id)
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
		Free   bool
		Price  float64
	}, 0)
	// selects reserved spots
	err = DB.Select(&spots, "SELECT s.id, s.number, s.free, s.price  FROM spots s JOIN reserves r on s.id = r.spot_id WHERE s.parking_id=? AND s.floor_id=? AND r.start_time BETWEEN ? AND ?", parkingID, floorID, startTime, startTime.Add(1*time.Hour))
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	var capacity int
	err = DB.Get(&capacity, "SELECT capacity FROM floors WHERE id=?", floorID)
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
	err = DB.Select(&price, "SELECT price FROM parking WHERE id=?", id)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, _ = context.JSON(iris.Map{"price": price})
}

func reserveSpot(context iris.Context) {
	userID := context.PostValue("user_id")
	carID := context.PostValue("car_id")
	spotID := context.PostValue("spot_id")
	startTime := context.PostValue("startTime")
	endTime := context.PostValue("endTime")
	date := context.PostValue("date")
	paidOnline := context.PostValue("paid_online")
	price := context.PostValue("price")
	_, err := DB.Query("INSERT INTO reserves(user_id, car_id, spot_id, start_time, end_time, date, paid_online, price) VALUE (?, ?, ?, ?, ?, ?, ?, ?)",
		userID, carID, spotID, startTime, endTime, date, paidOnline, price)
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
	err = DB.Select(&credit, "SELECT credit FROM wallets WHERE user_id=?", id)
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
	err = DB.Select(&car, "SELECT id, model, plate, color FROM car WHERE user_id=?", userID)
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
	_, err = DB.Query("UPDATE car SET model=?, plate=2, color=4 WHERE user_id=5", model, plate, color, userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
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
		return
	}
	_, _ = context.JSON(iris.Map{"success": true})
}

func allReserves(context iris.Context) {
	var err error
	userID, err := context.Params().GetInt("id")
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	all := make([]struct {
		Name      string
		Number    int
		Date      time.Time
		StartTime time.Time `db:"start_time"`
		Plate     string
	}, 0)
	err = DB.Select(&all, "SELECT p.name, f.number, date, reserves.start_time, plate  FROM reserves JOIN spots s on reserves.spot_id = s.id JOIN parking p on s.parking_id = p.id JOIN floors f on s.floor_id = f.id WHERE user_id=?", userID)
	if err != nil {
		context.StatusCode(iris.StatusBadRequest)
		return
	}
	_, _ = context.JSON(iris.Map{"spots": all})
}
