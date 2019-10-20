package main

import (
	"bytes"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
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
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = resetDB(db, dbname)
	if err != nil {
		panic(err)
	}
	db.Close()

	psqlInfo = fmt.Sprintf("%s dbname=%s", psqlInfo, dbname)
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = createPhoneNumbersTable(db)
	if err != nil {
		panic(err)
	}

	_, err = insertPhone(db, "1234567890")
	_, err = insertPhone(db, "123 456 7891")
	id, err := insertPhone(db, "(123) 456 7892")
	_, err = insertPhone(db, "(123) 456-7893")
	_, err = insertPhone(db, "123-456-7894")
	_, err = insertPhone(db, "123-456-7890")
	_, err = insertPhone(db, "1234567892")
	_, err = insertPhone(db, "(123)456-7892")
	if err != nil {
		panic(err)
	}

	number, err := getPhone(db, id)
	if err != nil {
		panic(err)
	}
	fmt.Println("Number is:", number)

	phones, err := allPhones(db)
	if err != nil {
		panic(err)
	}
	for _, p := range phones {
		fmt.Printf("Working on... %+v\n", p)
		number := normalize(p.number)
		if number != p.number {
			fmt.Println("Updating or removing...", number)
			existing, err := findPhone(db, number)
			if err != nil {
				panic(err)
			}
			if existing != nil {
				err := deletePhone(db, p.id)
				if err != nil {
					panic(err)
				}
			} else {
				p.number = number
				updatePhone(db, p)
			}
		} else {
			fmt.Println("No changes required")
		}
	}

	// id, err := insertPhone(db, "1234567890")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("id=", id)
}

func getPhone(db *sql.DB, id int) (string, error) {
	var number string
	err := db.QueryRow("SELECT value FROM phone_numbers WHERE id=$1", id).Scan(&number)
	if err != nil {
		return "", err
	}
	return number, nil
}

func findPhone(db *sql.DB, number string) (*phoneNumber, error) {
	var p phoneNumber
	err := db.QueryRow("SELECT * FROM phone_numbers WHERE value=$1", number).Scan(&p.id, &p.number)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func updatePhone(db *sql.DB, p phoneNumber) error {
	statement := `UPDATE phone_numbers SET value=$2 WHERE id=$1`
	_, err := db.Exec(statement, p.id, p.number)
	return err
}

func deletePhone(db *sql.DB, id int) error {
	statement := `DELETE FROM phone_numbers WHERE id=$1`
	_, err := db.Exec(statement, id)
	return err
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

func createPhoneNumbersTable(db *sql.DB) error {
	statement := `
		CREATE TABLE IF NOT EXISTS phone_numbers (
			id SERIAL,
			value VARCHAR(255)
		)
	`
	_, err := db.Exec(statement)
	return err
}

func resetDB(db *sql.DB, name string) error {
	_, err := db.Exec("DROP DATABASE IF EXISTS " + name)
	if err != nil {
		return err
	}
	return createDB(db, name)
}

func createDB(db *sql.DB, name string) error {
	_, err := db.Exec("CREATE DATABASE " + name)
	if err != nil {
		return err
	}
	return nil
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

// Same function using regex
// func normalize(phone string) string {
// 	re := regexp.MustCompile("\\D")
// 	return re.ReplaceAllString(phone, "")
// }
