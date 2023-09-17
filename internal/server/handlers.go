package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CvitoyBamp/gopher/internal/customerror"
	"github.com/CvitoyBamp/gopher/internal/jwt"
	"github.com/CvitoyBamp/gopher/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/matthewhartstonge/argon2"
	"io"
	"log"
	"net/http"
)

func (bs *BackendServer) registerHandler(w http.ResponseWriter, r *http.Request) {

	var registerStruct model.Register

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Bad request body, error: ", err.Error())
		return
	}

	errUn := json.Unmarshal(body, &registerStruct)
	if errUn != nil {
		http.Error(w, errUn.Error(), http.StatusBadRequest)
		log.Println("Impossible to unmarshal, error: ", errUn.Error())
		return
	}

	argon := argon2.DefaultConfig()

	encoded, errEnc := argon.HashEncoded([]byte(registerStruct.Password))
	if errEnc != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Println(errEnc.Error())
		return
	}

	errDB := bs.DB.SetNewUser(registerStruct.Username, string(encoded))
	if errDB != nil {
		var pgErr *pgconn.PgError
		if errors.As(errDB, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				log.Println("Impossible to add user to DB, username is already exists.")
				http.Error(w, "Username is already used.", http.StatusConflict)
				return
			}
			http.Error(w, errDB.Error(), http.StatusBadRequest)
			log.Println("Impossible to add user to DB, error: ", errDB.Error())
			return
		}
		http.Error(w, errDB.Error(), http.StatusInternalServerError)
		log.Println(http.StatusText(http.StatusInternalServerError), errDB.Error())
		return
	}

	token, errJWT := jwt.CreateJWTToken(registerStruct.Username, registerStruct.Password)
	if errJWT != nil {
		http.Error(w, errJWT.Error(), http.StatusBadGateway)
		log.Println("Can't create Bearer token", errJWT.Error())
		return
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))

	_, errResp := fmt.Fprintf(w, "Successfully registred, your Bearer token: %s", token)
	if errResp != nil {
		log.Println("Error while response after registration, error: ", errResp.Error())
	}
}

func (bs *BackendServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	var authUser model.Register

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Bad request body, error: ", err.Error())
		return
	}

	errUn := json.Unmarshal(body, &authUser)
	if errUn != nil {
		http.Error(w, errUn.Error(), http.StatusBadRequest)
		log.Println("Impossible to unmarshal, error: ", errUn.Error())
		return
	}

	_, pass, errUser := bs.DB.GetUserData(authUser.Username)
	if errUser != nil {
		http.Error(w, "Such user doesn't registered", http.StatusUnauthorized)
		log.Println(errUser.Error())
		return
	}

	ok, errDecode := argon2.VerifyEncoded([]byte(authUser.Password), []byte(pass))
	if errDecode != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		log.Println(errDecode.Error())
		return
	}

	if !ok {
		http.Error(w, "No such user exist or incorrect pass.", http.StatusUnauthorized)
		return
	}

	token, errJWT := jwt.CreateJWTToken(authUser.Username, authUser.Password)
	if errJWT != nil {
		http.Error(w, errJWT.Error(), http.StatusBadGateway)
		log.Println("Can't create Bearer token", errJWT.Error())
		return
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))

	_, errResp := fmt.Fprintf(w, "Successfully authorized, your refreshed Bearer token: %s", token)
	if errResp != nil {
		log.Println("Error while response after authorize, error: ", errResp.Error())
	}
}

func (bs *BackendServer) postOrdersHandler(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	orderNum, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Bad request body, error: ", err.Error())
		return
	}

	if !checkLuhn(string(orderNum)) {
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	ui := r.Header.Get("Gopher-User-Id")

	errSetOrder := bs.DB.SetOrder(string(orderNum), ui)
	if errSetOrder != nil {
		var pgErr *pgconn.PgError
		if errors.As(errSetOrder, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				order, errGetOrder := bs.DB.GetOrderByID(string(orderNum))
				if errGetOrder != nil {
					http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
					return
				}

				if order.Userid != ui {
					w.WriteHeader(http.StatusConflict)
					_, errSC := fmt.Fprintf(w, "The order number has already been uploaded by the other user.")
					if errSC != nil {
						http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
						return
					}
					return
				}

				w.WriteHeader(http.StatusOK)
				_, errSC := fmt.Fprintf(w, "The order number has already been uploaded by this user.")
				if errSC != nil {
					http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
					return
				}
				return

			}
		}
	}

	w.WriteHeader(http.StatusAccepted)
	_, errAccept := fmt.Fprintf(w, "Order accepted for processing.")
	if errAccept != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

}

func (bs *BackendServer) getOrdersHandler(w http.ResponseWriter, r *http.Request) {

	ui := r.Header.Get("Gopher-User-Id")

	orders, errOrders := bs.DB.GetOrderByUserID(ui)
	if errOrders != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	if orders == nil {
		w.WriteHeader(http.StatusNoContent)
		_, errStatusNoContent := fmt.Fprintf(w, "No orders yet.")
		if errStatusNoContent != nil {
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}
		return
	}

	body, errMarshal := json.Marshal(orders)
	if errMarshal != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	_, errResp := fmt.Fprint(w, string(body))
	if errResp != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
}

func (bs *BackendServer) getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	ui := r.Header.Get("Gopher-User-Id")

	balance, errBalance := bs.DB.GetBalanceByUserID(ui)
	if errBalance != nil {
		return
	}

	body, errMarshal := json.Marshal(balance)
	if errMarshal != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	_, errResp := fmt.Fprint(w, string(body))
	if errResp != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

}

func (bs *BackendServer) withdrawHandler(w http.ResponseWriter, r *http.Request) {

	var withdraw model.Withdrawn

	ui := r.Header.Get("Gopher-User-Id")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Bad request body, error: ", err.Error())
		return
	}

	errUn := json.Unmarshal(body, &withdraw)
	if errUn != nil {
		http.Error(w, errUn.Error(), http.StatusBadRequest)
		log.Println("Impossible to unmarshal, error: ", errUn.Error())
		return
	}

	order, errOrder := bs.DB.GetOrderByID(withdraw.Orderid)

	if errOrder != nil {
		if errOrder != errors.New("no rows in result set") {
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	errSum := bs.DB.BuyOrder(order.Orderid, withdraw.Sum, ui)

	if errSum != nil {
		if errors.Is(errSum, customerror.ErrNotEnoughMoney) {
			http.Error(w, http.StatusText(http.StatusPaymentRequired), http.StatusPaymentRequired)
			return
		}
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
}

func (bs *BackendServer) withdrawalsHandlers(w http.ResponseWriter, r *http.Request) {
	ui := r.Header.Get("Gopher-User-Id")

	withdrawns, errDB := bs.DB.GetWithdrawn(ui)
	if errDB != nil {
		if errDB != errors.New("no rows in result set") {
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, errMarshall := json.Marshal(withdrawns)
	if errMarshall != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	_, errResp := fmt.Fprint(w, string(body))
	if errResp != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

}
