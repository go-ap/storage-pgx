package pgx

import (
	"fmt"

	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-ap/storage-pgx/internal/log"
	"github.com/jackc/pgx"
)

var RootDb = "postgres"

func openConn(c pgx.ConnConfig) (*pgx.Conn, error) {
	return pgx.Connect(c)
}

func Bootstrap(opt Config, rootUser string, rootPw []byte) error {
	logger := lw.Dev(lw.SetLevel(lw.DebugLevel))
	var conn *pgx.Conn
	var err error

	// @todo(marius): we're no longer loading SQL db config env variables
	conf := BackendConfig{}
	if conf.User == "" {
		return errors.Newf("empty user")
	}
	if conf.Name == "" {
		return errors.Newf("empty name")
	}
	if conf.Host == "" {
		return errors.Newf("empty host")
	}
	if opt.BaseURL == "" {
		return errors.Newf("empty base URL")
	}

	conn, err = openConn(pgx.ConnConfig{
		Host:     conf.Host,
		Port:     uint16(conf.Port),
		Database: RootDb,
		User:     rootUser,
		Password: string(rootPw),
		Logger:   log.NewPgxLogger(logger),
	})
	if err != nil {
		return err
	}

	exec := func(qRaw string, par ...interface{}) error {
		qSql := fmt.Sprintf(qRaw, par...)
		_, err = conn.Exec(qSql)
		if err != nil {
			return errors.Annotatef(err, "unable to execute: %s", qRaw)
		}
		return nil
	}

	// Root queries
	err = exec(createRoleWithPass, conf.User, conf.Pw)
	if err != nil {
		return err
	}
	err = exec(createDbForRole, conf.Name, conf.User)
	if err != nil {
		return err
	}
	conn.Close()

	// non-root queries
	conn, err = openConn(pgx.ConnConfig{
		Host:     conf.Host,
		Port:     uint16(conf.Port),
		Database: conf.Name,
		User:     conf.User,
		Password: conf.Pw,
		Logger:   log.NewPgxLogger(logger),
	})
	if err != nil {
		return err
	}
	defer conn.Close()
	err = exec(createActivityPubObjects)
	if err != nil {
		return err
	}
	err = exec(createActivityPubActivities)
	if err != nil {
		return err
	}
	err = exec(createActivityPubActors)
	if err != nil {
		return err
	}
	err = exec(createActivityPubCollections)
	if err != nil {
		return err
	}
	return nil
}

type BackendConfig struct {
	User string
	Name string
	Host string
	Pw   string
	Port int
}

func Clean(opt Config, rootUser string, rootPw []byte) error {
	logger := lw.Dev(lw.SetLevel(lw.DebugLevel))
	var conn *pgx.Conn
	var err error

	// @todo(marius): we're no longer loading SQL db config env variables
	conf := BackendConfig{}
	if conf.User == "" {
		return errors.Newf("empty user")
	}
	if conf.Name == "" {
		return errors.Newf("empty name")
	}
	if conf.Host == "" {
		return errors.Newf("empty host")
	}

	conn, err = openConn(pgx.ConnConfig{
		Host:     conf.Host,
		Port:     uint16(conf.Port),
		Database: RootDb,
		User:     rootUser,
		Password: string(rootPw),
		Logger:   log.NewPgxLogger(logger),
		LogLevel: pgx.LogLevelWarn,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	exec := func(qRaw string, par ...interface{}) error {
		qSql := fmt.Sprintf(qRaw, par...)
		_, err = conn.Exec(qSql)
		if err != nil {
			return errors.Annotatef(err, "unable to execute: %s", qRaw)
		}
		return nil
	}
	err = exec(dropDatabase, conf.Name)
	if err != nil {
		return err
	}
	err = exec(dropRole, conf.User)
	if err != nil {
		return err
	}
	return nil
}
