package main

import (
	"bytes"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	phoneDB "github.com/mrpineapples/phonenumbers/db"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "Michael"
	password = "your-password"
	dbname   = "phone_number_normalizer"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", host, port, user, password)
	err := phoneDB.Reset("postgres", psqlInfo, dbname)
	if err != nil {
		panic(err)
	}

	psqlInfo = fmt.Sprintf("%s dbname=%s", psqlInfo, dbname)
	err = phoneDB.Migrate("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	db, err := phoneDB.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Seed(); err != nil {
		panic(err)
	}

	phones, err := db.AllPhones()
	if err != nil {
		panic(err)
	}
	for _, p := range phones {
		fmt.Printf("Working on... %+v\n", p)
		number := normalize(p.Number)
		if number != p.Number {
			fmt.Println("Updating or removing...", number)
			existing, err := db.FindPhone(number)
			if err != nil {
				panic(err)
			}
			if existing != nil {
				err := db.DeletePhone(p.ID)
				if err != nil {
					panic(err)
				}
			} else {
				p.Number = number
				db.UpdatePhone(&p)
			}
		} else {
			fmt.Println("No changes required")
		}
	}
}

type phoneNumber struct {
	id     int
	number string
}

func allPhones(db *sql.DB) ([]phoneNumber, error) {
	rows, err := db.Query("SELECT id, value FROM phone_numbers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ret []phoneNumber
	for rows.Next() {
		var p phoneNumber
		if err := rows.Scan(&p.id, &p.number); err != nil {
			return nil, err
		}
		ret = append(ret, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func insertPhone(db *sql.DB, phone string) (int, error) {
	statement := `INSERT INTO phone_numbers(value) VALUES($1) RETURNING id`
	var id int
	err := db.QueryRow(statement, phone).Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func normalize(phone string) string {
	var buf bytes.Buffer
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			buf.WriteRune(ch)
		}
	}
	return buf.String()
}
