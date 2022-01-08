package db

import (
	"database/sql"
	"errors"
	"github.com/MajaSuite/mqtt/transport"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

const (
	createTables = `
		CREATE TABLE IF NOT EXISTS auth (
			ena bool default false,
			login varchar2(64) not null,
			pass varchar2(64) not null,
			UNIQUE(login));
		CREATE TABLE IF NOT EXISTS subscr (
			id varchar2(64) not null,
			topic varchar2(128),
			qos number,
		    UNIQUE(id, topic));
		CREATE TABLE IF NOT EXISTS retain (
			topic varchar2(128),
			payload varchar2(128),
			qos number,
			UNIQUE(topic));`

	insertRetain = `INSERT INTO retain (topic, payload, qos) VALUES (?, ?, ?) 
						ON CONFLICT (topic) DO
						UPDATE SET payload = executed.payload AND qos = executed.qos;`
	deleteRetain       = `DELETE FROM retain WHERE topic = ? AND qos = ?;`
	fetchRetain        = `SELECT topic, payload, qos FROM retain;`
	insertSubscr       = `INSERT INTO subscr (id, topic, qos) VALUES (?, ?, ?);`
	deleteSubscription = `DELETE FROM subscr WHERE id = ? AND topic = ?;`
	fetchSubscription  = `SELECT topic, qos FROM subscr WHERE id = ?;`
	auth               = `SELECT login FROM auth WHERE ena = true AND login = ? AND pass = ?;`
)

var (
	ErrNotFound = errors.New("empty result set")

	db *sql.DB
)

func Open(dbName string) error {
	var err error

	db, err = sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}

	// create database
	statement, err := db.Prepare(createTables)
	if err != nil {
		panic(err)
	}
	statement.Exec()

	return nil
}

func Close() {
	db.Close()
}

func SaveRetain(topic string, payload string, qos int) error {
	statement, err := db.Prepare(insertRetain)
	if err != nil {
		log.Println("error prepare retain: %s", err)
		return err
	}

	if _, err = statement.Exec(topic, payload, qos); err != nil {
		log.Println("error save retain data: %s", err)
		return err
	}

	log.Println("saved retain message {topic: %s, payload: %s, qos: %d}", topic, payload, qos)

	return nil
}

func DeleteRetain(topic string, qos int) error {
	statement, err := db.Prepare(deleteRetain)
	if err != nil {
		log.Println("error delete retain: %s", err)
		return err
	}

	if _, err = statement.Exec(topic, qos); err != nil {
		log.Println("error delete retain data: %s", err)
		return err
	}

	log.Println("delete retain for topic %s qos %d", topic, qos)

	return nil
}

func FetchRetain() (map[string]transport.Event, error) {
	query, err := db.Query(fetchRetain)
	if err != nil {
		log.Println("error prepare fetch retain: %s", err)
		return nil, err
	}
	defer query.Close()

	res := make(map[string]transport.Event)

	for query.Next() {
		var topic, payload string
		var qos int
		if err := query.Scan(&topic, &payload, &qos); err != nil {
			log.Println("error fetch retain: %s", err)
		}
		res[topic] = transport.Event{
			Topic:   transport.EventTopic{Name: topic, Qos: qos},
			Payload: payload,
			Qos:     qos,
			Retain:  true,
		}
	}

	if query.Err() != nil {
		return nil, ErrNotFound
	}

	return res, nil
}

func SaveSubscription(id string, topic string, qos int) error {
	statement, err := db.Prepare(insertSubscr)
	if err != nil {
		log.Println("error prepare subscription: %s", err)
		return err
	}

	if _, err = statement.Exec(id, topic, qos); err != nil {
		log.Println("error save subscription data: %s", err)
		return err
	}

	log.Println("saved subscription {id: %s, topic: %s, qos: %d}", id, topic, qos)

	return nil
}

func DeleteSubscription(id string, topic string) error {
	statement, err := db.Prepare(deleteSubscription)
	if err != nil {
		log.Println("error delete subscription: %s", err)
		return err
	}

	if _, err = statement.Exec(id, topic); err != nil {
		log.Println("error delete subscription data: %s", err)
		return err
	}

	log.Println("delete subscription for client-id %s", id)

	return nil
}

func FetchSubcription(id string) (map[string]int, error) {
	query, err := db.Query(fetchSubscription, id)
	if err != nil {
		log.Println("error prepare fetch subscription: %s", err)
		return nil, err
	}
	defer query.Close()

	res := make(map[string]int)

	for query.Next() {
		var topic string
		var qos int
		if err := query.Scan(&topic, &qos); err != nil {
			log.Println("error fetch subscription: %s", err)
		}
		res[topic] = qos
	}

	if query.Err() != nil {
		return nil, ErrNotFound
	}

	return res, nil
}

// TODO use bcrypt to store passwords
func CheckAuth(login string, pass string) error {
	var res string
	if err := db.QueryRow(auth, login, pass).Scan(&res); err != nil {
		return err
	}

	if res == login {
		return nil
	}

	return ErrNotFound
}