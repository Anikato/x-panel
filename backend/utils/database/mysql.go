package database

import (
	"database/sql"
	"fmt"
	"io"
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
	if port != 3306 || !isLocalMysqlAddress(address) {
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
	charset = normalizeMysqlCharset(charset)
	_, err := c.db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET %s", quoteMysqlIdentifier(name), charset))
	return err
}

func (c *MysqlClient) DeleteDatabase(name string) error {
	_, err := c.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteMysqlIdentifier(name)))
	return err
}

type DBInfo struct {
	Name       string
	Charset    string
	Owner      string
	Username   string
	Permission string
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
		info.Username, info.Permission = c.loadDatabaseUser(info.Name)
		dbs = append(dbs, info)
	}
	return dbs, nil
}

func (c *MysqlClient) CreateDatabaseWithUser(name, charset, username, password, permission string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if permission == "" {
		permission = "%"
	}
	if err := c.CreateDatabase(name, charset); err != nil {
		return err
	}
	if err := c.CreateUser(username, password, name, permission); err != nil {
		_ = c.DeleteDatabase(name)
		return err
	}
	return nil
}

func (c *MysqlClient) CreateUser(username, password, database string, permissions ...string) error {
	permissionList := normalizeMysqlPermissions(permissions...)
	for _, permission := range permissionList {
		_, err := c.db.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED BY %s", quoteMysqlUser(username, permission), quoteMysqlLiteral(password)))
		if err != nil {
			return err
		}
		_, err = c.db.Exec(fmt.Sprintf("ALTER USER %s IDENTIFIED BY %s", quoteMysqlUser(username, permission), quoteMysqlLiteral(password)))
		if err != nil {
			return err
		}
		_, err = c.db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s", quoteMysqlIdentifier(database), quoteMysqlUser(username, permission)))
		if err != nil {
			return err
		}
	}
	_, err := c.db.Exec("FLUSH PRIVILEGES")
	if err != nil {
		return err
	}
	return nil
}

func (c *MysqlClient) ChangePassword(username, password string, permissions ...string) error {
	var err error
	for _, permission := range normalizeMysqlPermissions(permissions...) {
		_, err = c.db.Exec(fmt.Sprintf("ALTER USER %s IDENTIFIED BY %s", quoteMysqlUser(username, permission), quoteMysqlLiteral(password)))
		if err != nil {
			return err
		}
	}
	_, err = c.db.Exec("FLUSH PRIVILEGES")
	return err
}

func (c *MysqlClient) DeleteUser(username string, permissions ...string) error {
	for _, permission := range normalizeMysqlPermissions(permissions...) {
		if _, err := c.db.Exec(fmt.Sprintf("DROP USER IF EXISTS %s", quoteMysqlUser(username, permission))); err != nil {
			return err
		}
	}
	return nil
}

func (c *MysqlClient) Backup(database, outFile string) error {
	return c.runClient("mysqldump", nil, "--single-transaction", "--result-file="+outFile, "--", database)
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

	return c.runClient("mysql", f, "--", database)
}

func (c *MysqlClient) runClient(program string, stdin io.Reader, options ...string) error {
	var args []string
	if c.Password != "" {
		if strings.ContainsRune(c.Password, '\x00') {
			return fmt.Errorf("MySQL password cannot contain NUL")
		}
		file, err := os.CreateTemp("", "xpanel-mysql-*.cnf")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		escape := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\r", "\\r", "\t", "\\t", "\b", "\\b")
		_, err = fmt.Fprintf(file, "[client]\npassword=\"%s\"\n", escape.Replace(c.Password))
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		// MySQL requires option-file arguments to come before connection options.
		args = append(args, "--defaults-extra-file="+file.Name())
	}
	args = append(args, "-u"+c.Username)
	if c.SocketPath != "" {
		args = append(args, "--protocol=SOCKET", "--socket="+c.SocketPath)
	} else {
		args = append(args, "--protocol=TCP", "-h"+c.Address, fmt.Sprintf("-P%d", c.Port))
	}
	command := exec.Command(program, append(args, options...)...)
	command.Stdin = stdin
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", program, err, output)
	}
	return nil
}

func (c *MysqlClient) loadDatabaseUser(database string) (string, string) {
	rows, err := c.db.Query("SELECT User, Host FROM mysql.db WHERE Db = ? AND User <> 'root' ORDER BY User, Host", database)
	if err != nil {
		return "", ""
	}
	defer rows.Close()
	var username string
	var hosts []string
	for rows.Next() {
		var user, host string
		if err := rows.Scan(&user, &host); err != nil {
			continue
		}
		if username == "" {
			username = user
		}
		if user == username {
			hosts = append(hosts, host)
		}
	}
	return username, strings.Join(hosts, ",")
}

func normalizeMysqlPermissions(permissions ...string) []string {
	var items []string
	for _, permission := range permissions {
		for _, item := range strings.Split(permission, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				items = append(items, item)
			}
		}
	}
	if len(items) == 0 {
		return []string{"%"}
	}
	return items
}

func normalizeMysqlCharset(charset string) string {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "utf8", "latin1", "gbk":
		return strings.ToLower(strings.TrimSpace(charset))
	default:
		return "utf8mb4"
	}
}

func quoteMysqlIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func quoteMysqlLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func quoteMysqlUser(username, permission string) string {
	return quoteMysqlLiteral(username) + "@" + quoteMysqlLiteral(permission)
}
