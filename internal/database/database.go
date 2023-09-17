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

	withdrawnTable = `CREATE TABLE IF NOT EXISTS withdrawn(
        id        serial PRIMARY KEY,
        userid    bigint NOT NULL REFERENCES users (id),
        orderid   bigint NOT NULL REFERENCES orders (orderid),
        sum       numeric,
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

	_, errWithdrawn := pg.conn.Exec(ctx, withdrawnTable)
	if errWithdrawn != nil {
		return fmt.Errorf("withdrawn table did't create, error: %v", errWithdrawn)
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

	_, errNoAdd := pg.conn.Exec(ctx, `INSERT INTO orders (orderid, userid, status, accrual, timestamp)
	                       VALUES ($1, $2, $3, $4, $5)`, orderid, userid, "NEW", nil, time.Now().Format(time.RFC3339))
	if errNoAdd != nil {
		return errNoAdd
	}

	return nil
}

func (pg *Postgres) GetOrderByID(orderid string) (model.Order, error) {

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

func (pg *Postgres) GetOrderByUserID(userid string) ([]model.Order, error) {

	var order model.Order
	var orders []model.Order

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ors, errQuery := pg.conn.Query(ctx, `select orderid, status, accrual, timestamp from orders where userid = $1 order by timestamp desc`, userid)

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

func (pg *Postgres) GetBalanceByUserID(userid string) (model.Balance, error) {

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

func (pg *Postgres) BuyOrder(orderID, sum, userid string) error {

	var enough bool

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, errTx := pg.conn.Begin(ctx)
	if errTx != nil {
		return errTx
	}
	defer tx.Rollback(ctx)

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

	_, errNoAdd := pg.conn.Exec(ctx, `INSERT INTO withdrawn (orderid, userid, sum, timestamp)
	                       VALUES ($1, $2, $3, $4)`, orderID, userid, sum, time.Now().Format(time.RFC3339))
	if errNoAdd != nil {
		return errNoAdd
	}

	errCommit := tx.Commit(ctx)
	if errCommit != nil {
		return errCommit
	}

	return nil
}

func (pg *Postgres) ParseAccrualByStatus(status string) ([]model.Accrual, error) {

	var accrual model.Accrual
	var accruals []model.Accrual

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orderIDs, errQuery := pg.conn.Query(ctx, `select orderid, status, accrual from orders where status = $1`, status)

	if errQuery != nil {
		return accruals, errQuery
	}

	for orderIDs.Next() {
		errScan := orderIDs.Scan(&accrual.Orderid, &accrual.Status, &accrual.Accrual)
		if errScan != nil {
			return accruals, errScan
		}

		accruals = append(accruals, accrual)
	}

	return accruals, nil
}

func (pg *Postgres) GetAccrual(orderID string) (model.Accrual, error) {

	var accrual model.Accrual

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := pg.conn.QueryRow(ctx, `select orderid, status, accrual from orders where orderid = $1`, orderID)

	errQuery := row.Scan(&accrual.Orderid, &accrual.Status, &accrual.Accrual)

	if errQuery != nil {
		return accrual, errQuery
	}

	return accrual, nil
}

func (pg *Postgres) UpdateAccrual(acc model.Accrual) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, errNoAdd := pg.conn.Exec(ctx, `UPDATE orders SET status = $1, accrual = $2, timestamp = $3 WHERE orderid = $4`,
		acc.Status, acc.Accrual, time.Now().Format(time.RFC3339), acc.Orderid)
	if errNoAdd != nil {
		return errNoAdd
	}

	return nil
}

func (pg *Postgres) GetWithdrawn(userid string) (model.Withdrawn, error) {
	var withdrawn model.Withdrawn

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, errQuery := pg.conn.Query(ctx, `select orderid, sum, timestamp from orders where userid = $1 order by timestamp`, userid)

	if errQuery != nil {
		return withdrawn, errQuery
	}

	for rows.Next() {
		errScan := rows.Scan(&withdrawn.Orderid, &withdrawn.Sum, &withdrawn.Timestamp)
		if errScan != nil {
			return withdrawn, errScan
		}
	}

	return withdrawn, nil
}
