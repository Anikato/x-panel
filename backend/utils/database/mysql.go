package database

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlClient struct {
	db         *sql.DB
	Address    string
	Port       uint
	Username   string
	Password   string
	SocketPath string
}

func NewMysqlClient(address string, port uint, username, password string) (*MysqlClient, error) {
	client, err := newMysqlTCPClient(address, port, username, password)
	if err == nil {
		return client, nil
	}
	tcpErr := err
	if !isLocalMysqlAddress(address) {
		return nil, tcpErr
	}
	for _, socketPath := range mysqlSocketCandidates() {
		if _, statErr := os.Stat(socketPath); statErr != nil {
			continue
		}
		client, err = newMysqlSocketClient(address, port, username, password, socketPath)
		if err == nil {
			return client, nil
		}
	}
	return nil, tcpErr
}

func newMysqlTCPClient(address string, port uint, username, password string) (*MysqlClient, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", username, password, address, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &MysqlClient{db: db, Address: address, Port: port, Username: username, Password: password}, nil
}

func newMysqlSocketClient(address string, port uint, username, password, socketPath string) (*MysqlClient, error) {
	dsn := fmt.Sprintf("%s:%s@unix(%s)/", username, password, socketPath)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &MysqlClient{db: db, Address: address, Port: port, Username: username, Password: password, SocketPath: socketPath}, nil
}

func isLocalMysqlAddress(address string) bool {
	addr := strings.TrimSpace(strings.ToLower(address))
	return addr == "" || addr == "127.0.0.1" || addr == "localhost" || addr == "::1"
}

func mysqlSocketCandidates() []string {
	return []string{
		"/run/mysqld/mysqld.sock",
		"/var/run/mysqld/mysqld.sock",
		"/var/lib/mysql/mysql.sock",
		"/tmp/mysql.sock",
	}
}

func (c *MysqlClient) Close() { c.db.Close() }

func (c *MysqlClient) CreateDatabase(name, charset string) error {
	if charset == "" {
		charset = "utf8mb4"
	}
	_, err := c.db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s", name, charset))
	return err
}

func (c *MysqlClient) DeleteDatabase(name string) error {
	_, err := c.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
	return err
}

type DBInfo struct {
	Name    string
	Charset string
	Owner   string
}

func (c *MysqlClient) ListDatabases() ([]string, error) {
	rows, err := c.db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if name == "information_schema" || name == "performance_schema" || name == "mysql" || name == "sys" {
			continue
		}
		dbs = append(dbs, name)
	}
	return dbs, nil
}

func (c *MysqlClient) ListDatabasesWithInfo() ([]DBInfo, error) {
	rows, err := c.db.Query("SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME NOT IN ('information_schema','performance_schema','mysql','sys')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []DBInfo
	for rows.Next() {
		var info DBInfo
		rows.Scan(&info.Name, &info.Charset)
		dbs = append(dbs, info)
	}
	return dbs, nil
}

func (c *MysqlClient) CreateUser(username, password, database string) error {
	_, err := c.db.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", username, password))
	if err != nil {
		return err
	}
	_, err = c.db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", database, username))
	if err != nil {
		return err
	}
	_, err = c.db.Exec("FLUSH PRIVILEGES")
	return err
}

func (c *MysqlClient) ChangePassword(username, password string) error {
	_, err := c.db.Exec(fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s'", username, password))
	if err != nil {
		return err
	}
	_, err = c.db.Exec("FLUSH PRIVILEGES")
	return err
}

func (c *MysqlClient) DeleteUser(username string) error {
	_, err := c.db.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	return err
}

func (c *MysqlClient) Backup(database, outFile string) error {
	args := append(c.commandConnectionArgs(), "--single-transaction", database, fmt.Sprintf("--result-file=%s", outFile))
	cmd := exec.Command("mysqldump", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func (c *MysqlClient) Restore(database, inFile string) error {
	sqlFile, err := PrepareSQLRestoreFile(inFile)
	if err != nil {
		return err
	}
	defer sqlFile.Cleanup()

	f, err := os.Open(sqlFile.Path)
	if err != nil {
		return fmt.Errorf("open sql file: %v", err)
	}
	defer f.Close()

	args := append(c.commandConnectionArgs(), database)
	cmd := exec.Command("mysql", args...)
	cmd.Stdin = f
	if c.Password != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", c.Password))
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func (c *MysqlClient) commandConnectionArgs() []string {
	args := []string{fmt.Sprintf("-u%s", c.Username)}
	if c.SocketPath != "" {
		args = append(args, "--protocol=SOCKET", fmt.Sprintf("--socket=%s", c.SocketPath))
	} else {
		args = append(args, fmt.Sprintf("-h%s", c.Address), fmt.Sprintf("-P%d", c.Port))
	}
	if c.Password != "" {
		args = append(args, fmt.Sprintf("-p%s", c.Password))
	}
	return args
}
