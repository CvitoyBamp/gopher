package database

import (
	"context"
	"fmt"
	"github.com/CvitoyBamp/gopher/internal/customerror"
	"github.com/CvitoyBamp/gopher/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"runtime"
	"sync"
	"time"
)

var (
	usersTable = `CREATE TABLE IF NOT EXISTS users(
        id        serial PRIMARY KEY,
        username  text NOT NULL,
        password  text NOT NULL,
        timestamp timestamp,
        UNIQUE (username))`

	userBalanceTable = `CREATE TABLE IF NOT EXISTS balance(
        userid           bigint NOT NULL REFERENCES users (id),
        currentbalance   numeric,
        withdrawn        numeric,
        timestamp timestamp,
        UNIQUE (userid))`

	ordersTable = `CREATE TABLE IF NOT EXISTS orders(
        id        serial PRIMARY KEY,
        orderid   bigint NOT NULL,
        userid    bigint NOT NULL REFERENCES users (id),
        status    text NOT NULL check (status in ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')),
        accrual   numeric,
        timestamp timestamp,
        UNIQUE    (orderid))`
)

type Postgres struct {
	conn *pgxpool.Pool
}

func NewPostgresInstance(ctx context.Context, pgConfig *pgxpool.Config) *Postgres {

	var (
		so         sync.Once
		pgInstance *Postgres
	)

	so.Do(func() {
		db, err := pgxpool.New(ctx, pgConfig.ConnConfig.ConnString())
		if err != nil {
			log.Fatalf("It's not possible to initialise database, error: %v", err)
		}

		pgInstance = &Postgres{conn: db}
	})

	return pgInstance
}

func PGConfigParser(connString string) (*pgxpool.Config, error) {

	runtimeParams := make(map[string]string, 1)
	runtimeParams["application_name"] = "gopherMarket"

	connCfg, err := pgxpool.ParseConfig(connString)

	if err != nil {
		return nil, err
	}

	connCfg.MaxConns = int32(runtime.NumCPU())
	connCfg.MinConns = 1
	connCfg.ConnConfig.Config.RuntimeParams = runtimeParams

	return connCfg, err

}

func (pg *Postgres) CreateTables() error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, errUsers := pg.conn.Exec(ctx, usersTable)
	if errUsers != nil {
		return fmt.Errorf("users table did't create, error: %v", errUsers)
	}

	_, errOrders := pg.conn.Exec(ctx, ordersTable)
	if errOrders != nil {
		return fmt.Errorf("balance table did't create, error: %v", errOrders)
	}

	_, errBalance := pg.conn.Exec(ctx, userBalanceTable)
	if errBalance != nil {
		return fmt.Errorf("balance table did't create, error: %v", errBalance)
	}

	return nil
}

func (pg *Postgres) SetNewUser(username, password string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pg.conn.Exec(ctx, `INSERT INTO users (username, password, timestamp)
	                       VALUES ($1, $2, $3)`, username, password, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	return nil
}

func (pg *Postgres) CheckUser(username, password string) (bool, error) {

	var userExist bool

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exist := pg.conn.QueryRow(ctx, `select
    case when exists(select * from users where username = $1 and password = $2)
       then true
       else false
    end`, username, password)

	err := exist.Scan(&userExist)

	if err != nil {
		return false, err
	}

	return userExist, nil
}

func (pg *Postgres) GetUserData(username string) (int, string, error) {

	var data struct {
		id   int
		pass string
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := pg.conn.QueryRow(ctx, `select id, password from users where username = $1`, username)

	err := row.Scan(&data.id, &data.pass)

	if err != nil {
		return 0, "", err
	}

	return data.id, data.pass, nil
}

func (pg *Postgres) SetOrder(orderid, userid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pg.conn.Exec(ctx, `INSERT INTO orders (orderid, userid, status, accrual, timestamp)
	                       VALUES ($1, $2, $3, $4, $5)`, orderid, userid, "NEW", nil, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	return nil
}

func (pg *Postgres) GetOrderById(orderid string) (model.Order, error) {

	var order model.Order

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exist := pg.conn.QueryRow(ctx, `select userid, status from orders where orderid = $1`, orderid)

	err := exist.Scan(&order.Userid, &order.Status)

	if err != nil {
		return order, err
	}

	return order, nil
}

func (pg *Postgres) GetOrderByUserId(userid string) ([]model.Order, error) {

	var order model.Order
	var orders []model.Order

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ors, errQuery := pg.conn.Query(ctx, `select orderid, status, accrual, timestamp from orders where userid = $1`, userid)

	if errQuery != nil {
		return orders, errQuery
	}

	for ors.Next() {
		errScan := ors.Scan(&order.Orderid, &order.Status, &order.Accrual, &order.Timestamp)
		if errScan != nil {
			return orders, errScan
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (pg *Postgres) GetBalanceByUserId(userid string) (model.Balance, error) {

	var balance model.Balance

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bs := pg.conn.QueryRow(ctx, `select currentbalance, withdrawn from balance where userid = $1`, userid)

	errQuery := bs.Scan(&balance.CurBalance, &balance.Withdrawn)

	if errQuery != nil {
		return balance, errQuery
	}

	return balance, nil
}

func (pg *Postgres) BuyOrder(sum, userid string) error {

	var enough bool

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := pg.conn.QueryRow(ctx, "SELECT (currentbalance >= $1) FROM balance WHERE userid = $2", sum, userid)
	errScan := row.Scan(&enough)
	if errScan != nil {
		return errScan
	}

	if !enough {
		return customerror.ErrNotEnoughMoney
	}

	_, errUpdate := pg.conn.Exec(ctx, `UPDATE balance SET currentbalance = currentbalance - $1, withdrawn = withdrawn + $1 where userid = $2`)
	if errUpdate != nil {
		return errUpdate
	}

	return nil
}

func (pg *Postgres) Ping(ctx context.Context) error {
	return pg.Ping(ctx)
}

func (pg *Postgres) Close() {
	pg.Close()
}
