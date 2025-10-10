package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID   uint
	Name string
	Age  int
}

func main() {
	db, err := gorm.Open(sqlite.Open("gorm_test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Автомиграция
	db.AutoMigrate(&User{})

	// Вставка
	user := User{Name: "Alexey", Age: 45}
	db.Debug().Create(&user)

	// Чтение
	var users []User
	db.Debug().Find(&users)

	for _, u := range users {
		fmt.Printf("ID: %d, Имя: %s, Возраст: %d\n", u.ID, u.Name, u.Age)
	}
}
