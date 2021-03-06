package router

import (
	"FD/util"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"golang.org/x/crypto/argon2"
)

func Login(w http.ResponseWriter, r *http.Request) {
	if userId := util.LoginCheck(r); userId != nil {
		util.GlobalErr("already login", nil, 400, w)
		return
	}

	var loginData util.Login
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		util.GlobalErr("data isn't json", err, 400, w)
		return
	}

	var confirmData util.ConfirmLoginData
	var userId int
	err = db.QueryRow("SELECT user_id, password, salt FROM \"user\" WHERE login_id=$1", loginData.LoginId).
		Scan(&userId, &confirmData.Password, &confirmData.Salt)

	if err != nil {
		util.GlobalErr("id error", err, http.StatusUnauthorized, w)
		return
	}

	confirmData.DecodeSalt, _ = hex.DecodeString(confirmData.Salt)

	encodedPwd := hex.EncodeToString(
		argon2.IDKey([]byte(loginData.Password), confirmData.DecodeSalt,
			argonConfig.Time, argonConfig.Memory, argonConfig.Thread, argonConfig.KeyLen))

	if encodedPwd != confirmData.Password {
		util.GlobalErr("password error", err, http.StatusUnauthorized, w)
		return
	}

	sessionID := [8]byte{}
	rand.Read(sessionID[:])

	_, err = util.Rdb.Set(ctx, hex.EncodeToString(sessionID[:]), userId, 0).Result()
	if err != nil {
		util.GlobalErr("generate session error", err, 500, w)
		return
	}

	data, _ := util.Rdb.Get(ctx, hex.EncodeToString(sessionID[:])).Result()
	fmt.Println(data)
	fmt.Println(hex.EncodeToString(sessionID[:]))

	http.SetCookie(w, &http.Cookie{
		Name:     "sessionID",
		Value:    hex.EncodeToString(sessionID[:]),
		HttpOnly: true,
	})
	resData, _ := json.Marshal(util.Res{
		Data: "login sucess",
		Err:  false,
	})
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(resData))
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	var signUpData util.SignUp
	err := json.NewDecoder(r.Body).Decode(&signUpData)
	if err != nil {
		util.GlobalErr("data isn't json", err, 400, w)
		return
	}

	if signUpData.IsValidLen() {
		util.GlobalErr("data is not enough", nil, 400, w)
		return
	}

	salt := make([]byte, 32)
	rand.Read(salt)
	encryptedPwd := argon2.IDKey([]byte(signUpData.Password), salt, argonConfig.Time, argonConfig.Memory, argonConfig.Thread, argonConfig.KeyLen)

	_, err = db.Exec("INSERT INTO public.user (club_id, user_name, email, login_id, password, phone_num, salt) VALUES ($1, $2, $3, $4, $5, $6, $7);",
		signUpData.ClubId, signUpData.UserName, signUpData.Email, signUpData.LoginId, hex.EncodeToString(encryptedPwd), signUpData.PhoneNum, hex.EncodeToString(salt))

	if err != nil {
		util.GlobalErr("cannot sign up", err, 400, w)
		return
	}

	resData, _ := json.Marshal(util.Res{
		Data: "success",
		Err:  false,
	})

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, string(resData))
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if userId := util.LoginCheck(r); userId == nil {
		util.GlobalErr("didn't login", nil, 400, w)
		return
	}
	c, err := r.Cookie("sessionID")
	if err != nil {
		util.GlobalErr("cannot logout", err, 500, w)
		return
	}
	c.MaxAge = -1
	http.SetCookie(w, c)

	resData, _ := json.Marshal(util.Res{
		Data: "logout success",
		Err:  false,
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(resData))
}
