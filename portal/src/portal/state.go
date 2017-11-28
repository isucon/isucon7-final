package portal

import (
	"database/sql"
	"flag"
	"time"

	"github.com/pkg/errors"
)

var (
	locJST      *time.Location
	db          *sql.DB
	databaseDSN = flag.String("database-dsn", "root:@/isu7fportal_day0", "database `dsn`")
	debugMode   = flag.Bool("debug", false, "enable debug mode")

	infoText         string
	contestDayNumber int
	contestStatus    ContestStatus
)

type ContestStatus int

const (
	ContestStatusNotStarted ContestStatus = iota
	ContestStatusStarted
	ContestStatusEnded
)

func GetContestStatus() ContestStatus {
	return contestStatus
}

func InitState() error {
	var err error

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return err
	}
	locJST = loc

	dsn := *databaseDSN + "?charset=utf8mb4&parseTime=true&loc=Asia%2FTokyo&time_zone='Asia%2FTokyo'"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return errors.Wrapf(err, "sql.Open %q", dsn)
	}

	err = db.Ping()
	if err != nil {
		return errors.Wrapf(err, "db.Ping %q", dsn)
	}

	err = db.QueryRow("SELECT CONVERT(value,SIGNED) FROM setting WHERE name = 'day'").Scan(&contestDayNumber)
	if err != nil {
		return errors.Wrap(err, "SELECT day")
	}

	if *debugMode {
		// debug
		contestStatus = ContestStatusStarted
	}

	return nil
}

func CheckTimeoutJob() (int, error) {
	return checkTimeoutJob()
}
