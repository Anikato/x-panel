package database

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"

	_ "github.com/lib/pq"
)

type PostgresClient struct {
	db       *sql.DB
	Address  string
	Port     uint
	Username string
	Password string
}

func NewPostgresClient(address string, port uint, username, password string) (*PostgresClient, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", address, port, username, password)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &PostgresClient{db: db, Address: address, Port: port, Username: username, Password: password}, nil
}

func (c *PostgresClient) Close() { c.db.Close() }

func (c *PostgresClient) CreateDatabase(name, owner string) error {
	if owner == "" {
		owner = c.Username
	}
	_, err := c.db.Exec(fmt.Sprintf("CREATE DATABASE \"%s\" OWNER \"%s\"", name, owner))
	return err
}

func (c *PostgresClient) DeleteDatabase(name string) error {
	_, err := c.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", name))
	return err
}

func (c *PostgresClient) ListDatabases() ([]string, error) {
	rows, err := c.db.Query("SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		dbs = append(dbs, name)
	}
	return dbs, nil
}

func (c *PostgresClient) CreateUser(username, password string) error {
	_, err := c.db.Exec(fmt.Sprintf("CREATE ROLE \"%s\" WITH LOGIN PASSWORD '%s'", username, password))
	return err
}

func (c *PostgresClient) ChangePassword(username, password string) error {
	_, err := c.db.Exec(fmt.Sprintf("ALTER ROLE \"%s\" WITH PASSWORD '%s'", username, password))
	return err
}

func (c *PostgresClient) DeleteUser(username string) error {
	_, err := c.db.Exec(fmt.Sprintf("DROP ROLE IF EXISTS \"%s\"", username))
	return err
}

func (c *PostgresClient) Backup(database, outFile string) error {
	cmd := exec.Command("pg_dump",
		"-h", c.Address,
		"-p", fmt.Sprintf("%d", c.Port),
		"-U", c.Username,
		"-Fc",
		"-f", outFile,
		database,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", c.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func (c *PostgresClient) Restore(database, inFile string) error {
	cmd := exec.Command("pg_restore",
		"-h", c.Address,
		"-p", fmt.Sprintf("%d", c.Port),
		"-U", c.Username,
		"-d", database,
		inFile,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", c.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}
