package database

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

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
	_, err := c.db.Exec(fmt.Sprintf("CREATE DATABASE %s OWNER %s", quotePostgresIdentifier(name), quotePostgresIdentifier(owner)))
	return err
}

func (c *PostgresClient) CreateDatabaseWithUser(name, username, password string, superUser bool) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if err := c.CreateUser(username, password); err != nil {
		return err
	}
	if superUser {
		if err := c.ChangePrivileges(username, true); err != nil {
			_ = c.DeleteUser(username)
			return err
		}
	}
	if err := c.CreateDatabase(name, username); err != nil {
		_ = c.DeleteUser(username)
		return err
	}
	if err := c.GrantAllPrivileges(name, username); err != nil {
		_ = c.DeleteDatabase(name)
		_ = c.DeleteUser(username)
		return err
	}
	return nil
}

func (c *PostgresClient) DeleteDatabase(name string) error {
	_, err := c.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", quotePostgresIdentifier(name)))
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

func (c *PostgresClient) ListDatabasesWithInfo() ([]DBInfo, error) {
	rows, err := c.db.Query("SELECT datname, pg_catalog.pg_get_userbyid(datdba) AS owner, pg_encoding_to_char(encoding) AS encoding FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []DBInfo
	for rows.Next() {
		var info DBInfo
		rows.Scan(&info.Name, &info.Owner, &info.Charset)
		dbs = append(dbs, info)
	}
	return dbs, nil
}

func (c *PostgresClient) CreateUser(username, password string) error {
	_, err := c.db.Exec(fmt.Sprintf("CREATE ROLE %s WITH LOGIN PASSWORD %s", quotePostgresIdentifier(username), quotePostgresLiteral(password)))
	return err
}

func (c *PostgresClient) ChangePassword(username, password string) error {
	_, err := c.db.Exec(fmt.Sprintf("ALTER ROLE %s WITH PASSWORD %s", quotePostgresIdentifier(username), quotePostgresLiteral(password)))
	return err
}

func (c *PostgresClient) ChangePrivileges(username string, superUser bool) error {
	privilege := "NOSUPERUSER"
	if superUser {
		privilege = "SUPERUSER"
	}
	_, err := c.db.Exec(fmt.Sprintf("ALTER ROLE %s WITH %s", quotePostgresIdentifier(username), privilege))
	return err
}

func (c *PostgresClient) GrantAllPrivileges(database, username string) error {
	_, err := c.db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", quotePostgresIdentifier(database), quotePostgresIdentifier(username)))
	return err
}

func (c *PostgresClient) DeleteUser(username string) error {
	_, err := c.db.Exec(fmt.Sprintf("DROP ROLE IF EXISTS %s", quotePostgresIdentifier(username)))
	return err
}

func (c *PostgresClient) Backup(database, outFile string) error {
	cmd := exec.Command("pg_dump",
		"-h", c.Address,
		"-p", fmt.Sprintf("%d", c.Port),
		"-U", c.Username,
		"-Fc",
		"--no-owner",
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
	if IsSQLRestoreFile(inFile) {
		return c.restoreSQL(database, inFile)
	}

	cmd := exec.Command("pg_restore",
		"-h", c.Address,
		"-p", fmt.Sprintf("%d", c.Port),
		"-U", c.Username,
		"--no-owner",
		"--no-privileges",
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

func (c *PostgresClient) restoreSQL(database, inFile string) error {
	sqlFile, err := PrepareSQLRestoreFile(inFile)
	if err != nil {
		return err
	}
	defer sqlFile.Cleanup()

	cmd := exec.Command("psql",
		"-h", c.Address,
		"-p", fmt.Sprintf("%d", c.Port),
		"-U", c.Username,
		"-d", database,
		"-f", sqlFile.Path,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", c.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func quotePostgresIdentifier(value string) string {
	if value == "" || !utf8.ValidString(value) {
		return `""`
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quotePostgresLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
