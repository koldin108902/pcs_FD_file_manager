package router

import (
	"FD/util"

	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-session/session/v3"
	"github.com/gorilla/mux"
)

func GetPosts(w http.ResponseWriter, _ *http.Request) {
	var postList []util.PostList
	data, err := db.Query("SELECT post_id, user_name, title, created FROM post ORDER BY post_id DESC LIMIT 30;") // 추후 page searching도 만들어야함.
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		fmt.Fprint(w, string(resData))
		return
	}

	for data.Next() {
		var post util.PostList
		data.Scan(&post.PostId, &post.UserName, &post.Title, &post.Created)

		postList = append(postList, post)
	}

	resData, _ := json.Marshal(util.Res{
		Data: postList,
		Err:  false,
	})

	fmt.Fprint(w, string(resData))
}

func SearchPost(w http.ResponseWriter, r *http.Request) {
	sql := "SELECT post_id, user_name, title, created FROM post WHERE "

	var searchSetting util.SearchBody
	err := json.NewDecoder(r.Body).Decode(&searchSetting)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		fmt.Fprint(w, string(resData))
		return
	}

	// word search
	if len(searchSetting.Word) < 3 {
		sql += "title LIKE %" + searchSetting.Word + "% AND "
	}

	// club search
	if len(searchSetting.Club) > 0 {
		sql += "club=" + searchSetting.Club + " AND "
	}

	// time search
	if len(searchSetting.StartDate) > 9 {
		// 2022-03-23
		sql += "post_id=(SELECT post_id WHERE create BETWEEN" + searchSetting.StartDate
		if len(searchSetting.EndDate) > 9 {
			sql += "AND " + searchSetting.EndDate
		}
		sql += ")"
	}

	sql += "ORDER BY post_id LIMIT 30;"

	data, err := db.Query(sql)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		fmt.Fprint(w, string(resData))
		return
	}

	var postList []util.PostList
	for data.Next() {
		var row util.PostList
		data.Scan(&row.PostId, &row.UserName, &row.Title, &row.Created)
		postList = append(postList, row)
	}

	resData, _ := json.Marshal(util.Res{
		Data: postList,
		Err:  false,
	})
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(resData))
}

func PostDetail(w http.ResponseWriter, r *http.Request) {
	postId, ok := mux.Vars(r)["postId"]
	if !ok {
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, string(resData))
		return
	}

	var postDetail util.PostDetail
	err := db.QueryRow("SELECT club_id, title, readme, file_path, created FROMM post WHERE id=?;", postId).
		Scan(&postDetail.ClubId, &postDetail.Title, &postDetail.FilePath, &postDetail.Created)
	if err != nil {
		log.Println(err)
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, string(resData))
		return
	}

	err = db.QueryRow("SELECT user_name FROM user WHERE user_id=(SELECT writer_id FROM post WHERE post_id=?);", postId).
		Scan(&postDetail.WriterName)
	if err != nil {
		log.Println(err)
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, string(resData))
		return
	}

	resData, _ := json.Marshal(util.Res{
		Data: postDetail,
		Err:  false,
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(resData))
}

func WritePost(w http.ResponseWriter, r *http.Request) {
	store, err := session.Start(ctx, w, r)
	if err != nil {
		resData, _ := json.Marshal(util.Res{
			Data: "need login",
			Err:  true,
		})
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, string(resData))
		return
	}

	data, ok := store.Get("userId")
	if !ok {
		return
	}
	fmt.Print(data)

	// post data
	var pd util.WritePost

	err = json.NewDecoder(r.Body).Decode(&pd)
	if err != nil {
		log.Println(err)
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		w.WriteHeader(400)
		fmt.Fprint(w, string(resData))
		return
	}

	postId, err := db.Exec(`INSERT INTO public.post(
		title, readme, file_path, created, user_id, club_id) VALUES ($1, $2, $3, $4, $5, $6);`,
		pd.Title, pd.Readme, pd.FilePath, pd.Created, pd.UserId, pd.ClubId)

	if err != nil {
		log.Println(err)
		resData, _ := json.Marshal(util.Res{
			Data: nil,
			Err:  true,
		})
		w.WriteHeader(500)
		fmt.Fprint(w, string(resData))
		return
	}
	resData, _ := json.Marshal(util.Res{
		Data: postId,
		Err:  false,
	})
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, string(resData))
}
